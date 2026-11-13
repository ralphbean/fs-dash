package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fullsend-ai/fs-dash/internal/domain"
	"github.com/fullsend-ai/fs-dash/internal/fetch"
)

type detailReaderFunc func(context.Context, domain.DetailKey) (any, domain.Provenance, error)

func (fn detailReaderFunc) FetchDetail(ctx context.Context, key domain.DetailKey) (any, domain.Provenance, error) {
	return fn(ctx, key)
}

type detailActionsFake struct {
	state      domain.PRActionState
	postID     string
	replyID    string
	postErr    error
	replyErr   error
	mergeErr   error
	closeErr   error
	mergeCalls int
	closeCalls int
}

func (f *detailActionsFake) InspectPR(context.Context, domain.ItemID) (domain.PRActionState, error) {
	return f.state, nil
}
func (f *detailActionsFake) PostPRComment(context.Context, domain.ItemID, string) (string, error) {
	return f.postID, f.postErr
}
func (f *detailActionsFake) ReplyToReviewThread(context.Context, domain.ItemID, string, string) (string, error) {
	return f.replyID, f.replyErr
}
func (f *detailActionsFake) ResolveReviewThread(context.Context, domain.ItemID, string) error {
	return nil
}
func (f *detailActionsFake) MergePR(context.Context, domain.ItemID, string, domain.PRActionState) error {
	f.mergeCalls++
	if f.mergeErr == nil {
		f.state.QueueEntryPresent = true
		f.state.QueueEntryState = "QUEUED"
	}
	return f.mergeErr
}
func (f *detailActionsFake) ClosePR(context.Context, domain.ItemID, string) error {
	f.closeCalls++
	if f.closeErr == nil {
		f.state.State = "CLOSED"
	}
	return f.closeErr
}

func detailTestField(value any, now time.Time) *domain.Field[any] {
	field := domain.NewField[any]()
	generation := field.Queue()
	field.Begin(generation)
	field.Succeed(generation, value, domain.Provenance{Source: "github", FetchedAt: now, FreshUntil: now.Add(time.Hour)})
	return &field
}

func TestDetailsKeepFieldFailuresIndependentAndIgnoreOldHeadResponses(t *testing.T) {
	now := time.Date(2026, 11, 9, 14, 0, 0, 0, time.UTC)
	pr := testUIPr(17, now)
	m := loadedModel(t, []domain.PR{pr})
	m.opts.Clock = &testClock{now: now}
	m.opts.Details = detailReaderFunc(func(context.Context, domain.DetailKey) (any, domain.Provenance, error) {
		return nil, domain.Provenance{}, nil
	})
	m.openDetails()
	descKey := detailKey(pr, "description")
	checksKey := detailKey(pr, "checks")
	threadsKey := detailKey(pr, "threads")
	desc := domain.PRDescription{Body: "description for head one"}
	if cmds := m.applyDetailMessage(detailFetchMsg{key: descKey, generation: m.detailFields[descKey].Generation, value: desc, at: now}); len(cmds) != 0 {
		t.Fatalf("unexpected follow-up commands: %d", len(cmds))
	}
	m.applyDetailMessage(detailFetchMsg{key: checksKey, generation: m.detailFields[checksKey].Generation, err: errors.New("checks unavailable"), at: now, attempt: 3})
	if m.detailFields[descKey].Phase != domain.Ready || m.detailFields[checksKey].Phase != domain.Failed || m.mode != "details" {
		t.Fatalf("independent detail states: description=%s checks=%s mode=%s", m.detailFields[descKey].Phase, m.detailFields[checksKey].Phase, m.mode)
	}
	if !strings.Contains(m.View(), "description for head one") || !strings.Contains(m.View(), "unavailable") {
		t.Fatalf("ready detail or failed field state missing from view: %s", m.View())
	}
	oldGeneration := m.detailFields[threadsKey].Generation
	newPR := pr
	newPR.HeadSHA = "head-new"
	listGeneration := m.sections[0].field.Queue()
	m.sections[0].field.Begin(listGeneration)
	next, _ := m.Update(listMsg{repo: pr.ID.Repo, generation: listGeneration, value: []domain.PR{newPR}, at: now.Add(time.Second)})
	m = next.(Model)
	m.applyDetailMessage(detailFetchMsg{key: threadsKey, generation: oldGeneration, value: domain.ReviewThreadList{Threads: []domain.ReviewThread{{ID: "old-thread"}}}, at: now.Add(2 * time.Second)})
	if field := m.detailFields[threadsKey]; field.HasValue || field.Phase != domain.Canceled {
		t.Fatalf("old head detail was retained: %+v", field)
	}
	if current, ok := m.selectedPR(); !ok || current.HeadSHA != "head-new" {
		t.Fatalf("selection did not remain on the revised PR: %+v, %t", current, ok)
	}
}

