package fetch

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fullsend-ai/fs-dash/internal/domain"
)

func TestRateLimitGateWaitsForPrimaryResetAndRetryAfter(t *testing.T) {
	for _, tc := range []struct {
		name       string
		quota      *domain.QuotaObservation
		retryAfter time.Duration
	}{
		{name: "primary reset", quota: &domain.QuotaObservation{Remaining: 0, ResetAt: time.Now().Add(35 * time.Millisecond)}},
		{name: "secondary retry after", retryAfter: 35 * time.Millisecond},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gate := NewRateLimitGate()
			issue := &domain.FetchError{Kind: domain.ErrorRateLimit}
			until := gate.ObserveResponse(issue, tc.quota, tc.retryAfter)
			if !until.After(time.Now()) || !issue.RetryAt.Equal(until) {
				t.Fatalf("cooldown=%v issue=%+v", until, issue)
			}
			started := time.Now()
			if err := gate.Wait(context.Background()); err != nil {
				t.Fatal(err)
			}
			if elapsed := time.Since(started); elapsed < 20*time.Millisecond {
				t.Fatalf("request passed before cooldown: %s", elapsed)
			}
		})
	}
}

func TestRateLimitGateUsesBoundedSecondaryBackoffAndResetsAfterSuccess(t *testing.T) {
	now := time.Date(2026, 11, 6, 17, 0, 0, 0, time.UTC)
	gate := NewRateLimitGateWithClock(func() time.Time { return now })
	want := []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute, 15 * time.Minute, 15 * time.Minute}
	var until time.Time
	for i, delay := range want {
		issue := &domain.FetchError{Kind: domain.ErrorRateLimit}
		until = gate.ObserveResponse(issue, nil, 0)
		if got := until.Sub(now); got != delay || !issue.RetryAt.Equal(until) {
			t.Fatalf("failure %d cooldown=%s issue=%+v, want %s", i+1, got, issue, delay)
		}
	}
	now = until.Add(time.Second)
	gate.ObserveResponse(nil, nil, 0)
	issue := &domain.FetchError{Kind: domain.ErrorRateLimit}
	if got := gate.ObserveResponse(issue, nil, 0).Sub(now); got != time.Minute {
		t.Fatalf("secondary cooldown after successful response = %s, want one minute", got)
	}
}

func TestSharedSchedulerBoundsDifferentGitHubQueuesTogether(t *testing.T) {
	scheduler := NewRateLimitedScheduler(1, 0, time.Second)
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	var active, peak atomic.Int32
	run := func(source string) {
		_, _ = scheduler.Do(context.Background(), source, source, func(context.Context) (any, error) {
			current := active.Add(1)
			for {
				old := peak.Load()
				if current <= old || peak.CompareAndSwap(old, current) {
					break
				}
			}
			started <- struct{}{}
			<-release
			active.Add(-1)
			return nil, nil
		})
	}
	go run("github")
	go run("github-activity")
	<-started
	select {
	case <-started:
		t.Fatal("a second source entered the shared request slot")
	case <-time.After(25 * time.Millisecond):
	}
	close(release)
	<-started
	if got := peak.Load(); got != 1 {
		t.Fatalf("cross-source peak concurrency=%d, want 1", got)
	}
}
