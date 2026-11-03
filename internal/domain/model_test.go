package domain

import (
	"testing"
	"time"
)

func TestFieldFailureSettlesAndRetainsValue(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	f := NewField[string]()
	g1 := f.Queue()
	if !f.Begin(g1) || !f.Succeed(g1, "old", Provenance{Source: "github", Version: "v1", FetchedAt: now, FreshUntil: now.Add(time.Minute)}) {
		t.Fatal("initial fetch did not complete")
	}
	g2 := f.Queue()
	if !f.Begin(g2) {
		t.Fatal("refresh did not begin")
	}
	issue := &FetchError{Kind: ErrorTransient, Message: "502 Bad Gateway"}
	if !f.Fail(g2, issue, now.Add(time.Second)) {
		t.Fatal("failure not accepted")
	}
	if f.Phase != Cooldown || f.Phase.Active() || !f.HasValue || f.Value != "old" {
		t.Fatalf("incorrect settled state: %+v", f)
	}
	if !f.Stale(now.Add(2 * time.Minute)) {
		t.Fatal("retained value must become stale")
	}
}

func TestFieldRejectsOldGeneration(t *testing.T) {
	f := NewField[int]()
	old := f.Queue()
	newer := f.Queue()
	f.Begin(newer)
	if f.Succeed(old, 1, Provenance{}) || f.Fail(old, &FetchError{Kind: ErrorPermanent}, time.Time{}) {
		t.Fatal("old generation changed current state")
	}
	if !f.Succeed(newer, 2, Provenance{}) || f.Value != 2 {
		t.Fatal("new generation did not win")
	}
}

func TestBotApprovalNeedsCompleteCurrentReviews(t *testing.T) {
	now := time.Now()
	reviews := []Review{{Author: "review[bot]", IsBot: true, State: "APPROVED", At: now, ID: "a"}}
	if BotApproval(reviews, false, "v1").Known {
		t.Fatal("partial reviews should be unknown")
	}
	got := BotApproval(reviews, true, "v1")
	if !got.Known || !got.Value || got.Dependencies[0].Version != "v1" {
		t.Fatalf("unexpected derivation: %+v", got)
	}
	reviews = append(reviews, Review{Author: "review[bot]", IsBot: true, State: "CHANGES_REQUESTED", At: now.Add(time.Second), ID: "b"})
	if BotApproval(reviews, true, "v2").Value {
		t.Fatal("latest review must supersede old approval")
	}
}

func TestDerivationInvalidatesOnDependencyVersion(t *testing.T) {
	var cache BoolDerivation
	calls := 0
	compute := func() DerivedBool { calls++; return DerivedBool{Known: true, Value: calls == 1} }
	first := cache.Resolve([]Dependency{{Name: "reviews", Version: "v1"}}, compute)
	second := cache.Resolve([]Dependency{{Name: "reviews", Version: "v1"}}, compute)
	third := cache.Resolve([]Dependency{{Name: "reviews", Version: "v2"}}, compute)
	if !first.Value || !second.Value || third.Value || calls != 2 {
		t.Fatalf("bad cache behavior: %+v %+v %+v calls=%d", first, second, third, calls)
	}
}
