package ui

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/fullsend-ai/fs-dash/internal/domain"
	"github.com/fullsend-ai/fs-dash/internal/ports"
	"github.com/fullsend-ai/fs-dash/internal/reminder"
	"github.com/fullsend-ai/fs-dash/internal/triage"
)

var triageFieldNames = []string{"reviews", "review_requests", "assignees", "checks", "review_threads", "last_touched", "activity"}

const maxConcurrentActivityFetches = 5
const maxConcurrentFieldFetches = 2
const maxQueueItemsPerUpdate = 64
const offscreenFieldPace = 500 * time.Millisecond
const offscreenActivityPace = 750 * time.Millisecond
const queueContinuationDelay = 5 * time.Millisecond

var tableSummaryFieldNames = []string{"last_touched", "checks", "reviews", "review_threads", "assignees"}

type prRevision struct {
	item     domain.ItemID
	revision string
}

type queueRef struct {
	key        domain.FieldKey
	generation uint64
}

type listPrefetchBatch struct {
	repo       string
	generation uint64
	prs        []domain.PR
}

type interactivePRSet struct {
	selected    domain.PR
	hasSelected bool
	visible     []domain.PR
	identities  map[prRevision]struct{}
	listed      map[prRevision]struct{}
}

func (set interactivePRSet) contains(key domain.FieldKey) bool {
	_, ok := set.identities[prRevision{item: key.Item, revision: key.Revision}]
	return ok
}

func (m Model) interactivePRs() interactivePRSet {
	sections := m.projection()
	set := interactivePRSet{identities: make(map[prRevision]struct{}), listed: make(map[prRevision]struct{})}
	for _, section := range m.sections {
		if !section.field.HasValue {
			continue
		}
		for _, pr := range section.field.Value {
			set.listed[prRevision{item: pr.ID, revision: pr.HeadSHA}] = struct{}{}
		}
	}
	if selected, ok := selectedPRFromSections(sections, m.selected); ok {
		set.selected = selected
		set.hasSelected = true
		set.identities[prRevision{item: selected.ID, revision: selected.HeadSHA}] = struct{}{}
	}
	set.visible = m.visibleRowsForPrefetchFrom(sections)
	for _, pr := range set.visible {
		set.identities[prRevision{item: pr.ID, revision: pr.HeadSHA}] = struct{}{}
	}
	return set
}

func selectedPRFromSections(sections triage.Sections, id domain.ItemID) (domain.PR, bool) {
	row, ok := selectedRowFromSections(sections, id)
	return row.PR, ok
}

func selectedRowFromSections(sections triage.Sections, id domain.ItemID) (triage.Row, bool) {
	for _, rows := range [][]triage.Row{sections.Review, sections.Active, sections.Snoozed} {
		for _, row := range rows {
			if row.PR.ID == id {
				return row, true
			}
		}
	}
	return triage.Row{}, false
}

func (m Model) showingSnoozed() bool {
	return m.filter == triage.FilterSnoozed
}

func filterTabLabel(filter triage.Filter, width int) string {
	if width < 28 {
		switch filter {
		case triage.FilterAll:
			return "All"
		case triage.FilterAuthored:
			return "Au"
		case triage.FilterAssigned:
			return "As"
		case triage.FilterReviewRequested:
			return "Rq"
		case triage.FilterStarred:
			return "St"
		case triage.FilterSnoozed:
			return "Sn"
		}
	}
	if width < 50 {
		switch filter {
		case triage.FilterAll:
			return "All"
		case triage.FilterAuthored:
			return "Auth"
		case triage.FilterAssigned:
			return "Asgn"
		case triage.FilterReviewRequested:
			return "Req"
		case triage.FilterStarred:
			return "Star"
		case triage.FilterSnoozed:
			return "Snooz"
		}
	}
	switch filter {
	case triage.FilterAll:
		return "All"
	case triage.FilterAuthored:
		return "Authored"
	case triage.FilterAssigned:
		return "Assigned"
	case triage.FilterReviewRequested:
		return "Review req"
	case triage.FilterStarred:
		return "Starred"
	case triage.FilterSnoozed:
		return "Snoozed"
	default:
		return filter.Label()
	}
}

func (m Model) filterTabs() string {
	filters := triage.Filters()
	tabs := make([]string, 0, len(filters))
	for _, filter := range filters {
		label := filterTabLabel(filter, m.width)
		role := roleNeutral
		if filter == m.filter {
			label = "[" + label + "]"
			role = roleFocus
		}
		tabs = append(tabs, m.visual.paint(role, label))
	}
	return m.visual.fit(strings.Join(tabs, " "), m.width)
}

func (m Model) fieldFor(pr domain.PR, name string) *domain.Field[any] {
	return m.fields[domain.FieldKey{Item: pr.ID, Name: name, Revision: pr.HeadSHA}]
}

func (m Model) freshField(pr domain.PR, name string) *domain.Field[any] {
	f := m.fieldFor(pr, name)
	if f == nil || !f.HasValue || f.Stale(m.opts.Clock.Now()) {
		return nil
	}
	return f
}

func (m Model) projection() triage.Sections {
	now := m.opts.Clock.Now()
	reminders := m.opts.Reminders.Snapshot()
	var inputs []triage.Input
	seen := make(map[domain.ItemID]bool)
	for _, section := range m.sections {
		if !section.field.HasValue {
			continue
		}
		for _, pr := range section.field.Value {
			seen[pr.ID] = true
			inputs = append(inputs, m.triageInput(pr, reminders[pr.ID]))
		}
	}
	for id, entry := range reminders {
		if seen[id] || entry.SnoozedUntil.IsZero() {
			continue
		}
		host := id.Host
		if host == "" {
			host = "github.com"
		}
		pr := domain.PR{ID: id, Title: "No longer open in the current list", URL: fmt.Sprintf("https://%s/%s/pull/%d", host, id.Repo, id.Number)}
		inputs = append(inputs, m.triageInput(pr, entry))
	}
	return triage.Project(inputs, m.filter, now)
}

func (m Model) triageInput(pr domain.PR, saved reminder.Entry) triage.Input {
	in := triage.Input{PR: pr, Starred: saved.Starred, SnoozedUntil: saved.SnoozedUntil}
	in.AuthoredByMe = authorSignal(pr.Author, m.viewer)
	if f := m.freshField(pr, "review_requests"); f != nil {
		if requests, ok := f.Value.(domain.LoginSet); ok {
			in.ReviewRequest = loginSignal(requests, m.viewer, "review_requests", f.Provenance.Version)
		}
	}
	in.Assigned = listAssignmentSignal(pr, m.viewer)
	if !in.Assigned.Known {
		if f := m.freshField(pr, "assignees"); f != nil {
			if assignees, ok := f.Value.(domain.LoginSet); ok {
				in.Assigned = loginSignal(assignees, m.viewer, "assignees", f.Provenance.Version)
			}
		}
	}
	if f := m.freshField(pr, "checks"); f != nil {
		if checks, ok := f.Value.(domain.CheckSummary); ok {
			in.Checks = checks
		}
	}
	if f := m.freshField(pr, "review_threads"); f != nil {
		if threads, ok := f.Value.(domain.ThreadSummary); ok {
			in.Threads = threads
		}
	}
	if f := m.freshField(pr, "reviews"); f != nil {
		if reviews, ok := f.Value.(domain.ReviewSet); ok {
			in.BotApproval = domain.BotApproval(reviews.Reviews, reviews.Complete, f.Provenance.Version)
		}
	}
	if f := m.fieldFor(pr, "activity"); f != nil {
		if f.HasValue {
			if activity, ok := f.Value.(domain.ActivityEvidence); ok {
				in.Activity = activity
				if f.Stale(m.opts.Clock.Now()) && activity.State == domain.ActivityActive {
					in.Activity.State = domain.ActivityStale
				}
				if in.Activity.State == domain.ActivityActive && len(in.Activity.Runs) == 0 && in.Activity.RunID != "" {
					in.Activity.Runs = []domain.ActivityRun{{
						RunID:     in.Activity.RunID,
						Name:      fallbackActivityRunName(in.Activity.Author),
						StartedAt: in.Activity.StartedAt,
					}}
				}
			}
		}
	}
	return in
}

