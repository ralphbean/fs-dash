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

// Scheduler bounds source concurrency and coalesces identical in-flight reads.
// Each Do performs one attempt; callers represent cooldown before scheduling a retry.
type Scheduler struct {
	mu        sync.Mutex
	inflight  map[string]*call
	limits    map[string]chan struct{}
	perSource int
	timeout   time.Duration
}

func NewScheduler(perSource int, timeout time.Duration) *Scheduler {
	if perSource < 1 {
		perSource = 2
	}
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	return &Scheduler{inflight: map[string]*call{}, limits: map[string]chan struct{}{}, perSource: perSource, timeout: timeout}
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
	limit := s.limits[source]
	if limit == nil {
		limit = make(chan struct{}, s.perSource)
		s.limits[source] = limit
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

	requestCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	select {
	case limit <- struct{}{}:
		defer func() { <-limit }()
	case <-requestCtx.Done():
		pending.err = requestCtx.Err()
		return nil, pending.err
	}
	pending.value, pending.err = fn(requestCtx)
	if pending.err == nil && requestCtx.Err() != nil {
		pending.err = requestCtx.Err()
		pending.value = nil
	}
	return pending.value, pending.err
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
