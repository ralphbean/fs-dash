package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/fullsend-ai/fs-dash/internal/config"
	"github.com/fullsend-ai/fs-dash/internal/domain"
	"github.com/fullsend-ai/fs-dash/internal/testutil"
	"github.com/fullsend-ai/fs-dash/internal/triage"
)

func TestActiveRunTreeRendersAndAdvancesWithoutFetching(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	now := time.Date(2026, 11, 6, 17, 0, 0, 0, time.UTC)
	clock := &testClock{now: now}
	calls := 0
	cfg := config.Default()
	cfg.Color = "auto"
	pr := testUIPr(19, now)
	workflowURL := "https://github.com/owner/repo/actions/runs/42"
	m := New(Options{
		Config: cfg,
		Repos:  []string{"owner/repo"},
		Clock:  clock,
		Enricher: testutil.Enricher{Fn: func(_ context.Context, _ domain.FieldKey) (any, domain.Provenance, error) {
			calls++
			return nil, domain.Provenance{}, nil
		}},
	})
	m.width, m.height, m.filter = 120, 32, triage.FilterAll
	m.visual.compact = false
	m.sections[0].field.Succeed(m.sections[0].field.Generation, []domain.PR{pr}, domain.Provenance{FreshUntil: now.Add(time.Hour)})
	m.selected, m.hasSelection = pr.ID, true
	activity := domain.ActivityEvidence{
		State:        domain.ActivityActive,
		RunsComplete: true,
		Complete:     true,
		Runs: []domain.ActivityRun{
			{RunID: "run-code", Name: "Code", WorkflowURL: workflowURL, StartedAt: now.Add(-125 * time.Second)},
			{RunID: "run-review", Name: "Review", StartedAt: now.Add(-30 * time.Second)},
		},
	}
	putField(&m, pr, "activity", activity, now)

	view := m.View()
	plain := ansi.Strip(view)
	if strings.Contains(view, "\x1b[") {
		t.Fatalf("NO_COLOR view contains ANSI styling: %q", view)
	}
	if !strings.Contains(view, ansi.SetHyperlink(workflowURL)) {
		t.Fatalf("workflow URL is not rendered as a terminal hyperlink: %q", view)
	}
	lines := strings.Split(strings.TrimSuffix(plain, "\n"), "\n")
	prLine, codeLine, reviewLine := -1, -1, -1
	for i, line := range lines {
		switch {
		case strings.Contains(line, "owner/repo#19"):
			prLine = i
		case strings.Contains(line, "Code") && strings.Contains(line, "2m 05s"):
			codeLine = i
		case strings.Contains(line, "Review") && strings.Contains(line, "30s"):
			reviewLine = i
		}
	}
	if prLine < 0 || codeLine <= prLine || reviewLine <= codeLine {
		t.Fatalf("PR and run tree rows are missing or out of order: %s", plain)
	}
	prIdentityColumn := strings.Index(lines[prLine], "> owner/repo#19")
	if prIdentityColumn < 0 || prIdentityColumn != strings.Index(lines[codeLine], "├─") || prIdentityColumn != strings.Index(lines[reviewLine], "└─") {
		t.Fatalf("run children are not aligned under the PR identity column: %s", plain)
	}
	if strings.Contains(lines[reviewLine], "workflow") {
		t.Fatalf("run without a workflow URL is shown as linked: %s", lines[reviewLine])
	}

	// The local clock message updates presentation while leaving the activity field alone.
	next, _ := m.Update(runClockTickMsg{})
	m = next.(Model)
	if !m.runClockTickRunning {
		t.Fatal("active run did not schedule a local clock refresh")
	}
	clock.now = clock.now.Add(time.Second)
	next, _ = m.Update(runClockTickMsg{})
	m = next.(Model)
	if got := ansi.Strip(m.View()); !strings.Contains(got, "2m 06s") {
		t.Fatalf("elapsed label did not advance after a local tick: %s", got)
	}
	if calls != 0 {
		t.Fatalf("local elapsed-time refresh made %d enrichment calls", calls)
	}

	// Narrow terminals keep the readable run labels and stay within the frame width.
	m.width, m.height, m.visual.compact = 40, 24, true
	narrow := m.View()
	if strings.Contains(narrow, "\x1b[") {
		t.Fatalf("narrow NO_COLOR view contains ANSI styling: %q", narrow)
	}
	for i, line := range strings.Split(strings.TrimSuffix(ansi.Strip(narrow), "\n"), "\n") {
		if width := ansi.StringWidth(line); width > 40 {
			t.Errorf("narrow line %d is %d cells wide: %q", i+1, width, line)
		}
	}
	if got := ansi.Strip(narrow); !strings.Contains(got, "Code") || !strings.Contains(got, "Review") {
		t.Fatalf("compact view lost readable run names: %s", got)
	}

	// An incomplete comment scan remains explicit beneath the known runs.
	activity.RunsComplete = false
	putField(&m, pr, "activity", activity, clock.Now())
	if got := ansi.Strip(m.View()); !strings.Contains(got, "run history may be incomplete") {
		t.Fatalf("incomplete run history is not indicated: %s", got)
	}

	// Once all runs are terminal, the active section disappears and its clock stops.
	putField(&m, pr, "activity", domain.ActivityEvidence{State: domain.ActivityInactive, Complete: true, RunsComplete: true}, clock.Now())
	next, _ = m.Update(runClockTickMsg{})
	m = next.(Model)
	if m.runClockTickRunning {
		t.Fatal("clock kept ticking after the active-agent section disappeared")
	}
	if got := ansi.Strip(m.View()); strings.Contains(got, "ACTIVE FULLSEND AGENTS") || strings.Contains(got, "run-code") {
		t.Fatalf("completed runs remained in the active section: %s", got)
	}
}
