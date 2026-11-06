package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/fullsend-ai/fs-dash/internal/config"
	"github.com/fullsend-ai/fs-dash/internal/domain"
	"github.com/fullsend-ai/fs-dash/internal/reminder"
	"github.com/fullsend-ai/fs-dash/internal/triage"
	"github.com/muesli/termenv"
)

func TestListTableColumnOrderAndAlignment(t *testing.T) {
	specs := listTableSpecs(false)
	wantColumns := []tableColumn{columnStar, columnPR, columnTitle, columnAuthor, columnAssignees, columnUpdated, columnChecks, columnHumanReview, columnBotReview, columnThreads}
	wantAlignments := []tableAlignment{alignCenter, alignLeft, alignLeft, alignLeft, alignLeft, alignRight, alignCenter, alignCenter, alignCenter, alignRight}
	if len(specs) != len(wantColumns) {
		t.Fatalf("normal table has %d columns, want %d", len(specs), len(wantColumns))
	}
	for i, spec := range specs {
		if spec.column != wantColumns[i] || spec.align != wantAlignments[i] {
			t.Errorf("column %d = (%s, alignment %d), want (%s, alignment %d)", i, spec.column, spec.align, wantColumns[i], wantAlignments[i])
		}
	}
	if specs[0].label != "" {
		t.Fatalf("Star header = %q, want an empty cell", specs[0].label)
	}
	snoozed := listTableSpecs(true)
	if got := snoozed[len(snoozed)-1].column; got != columnWake {
		t.Fatalf("last snoozed column = %s, want Wake", got)
	}
}

func TestTableRendersLinkedSummariesAndSelectedRow(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	pr := testUIPr(42, now.Add(-48*time.Hour))
	pr.Title = "Table title"
	pr.Author = "ada"
	pr.Assignees = domain.AssigneeSet{Logins: []string{"me", "reviewer"}, Complete: true}
	store := reminder.NewMemoryStore()
	if err := store.ToggleStar(pr.ID); err != nil {
		t.Fatal(err)
	}
	m := New(Options{Config: config.Default(), Repos: []string{"owner/repo"}, Clock: &testClock{now: now}, Reminders: store})
	m.width, m.height = 180, 40
	m.visual = m.visual.withProfile(termenv.TrueColor)
	m.visual.noColor = false
	m.viewer = "me"
	m.filter = triage.FilterAll
	m.sections[0].field.Succeed(m.sections[0].field.Generation, []domain.PR{pr}, domain.Provenance{Version: "list-v1", FreshUntil: now.Add(time.Hour)})
	m.ensureSelection()
	putField(&m, pr, "checks", domain.CheckSummary{Total: 3, Complete: true}, now)
	putField(&m, pr, "reviews", domain.ReviewSet{Reviews: []domain.Review{{Author: "reviewer", State: "APPROVED"}}, Complete: true}, now)
	putField(&m, pr, "review_threads", domain.ThreadSummary{Unresolved: 2, Complete: true}, now)
	putField(&m, pr, "last_touched", domain.TimelineActivity{UpdatedAt: now.Add(-48 * time.Hour)}, now)
	row := m.projection().Review[0]
	specs, widths := m.tableColumns([]triage.Row{row}, false)
	line := m.renderTableRow(row, specs, widths, true)
	plain := ansi.Strip(line)
	for _, want := range []string{"⭐", "owner/repo#42", "Table title", "ada", "me, reviewer", "2 days ago", "✅", "2"} {
		if !strings.Contains(plain, want) {
			t.Errorf("table row lacks %q: %s", want, plain)
		}
	}
	if !strings.Contains(line, ansi.SetHyperlink(pr.URL)) {
		t.Fatalf("PR identity is not linked to %q: %q", pr.URL, line)
	}
	if !strings.Contains(line, "\x1b[") {
		t.Fatal("selected row is not highlighted on a color terminal")
	}
	wantWidth := 0
	for _, width := range widths {
		wantWidth += width
	}
	wantWidth += len(widths) - 1
	plainWidth := ansi.StringWidth(line)
	if plainWidth != wantWidth {
		t.Fatalf("rendered row width = %d, want %d; specs=%+v widths=%v", plainWidth, wantWidth, specs, widths)
	}
}

