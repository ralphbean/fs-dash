package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fullsend-ai/fs-dash/internal/domain"
	"github.com/fullsend-ai/fs-dash/internal/fetch"
)

var detailGroups = []string{"description", "commits", "checks", "reviews", "conversation", "threads"}

type detailFetchMsg struct {
	key        domain.DetailKey
	generation uint64
	value      any
	provenance domain.Provenance
	err        error
	at         time.Time
	attempt    int
}

type detailRetryMsg struct {
	key        domain.DetailKey
	generation uint64
}

type actionInspectMsg struct {
	item   domain.ItemID
	action string
	state  domain.PRActionState
	err    error
}

type actionMutationMsg struct {
	item   domain.ItemID
	action string
	head   string
	state  domain.PRActionState
	err    error
}

type commentMutationMsg struct {
	item      domain.ItemID
	threadID  string
	body      string
	startedAt time.Time
	commentID string
	err       error
}

type threadResolveMsg struct {
	item     domain.ItemID
	threadID string
	err      error
}

type threadResolutionKey struct {
	item     domain.ItemID
	threadID string
}

type pendingWrite struct {
	key       domain.DetailKey
	threadID  string
	commentID string
	body      string
	startedAt time.Time
	unknown   bool
}

func detailKey(pr domain.PR, group string) domain.DetailKey {
	return domain.DetailKey{Item: pr.ID, Group: group, Revision: pr.HeadSHA}
}

func detailThreadKey(pr domain.PR, threadID string) domain.DetailKey {
	return domain.DetailKey{Item: pr.ID, Group: "thread_context", Revision: pr.HeadSHA, ThreadID: threadID}
}

func (m *Model) openDetails() []tea.Cmd {
	pr, ok := m.selectedPR()
	if !ok {
		m.status = "Select an open PR before opening details"
		return nil
	}
	m.mode = "details"
	m.detailsScroll = 0
	m.status = ""
	return m.loadDetailGroups(pr, false)
}

func (m *Model) loadDetailGroups(pr domain.PR, force bool) []tea.Cmd {
	var cmds []tea.Cmd
	for _, group := range detailGroups {
		cmds = append(cmds, m.loadDetail(pr, detailKey(pr, group), force))
	}
	return cmds
}

func (m *Model) loadDetail(pr domain.PR, key domain.DetailKey, force bool) tea.Cmd {
	field := m.detailFields[key]
	if field == nil {
		created := domain.NewField[any]()
		field = &created
		m.detailFields[key] = field
	}
	if field.Phase.Active() && !force {
		return nil
	}
	if !force && field.HasValue && !field.Stale(m.opts.Clock.Now()) {
		return nil
	}
	generation := field.Queue()
	field.Begin(generation)
	attempt := m.detailAttempts[key] + 1
	m.detailAttempts[key] = attempt
	if m.opts.Details == nil {
		field.Fail(generation, &domain.FetchError{Kind: domain.ErrorPermanent, Message: "PR details are unavailable"}, time.Time{})
		return nil
	}
	reader := m.opts.Details
	scheduler := m.opts.Scheduler
	if scheduler == nil {
		scheduler = fetch.NewRateLimitedScheduler(2, 150*time.Millisecond, 20*time.Second)
	}
	return func() tea.Msg {
		value, err := scheduler.Do(context.Background(), "github-details", fmt.Sprintf("%s:%d", key.String(), generation), func(ctx context.Context) (any, error) {
			v, p, requestErr := reader.FetchDetail(ctx, key)
			return struct {
				value any
				prov  domain.Provenance
			}{v, p}, requestErr
		})
		at := m.opts.Clock.Now()
		if err != nil {
			var provenance domain.Provenance
			if value != nil {
				provenance = value.(struct {
					value any
					prov  domain.Provenance
				}).prov
			}
			return detailFetchMsg{key: key, generation: generation, provenance: provenance, err: err, at: at, attempt: attempt}
		}
		result := value.(struct {
			value any
			prov  domain.Provenance
		})
		return detailFetchMsg{key: key, generation: generation, value: result.value, provenance: result.prov, at: at, attempt: attempt}
	}
}

func (m *Model) applyDetailMessage(msg detailFetchMsg) []tea.Cmd {
	field := m.detailFields[msg.key]
	if field == nil || field.Generation != msg.generation {
		return nil
	}
	pr, exists := m.currentPR(msg.key.Item)
	if !exists || pr.HeadSHA != msg.key.Revision {
		field.Cancel(msg.generation)
		if m.mode == "details" || m.mode == "thread" {
			if selected, ok := m.selectedPR(); ok && selected.ID == msg.key.Item {
				return m.loadDetailGroups(selected, false)
			}
		}
		return nil
	}
	selectedThreadID, selectedThreadIndex := m.currentThreadSelection(msg.key.Item)
	if msg.err != nil {
		issue := fetch.Classify(msg.err)
		retryAt := fetch.RetryAt(issue, msg.attempt, msg.at)
		field.Fail(msg.generation, issue, retryAt)
		m.status = fmt.Sprintf("%s detail failed: %s", msg.key.Group, issue.Message)
		m.log("error", "fetch-pr-detail", msg.key.String(), msg.attempt, 0, issue.Message)
		if m.writePending != nil && m.writePending.key == msg.key {
			m.status = "Could not verify the posted text; draft retained. Refresh before retrying."
			m.writePending = nil
		}
		if msg.key.Group == "threads" {
			m.failPendingThreadResolutions(msg.key.Item, msg.generation)
			m.restoreThreadSelection(msg.key.Item, selectedThreadID, selectedThreadIndex)
		}
		if !retryAt.IsZero() {
			return []tea.Cmd{tea.Tick(max(time.Until(retryAt), time.Millisecond), func(time.Time) tea.Msg {
				return detailRetryMsg{key: msg.key, generation: msg.generation}
			})}
		}
		return nil
	}
	if msg.key.Group == "threads" {
		threads, ok := msg.value.(domain.ReviewThreadList)
		if !ok || !threads.Complete {
			issue := &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub returned an incomplete review thread list"}
			field.Fail(msg.generation, issue, time.Time{})
			m.status = issue.Message
			m.failPendingThreadResolutions(msg.key.Item, msg.generation)
			m.restoreThreadSelection(msg.key.Item, selectedThreadID, selectedThreadIndex)
			return nil
		}
	}
	field.Succeed(msg.generation, msg.value, msg.provenance)
	m.detailAttempts[msg.key] = 0
	if m.writePending != nil && m.writePending.key == msg.key {
		if m.pendingWriteVisible(*m.writePending, msg.value) {
			m.completePendingWrite()
		} else {
			m.status = "Post not verified in refreshed GitHub data; draft retained. Review before retrying."
			m.writePending = nil
		}
	}
	if msg.key.Group == "threads" {
		m.completePendingThreadResolutions(msg.key.Item, msg.generation, msg.value.(domain.ReviewThreadList))
		m.restoreThreadSelection(msg.key.Item, selectedThreadID, selectedThreadIndex)
		if m.mode == "thread" {
			return m.loadSelectedThreadContext()
		}
	}
	return nil
}