func fallbackActivityRunName(author string) string {
	switch author {
	case "fullsend-ai-review":
		return "Review"
	case "fullsend-ai-coder":
		return "Code"
	case "fullsend-ai-triage":
		return "Triage"
	case "fullsend-ai-retro":
		return "Retro"
	case "fullsend-ai-prioritize":
		return "Prioritize"
	default:
		return "Fullsend agent"
	}
}

func loginSignal(set domain.LoginSet, viewer, source, version string) domain.DerivedBool {
	return assignmentSignal(set.Logins, set.Complete, viewer, source, version)
}

func assignmentSignal(logins []string, complete bool, viewer, source, version string) domain.DerivedBool {
	dependencies := []domain.Dependency{{Name: source, Version: version}, {Name: "viewer", Version: viewer}}
	if viewer == "" {
		return domain.DerivedBool{Dependencies: dependencies}
	}
	for _, login := range logins {
		if strings.EqualFold(login, viewer) {
			return domain.DerivedBool{Known: true, Value: true, Dependencies: dependencies}
		}
	}
	return domain.DerivedBool{Known: complete, Dependencies: dependencies}
}

func authorSignal(author, viewer string) domain.DerivedBool {
	dependencies := []domain.Dependency{{Name: "author", Version: author}, {Name: "viewer", Version: viewer}}
	if viewer == "" || author == "" {
		return domain.DerivedBool{Dependencies: dependencies}
	}
	return domain.DerivedBool{Known: true, Value: strings.EqualFold(author, viewer), Dependencies: dependencies}
}

func listAssignmentSignal(pr domain.PR, viewer string) domain.DerivedBool {
	if viewer == "" || (!pr.Assignees.Complete && len(pr.Assignees.Logins) == 0) {
		return domain.DerivedBool{}
	}
	return assignmentSignal(pr.Assignees.Logins, pr.Assignees.Complete, viewer, "list-assignees", pr.HeadSHA)
}

func (m Model) projectedRows() []domain.PR {
	sections := m.projection()
	rows := make([]domain.PR, 0, len(sections.Review)+len(sections.Active)+len(sections.Snoozed))
	for _, row := range sections.Review {
		rows = append(rows, row.PR)
	}
	for _, row := range sections.Active {
		rows = append(rows, row.PR)
	}
	for _, row := range sections.Snoozed {
		rows = append(rows, row.PR)
	}
	return rows
}

func (m *Model) selectedFields(force bool) []tea.Cmd {
	if m.opts.Enricher == nil {
		return nil
	}
	pr, ok := m.selectedPR()
	if !ok {
		return append(m.startFieldFetches(), m.startActivityFetches()...)
	}
	var selectedKeys []domain.FieldKey
	for _, name := range triageFieldNames {
		key := domain.FieldKey{Item: pr.ID, Name: name, Revision: pr.HeadSHA}
		if name == "activity" {
			m.queueActivity(key, force, true)
			continue
		}
		if name == "assignees" && listAssignmentSignal(pr, m.viewer).Known {
			continue
		}
		m.queueField(key, force)
		selectedKeys = append(selectedKeys, key)
	}
	for _, key := range selectedKeys {
		m.promoteField(key)
	}
	cmds := m.startFieldFetches()
	cmds = append(cmds, m.startActivityFetches()...)
	return cmds
}

func (m Model) isTableSummaryField(name string) bool {
	for _, summaryName := range tableSummaryFieldNames {
		if name == summaryName {
			return true
		}
	}
	return false
}

func (m *Model) prioritizeQueuedFields() {
	m.prioritizeQueuedFieldsFor(m.interactivePRs())
}

func (m *Model) prioritizeQueuedFieldsFor(interactive interactivePRSet) {
	if len(m.fieldQueue)-m.fieldQueueHead < 2 {
		return
	}
	type entry struct {
		key        domain.FieldKey
		generation uint64
	}
	pending := make([]entry, len(m.fieldQueue)-m.fieldQueueHead)
	byKey := make(map[domain.FieldKey]entry, len(pending))
	for i := range pending {
		pending[i] = entry{key: m.fieldQueue[m.fieldQueueHead+i], generation: m.fieldQueueTokens[m.fieldQueueHead+i]}
		byKey[pending[i].key] = pending[i]
	}
	priority := make([]entry, 0, len(pending))
	seen := make(map[domain.FieldKey]bool, len(m.fieldQueue))
	add := func(key domain.FieldKey) {
		if !m.fieldQueued[key] || seen[key] {
			return
		}
		if candidate, ok := byKey[key]; ok {
			priority = append(priority, candidate)
			seen[key] = true
		}
	}
	if interactive.hasSelected {
		for _, key := range m.fieldQueue {
			if key.Item == interactive.selected.ID && key.Revision == interactive.selected.HeadSHA {
				add(key)
			}
		}
	}
	for _, pr := range interactive.visible {
		for _, name := range tableSummaryFieldNames {
			add(domain.FieldKey{Item: pr.ID, Name: name, Revision: pr.HeadSHA})
		}
	}
	for _, candidate := range pending {
		if m.fieldQueued[candidate.key] && !seen[candidate.key] {
			priority = append(priority, candidate)
			seen[candidate.key] = true
		}
	}
	for i, candidate := range priority {
		m.fieldQueue[m.fieldQueueHead+i] = candidate.key
		m.fieldQueueTokens[m.fieldQueueHead+i] = candidate.generation
	}
}

func (m Model) visibleRowsForPrefetch() []domain.PR {
	return m.visibleRowsForPrefetchFrom(m.projection())
}