func TestCompactColumnPresentationAndWidths(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	pr := testUIPr(43, now)
	pr.Assignees = domain.AssigneeSet{Logins: []string{"me"}, Complete: true}
	m := New(Options{Config: config.Default(), Repos: []string{"owner/repo"}, Clock: &testClock{now: now}})
	m.width = 220
	m.sections[0].field.Succeed(m.sections[0].field.Generation, []domain.PR{pr}, domain.Provenance{Version: "list-v1", FreshUntil: now.Add(time.Hour)})
	putField(&m, pr, "checks", domain.CheckSummary{Failed: 12, Total: 12, Complete: true}, now)
	putField(&m, pr, "reviews", domain.ReviewSet{Reviews: []domain.Review{
		{Author: "reviewer", State: "APPROVED"},
		{Author: "renovate[bot]", IsBot: true, State: "CHANGES_REQUESTED"},
	}, Complete: true}, now)
	putField(&m, pr, "review_threads", domain.ThreadSummary{Unresolved: 123, Complete: false}, now)
	row := m.projection().Review[0]

	emojiSpecs, emojiWidths := m.tableColumns([]triage.Row{row}, false)
	emojiHeader := ansi.Strip(m.renderTableHeader(emojiSpecs, emojiWidths))
	emojiRow := ansi.Strip(m.renderTableRow(row, emojiSpecs, emojiWidths, false))
	for _, want := range []string{"🧪", "👤", "🤖", "🧵"} {
		if !strings.Contains(emojiHeader, want) {
			t.Errorf("emoji header lacks %q: %q", want, emojiHeader)
		}
	}
	for _, want := range []string{"❌12", "✅", "🛑", "123+"} {
		if !strings.Contains(emojiRow, want) {
			t.Errorf("emoji row lacks %q: %q", want, emojiRow)
		}
	}

	emojiWidth := func(column tableColumn) int {
		for i, spec := range emojiSpecs {
			if spec.column == column {
				return emojiWidths[i]
			}
		}
		t.Fatalf("emoji table lacks %s", column)
		return 0
	}
	m.emojiMode = false
	textSpecs, textWidths := m.tableColumns([]triage.Row{row}, false)
	textHeader := ansi.Strip(m.renderTableHeader(textSpecs, textWidths))
	textRow := ansi.Strip(m.renderTableRow(row, textSpecs, textWidths, false))
	for _, want := range []string{"Checks", "Human review", "Bot review", "Threads"} {
		if !strings.Contains(textHeader, want) {
			t.Errorf("text header lacks %q: %q", want, textHeader)
		}
	}
	for _, want := range []string{"fail 12", "approved", "changes", "123+"} {
		if !strings.Contains(textRow, want) {
			t.Errorf("text row lacks %q: %q", want, textRow)
		}
	}
	for _, column := range []tableColumn{columnChecks, columnHumanReview, columnBotReview, columnThreads} {
		for i, spec := range textSpecs {
			if spec.column == column && emojiWidth(column) >= textWidths[i] {
				t.Errorf("%s width = %d in emoji mode, want narrower than text width %d", column, emojiWidth(column), textWidths[i])
			}
		}
	}
}

func TestTableCellStatusMappings(t *testing.T) {
	cases := []struct {
		name  string
		cell  tableCell
		text  string
		emoji string
	}{
		{name: "pending", cell: stateCell(cellPending, "pending", rolePending), text: "pending", emoji: "⏳"},
		{name: "loading", cell: loadingCell("loading", frames[0], rolePending), text: "loading", emoji: frames[0]},
		{name: "stale", cell: stateCell(cellStale, "stale", roleStale), text: "stale", emoji: "🕰️"},
		{name: "failed", cell: stateCell(cellFailed, "failed", roleFailure), text: "failed", emoji: "❌"},
		{name: "unknown", cell: stateCell(cellUnknown, "unknown", roleMuted), text: "unknown", emoji: "❔"},
		{name: "check failures", cell: tableCell{kind: cellChecksFailed, count: 2}, text: "fail 2", emoji: "❌2"},
		{name: "running checks", cell: tableCell{kind: cellChecksRunning, count: 3}, text: "running 3", emoji: "🔄3"},
		{name: "partial checks", cell: stateCell(cellChecksPartial, "partial", roleStale), text: "partial", emoji: "⚠️"},
		{name: "no checks", cell: stateCell(cellChecksNone, "none", roleMuted), text: "none", emoji: "➖"},
		{name: "passed checks", cell: stateCell(cellChecksPass, "pass", roleSuccess), text: "pass", emoji: "✅"},
		{name: "changes requested", cell: stateCell(cellReviewChanges, "changes", roleFailure), text: "changes", emoji: "🛑"},
		{name: "approved review", cell: stateCell(cellReviewApproved, "approved", roleSuccess), text: "approved", emoji: "✅"},
		{name: "commented review", cell: stateCell(cellReviewCommented, "commented", roleAccent), text: "commented", emoji: "💬"},
		{name: "awaiting review", cell: stateCell(cellReviewAwaiting, "awaiting", roleMuted), text: "awaiting", emoji: "💤"},
		{name: "incomplete threads", cell: tableCell{kind: cellThreadCount, count: 4, incomplete: true}, text: "4+", emoji: "4+"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.cell.render(tableDisplayText); got != tc.text {
				t.Errorf("text rendering = %q, want %q", got, tc.text)
			}
			if got := tc.cell.render(tableDisplayEmoji); got != tc.emoji {
				t.Errorf("emoji rendering = %q, want %q", got, tc.emoji)
			}
		})
	}
}

