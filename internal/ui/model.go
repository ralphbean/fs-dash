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
	"github.com/mattn/go-runewidth"
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
	if opts.Scheduler == nil {
		opts.Scheduler = fetch.NewScheduler(2, 20*time.Second)
	}
	if opts.Logs == nil {
		opts.Logs = diag.New(opts.Config.LogLimit)
	}
	if opts.Clock == nil {
		opts.Clock = realClock{}
	}
	m := Model{opts: opts, mode: "list", logLevel: "all", width: 100, height: 30, fields: map[domain.FieldKey]*domain.Field[any]{}, fieldAttempts: map[domain.FieldKey]int{}}
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
func (m Model) sectionStatus(s section) string {
	switch s.field.Phase {
	case domain.Queued, domain.Fetching:
		return m.spinner() + " loading"
	case domain.Cooldown:
		return fmt.Sprintf("! cooldown until %s: %s", s.field.RetryAt.Format("15:04:05"), s.field.Error.Message)
	case domain.Failed:
		return "! " + s.field.Error.Message + " (r to retry)"
	case domain.Canceled:
		return "canceled (r to retry)"
	case domain.Ready:
		if s.field.Stale(m.opts.Clock.Now()) {
			return "stale · r to refresh"
		}
		if len(s.field.Value) == 0 {
			return "empty"
		}
		return fmt.Sprintf("%d open", len(s.field.Value))
	default:
		return "not requested"
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
	var out strings.Builder
	scope := "no repository configured"
	if len(m.opts.Repos) > 0 {
		scope = strings.Join(m.opts.Repos, ", ")
	}
	fmt.Fprintf(&out, "fs-dash · Pull requests · %s\n\n", scope)
	if len(m.sections) == 0 {
		out.WriteString("No GitHub repository found. Add repositories = [\"owner/name\"] to ~/.config/fs-dash/config.toml or run from a GitHub checkout.\n")
	}
	for _, s := range m.sections {
		fmt.Fprintf(&out, "%s  %s\n", s.repo, m.sectionStatus(s))
		if s.field.HasValue {
			selectedRepo := ""
			if m.hasSelection {
				selectedRepo = m.selected.Repo
			}
			if selectedRepo != s.repo && len(s.field.Value) > 0 {
				out.WriteString("  Rows hidden; j/k selects across repositories.\n")
			}
			if selectedRepo == s.repo {
				reserved := 9 + len(m.sections)
				if m.opts.Enricher != nil {
					reserved++
				}
				visible := max(1, m.height-reserved)
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
				if start > 0 {
					fmt.Fprintf(&out, "  ↑ %d earlier PRs\n", start)
				}
				for _, pr := range s.field.Value[start:end] {
					marker := " "
					if m.hasSelection && pr.ID == m.selected {
						marker = ">"
					}
					titleWidth := max(12, m.width-48)
					title := runewidth.Truncate(pr.Title, titleWidth, "…")
					indicator := ""
					if m.opts.Enricher != nil {
						key := domain.FieldKey{Item: pr.ID, Name: "review_summary", Revision: pr.HeadSHA}
						if f := m.fields[key]; f != nil {
							indicator = " " + f.DisplayState(m.opts.Clock.Now())
							if f.Phase.Active() {
								indicator = " " + m.spinner()
							}
						} else {
							indicator = " ·"
						}
					}
					line := fmt.Sprintf("%s #%d %-*s @%s %s%s", marker, pr.ID.Number, titleWidth, title, pr.Author, pr.UpdatedAt.Local().Format("Jan 02 15:04"), indicator)
					out.WriteString(runewidth.Truncate(line, max(20, m.width), "…"))
					out.WriteByte('\n')
				}
				if end < len(s.field.Value) {
					fmt.Fprintf(&out, "  ↓ %d more PRs\n", len(s.field.Value)-end)
				}
			}
		}
		out.WriteByte('\n')
	}
	if pr, ok := m.selectedPR(); ok {
		fmt.Fprintf(&out, "Selected: %s\n", pr.URL)
		if m.opts.Enricher != nil {
			key := domain.FieldKey{Item: pr.ID, Name: "review_summary", Revision: pr.HeadSHA}
			if f := m.fields[key]; f != nil {
				fmt.Fprintf(&out, "Review summary: %s", f.DisplayState(m.opts.Clock.Now()))
				if f.Phase == domain.Cooldown {
					fmt.Fprintf(&out, " until %s", f.RetryAt.Local().Format("15:04:05"))
				}
				if f.Error != nil {
					fmt.Fprintf(&out, " · %s", f.Error.Message)
				}
				out.WriteByte('\n')
			}
		}
	}
	if m.status != "" {
		fmt.Fprintf(&out, "%s\n", m.status)
	}
	out.WriteString("j/k move · enter details · r refresh · o open · L logs · ? help · q quit\n")
	return out.String()
}
func (m Model) detailsView() string {
	pr, ok := m.selectedPR()
	if !ok {
		return "PR no longer in list · esc back\n"
	}
	var out strings.Builder
	fmt.Fprintf(&out, "fs-dash · PR details\n\n%s #%d\n%s\n\nAuthor: %s\nUpdated: %s\nHead: %s\nURL: %s\n\n", pr.ID.Repo, pr.ID.Number, pr.Title, pr.Author, pr.UpdatedAt.Local().Format(time.RFC822), pr.HeadSHA, pr.URL)
	out.WriteString("Detailed reviews, checks, and threads arrive in change 03.\n")
	if m.status != "" {
		fmt.Fprintf(&out, "\n%s\n", m.status)
	}
	out.WriteString("\nesc back · o open · L logs · ? help · q quit\n")
	return out.String()
}
func (m Model) helpView() string {
	keys := make([]string, 0, len(m.opts.Config.Bindings))
	for key := range m.opts.Config.Bindings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var out strings.Builder
	out.WriteString("fs-dash help\n\n j/k or arrows  move selection\n enter          PR details\n esc            back\n r              refresh all sections\n L              log console\n ?              toggle help\n q              quit\n")
	for _, key := range keys {
		fmt.Fprintf(&out, " %s              %s\n", key, m.opts.Config.Bindings[key].Executable)
	}
	out.WriteString("\nLog console: 0 all, 1 debug, 2 info, 3 warn, 4 error; j/k scroll; L close.\n")
	return out.String()
}
func (m Model) consoleView() string {
	events := m.opts.Logs.Snapshot()
	filtered := make([]ports.LogEvent, 0, len(events))
	for _, event := range events {
		if m.logLevel == "all" || event.Level == m.logLevel {
			filtered = append(filtered, event)
		}
	}
	visible := max(1, m.height-4)
	start := max(0, len(filtered)-visible-m.logOffset)
	end := min(len(filtered), start+visible)
	var out strings.Builder
	fmt.Fprintf(&out, "fs-dash · logs · %s · %d events\n\n", m.logLevel, len(filtered))
	for _, event := range filtered[start:end] {
		line := fmt.Sprintf("%s %-5s %s %s attempt=%d %s", event.Time.Local().Format("15:04:05"), event.Level, event.Operation, event.Item, event.Attempt, event.Message)
		out.WriteString(runewidth.Truncate(line, max(20, m.width), "…"))
		out.WriteByte('\n')
	}
	out.WriteString("\n0 all · 1 debug · 2 info · 3 warn · 4 error · j/k scroll · L close\n")
	return out.String()
}
