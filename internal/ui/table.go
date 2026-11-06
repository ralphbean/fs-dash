package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/fullsend-ai/fs-dash/internal/domain"
	"github.com/fullsend-ai/fs-dash/internal/triage"
)

type tableColumn string
type tableAlignment int
type tableDisplayMode int
type tableCellKind int

const (
	tableDisplayText tableDisplayMode = iota
	tableDisplayEmoji
)

const (
	cellText tableCellKind = iota
	cellPending
	cellLoading
	cellStale
	cellFailed
	cellUnknown
	cellChecksFailed
	cellChecksRunning
	cellChecksPartial
	cellChecksNone
	cellChecksPass
	cellReviewChanges
	cellReviewApproved
	cellReviewCommented
	cellReviewAwaiting
	cellThreadCount
)

const (
	columnStar      tableColumn = "star"
	columnPR        tableColumn = "pr"
	columnTitle     tableColumn = "title"
	columnAuthor    tableColumn = "author"
	columnAssignees tableColumn = "assignees"
	columnUpdated   tableColumn = "updated"
	columnChecks    tableColumn = "checks"
	columnReview    tableColumn = "review"
	columnThreads   tableColumn = "threads"
	columnWake      tableColumn = "wake"

	alignLeft tableAlignment = iota
	alignCenter
	alignRight
)

type tableCell struct {
	kind       tableCellKind
	text       string
	indicator  string
	count      int
	incomplete bool
	role       visualRole
	url        string
	prefix     string
}

type tableColumnSpec struct {
	column     tableColumn
	label      string
	emojiLabel string
	min        int
	max        int
	align      tableAlignment
	flex       bool
}

func (cell tableCell) render(mode tableDisplayMode) string {
	if mode == tableDisplayEmoji {
		switch cell.kind {
		case cellPending:
			return "⏳"
		case cellLoading:
			if cell.indicator != "" {
				return cell.indicator
			}
			return "🔄"
		case cellStale:
			return "🕰️"
		case cellFailed:
			return "❌"
		case cellUnknown:
			return "❔"
		case cellChecksFailed:
			return fmt.Sprintf("❌%d", cell.count)
		case cellChecksRunning:
			return fmt.Sprintf("🔄%d", cell.count)
		case cellChecksPartial:
			return "⚠️"
		case cellChecksNone:
			return "➖"
		case cellChecksPass, cellReviewApproved:
			return "✅"
		case cellReviewChanges:
			return "🛑"
		case cellReviewCommented:
			return "💬"
		case cellReviewAwaiting:
			return "💤"
		case cellThreadCount:
			text := fmt.Sprint(cell.count)
			if cell.incomplete {
				text += "+"
			}
			return text
		}
	}
	switch cell.kind {
	case cellChecksFailed:
		return fmt.Sprintf("fail %d", cell.count)
	case cellChecksRunning:
		return fmt.Sprintf("running %d", cell.count)
	case cellChecksPartial:
		return "partial"
	case cellChecksNone:
		return "none"
	case cellChecksPass:
		return "pass"
	case cellReviewChanges:
		return "changes"
	case cellReviewApproved:
		return "approved"
	case cellReviewCommented:
		return "commented"
	case cellReviewAwaiting:
		return "awaiting"
	case cellThreadCount:
		text := fmt.Sprint(cell.count)
		if cell.incomplete {
			text += "+"
		}
		return text
	default:
		return cell.text
	}
}

func listTableSpecs(withWake bool) []tableColumnSpec {
	specs := []tableColumnSpec{
		{column: columnStar, label: "", min: 4, max: 4, align: alignCenter},
		{column: columnPR, label: "PR", min: 12, max: 32, align: alignLeft},
		{column: columnTitle, label: "Title", min: 12, max: 0, align: alignLeft, flex: true},
		{column: columnAuthor, label: "Author", min: 8, max: 18, align: alignLeft},
		{column: columnAssignees, label: "Assignees", min: 10, max: 22, align: alignLeft},
		{column: columnUpdated, label: "Updated", min: 13, max: 13, align: alignRight},
		{column: columnChecks, label: "Checks", emojiLabel: "🧪", min: 9, max: 9, align: alignCenter},
		{column: columnReview, label: "Review", emojiLabel: "👀", min: 12, max: 12, align: alignCenter},
		{column: columnThreads, label: "Threads", emojiLabel: "🧵", min: 7, max: 7, align: alignRight},
	}
	if withWake {
		specs = append(specs, tableColumnSpec{column: columnWake, label: "Wake", min: 16, max: 16, align: alignRight})
	}
	return specs
}