func (m *Model) currentThreadSelection(item domain.ItemID) (string, int) {
	selected, ok := m.selectedPR()
	if !ok || selected.ID != item || m.mode != "thread" {
		return "", m.threadIndex
	}
	threads := m.threadList(selected)
	if m.threadIndex < 0 || m.threadIndex >= len(threads) {
		return "", m.threadIndex
	}
	return threads[m.threadIndex].ID, m.threadIndex
}

func (m *Model) restoreThreadSelection(item domain.ItemID, threadID string, previousIndex int) {
	selected, ok := m.selectedPR()
	if !ok || selected.ID != item || m.mode != "thread" {
		return
	}
	threads := m.threadList(selected)
	if len(threads) == 0 {
		m.threadIndex = 0
		return
	}
	for index, thread := range threads {
		if threadID != "" && thread.ID == threadID {
			m.threadIndex = index
			return
		}
	}
	m.threadIndex = min(max(previousIndex, 0), len(threads)-1)
}

func (m *Model) setThreadResolutionStatus(item domain.ItemID, status string) {
	selected, ok := m.selectedPR()
	if ok && selected.ID == item && m.mode == "thread" {
		m.status = status
	}
}

func (m *Model) failPendingThreadResolutions(item domain.ItemID, generation uint64) int {
	failed := 0
	for key, verifyGeneration := range m.pendingThreadResolutions {
		if key.item == item && verifyGeneration > 0 && verifyGeneration <= generation {
			delete(m.pendingThreadResolutions, key)
			failed++
		}
	}
	if failed > 0 {
		m.setThreadResolutionStatus(item, fmt.Sprintf("Could not verify %d thread resolution(s); the thread(s) are available again", failed))
	}
	return failed
}

func (m *Model) completePendingThreadResolutions(item domain.ItemID, generation uint64, value domain.ReviewThreadList) {
	verified, unresolved := 0, 0
	for key, verifyGeneration := range m.pendingThreadResolutions {
		if key.item != item || verifyGeneration == 0 || verifyGeneration > generation {
			continue
		}
		found := false
		for _, thread := range value.Threads {
			if thread.ID == key.threadID {
				found = true
				break
			}
		}
		delete(m.pendingThreadResolutions, key)
		if found {
			unresolved++
		} else {
			verified++
		}
	}
	if verified > 0 && unresolved > 0 {
		m.setThreadResolutionStatus(item, fmt.Sprintf("%d thread resolution(s) verified; %d thread(s) remain unresolved", verified, unresolved))
	} else if verified > 0 {
		m.setThreadResolutionStatus(item, fmt.Sprintf("%d thread resolution(s) verified", verified))
	} else if unresolved > 0 {
		m.setThreadResolutionStatus(item, fmt.Sprintf("%d thread(s) remain unresolved and are available again", unresolved))
	}
}

func (m Model) currentPR(item domain.ItemID) (domain.PR, bool) {
	for _, section := range m.sections {
		if !section.field.HasValue {
			continue
		}
		for _, pr := range section.field.Value {
			if pr.ID == item {
				return pr, true
			}
		}
	}
	return domain.PR{}, false
}

func (m *Model) retryDetail(msg detailRetryMsg) []tea.Cmd {
	field := m.detailFields[msg.key]
	if field == nil || field.Generation != msg.generation || field.Phase != domain.Cooldown || m.opts.Clock.Now().Before(field.RetryAt) {
		return nil
	}
	pr, ok := m.currentPR(msg.key.Item)
	if !ok || pr.HeadSHA != msg.key.Revision {
		return nil
	}
	return []tea.Cmd{m.loadDetail(pr, msg.key, true)}
}

func (m *Model) handleDetailsKey(key string) []tea.Cmd {
	pr, ok := m.selectedPR()
	if !ok {
		if key == "esc" {
			m.mode = "list"
		}
		return nil
	}
	switch key {
	case "esc":
		m.mode = "list"
		m.status = ""
	case "j", "down":
		m.detailsScroll++
	case "k", "up":
		m.detailsScroll = max(0, m.detailsScroll-1)
	case "r":
		m.status = "Refreshing PR details"
		return m.loadDetailGroups(pr, true)
	case "T":
		m.mode = "thread"
		m.threadIndex, m.threadScroll = 0, 0
		return m.openThreadWalk(pr)
	case "c":
		m.beginEditor("", "details")
	case "m", "x":
		action := "merge"
		if key == "x" {
			action = "close"
		}
		return m.inspectAction(pr, action)
	}
	return nil
}

func (m *Model) openThreadWalk(pr domain.PR) []tea.Cmd {
	field := m.detailFields[detailKey(pr, "threads")]
	if field == nil || !field.HasValue || field.Stale(m.opts.Clock.Now()) {
		return []tea.Cmd{m.loadDetail(pr, detailKey(pr, "threads"), false)}
	}
	return m.loadSelectedThreadContext()
}

func (m Model) threadList(pr domain.PR) []domain.ReviewThread {
	field := m.detailFields[detailKey(pr, "threads")]
	if field == nil || !field.HasValue {
		return nil
	}
	value, ok := field.Value.(domain.ReviewThreadList)
	if !ok {
		return nil
	}
	threads := append([]domain.ReviewThread(nil), value.Threads...)
	visible := threads[:0]
	for _, thread := range threads {
		if _, pending := m.pendingThreadResolutions[threadResolutionKey{item: pr.ID, threadID: thread.ID}]; pending {
			continue
		}
		visible = append(visible, thread)
	}
	threads = visible
	sort.SliceStable(threads, func(i, j int) bool {
		if threads[i].Path != threads[j].Path {
			return threads[i].Path < threads[j].Path
		}
		line := func(thread domain.ReviewThread) int {
			if thread.Line > 0 {
				return thread.Line
			}
			return thread.OriginalLine
		}
		if line(threads[i]) != line(threads[j]) {
			return line(threads[i]) < line(threads[j])
		}
		return threads[i].ID < threads[j].ID
	})
	return threads
}

