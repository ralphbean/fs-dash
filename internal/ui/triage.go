package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fullsend-ai/fs-dash/internal/domain"
	"github.com/fullsend-ai/fs-dash/internal/reminder"
	"github.com/fullsend-ai/fs-dash/internal/triage"
)

var triageFieldNames = []string{"reviews", "review_requests", "assignees", "checks", "review_threads", "last_touched", "activity"}

const maxConcurrentActivityFetches = 5
const maxConcurrentFieldFetches = 2

var tableSummaryFieldNames = []string{"last_touched", "checks", "reviews", "review_threads", "assignees"}

func (m Model) showingSnoozed() bool {
	return m.filter == triage.FilterSnoozed
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
			}
		}
	}
	return in
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
	m.discardQueuedFieldsExcept(pr.ID, pr.HeadSHA)
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
	for i := len(selectedKeys) - 1; i >= 0; i-- {
		m.promoteField(selectedKeys[i])
	}
	m.prioritizeQueuedFields()
	cmds := m.startFieldFetches()
	cmds = append(cmds, m.startActivityFetches()...)
	return cmds
}

func (m *Model) discardQueuedFieldsExcept(item domain.ItemID, revision string) {
	kept := m.fieldQueue[:0]
	for _, key := range m.fieldQueue {
		keepSummary := m.isTableSummaryField(key.Name) && m.hasListedPR(key)
		keepSelected := key.Item == item && key.Revision == revision
		if keepSummary || keepSelected {
			kept = append(kept, key)
			continue
		}
		delete(m.fieldQueued, key)
		if f := m.fields[key]; f != nil && f.Phase == domain.Queued {
			f.Cancel(f.Generation)
		}
	}
	m.fieldQueue = kept
}

func (m Model) isTableSummaryField(name string) bool {
	for _, summaryName := range tableSummaryFieldNames {
		if name == summaryName {
			return true
		}
	}
	return false
}

func (m Model) hasListedPR(key domain.FieldKey) bool {
	for _, section := range m.sections {
		if !section.field.HasValue {
			continue
		}
		for _, pr := range section.field.Value {
			if pr.ID == key.Item && pr.HeadSHA == key.Revision {
				return true
			}
		}
	}
	return false
}

func (m *Model) prioritizeQueuedFields() {
	if len(m.fieldQueue) < 2 {
		return
	}
	priority := make([]domain.FieldKey, 0, len(m.fieldQueue))
	seen := make(map[domain.FieldKey]bool, len(m.fieldQueue))
	add := func(key domain.FieldKey) {
		if m.fieldQueued[key] && !seen[key] {
			priority = append(priority, key)
			seen[key] = true
		}
	}
	if pr, ok := m.selectedPR(); ok {
		for _, key := range m.fieldQueue {
			if key.Item == pr.ID && key.Revision == pr.HeadSHA {
				add(key)
			}
		}
	}
	for _, pr := range m.visibleRowsForPrefetch() {
		for _, name := range tableSummaryFieldNames {
			add(domain.FieldKey{Item: pr.ID, Name: name, Revision: pr.HeadSHA})
		}
	}
	for _, key := range m.fieldQueue {
		add(key)
	}
	m.fieldQueue = priority
}

func (m Model) visibleRowsForPrefetch() []domain.PR {
	sections := m.projection()
	type rowPosition struct {
		pr   domain.PR
		line int
	}
	var positioned []rowPosition
	selectedLine := -1
	contentLine := 0
	if m.showingSnoozed() {
		contentLine = 2 // Snoozed heading and table header.
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
	selected, hasSelected := m.selectedPR()
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
	for i, queued := range m.fieldQueue {
		if queued == key {
			copy(m.fieldQueue[1:i+1], m.fieldQueue[:i])
			m.fieldQueue[0] = key
			return
		}
	}
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
	} else if !force && (f.Phase == domain.Cooldown || (f.Phase == domain.Ready && !f.Stale(m.opts.Clock.Now()))) {
		return
	}
	f.Queue()
	if m.fieldQueued == nil {
		m.fieldQueued = map[domain.FieldKey]bool{}
	}
	m.fieldQueued[key] = true
	m.fieldQueue = append(m.fieldQueue, key)
}