func (m Model) visibleRowsForPrefetchFrom(sections triage.Sections) []domain.PR {
	type rowPosition struct {
		pr   domain.PR
		line int
	}
	var positioned []rowPosition
	selectedLine := -1
	contentLine := 0
	if m.showingSnoozed() {
		contentLine = 1 // Table header.
		for _, row := range sections.Snoozed {
			if row.PR.ID == m.selected {
				selectedLine = contentLine
			}
			positioned = append(positioned, rowPosition{pr: row.PR, line: contentLine})
			contentLine++
		}
	} else {
		contentLine = 2 + len(m.sections) // Review heading, repository summaries, table header.
		for _, row := range sections.Review {
			if row.PR.ID == m.selected {
				selectedLine = contentLine
			}
			positioned = append(positioned, rowPosition{pr: row.PR, line: contentLine})
			contentLine++
		}
		if len(sections.Active) > 0 {
			contentLine += 2 // Active heading and repeated table header.
			for _, row := range sections.Active {
				if row.PR.ID == m.selected {
					selectedLine = contentLine
				}
				positioned = append(positioned, rowPosition{pr: row.PR, line: contentLine})
				contentLine++
			}
		}
	}
	if len(positioned) == 0 {
		return nil
	}
	scope := "no repository configured"
	if len(m.opts.Repos) > 0 {
		scope = strings.Join(m.opts.Repos, ", ")
	}
	lineCount := len(m.visual.header("fs-dash · Pull requests", scope, m.width)) + 1
	selected, hasSelected := selectedPRFromSections(sections, m.selected)
	if hasSelected {
		var selectedRow triage.Row
		if m.showingSnoozed() {
			for _, row := range sections.Snoozed {
				if row.PR.ID == selected.ID {
					selectedRow = row
					break
				}
			}
		} else {
			for _, row := range sections.Review {
				if row.PR.ID == selected.ID {
					selectedRow = row
					break
				}
			}
			for _, row := range sections.Active {
				if row.PR.ID == selected.ID {
					selectedRow = row
					break
				}
			}
		}
		lineCount += len(m.selectedContext(selected, selectedRow))
	}
	statusLines := 0
	if m.status != "" {
		statusLines++
	}
	if m.reminderWarning != "" {
		statusLines++
	}
	available := max(1, m.height-lineCount-3-statusLines)
	start := 0
	if selectedLine >= 0 {
		start = max(0, selectedLine-available/2)
		if start+available > contentLine {
			start = max(0, contentLine-available)
		}
	}
	end := min(contentLine, start+available)
	visible := make([]domain.PR, 0, end-start)
	for _, row := range positioned {
		if row.line >= start && row.line < end {
			visible = append(visible, row.pr)
		}
	}
	return visible
}

func (m *Model) promoteField(key domain.FieldKey) {
	if !m.fieldQueued[key] {
		return
	}
	if m.fieldPriorityGeneration == nil {
		m.fieldPriorityGeneration = map[domain.FieldKey]uint64{}
	}
	generation := m.fieldQueueGeneration[key]
	if m.fieldPriorityGeneration[key] == generation {
		return
	}
	m.fieldPriorityGeneration[key] = generation
	m.fieldPriorityQueue = append(m.fieldPriorityQueue, queueRef{key: key, generation: generation})
}

func (m *Model) queueField(key domain.FieldKey, force bool) {
	if m.opts.Enricher == nil || key.Name == "activity" {
		return
	}
	if _, active := m.fieldInFlight[key]; active {
		return
	}
	if m.fieldQueued[key] {
		return
	}
	f := m.fields[key]
	if f == nil {
		fresh := domain.NewField[any]()
		f = &fresh
		m.fields[key] = f
	} else if f.Phase.Active() {
		return
	} else if !force {
		now := m.opts.Clock.Now()
		if (f.Phase == domain.Cooldown && now.Before(f.RetryAt)) || (f.Phase == domain.Ready && !f.Stale(now)) {
			return
		}
	}
	f.Queue()
	if m.fieldQueued == nil {
		m.fieldQueued = map[domain.FieldKey]bool{}
	}
	if m.fieldQueueGeneration == nil {
		m.fieldQueueGeneration = map[domain.FieldKey]uint64{}
	}
	if m.fieldQueueHead == len(m.fieldQueue) {
		m.fieldQueue = m.fieldQueue[:0]
		m.fieldQueueTokens = m.fieldQueueTokens[:0]
		m.fieldQueueHead = 0
	}
	generation := m.fieldQueueGeneration[key] + 1
	m.fieldQueueGeneration[key] = generation
	m.fieldQueued[key] = true
	m.fieldQueue = append(m.fieldQueue, key)
	m.fieldQueueTokens = append(m.fieldQueueTokens, generation)
}

func (m *Model) startFieldFetches() []tea.Cmd {
	var cmds []tea.Cmd
	if !m.hasPendingFieldQueue() || len(m.fieldRequestsInFlight) >= maxConcurrentFieldFetches {
		return nil
	}
	interactive := m.interactivePRs()
	m.promoteInteractiveFields(interactive)
	budget := maxQueueItemsPerUpdate
	blocked := false
	for len(m.fieldRequestsInFlight) < maxConcurrentFieldFetches && budget > 0 {
		ref, priority, ok := m.nextFieldCandidate(interactive, &budget)
		if !ok {
			break
		}
		background := !interactive.contains(ref.key)
		if background {
			if m.hasInteractiveFieldWork(interactive) || m.hasInteractiveActivityWork(interactive) || m.hasOffscreenEnrichmentInFlight(interactive) {
				blocked = true
				break
			}
			if !m.fieldOffscreenWakeReady {
				if !m.fieldOffscreenWakeRunning {
					m.fieldOffscreenWakeRunning = true
					m.fieldOffscreenGeneration++
					generation := m.fieldOffscreenGeneration
					cmds = append(cmds, tea.Tick(offscreenFieldPace, func(time.Time) tea.Msg { return fieldOffscreenWakeMsg{generation: generation} }))
				}
				blocked = true
				break
			}
			m.fieldOffscreenWakeReady = false
		}
		m.consumeFieldCandidate(ref, priority)
		if _, ok := m.opts.Enricher.(ports.BatchEnricher); ok && combinedSummaryField(ref.key.Name) {
			if cmd := m.startCombinedSummary(ref.key); cmd != nil {
				cmds = append(cmds, cmd)
			}
			if background {
				break
			}
			continue
		}
		f := m.fields[ref.key]
		if f == nil || f.Phase != domain.Queued {
			continue
		}
		generation := f.Generation
		if !f.Begin(generation) {
			continue
		}
		m.fieldInFlight[ref.key] = generation
		requestKey := ref.key.String()
		m.fieldRequestForKey[ref.key] = requestKey
		m.fieldRequestsInFlight[requestKey] = 1
		attempt := m.fieldAttempts[ref.key] + 1
		if attempt < 1 {
			attempt = 1
		}
		m.fieldAttempts[ref.key] = attempt
		cmds = append(cmds, m.fetchField(ref.key, generation, attempt))
		if background {
			break
		}
	}
	if budget == 0 && !blocked && len(m.fieldRequestsInFlight) < maxConcurrentFieldFetches && m.hasPendingFieldQueue() {
		if !m.fieldQueueChunkRunning {
			m.fieldQueueChunkRunning = true
			cmds = append(cmds, tea.Tick(queueContinuationDelay, func(time.Time) tea.Msg { return fieldQueueChunkMsg{} }))
		}
	}
	return cmds
}

func (m Model) hasPendingFieldQueue() bool {
	return m.fieldPriorityHead < len(m.fieldPriorityQueue) || m.fieldQueueHead < len(m.fieldQueue)
}