func (m Model) pendingThreadResolutionCount(item domain.ItemID) int {
	count := 0
	for key := range m.pendingThreadResolutions {
		if key.item == item {
			count++
		}
	}
	return count
}

func (m *Model) loadSelectedThreadContext() []tea.Cmd {
	pr, ok := m.selectedPR()
	if !ok {
		return nil
	}
	threads := m.threadList(pr)
	if len(threads) == 0 {
		return nil
	}
	if m.threadIndex >= len(threads) {
		m.threadIndex = len(threads) - 1
	}
	if m.threadIndex < 0 {
		m.threadIndex = 0
	}
	return []tea.Cmd{m.loadDetail(pr, detailThreadKey(pr, threads[m.threadIndex].ID), false)}
}

func (m *Model) handleThreadKey(key string) []tea.Cmd {
	pr, ok := m.selectedPR()
	if !ok {
		m.mode = "details"
		return nil
	}
	threads := m.threadList(pr)
	switch key {
	case "esc", "T":
		m.mode = "details"
		m.detailsScroll = 0
	case "n", "right":
		if len(threads) > 0 {
			m.threadIndex = (m.threadIndex + 1) % len(threads)
			m.threadScroll = 0
			return m.loadSelectedThreadContext()
		}
	case "p", "left":
		if len(threads) > 0 {
			m.threadIndex = (m.threadIndex - 1 + len(threads)) % len(threads)
			m.threadScroll = 0
			return m.loadSelectedThreadContext()
		}
	case "j", "down":
		m.threadScroll++
	case "k", "up":
		m.threadScroll = max(0, m.threadScroll-1)
	case "R":
		return []tea.Cmd{m.loadDetail(pr, detailKey(pr, "threads"), true)}
	case "c":
		if len(threads) > 0 {
			m.beginEditor(threads[m.threadIndex].ID, "thread")
		}
	case "r":
		threadField := m.detailFields[detailKey(pr, "threads")]
		if threadField == nil || !threadField.HasValue || threadField.Phase == domain.Failed || threadField.Stale(m.opts.Clock.Now()) {
			m.status = "Refreshing thread state before resolving"
			return []tea.Cmd{m.loadDetail(pr, detailKey(pr, "threads"), true)}
		}
		if len(threads) > 0 {
			return m.resolveReviewThread(pr, threads[m.threadIndex].ID)
		}
	}
	return nil
}

func (m *Model) resolveReviewThread(pr domain.PR, threadID string) []tea.Cmd {
	if m.opts.Actions == nil {
		m.status = "GitHub thread actions are unavailable"
		return nil
	}
	if m.actionRunning || m.actionChecking || m.writePending != nil {
		m.status = "Wait for the current GitHub action to finish"
		return nil
	}
	resolutionKey := threadResolutionKey{item: pr.ID, threadID: threadID}
	if _, alreadyPending := m.pendingThreadResolutions[resolutionKey]; alreadyPending {
		m.status = "Thread resolution is already running in the background"
		return nil
	}
	client, scheduler, item := m.opts.Actions, m.opts.Scheduler, pr.ID
	if scheduler == nil {
		scheduler = fetch.NewRateLimitedScheduler(2, 150*time.Millisecond, 20*time.Second)
	}
	if m.pendingThreadResolutions == nil {
		m.pendingThreadResolutions = make(map[threadResolutionKey]uint64)
	}
	previousIndex := max(0, m.threadIndex)
	m.pendingThreadResolutions[resolutionKey] = 0
	threads := m.threadList(pr)
	if len(threads) > 0 {
		m.threadIndex = previousIndex % len(threads)
	} else {
		m.threadIndex = 0
	}
	m.threadScroll = 0
	m.status = "Resolving thread in the background; advanced to the next thread"
	cmds := []tea.Cmd{func() tea.Msg {
		_, err := scheduler.Do(context.Background(), "github-mutation", fmt.Sprintf("resolve-thread:%s:%s:%d", item.String(), threadID, time.Now().UnixNano()), func(ctx context.Context) (any, error) {
			return nil, client.ResolveReviewThread(ctx, item, threadID)
		})
		return threadResolveMsg{item: item, threadID: threadID, err: err}
	}}
	return append(cmds, m.loadSelectedThreadContext()...)
}

func (m *Model) applyThreadResolve(msg threadResolveMsg) []tea.Cmd {
	resolutionKey := threadResolutionKey{item: msg.item, threadID: msg.threadID}
	if _, pending := m.pendingThreadResolutions[resolutionKey]; !pending {
		return nil
	}
	pr, ok := m.currentPR(msg.item)
	if !ok {
		delete(m.pendingThreadResolutions, resolutionKey)
		return nil
	}
	key := detailKey(pr, "threads")
	cmd := m.loadDetail(pr, key, true)
	field := m.detailFields[key]
	if cmd == nil || field == nil || !field.Phase.Active() {
		delete(m.pendingThreadResolutions, resolutionKey)
		m.setThreadResolutionStatus(msg.item, "Could not refresh GitHub thread state; the thread is available again")
		return nil
	}
	m.pendingThreadResolutions[resolutionKey] = field.Generation
	if msg.err != nil {
		m.setThreadResolutionStatus(msg.item, "Resolve response was uncertain; checking GitHub thread state in the background")
	} else {
		m.setThreadResolutionStatus(msg.item, "Checking GitHub thread state in the background")
	}
	return []tea.Cmd{cmd}
}

func (m *Model) beginEditor(threadID, returnMode string) {
	if m.opts.Actions == nil {
		m.status = "GitHub write actions are unavailable"
		return
	}
	m.editorReturnMode, m.editorThreadID = returnMode, threadID
	if threadID == "" {
		m.editorBuffer = []rune(m.prCommentDraft)
	} else {
		m.editorBuffer = []rune(m.replyDrafts[threadID])
	}
	m.editorCursor = len(m.editorBuffer)
	m.mode = "editor"
	m.saveEditorDraft()
}

