package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fullsend-ai/fs-dash/internal/config"
	"github.com/fullsend-ai/fs-dash/internal/diag"
	"github.com/fullsend-ai/fs-dash/internal/domain"
	"github.com/fullsend-ai/fs-dash/internal/enrichmentcache"
	"github.com/fullsend-ai/fs-dash/internal/fetch"
	"github.com/fullsend-ai/fs-dash/internal/ports"
	"github.com/fullsend-ai/fs-dash/internal/preferences"
	"github.com/fullsend-ai/fs-dash/internal/reminder"
	"github.com/fullsend-ai/fs-dash/internal/triage"
)

type Options struct {
	Config              config.Config
	Repos               []string
	Lister              ports.PRLister
	Enricher            ports.Enricher
	Scheduler           *fetch.Scheduler
	ActivityScheduler   *fetch.Scheduler
	Runner              ports.CommandRunner
	Logs                *diag.Buffer
	Clock               ports.Clock
	Reminders           reminder.Store
	ReminderLoadError   error
	DisplayPreferences  preferences.Store
	PreferenceLoadError error
	EnrichmentCache     enrichmentcache.Store
	CacheLoadError      error
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
	reminderWarning                 string
	fields                          map[domain.FieldKey]*domain.Field[any]
	cacheEntries                    map[domain.FieldKey]enrichmentcache.Entry
	cacheGeneration                 uint64
	cacheCompletedGeneration        uint64
	cacheSaveGeneration             uint64
	cacheSaveRunning                bool
	fieldAttempts                   map[domain.FieldKey]int
	fieldQueue                      []domain.FieldKey
	fieldQueued                     map[domain.FieldKey]bool
	fieldQueueTokens                []uint64
	fieldQueueHead                  int
	fieldQueueGeneration            map[domain.FieldKey]uint64
	fieldPriorityQueue              []queueRef
	fieldPriorityHead               int
	fieldPriorityGeneration         map[domain.FieldKey]uint64
	fieldQueueChunkRunning          bool
	prefetchBatches                 []listPrefetchBatch
	prefetchBatchHead               int
	prefetchPRIndex                 int
	prefetchChunkRunning            bool
	fieldInFlight                   map[domain.FieldKey]uint64
	fieldRequestForKey              map[domain.FieldKey]string
	fieldRequestsInFlight           map[string]int
	fieldOffscreenWakeRunning       bool
	fieldOffscreenWakeReady         bool
	fieldOffscreenGeneration        uint64
	filter                          triage.Filter
	emojiMode                       bool
	viewer                          string
	quota                           *domain.QuotaObservation
	quotaGeneration                 uint64
	wakeTickRunning                 bool
	wakeGeneration                  uint64
	nextWake                        time.Time
	activityQueue                   []domain.FieldKey
	activityQueued                  map[domain.FieldKey]bool
	activityQueueTokens             []uint64
	activityQueueHead               int
	activityQueueGeneration         map[domain.FieldKey]uint64
	activityPriorityQueue           []queueRef
	activityPriorityHead            int
	activityPriorityGeneration      map[domain.FieldKey]uint64
	activityQueueChunkRunning       bool
	activityInFlight                map[domain.FieldKey]uint64
	activityOffscreenWakeRunning    bool
	activityOffscreenWakeReady      bool
	activityOffscreenGeneration     uint64
	visual                          visualTheme
}
type listMsg struct {
	repo       string
	generation uint64
	value      []domain.PR
	viewer     string
	quota      *domain.QuotaObservation
	err        error
	at         time.Time
	attempt    int
	duration   time.Duration
}
type fieldMsg struct {
	key        domain.FieldKey
	requestKey string
	generation uint64
	value      any
	provenance domain.Provenance
	err        error
	at         time.Time
	attempt    int
}
type fieldBatchMsg struct{ fields []fieldMsg }
type cacheSaveMsg struct {
	generation uint64
	entries    []enrichmentcache.Entry
	err        error
}
type fieldQueueChunkMsg struct{}
type activityQueueChunkMsg struct{}
type listPrefetchChunkMsg struct{}
type fieldRetryMsg struct {
	key        domain.FieldKey
	generation uint64
}
type retryMsg struct {
	repo       string
	generation uint64
}
type tickMsg time.Time
type wakeMsg struct{ generation uint64 }
type fieldOffscreenWakeMsg struct{ generation uint64 }
type activityOffscreenWakeMsg struct{ generation uint64 }
type quotaTickMsg struct{ generation uint64 }
type commandMsg struct {
	key string
	err error
}

