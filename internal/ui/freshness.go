package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fullsend-ai/fs-dash/internal/domain"
)

const freshnessRefreshLead = 20 * time.Second

func (m *Model) scheduleFreshnessRefresh() tea.Cmd {
	if m.mode != "list" {
		if m.freshnessTickRunning || !m.nextFreshnessRefresh.IsZero() {
			m.freshnessGeneration++
		}
		m.freshnessTickRunning = false
		m.nextFreshnessRefresh = time.Time{}
		return nil
	}

	var earliest time.Time
	consider := func(field domain.Provenance) {
		if field.FreshUntil.IsZero() {
			return
		}
		refreshAt := field.FreshUntil.Add(-freshnessRefreshLead)
		if earliest.IsZero() || refreshAt.Before(earliest) {
			earliest = refreshAt
		}
	}
	if m.opts.Lister != nil {
		for _, s := range m.sections {
			if s.field.Phase == domain.Ready && s.field.HasValue {
				consider(s.field.Provenance)
			}
		}
	}
	if m.opts.Enricher != nil {
		interactive := m.interactivePRs()
		prs := append([]domain.PR(nil), interactive.visible...)
		if interactive.hasSelected {
			prs = append(prs, interactive.selected)
		}
		seen := make(map[prRevision]struct{}, len(prs))
		for _, pr := range prs {
			identity := prRevision{item: pr.ID, revision: pr.HeadSHA}
			if _, ok := seen[identity]; ok {
				continue
			}
			seen[identity] = struct{}{}
			for _, name := range triageFieldNames {
				key := domain.FieldKey{Item: pr.ID, Name: name, Revision: pr.HeadSHA}
				if field := m.fields[key]; field != nil && field.Phase == domain.Ready && field.HasValue {
					consider(field.Provenance)
				}
			}
		}
	}

	if earliest.IsZero() {
		if m.freshnessTickRunning {
			m.freshnessGeneration++
		}
		m.freshnessTickRunning = false
		m.nextFreshnessRefresh = time.Time{}
		return nil
	}
	if m.freshnessTickRunning && earliest.Equal(m.nextFreshnessRefresh) {
		return nil
	}
	m.freshnessTickRunning = true
	m.freshnessGeneration++
	m.nextFreshnessRefresh = earliest
	generation := m.freshnessGeneration
	delay := earliest.Sub(m.opts.Clock.Now())
	if delay < time.Millisecond {
		delay = time.Millisecond
	}
	return tea.Tick(delay, func(time.Time) tea.Msg { return freshnessRefreshMsg{generation: generation} })
}

func (m *Model) refreshDueListData() []tea.Cmd {
	if m.mode != "list" {
		return nil
	}
	now := m.opts.Clock.Now()
	cutoff := now.Add(freshnessRefreshLead)
	var cmds []tea.Cmd
	if m.opts.Lister != nil {
		for i := range m.sections {
			s := &m.sections[i]
			if s.field.Phase != domain.Ready || !s.field.HasValue || s.field.Provenance.FreshUntil.IsZero() || s.field.Provenance.FreshUntil.After(cutoff) {
				continue
			}
			generation := s.field.Queue()
			s.field.Begin(generation)
			s.attempts = 1
			cmds = append(cmds, m.fetchListWithStatus(s.repo, generation, s.attempts, true))
		}
	}

	if m.opts.Enricher == nil {
		return cmds
	}
	interactive := m.interactivePRs()
	prs := append([]domain.PR(nil), interactive.visible...)
	if interactive.hasSelected {
		prs = append(prs, interactive.selected)
	}
	seen := make(map[prRevision]struct{}, len(prs))
	for _, pr := range prs {
		identity := prRevision{item: pr.ID, revision: pr.HeadSHA}
		if _, ok := seen[identity]; ok {
			continue
		}
		seen[identity] = struct{}{}
		for _, name := range triageFieldNames {
			key := domain.FieldKey{Item: pr.ID, Name: name, Revision: pr.HeadSHA}
			field := m.fields[key]
			if field == nil || field.Phase != domain.Ready || !field.HasValue || field.Provenance.FreshUntil.IsZero() || field.Provenance.FreshUntil.After(cutoff) {
				continue
			}
			if name == "activity" {
				m.queueActivity(key, true, true)
			} else {
				m.queueField(key, true)
			}
		}
	}
	cmds = append(cmds, m.startFieldFetches()...)
	cmds = append(cmds, m.startActivityFetches()...)
	return cmds
}
