package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fullsend-ai/fs-dash/internal/config"
	"github.com/fullsend-ai/fs-dash/internal/diag"
	"github.com/fullsend-ai/fs-dash/internal/domain"
	"github.com/fullsend-ai/fs-dash/internal/fetch"
	"github.com/fullsend-ai/fs-dash/internal/ports"
)

type Options struct {
	Config    config.Config
	Repos     []string
	Lister    ports.PRLister
	Enricher  ports.Enricher
	Scheduler *fetch.Scheduler
	Runner    ports.CommandRunner
	Logs      *diag.Buffer
	Clock     ports.Clock
}
type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

type section struct {
	repo     string
	field    domain.Field[[]domain.PR]
	attempts int
}
type Model struct {
	opts                            Options
	sections                        []section
	selected                        domain.ItemID
	hasSelection                    bool
	mode                            string
	help, console                   bool
	logLevel                        string
	logOffset, width, height, frame int
	tickRunning                     bool
	status                          string
	fields                          map[domain.FieldKey]*domain.Field[any]
	fieldAttempts                   map[domain.FieldKey]int
	visual                          visualTheme
}
type listMsg struct {
	repo       string
	generation uint64
	value      []domain.PR
	err        error
	at         time.Time
	attempt    int
	duration   time.Duration
}
type fieldMsg struct {
	key        domain.FieldKey
	generation uint64
	value      any
	provenance domain.Provenance
	err        error
	at         time.Time
	attempt    int
}
type fieldRetryMsg struct {
	key        domain.FieldKey
	generation uint64
}
type retryMsg struct {
	repo       string
	generation uint64
}
type tickMsg time.Time
type commandMsg struct {
	key string
	err error
}