func (m *Model) startFieldFetches() []tea.Cmd {
	var cmds []tea.Cmd
	for len(m.fieldInFlight) < maxConcurrentFieldFetches && len(m.fieldQueue) > 0 {
		key := m.fieldQueue[0]
		m.fieldQueue = m.fieldQueue[1:]
		delete(m.fieldQueued, key)
		f := m.fields[key]
		if f == nil || f.Phase != domain.Queued {
			continue
		}
		generation := f.Generation
		if !f.Begin(generation) {
			continue
		}
		m.fieldInFlight[key] = generation
		attempt := m.fieldAttempts[key] + 1
		if attempt < 1 {
			attempt = 1
		}
		m.fieldAttempts[key] = attempt
		cmds = append(cmds, m.fetchField(key, generation, attempt))
	}
	return cmds
}

func (m *Model) enqueueListActivity(prs []domain.PR) {
	if m.opts.Enricher == nil {
		return
	}
	for _, pr := range prs {
		m.queueActivity(domain.FieldKey{Item: pr.ID, Name: "activity", Revision: pr.HeadSHA}, false, false)
	}
}

func (m *Model) enqueueListSummaries(prs []domain.PR) {
	if m.opts.Enricher == nil {
		return
	}
	for _, pr := range prs {
		for _, name := range []string{"last_touched", "checks", "reviews", "review_threads"} {
			m.queueField(domain.FieldKey{Item: pr.ID, Name: name, Revision: pr.HeadSHA}, false)
		}
		if !pr.Assignees.Complete {
			m.queueField(domain.FieldKey{Item: pr.ID, Name: "assignees", Revision: pr.HeadSHA}, false)
		}
	}
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
	} else if !force && (f.Phase == domain.Cooldown || (f.Phase == domain.Ready && !f.Stale(m.opts.Clock.Now()))) {
		return
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
	m.activityQueued[key] = true
	m.activityQueue = append(m.activityQueue, key)
	if prioritize {
		m.promoteActivity(key)
	}
}

func (m *Model) promoteActivity(key domain.FieldKey) {
	if !m.activityQueued[key] {
		return
	}
	for i, queued := range m.activityQueue {
		if queued == key {
			copy(m.activityQueue[1:i+1], m.activityQueue[:i])
			m.activityQueue[0] = key
			return
		}
	}
}