func (m *Model) promoteInteractiveFields(interactive interactivePRSet) {
	if interactive.hasSelected {
		for _, name := range triageFieldNames {
			if name != "activity" {
				m.promoteField(domain.FieldKey{Item: interactive.selected.ID, Name: name, Revision: interactive.selected.HeadSHA})
			}
		}
	}
	for _, pr := range interactive.visible {
		for _, name := range tableSummaryFieldNames {
			m.promoteField(domain.FieldKey{Item: pr.ID, Name: name, Revision: pr.HeadSHA})
		}
	}
}

func (m *Model) nextFieldCandidate(interactive interactivePRSet, budget *int) (queueRef, bool, bool) {
	for m.fieldPriorityHead < len(m.fieldPriorityQueue) && *budget > 0 {
		ref := m.fieldPriorityQueue[m.fieldPriorityHead]
		if !m.fieldQueued[ref.key] || m.fieldQueueGeneration[ref.key] != ref.generation {
			m.fieldPriorityHead++
			*budget = *budget - 1
			if m.fieldPriorityGeneration[ref.key] == ref.generation {
				delete(m.fieldPriorityGeneration, ref.key)
			}
			continue
		}
		if !interactive.contains(ref.key) {
			m.fieldPriorityHead++
			*budget = *budget - 1
			if m.fieldPriorityGeneration[ref.key] == ref.generation {
				delete(m.fieldPriorityGeneration, ref.key)
			}
			continue
		}
		*budget = *budget - 1
		return ref, true, true
	}
	for m.fieldQueueHead < len(m.fieldQueue) && *budget > 0 {
		ref := queueRef{key: m.fieldQueue[m.fieldQueueHead], generation: m.fieldQueueTokens[m.fieldQueueHead]}
		if !m.fieldQueued[ref.key] || m.fieldQueueGeneration[ref.key] != ref.generation {
			m.fieldQueueHead++
			*budget = *budget - 1
			continue
		}
		if !m.eligibleField(ref.key, interactive) {
			m.cancelQueuedField(ref)
			m.fieldQueueHead++
			*budget = *budget - 1
			continue
		}
		*budget = *budget - 1
		return ref, false, true
	}
	if m.fieldQueueHead == len(m.fieldQueue) {
		m.fieldQueue = m.fieldQueue[:0]
		m.fieldQueueTokens = m.fieldQueueTokens[:0]
		m.fieldQueueHead = 0
	}
	if m.fieldPriorityHead == len(m.fieldPriorityQueue) {
		m.fieldPriorityQueue = m.fieldPriorityQueue[:0]
		m.fieldPriorityHead = 0
	}
	return queueRef{}, false, false
}

func (m Model) eligibleField(key domain.FieldKey, interactive interactivePRSet) bool {
	if interactive.contains(key) {
		return true
	}
	if !m.isTableSummaryField(key.Name) {
		return false
	}
	_, ok := interactive.listed[prRevision{item: key.Item, revision: key.Revision}]
	return ok
}

func (m *Model) cancelQueuedField(ref queueRef) {
	if m.fieldQueueGeneration[ref.key] != ref.generation {
		return
	}
	delete(m.fieldQueued, ref.key)
	if m.fieldPriorityGeneration[ref.key] == ref.generation {
		delete(m.fieldPriorityGeneration, ref.key)
	}
	if field := m.fields[ref.key]; field != nil && field.Phase == domain.Queued {
		field.Cancel(field.Generation)
	}
}

func (m *Model) consumeFieldCandidate(ref queueRef, priority bool) {
	if priority {
		m.fieldPriorityHead++
		if m.fieldPriorityGeneration[ref.key] == ref.generation {
			delete(m.fieldPriorityGeneration, ref.key)
		}
	} else {
		m.fieldQueueHead++
	}
	delete(m.fieldQueued, ref.key)
}

func (m Model) hasInteractiveFieldWork(interactive interactivePRSet) bool {
	if interactive.hasSelected {
		for _, name := range triageFieldNames {
			key := domain.FieldKey{Item: interactive.selected.ID, Name: name, Revision: interactive.selected.HeadSHA}
			if name != "activity" && m.fieldQueued[key] {
				return true
			}
		}
	}
	for _, pr := range interactive.visible {
		for _, name := range tableSummaryFieldNames {
			if m.fieldQueued[domain.FieldKey{Item: pr.ID, Name: name, Revision: pr.HeadSHA}] {
				return true
			}
		}
	}
	for key := range m.fieldInFlight {
		if interactive.contains(key) {
			return true
		}
	}
	return false
}

func (m Model) hasOffscreenEnrichmentInFlight(interactive interactivePRSet) bool {
	for key := range m.fieldInFlight {
		if !interactive.contains(key) {
			return true
		}
	}
	for key := range m.activityInFlight {
		if !interactive.contains(key) {
			return true
		}
	}
	return false
}

func combinedSummaryField(name string) bool {
	switch name {
	case "checks", "reviews", "review_requests", "review_threads":
		return true
	default:
		return false
	}
}

func (m *Model) startCombinedSummary(first domain.FieldKey) tea.Cmd {
	requestKey := "summary:" + first.Item.String() + ":" + first.Revision
	var keys []domain.FieldKey
	var generations []uint64
	var attempts []int
	for _, name := range []string{"checks", "reviews", "review_requests", "review_threads"} {
		key := domain.FieldKey{Item: first.Item, Name: name, Revision: first.Revision}
		if key != first {
			m.queueField(key, false)
		}
		field := m.fields[key]
		if field == nil || field.Phase != domain.Queued {
			continue
		}
		generation := field.Generation
		if !field.Begin(generation) {
			continue
		}
		delete(m.fieldQueued, key)
		if m.fieldPriorityGeneration[key] == m.fieldQueueGeneration[key] {
			delete(m.fieldPriorityGeneration, key)
		}
		attempt := m.fieldAttempts[key] + 1
		if attempt < 1 {
			attempt = 1
		}
		m.fieldAttempts[key] = attempt
		m.fieldInFlight[key] = generation
		m.fieldRequestForKey[key] = requestKey
		keys = append(keys, key)
		generations = append(generations, generation)
		attempts = append(attempts, attempt)
	}
	if len(keys) == 0 {
		return nil
	}
	m.fieldRequestsInFlight[requestKey] = len(keys)
	return m.fetchFields(keys, generations, attempts, requestKey)
}

func (m *Model) enqueueListPrefetch(repo string, generation uint64, prs []domain.PR) tea.Cmd {
	if m.opts.Enricher == nil || len(prs) == 0 {
		return nil
	}
	if m.prefetchBatchHead == len(m.prefetchBatches) {
		m.prefetchBatches = m.prefetchBatches[:0]
		m.prefetchBatchHead = 0
		m.prefetchPRIndex = 0
	}
	m.prefetchBatches = append(m.prefetchBatches, listPrefetchBatch{repo: repo, generation: generation, prs: prs})
	if m.prefetchChunkRunning {
		return nil
	}
	return m.processListPrefetchChunk()
}

