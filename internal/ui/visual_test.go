package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/fullsend-ai/fs-dash/internal/config"
	"github.com/fullsend-ai/fs-dash/internal/domain"
	"github.com/fullsend-ai/fs-dash/internal/testutil"
	"github.com/muesli/termenv"
)

func TestVisualRolesDistinguishFieldAndLogStatuses(t *testing.T) {
	phaseRoles := []visualRole{
		phaseRole(domain.Unrequested, false),
		phaseRole(domain.Queued, false),
		phaseRole(domain.Fetching, false),
		phaseRole(domain.Ready, false),
		phaseRole(domain.Ready, true),
		phaseRole(domain.Cooldown, false),
		phaseRole(domain.Failed, false),
		phaseRole(domain.Canceled, false),
	}
	seen := map[visualRole]bool{}
	for _, role := range phaseRoles {
		if seen[role] {
			t.Fatalf("phase status shares a semantic treatment: %q", role)
		}
		seen[role] = true
	}

	logRoles := []visualRole{logRole("debug"), logRole("info"), logRole("warn"), logRole("error")}
	seen = map[visualRole]bool{}
	for _, role := range logRoles {
		if seen[role] {
			t.Fatalf("log level shares a semantic treatment: %q", role)
		}
		seen[role] = true
	}
}

func TestRenderedSectionStatesUseDistinctSemanticRoles(t *testing.T) {
	m := New(Options{Config: config.Default()})
	now := time.Now()
	current := domain.NewField[[]domain.PR]()
	current.Phase, current.HasValue, current.Value = domain.Ready, true, []domain.PR{{}}
	current.Provenance.FreshUntil = now.Add(time.Hour)
	stale := current
	stale.Provenance.FreshUntil = now.Add(-time.Hour)
	cooldown := domain.NewField[[]domain.PR]()
	cooldown.Phase = domain.Cooldown
	cooldown.Error = &domain.FetchError{Message: "retrying"}
	failed := domain.NewField[[]domain.PR]()
	failed.Phase = domain.Failed
	failed.Error = &domain.FetchError{Message: "failed"}
	states := []domain.Field[[]domain.PR]{
		domain.NewField[[]domain.PR](),
		{Phase: domain.Queued},
		{Phase: domain.Fetching},
		current,
		stale,
		cooldown,
		failed,
		{Phase: domain.Canceled},
		{Phase: domain.Ready},
	}
	seen := map[visualRole]bool{}
	for _, field := range states {
		_, role := m.sectionStatus(section{field: field})
		if seen[role] {
			t.Fatalf("rendered section states share a semantic treatment: %q", role)
		}
		seen[role] = true
	}
}

func TestSelectedErrorKeepsStatusColorOnFocusedRow(t *testing.T) {
	theme := newVisualTheme(config.Default()).withProfile(termenv.TrueColor)
	line := theme.selected(">", "9", "A pull request", "author", "Nov 03 12:00", "failed", roleFailure, 60)
	failure := theme.style(roleFailure).Background(theme.colors[roleSurface]).Render("failed")
	if !strings.Contains(line, "48;2;") || !strings.Contains(line, failure) {
		t.Fatalf("focused row lost its background or failure color: %q", line)
	}
}

func TestNoColorKeepsStatusLabelsAndFocusMarker(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	cfg := config.Default()
	cfg.Color = "auto"
	id := domain.ItemID{Source: "github", Host: "github.com", Repo: "owner/repo", Number: 7}
	m := New(Options{Config: cfg, Repos: []string{"owner/repo"}, Enricher: testutil.Enricher{}})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 32})
	m = next.(Model)
	m.sections[0].field.Succeed(m.sections[0].field.Generation, []domain.PR{{ID: id, Title: "A selected pull request", HeadSHA: "head", URL: "https://github.com/owner/repo/pull/7"}}, domain.Provenance{FreshUntil: time.Now().Add(time.Hour)})
	m.ensureSelection()
	m.emojiMode = false
	key := domain.FieldKey{Item: id, Name: "reviews", Revision: "head"}
	f := domain.NewField[any]()
	generation := f.Queue()
	f.Begin(generation)
	f.Fail(generation, &domain.FetchError{Kind: domain.ErrorPermanent, Message: "review fetch failed"}, time.Time{})
	m.fields[key] = &f
	m.selected = id
	view := m.View()
	if strings.Contains(view, "\x1b[") {
		t.Fatalf("NO_COLOR output contains ANSI styling: %q", view)
	}
	plain := ansi.Strip(view)
	for _, label := range []string{"> owner/repo#7", "open", "failed"} {
		if !strings.Contains(plain, label) {
			t.Fatalf("no-color output lost %q: %s", label, view)
		}
	}
}

func TestPaletteAndLayoutAtCommonAndNarrowSizes(t *testing.T) {
	cfg := config.Default()
	dark := newVisualTheme(cfg).withProfile(termenv.TrueColor)
	cfg.Theme = "light"
	light := newVisualTheme(cfg).withProfile(termenv.TrueColor)
	if dark.colors[roleFocus] == light.colors[roleFocus] || dark.colors[roleBorder] == light.colors[roleBorder] {
		t.Fatal("dark and light palettes did not change foreground and border colors")
	}

	rows := make([]domain.PR, 100)
	for i := range rows {
		rows[i] = domain.PR{ID: domain.ItemID{Source: "github", Host: "github.com", Repo: "owner/repo", Number: i + 1}, Title: strings.Repeat("wide title ", 8), Author: "contributor", UpdatedAt: time.Now(), URL: "https://github.com/owner/repo/pull/1"}
	}
	m := New(Options{Config: config.Default(), Repos: []string{"owner/repo"}})
	m.visual = m.visual.withProfile(termenv.TrueColor)
	m.sections[0].field.Succeed(m.sections[0].field.Generation, rows, domain.Provenance{FreshUntil: time.Now().Add(time.Hour)})
	m.selected = rows[60].ID
	m.hasSelection = true
	for _, size := range []struct{ width, height int }{{80, 24}, {40, 12}} {
		next, _ := m.Update(tea.WindowSizeMsg{Width: size.width, Height: size.height})
		m = next.(Model)
		view := m.View()
		plain := ansi.Strip(view)
		if got := strings.Count(view, "\n"); got > size.height {
			t.Fatalf("%dx%d view has %d lines: %s", size.width, size.height, got, view)
		}
		for n, line := range strings.Split(strings.TrimSuffix(plain, "\n"), "\n") {
			if width := lipgloss.Width(line); width > size.width {
				t.Fatalf("%dx%d line %d wraps to %d cells: %q", size.width, size.height, n+1, width, line)
			}
		}
		if !strings.Contains(plain, "owner/repo#61") {
			t.Fatalf("selected row is not visible after resize to %dx%d: %s", size.width, size.height, fmt.Sprint(plain))
		}
	}
}