func (spec tableColumnSpec) displayLabel(mode tableDisplayMode) string {
	if mode == tableDisplayEmoji && spec.emojiLabel != "" {
		return spec.emojiLabel
	}
	return spec.label
}

func textCell(text string, role visualRole) tableCell {
	return tableCell{kind: cellText, text: text, role: role}
}

func stateCell(kind tableCellKind, text string, role visualRole) tableCell {
	return tableCell{kind: kind, text: text, role: role}
}

func loadingCell(text, indicator string, role visualRole) tableCell {
	return tableCell{kind: cellLoading, text: text, indicator: indicator, role: role}
}

func (m Model) tableCellFor(row triage.Row, column tableColumn) tableCell {
	switch column {
	case columnStar:
		if row.Starred {
			return tableCell{text: "⭐", role: roleWarning}
		}
		return tableCell{text: "", role: roleMuted}
	case columnPR:
		label := fmt.Sprintf("%s#%d", row.PR.ID.Repo, row.PR.ID.Number)
		url := row.PR.URL
		if url == "" {
			host := row.PR.ID.Host
			if host == "" {
				host = "github.com"
			}
			url = fmt.Sprintf("https://%s/%s/pull/%d", host, row.PR.ID.Repo, row.PR.ID.Number)
		}
		return tableCell{text: label, role: roleLink, url: url}
	case columnTitle:
		return tableCell{text: row.PR.Title, role: roleNeutral}
	case columnAuthor:
		return tableCell{text: row.PR.Author, role: roleMuted}
	case columnAssignees:
		return m.assigneeCell(row.PR)
	case columnUpdated:
		return m.updatedCell(row.PR)
	case columnChecks:
		return m.checksCell(row.PR)
	case columnReview:
		return m.reviewValue(row.PR)
	case columnThreads:
		return m.threadsCell(row.PR)
	case columnWake:
		if row.SnoozedUntil.IsZero() {
			return tableCell{text: "—", role: roleMuted}
		}
		return tableCell{text: row.SnoozedUntil.Local().Format("Jan 02 15:04"), role: roleMuted}
	default:
		return tableCell{text: "", role: roleMuted}
	}
}

func (m Model) updatedCell(pr domain.PR) tableCell {
	field := m.fieldFor(pr, "last_touched")
	if field == nil {
		return tableCell{text: "pending", role: roleUnrequested}
	}
	if field.Phase.Active() {
		return tableCell{text: "pending", role: rolePending}
	}
	if field.Phase == domain.Failed || field.Phase == domain.Cooldown {
		return tableCell{text: "unavailable", role: roleFailure}
	}
	if field.Phase == domain.Canceled || !field.HasValue {
		return tableCell{text: "unavailable", role: roleMuted}
	}
	activity, ok := field.Value.(domain.TimelineActivity)
	if !ok || activity.UpdatedAt.IsZero() {
		return tableCell{text: "unavailable", role: roleMuted}
	}
	role := roleMuted
	if field.Stale(m.opts.Clock.Now()) {
		role = roleStale
	}
	return tableCell{text: relativeTime(m.opts.Clock.Now(), activity.UpdatedAt), role: role}
}