var frames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func New(opts Options) Model {
	if opts.Config.Version == 0 {
		opts.Config = config.Default()
	}
	if opts.Config.Theme == "" {
		opts.Config.Theme = "dark"
	}
	if opts.Config.Color == "" {
		opts.Config.Color = "auto"
	}
	if opts.Scheduler == nil {
		opts.Scheduler = fetch.NewScheduler(2, 20*time.Second)
	}
	if opts.Logs == nil {
		opts.Logs = diag.New(opts.Config.LogLimit)
	}
	if opts.Clock == nil {
		opts.Clock = realClock{}
	}
	m := Model{opts: opts, mode: "list", logLevel: "all", width: 100, height: 30, fields: map[domain.FieldKey]*domain.Field[any]{}, fieldAttempts: map[domain.FieldKey]int{}, visual: newVisualTheme(opts.Config)}
	m.visual.compact = m.width < 64
	for _, repo := range opts.Repos {
		f := domain.NewField[[]domain.PR]()
		g := f.Queue()
		f.Begin(g)
		m.sections = append(m.sections, section{repo: repo, field: f, attempts: 1})
		m.log("debug", "queue-list", repo, 1, 0, "queued initial list request")
	}
	return m
}
func spinnerTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, s := range m.sections {
		cmds = append(cmds, m.fetchList(s.repo, s.field.Generation, s.attempts))
	}
	if len(cmds) > 0 {
		cmds = append(cmds, spinnerTick())
	}
	return tea.Batch(cmds...)
}
func (m Model) fetchList(repo string, generation uint64, attempt int) tea.Cmd {
	return func() tea.Msg {
		start := m.opts.Clock.Now()
		value, err := m.opts.Scheduler.Do(context.Background(), "github", fmt.Sprintf("list:%s:%d", repo, generation), func(ctx context.Context) (any, error) { return m.opts.Lister.ListOpenPRs(ctx, repo) })
		var prs []domain.PR
		if value != nil {
			prs = value.([]domain.PR)
		}
		at := m.opts.Clock.Now()
		return listMsg{repo: repo, generation: generation, value: prs, err: err, at: at, attempt: attempt, duration: at.Sub(start)}
	}
}
func (m Model) fetchField(key domain.FieldKey, generation uint64, attempt int) tea.Cmd {
	return func() tea.Msg {
		value, err := m.opts.Scheduler.Do(context.Background(), "github", fmt.Sprintf("%s:%d", key.String(), generation), func(ctx context.Context) (any, error) {
			v, p, e := m.opts.Enricher.FetchField(ctx, key)
			return struct {
				v any
				p domain.Provenance
			}{v, p}, e
		})
		if err != nil {
			return fieldMsg{key: key, generation: generation, err: err, at: m.opts.Clock.Now(), attempt: attempt}
		}
		result := value.(struct {
			v any
			p domain.Provenance
		})
		return fieldMsg{key: key, generation: generation, value: result.v, provenance: result.p, at: m.opts.Clock.Now(), attempt: attempt}
	}
}
func (m *Model) section(repo string) *section {
	for i := range m.sections {
		if m.sections[i].repo == repo {
			return &m.sections[i]
		}
	}
	return nil
}
func (m Model) allRows() []domain.PR {
	var rows []domain.PR
	for _, s := range m.sections {
		if s.field.HasValue {
			rows = append(rows, s.field.Value...)
		}
	}
	return rows
}
func (m *Model) ensureSelection() {
	rows := m.allRows()
	if len(rows) == 0 {
		m.hasSelection = false
		return
	}
	for _, pr := range rows {
		if m.hasSelection && pr.ID == m.selected {
			return
		}
	}
	m.selected, m.hasSelection = rows[0].ID, true
}
func (m Model) selectedPR() (domain.PR, bool) {
	for _, pr := range m.allRows() {
		if m.hasSelection && pr.ID == m.selected {
			return pr, true
		}
	}
	return domain.PR{}, false
}
func (m *Model) move(delta int) {
	rows := m.allRows()
	if len(rows) == 0 {
		return
	}
	i := 0
	for n, row := range rows {
		if row.ID == m.selected {
			i = n
			break
		}
	}
	i = (i + delta + len(rows)) % len(rows)
	m.selected, m.hasSelection = rows[i].ID, true
}
func (m *Model) log(level, operation, item string, attempt int, duration time.Duration, message string) {
	m.opts.Logs.Log(ports.LogEvent{Time: m.opts.Clock.Now(), Level: level, Source: "github", Operation: operation, Item: item, Attempt: attempt, Duration: duration, Message: message})
}
func (m Model) active() bool {
	for _, s := range m.sections {
		if s.field.Phase.Active() {
			return true
		}
	}
	for _, f := range m.fields {
		if f.Phase.Active() {
			return true
		}
	}
	return false
}
func (m *Model) maybeTick(cmds *[]tea.Cmd) {
	if m.active() && !m.tickRunning {
		m.tickRunning = true
		*cmds = append(*cmds, spinnerTick())
	}
}
func (m *Model) selectedField(force bool) tea.Cmd {
	if m.opts.Enricher == nil {
		return nil
	}
	pr, ok := m.selectedPR()
	if !ok {
		return nil
	}
	key := domain.FieldKey{Item: pr.ID, Name: "review_summary", Revision: pr.HeadSHA}
	f := m.fields[key]
	if f != nil && !force && (f.Phase == domain.Cooldown || f.Phase.Active() || (f.Phase == domain.Ready && !f.Stale(m.opts.Clock.Now()))) {
		return nil
	}
	if f == nil {
		fresh := domain.NewField[any]()
		f = &fresh
		m.fields[key] = f
	}
	g := f.Queue()
	f.Begin(g)
	m.fieldAttempts[key] = 1
	return m.fetchField(key, g, 1)
}
func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.visual.compact = msg.Width < 64
	case tickMsg:
		m.tickRunning = false
		m.frame = (m.frame + 1) % len(frames)
	case listMsg:
		s := m.section(msg.repo)
		if s == nil || s.field.Generation != msg.generation {
			break
		}
		if msg.err != nil {
			issue := fetch.Classify(msg.err)
			if issue.Kind == domain.ErrorCanceled {
				s.field.Cancel(msg.generation)
			} else {
				retryAt := fetch.RetryAt(issue, msg.attempt, msg.at)
				s.field.Fail(msg.generation, issue, retryAt)
				if !retryAt.IsZero() {
					m.log("warn", "retry-cooldown", msg.repo, msg.attempt, 0, "waiting until "+retryAt.Local().Format(time.RFC3339))
					cmds = append(cmds, tea.Tick(max(time.Until(retryAt), time.Millisecond), func(time.Time) tea.Msg { return retryMsg{repo: msg.repo, generation: msg.generation} }))
				}
			}
			m.log("error", "list-open-prs", msg.repo, msg.attempt, msg.duration, issue.Message)
			m.status = msg.repo + ": " + issue.Message
		} else {
			s.field.Succeed(msg.generation, msg.value, domain.Provenance{Source: "github", FetchedAt: msg.at, Version: msg.at.Format(time.RFC3339Nano), FreshUntil: msg.at.Add(2 * time.Minute)})
			s.attempts = 0
			m.log("info", "list-open-prs", msg.repo, msg.attempt, msg.duration, fmt.Sprintf("loaded %d open PRs", len(msg.value)))
			m.status = fmt.Sprintf("%s: %d open PRs", msg.repo, len(msg.value))
			m.ensureSelection()
			if cmd := m.selectedField(false); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case retryMsg:
		s := m.section(msg.repo)
		if s != nil && s.field.Generation == msg.generation && s.field.Phase == domain.Cooldown && !m.opts.Clock.Now().Before(s.field.RetryAt) {
			g := s.field.Queue()
			s.field.Begin(g)
			s.attempts++
			cmds = append(cmds, m.fetchList(s.repo, g, s.attempts))
			m.log("info", "retry-list", s.repo, s.attempts, 0, "retrying after cooldown")
		}
	case fieldMsg:
		f := m.fields[msg.key]
		if f == nil || f.Generation != msg.generation {
			break
		}
		if msg.err != nil {
			issue := fetch.Classify(msg.err)
			retryAt := fetch.RetryAt(issue, msg.attempt, msg.at)
			f.Fail(msg.generation, issue, retryAt)
			if !retryAt.IsZero() {
				cmds = append(cmds, tea.Tick(max(time.Until(retryAt), time.Millisecond), func(time.Time) tea.Msg { return fieldRetryMsg{key: msg.key, generation: msg.generation} }))
			}
			m.log("error", "fetch-field", msg.key.String(), msg.attempt, 0, issue.Message)
		} else {
			f.Succeed(msg.generation, msg.value, msg.provenance)
			m.fieldAttempts[msg.key] = 0
		}
	case fieldRetryMsg:
		f := m.fields[msg.key]
		if f != nil && f.Generation == msg.generation && f.Phase == domain.Cooldown && !m.opts.Clock.Now().Before(f.RetryAt) {
			g := f.Queue()
			f.Begin(g)
			m.fieldAttempts[msg.key]++
			cmds = append(cmds, m.fetchField(msg.key, g, m.fieldAttempts[msg.key]))
		}
	case commandMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("%s command failed: %v", msg.key, msg.err)
			m.log("error", "command", m.selected.String(), 1, 0, m.status)
		} else {
			m.status = msg.key + " command launched"
			m.log("info", "command", m.selected.String(), 1, 0, m.status)
		}
	case tea.KeyMsg:
		key := msg.String()
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		if m.console {
			switch key {
			case "L", "esc":
				m.console = false
			case "q":
				return m, tea.Quit
			case "j", "down":
				m.logOffset++
			case "k", "up":
				m.logOffset = max(0, m.logOffset-1)
			case "1":
				m.logLevel = "debug"
				m.logOffset = 0
			case "2":
				m.logLevel = "info"
				m.logOffset = 0
			case "3":
				m.logLevel = "warn"
				m.logOffset = 0
			case "4":
				m.logLevel = "error"
				m.logOffset = 0
			case "0":
				m.logLevel = "all"
				m.logOffset = 0
			}
			break
		}
		if key == "L" {
			m.console = true
			m.logOffset = 0
			break
		}
		if key == "?" {
			m.help = !m.help
			break
		}
		if m.help {
			if key == "esc" {
				m.help = false
			}
			break
		}
		if key == "q" {
			return m, tea.Quit
		}
		if m.mode == "details" && key == "esc" {
			m.mode = "list"
			break
		}
		switch key {
		case "j", "down":
			if m.mode == "list" {
				m.move(1)
				if cmd := m.selectedField(false); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		case "k", "up":
			if m.mode == "list" {
				m.move(-1)
				if cmd := m.selectedField(false); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		case "enter":
			if m.hasSelection {
				m.mode = "details"
			}
		case "r":
			for i := range m.sections {
				s := &m.sections[i]
				g := s.field.Queue()
				s.field.Begin(g)
				s.attempts = 1
				cmds = append(cmds, m.fetchList(s.repo, g, 1))
			}
			if cmd := m.selectedField(true); cmd != nil {
				cmds = append(cmds, cmd)
			}
		default:
			if binding, ok := m.opts.Config.Bindings[key]; ok {
				if pr, selected := m.selectedPR(); selected {
					command, err := binding.Expand(pr)
					if err != nil {
						m.status = err.Error()
						break
					}
					cmds = append(cmds, func() tea.Msg { return commandMsg{key: key, err: m.opts.Runner.Run(context.Background(), command)} })
				}
			}
		}
	}
	m.maybeTick(&cmds)
	return m, tea.Batch(cmds...)
}
func (m Model) spinner() string { return frames[m.frame%len(frames)] }
func (m Model) sectionStatus(s section) (string, visualRole) {
	switch s.field.Phase {
	case domain.Queued:
		return "◌ queued", roleQueued
	case domain.Fetching:
		return m.spinner() + " loading", rolePending
	case domain.Cooldown:
		return fmt.Sprintf("◷ cooldown until %s: %s", s.field.RetryAt.Format("15:04:05"), s.field.Error.Message), roleCooling
	case domain.Failed:
		return "× failed: " + s.field.Error.Message + " (r to retry)", roleFailure
	case domain.Canceled:
		return "○ canceled (r to retry)", roleCanceled
	case domain.Ready:
		if s.field.Stale(m.opts.Clock.Now()) {
			return "△ stale · r to refresh", roleStale
		}
		if len(s.field.Value) == 0 {
			return "— empty", roleMuted
		}
		return fmt.Sprintf("● %d open", len(s.field.Value)), roleSuccess
	default:
		return "○ not requested", roleUnrequested
	}
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
	scope := "no repository configured"
	if len(m.opts.Repos) > 0 {
		scope = strings.Join(m.opts.Repos, ", ")
	}
	lines := m.visual.header("fs-dash · Pull requests", scope, m.width)
	compact := m.visual.compact || m.width < 52
	panelOverhead := 2
	selected, hasSelected := m.selectedPR()
	contextLines := 0
	if hasSelected {
		contextLines = 1 + panelOverhead
		if m.opts.Enricher != nil {
			contextLines++
		}
	}
	statusLines := 0
	if m.status != "" {
		statusLines = 1
	}
	sectionBase := len(m.sections) * (panelOverhead + 1)
	if len(m.sections) == 0 {
		sectionBase = panelOverhead + 1
	}
	visibleRows := max(1, m.height-len(lines)-1-contextLines-statusLines-sectionBase)
	selectedSection := ""
	if hasSelected {
		selectedSection = selected.ID.Repo
	}

	if len(m.sections) == 0 {
		lines = append(lines, m.visual.panel("Repository scope", []string{"No GitHub repository found. Add repositories = [\"owner/name\"] to ~/.config/fs-dash/config.toml or run from a GitHub checkout."}, m.width)...)
	}
	for _, s := range m.sections {
		state, stateRole := m.sectionStatus(s)
		title := m.visual.paint(roleAccent, s.repo) + " · " + m.visual.badge(state, stateRole)
		content := []string{}
		if s.field.HasValue {
			if selectedSection != s.repo && len(s.field.Value) > 0 {
				content = append(content, m.visual.paint(roleMuted, "Rows hidden; j/k selects across repositories."))
			}
			if selectedSection == s.repo {
				visible := min(visibleRows, len(s.field.Value))
				index := 0
				for i, pr := range s.field.Value {
					if pr.ID == m.selected {
						index = i
						break
					}
				}
				start := max(0, index-visible/2)
				if start+visible > len(s.field.Value) {
					start = max(0, len(s.field.Value)-visible)
				}
				end := min(len(s.field.Value), start+visible)
				for _, pr := range s.field.Value[start:end] {
					marker := " "
					if m.hasSelection && pr.ID == m.selected {
						marker = ">"
					}
					indicator := ""
					indicatorRole := roleMuted
					if m.opts.Enricher != nil {
						key := domain.FieldKey{Item: pr.ID, Name: "review_summary", Revision: pr.HeadSHA}
						if f := m.fields[key]; f != nil {
							indicator = " " + f.DisplayState(m.opts.Clock.Now())
							indicatorRole = phaseRole(f.Phase, f.Stale(m.opts.Clock.Now()))
							if f.Phase.Active() {
								indicator = " " + m.spinner() + " fetching"
							}
						} else {
							indicator = " not requested"
						}
					}
					lineWidth := m.width
					if !compact {
						lineWidth -= 4
					}
					content = append(content, m.visual.selected(marker, fmt.Sprint(pr.ID.Number), pr.Title, pr.Author, pr.UpdatedAt.Local().Format("Jan 02 15:04"), strings.TrimSpace(indicator), indicatorRole, max(1, lineWidth)))
				}
			}
		}
		if len(content) == 0 {
			content = []string{m.visual.paint(roleMuted, "No pull requests to show.")}
		}
		lines = append(lines, m.visual.panel(title, content, m.width)...)
	}
	if hasSelected {
		context := []string{m.visual.paint(roleMuted, "Selected: ") + m.visual.paint(roleLink, selected.URL)}
		if m.opts.Enricher != nil {
			key := domain.FieldKey{Item: selected.ID, Name: "review_summary", Revision: selected.HeadSHA}
			if f := m.fields[key]; f != nil {
				role := phaseRole(f.Phase, f.Stale(m.opts.Clock.Now()))
				state := f.DisplayState(m.opts.Clock.Now())
				if f.Phase == domain.Cooldown {
					state += " until " + f.RetryAt.Local().Format("15:04:05")
				}
				if f.Error != nil {
					state += " · " + f.Error.Message
				}
				context = append(context, m.visual.paint(roleMuted, "Review summary: ")+m.visual.paint(role, state))
			} else {
				context = append(context, m.visual.paint(roleMuted, "Review summary: ")+m.visual.paint(rolePending, "queued"))
			}
		}
		lines = append(lines, m.visual.context("Selected PR", context, m.width)...)
	}
	if m.status != "" {
		lines = append(lines, m.visual.paint(roleWarning, m.visual.fit(m.status, m.width)))
	}
	lines = append(lines, m.visual.footer("j/k move · enter details · r refresh · o open · L logs · ? help · q quit", m.width))
	return strings.Join(lines, "\n") + "\n"
}