func TestThreadWalkStableOrderAndReplyDraftSurvivesAmbiguousFailure(t *testing.T) {
	now := time.Date(2026, 11, 9, 14, 0, 0, 0, time.UTC)
	pr := testUIPr(18, now)
	m := loadedModel(t, []domain.PR{pr})
	m.opts.Clock = &testClock{now: now}
	m.opts.Scheduler = fetch.NewScheduler(2, time.Second)
	m.opts.Details = detailReaderFunc(func(_ context.Context, key domain.DetailKey) (any, domain.Provenance, error) {
		if key.Group == "threads" {
			return domain.ReviewThreadList{Threads: []domain.ReviewThread{{ID: "z", Path: "b.go", Line: 3}, {ID: "a", Path: "a.go", Line: 8}, {ID: "b", Path: "a.go", Line: 4}}}, domain.Provenance{}, nil
		}
		return domain.ReviewThreadContext{ThreadID: key.ThreadID}, domain.Provenance{}, nil
	})
	fake := &detailActionsFake{replyErr: errors.New("request timed out")}
	m.opts.Actions = fake
	m.mode = "thread"
	m.detailFields[detailKey(pr, "threads")] = detailTestField(domain.ReviewThreadList{Threads: []domain.ReviewThread{{ID: "z", Path: "b.go", Line: 3}, {ID: "a", Path: "a.go", Line: 8}, {ID: "b", Path: "a.go", Line: 4}}}, now)
	ordered := m.threadList(pr)
	if len(ordered) != 3 || ordered[0].ID != "b" || ordered[1].ID != "a" || ordered[2].ID != "z" {
		t.Fatalf("thread order = %+v", ordered)
	}
	m.beginEditor("b", "thread")
	m.editorBuffer = []rune("please change this")
	m.saveEditorDraft()
	cmds := m.handleEditorKey(tea.KeyMsg{Type: tea.KeyCtrlD})
	if len(cmds) != 1 {
		t.Fatalf("submit commands = %d", len(cmds))
	}
	mutation := cmds[0]()
	msg, ok := mutation.(commentMutationMsg)
	if !ok || msg.err == nil {
		t.Fatalf("expected uncertain reply result, got %#v", mutation)
	}
	refresh := m.applyCommentMutation(msg)
	if len(refresh) != 1 {
		t.Fatalf("verification refresh commands = %d", len(refresh))
	}
	verification := refresh[0]()
	detailMsg, ok := verification.(detailFetchMsg)
	if !ok {
		t.Fatalf("verification message = %#v", verification)
	}
	m.applyDetailMessage(detailMsg)
	if m.mode != "editor" || string(m.editorBuffer) != "please change this" || m.replyDrafts["b"] != "please change this" {
		t.Fatalf("reply draft lost after ambiguous failure: mode=%s buffer=%q drafts=%v", m.mode, string(m.editorBuffer), m.replyDrafts)
	}
	if !strings.Contains(strings.ToLower(m.status), "not verified") {
		t.Fatalf("verification outcome not shown: %q", m.status)
	}
}

