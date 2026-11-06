package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/fullsend-ai/fs-dash/internal/config"
	"github.com/fullsend-ai/fs-dash/internal/domain"
	"github.com/muesli/termenv"
)

type visualRole string

const (
	roleFocus       visualRole = "focus"
	roleAccent      visualRole = "accent"
	roleNeutral     visualRole = "neutral"
	roleMuted       visualRole = "muted"
	roleLink        visualRole = "link"
	roleQueued      visualRole = "queued"
	rolePending     visualRole = "pending"
	roleSuccess     visualRole = "success"
	roleStale       visualRole = "stale"
	roleCooling     visualRole = "cooldown"
	roleWarning     visualRole = "warning"
	roleFailure     visualRole = "failure"
	roleCanceled    visualRole = "canceled"
	roleUnrequested visualRole = "unrequested"
	roleBorder      visualRole = "border"
	roleSurface     visualRole = "surface"
)

type visualTheme struct {
	renderer *lipgloss.Renderer
	colors   map[visualRole]lipgloss.Color
	compact  bool
	noColor  bool
}

func newVisualTheme(cfg config.Config) visualTheme {
	renderer := lipgloss.NewRenderer(os.Stdout)
	noColor := cfg.Color == "never"
	if _, noColorEnv := os.LookupEnv("NO_COLOR"); noColorEnv {
		noColor = true
	}
	if noColor {
		renderer.SetColorProfile(termenv.Ascii)
	}

	palette := map[visualRole]lipgloss.Color{
		roleFocus:       "#f8fafc",
		roleAccent:      "#7dd3fc",
		roleNeutral:     "#e2e8f0",
		roleMuted:       "#94a3b8",
		roleLink:        "#60a5fa",
		roleQueued:      "#a5b4fc",
		rolePending:     "#c4b5fd",
		roleSuccess:     "#86efac",
		roleStale:       "#facc15",
		roleCooling:     "#fdba74",
		roleWarning:     "#fbbf24",
		roleFailure:     "#fda4af",
		roleCanceled:    "#cbd5e1",
		roleUnrequested: "#64748b",
		roleBorder:      "#475569",
		roleSurface:     "#1e293b",
	}
	if cfg.Theme == "light" {
		palette = map[visualRole]lipgloss.Color{
			roleFocus:       "#111827",
			roleAccent:      "#075985",
			roleNeutral:     "#1f2937",
			roleMuted:       "#64748b",
			roleLink:        "#1d4ed8",
			roleQueued:      "#4338ca",
			rolePending:     "#6d28d9",
			roleSuccess:     "#15803d",
			roleStale:       "#a16207",
			roleCooling:     "#c2410c",
			roleWarning:     "#a16207",
			roleFailure:     "#be123c",
			roleCanceled:    "#475569",
			roleUnrequested: "#94a3b8",
			roleBorder:      "#94a3b8",
			roleSurface:     "#e2e8f0",
		}
	}
	return visualTheme{renderer: renderer, colors: palette, noColor: noColor}
}

func (t visualTheme) style(role visualRole) lipgloss.Style {
	style := t.renderer.NewStyle().Foreground(t.colors[role])
	if role == roleSurface {
		return t.renderer.NewStyle().Background(t.colors[role])
	}
	switch role {
	case roleFocus, roleAccent, roleFailure, roleWarning, roleQueued:
		style = style.Bold(true)
	}
	return style
}

func (t visualTheme) paint(role visualRole, text string) string {
	return t.style(role).Render(text)
}

func phaseRole(phase domain.Phase, stale bool) visualRole {
	if phase == domain.Ready && stale {
		return roleStale
	}
	switch phase {
	case domain.Queued:
		return roleQueued
	case domain.Fetching:
		return rolePending
	case domain.Ready:
		return roleSuccess
	case domain.Cooldown:
		return roleCooling
	case domain.Failed:
		return roleFailure
	case domain.Canceled:
		return roleCanceled
	case domain.Unrequested:
		return roleUnrequested
	default:
		return roleMuted
	}
}