func (m *Model) handleEditorKey(msg tea.KeyMsg) []tea.Cmd {
	key := msg.String()
	if m.editorCursor < 0 {
		m.editorCursor = 0
	}
	if m.editorCursor > len(m.editorBuffer) {
		m.editorCursor = len(m.editorBuffer)
	}
	switch key {
	case "esc", "ctrl+c":
		m.saveEditorDraft()
		m.mode = m.editorReturnMode
	case "ctrl+d", "ctrl+enter":
		return m.submitEditor()
	case "backspace", "ctrl+h":
		if m.editorCursor > 0 {
			m.editorBuffer = append(m.editorBuffer[:m.editorCursor-1], m.editorBuffer[m.editorCursor:]...)
			m.editorCursor--
		}
		m.saveEditorDraft()
	case "delete":
		if m.editorCursor < len(m.editorBuffer) {
			m.editorBuffer = append(m.editorBuffer[:m.editorCursor], m.editorBuffer[m.editorCursor+1:]...)
		}
		m.saveEditorDraft()
	case "left":
		m.editorCursor = max(0, m.editorCursor-1)
		m.saveEditorDraft()
	case "right":
		m.editorCursor = min(len(m.editorBuffer), m.editorCursor+1)
		m.saveEditorDraft()
	case "ctrl+left":
		for m.editorCursor > 0 && unicode.IsSpace(m.editorBuffer[m.editorCursor-1]) {
			m.editorCursor--
		}
		for m.editorCursor > 0 && !unicode.IsSpace(m.editorBuffer[m.editorCursor-1]) {
			m.editorCursor--
		}
		m.saveEditorDraft()
	case "ctrl+right":
		for m.editorCursor < len(m.editorBuffer) && !unicode.IsSpace(m.editorBuffer[m.editorCursor]) {
			m.editorCursor++
		}
		for m.editorCursor < len(m.editorBuffer) && unicode.IsSpace(m.editorBuffer[m.editorCursor]) {
			m.editorCursor++
		}
		m.saveEditorDraft()
	case "ctrl+a":
		for m.editorCursor > 0 && m.editorBuffer[m.editorCursor-1] != '\n' {
			m.editorCursor--
		}
		m.saveEditorDraft()
	case "home":
		for m.editorCursor > 0 && m.editorBuffer[m.editorCursor-1] != '\n' {
			m.editorCursor--
		}
		m.saveEditorDraft()
	case "end", "ctrl+e":
		for m.editorCursor < len(m.editorBuffer) && m.editorBuffer[m.editorCursor] != '\n' {
			m.editorCursor++
		}
		m.saveEditorDraft()
	case "ctrl+k":
		lineEnd := m.editorCursor
		for lineEnd < len(m.editorBuffer) && m.editorBuffer[lineEnd] != '\n' {
			lineEnd++
		}
		if lineEnd == m.editorCursor && lineEnd < len(m.editorBuffer) {
			lineEnd++
		}
		m.editorBuffer = append(m.editorBuffer[:m.editorCursor], m.editorBuffer[lineEnd:]...)
		m.saveEditorDraft()
	case "ctrl+w":
		start := m.editorCursor
		for start > 0 && unicode.IsSpace(m.editorBuffer[start-1]) {
			start--
		}
		for start > 0 && !unicode.IsSpace(m.editorBuffer[start-1]) {
			start--
		}
		m.editorBuffer = append(m.editorBuffer[:start], m.editorBuffer[m.editorCursor:]...)
		m.editorCursor = start
		m.saveEditorDraft()
	case "enter":
		m.editorBuffer = insertEditorRunes(m.editorBuffer, m.editorCursor, []rune{'\n'})
		m.editorCursor++
		m.saveEditorDraft()
	case "up":
		m.editorScroll = max(0, m.editorScroll-1)
	case "down":
		m.editorScroll++
	default:
		if len(msg.Runes) > 0 {
			m.editorBuffer = insertEditorRunes(m.editorBuffer, m.editorCursor, msg.Runes)
			m.editorCursor += len(msg.Runes)
			m.saveEditorDraft()
		}
	}
	return nil
}

func insertEditorRunes(buffer []rune, cursor int, inserted []rune) []rune {
	cursor = max(0, min(cursor, len(buffer)))
	result := make([]rune, 0, len(buffer)+len(inserted))
	result = append(result, buffer[:cursor]...)
	result = append(result, inserted...)
	result = append(result, buffer[cursor:]...)
	return result
}

func (m *Model) saveEditorDraft() {
	draft := string(m.editorBuffer)
	if m.editorThreadID == "" {
		m.prCommentDraft = draft
	} else {
		m.replyDrafts[m.editorThreadID] = draft
	}
	_, cursorLine := editorDisplayLines(m.editorBuffer, m.editorCursor, max(20, m.width-6))
	pageSize := max(1, m.height-9)
	if m.editorThreadID != "" {
		pageSize = max(1, min(5, m.height/5))
	}
	if cursorLine < m.editorScroll {
		m.editorScroll = cursorLine
	} else if cursorLine >= m.editorScroll+pageSize {
		m.editorScroll = cursorLine - pageSize + 1
	}
}

func editorDisplayLines(buffer []rune, cursor, width int) ([]string, int) {
	width = max(1, width)
	cursor = max(0, min(cursor, len(buffer)))
	lines := make([]string, 0, 1)
	var current strings.Builder
	currentWidth := 0
	cursorLine := 0
	flush := func() {
		lines = append(lines, current.String())
		current.Reset()
		currentWidth = 0
	}
	appendRune := func(value rune) {
		valueWidth := lipgloss.Width(string(value))
		if current.Len() > 0 && currentWidth+valueWidth > width {
			flush()
		}
		current.WriteRune(value)
		currentWidth += valueWidth
	}
	for index := 0; index <= len(buffer); index++ {
		if index == cursor {
			caretWidth := lipgloss.Width("▏")
			if current.Len() > 0 && currentWidth+caretWidth > width {
				flush()
			}
			cursorLine = len(lines)
			current.WriteString("▏")
			currentWidth += caretWidth
		}
		if index == len(buffer) {
			break
		}
		if buffer[index] == '\n' {
			flush()
			continue
		}
		appendRune(buffer[index])
	}
	lines = append(lines, current.String())
	return lines, cursorLine
}

func (m *Model) submitEditor() []tea.Cmd {
	pr, ok := m.selectedPR()
	if !ok {
		m.status = "PR is no longer open in the current list"
		return nil
	}
	body := string(m.editorBuffer)
	if strings.TrimSpace(body) == "" {
		m.status = "Comment cannot be empty"
		return nil
	}
	if m.actionRunning || m.writePending != nil {
		m.status = "Waiting to verify the previous comment before retrying"
		return nil
	}
	if m.opts.Actions == nil {
		m.status = "GitHub write actions are unavailable"
		return nil
	}
	item, threadID, client, scheduler := pr.ID, m.editorThreadID, m.opts.Actions, m.opts.Scheduler
	if scheduler == nil {
		scheduler = fetch.NewRateLimitedScheduler(2, 150*time.Millisecond, 20*time.Second)
	}
	startedAt := m.opts.Clock.Now()
	m.actionRunning = true
	m.status = "Posting reply"
	if threadID == "" {
		m.status = "Posting PR comment"
	}
	return []tea.Cmd{func() tea.Msg {
		value, err := scheduler.Do(context.Background(), "github-mutation", fmt.Sprintf("comment:%s:%d", item.String(), startedAt.UnixNano()), func(ctx context.Context) (any, error) {
			if threadID == "" {
				return client.PostPRComment(ctx, item, body)
			}
			return client.ReplyToReviewThread(ctx, item, threadID, body)
		})
		var commentID string
		if value != nil {
			commentID = value.(string)
		}
		return commentMutationMsg{item: item, threadID: threadID, body: body, startedAt: startedAt, commentID: commentID, err: err}
	}}
}