func TestCommentEditorWordAndLineNavigation(t *testing.T) {
	now := time.Now()
	pr := testUIPr(23, now)
	m := loadedModel(t, []domain.PR{pr})
	m.mode = "editor"
	m.editorReturnMode = "details"
	m.editorBuffer = []rune("alpha beta\nthird word")
	m.editorCursor = len("alpha beta")
	m.saveEditorDraft()

	steps := []struct {
		key      tea.KeyType
		wantPos  int
		keyLabel string
	}{
		{tea.KeyCtrlLeft, 6, "ctrl+left to the previous word"},
		{tea.KeyCtrlLeft, 0, "ctrl+left to the start"},
		{tea.KeyCtrlRight, 6, "ctrl+right to the next word"},
		{tea.KeyCtrlRight, 11, "ctrl+right across a line break"},
		{tea.KeyCtrlRight, 17, "ctrl+right past the next word"},
		{tea.KeyCtrlA, 11, "ctrl+a to the current line start"},
	}
	for _, step := range steps {
		m.handleEditorKey(tea.KeyMsg{Type: step.key})
		if m.editorCursor != step.wantPos {
			t.Fatalf("%s moved cursor to %d, want %d", step.keyLabel, m.editorCursor, step.wantPos)
		}
	}
	if string(m.editorBuffer) != "alpha beta\nthird word" {
		t.Fatalf("navigation changed the draft: %q", string(m.editorBuffer))
	}
	if !strings.Contains(m.View(), "▏third word") {
		t.Fatalf("caret is not visible at the line start: %s", m.View())
	}
}

func TestMergeRequiresConfirmationAndReportsVerifiedQueueState(t *testing.T) {
	now := time.Now()
	pr := testUIPr(19, now)
	m := loadedModel(t, []domain.PR{pr})
	m.opts.Scheduler = fetch.NewScheduler(2, time.Second)
	m.mode = "details"
	fake := &detailActionsFake{state: domain.PRActionState{HeadSHA: pr.HeadSHA, State: "OPEN", MergeQueueKnown: true, MergeQueueRequired: true}}
	m.opts.Actions = fake
	inspect := m.inspectAction(pr, "merge")
	if len(inspect) != 1 {
		t.Fatalf("inspect commands = %d", len(inspect))
	}
	next, _ := m.Update(inspect[0]())
	m = next.(Model)
	if m.mode != "confirm" || !strings.Contains(m.confirmView(), "merge queue") || fake.mergeCalls != 0 {
		t.Fatalf("merge was not held for confirmation: mode=%s calls=%d view=%s", m.mode, fake.mergeCalls, m.View())
	}
	cmds := m.handleConfirmKey("y")
	if len(cmds) != 1 {
		t.Fatalf("confirmed action commands = %d", len(cmds))
	}
	mutation := cmds[0]()
	next, _ = m.Update(mutation)
	m = next.(Model)
	if fake.mergeCalls != 1 || !strings.Contains(m.status, "QUEUED") {
		t.Fatalf("queue action was not verified: calls=%d status=%q", fake.mergeCalls, m.status)
	}
}

func TestCloseConfirmationCancelSendsNoMutation(t *testing.T) {
	now := time.Now()
	pr := testUIPr(20, now)
	m := loadedModel(t, []domain.PR{pr})
	fake := &detailActionsFake{state: domain.PRActionState{HeadSHA: pr.HeadSHA, State: "OPEN"}}
	m.opts.Actions = fake
	m.applyActionInspect(actionInspectMsg{item: pr.ID, action: "close", state: fake.state})
	if m.mode != "confirm" {
		t.Fatalf("close confirmation mode = %s", m.mode)
	}
	m.handleConfirmKey("esc")
	if fake.closeCalls != 0 || m.mode != "details" {
		t.Fatalf("cancel sent close: calls=%d mode=%s", fake.closeCalls, m.mode)
	}
}