func TestEmojiLoadingIndicatorUsesAnimatedFrame(t *testing.T) {
	first := loadingCell("loading", frames[0], rolePending).render(tableDisplayEmoji)
	second := loadingCell("loading", frames[1], rolePending).render(tableDisplayEmoji)
	if first != frames[0] || second != frames[1] || first == second {
		t.Fatalf("emoji loading frames = %q, %q; want distinct animated frames %q, %q", first, second, frames[0], frames[1])
	}
}

func TestNarrowTableKeepsPRAndTitle(t *testing.T) {
	pr := testUIPr(61, time.Now())
	pr.Title = "A narrow terminal title"
	m := loadedModel(t, []domain.PR{pr})
	m.width, m.height, m.visual.compact = 40, 12, true
	rows := m.projection().Review
	for _, emojiMode := range []bool{true, false} {
		m.emojiMode = emojiMode
		specs, widths := m.tableColumns(rows, false)
		if len(specs) < 3 || specs[1].column != columnPR || specs[2].column != columnTitle {
			t.Fatalf("narrow table hid PR or Title (emoji=%t): %+v", emojiMode, specs)
		}
		line := ansi.Strip(m.renderTableRow(rows[0], specs, widths, false))
		if ansi.StringWidth(line) > 40 || !strings.Contains(line, "owner/repo#61") || !strings.Contains(line, "A narrow") {
			t.Fatalf("narrow row does not preserve the PR and title within 40 cells (emoji=%t): specs=%+v widths=%v row=%q", emojiMode, specs, widths, line)
		}
	}
}

func TestSnoozedTableAddsWakeColumn(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	pr := testUIPr(9, now)
	pr.Assignees = domain.AssigneeSet{Logins: []string{"me"}, Complete: true}
	store := reminder.NewMemoryStore()
	wake := now.Add(26 * time.Hour)
	if err := store.Snooze(pr.ID, wake); err != nil {
		t.Fatal(err)
	}
	m := New(Options{Config: config.Default(), Repos: []string{"owner/repo"}, Clock: &testClock{now: now}, Reminders: store})
	m.width, m.height, m.viewer, m.filter = 180, 30, "me", triage.FilterSnoozed
	m.sections[0].field.Succeed(m.sections[0].field.Generation, []domain.PR{pr}, domain.Provenance{Version: "list-v1", FreshUntil: now.Add(time.Hour)})
	m.ensureSelection()
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "Wake") || !strings.Contains(view, wake.Local().Format("Jan 02 15:04")) {
		rows := m.projection().Snoozed
		specs, widths := m.tableColumns(rows, true)
		t.Fatalf("snoozed table is missing the Wake column value: specs=%+v widths=%v view=%s", specs, widths, view)
	}
	m.emojiMode = false
	textView := ansi.Strip(m.View())
	if !strings.Contains(textView, "Checks") || !strings.Contains(textView, "Human review") || !strings.Contains(textView, "Bot review") || !strings.Contains(textView, "Threads") {
		t.Fatalf("text mode was not used in the Snoozed section: %s", textView)
	}
}

func TestDisplayModeAppliesToReviewAndActiveSections(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	reviewPR := testUIPr(71, now)
	activePR := testUIPr(72, now.Add(-time.Minute))
	for _, pr := range []*domain.PR{&reviewPR, &activePR} {
		pr.Assignees = domain.AssigneeSet{Logins: []string{"me"}, Complete: true}
	}
	m := New(Options{Config: config.Default(), Repos: []string{"owner/repo"}, Clock: &testClock{now: now}})
	m.width, m.height, m.filter = 180, 40, triage.FilterAll
	m.sections[0].field.Succeed(m.sections[0].field.Generation, []domain.PR{reviewPR, activePR}, domain.Provenance{Version: "list-v1", FreshUntil: now.Add(time.Hour)})
	putField(&m, activePR, "activity", domain.ActivityEvidence{State: domain.ActivityActive, StartedAt: now, Complete: true}, now)
	m.ensureSelection()

	emojiView := ansi.Strip(m.View())
	if strings.Count(emojiView, "🧪") != 2 || !strings.Contains(emojiView, "ACTIVE FULLSEND AGENTS") {
		t.Fatalf("emoji mode was not used in both Review and Active sections: %s", emojiView)
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = next.(Model)
	textView := ansi.Strip(m.View())
	if strings.Count(textView, "Checks") != 2 || strings.Contains(textView, "🧪") {
		t.Fatalf("text mode was not used in both Review and Active sections: %s", textView)
	}
}