func (m Model) detailsView() string {
	pr, ok := m.selectedPR()
	if !ok {
		return strings.Join(m.visual.panel("PR details", []string{"PR no longer in list · esc back"}, m.width), "\n") + "\n"
	}
	content := []string{
		m.visual.paint(roleAccent, fmt.Sprintf("%s #%d", pr.ID.Repo, pr.ID.Number)),
		m.visual.paint(roleNeutral, pr.Title),
		"Author: " + m.visual.paint(roleNeutral, pr.Author),
		"Updated: " + m.visual.paint(roleNeutral, pr.UpdatedAt.Local().Format(time.RFC822)),
		"Head: " + m.visual.paint(roleMuted, pr.HeadSHA),
		"URL: " + m.visual.paint(roleLink, pr.URL),
		m.visual.paint(roleMuted, "Detailed reviews, checks, and threads arrive in change 03."),
	}
	lines := append(m.visual.header("fs-dash · PR details", pr.ID.Repo, m.width), m.visual.panel("Pull request", content, m.width)...)
	if m.status != "" {
		lines = append(lines, m.visual.paint(roleWarning, m.visual.fit(m.status, m.width)))
	}
	lines = append(lines, m.visual.footer("esc back · o open · L logs · ? help · q quit", m.width))
	return strings.Join(lines, "\n") + "\n"
}