func TestPRCommentDraftClearsOnlyAfterConversationReadback(t *testing.T) {
	now := time.Date(2026, 11, 9, 15, 0, 0, 0, time.UTC)
	pr := testUIPr(22, now)
	m := loadedModel(t, []domain.PR{pr})
	m.opts.Clock = &testClock{now: now}
	m.opts.Scheduler = fetch.NewScheduler(2, time.Second)
	m.opts.Actions = &detailActionsFake{postID: "new-comment"}
	m.opts.Details = detailReaderFunc(func(_ context.Context, key domain.DetailKey) (any, domain.Provenance, error) {
		if key.Group != "conversation" {
			return nil, domain.Provenance{}, errors.New("unexpected detail group")
		}
		return domain.PRConversation{Complete: true, Comments: []domain.PRComment{{ID: "new-comment", Body: "please update this", Author: "me", CreatedAt: now}}}, domain.Provenance{}, nil
	})
	m.beginEditor("", "details")
	m.editorBuffer = []rune("please update this")
	cmds := m.submitEditor()
	if len(cmds) != 1 {
		t.Fatalf("post commands = %d", len(cmds))
	}
	mutation, ok := cmds[0]().(commentMutationMsg)
	if !ok || mutation.err != nil {
		t.Fatalf("comment mutation = %#v", mutation)
	}
	refresh := m.applyCommentMutation(mutation)
	if len(refresh) != 1 {
		t.Fatalf("readback commands = %d", len(refresh))
	}
	verification, ok := refresh[0]().(detailFetchMsg)
	if !ok {
		t.Fatalf("readback message = %#v", verification)
	}
	m.applyDetailMessage(verification)
	if m.mode != "details" || m.prCommentDraft != "" || m.writePending != nil || !strings.Contains(m.status, "verified") {
		t.Fatalf("comment completed before readback or left the draft: mode=%s draft=%q pending=%+v status=%q", m.mode, m.prCommentDraft, m.writePending, m.status)
	}
}

func TestCloseActionRechecksAndVerifiesPersistedState(t *testing.T) {
	now := time.Now()
	pr := testUIPr(23, now)
	m := loadedModel(t, []domain.PR{pr})
	m.opts.Scheduler = fetch.NewScheduler(2, time.Second)
	fake := &detailActionsFake{state: domain.PRActionState{HeadSHA: pr.HeadSHA, State: "OPEN"}}
	m.opts.Actions = fake
	m.mode, m.confirmAction = "confirm", "close"
	m.actionState = &fake.state
	cmds := m.runConfirmedAction()
	if len(cmds) != 1 {
		t.Fatalf("close commands = %d", len(cmds))
	}
	msg, ok := cmds[0]().(actionMutationMsg)
	if !ok || msg.err != nil || msg.state.State != "CLOSED" {
		t.Fatalf("close readback = %#v", msg)
	}
	refresh := m.applyActionMutation(msg)
	if fake.closeCalls != 1 || m.mode != "list" || !strings.Contains(m.status, "closed and verified") || len(refresh) != 1 {
		t.Fatalf("close state/result = calls:%d mode:%s status:%q refresh:%d", fake.closeCalls, m.mode, m.status, len(refresh))
	}
}

func TestDetailsKeyOpensIndependentReadGroups(t *testing.T) {
	pr := testUIPr(21, time.Now())
	m := loadedModel(t, []domain.PR{pr})
	var keys []domain.DetailKey
	m.opts.Details = detailReaderFunc(func(_ context.Context, key domain.DetailKey) (any, domain.Provenance, error) {
		keys = append(keys, key)
		return nil, domain.Provenance{}, nil
	})
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if m.mode != "details" || len(m.detailFields) != len(detailGroups) {
		t.Fatalf("details did not start all independent groups: mode=%s fields=%d", m.mode, len(m.detailFields))
	}
	for _, group := range detailGroups {
		key := detailKey(pr, group)
		if field := m.detailFields[key]; field == nil || field.Phase != domain.Fetching {
			t.Errorf("%s state = %+v", group, field)
		}
	}
	if len(keys) != 0 {
		t.Fatalf("reader ran synchronously: %v", keys)
	}
	if !strings.Contains(m.View(), "T threads") || !strings.Contains(m.View(), "m merge/queue") {
		t.Fatalf("visible detail actions missing: %s", m.View())
	}
}
