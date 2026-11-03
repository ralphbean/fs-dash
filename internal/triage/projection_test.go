package triage

import (
	"reflect"
	"testing"
	"time"

	"github.com/fullsend-ai/fs-dash/internal/domain"
)

func TestReviewRequestedFilterKeepsUnknownRowsPending(t *testing.T) {
	now := time.Date(2026, 11, 3, 15, 0, 0, 0, time.UTC)
	inputs := []Input{
		{PR: testPR(1, now), ReviewRequest: domain.DerivedBool{Known: true, Value: true}},
		{PR: testPR(2, now), ReviewRequest: domain.DerivedBool{Known: true, Value: false}},
		{PR: testPR(3, now)},
	}
	got := Project(inputs, FilterReviewRequested, false, now)
	if len(got.Review) != 2 || got.Review[0].PR.ID.Number != 1 || got.Review[1].PR.ID.Number != 3 || !got.Review[1].FilterPending {
		t.Fatalf("review requested projection = %+v", got.Review)
	}
}

func TestAssignedFilterKeepsIncompleteRowsPending(t *testing.T) {
	now := time.Date(2026, 11, 3, 15, 0, 0, 0, time.UTC)
	inputs := []Input{
		{PR: testPR(1, now), Assigned: domain.DerivedBool{Known: true, Value: true}},
		{PR: testPR(2, now), Assigned: domain.DerivedBool{Known: true, Value: false}},
		{PR: testPR(3, now)},
	}
	got := Project(inputs, FilterAssigned, false, now).Review
	if len(got) != 2 || got[0].PR.ID.Number != 1 || got[1].PR.ID.Number != 3 || !got[1].FilterPending {
		t.Fatalf("assigned filter projection = %+v", got)
	}
}

func TestRankingReasonsPriorityAndNumericStableTies(t *testing.T) {
	now := time.Date(2026, 11, 3, 15, 0, 0, 0, time.UTC)
	inputs := []Input{
		{PR: testPR(10, now)},
		{PR: testPR(2, now)},
		{PR: testPR(3, now), ReviewRequest: domain.DerivedBool{Known: true, Value: true}},
		{PR: testPR(4, now), Checks: domain.CheckSummary{Failed: 1, Complete: true}},
		{PR: testPR(5, now), Threads: domain.ThreadSummary{Unresolved: 2, Complete: true}},
		{PR: testPR(6, now.Add(-8*24*time.Hour))},
		{PR: testPR(7, now), BotApproval: domain.DerivedBool{Known: true, Value: true}},
		{PR: testPR(8, now), BotApproval: domain.DerivedBool{}},
	}
	got := Project(inputs, FilterAll, false, now).Review
	want := []int{3, 4, 5, 6, 2, 8, 10, 7}
	var numbers []int
	for _, row := range got {
		numbers = append(numbers, row.PR.ID.Number)
	}
	if !reflect.DeepEqual(numbers, want) {
		t.Fatalf("priority order = %v, want %v", numbers, want)
	}
	if got[0].Score != 100 || got[0].Reasons[0].Code != "review_requested" {
		t.Fatalf("request reason = %+v", got[0])
	}
	if got[5].Score != 0 || got[7].Score != -30 {
		t.Fatalf("unknown approval and known bot approval scores = %d, %d", got[5].Score, got[7].Score)
	}
}

func TestActivityAndSnoozeSectionsAndWake(t *testing.T) {
	now := time.Date(2026, 11, 3, 15, 0, 0, 0, time.UTC)
	inputs := []Input{
		{PR: testPR(1, now), Activity: domain.ActivityEvidence{State: domain.ActivityActive}},
		{PR: testPR(2, now), SnoozedUntil: now.Add(time.Hour)},
	}
	visible := Project(inputs, FilterAll, false, now)
	if len(visible.Review) != 0 || len(visible.Active) != 1 || len(visible.Snoozed) != 0 {
		t.Fatalf("normal sections = %+v", visible)
	}
	hidden := Project(inputs, FilterAll, true, now)
	if len(hidden.Snoozed) != 1 || hidden.Snoozed[0].PR.ID.Number != 2 {
		t.Fatalf("snoozed view = %+v", hidden.Snoozed)
	}
	woken := Project(inputs, FilterAll, false, now.Add(2*time.Hour))
	if len(woken.Review) != 1 || woken.Review[0].PR.ID.Number != 2 {
		t.Fatalf("expired snooze did not reappear: %+v", woken)
	}
}

func testPR(number int, updated time.Time) domain.PR {
	return domain.PR{ID: domain.ItemID{Source: "github", Host: "github.com", Repo: "owner/repo", Number: number}, UpdatedAt: updated}
}
