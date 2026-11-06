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
	text   string
	role   visualRole
	url    string
	prefix string
}

type tableColumnSpec struct {
	column tableColumn
	label  string
	min    int
	max    int
	align  tableAlignment
	flex   bool
}

func listTableSpecs(withWake bool) []tableColumnSpec {
	specs := []tableColumnSpec{
		{column: columnStar, label: "", min: 4, max: 4, align: alignCenter},
		{column: columnPR, label: "PR", min: 12, max: 32, align: alignLeft},
		{column: columnTitle, label: "Title", min: 12, max: 0, align: alignLeft, flex: true},
		{column: columnAuthor, label: "Author", min: 8, max: 18, align: alignLeft},
		{column: columnAssignees, label: "Assignees", min: 10, max: 22, align: alignLeft},
		{column: columnUpdated, label: "Updated", min: 13, max: 13, align: alignRight},
		{column: columnChecks, label: "Checks", min: 9, max: 9, align: alignCenter},
		{column: columnReview, label: "Review", min: 12, max: 12, align: alignCenter},
		{column: columnThreads, label: "Threads", min: 7, max: 7, align: alignRight},
	}
	if withWake {
		specs = append(specs, tableColumnSpec{column: columnWake, label: "Wake", min: 16, max: 16, align: alignRight})
	}
	return specs
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
		text, role := m.checksCell(row.PR)
		return tableCell{text: text, role: role}
	case columnReview:
		text, role := m.reviewCell(row.PR)
		return tableCell{text: text, role: role}
	case columnThreads:
		text, role := m.threadsCell(row.PR)
		return tableCell{text: text, role: role}
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

func (m Model) checksCell(pr domain.PR) (string, visualRole) {
	f := m.fieldFor(pr, "checks")
	if f == nil {
		return "pending", roleUnrequested
	}
	if f.Phase.Active() {
		return m.loadingSummary(pr), rolePending
	}
	if f.Phase == domain.Failed || f.Phase == domain.Cooldown {
		return "failed", roleFailure
	}
	if !f.HasValue {
		return "pending", roleUnrequested
	}
	if f.Stale(m.opts.Clock.Now()) {
		return "stale", roleStale
	}
	summary, ok := f.Value.(domain.CheckSummary)
	if !ok {
		return "unknown", roleMuted
	}
	if summary.Failed > 0 {
		return fmt.Sprintf("fail %d", summary.Failed), roleFailure
	}
	if summary.Pending > 0 {
		return fmt.Sprintf("running %d", summary.Pending), rolePending
	}
	if !summary.Complete {
		return "partial", roleStale
	}
	if summary.Total == 0 {
		return "none", roleMuted
	}
	return "pass", roleSuccess
}

func (m Model) reviewCell(pr domain.PR) (string, visualRole) {
	f := m.fieldFor(pr, "reviews")
	if f == nil {
		return "pending", roleUnrequested
	}
	if f.Phase.Active() {
		return m.loadingSummary(pr), rolePending
	}
	if f.Phase == domain.Failed || f.Phase == domain.Cooldown {
		return "failed", roleFailure
	}
	if !f.HasValue {
		return "pending", roleUnrequested
	}
	if f.Stale(m.opts.Clock.Now()) {
		return "stale", roleStale
	}
	reviews, ok := f.Value.(domain.ReviewSet)
	if !ok {
		return "unknown", roleMuted
	}
	effective, known := domain.EffectiveReviews(reviews.Reviews, reviews.Complete, f.Provenance.Version)
	if !known {
		return "pending", rolePending
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
		return "changes", roleFailure
	}
	if approved {
		return "approved", roleSuccess
	}
	if commented {
		return "commented", roleAccent
	}
	return "awaiting", roleMuted
}

func (m Model) threadsCell(pr domain.PR) (string, visualRole) {
	f := m.fieldFor(pr, "review_threads")
	if f == nil {
		return "pending", roleUnrequested
	}
	if f.Phase.Active() {
		return m.loadingSummary(pr), rolePending
	}
	if f.Phase == domain.Failed || f.Phase == domain.Cooldown {
		return "failed", roleFailure
	}
	if !f.HasValue {
		return "pending", roleUnrequested
	}
	if f.Stale(m.opts.Clock.Now()) {
		return "stale", roleStale
	}
	summary, ok := f.Value.(domain.ThreadSummary)
	if !ok {
		return "unknown", roleMuted
	}
	text := fmt.Sprint(summary.Unresolved)
	role := roleMuted
	if summary.Unresolved > 0 {
		role = roleWarning
	}
	if !summary.Complete {
		text += "+"
		role = roleStale
	}
	return text, role
}

func (m Model) loadingSummary(pr domain.PR) string {
	if m.hasSelection && pr.ID == m.selected {
		return m.spinner() + " loading"
	}
	return "loading"
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
		if spec.column == columnPR || spec.column == columnAuthor || spec.column == columnAssignees {
			for _, row := range rows {
				cell := m.tableCellFor(row, spec.column)
				width = max(width, lipgloss.Width(cell.text))
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
		cells[i] = tableCell{text: spec.label, role: roleAccent}
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
		text := ansi.Truncate(cell.text, max(0, widths[i]-prefixWidth), "…")
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