func (m *Model) applyCommentMutation(msg commentMutationMsg) []tea.Cmd {
	m.actionRunning = false
	pr, ok := m.selectedPR()
	if !ok || pr.ID != msg.item {
		m.status = "PR changed before the comment result was verified; draft retained"
		return nil
	}
	group := "conversation"
	if msg.threadID != "" {
		group = "threads"
	}
	key := detailKey(pr, group)
	m.writePending = &pendingWrite{key: key, threadID: msg.threadID, commentID: msg.commentID, body: msg.body, startedAt: msg.startedAt, unknown: msg.err != nil}
	if msg.err != nil {
		m.status = "Comment outcome uncertain; refreshing GitHub before allowing a retry"
	} else {
		m.status = "Comment sent; verifying it appears in GitHub"
	}
	return []tea.Cmd{m.loadDetail(pr, key, true)}
}

func (m Model) pendingWriteVisible(pending pendingWrite, value any) bool {
	matches := func(id, body, author string, created time.Time) bool {
		if pending.commentID != "" && id == pending.commentID {
			return true
		}
		return pending.unknown && body == pending.body && strings.EqualFold(author, m.viewer) && !created.Before(pending.startedAt.Add(-time.Second))
	}
	if pending.threadID == "" {
		conversation, ok := value.(domain.PRConversation)
		if !ok {
			return false
		}
		for _, comment := range conversation.Comments {
			if matches(comment.ID, comment.Body, comment.Author, comment.CreatedAt) {
				return true
			}
		}
		return false
	}
	threads, ok := value.(domain.ReviewThreadList)
	if !ok {
		return false
	}
	for _, thread := range threads.Threads {
		if thread.ID != pending.threadID {
			continue
		}
		for _, comment := range thread.Comments {
			if matches(comment.ID, comment.Body, comment.Author, comment.CreatedAt) {
				return true
			}
		}
	}
	return false
}

func (m *Model) completePendingWrite() {
	if m.writePending == nil {
		return
	}
	threadID := m.writePending.threadID
	if threadID == "" {
		m.prCommentDraft = ""
	} else {
		delete(m.replyDrafts, threadID)
	}
	m.editorBuffer = nil
	m.editorCursor = 0
	m.mode = m.editorReturnMode
	if threadID == "" {
		m.status = "PR comment posted and verified"
	} else {
		m.status = "Thread reply posted and verified"
	}
	m.writePending = nil
}

func (m *Model) inspectAction(pr domain.PR, action string) []tea.Cmd {
	if m.opts.Actions == nil {
		m.status = "GitHub PR actions are unavailable"
		return nil
	}
	m.actionChecking, m.actionRunning = true, false
	m.actionState = nil
	m.status = "Checking current PR state and merge policy"
	client, scheduler, item := m.opts.Actions, m.opts.Scheduler, pr.ID
	if scheduler == nil {
		scheduler = fetch.NewRateLimitedScheduler(2, 150*time.Millisecond, 20*time.Second)
	}
	return []tea.Cmd{func() tea.Msg {
		value, err := scheduler.Do(context.Background(), "github-action-read", "inspect:"+item.String(), func(ctx context.Context) (any, error) { return client.InspectPR(ctx, item) })
		var state domain.PRActionState
		if value != nil {
			state = value.(domain.PRActionState)
		}
		return actionInspectMsg{item: item, action: action, state: state, err: err}
	}}
}

func (m *Model) applyActionInspect(msg actionInspectMsg) {
	m.actionChecking = false
	pr, ok := m.selectedPR()
	if !ok || pr.ID != msg.item {
		m.status = "PR selection changed while checking current state"
		return
	}
	if msg.err != nil {
		m.status = "Could not check current PR state: " + fetch.Classify(msg.err).Message
		return
	}
	if msg.state.HeadSHA != pr.HeadSHA {
		m.status = "PR head changed; refresh details before taking an action"
		return
	}
	if !strings.EqualFold(msg.state.State, "OPEN") {
		m.status = "PR is no longer open"
		return
	}
	if msg.action == "merge" && (!msg.state.MergeQueueKnown || msg.state.IsDraft) {
		if !msg.state.MergeQueueKnown {
			m.status = "Could not determine merge-queue policy; merge is disabled"
		} else {
			m.status = "Draft PRs cannot be merged"
		}
		return
	}
	if msg.action == "merge" && !msg.state.MergeQueueRequired && !msg.state.SquashMergeAllowed && !msg.state.MergeCommitAllowed && !msg.state.RebaseMergeAllowed {
		m.status = "Repository has no enabled merge method"
		return
	}
	if msg.action == "merge" && msg.state.Mergeable == "CONFLICTING" {
		m.status = "PR has merge conflicts and cannot be merged"
		return
	}
	m.actionState = &msg.state
	m.confirmAction = msg.action
	m.mode = "confirm"
}

func (m *Model) handleConfirmKey(key string) []tea.Cmd {
	switch key {
	case "esc", "n":
		m.mode = "details"
		m.confirmAction = ""
		m.actionState = nil
		m.status = "Action canceled"
	case "y", "enter":
		return m.runConfirmedAction()
	}
	return nil
}

