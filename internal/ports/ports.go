package ports

import (
	"context"
	"time"

	"github.com/fullsend-ai/fs-dash/internal/domain"
)

type PRLister interface {
	ListOpenPRs(ctx context.Context, repository string) (domain.PRList, error)
}

type Enricher interface {
	FetchField(ctx context.Context, key domain.FieldKey) (any, domain.Provenance, error)
}

type FieldResult struct {
	Value      any
	Provenance domain.Provenance
	Err        error
}

type BatchEnricher interface {
	FetchFields(ctx context.Context, keys []domain.FieldKey) map[domain.FieldKey]FieldResult
}

type PRDetailReader interface {
	FetchDetail(ctx context.Context, key domain.DetailKey) (any, domain.Provenance, error)
}

type PRActionClient interface {
	InspectPR(ctx context.Context, item domain.ItemID) (domain.PRActionState, error)
	PostPRComment(ctx context.Context, item domain.ItemID, body string) (string, error)
	ReplyToReviewThread(ctx context.Context, item domain.ItemID, threadID, body string) (string, error)
	MergePR(ctx context.Context, item domain.ItemID, expectedHead string, state domain.PRActionState) error
	ClosePR(ctx context.Context, item domain.ItemID, expectedHead string) error
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