var frames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

const quotaStaleAfter = 5 * time.Minute

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
		opts.Scheduler = fetch.NewRateLimitedScheduler(2, 150*time.Millisecond, 20*time.Second)
	}
	// Activity, detail, and list requests share one source limit and start pace.
	opts.ActivityScheduler = opts.Scheduler
	if opts.Logs == nil {
		opts.Logs = diag.New(opts.Config.LogLimit)
	}
	if opts.Clock == nil {
		opts.Clock = realClock{}
	}
	if opts.Reminders == nil {
		opts.Reminders = reminder.NewMemoryStore()
	}
	m := Model{opts: opts, mode: "list", logLevel: "all", width: 100, height: 30, fields: map[domain.FieldKey]*domain.Field[any]{}, cacheEntries: map[domain.FieldKey]enrichmentcache.Entry{}, fieldAttempts: map[domain.FieldKey]int{}, fieldQueued: map[domain.FieldKey]bool{}, fieldQueueGeneration: map[domain.FieldKey]uint64{}, fieldPriorityGeneration: map[domain.FieldKey]uint64{}, fieldInFlight: map[domain.FieldKey]uint64{}, fieldRequestForKey: map[domain.FieldKey]string{}, fieldRequestsInFlight: map[string]int{}, activityQueued: map[domain.FieldKey]bool{}, activityQueueGeneration: map[domain.FieldKey]uint64{}, activityPriorityGeneration: map[domain.FieldKey]uint64{}, activityInFlight: map[domain.FieldKey]uint64{}, filter: triage.FilterAuthored, emojiMode: true, visual: newVisualTheme(opts.Config)}
	m.visual.compact = m.width < 64
	if opts.ReminderLoadError != nil {
		m.reminderWarning = "Local triage state: " + opts.ReminderLoadError.Error()
	}
	if opts.DisplayPreferences != nil {
		if emojiMode, err := opts.DisplayPreferences.Load(); err != nil {
			m.status = "Local preferences: " + err.Error()
		} else {
			m.emojiMode = emojiMode
		}
	}
	if opts.PreferenceLoadError != nil {
		m.status = "Local preferences: " + opts.PreferenceLoadError.Error()
	}
	if opts.CacheLoadError != nil {
		m.appendLocalWarning("Local enrichment cache: " + opts.CacheLoadError.Error())
	}
	if opts.EnrichmentCache != nil {
		entries, err := opts.EnrichmentCache.Load()
		if err != nil {
			m.appendLocalWarning("Local enrichment cache: " + err.Error())
		} else {
			m.restoreCache(entries)
		}
	}
	for _, repo := range opts.Repos {
		f := domain.NewField[[]domain.PR]()
		g := f.Queue()
		f.Begin(g)
		m.sections = append(m.sections, section{repo: repo, field: f, attempts: 1})
		m.log("debug", "queue-list", repo, 1, 0, "queued initial list request")
	}
	return m
}

func (m *Model) appendLocalWarning(warning string) {
	if m.reminderWarning == "" {
		m.reminderWarning = warning
		return
	}
	m.reminderWarning += " · " + warning
}