func (m *Model) runConfirmedAction() []tea.Cmd {
	pr, ok := m.selectedPR()
	if !ok || m.actionState == nil {
		m.mode = "details"
		m.status = "Current PR state is unavailable"
		return nil
	}
	if m.actionRunning || m.actionChecking {
		return nil
	}
	action, state, item, expectedHead := m.confirmAction, *m.actionState, pr.ID, pr.HeadSHA
	client, scheduler := m.opts.Actions, m.opts.Scheduler
	if client == nil {
		m.status = "GitHub PR actions are unavailable"
		m.mode = "details"
		return nil
	}
	if scheduler == nil {
		scheduler = fetch.NewRateLimitedScheduler(2, 150*time.Millisecond, 20*time.Second)
	}
	m.actionRunning = true
	m.status = "Submitting " + action + " action"
	return []tea.Cmd{func() tea.Msg {
		value, err := scheduler.Do(context.Background(), "github-mutation", fmt.Sprintf("%s:%s:%d", action, item.String(), time.Now().UnixNano()), func(ctx context.Context) (any, error) {
			var mutationErr error
			if action == "merge" {
				mutationErr = client.MergePR(ctx, item, expectedHead, state)
			} else {
				mutationErr = client.ClosePR(ctx, item, expectedHead)
			}
			verified, verifyErr := client.InspectPR(ctx, item)
			if verifyErr != nil {
				if mutationErr != nil {
					return verified, mutationErr
				}
				return verified, verifyErr
			}
			if action == "close" {
				if !strings.EqualFold(verified.State, "CLOSED") || verified.HeadSHA != expectedHead {
					if mutationErr != nil {
						return verified, mutationErr
					}
					return verified, fmt.Errorf("close state could not be verified")
				}
			} else if !strings.EqualFold(verified.State, "MERGED") && !verified.QueueEntryPresent && !verified.AutoMergeEnabled {
				if mutationErr != nil {
					return verified, mutationErr
				}
				return verified, fmt.Errorf("merge or queue state could not be verified")
			}
			return verified, nil
		})
		var verified domain.PRActionState
		if value != nil {
			verified = value.(domain.PRActionState)
		}
		return actionMutationMsg{item: item, action: action, head: expectedHead, state: verified, err: err}
	}}
}

func (m *Model) applyActionMutation(msg actionMutationMsg) []tea.Cmd {
	m.actionRunning = false
	m.mode = "details"
	m.confirmAction = ""
	m.actionState = nil
	if msg.err != nil {
		m.status = "Action outcome could not be verified: " + fetch.Classify(msg.err).Message
		return nil
	}
	if msg.action == "close" {
		m.status = "PR closed and verified"
		m.mode = "list"
		return m.refreshLists()
	}
	if strings.EqualFold(msg.state.State, "MERGED") {
		m.status = "PR merged and verified"
	} else if msg.state.QueueEntryPresent {
		m.status = "PR added to merge queue · " + msg.state.QueueEntryState
	} else {
		m.status = "Auto-merge enabled and verified"
	}
	return nil
}

func (m *Model) refreshLists() []tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.sections {
		s := &m.sections[i]
		generation := s.field.Queue()
		s.field.Begin(generation)
		s.attempts = 1
		cmds = append(cmds, m.fetchList(s.repo, generation, s.attempts))
	}
	return cmds
}

func (m Model) detailStateLine(key domain.DetailKey) string {
	field := m.detailFields[key]
	if field == nil {
		return m.visual.paint(roleUnrequested, "pending")
	}
	if field.Phase.Active() {
		if field.HasValue {
			return m.visual.paint(rolePending, m.spinner()+" refreshing · retained value shown")
		}
		return m.visual.paint(rolePending, m.spinner()+" loading")
	}
	if field.Phase == domain.Failed || field.Phase == domain.Cooldown {
		if field.HasValue {
			return m.visual.paint(roleFailure, "refresh failed · retained value shown: "+field.Error.Message)
		}
		return m.visual.paint(roleFailure, "unavailable: "+field.Error.Message)
	}
	if field.Phase == domain.Canceled || !field.HasValue {
		return m.visual.paint(roleUnrequested, "not loaded")
	}
	if field.Stale(m.opts.Clock.Now()) {
		return m.visual.paint(roleStale, "stale · r refreshes")
	}
	return m.visual.paint(roleSuccess, "ready")
}

func appendBodyLines(lines []string, body string, width int) []string {
	if strings.TrimSpace(body) == "" {
		return append(lines, "(empty)")
	}
	for _, line := range strings.Split(body, "\n") {
		if line == "" {
			lines = append(lines, "")
			continue
		}
		if width < 1 {
			width = 1
		}
		var current strings.Builder
		for _, word := range strings.Fields(line) {
			if current.Len() == 0 {
				current.WriteString(word)
				continue
			}
			candidate := current.String() + " " + word
			if lipgloss.Width(candidate) <= width {
				current.WriteString(" " + word)
				continue
			}
			lines = append(lines, current.String())
			current.Reset()
			current.WriteString(word)
		}
		if current.Len() > 0 {
			lines = append(lines, current.String())
		}
	}
	return lines
}

func (m Model) detailLines(pr domain.PR) []string {
	var lines []string
	appendGroup := func(group, label string, render func(any) []string) {
		key := detailKey(pr, group)
		field := m.detailFields[key]
		lines = append(lines, m.visual.paint(roleAccent, label)+" · "+m.detailStateLine(key))
		if field == nil || !field.HasValue {
			return
		}
		if rendered := render(field.Value); len(rendered) > 0 {
			lines = append(lines, rendered...)
		}
		lines = append(lines, "")
	}
	appendGroup("description", "Description", func(value any) []string {
		data, ok := value.(domain.PRDescription)
		if !ok {
			return []string{"(unavailable)"}
		}
		content := []string{"State: " + data.State + " · " + data.HeadRefName + " → " + data.BaseRefName, "Draft: " + fmt.Sprint(data.IsDraft), "Review decision: " + fallback(data.ReviewDecision, "none"), "Mergeability: " + data.Mergeable, "Body:"}
		return appendBodyLines(content, data.Body, max(20, m.width-6))
	})
	appendGroup("commits", "Commits", func(value any) []string {
		data, ok := value.(domain.PRCommitList)
		if !ok {
			return []string{"(unavailable)"}
		}
		if len(data.Commits) == 0 {
			return []string{"(no commits returned)"}
		}
		out := make([]string, 0, len(data.Commits)+1)
		for _, commit := range data.Commits {
			oid := commit.OID
			if len(oid) > 8 {
				oid = oid[:8]
			}
			out = append(out, fmt.Sprintf("%s · %s · %s · %s", oid, commit.Author, commit.AuthoredAt.Local().Format("Jan 02 15:04"), commit.Message))
		}
		if !data.Complete {
			out = append(out, "Older commits not loaded")
		}
		return out
	})
	appendGroup("checks", "Checks", func(value any) []string {
		data, ok := value.(domain.PRCheckList)
		if !ok {
			return []string{"(unavailable)"}
		}
		out := make([]string, 0, len(data.Checks)+1)
		for _, check := range data.Checks {
			label := check.Status
			if check.Conclusion != "" {
				label += " / " + check.Conclusion
			}
			out = append(out, label+" · "+check.Name)
		}
		if len(out) == 0 {
			out = append(out, "(no checks reported)")
		}
		if !data.Complete {
			out = append(out, "More checks not loaded")
		}
		return out
	})
	appendGroup("reviews", "Reviews", func(value any) []string {
		data, ok := value.(domain.ReviewSet)
		if !ok {
			return []string{"(unavailable)"}
		}
		out := make([]string, 0, len(data.Reviews)+1)
		for _, review := range data.Reviews {
			out = append(out, fmt.Sprintf("%s · %s · %s", review.State, review.Author, review.At.Local().Format("Jan 02 15:04")))
			if strings.TrimSpace(review.Body) != "" {
				out = appendBodyLines(out, review.Body, max(20, m.width-6))
			}
		}
		if len(out) == 0 {
			out = append(out, "(no submitted reviews)")
		}
		if !data.Complete {
			out = append(out, "Older reviews not loaded")
		}
		return out
	})
	appendGroup("conversation", "Conversation", func(value any) []string {
		data, ok := value.(domain.PRConversation)
		if !ok {
			return []string{"(unavailable)"}
		}
		out := make([]string, 0, len(data.Comments)+1)
		for _, comment := range data.Comments {
			out = append(out, fmt.Sprintf("%s · %s", comment.Author, comment.CreatedAt.Local().Format("Jan 02 15:04")))
			out = appendBodyLines(out, comment.Body, max(20, m.width-6))
		}
		if len(out) == 0 {
			out = append(out, "(no PR comments)")
		}
		if !data.Complete {
			out = append(out, "Older comments not loaded")
		}
		return out
	})
	appendGroup("threads", "Unresolved review threads", func(value any) []string {
		data, ok := value.(domain.ReviewThreadList)
		if !ok {
			return []string{"(unavailable)"}
		}
		out := []string{fmt.Sprintf("%d unresolved · press T to walk threads", len(data.Threads))}
		if !data.Complete {
			out = append(out, "More threads not loaded")
		}
		return out
	})
	return lines
}