func (m Model) assigneeCell(pr domain.PR) tableCell {
	logins := pr.Assignees.Logins
	complete := pr.Assignees.Complete
	field := m.freshField(pr, "assignees")
	if field != nil {
		if set, ok := field.Value.(domain.LoginSet); ok {
			logins, complete = set.Logins, set.Complete
		}
	}
	if len(logins) == 0 {
		if complete {
			return tableCell{text: "—", role: roleMuted}
		}
		field = m.fieldFor(pr, "assignees")
		if field != nil && (field.Phase == domain.Failed || field.Phase == domain.Cooldown) {
			return tableCell{text: "failed", role: roleFailure}
		}
		return tableCell{text: "pending", role: roleUnrequested}
	}
	text := strings.Join(logins, ", ")
	role := roleNeutral
	if !complete {
		text += ", …"
		role = rolePending
	}
	if field != nil && (field.Phase == domain.Failed || field.Phase == domain.Cooldown) {
		text += " !"
		role = roleFailure
	}
	return tableCell{text: text, role: role}
}

func (m Model) checksCell(pr domain.PR) tableCell {
	f := m.fieldFor(pr, "checks")
	if f == nil {
		return stateCell(cellPending, "pending", roleUnrequested)
	}
	if f.Phase.Active() {
		return loadingCell(m.loadingSummary(pr), m.spinner(), rolePending)
	}
	if f.Phase == domain.Failed || f.Phase == domain.Cooldown {
		return stateCell(cellFailed, "failed", roleFailure)
	}
	if !f.HasValue {
		return stateCell(cellPending, "pending", roleUnrequested)
	}
	if f.Stale(m.opts.Clock.Now()) {
		return stateCell(cellStale, "stale", roleStale)
	}
	summary, ok := f.Value.(domain.CheckSummary)
	if !ok {
		return stateCell(cellUnknown, "unknown", roleMuted)
	}
	if summary.Failed > 0 {
		return tableCell{kind: cellChecksFailed, count: summary.Failed, role: roleFailure}
	}
	if summary.Pending > 0 {
		return tableCell{kind: cellChecksRunning, count: summary.Pending, role: rolePending}
	}
	if !summary.Complete {
		return stateCell(cellChecksPartial, "partial", roleStale)
	}
	if summary.Total == 0 {
		return stateCell(cellChecksNone, "none", roleMuted)
	}
	return stateCell(cellChecksPass, "pass", roleSuccess)
}

func (m Model) reviewCell(pr domain.PR) (string, visualRole) {
	cell := m.reviewValue(pr)
	return cell.render(tableDisplayText), cell.role
}

func (m Model) reviewValue(pr domain.PR) tableCell {
	f := m.fieldFor(pr, "reviews")
	if f == nil {
		return stateCell(cellPending, "pending", roleUnrequested)
	}
	if f.Phase.Active() {
		return loadingCell(m.loadingSummary(pr), m.spinner(), rolePending)
	}
	if f.Phase == domain.Failed || f.Phase == domain.Cooldown {
		return stateCell(cellFailed, "failed", roleFailure)
	}
	if !f.HasValue {
		return stateCell(cellPending, "pending", roleUnrequested)
	}
	if f.Stale(m.opts.Clock.Now()) {
		return stateCell(cellStale, "stale", roleStale)
	}
	reviews, ok := f.Value.(domain.ReviewSet)
	if !ok {
		return stateCell(cellUnknown, "unknown", roleMuted)
	}
	effective, known := domain.EffectiveReviews(reviews.Reviews, reviews.Complete, f.Provenance.Version)
	if !known {
		return stateCell(cellPending, "pending", rolePending)
	}
	approved, changesRequested, commented := false, false, false
	for _, review := range effective {
		switch strings.ToUpper(review.State) {
		case "CHANGES_REQUESTED":
			changesRequested = true
		case "APPROVED":
			approved = true
		case "COMMENTED":
			commented = true
		}
	}
	if changesRequested {
		return stateCell(cellReviewChanges, "changes", roleFailure)
	}
	if approved {
		return stateCell(cellReviewApproved, "approved", roleSuccess)
	}
	if commented {
		return stateCell(cellReviewCommented, "commented", roleAccent)
	}
	return stateCell(cellReviewAwaiting, "awaiting", roleMuted)
}