func (m *Model) restoreCache(entries []enrichmentcache.Entry) {
	for _, entry := range enrichmentcache.Prune(entries, m.opts.Clock.Now(), enrichmentcache.DefaultMaxSize) {
		value, ok := decodeCachedValue(entry.Key.Name, entry.Value)
		if !ok {
			continue
		}
		field := domain.NewField[any]()
		generation := field.Queue()
		field.Begin(generation)
		field.Succeed(generation, value, entry.Provenance)
		m.fields[entry.Key] = &field
		m.cacheEntries[entry.Key] = entry
	}
}

func (m *Model) cacheField(key domain.FieldKey, value any, provenance domain.Provenance) {
	if m.opts.EnrichmentCache == nil || provenance.FetchedAt.IsZero() || provenance.FreshUntil.IsZero() {
		return
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		m.appendLocalWarning("Local enrichment cache: encode failed")
		return
	}
	m.cacheEntries[key] = enrichmentcache.Entry{Key: key, Value: encoded, Provenance: provenance, LastAccess: m.opts.Clock.Now()}
	m.cacheGeneration++
}

func (m *Model) touchCacheForPRs(prs []domain.PR) {
	if len(m.cacheEntries) == 0 {
		return
	}
	touched := make(map[prRevision]struct{}, len(prs))
	for _, pr := range prs {
		touched[prRevision{item: pr.ID, revision: pr.HeadSHA}] = struct{}{}
	}
	now := m.opts.Clock.Now()
	updated := false
	for key, entry := range m.cacheEntries {
		if _, ok := touched[prRevision{item: key.Item, revision: key.Revision}]; ok && !entry.LastAccess.Equal(now) {
			entry.LastAccess = now
			m.cacheEntries[key] = entry
			updated = true
		}
	}
	if updated {
		m.cacheGeneration++
	}
}

func (m *Model) scheduleCacheSave() tea.Cmd {
	if m.opts.EnrichmentCache == nil || m.cacheSaveRunning || m.cacheGeneration <= m.cacheCompletedGeneration {
		return nil
	}
	entries := make([]enrichmentcache.Entry, 0, len(m.cacheEntries))
	for _, entry := range m.cacheEntries {
		entries = append(entries, entry)
	}
	generation := m.cacheGeneration
	now := m.opts.Clock.Now()
	store := m.opts.EnrichmentCache
	m.cacheSaveGeneration = generation
	m.cacheSaveRunning = true
	return func() tea.Msg {
		entries = enrichmentcache.Prune(entries, now, enrichmentcache.DefaultMaxSize)
		err := store.Save(entries)
		return cacheSaveMsg{generation: generation, entries: entries, err: err}
	}
}

func decodeCachedValue(name string, encoded []byte) (any, bool) {
	var value any
	switch name {
	case "checks":
		value = &domain.CheckSummary{}
	case "reviews":
		value = &domain.ReviewSet{}
	case "review_requests", "assignees":
		value = &domain.LoginSet{}
	case "review_threads":
		value = &domain.ThreadSummary{}
	case "activity":
		value = &domain.ActivityEvidence{}
	case "last_touched":
		value = &domain.TimelineActivity{}
	default:
		return nil, false
	}
	if err := json.Unmarshal(encoded, value); err != nil {
		return nil, false
	}
	return dereferenceCachedValue(value), true
}