func (m *Model) processListPrefetchChunk() tea.Cmd {
	processed := 0
	for m.prefetchBatchHead < len(m.prefetchBatches) && processed < maxQueueItemsPerUpdate {
		batch := m.prefetchBatches[m.prefetchBatchHead]
		section := m.section(batch.repo)
		if section == nil || section.field.Generation != batch.generation || section.field.Phase != domain.Ready {
			m.prefetchBatchHead++
			m.prefetchPRIndex = 0
			processed++
			continue
		}
		if m.prefetchPRIndex >= len(batch.prs) {
			m.prefetchBatchHead++
			m.prefetchPRIndex = 0
			continue
		}
		pr := batch.prs[m.prefetchPRIndex]
		m.prefetchPRIndex++
		processed++
		for _, name := range []string{"last_touched", "checks", "reviews", "review_threads"} {
			m.queueField(domain.FieldKey{Item: pr.ID, Name: name, Revision: pr.HeadSHA}, false)
		}
		if !pr.Assignees.Complete {
			m.queueField(domain.FieldKey{Item: pr.ID, Name: "assignees", Revision: pr.HeadSHA}, false)
		}
		m.queueActivity(domain.FieldKey{Item: pr.ID, Name: "activity", Revision: pr.HeadSHA}, false, false)
		if m.prefetchPRIndex == len(batch.prs) {
			m.prefetchBatchHead++
			m.prefetchPRIndex = 0
		}
	}
	if m.prefetchBatchHead == len(m.prefetchBatches) {
		m.prefetchBatches = m.prefetchBatches[:0]
		m.prefetchBatchHead = 0
		m.prefetchPRIndex = 0
		m.prefetchChunkRunning = false
		return nil
	}
	m.prefetchChunkRunning = true
	return tea.Tick(queueContinuationDelay, func(time.Time) tea.Msg { return listPrefetchChunkMsg{} })
}

func (m *Model) queueActivity(key domain.FieldKey, force, prioritize bool) {
	if m.opts.Enricher == nil || key.Name != "activity" {
		return
	}
	if _, active := m.activityInFlight[key]; active {
		return
	}
	if m.activityQueued[key] {
		if prioritize {
			m.promoteActivity(key)
		}
		return
	}
	f := m.fields[key]
	if f == nil {
		fresh := domain.NewField[any]()
		f = &fresh
		m.fields[key] = f
	} else if f.Phase.Active() {
		return
	} else if !force {
		now := m.opts.Clock.Now()
		if (f.Phase == domain.Cooldown && now.Before(f.RetryAt)) || (f.Phase == domain.Ready && !f.Stale(now)) {
			return
		}
	}
	f.Queue()
	attempt := m.fieldAttempts[key] + 1
	if attempt < 1 {
		attempt = 1
	}
	m.fieldAttempts[key] = attempt
	if m.activityQueued == nil {
		m.activityQueued = map[domain.FieldKey]bool{}
	}
	if m.activityQueueGeneration == nil {
		m.activityQueueGeneration = map[domain.FieldKey]uint64{}
	}
	if m.activityQueueHead == len(m.activityQueue) {
		m.activityQueue = m.activityQueue[:0]
		m.activityQueueTokens = m.activityQueueTokens[:0]
		m.activityQueueHead = 0
	}
	generation := m.activityQueueGeneration[key] + 1
	m.activityQueueGeneration[key] = generation
	m.activityQueued[key] = true
	m.activityQueue = append(m.activityQueue, key)
	m.activityQueueTokens = append(m.activityQueueTokens, generation)
	if prioritize {
		m.promoteActivity(key)
	}
}

func (m *Model) promoteActivity(key domain.FieldKey) {
	if !m.activityQueued[key] {
		return
	}
	if m.activityPriorityGeneration == nil {
		m.activityPriorityGeneration = map[domain.FieldKey]uint64{}
	}
	generation := m.activityQueueGeneration[key]
	if m.activityPriorityGeneration[key] == generation {
		return
	}
	m.activityPriorityGeneration[key] = generation
	m.activityPriorityQueue = append(m.activityPriorityQueue, queueRef{key: key, generation: generation})
}

func (m *Model) startActivityFetches() []tea.Cmd {
	var cmds []tea.Cmd
	if !m.hasPendingActivityQueue() || len(m.activityInFlight) >= maxConcurrentActivityFetches {
		return nil
	}
	interactive := m.interactivePRs()
	m.promoteInteractiveActivity(interactive)
	budget := maxQueueItemsPerUpdate
	blocked := false
	for len(m.activityInFlight) < maxConcurrentActivityFetches && budget > 0 {
		ref, priority, ok := m.nextActivityCandidate(interactive, &budget)
		if !ok {
			break
		}
		background := !interactive.contains(ref.key)
		if background {
			if m.hasInteractiveFieldWork(interactive) || m.hasInteractiveActivityWork(interactive) || m.hasOffscreenEnrichmentInFlight(interactive) {
				blocked = true
				break
			}
			if !m.activityOffscreenWakeReady {
				if !m.activityOffscreenWakeRunning {
					m.activityOffscreenWakeRunning = true
					m.activityOffscreenGeneration++
					generation := m.activityOffscreenGeneration
					cmds = append(cmds, tea.Tick(offscreenActivityPace, func(time.Time) tea.Msg { return activityOffscreenWakeMsg{generation: generation} }))
				}
				blocked = true
				break
			}
			m.activityOffscreenWakeReady = false
		}
		m.consumeActivityCandidate(ref, priority)
		f := m.fields[ref.key]
		if f == nil || f.Phase != domain.Queued {
			continue
		}
		generation := f.Generation
		if !f.Begin(generation) {
			continue
		}
		m.activityInFlight[ref.key] = generation
		attempt := m.fieldAttempts[ref.key]
		if attempt < 1 {
			attempt = 1
			m.fieldAttempts[ref.key] = attempt
		}
		cmds = append(cmds, m.fetchField(ref.key, generation, attempt))
		if background {
			break
		}
	}
	if budget == 0 && !blocked && len(m.activityInFlight) < maxConcurrentActivityFetches && m.hasPendingActivityQueue() {
		if !m.activityQueueChunkRunning {
			m.activityQueueChunkRunning = true
			cmds = append(cmds, tea.Tick(queueContinuationDelay, func(time.Time) tea.Msg { return activityQueueChunkMsg{} }))
		}
	}
	return cmds
}

func (m Model) hasPendingActivityQueue() bool {
	return m.activityPriorityHead < len(m.activityPriorityQueue) || m.activityQueueHead < len(m.activityQueue)
}

func (m *Model) promoteInteractiveActivity(interactive interactivePRSet) {
	if interactive.hasSelected {
		m.promoteActivity(domain.FieldKey{Item: interactive.selected.ID, Name: "activity", Revision: interactive.selected.HeadSHA})
	}
	for _, pr := range interactive.visible {
		m.promoteActivity(domain.FieldKey{Item: pr.ID, Name: "activity", Revision: pr.HeadSHA})
	}
}