func (m Model) threadsCell(pr domain.PR) tableCell {
	f := m.fieldFor(pr, "review_threads")
	if f == nil {
		return stateCell(cellPending, "pending", roleUnrequested)
	}
	if f.Phase.Active() {
		return loadingCell(m.loadingSummary(pr), m.spinner(), rolePending)
	}
	if f.Phase == domain.Failed || f.Phase == domain.Cooldown {
		return stateCell(cellFailed, "failed", roleFailure)
	}
	if !f.HasValue {
		return stateCell(cellPending, "pending", roleUnrequested)
	}
	if f.Stale(m.opts.Clock.Now()) {
		return stateCell(cellStale, "stale", roleStale)
	}
	summary, ok := f.Value.(domain.ThreadSummary)
	if !ok {
		return stateCell(cellUnknown, "unknown", roleMuted)
	}
	role := roleMuted
	if summary.Unresolved > 0 {
		role = roleWarning
	}
	if !summary.Complete {
		role = roleStale
	}
	return tableCell{kind: cellThreadCount, count: summary.Unresolved, incomplete: !summary.Complete, role: role}
}

func (m Model) loadingSummary(pr domain.PR) string {
	if m.hasSelection && pr.ID == m.selected {
		return m.spinner() + " loading"
	}
	return "loading"
}

func (m Model) tableDisplayMode() tableDisplayMode {
	if m.emojiMode {
		return tableDisplayEmoji
	}
	return tableDisplayText
}

func (m Model) tableModeHint() string {
	if m.emojiMode {
		return "e text"
	}
	return "e emoji"
}

func relativeTime(now, at time.Time) string {
	if at.IsZero() {
		return "unknown"
	}
	delta := now.Sub(at)
	if delta < 0 {
		delta = -delta
		if delta < time.Minute {
			return "in <1 min"
		}
		if delta < time.Hour {
			return relativeAge(int(delta.Minutes()), "min", "mins", "in ")
		}
		if delta < 24*time.Hour {
			return relativeAge(int(delta.Hours()), "hr", "hrs", "in ")
		}
		return relativeAge(int(delta.Hours()/24), "day", "days", "in ")
	}
	if delta < time.Minute {
		return "just now"
	}
	if delta < time.Hour {
		return relativeAge(int(delta.Minutes()), "min", "mins", "")
	}
	if delta < 24*time.Hour {
		return relativeAge(int(delta.Hours()), "hr", "hrs", "")
	}
	if delta < 7*24*time.Hour {
		return relativeAge(int(delta.Hours()/24), "day", "days", "")
	}
	if delta < 30*24*time.Hour {
		return relativeAge(int(delta.Hours()/(24*7)), "week", "weeks", "")
	}
	if delta < 365*24*time.Hour {
		return relativeAge(int(delta.Hours()/(24*30)), "month", "months", "")
	}
	return relativeAge(int(delta.Hours()/(24*365)), "year", "years", "")
}

func relativeAge(count int, singular, plural, prefix string) string {
	unit := plural
	if count == 1 {
		unit = singular
	}
	if prefix == "in " {
		return fmt.Sprintf("in %d %s", count, unit)
	}
	return fmt.Sprintf("%d %s ago", count, unit)
}

