package ui

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/fullsend-ai/fs-dash/internal/config"
	"github.com/fullsend-ai/fs-dash/internal/diag"
	"github.com/fullsend-ai/fs-dash/internal/domain"
	"github.com/fullsend-ai/fs-dash/internal/reminder"
	"github.com/fullsend-ai/fs-dash/internal/testutil"
	"github.com/fullsend-ai/fs-dash/internal/triage"
)

func TestFailedSectionSettlesWithoutBlockingOtherSection(t *testing.T) {
	logs := diag.New(20)
	lister := testutil.PRLister{Fn: func(_ context.Context, repo string) (domain.PRList, error) {
		if repo == "one/repo" {
			return domain.PRList{}, &domain.FetchError{Kind: domain.ErrorTransient, Message: "502 Bad Gateway"}
		}
		return domain.PRList{PullRequests: []domain.PR{{ID: domain.ItemID{Source: "github", Host: "github.com", Repo: repo, Number: 2}, Title: "Ready", URL: "https://github.com/two/repo/pull/2"}}}, nil
	}}
	m := New(Options{Config: config.Default(), Repos: []string{"one/repo", "two/repo"}, Lister: lister, Logs: logs})
	failed := m.fetchList("one/repo", m.sections[0].field.Generation, 1)()
	ready := m.fetchList("two/repo", m.sections[1].field.Generation, 1)()
	next, _ := m.Update(failed)
	m = next.(Model)
	if m.sections[0].field.Phase != domain.Cooldown || m.sections[0].field.Phase.Active() {
		t.Fatalf("failed section still loading: %+v", m.sections[0].field)
	}
	next, _ = m.Update(ready)
	m = next.(Model)
	if m.sections[1].field.Phase != domain.Ready || !m.hasSelection {
		t.Fatal("other section did not load")
	}
	view := m.View()
	if !strings.Contains(view, "cooldown") || !strings.Contains(view, "Ready") {
		t.Fatalf("bad partial view: %s", view)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	m = next.(Model)
	if !strings.Contains(m.View(), "502 Bad Gateway") {
		t.Fatal("error absent from in-app log")
	}
}

func TestSelectionSurvivesReorderedRows(t *testing.T) {
	m := New(Options{Config: config.Default(), Repos: []string{"one/repo"}, Lister: testutil.PRLister{Fn: func(context.Context, string) (domain.PRList, error) { return domain.PRList{}, errors.New("unused") }}})
	id := domain.ItemID{Source: "github", Host: "github.com", Repo: "one/repo", Number: 2}
	prs := []domain.PR{{ID: domain.ItemID{Source: "github", Host: "github.com", Repo: "one/repo", Number: 1}, Title: "one"}, {ID: id, Title: "two"}}
	next, _ := m.Update(listMsg{repo: "one/repo", generation: m.sections[0].field.Generation, value: prs, at: time.Now()})
	m = next.(Model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = next.(Model)
	if m.selected != id {
		t.Fatal("move failed")
	}
	next, _ = m.Update(listMsg{repo: "one/repo", generation: m.sections[0].field.Generation, value: []domain.PR{prs[1], prs[0]}, at: time.Now()})
	m = next.(Model)
	if m.selected != id {
		t.Fatal("selection changed after reorder")
	}
}

func TestListViewUsesTerminalHeight(t *testing.T) {
	m := New(Options{Config: config.Default(), Repos: []string{"one/repo"}})
	rows := make([]domain.PR, 100)
	for i := range rows {
		rows[i] = domain.PR{ID: domain.ItemID{Source: "github", Host: "github.com", Repo: "one/repo", Number: i + 1}, Title: "PR"}
	}
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(Model)
	next, _ = m.Update(listMsg{repo: "one/repo", generation: m.sections[0].field.Generation, value: rows, at: time.Now()})
	m = next.(Model)
	if lines := strings.Count(m.View(), "\n"); lines > 24 {
		t.Fatalf("view exceeds terminal height: %d lines", lines)
	}
}

func TestRepositorySummariesShareOneHorizontalRow(t *testing.T) {
	now := time.Now()
	repos := []string{"alpha/one", "bravo/two", "charlie/three"}
	m := New(Options{Config: config.Default(), Repos: repos})
	m.width, m.height, m.viewer = 260, 32, "me"
	for i, repo := range repos {
		pr := testUIPr(i+1, now)
		pr.ID.Repo = repo
		pr.URL = fmt.Sprintf("https://github.com/%s/pull/%d", repo, i+1)
		pr.Assignees = domain.AssigneeSet{Logins: []string{"me"}, Complete: true}
		m.sections[i].field.Succeed(m.sections[i].field.Generation, []domain.PR{pr}, domain.Provenance{Version: "list-v1", FreshUntil: now.Add(time.Hour)})
	}
	m.ensureSelection()

	assertSummaryRow := func(width int) (string, []string) {
		t.Helper()
		m.width = width
		m.visual.compact = width < 64
		view := m.View()
		lines := strings.Split(strings.TrimSuffix(ansi.Strip(view), "\n"), "\n")
		reviewLine, tableHeaderLine := -1, -1
		for i, line := range lines {
			if strings.Contains(line, "REVIEW ·") {
				reviewLine = i
			}
			if strings.Contains(line, "PR") && strings.Contains(line, "Title") {
				tableHeaderLine = i
			}
		}
		if reviewLine < 0 || tableHeaderLine != reviewLine+2 {
			t.Fatalf("repository summaries did not occupy exactly one row above the table at width %d: %s", width, view)
		}
		return lines[reviewLine+1], strings.Split(strings.TrimSuffix(view, "\n"), "\n")
	}

	wideSummary, _ := assertSummaryRow(260)
	for _, repo := range repos {
		if !strings.Contains(wideSummary, repo+" ·") {
			t.Errorf("horizontal summary row omitted %q: %s", repo, wideSummary)
		}
	}
	_, narrowLines := assertSummaryRow(80)
	for i, line := range narrowLines {
		if width := ansi.StringWidth(line); width > 80 {
			t.Errorf("narrow view line %d is %d cells wide: %q", i+1, width, line)
		}
	}
}

func TestBackgroundSummaryPrefetchIsBoundedAndDoesNotDelayList(t *testing.T) {
	now := time.Now()
	prs := make([]domain.PR, 18)
	for i := range prs {
		prs[i] = testUIPr(i+1, now.Add(-time.Duration(i)*time.Minute))
		prs[i].Title = fmt.Sprintf("PR %d", i+1)
		prs[i].Assignees = domain.AssigneeSet{Logins: []string{"me"}, Complete: true}
	}
	enricher := testutil.Enricher{Fn: func(context.Context, domain.FieldKey) (any, domain.Provenance, error) {
		return nil, domain.Provenance{}, &domain.FetchError{Kind: domain.ErrorPermanent, Message: "summary failed"}
	}}
	m := New(Options{Config: config.Default(), Repos: []string{"owner/repo"}, Enricher: enricher})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	m = next.(Model)
	next, _ = m.Update(listMsg{repo: "owner/repo", generation: m.sections[0].field.Generation, value: prs, viewer: "me", at: now})
	m = next.(Model)
	if m.sections[0].field.Phase != domain.Ready || !strings.Contains(m.View(), "owner/repo#1") {
		t.Fatalf("list did not render before enrichment finished: %s", m.View())
	}
	if got := len(m.fieldInFlight); got != maxConcurrentFieldFetches {
		t.Fatalf("summary concurrency = %d, want cap %d", got, maxConcurrentFieldFetches)
	}
	if got := len(m.activityInFlight); got > maxConcurrentActivityFetches {
		t.Fatalf("activity concurrency = %d, exceeds cap %d", got, maxConcurrentActivityFetches)
	}
	last := prs[len(prs)-1]
	for _, name := range []string{"checks", "reviews", "review_threads"} {
		key := domain.FieldKey{Item: last.ID, Name: name, Revision: last.HeadSHA}
		if !m.fieldQueued[key] {
			t.Errorf("off-screen PR summary %q was not queued", name)
		}
	}

	var reviewKey domain.FieldKey
	for key := range m.fieldInFlight {
		if key.Name == "reviews" {
			reviewKey = key
			break
		}
	}
	if reviewKey.Name == "" {
		t.Fatal("selected PR review summary was not started")
	}
	generation := m.fieldInFlight[reviewKey]
	failed := m.fetchField(reviewKey, generation, m.fieldAttempts[reviewKey])()
	next, _ = m.Update(failed)
	m = next.(Model)
	if field := m.fields[reviewKey]; field == nil || field.Phase != domain.Failed {
		t.Fatalf("terminal summary failure did not settle: %+v", field)
	}
	if text, _ := m.reviewCell(prs[0]); text != "failed" {
		t.Fatalf("failed review summary rendered as %q", text)
	}
}

func TestBackgroundPrefetchPromotesVisibleRowsAheadOfOffscreenRows(t *testing.T) {
	now := time.Now()
	prs := make([]domain.PR, 24)
	for i := range prs {
		prs[i] = testUIPr(i+1, now.Add(-time.Duration(i)*time.Minute))
		prs[i].Assignees = domain.AssigneeSet{Logins: []string{"me"}, Complete: true}
	}
	m := New(Options{Config: config.Default(), Repos: []string{"owner/repo"}})
	m.width, m.height, m.viewer = 100, 12, "me"
	m.sections[0].field.Succeed(m.sections[0].field.Generation, prs, domain.Provenance{Version: "list-v1", FreshUntil: now.Add(time.Hour)})
	m.selected, m.hasSelection = prs[12].ID, true
	m.opts.Enricher = testutil.Enricher{}
	m.enqueueListSummaries(prs)
	visibleRows := m.visibleRowsForPrefetch()
	if len(visibleRows) == 0 || len(visibleRows) >= len(prs) {
		t.Fatalf("expected a partial visible viewport, got %d of %d rows", len(visibleRows), len(prs))
	}
	visible := make(map[domain.ItemID]bool, len(visibleRows))
	for _, pr := range visibleRows {
		visible[pr.ID] = true
	}
	if !visible[m.selected] {
		t.Fatalf("selected row %v is outside calculated viewport: %+v", m.selected, visibleRows)
	}
	m.prioritizeQueuedFields()
	lastVisible, firstOffscreen := -1, len(m.fieldQueue)
	for i, key := range m.fieldQueue {
		if visible[key.Item] {
			lastVisible = i
		} else if i < firstOffscreen {
			firstOffscreen = i
		}
	}
	if lastVisible < 0 || firstOffscreen == len(m.fieldQueue) || lastVisible >= firstOffscreen {
		t.Fatalf("visible summary queue is not ahead of off-screen work: visible through %d, off-screen starts %d", lastVisible, firstOffscreen)
	}
}

func TestSelectedTriageFieldsFailIndependently(t *testing.T) {
	id := domain.ItemID{Source: "github", Host: "github.com", Repo: "one/repo", Number: 1}
	m := New(Options{Config: config.Default(), Repos: []string{"one/repo"}, Enricher: testutil.Enricher{Fn: func(context.Context, domain.FieldKey) (any, domain.Provenance, error) {
		return nil, domain.Provenance{}, &domain.FetchError{Kind: domain.ErrorTransient, Message: "enrichment failed"}
	}}})
	next, _ := m.Update(listMsg{repo: "one/repo", generation: m.sections[0].field.Generation, value: []domain.PR{{ID: id, Title: "PR", HeadSHA: "abc"}}, at: time.Now()})
	m = next.(Model)
	key := domain.FieldKey{Item: id, Name: "activity", Revision: "abc"}
	if m.fields[key] == nil || !m.fields[key].Phase.Active() {
		t.Fatal("selected activity field not requested")
	}
	msg := m.fetchField(key, m.fields[key].Generation, 1)()
	next, _ = m.Update(msg)
	m = next.(Model)
	if m.fields[key].Phase != domain.Cooldown || m.sections[0].field.Phase != domain.Ready {
		t.Fatalf("field failure blocked list: %s %s", m.fields[key].Phase, m.sections[0].field.Phase)
	}
}

func TestNoRepositoryShowsSetup(t *testing.T) {
	m := New(Options{Config: config.Default()})
	if !strings.Contains(m.View(), "No GitHub repository found") {
		t.Fatal("missing setup state")
	}
}

func TestHotkeyRunsSelectedURLCommand(t *testing.T) {
	runner := &testutil.Runner{}
	m := New(Options{Config: config.Default(), Repos: []string{"one/repo"}, Runner: runner})
	pr := domain.PR{ID: domain.ItemID{Source: "github", Host: "github.com", Repo: "one/repo", Number: 7}, URL: "https://github.com/one/repo/pull/7"}
	next, _ := m.Update(listMsg{repo: "one/repo", generation: m.sections[0].field.Generation, value: []domain.PR{pr}, at: time.Now()})
	m = next.(Model)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m = next.(Model)
	if cmd == nil {
		t.Fatal("hotkey did not launch")
	}
	result := cmd()
	if batch, ok := result.(tea.BatchMsg); ok {
		for _, part := range batch {
			part()
		}
	}
	got := runner.Snapshot()
	if len(got) != 1 || len(got[0].Args) != 1 || got[0].Args[0] != pr.URL {
		t.Fatalf("wrong command: %+v", got)
	}
}

func TestReviewFilterIncludesIncompleteRowsAsPending(t *testing.T) {
	now := time.Now()
	prs := []domain.PR{testUIPr(1, now), testUIPr(2, now), testUIPr(3, now)}
	m := loadedModel(t, prs)
	m.viewer = "me"
	putField(&m, prs[0], "review_requests", domain.LoginSet{Logins: []string{"someone-else"}, Complete: true, ViewerLogin: "me"}, now)
	putField(&m, prs[1], "review_requests", domain.LoginSet{Logins: []string{"me"}, Complete: true, ViewerLogin: "me"}, now)
	m.filter = triage.FilterReviewRequested
	m.ensureSelection()
	sections := m.projection()
	if len(sections.Review) != 2 || sections.Review[0].PR.ID.Number != 2 || sections.Review[1].PR.ID.Number != 3 || !sections.Review[1].FilterPending {
		t.Fatalf("review filter rows = %+v", sections.Review)
	}
	view := m.View()
	if !strings.Contains(view, "REVIEW REQUESTED FROM ME") || !strings.Contains(view, "1 pending") {
		t.Fatalf("filter or pending state missing: %s", view)
	}
}

func TestActivitySectionAndSelectionSurviveRankingUpdate(t *testing.T) {
	now := time.Now()
	prs := []domain.PR{testUIPr(1, now), testUIPr(2, now)}
	m := loadedModel(t, prs)
	selected := m.selected
	key := domain.FieldKey{Item: prs[1].ID, Name: "checks", Revision: prs[1].HeadSHA}
	f := domain.NewField[any]()
	generation := f.Queue()
	f.Begin(generation)
	m.fields[key] = &f
	next, _ := m.Update(fieldMsg{key: key, generation: generation, value: domain.CheckSummary{Failed: 1, Complete: true}, provenance: domain.Provenance{Source: "github", Version: "checks-v1", FetchedAt: now, FreshUntil: now.Add(time.Hour)}, at: now})
	m = next.(Model)
	if m.selected != selected {
		t.Fatalf("selection changed after ranking update: got %v, want %v", m.selected, selected)
	}
	rows := m.allRows()
	if rows[0].ID.Number != 2 || m.selected.Number != 1 {
		t.Fatalf("rank did not update while preserving selection: rows=%v selected=%v", []int{rows[0].ID.Number, rows[1].ID.Number}, m.selected)
	}
	active := domain.ActivityEvidence{State: domain.ActivityActive, Author: "fullsend-ai-coder", RunID: "run-3", StartedAt: now, URL: "https://github.com/owner/repo/pull/1#issuecomment-3", FetchedAt: now, FreshUntil: now.Add(time.Hour), Complete: true}
	putField(&m, prs[0], "activity", active, now)
	sections := m.projection()
	if len(sections.Active) != 1 || sections.Active[0].PR.ID.Number != 1 || len(sections.Review) != 1 || sections.Review[0].PR.ID.Number != 2 {
		t.Fatalf("active-agent section = %+v; review = %+v", sections.Active, sections.Review)
	}
	m.selected = prs[0].ID
	m.ensureSelection()
	if view := m.View(); !strings.Contains(view, "ACTIVE FULLSEND AGENTS") || !strings.Contains(view, active.URL) {
		t.Fatalf("active evidence absent from view: %s", view)
	}
}

func TestStarSnoozeHiddenViewAndManualUnsnooze(t *testing.T) {
	now := time.Now()
	pr := testUIPr(8, now)
	m := loadedModel(t, []domain.PR{pr})
	runner := &testutil.Runner{}
	m.opts.Runner = runner
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if !m.opts.Reminders.Snapshot()[pr.ID].Starred {
		t.Fatal("star was not persisted to local store")
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	m = next.(Model)
	if m.mode != "snooze" {
		t.Fatal("snooze duration prompt did not open")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = next.(Model)
	entry := m.opts.Reminders.Snapshot()[pr.ID]
	if !entry.Starred || entry.SnoozedUntil.Sub(now) < 23*time.Hour {
		t.Fatalf("star or one-day snooze was not retained: %+v", entry)
	}
	if len(m.allRows()) != 0 {
		t.Fatal("snoozed PR remained in the normal view")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	m = next.(Model)
	if !strings.Contains(m.View(), "SNOOZED") || !strings.Contains(m.View(), "★ starred") {
		t.Fatalf("hidden snoozed item is not inspectable: %s", m.View())
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	m = next.(Model)
	if !m.opts.Reminders.Snapshot()[pr.ID].SnoozedUntil.IsZero() {
		t.Fatal("manual unsnooze did not clear wake time")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	m = next.(Model)
	if len(m.allRows()) != 1 || m.allRows()[0].ID != pr.ID {
		t.Fatal("unsnoozed PR did not return to normal view")
	}
	if got := runner.Snapshot(); len(got) != 0 {
		t.Fatalf("local reminder actions launched remote-related commands: %+v", got)
	}
}

func TestSnoozeWakeUsesInjectedClock(t *testing.T) {
	now := time.Date(2026, 11, 3, 15, 0, 0, 0, time.UTC)
	clock := &testClock{now: now}
	pr := testUIPr(5, now)
	store := reminder.NewMemoryStore()
	if err := store.Snooze(pr.ID, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	m := New(Options{Config: config.Default(), Repos: []string{"owner/repo"}, Clock: clock, Reminders: store})
	m.sections[0].field.Succeed(m.sections[0].field.Generation, []domain.PR{pr}, domain.Provenance{Version: "list-v1", FreshUntil: now.Add(time.Hour)})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(Model)
	if !m.wakeTickRunning {
		t.Fatal("no wake timer scheduled")
	}
	clock.now = now.Add(2 * time.Hour)
	next, _ = m.Update(wakeMsg{generation: m.wakeGeneration})
	m = next.(Model)
	if len(m.allRows()) != 1 || m.allRows()[0].ID != pr.ID {
		t.Fatal("expired snooze did not automatically reappear")
	}
}

func TestExpiredSnoozeIsVisibleAfterReload(t *testing.T) {
	now := time.Date(2026, 11, 3, 15, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "reminders.json")
	pr := testUIPr(6, now)
	first := reminder.NewFileStore(path)
	if err := first.Load(); err != nil {
		t.Fatal(err)
	}
	if err := first.Snooze(pr.ID, now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	reloaded := reminder.NewFileStore(path)
	if err := reloaded.Load(); err != nil {
		t.Fatal(err)
	}
	m := New(Options{Config: config.Default(), Repos: []string{"owner/repo"}, Clock: &testClock{now: now}, Reminders: reloaded})
	m.sections[0].field.Succeed(m.sections[0].field.Generation, []domain.PR{pr}, domain.Provenance{Version: "list-v1", FreshUntil: now.Add(time.Hour)})
	if rows := m.allRows(); len(rows) != 1 || rows[0].ID != pr.ID {
		t.Fatalf("expired snooze did not reappear after reload: %+v", rows)
	}
	if hidden := m.projection(); len(hidden.Snoozed) != 0 {
		t.Fatalf("expired PR remained in snoozed view: %+v", hidden.Snoozed)
	}
}

func TestReminderLoadWarningSurvivesListSuccess(t *testing.T) {
	warning := errors.New("invalid JSON; backup retained")
	m := New(Options{Config: config.Default(), Repos: []string{"owner/repo"}, ReminderLoadError: warning})
	pr := testUIPr(2, time.Now())
	next, _ := m.Update(listMsg{repo: "owner/repo", generation: m.sections[0].field.Generation, value: []domain.PR{pr}, at: time.Now()})
	m = next.(Model)
	if !strings.Contains(m.View(), "backup retained") {
		t.Fatalf("reminder warning disappeared after list load: %s", m.View())
	}
}

func TestActivityIndicatorsSeparatePendingInactiveStaleAndFailed(t *testing.T) {
	now := time.Now()
	pr := testUIPr(14, now)
	m := loadedModel(t, []domain.PR{pr})
	if got := m.rowStatus(m.projection().Review[0]); !strings.Contains(got, "activity pending") {
		t.Fatalf("unrequested activity indicator = %q", got)
	}
	for _, tc := range []struct {
		state domain.ActivityState
		want  string
	}{{domain.ActivityUnknown, "activity unknown"}, {domain.ActivityInactive, "agent inactive"}, {domain.ActivityStale, "agent status stale"}} {
		putField(&m, pr, "activity", domain.ActivityEvidence{State: tc.state, StartedAt: now.Add(-7 * time.Hour)}, now)
		row := m.projection().Review[0]
		if got := m.rowStatus(row); !strings.Contains(got, tc.want) {
			t.Errorf("%s indicator = %q, want %q", tc.state, got, tc.want)
		}
	}
	key := domain.FieldKey{Item: pr.ID, Name: "activity", Revision: pr.HeadSHA}
	f := domain.NewField[any]()
	generation := f.Queue()
	f.Begin(generation)
	f.Fail(generation, &domain.FetchError{Kind: domain.ErrorPermanent, Message: "comment query failed"}, time.Time{})
	m.fields[key] = &f
	if got := m.rowStatus(m.projection().Review[0]); !strings.Contains(got, "activity failed") {
		t.Fatalf("failed activity indicator = %q", got)
	}
}

type testClock struct{ now time.Time }

func (c *testClock) Now() time.Time { return c.now }

func loadedModel(t *testing.T, prs []domain.PR) Model {
	t.Helper()
	repo := "owner/repo"
	m := New(Options{Config: config.Default(), Repos: []string{repo}})
	next, _ := m.Update(listMsg{repo: repo, generation: m.sections[0].field.Generation, value: prs, at: time.Now()})
	return next.(Model)
}

func testUIPr(number int, updated time.Time) domain.PR {
	return domain.PR{ID: domain.ItemID{Source: "github", Host: "github.com", Repo: "owner/repo", Number: number}, Title: "PR", Author: "contributor", UpdatedAt: updated, HeadSHA: fmt.Sprintf("head-%d", number), URL: fmt.Sprintf("https://github.com/owner/repo/pull/%d", number)}
}

func putField(m *Model, pr domain.PR, name string, value any, now time.Time) {
	key := domain.FieldKey{Item: pr.ID, Name: name, Revision: pr.HeadSHA}
	f := domain.NewField[any]()
	generation := f.Queue()
	f.Begin(generation)
	f.Succeed(generation, value, domain.Provenance{Source: "github", Version: name + "-v1", FetchedAt: now, FreshUntil: now.Add(time.Hour)})
	m.fields[key] = &f
}