func (m *Model) nextActivityCandidate(interactive interactivePRSet, budget *int) (queueRef, bool, bool) {
	for m.activityPriorityHead < len(m.activityPriorityQueue) && *budget > 0 {
		ref := m.activityPriorityQueue[m.activityPriorityHead]
		if !m.activityQueued[ref.key] || m.activityQueueGeneration[ref.key] != ref.generation || !interactive.contains(ref.key) {
			m.activityPriorityHead++
			*budget = *budget - 1
			if m.activityPriorityGeneration[ref.key] == ref.generation {
				delete(m.activityPriorityGeneration, ref.key)
			}
			continue
		}
		*budget = *budget - 1
		return ref, true, true
	}
	for m.activityQueueHead < len(m.activityQueue) && *budget > 0 {
		ref := queueRef{key: m.activityQueue[m.activityQueueHead], generation: m.activityQueueTokens[m.activityQueueHead]}
		if !m.activityQueued[ref.key] || m.activityQueueGeneration[ref.key] != ref.generation {
			m.activityQueueHead++
			*budget = *budget - 1
			continue
		}
		if !m.eligibleActivity(ref.key, interactive) {
			m.cancelQueuedActivity(ref)
			m.activityQueueHead++
			*budget = *budget - 1
			continue
		}
		*budget = *budget - 1
		return ref, false, true
	}
	if m.activityQueueHead == len(m.activityQueue) {
		m.activityQueue = m.activityQueue[:0]
		m.activityQueueTokens = m.activityQueueTokens[:0]
		m.activityQueueHead = 0
	}
	if m.activityPriorityHead == len(m.activityPriorityQueue) {
		m.activityPriorityQueue = m.activityPriorityQueue[:0]
		m.activityPriorityHead = 0
	}
	return queueRef{}, false, false
}

func (m Model) eligibleActivity(key domain.FieldKey, interactive interactivePRSet) bool {
	if interactive.contains(key) {
		return true
	}
	_, ok := interactive.listed[prRevision{item: key.Item, revision: key.Revision}]
	return ok
}

func (m *Model) cancelQueuedActivity(ref queueRef) {
	if m.activityQueueGeneration[ref.key] != ref.generation {
		return
	}
	delete(m.activityQueued, ref.key)
	if m.activityPriorityGeneration[ref.key] == ref.generation {
		delete(m.activityPriorityGeneration, ref.key)
	}
	if field := m.fields[ref.key]; field != nil && field.Phase == domain.Queued {
		field.Cancel(field.Generation)
	}
}

func (m *Model) consumeActivityCandidate(ref queueRef, priority bool) {
	if priority {
		m.activityPriorityHead++
		if m.activityPriorityGeneration[ref.key] == ref.generation {
			delete(m.activityPriorityGeneration, ref.key)
		}
	} else {
		m.activityQueueHead++
	}
	delete(m.activityQueued, ref.key)
}

func (m Model) hasInteractiveActivityWork(interactive interactivePRSet) bool {
	if interactive.hasSelected && m.activityQueued[domain.FieldKey{Item: interactive.selected.ID, Name: "activity", Revision: interactive.selected.HeadSHA}] {
		return true
	}
	for _, pr := range interactive.visible {
		if m.activityQueued[domain.FieldKey{Item: pr.ID, Name: "activity", Revision: pr.HeadSHA}] {
			return true
		}
	}
	for key := range m.activityInFlight {
		if interactive.contains(key) {
			return true
		}
	}
	return false
}

func (m *Model) scheduleWake() tea.Cmd {
	now := m.opts.Clock.Now()
	var earliest time.Time
	for _, entry := range m.opts.Reminders.Snapshot() {
		if entry.SnoozedUntil.After(now) && (earliest.IsZero() || entry.SnoozedUntil.Before(earliest)) {
			earliest = entry.SnoozedUntil
		}
	}
	if earliest.IsZero() {
		if m.wakeTickRunning {
			m.wakeGeneration++
		}
		m.wakeTickRunning, m.nextWake = false, time.Time{}
		return nil
	}
	if m.wakeTickRunning && earliest.Equal(m.nextWake) {
		return nil
	}
	m.wakeTickRunning = true
	m.nextWake = earliest
	m.wakeGeneration++
	generation := m.wakeGeneration
	delay := earliest.Sub(now)
	if delay < time.Millisecond {
		delay = time.Millisecond
	}
	return tea.Tick(delay, func(time.Time) tea.Msg { return wakeMsg{generation: generation} })
}

type listContentKind uint8

const (
	listContentText listContentKind = iota
	listContentTableHeader
	listContentPR
	listContentActivityRun
	listContentActivityNotice
)

type listContentLine struct {
	kind    listContentKind
	text    string
	row     triage.Row
	run     domain.ActivityRun
	lastRun bool
}