func dereferenceCachedValue(value any) any {
	switch typed := value.(type) {
	case *domain.CheckSummary:
		return *typed
	case *domain.ReviewSet:
		return *typed
	case *domain.LoginSet:
		return *typed
	case *domain.ThreadSummary:
		return *typed
	case *domain.ActivityEvidence:
		return *typed
	case *domain.TimelineActivity:
		return *typed
	default:
		return value
	}
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
		var viewer string
		var quota *domain.QuotaObservation
		if value != nil {
			result := value.(domain.PRList)
			prs, viewer, quota = result.PullRequests, result.ViewerLogin, result.Quota
		}
		at := m.opts.Clock.Now()
		return listMsg{repo: repo, generation: generation, value: prs, viewer: viewer, quota: quota, err: err, at: at, attempt: attempt, duration: at.Sub(start)}
	}
}
func (m Model) fetchField(key domain.FieldKey, generation uint64, attempt int) tea.Cmd {
	return func() tea.Msg {
		scheduler, source := m.opts.Scheduler, "github"
		if key.Name == "activity" {
			scheduler, source = m.opts.ActivityScheduler, "github-activity"
		}
		value, err := scheduler.Do(context.Background(), source, fmt.Sprintf("%s:%d", key.String(), generation), func(ctx context.Context) (any, error) {
			v, p, e := m.opts.Enricher.FetchField(ctx, key)
			return struct {
				v any
				p domain.Provenance
			}{v, p}, e
		})
		if err != nil {
			var provenance domain.Provenance
			if value != nil {
				provenance = value.(struct {
					v any
					p domain.Provenance
				}).p
			}
			return fieldMsg{key: key, requestKey: key.String(), generation: generation, provenance: provenance, err: err, at: m.opts.Clock.Now(), attempt: attempt}
		}
		result := value.(struct {
			v any
			p domain.Provenance
		})
		return fieldMsg{key: key, requestKey: key.String(), generation: generation, value: result.v, provenance: result.p, at: m.opts.Clock.Now(), attempt: attempt}
	}
}

func (m Model) fetchFields(keys []domain.FieldKey, generations []uint64, attempts []int, requestKey string) tea.Cmd {
	return func() tea.Msg {
		batch, ok := m.opts.Enricher.(ports.BatchEnricher)
		if !ok {
			return fieldBatchMsg{}
		}
		value, err := m.opts.Scheduler.Do(context.Background(), "github", requestKey, func(ctx context.Context) (any, error) {
			return batch.FetchFields(ctx, keys), nil
		})
		results := map[domain.FieldKey]ports.FieldResult{}
		if value != nil {
			results = value.(map[domain.FieldKey]ports.FieldResult)
		}
		at := m.opts.Clock.Now()
		fields := make([]fieldMsg, 0, len(keys))
		for i, key := range keys {
			result, found := results[key]
			if !found {
				result.Err = err
				if result.Err == nil {
					result.Err = fmt.Errorf("combined GitHub response omitted %s", key.Name)
				}
			}
			fields = append(fields, fieldMsg{key: key, requestKey: requestKey, generation: generations[i], value: result.Value, provenance: result.Provenance, err: result.Err, at: at, attempt: attempts[i]})
		}
		return fieldBatchMsg{fields: fields}
	}
}
func (m *Model) observeQuota(observation *domain.QuotaObservation) bool {
	if observation == nil || observation.Limit <= 0 || observation.Used < 0 || observation.Used > observation.Limit || observation.Remaining < 0 || observation.Remaining > observation.Limit || observation.ResetAt.IsZero() || observation.ObservedAt.IsZero() {
		return false
	}
	if m.quota != nil {
		sameResource := strings.EqualFold(observation.Resource, m.quota.Resource)
		if sameResource {
			if observation.ResetAt.Before(m.quota.ResetAt) {
				return false
			}
			if observation.ResetAt.Equal(m.quota.ResetAt) {
				if observation.Used < m.quota.Used || observation.Remaining > m.quota.Remaining {
					return false
				}
				if observation.Used == m.quota.Used && observation.Remaining == m.quota.Remaining && observation.ObservedAt.Before(m.quota.ObservedAt) {
					return false
				}
			}
		} else if observation.ObservedAt.Before(m.quota.ObservedAt) {
			return false
		}
	}
	copy := *observation
	m.quota = &copy
	m.quotaGeneration++
	return true
}

