package fetch

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/fullsend-ai/fs-dash/internal/domain"
)

type call struct {
	done  chan struct{}
	value any
	err   error
}

// Scheduler bounds global concurrency, paces request starts, and coalesces identical in-flight reads.
// Each Do performs one attempt; callers represent cooldown before scheduling a retry.
type Scheduler struct {
	mu            sync.Mutex
	inflight      map[string]*call
	limit         chan struct{}
	timeout       time.Duration
	startInterval time.Duration
	nextStart     time.Time
}

func NewScheduler(maxConcurrent int, timeout time.Duration) *Scheduler {
	if maxConcurrent < 1 {
		maxConcurrent = 2
	}
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	return &Scheduler{inflight: map[string]*call{}, limit: make(chan struct{}, maxConcurrent), timeout: timeout}
}

func NewRateLimitedScheduler(maxConcurrent int, startInterval, timeout time.Duration) *Scheduler {
	scheduler := NewScheduler(maxConcurrent, timeout)
	if startInterval > 0 {
		scheduler.startInterval = startInterval
	}
	return scheduler
}

func (s *Scheduler) Do(ctx context.Context, source, key string, fn func(context.Context) (any, error)) (any, error) {
	fullKey := source + ":" + key
	s.mu.Lock()
	if pending := s.inflight[fullKey]; pending != nil {
		s.mu.Unlock()
		select {
		case <-pending.done:
			return pending.value, pending.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	pending := &call{done: make(chan struct{})}
	s.inflight[fullKey] = pending
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.inflight, fullKey)
		close(pending.done)
		s.mu.Unlock()
	}()

	// Time spent waiting for a source slot is queue time, not request time.
	select {
	case s.limit <- struct{}{}:
		defer func() { <-s.limit }()
	case <-ctx.Done():
		pending.err = ctx.Err()
		return nil, pending.err
	}
	if err := s.waitForStart(ctx); err != nil {
		pending.err = err
		return nil, pending.err
	}
	requestCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	pending.value, pending.err = fn(requestCtx)
	if pending.err == nil && requestCtx.Err() != nil {
		pending.err = requestCtx.Err()
		pending.value = nil
	}
	return pending.value, pending.err
}

func (s *Scheduler) waitForStart(ctx context.Context) error {
	s.mu.Lock()
	if s.startInterval <= 0 {
		s.mu.Unlock()
		return nil
	}
	now := time.Now()
	startAt := s.nextStart
	if startAt.Before(now) {
		startAt = now
	}
	s.nextStart = startAt.Add(s.startInterval)
	delay := startAt.Sub(now)
	s.mu.Unlock()
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func Classify(err error) *domain.FetchError {
	if err == nil {
		return nil
	}
	var issue *domain.FetchError
	if errors.As(err, &issue) {
		return issue
	}
	if errors.Is(err, context.Canceled) {
		return &domain.FetchError{Kind: domain.ErrorCanceled, Message: "request canceled"}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &domain.FetchError{Kind: domain.ErrorTransient, Message: "request timed out"}
	}
	return &domain.FetchError{Kind: domain.ErrorTransient, Message: fmt.Sprintf("request failed: %v", err)}
}

func RetryAt(issue *domain.FetchError, attempt int, now time.Time) time.Time {
	if issue == nil || !issue.Retryable() || attempt >= 3 {
		return time.Time{}
	}
	if issue.Kind == domain.ErrorRateLimit && issue.RetryAt.After(now) {
		return issue.RetryAt
	}
	delay := time.Second * time.Duration(1<<min(attempt-1, 4))
	return now.Add(delay + time.Duration(rand.Int63n(int64(delay/2)+1)))
}