func (m Model) listView() string {
	sections := m.projection()
	scope := "no repository configured"
	if len(m.opts.Repos) > 0 {
		scope = strings.Join(m.opts.Repos, ", ")
	}
	lines := m.visual.header("fs-dash · Pull requests", scope, m.width)
	lines = append(lines, m.filterTabs())
	rowCount := len(sections.Review) + len(sections.Active)
	if m.showingSnoozed() {
		rowCount = len(sections.Snoozed)
	}
	content := make([]listContentLine, 0, rowCount+len(m.sections)+8)
	appendText := func(text string) { content = append(content, listContentLine{kind: listContentText, text: text}) }
	appendHeader := func() { content = append(content, listContentLine{kind: listContentTableHeader}) }
	appendPR := func(row triage.Row) { content = append(content, listContentLine{kind: listContentPR, row: row}) }
	appendRun := func(run domain.ActivityRun, last bool) {
		content = append(content, listContentLine{kind: listContentActivityRun, run: run, lastRun: last})
	}
	appendRunNotice := func() { content = append(content, listContentLine{kind: listContentActivityNotice}) }
	selectedLine := -1
	if m.showingSnoozed() {
		appendHeader()
		for _, row := range sections.Snoozed {
			if row.PR.ID == m.selected {
				selectedLine = len(content)
			}
			appendPR(row)
		}
	} else {
		appendText(m.visual.paint(roleAccent, "REVIEW"))
		if summaries := m.repositorySummaryLine(sections); summaries != "" {
			appendText(summaries)
		}
		appendHeader()
		for _, row := range sections.Review {
			if row.PR.ID == m.selected {
				selectedLine = len(content)
			}
			appendPR(row)
		}
		if len(sections.Active) > 0 {
			appendText(m.visual.paint(roleAccent, "ACTIVE FULLSEND AGENTS"))
			appendHeader()
			for _, row := range sections.Active {
				if row.PR.ID == m.selected {
					selectedLine = len(content)
				}
				appendPR(row)
				for i, run := range row.Activity.Runs {
					appendRun(run, i == len(row.Activity.Runs)-1 && row.Activity.RunsComplete)
				}
				if !row.Activity.RunsComplete {
					appendRunNotice()
				}
			}
		}
	}
	if rowCount == 0 {
		if m.showingSnoozed() {
			appendText(m.visual.paint(roleMuted, "No PRs are currently snoozed."))
		} else {
			appendText(m.visual.paint(roleMuted, "No PRs match this filter. Pending filter data remains visible when its result is unknown."))
		}
	}
	if len(m.opts.Repos) == 0 {
		appendText(m.visual.paint(roleMuted, "No GitHub repository found. Add repositories = [\"owner/name\"] to config.toml or run from a GitHub checkout."))
	}
	selectedRow, hasSelected := selectedRowFromSections(sections, m.selected)
	var context []string
	if hasSelected {
		context = m.selectedContext(selectedRow.PR, selectedRow)
	}
	statusLines := 1 // Quota status is always present, including the unknown state.
	if m.status != "" {
		statusLines++
	}
	if m.reminderWarning != "" {
		statusLines++
	}
	available := max(1, m.height-len(lines)-3-len(context)-statusLines)
	start := 0
	if selectedLine >= 0 {
		start = max(0, selectedLine-available/2)
		if start+available > len(content) {
			start = max(0, len(content)-available)
		}
	}
	end := min(len(content), start+available)
	columns, widths := m.tableColumns(visibleContentRows(content[start:end]), m.showingSnoozed())
	prIndent := tableIndentForPR(columns, widths)
	visibleContent := make([]string, 0, end-start)
	for _, line := range content[start:end] {
		switch line.kind {
		case listContentTableHeader:
			visibleContent = append(visibleContent, m.renderTableHeader(columns, widths))
		case listContentPR:
			visibleContent = append(visibleContent, m.renderTableRow(line.row, columns, widths, m.hasSelection && line.row.PR.ID == m.selected))
		case listContentActivityRun:
			visibleContent = append(visibleContent, m.activeRunLine(line.run, line.lastRun, prIndent))
		case listContentActivityNotice:
			visibleContent = append(visibleContent, m.incompleteActivityRunsLine(prIndent))
		default:
			visibleContent = append(visibleContent, line.text)
		}
	}
	lines = append(lines, m.visual.panel("PR attention", visibleContent, m.width)...)
	if hasSelected {
		lines = append(lines, context...)
	}
	if m.reminderWarning != "" {
		lines = append(lines, m.visual.paint(roleWarning, m.visual.fit(m.reminderWarning, m.width)))
	}
	lines = append(lines, m.quotaStatusLine())
	if m.status != "" {
		lines = append(lines, m.visual.paint(roleWarning, m.visual.fit(m.status, m.width)))
	}
	footer := m.tableModeHint() + " · h/l filters · j/k move · enter details · s star · z snooze · r refresh · L logs · ? help · q quit"
	if m.showingSnoozed() {
		footer = m.tableModeHint() + " · h/l filters · j/k move · s star · z snooze · u unsnooze · L logs · ? help · q quit"
	}
	lines = append(lines, m.visual.footer(footer, m.width))
	return strings.Join(lines, "\n") + "\n"
}

func visibleContentRows(lines []listContentLine) []triage.Row {
	rows := make([]triage.Row, 0, len(lines))
	for _, line := range lines {
		if line.kind == listContentPR {
			rows = append(rows, line.row)
		}
	}
	return rows
}

func tableIndentForPR(specs []tableColumnSpec, widths []int) int {
	for i, spec := range specs {
		if spec.column != columnPR {
			continue
		}
		indent := i // one separator space before each column after the first
		for j := 0; j < i && j < len(widths); j++ {
			indent += widths[j]
		}
		return indent
	}
	return 0
}

func (m Model) activeRunLine(run domain.ActivityRun, last bool, indent int) string {
	branch := "├─ "
	if last {
		branch = "└─ "
	}
	branch = strings.Repeat(" ", max(0, indent)) + branch
	name := strings.TrimSpace(run.Name)
	if name == "" {
		name = "Fullsend agent"
	}
	duration := formatRunElapsed(m.opts.Clock.Now().Sub(run.StartedAt))
	suffix := " · " + duration
	linkLabel := " · ↗ workflow"
	url := safeWorkflowURL(run.WorkflowURL)
	width := m.width
	if !m.visual.compact && m.width >= 52 {
		width -= 4
	}
	width = max(1, width)
	nameWidth := width - lipgloss.Width(branch) - lipgloss.Width(suffix)
	if url != "" {
		nameWidth -= lipgloss.Width(linkLabel)
	}
	if nameWidth < 1 && url != "" {
		url = ""
		nameWidth = width - lipgloss.Width(branch) - lipgloss.Width(suffix)
	}
	nameWidth = max(1, nameWidth)
	line := m.visual.paint(roleMuted, branch) + m.visual.paint(roleNeutral, m.visual.fit(name, nameWidth)) + m.visual.paint(roleAccent, suffix)
	if url != "" {
		line += ansi.SetHyperlink(url) + m.visual.paint(roleLink, linkLabel) + ansi.ResetHyperlink()
	}
	return m.visual.fit(line, width)
}

func (m Model) incompleteActivityRunsLine(indent int) string {
	width := m.width
	if !m.visual.compact && m.width >= 52 {
		width -= 4
	}
	branch := strings.Repeat(" ", max(0, indent)) + "└─ "
	return m.visual.fit(m.visual.paint(roleMuted, branch+"run history may be incomplete"), max(1, width))
}

func formatRunElapsed(elapsed time.Duration) string {
	seconds := int64(elapsed / time.Second)
	if seconds < 0 {
		seconds = 0
	}
	minutes := seconds / 60
	if minutes >= 60 {
		return fmt.Sprintf("%dh %02dm", minutes/60, minutes%60)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %02ds", minutes, seconds%60)
	}
	return fmt.Sprintf("%ds", seconds)
}

func safeWorkflowURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return ""
	}
	for _, r := range raw {
		if r < 0x20 || r == 0x7f {
			return ""
		}
	}
	return parsed.String()
}

func (m Model) View() string {
	if m.console {
		return m.consoleView()
	}
	if m.help {
		return m.helpView()
	}
	if m.mode == "details" {
		return m.detailsView()
	}
	if m.mode == "snooze" {
		return m.snoozeView()
	}
	return m.listView()
}

func (m Model) repositorySummaryLine(projected triage.Sections) string {
	var parts []string
	for _, section := range m.sections {
		state, role := m.sectionStatus(section, projected)
		parts = append(parts, m.visual.paint(roleMuted, section.repo+" · ")+m.visual.paint(role, state))
	}
	separator := m.visual.paint(roleMuted, "  │  ")
	return strings.Join(parts, separator)
}

