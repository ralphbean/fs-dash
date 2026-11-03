package testutil

import (
	"context"
	"sync"
	"time"

	"github.com/fullsend-ai/fs-dash/internal/domain"
	"github.com/fullsend-ai/fs-dash/internal/ports"
)

type PRLister struct {
	Fn func(context.Context, string) (domain.PRList, error)
}

func (f PRLister) ListOpenPRs(ctx context.Context, repo string) (domain.PRList, error) {
	return f.Fn(ctx, repo)
}

type Enricher struct {
	Fn func(context.Context, domain.FieldKey) (any, domain.Provenance, error)
}

func (f Enricher) FetchField(ctx context.Context, key domain.FieldKey) (any, domain.Provenance, error) {
	return f.Fn(ctx, key)
}

type Clock struct{ Time time.Time }

func (c Clock) Now() time.Time { return c.Time }

type Logger struct {
	mu     sync.Mutex
	Events []ports.LogEvent
}

func (l *Logger) Log(event ports.LogEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.Events = append(l.Events, event)
}
func (l *Logger) Snapshot() []ports.LogEvent {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]ports.LogEvent(nil), l.Events...)
}

type Runner struct {
	mu       sync.Mutex
	Commands []ports.Command
	Err      error
}

func (r *Runner) Run(_ context.Context, command ports.Command) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Commands = append(r.Commands, command)
	return r.Err
}
func (r *Runner) Snapshot() []ports.Command {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]ports.Command(nil), r.Commands...)
}

type Config[T any] struct {
	Value T
	Err   error
}

func (c Config[T]) Load() (T, error) { return c.Value, c.Err }
