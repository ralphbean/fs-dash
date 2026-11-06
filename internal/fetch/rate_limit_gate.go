package fetch

import (
	"context"
	"sync"
	"time"

	"github.com/fullsend-ai/fs-dash/internal/domain"
)

const (
	secondaryCooldownBase = time.Minute
	secondaryCooldownMax  = 15 * time.Minute
)

type RateLimitGate struct {
	mu                sync.Mutex
	until             time.Time
	changed           chan struct{}
	secondaryFailures int
	now               func() time.Time
}

func NewRateLimitGate() *RateLimitGate {
	return NewRateLimitGateWithClock(time.Now)
}

func NewRateLimitGateWithClock(now func() time.Time) *RateLimitGate {
	if now == nil {
		now = time.Now
	}
	return &RateLimitGate{changed: make(chan struct{}), now: now}
}

func (g *RateLimitGate) Wait(ctx context.Context) error {
	for {
		g.mu.Lock()
		now := g.now()
		if !g.until.After(now) {
			g.until = time.Time{}
			g.mu.Unlock()
			return nil
		}
		until, changed := g.until, g.changed
		g.mu.Unlock()

		timer := time.NewTimer(until.Sub(now))
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-changed:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		case <-timer.C:
		}
	}
}

// ObserveResponse applies a primary reset or secondary cooldown and returns
// the latest shared cooldown deadline.
func (g *RateLimitGate) ObserveResponse(issue *domain.FetchError, quota *domain.QuotaObservation, retryAfter time.Duration) time.Time {
	now := g.now()
	desired := time.Time{}
	if quota != nil && quota.Remaining == 0 && quota.ResetAt.After(now) {
		desired = quota.ResetAt
	}
	if issue != nil && issue.Kind == domain.ErrorRateLimit {
		if retryAfter > 0 {
			secondaryUntil := now.Add(retryAfter)
			if secondaryUntil.After(desired) {
				desired = secondaryUntil
			}
		} else if desired.IsZero() {
			g.mu.Lock()
			g.secondaryFailures++
			shift := min(g.secondaryFailures-1, 8)
			delay := secondaryCooldownBase * time.Duration(1<<shift)
			if delay > secondaryCooldownMax {
				delay = secondaryCooldownMax
			}
			g.mu.Unlock()
			desired = now.Add(delay)
		}
	}

	g.mu.Lock()
	if !desired.IsZero() && desired.After(g.until) {
		g.until = desired
		close(g.changed)
		g.changed = make(chan struct{})
	}
	if issue == nil && !g.until.After(now) {
		g.secondaryFailures = 0
	}
	until := g.until
	g.mu.Unlock()
	if issue != nil && issue.Kind == domain.ErrorRateLimit && until.After(issue.RetryAt) {
		issue.RetryAt = until
	}
	return until
}