func fallback(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func (m Model) detailsView() string {
	pr, ok := m.selectedPR()
	if !ok {
		return strings.Join(m.visual.panel("PR details", []string{"PR no longer in list · esc back"}, m.width), "\n") + "\n"
	}
	head := []string{m.visual.paint(roleAccent, fmt.Sprintf("%s #%d", pr.ID.Repo, pr.ID.Number)), m.visual.paint(roleNeutral, pr.Title), "Author: " + pr.Author, "Head: " + pr.HeadSHA, "URL: " + m.visual.paint(roleLink, pr.URL), ""}
	content := append(head, m.detailLines(pr)...)
	pageSize := max(1, m.height-8)
	maxScroll := max(0, len(content)-pageSize)
	start := min(m.detailsScroll, maxScroll)
	end := min(len(content), start+pageSize)
	visible := make([]string, 0, pageSize+2)
	if start > 0 {
		visible = append(visible, m.visual.paint(roleMuted, "↑ more above · k scrolls"))
	}
	for _, line := range content[start:end] {
		visible = append(visible, m.visual.fit(line, max(1, m.width-4)))
	}
	if end < len(content) {
		visible = append(visible, m.visual.paint(roleMuted, "↓ more below · j scrolls"))
	}
	lines := append(m.visual.header("fs-dash · PR details", pr.ID.Repo, m.width), m.visual.panel("Pull request", visible, m.width)...)
	if m.status != "" {
		lines = append(lines, m.visual.paint(roleWarning, m.visual.fit(m.status, m.width)))
	}
	lines = append(lines, m.visual.footer("esc back · j/k scroll · T threads · c comment · m merge/queue · x close · r refresh · L logs · ? help", m.width))
	return strings.Join(lines, "\n") + "\n"
}

func diffHunkLines(theme visualTheme, hunk string) []string {
	if hunk == "" {
		return []string{theme.paint(roleMuted, "(diff context unavailable)")}
	}
	lines := make([]string, 0, strings.Count(hunk, "\n")+1)
	for _, line := range strings.Split(hunk, "\n") {
		role := roleNeutral
		switch {
		case strings.HasPrefix(line, "@@"):
			role = roleAccent
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") || strings.HasPrefix(line, "\\"):
			role = roleMuted
		case strings.HasPrefix(line, "+"):
			role = roleSuccess
		case strings.HasPrefix(line, "-"):
			role = roleFailure
		}
		lines = append(lines, theme.paint(role, line))
	}
	return lines
}

func (m Model) commentWindow(comment domain.ReviewThreadComment, width int) []string {
	width = max(8, width)
	innerWidth := max(1, width-4)
	title := fmt.Sprintf("%s · %s", comment.Author, comment.CreatedAt.Local().Format("Jan 02 15:04"))
	title = m.visual.fit(title, max(0, width-6))
	label := m.visual.paint(roleBorder, "─ ") + m.visual.paint(roleAccent, title) + m.visual.paint(roleBorder, " ")
	top := m.visual.paint(roleBorder, "┌") + label + m.visual.paint(roleBorder, strings.Repeat("─", max(0, width-2-lipgloss.Width(label)))) + m.visual.paint(roleBorder, "┐")
	lines := []string{top}
	for _, bodyLine := range appendBodyLines(nil, comment.Body, innerWidth) {
		bodyLine = m.visual.fit(bodyLine, innerWidth)
		padding := max(0, innerWidth-lipgloss.Width(bodyLine))
		lines = append(lines, m.visual.paint(roleBorder, "│ ")+bodyLine+strings.Repeat(" ", padding)+m.visual.paint(roleBorder, " │"))
	}
	lines = append(lines, m.visual.paint(roleBorder, "└"+strings.Repeat("─", max(0, width-2))+"┘"))
	return lines
}

func (m Model) threadView() string {
	pr, ok := m.selectedPR()
	if !ok {
		return strings.Join(m.visual.panel("Review thread", []string{"PR no longer in list · esc back"}, m.width), "\n") + "\n"
	}
	threadField := m.detailFields[detailKey(pr, "threads")]
	if threadField == nil || !threadField.HasValue {
		lines := []string{"Loading unresolved review threads", m.detailStateLine(detailKey(pr, "threads")), "esc back · R refresh"}
		return strings.Join(m.visual.panel("Review threads", lines, m.width), "\n") + "\n"
	}
	threads := m.threadList(pr)
	if len(threads) == 0 {
		lines := []string{"No unresolved review threads", "esc back · T return to details · R refresh"}
		if pending := m.pendingThreadResolutionCount(pr.ID); pending > 0 {
			lines = []string{fmt.Sprintf("Resolving %d thread(s) in the background", pending), "No other unresolved review threads", "esc back · T return to details · R refresh"}
		}
		if m.status != "" {
			lines = append(lines, m.visual.paint(roleWarning, m.visual.fit(m.status, m.width)))
		}
		return strings.Join(m.visual.panel("Review threads", lines, m.width), "\n") + "\n"
	}
	if m.threadIndex >= len(threads) {
		m.threadIndex = len(threads) - 1
	}
	if m.mode == "editor" && m.editorThreadID != "" {
		for index, thread := range threads {
			if thread.ID == m.editorThreadID {
				m.threadIndex = index
				break
			}
		}
	}
	thread := threads[m.threadIndex]
	lines := []string{fmt.Sprintf("Thread %d of %d · %s", m.threadIndex+1, len(threads), thread.ID), "File: " + thread.Path}
	line := thread.Line
	if line == 0 {
		line = thread.OriginalLine
	}
	location := fmt.Sprintf("Line %d", line)
	if thread.IsOutdated {
		location = m.visual.paint(roleWarning, fmt.Sprintf("⚠ Outdated · original line %d", thread.OriginalLine))
	}
	lines = append(lines, location)
	contextKey := detailThreadKey(pr, thread.ID)
	contextField := m.detailFields[contextKey]
	if contextField == nil || !contextField.HasValue {
		lines = append(lines, "Diff context · "+m.detailStateLine(contextKey))
	} else if context, ok := contextField.Value.(domain.ReviewThreadContext); ok {
		lines = append(lines, "Diff context:")
		lines = append(lines, diffHunkLines(m.visual, context.DiffHunk)...)
		if context.IsOutdated {
			lines = append(lines, "Original location is outdated; this hunk is historical context.")
		}
	}
	lines = append(lines, "", "Thread history:")
	for _, comment := range thread.Comments {
		lines = append(lines, m.commentWindow(comment, max(8, m.width-4))...)
		lines = append(lines, "")
	}
	if !thread.CommentsComplete {
		lines = append(lines, "Older thread comments not loaded")
	}
	editing := m.mode == "editor" && m.editorThreadID == thread.ID
	pageSize := max(1, m.height-8)
	if editing {
		pageSize = max(1, m.height-15)
	}
	maxScroll := max(0, len(lines)-pageSize)
	start := min(m.threadScroll, maxScroll)
	end := min(len(lines), start+pageSize)
	visible := make([]string, 0, pageSize+2)
	if start > 0 {
		visible = append(visible, "↑ more above")
	}
	for _, entry := range lines[start:end] {
		visible = append(visible, m.visual.fit(entry, max(1, m.width-4)))
	}
	if end < len(lines) {
		visible = append(visible, "↓ more below")
	}
	view := append(m.visual.header("fs-dash · Review threads", pr.ID.Repo, m.width), m.visual.panel("Unresolved review thread", visible, m.width)...)
	if editing {
		draftLines, _ := editorDisplayLines(m.editorBuffer, m.editorCursor, max(20, m.width-6))
		composerPageSize := max(1, min(5, m.height/5))
		composerStart := min(m.editorScroll, max(0, len(draftLines)-composerPageSize))
		composerEnd := min(len(draftLines), composerStart+composerPageSize)
		composerLines := []string{"ctrl+d post · ←/→ cursor · ↑/↓ scroll · enter newline"}
		if composerStart > 0 {
			composerLines = append(composerLines, "↑ earlier draft lines")
		}
		composerLines = append(composerLines, draftLines[composerStart:composerEnd]...)
		if composerEnd < len(draftLines) {
			composerLines = append(composerLines, "↓ more draft lines")
		}
		view = append(view, m.visual.panel("Comment draft", composerLines, m.width)...)
	}
	if m.status != "" {
		view = append(view, m.visual.paint(roleWarning, m.visual.fit(m.status, m.width)))
	}
	footer := "esc back · n/p next/previous thread · j/k scroll · c comment · r resolve · R refresh · T details"
	if editing {
		footer = "ctrl+d post · ctrl+k kill line · ctrl+w delete word · ctrl+e end line · esc cancel"
	}
	view = append(view, m.visual.footer(footer, m.width))
	return strings.Join(view, "\n") + "\n"
}

func (m Model) editorView() string {
	target := "PR comment"
	if m.editorThreadID != "" {
		target = "Review-thread reply"
	}
	content, _ := editorDisplayLines(m.editorBuffer, m.editorCursor, max(20, m.width-6))
	pageSize := max(1, m.height-9)
	maxScroll := max(0, len(content)-pageSize)
	start := min(m.editorScroll, maxScroll)
	end := min(len(content), start+pageSize)
	lines := []string{m.visual.paint(roleAccent, target), "Type a multiline draft. ←/→ moves cursor; ↑/↓ scrolls; ctrl+d posts; ctrl+k kills line; ctrl+w deletes word; ctrl+e ends line; esc cancels.", ""}
	if start > 0 {
		lines = append(lines, "↑ earlier draft lines")
	}
	lines = append(lines, content[start:end]...)
	if end < len(content) {
		lines = append(lines, "↓ more draft lines")
	}
	if m.status != "" {
		lines = append(lines, m.visual.paint(roleWarning, m.status))
	}
	return strings.Join(m.visual.panel("Compose "+target, lines, m.width), "\n") + "\n"
}

func (m Model) confirmView() string {
	pr, ok := m.selectedPR()
	if !ok {
		return strings.Join(m.visual.panel("Confirm action", []string{"PR no longer available", "esc cancel"}, m.width), "\n") + "\n"
	}
	state := ""
	if m.actionState != nil {
		switch m.confirmAction {
		case "merge":
			if m.actionState.MergeQueueRequired {
				state = "GitHub rules require the merge queue. The PR will be enqueued when eligible."
			} else {
				method := ""
				if m.actionState.SquashMergeAllowed {
					method = "squash"
				} else if m.actionState.MergeCommitAllowed {
					method = "merge commit"
				} else if m.actionState.RebaseMergeAllowed {
					method = "rebase"
				}
				state = "Request auto-merge using " + method + " after requirements pass."
			}
		case "close":
			state = "Close this pull request."
		}
	}
	lines := []string{fmt.Sprintf("Target: %s #%d · %s", pr.ID.Repo, pr.ID.Number, pr.Title), "Head: " + pr.HeadSHA, "Action: " + state, "", "y or enter confirm · esc cancel"}
	if m.actionChecking {
		lines = append(lines, m.visual.paint(rolePending, "Checking GitHub policy…"))
	}
	return strings.Join(m.visual.panel("Confirm "+m.confirmAction, lines, m.width), "\n") + "\n"
}