func (m Model) quotaStatusLine() string {
	if m.quota == nil {
		label := "GitHub API quota: unknown"
		if m.visual.compact || m.width < 64 {
			label = "API quota: unknown"
		}
		return m.visual.paint(roleMuted, m.visual.fit(label, m.width))
	}
	observation := m.quota
	now := m.opts.Clock.Now()
	age := now.Sub(observation.ObservedAt)
	if age < 0 {
		age = 0
	}
	state := ""
	role := roleNeutral
	if age >= quotaStaleAfter {
		state = "stale"
		role = roleMuted
	}
	if observation.Remaining == 0 {
		if state != "" {
			state += " · "
		}
		state += "exhausted"
		role = roleWarning
	} else if int64(observation.Remaining)*5 < int64(observation.Limit) {
		if state != "" {
			state += " · "
		}
		state += "low"
		role = roleWarning
	}
	resource := "API"
	if strings.EqualFold(observation.Resource, "graphql") {
		resource = "GraphQL"
	} else if observation.Resource != "" {
		resource = strings.ToUpper(observation.Resource[:1]) + observation.Resource[1:]
	}
	ageText := "now"
	if age >= time.Minute {
		if age >= time.Hour {
			ageText = fmt.Sprintf("%dh", int(age/time.Hour))
		} else {
			ageText = fmt.Sprintf("%dm", int(age/time.Minute))
		}
	}
	resetText := quotaResetText(observation.ResetAt, now, false)
	stateText := "current"
	if state != "" {
		stateText = state
	}
	label := fmt.Sprintf("%s quota: %s · %d/%d left · %d used · %s · %s", resource, stateText, observation.Remaining, observation.Limit, observation.Used, resetText, ageText)
	if m.visual.compact || m.width < 64 {
		compactResource := resource
		if resource == "GraphQL" {
			compactResource = "GQL"
		}
		compactState := strings.ToUpper(stateText)
		resetText = quotaResetText(observation.ResetAt, now, true)
		label = fmt.Sprintf("%s %s · %d/%d left · %d used · %s · %s", compactResource, compactState, observation.Remaining, observation.Limit, observation.Used, resetText, ageText)
	}
	return m.visual.paint(role, m.visual.fit(label, m.width))
}

func quotaResetText(resetAt, now time.Time, compact bool) string {
	if resetAt.IsZero() {
		return "reset time unknown"
	}
	remaining := resetAt.Sub(now)
	if remaining <= 0 {
		return "reset now"
	}
	var unit time.Duration
	var fullUnit, shortUnit string
	switch {
	case remaining < time.Minute:
		unit, fullUnit, shortUnit = time.Second, "second", "s"
	case remaining < time.Hour:
		unit, fullUnit, shortUnit = time.Minute, "minute", "m"
	case remaining < 24*time.Hour:
		unit, fullUnit, shortUnit = time.Hour, "hour", "h"
	default:
		unit, fullUnit, shortUnit = 24*time.Hour, "day", "d"
	}
	count := int((remaining + unit - 1) / unit)
	if compact {
		return fmt.Sprintf("reset in %d%s", count, shortUnit)
	}
	if count != 1 {
		fullUnit += "s"
	}
	return fmt.Sprintf("reset in %d %s", count, fullUnit)
}

func (m Model) renderRow(row triage.Row) string {
	marker := " "
	if m.hasSelection && row.PR.ID == m.selected {
		marker = ">"
	}
	label := fmt.Sprint(row.PR.ID.Number)
	title := row.PR.ID.Repo + " · " + row.PR.Title
	status := m.rowStatus(row)
	lineWidth := m.width - 4
	if m.visual.compact || m.width < 52 {
		lineWidth = m.width
	}
	return m.visual.selected(marker, label, title, row.PR.Author, row.PR.UpdatedAt.Local().Format("Jan 02 15:04"), status, m.rowRole(row), max(1, lineWidth))
}

func (m Model) rowStatus(row triage.Row) string {
	var parts []string
	if row.Starred {
		parts = append(parts, "★ starred")
	}
	if row.FilterPending {
		parts = append(parts, "filter pending")
	}
	if !m.showingSnoozed() {
		for _, name := range triageFieldNames {
			if name == "activity" {
				continue
			}
			field := m.fieldFor(row.PR, name)
			if field != nil && (field.Phase == domain.Failed || field.Phase == domain.Cooldown) {
				parts = append(parts, name+" failed")
			} else if field != nil && field.Stale(m.opts.Clock.Now()) {
				parts = append(parts, name+" stale")
			}
		}
		field := m.fieldFor(row.PR, "activity")
		switch {
		case field == nil:
			parts = append(parts, "activity pending")
		case field.Phase.Active():
			parts = append(parts, m.spinner()+" activity pending")
		case field.Phase == domain.Failed || field.Phase == domain.Cooldown:
			parts = append(parts, "activity failed")
		case field.Stale(m.opts.Clock.Now()):
			parts = append(parts, "activity stale · refresh")
		case field.Phase == domain.Canceled:
			parts = append(parts, "activity canceled")
		case row.Activity.State == domain.ActivityActive:
			parts = append(parts, "agent active · "+row.Activity.Author+" · "+row.Activity.StartedAt.Local().Format("15:04"))
		case row.Activity.State == domain.ActivityStale:
			parts = append(parts, "agent status stale · "+row.Activity.StartedAt.Local().Format("Jan 02 15:04"))
		case row.Activity.State == domain.ActivityUnknown:
			parts = append(parts, "activity unknown")
		case row.Activity.State == domain.ActivityInactive:
			parts = append(parts, "agent inactive")
		}
	}
	if row.SnoozedUntil.After(m.opts.Clock.Now()) {
		parts = append(parts, "wake "+row.SnoozedUntil.Local().Format("Jan 02 15:04"))
	}
	return strings.Join(parts, " · ")
}

func (m Model) rowRole(row triage.Row) visualRole {
	field := m.fieldFor(row.PR, "activity")
	if field != nil && (field.Phase == domain.Failed || field.Phase == domain.Cooldown) {
		return roleFailure
	}
	if field != nil && field.Phase == domain.Canceled {
		return roleCanceled
	}
	if row.Activity.State == domain.ActivityActive {
		return roleSuccess
	}
	if field != nil && field.Stale(m.opts.Clock.Now()) || row.Activity.State == domain.ActivityStale {
		return roleStale
	}
	if field != nil && field.Phase.Active() {
		return rolePending
	}
	return roleMuted
}

func (m Model) selectedContext(pr domain.PR, row triage.Row) []string {
	lines := []string{m.visual.paint(roleMuted, "Selected: ") + m.visual.paint(roleLink, pr.URL)}
	if row.Starred {
		lines = append(lines, m.visual.paint(roleSuccess, "★ starred locally"))
	}
	if len(row.Reasons) > 0 {
		parts := make([]string, 0, len(row.Reasons))
		for _, reason := range row.Reasons {
			parts = append(parts, fmt.Sprintf("%s %+d", reason.Code, reason.Points))
		}
		lines = append(lines, m.visual.paint(roleAccent, "Priority: "+strings.Join(parts, " · ")))
	}
	if row.Activity.URL != "" {
		lines = append(lines, m.visual.paint(roleMuted, fmt.Sprintf("Fullsend evidence: %s · %s", row.Activity.Author, row.Activity.StartedAt.Local().Format(time.RFC822))))
		lines = append(lines, m.visual.paint(roleLink, m.visual.fit(row.Activity.URL, m.width-4)))
	}
	return m.visual.context("Selected PR", lines, m.width)
}

func (m Model) snoozeView() string {
	content := []string{"Snooze this PR locally", "1  one hour", "2  one day", "3  one week", "esc cancel"}
	lines := append(m.visual.header("fs-dash · Snooze", "Choose a wake time", m.width), m.visual.panel("Snooze duration", content, m.width)...)
	lines = append(lines, m.visual.footer("1 one hour · 2 one day · 3 one week · esc cancel", m.width))
	return strings.Join(lines, "\n") + "\n"
}
