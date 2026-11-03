package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fullsend-ai/fs-dash/internal/config"
	"github.com/fullsend-ai/fs-dash/internal/diag"
	"github.com/fullsend-ai/fs-dash/internal/domain"
	"github.com/fullsend-ai/fs-dash/internal/testutil"
)

func TestFailedSectionSettlesWithoutBlockingOtherSection(t *testing.T) {
	logs := diag.New(20)
	lister := testutil.PRLister{Fn: func(_ context.Context, repo string) ([]domain.PR, error) {
		if repo == "one/repo" {
			return nil, &domain.FetchError{Kind: domain.ErrorTransient, Message: "502 Bad Gateway"}
		}
		return []domain.PR{{ID: domain.ItemID{Source: "github", Host: "github.com", Repo: repo, Number: 2}, Title: "Ready", URL: "https://github.com/two/repo/pull/2"}}, nil
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
	m := New(Options{Config: config.Default(), Repos: []string{"one/repo"}, Lister: testutil.PRLister{Fn: func(context.Context, string) ([]domain.PR, error) { return nil, errors.New("unused") }}})
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

func TestSelectedFieldFailureSettlesIndependently(t *testing.T) {
	id := domain.ItemID{Source: "github", Host: "github.com", Repo: "one/repo", Number: 1}
	m := New(Options{Config: config.Default(), Repos: []string{"one/repo"}, Enricher: testutil.Enricher{Fn: func(context.Context, domain.FieldKey) (any, domain.Provenance, error) {
		return nil, domain.Provenance{}, &domain.FetchError{Kind: domain.ErrorTransient, Message: "enrichment failed"}
	}}})
	next, _ := m.Update(listMsg{repo: "one/repo", generation: m.sections[0].field.Generation, value: []domain.PR{{ID: id, Title: "PR", HeadSHA: "abc"}}, at: time.Now()})
	m = next.(Model)
	key := domain.FieldKey{Item: id, Name: "review_summary", Revision: "abc"}
	if m.fields[key] == nil || !m.fields[key].Phase.Active() {
		t.Fatal("selected field not requested")
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
