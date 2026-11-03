package ports

import (
	"context"
	"time"

	"github.com/fullsend-ai/fs-dash/internal/domain"
)

type PRLister interface {
	ListOpenPRs(ctx context.Context, repository string) ([]domain.PR, error)
}

type Enricher interface {
	FetchField(ctx context.Context, key domain.FieldKey) (any, domain.Provenance, error)
}

type Clock interface{ Now() time.Time }

type LogEvent struct {
	Time      time.Time
	Level     string
	Source    string
	Operation string
	Item      string
	Attempt   int
	Duration  time.Duration
	Message   string
}

type Logger interface{ Log(LogEvent) }

type Command struct {
	Executable string
	Args       []string
	Dir        string
	Detach     bool
}

type CommandRunner interface {
	Run(context.Context, Command) error
}

type ConfigLoader[T any] interface{ Load() (T, error) }