func (m Model) tableColumns(rows []triage.Row, withWake bool) ([]tableColumnSpec, []int) {
	specs := listTableSpecs(withWake)
	widths := make([]int, len(specs))
	for i, spec := range specs {
		width := spec.min
		if m.emojiMode && spec.emojiLabel != "" {
			width = lipgloss.Width(spec.emojiLabel)
			for _, row := range rows {
				cell := m.tableCellFor(row, spec.column)
				width = max(width, lipgloss.Width(cell.render(tableDisplayEmoji)))
			}
		} else if spec.column == columnPR || spec.column == columnAuthor || spec.column == columnAssignees {
			for _, row := range rows {
				cell := m.tableCellFor(row, spec.column)
				width = max(width, lipgloss.Width(cell.render(tableDisplayText)))
			}
			width = min(width, spec.max)
			width = max(width, spec.min)
			if spec.column == columnPR && m.visual.noColor && m.hasSelection {
				width = min(spec.max, width+2)
			}
		} else if spec.max > 0 && !spec.flex {
			width = spec.max
		}
		widths[i] = width
	}
	available := m.width
	if !m.visual.compact && m.width >= 52 {
		available -= 4
	}
	if available < 1 {
		available = 1
	}
	for len(specs) > 3 && tableWidth(widths, specs, 12) > available {
		specs = specs[:len(specs)-1]
		widths = widths[:len(widths)-1]
	}
	if len(specs) >= 3 {
		titleIndex := -1
		for i, spec := range specs {
			if spec.column == columnTitle {
				titleIndex = i
				break
			}
		}
		if titleIndex >= 0 {
			fixed := tableWidth(widths, specs, 0)
			titleWidth := available - fixed
			if titleWidth < 1 {
				idIndex, starIndex := -1, -1
				for i, spec := range specs {
					switch spec.column {
					case columnPR:
						idIndex = i
					case columnStar:
						starIndex = i
					}
				}
				if idIndex >= 0 {
					idWidth := available - (len(specs) - 1) - 1
					for i, width := range widths {
						if i != idIndex && i != titleIndex {
							idWidth -= width
						}
					}
					widths[idIndex] = max(1, min(widths[idIndex], idWidth))
					fixed = tableWidth(widths, specs, 0)
					titleWidth = available - fixed
				}
				if titleWidth < 1 && starIndex >= 0 {
					specs = append(specs[:starIndex], specs[starIndex+1:]...)
					widths = append(widths[:starIndex], widths[starIndex+1:]...)
					if starIndex < titleIndex {
						titleIndex--
					}
					fixed = tableWidth(widths, specs, 0)
					titleWidth = available - fixed
				}
				titleWidth = max(1, titleWidth)
			}
			widths[titleIndex] = titleWidth
		}
	}
	return specs, widths
}

func tableWidth(widths []int, specs []tableColumnSpec, titleWidth int) int {
	total := max(0, len(specs)-1)
	for i, spec := range specs {
		if spec.flex {
			total += titleWidth
		} else {
			total += widths[i]
		}
	}
	return total
}

func (m Model) renderTableHeader(specs []tableColumnSpec, widths []int) string {
	cells := make([]tableCell, len(specs))
	for i, spec := range specs {
		cells[i] = textCell(spec.displayLabel(m.tableDisplayMode()), roleAccent)
	}
	return m.renderTableCells(cells, specs, widths, false)
}

func (m Model) renderTableRow(row triage.Row, specs []tableColumnSpec, widths []int, selected bool) string {
	cells := make([]tableCell, len(specs))
	for i, spec := range specs {
		cells[i] = m.tableCellFor(row, spec.column)
		if selected && m.visual.noColor && spec.column == columnPR {
			cells[i].prefix = "> "
		}
	}
	return m.renderTableCells(cells, specs, widths, selected)
}

func (m Model) renderTableCells(cells []tableCell, specs []tableColumnSpec, widths []int, selected bool) string {
	parts := make([]string, 0, len(cells)*2-1)
	for i, cell := range cells {
		prefixWidth := lipgloss.Width(cell.prefix)
		text := ansi.Truncate(cell.render(m.tableDisplayMode()), max(0, widths[i]-prefixWidth), "…")
		padding := max(0, widths[i]-prefixWidth-lipgloss.Width(text))
		left, right := 0, padding
		switch specs[i].align {
		case alignCenter:
			left, right = padding/2, padding-padding/2
		case alignRight:
			left, right = padding, 0
		}
		style := m.visual.style(cell.role)
		if selected && !m.visual.noColor {
			style = style.Reverse(true).Bold(true)
		}
		paddedText := strings.Repeat(" ", left) + text + strings.Repeat(" ", right)
		rendered := style.Render(cell.prefix + paddedText)
		if cell.url != "" {
			linked := ansi.SetHyperlink(cell.url) + style.Render(paddedText) + ansi.ResetHyperlink()
			if cell.prefix != "" {
				rendered = cell.prefix + linked
			} else {
				rendered = linked
			}
		}
		parts = append(parts, rendered)
		if i < len(cells)-1 {
			gap := " "
			if selected && !m.visual.noColor {
				gap = m.visual.style(roleFocus).Reverse(true).Bold(true).Render(gap)
			}
			parts = append(parts, gap)
		}
	}
	return strings.Join(parts, "")
}