func (m Model) helpView() string {
	keys := make([]string, 0, len(m.opts.Config.Bindings))
	for key := range m.opts.Config.Bindings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	content := []string{"j/k or arrows  move selection", "enter          PR details", "esc            back", "r              refresh all sections", "L              log console", "?              toggle help", "q              quit"}
	for _, key := range keys {
		content = append(content, fmt.Sprintf("%s              %s", key, m.opts.Config.Bindings[key].Executable))
	}
	content = append(content, "", "Log console: 0 all, 1 debug, 2 info, 3 warn, 4 error; j/k scroll; L close.")
	lines := append(m.visual.header("fs-dash · Help", "Keyboard shortcuts", m.width), m.visual.panel("Navigation", content, m.width)...)
	return strings.Join(lines, "\n") + "\n"
}

func (m Model) consoleView() string {
	events := m.opts.Logs.Snapshot()
	filtered := make([]ports.LogEvent, 0, len(events))
	for _, event := range events {
		if m.logLevel == "all" || event.Level == m.logLevel {
			filtered = append(filtered, event)
		}
	}
	panelOverhead := 2
	visible := max(1, m.height-panelOverhead-2)
	start := max(0, len(filtered)-visible-m.logOffset)
	end := min(len(filtered), start+visible)
	content := make([]string, 0, max(0, end-start))
	for _, event := range filtered[start:end] {
		label := fmt.Sprintf("%-5s", strings.ToUpper(event.Level))
		line := m.visual.paint(roleMuted, event.Time.Local().Format("15:04:05")) + " " + m.visual.paint(logRole(event.Level), label) + " " + m.visual.paint(roleNeutral, fmt.Sprintf("%s %s attempt=%d %s", event.Operation, event.Item, event.Attempt, event.Message))
		content = append(content, line)
	}
	if len(content) == 0 {
		content = []string{m.visual.paint(roleMuted, "No log events for this level.")}
	}
	lines := m.visual.panel(fmt.Sprintf("fs-dash · Logs · %s · %d events", strings.ToUpper(m.logLevel), len(filtered)), content, m.width)
	lines = append(lines, m.visual.footer("0 all · 1 debug · 2 info · 3 warn · 4 error · j/k scroll · L close", m.width))
	return strings.Join(lines, "\n") + "\n"
}