func (m Model) quotaRefreshCmd() tea.Cmd {
	if m.quota == nil {
		return nil
	}
	now := m.opts.Clock.Now()
	age := now.Sub(m.quota.ObservedAt)
	if age < 0 {
		age = 0
	}
	untilReset := m.quota.ResetAt.Sub(now)
	if untilReset <= 0 {
		return nil
	}
	delay := time.Minute
	if untilReset <= time.Minute {
		delay = time.Second
	}
	if ageDelay := time.Minute - age%time.Minute; ageDelay < delay {
		delay = ageDelay
	}
	if untilReset < delay {
		delay = untilReset
	}
	generation := m.quotaGeneration
	return tea.Tick(max(delay, time.Millisecond), func(time.Time) tea.Msg { return quotaTickMsg{generation: generation} })
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
	return m.projectedRows()
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
	if !m.hasSelection {
		return domain.PR{}, false
	}
	return selectedPRFromSections(m.projection(), m.selected)
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
	for _, section := range m.sections {
		for _, pr := range section.field.Value {
			for _, name := range []string{"checks", "reviews", "review_requests", "assignees", "review_threads"} {
				if f := m.fieldFor(pr, name); f != nil && f.Phase.Active() {
					return true
				}
			}
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
func (m *Model) applySnooze(duration time.Duration) {
	if !m.hasSelection {
		m.mode = "list"
		return
	}
	until := m.opts.Clock.Now().Add(duration)
	if err := m.opts.Reminders.Snooze(m.selected, until); err != nil {
		m.status = "Snooze failed: " + err.Error()
	} else {
		m.status = "PR snoozed until " + until.Local().Format("Jan 02 15:04")
	}
	m.mode = "list"
	m.ensureSelection()
}
func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.visual.compact = msg.Width < 64
		cmds = append(cmds, m.startFieldFetches()...)
		cmds = append(cmds, m.startActivityFetches()...)
	case tickMsg:
		m.tickRunning = false
		m.frame = (m.frame + 1) % len(frames)
	case wakeMsg:
		if msg.generation == m.wakeGeneration {
			m.wakeTickRunning = false
			m.nextWake = time.Time{}
			m.ensureSelection()
		}
	case fieldOffscreenWakeMsg:
		if msg.generation == m.fieldOffscreenGeneration {
			m.fieldOffscreenWakeRunning = false
			m.fieldOffscreenWakeReady = true
			cmds = append(cmds, m.startFieldFetches()...)
		}
	case activityOffscreenWakeMsg:
		if msg.generation == m.activityOffscreenGeneration {
			m.activityOffscreenWakeRunning = false
			m.activityOffscreenWakeReady = true
			cmds = append(cmds, m.startActivityFetches()...)
		}
	case fieldQueueChunkMsg:
		m.fieldQueueChunkRunning = false
		cmds = append(cmds, m.startFieldFetches()...)
	case activityQueueChunkMsg:
		m.activityQueueChunkRunning = false
		cmds = append(cmds, m.startActivityFetches()...)
	case listPrefetchChunkMsg:
		m.prefetchChunkRunning = false
		cmds = append(cmds, m.processListPrefetchChunk())
		cmds = append(cmds, m.startFieldFetches()...)
		cmds = append(cmds, m.startActivityFetches()...)
	case quotaTickMsg:
		if msg.generation == m.quotaGeneration {
			cmds = append(cmds, m.quotaRefreshCmd())
		}
	case listMsg:
		if m.observeQuota(msg.quota) {
			cmds = append(cmds, m.quotaRefreshCmd())
		}
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
			if msg.viewer != "" {
				m.viewer = msg.viewer
			}
			m.ensureSelection()
			m.touchCacheForPRs(msg.value)
			cmds = append(cmds, m.enqueueListPrefetch(msg.repo, msg.generation, msg.value))
			cmds = append(cmds, m.selectedFields(false)...)
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
		cmds = append(cmds, m.applyFieldMessage(msg, true)...)
	case fieldBatchMsg:
		for _, field := range msg.fields {
			cmds = append(cmds, m.applyFieldMessage(field, false)...)
		}
		cmds = append(cmds, m.startFieldFetches()...)
		cmds = append(cmds, m.startActivityFetches()...)
	case cacheSaveMsg:
		if m.cacheSaveRunning && msg.generation == m.cacheSaveGeneration {
			m.cacheSaveRunning = false
			m.cacheCompletedGeneration = msg.generation
			if msg.err != nil {
				m.appendLocalWarning("Local enrichment cache: save failed")
			} else if msg.generation == m.cacheGeneration {
				m.cacheEntries = make(map[domain.FieldKey]enrichmentcache.Entry, len(msg.entries))
				for _, entry := range msg.entries {
					m.cacheEntries[entry.Key] = entry
				}
			}
		}
	case fieldRetryMsg:
		f := m.fields[msg.key]
		if f != nil && f.Generation == msg.generation && f.Phase == domain.Cooldown && !m.opts.Clock.Now().Before(f.RetryAt) {
			if msg.key.Name == "activity" {
				m.queueActivity(msg.key, true, false)
				cmds = append(cmds, m.startActivityFetches()...)
			} else {
				m.queueField(msg.key, true)
				cmds = append(cmds, m.startFieldFetches()...)
			}
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
		if m.mode == "snooze" {
			switch key {
			case "esc":
				m.mode = "list"
			case "1":
				m.applySnooze(time.Hour)
			case "2":
				m.applySnooze(24 * time.Hour)
			case "3":
				m.applySnooze(7 * 24 * time.Hour)
			}
			break
		}
		if m.mode == "details" && key == "esc" {
			m.mode = "list"
			break
		}
		switch key {
		case "j", "down":
			if m.mode == "list" {
				m.move(1)
				cmds = append(cmds, m.selectedFields(false)...)
			}
		case "k", "up":
			if m.mode == "list" {
				m.move(-1)
				cmds = append(cmds, m.selectedFields(false)...)
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
			cmds = append(cmds, m.selectedFields(true)...)
		case "h":
			if m.mode == "list" {
				m.filter = m.filter.Prev()
				m.ensureSelection()
				cmds = append(cmds, m.selectedFields(false)...)
			}
		case "l":
			if m.mode == "list" {
				m.filter = m.filter.Next()
				m.ensureSelection()
				cmds = append(cmds, m.selectedFields(false)...)
			}
		case "s":
			if m.hasSelection {
				if err := m.opts.Reminders.ToggleStar(m.selected); err != nil {
					m.status = "Star update failed: " + err.Error()
				} else {
					m.status = "Star updated locally"
					m.ensureSelection()
				}
			}
		case "e":
			if m.mode == "list" {
				m.emojiMode = !m.emojiMode
				if m.opts.DisplayPreferences != nil {
					if err := m.opts.DisplayPreferences.SaveEmojiMode(m.emojiMode); err != nil {
						m.status = "Display preference save failed: " + err.Error()
					} else if strings.HasPrefix(m.status, "Local preferences:") || strings.HasPrefix(m.status, "Display preference save failed:") {
						m.status = ""
					}
				} else if m.opts.PreferenceLoadError != nil {
					m.status = "Display preference not saved: " + m.opts.PreferenceLoadError.Error()
				}
			}
		case "z":
			if m.hasSelection {
				m.mode = "snooze"
			} else {
				m.status = "Select a PR before snoozing"
			}
		case "u":
			if m.hasSelection {
				if err := m.opts.Reminders.Unsnooze(m.selected); err != nil {
					m.status = "Unsnooze failed: " + err.Error()
				} else {
					m.status = "PR unsnoozed locally"
					m.ensureSelection()
				}
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
	if cacheSave := m.scheduleCacheSave(); cacheSave != nil {
		cmds = append(cmds, cacheSave)
	}
	if wake := m.scheduleWake(); wake != nil {
		cmds = append(cmds, wake)
	}
	m.maybeTick(&cmds)
	return m, tea.Batch(cmds...)
}

func (m *Model) applyFieldMessage(msg fieldMsg, startNext bool) []tea.Cmd {
	var cmds []tea.Cmd
	if m.observeQuota(msg.provenance.Quota) {
		cmds = append(cmds, m.quotaRefreshCmd())
	}
	fieldSlotFreed := false
	if msg.key.Name != "activity" && m.fieldInFlight[msg.key] == msg.generation {
		delete(m.fieldInFlight, msg.key)
		requestKey := msg.requestKey
		if requestKey == "" {
			requestKey = m.fieldRequestForKey[msg.key]
		}
		delete(m.fieldRequestForKey, msg.key)
		if requestKey == "" {
			fieldSlotFreed = true
		} else if remaining, ok := m.fieldRequestsInFlight[requestKey]; ok {
			remaining--
			if remaining <= 0 {
				delete(m.fieldRequestsInFlight, requestKey)
				fieldSlotFreed = true
			} else {
				m.fieldRequestsInFlight[requestKey] = remaining
			}
		}
	}
	if msg.key.Name == "activity" && m.activityInFlight[msg.key] == msg.generation {
		delete(m.activityInFlight, msg.key)
		cmds = append(cmds, m.startActivityFetches()...)
		cmds = append(cmds, m.startFieldFetches()...)
	}
	f := m.fields[msg.key]
	if f == nil || f.Generation != msg.generation {
		if startNext && fieldSlotFreed {
			cmds = append(cmds, m.startFieldFetches()...)
		}
		return cmds
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
		m.cacheField(msg.key, msg.value, msg.provenance)
		m.fieldAttempts[msg.key] = 0
		switch value := msg.value.(type) {
		case domain.ReviewSet:
			if value.ViewerLogin != "" {
				m.viewer = value.ViewerLogin
			}
		case domain.LoginSet:
			if value.ViewerLogin != "" {
				m.viewer = value.ViewerLogin
			}
		}
		m.ensureSelection()
	}
	if startNext && fieldSlotFreed {
		cmds = append(cmds, m.startFieldFetches()...)
		cmds = append(cmds, m.startActivityFetches()...)
	}
	return cmds
}

func (m Model) spinner() string { return frames[m.frame%len(frames)] }
func (m Model) sectionStatus(s section, projected triage.Sections) (string, visualRole) {
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
		summary := m.filterCountSummary(s, projected)
		if s.field.Stale(m.opts.Clock.Now()) {
			return "△ stale · " + summary + " · r to refresh", roleStale
		}
		if len(s.field.Value) == 0 {
			return "— " + summary, roleMuted
		}
		return "● " + summary, roleSuccess
	default:
		return "○ not requested", roleUnrequested
	}
}

func (m Model) filterCountSummary(s section, projected triage.Sections) string {
	filtered, pending := 0, 0
	for _, rows := range [][]triage.Row{projected.Review, projected.Active} {
		for _, row := range rows {
			if row.PR.ID.Repo != s.repo {
				continue
			}
			filtered++
			if row.FilterPending {
				pending++
			}
		}
	}
	summary := fmt.Sprintf("%d open · %d in filter", len(s.field.Value), filtered)
	if pending > 0 {
		summary += fmt.Sprintf(" (%d pending)", pending)
	}
	return summary
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
		m.visual.paint(roleMuted, "Review and check signals are summarized in the list; detailed review threads arrive in change 03."),
	}
	lines := append(m.visual.header("fs-dash · PR details", pr.ID.Repo, m.width), m.visual.panel("Pull request", content, m.width)...)
	lines = append(lines, m.quotaStatusLine())
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
	content := []string{"j/k or arrows  move selection", "h/l            previous/next filter", "e              toggle emoji/text table labels", "enter          PR details", "esc            back", "s              toggle local star", "z              choose snooze duration", "u              unsnooze selected PR", "r              refresh all sections", "L              log console", "?              toggle help", "q              quit"}
	for _, key := range keys {
		binding := m.opts.Config.Bindings[key]
		summary := binding.Summary
		if summary == "" {
			summary = binding.Executable
		}
		content = append(content, fmt.Sprintf("%s              %s", key, summary))
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