func logRole(level string) visualRole {
	switch strings.ToLower(level) {
	case "debug":
		return roleMuted
	case "info":
		return roleLink
	case "warn", "warning":
		return roleWarning
	case "error", "fatal":
		return roleFailure
	default:
		return roleNeutral
	}
}

func (t visualTheme) panel(title string, lines []string, width int) []string {
	if width < 1 {
		width = 1
	}
	if t.compact || width < 52 {
		out := []string{t.fit(t.paint(roleAccent, title), width), t.divider(width)}
		for _, line := range lines {
			out = append(out, t.fit(line, width))
		}
		return out
	}
	inner := width - 4
	if inner < 1 {
		inner = 1
	}
	label := "─ " + t.fit(title, max(0, width-6)) + " "
	top := "╭" + label + strings.Repeat("─", max(0, width-2-lipgloss.Width(label))) + "╮"
	out := []string{t.paint(roleBorder, top)}
	for _, line := range lines {
		body := t.fit(line, inner)
		leftPad := inner - lipgloss.Width(body)
		out = append(out, t.paint(roleBorder, "│ ")+body+strings.Repeat(" ", max(0, leftPad))+t.paint(roleBorder, " │"))
	}
	out = append(out, t.paint(roleBorder, "╰"+strings.Repeat("─", max(0, width-2))+"╯"))
	return out
}

func (t visualTheme) header(title, scope string, width int) []string {
	lines := []string{t.paint(roleNeutral, scope)}
	return t.panel(t.paint(roleAccent, title), lines, width)
}

func (t visualTheme) badge(label string, role visualRole) string {
	return t.paint(role, label)
}

func (t visualTheme) context(title string, lines []string, width int) []string {
	return t.panel(title, lines, width)
}

func (t visualTheme) divider(width int) string {
	if width < 1 {
		return ""
	}
	return t.paint(roleBorder, strings.Repeat("─", width))
}

func (t visualTheme) footer(text string, width int) string {
	return t.paint(roleMuted, t.fit(text, width))
}

func (t visualTheme) fit(text string, width int) string {
	if width < 0 {
		width = 0
	}
	return ansi.Truncate(text, width, "…")
}

func (t visualTheme) selected(marker, number, title, author, updated, indicator string, indicatorRole visualRole, width int) string {
	paintOnSurface := func(role visualRole, text string) string {
		return t.style(role).Background(t.colors[roleSurface]).Render(text)
	}
	markerRole := roleMuted
	if marker == ">" {
		markerRole = roleFocus
	}
	prefix := fmt.Sprintf("%s #%s ", marker, number)
	indicatorText := ""
	if indicator != "" {
		indicatorText = " " + indicator
	}
	available := max(0, width-lipgloss.Width(prefix)-lipgloss.Width(indicatorText))
	meta := " @" + author + " " + updated
	minimumTitle := min(8, max(1, available/2))
	metaWidth := min(lipgloss.Width(meta), max(0, available-minimumTitle))
	titleWidth := max(1, available-metaWidth)
	line := paintOnSurface(markerRole, marker) + paintOnSurface(roleFocus, " ") + paintOnSurface(roleLink, "#"+number) + paintOnSurface(roleFocus, " ") + paintOnSurface(roleNeutral, t.fit(title, titleWidth))
	line += paintOnSurface(roleMuted, ansi.Truncate(meta, metaWidth, ""))
	if indicator != "" {
		line += paintOnSurface(roleFocus, " ") + paintOnSurface(indicatorRole, indicator)
	}
	line = ansi.Truncate(line, width, "…")
	return line + t.style(roleSurface).Render(strings.Repeat(" ", max(0, width-lipgloss.Width(line))))
}

func fitCells(text string, width int) string {
	if width <= 0 {
		return ""
	}
	text = ansi.Truncate(text, width, "…")
	return text + strings.Repeat(" ", max(0, width-lipgloss.Width(text)))
}

func (t visualTheme) withProfile(profile termenv.Profile) visualTheme {
	t.renderer.SetColorProfile(profile)
	return t
}
