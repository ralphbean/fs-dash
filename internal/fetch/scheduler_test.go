package fetch

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fullsend-ai/fs-dash/internal/domain"
)

func TestSchedulerCoalescesSameKey(t *testing.T) {
	s := NewScheduler(2, time.Second)
	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	fn := func(ctx context.Context) (any, error) {
		calls.Add(1)
		close(started)
		select {
		case <-release:
			return "ok", nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	first := make(chan any, 1)
	go func() { value, _ := s.Do(context.Background(), "github", "repo", fn); first <- value }()
	<-started
	second := make(chan any, 1)
	go func() { value, _ := s.Do(context.Background(), "github", "repo", fn); second <- value }()
	time.Sleep(10 * time.Millisecond)
	close(release)
	if <-first != "ok" || <-second != "ok" || calls.Load() != 1 {
		t.Fatalf("request was not coalesced; calls=%d", calls.Load())
	}
}

func TestRetryPolicy(t *testing.T) {
	now := time.Now()
	rate := &domain.FetchError{Kind: domain.ErrorRateLimit, RetryAt: now.Add(3 * time.Minute)}
	if got := RetryAt(rate, 1, now); !got.Equal(rate.RetryAt) {
		t.Fatalf("rate reset ignored: %v", got)
	}
	if got := RetryAt(rate, 3, now); !got.IsZero() {
		t.Fatalf("retry budget ignored: %v", got)
	}
	if Classify(context.DeadlineExceeded).Kind != domain.ErrorTransient {
		t.Fatal("timeout should retry")
	}
	if Classify(context.Canceled).Kind != domain.ErrorCanceled {
		t.Fatal("cancel should be explicit")
	}
	if Classify(errors.New("broken")).Kind != domain.ErrorTransient {
		t.Fatal("unknown error classification")
	}
}
