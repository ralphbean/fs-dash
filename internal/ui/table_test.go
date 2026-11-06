package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/fullsend-ai/fs-dash/internal/config"
	"github.com/fullsend-ai/fs-dash/internal/domain"
	"github.com/fullsend-ai/fs-dash/internal/reminder"
	"github.com/fullsend-ai/fs-dash/internal/triage"
	"github.com/muesli/termenv"
)

func TestListTableColumnOrderAndAlignment(t *testing.T) {
	specs := listTableSpecs(false)
	wantColumns := []tableColumn{columnStar, columnPR, columnTitle, columnAuthor, columnAssignees, columnUpdated, columnChecks, columnReview, columnThreads}
	wantAlignments := []tableAlignment{alignCenter, alignLeft, alignLeft, alignLeft, alignLeft, alignRight, alignCenter, alignCenter, alignRight}
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
	m.sections[0].field.Succeed(m.sections[0].field.Generation, []domain.PR{pr}, domain.Provenance{Version: "list-v1", FreshUntil: now.Add(time.Hour)})
	m.ensureSelection()
	putField(&m, pr, "checks", domain.CheckSummary{Total: 3, Complete: true}, now)
	putField(&m, pr, "reviews", domain.ReviewSet{Reviews: []domain.Review{{Author: "reviewer", State: "APPROVED"}}, Complete: true}, now)
	putField(&m, pr, "review_threads", domain.ThreadSummary{Unresolved: 2, Complete: true}, now)
	row := m.projection().Review[0]
	specs, widths := m.tableColumns([]triage.Row{row}, false)
	line := m.renderTableRow(row, specs, widths, true)
	plain := ansi.Strip(line)
	for _, want := range []string{"⭐", "owner/repo#42", "Table title", "ada", "me, reviewer", "2 days ago", "pass", "approved", "2"} {
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

func TestNarrowTableKeepsPRAndTitle(t *testing.T) {
	pr := testUIPr(61, time.Now())
	pr.Title = "A narrow terminal title"
	m := loadedModel(t, []domain.PR{pr})
	m.width, m.height, m.visual.compact = 40, 12, true
	rows := m.projection().Review
	specs, widths := m.tableColumns(rows, false)
	if len(specs) < 3 || specs[1].column != columnPR || specs[2].column != columnTitle {
		t.Fatalf("narrow table hid PR or Title: %+v", specs)
	}
	line := ansi.Strip(m.renderTableRow(rows[0], specs, widths, false))
	if ansi.StringWidth(line) > 40 || !strings.Contains(line, "owner/repo#61") || !strings.Contains(line, "A narrow") {
		t.Fatalf("narrow row does not preserve the PR and title within 40 cells: specs=%+v widths=%v row=%q", specs, widths, line)
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
	m.width, m.height, m.viewer, m.showSnoozed = 180, 30, "me", true
	m.sections[0].field.Succeed(m.sections[0].field.Generation, []domain.PR{pr}, domain.Provenance{Version: "list-v1", FreshUntil: now.Add(time.Hour)})
	m.ensureSelection()
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "Wake") || !strings.Contains(view, wake.Local().Format("Jan 02 15:04")) {
		rows := m.projection().Snoozed
		specs, widths := m.tableColumns(rows, true)
		t.Fatalf("snoozed table is missing the Wake column value: specs=%+v widths=%v view=%s", specs, widths, view)
	}
}