func (m *Model) startActivityFetches() []tea.Cmd {
	var cmds []tea.Cmd
	for len(m.activityInFlight) < maxConcurrentActivityFetches && len(m.activityQueue) > 0 {
		key := m.activityQueue[0]
		m.activityQueue = m.activityQueue[1:]
		delete(m.activityQueued, key)
		f := m.fields[key]
		if f == nil || f.Phase != domain.Queued {
			continue
		}
		generation := f.Generation
		if !f.Begin(generation) {
			continue
		}
		m.activityInFlight[key] = generation
		attempt := m.fieldAttempts[key]
		if attempt < 1 {
			attempt = 1
			m.fieldAttempts[key] = attempt
		}
		cmds = append(cmds, m.fetchField(key, generation, attempt))
	}
	return cmds
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

func (m Model) listView() string {
	sections := m.projection()
	scope := "no repository configured"
	if len(m.opts.Repos) > 0 {
		scope = strings.Join(m.opts.Repos, ", ")
	}
	lines := m.visual.header("fs-dash · Pull requests", scope, m.width)
	if m.showingSnoozed() {
		lines = append(lines, m.visual.paint(roleAccent, m.visual.fit("Snoozed PRs · f cycle · u unsnooze", m.width)))
	} else {
		filterLabel := "Filter: " + m.filter.Label() + " · f cycle · s star · z snooze"
		if m.width < 52 {
			filterLabel = "Filter: " + m.filter.Label() + " · f · s · z"
		}
		lines = append(lines, m.visual.paint(roleAccent, m.visual.fit(filterLabel, m.width)))
	}
	tableRows := make([]triage.Row, 0, len(sections.Review)+len(sections.Active)+len(sections.Snoozed))
	if m.showingSnoozed() {
		tableRows = append(tableRows, sections.Snoozed...)
	} else {
		tableRows = append(tableRows, sections.Review...)
		tableRows = append(tableRows, sections.Active...)
	}
	columns, widths := m.tableColumns(tableRows, m.showingSnoozed())
	content := make([]string, 0, len(tableRows)+len(m.sections)+8)
	rows := make(map[domain.ItemID]triage.Row)
	selectedLine := -1
	if m.showingSnoozed() {
		content = append(content, m.visual.paint(roleAccent, "SNOOZED"))
		content = append(content, m.renderTableHeader(columns, widths))
		for _, row := range sections.Snoozed {
			rows[row.PR.ID] = row
			if row.PR.ID == m.selected {
				selectedLine = len(content)
			}
			content = append(content, m.renderTableRow(row, columns, widths, m.hasSelection && row.PR.ID == m.selected))
		}
	} else {
		content = append(content, m.visual.paint(roleAccent, "REVIEW · "+strings.ToUpper(m.filter.Label())))
		if summaries := m.repositorySummaryLine(); summaries != "" {
			content = append(content, summaries)
		}
		content = append(content, m.renderTableHeader(columns, widths))
		for _, row := range sections.Review {
			rows[row.PR.ID] = row
			if row.PR.ID == m.selected {
				selectedLine = len(content)
			}
			content = append(content, m.renderTableRow(row, columns, widths, m.hasSelection && row.PR.ID == m.selected))
		}
		if len(sections.Active) > 0 {
			content = append(content, m.visual.paint(roleAccent, "ACTIVE FULLSEND AGENTS"))
			content = append(content, m.renderTableHeader(columns, widths))
			for _, row := range sections.Active {
				rows[row.PR.ID] = row
				if row.PR.ID == m.selected {
					selectedLine = len(content)
				}
				content = append(content, m.renderTableRow(row, columns, widths, m.hasSelection && row.PR.ID == m.selected))
			}
		}
	}
	if len(rows) == 0 {
		if m.showingSnoozed() {
			content = append(content, m.visual.paint(roleMuted, "No PRs are currently snoozed."))
		} else {
			content = append(content, m.visual.paint(roleMuted, "No PRs match this filter. Pending filter data remains visible when its result is unknown."))
		}
	}
	if len(m.opts.Repos) == 0 {
		content = append(content, m.visual.paint(roleMuted, "No GitHub repository found. Add repositories = [\"owner/name\"] to config.toml or run from a GitHub checkout."))
	}
	selected, hasSelected := m.selectedPR()
	var context []string
	if hasSelected {
		context = m.selectedContext(selected, rows[selected.ID])
	}
	statusLines := 0
	if m.status != "" {
		statusLines = 1
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
	lines = append(lines, m.visual.panel("PR attention", content[start:end], m.width)...)
	if hasSelected {
		lines = append(lines, context...)
	}
	if m.reminderWarning != "" {
		lines = append(lines, m.visual.paint(roleWarning, m.visual.fit(m.reminderWarning, m.width)))
	}
	if m.status != "" {
		lines = append(lines, m.visual.paint(roleWarning, m.visual.fit(m.status, m.width)))
	}
	footer := "j/k move · enter details · f filters · s star · z snooze · r refresh · L logs · ? help · q quit"
	if m.showingSnoozed() {
		footer = "j/k move · f filters · u unsnooze · L logs · ? help · q quit"
	}
	lines = append(lines, m.visual.footer(footer, m.width))
	return strings.Join(lines, "\n") + "\n"
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

func (m Model) repositorySummaryLine() string {
	var parts []string
	for _, section := range m.sections {
		state, role := m.sectionStatus(section)
		parts = append(parts, m.visual.paint(roleMuted, section.repo+" · ")+m.visual.paint(role, state))
	}
	separator := m.visual.paint(roleMuted, "  │  ")
	return strings.Join(parts, separator)
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
