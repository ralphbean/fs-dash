package diag

import (
	"regexp"
	"strings"
	"sync"

	"github.com/fullsend-ai/fs-dash/internal/ports"
)

var secret = regexp.MustCompile(`(?i)(authorization:\s*\S+(?:\s+\S+)?|gh[opusr]_[A-Za-z0-9_]+|github_pat_[A-Za-z0-9_]+|sk-[A-Za-z0-9_-]{12,})`)

type Buffer struct {
	mu     sync.RWMutex
	limit  int
	events []ports.LogEvent
}

func New(limit int) *Buffer {
	if limit < 1 {
		limit = 500
	}
	return &Buffer{limit: limit}
}

func Redact(s string) string {
	s = secret.ReplaceAllString(s, "[REDACTED_SECRET]")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}

func (b *Buffer) Log(event ports.LogEvent) {
	event.Message = Redact(event.Message)
	event.Item = Redact(event.Item)
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.events) == b.limit {
		copy(b.events, b.events[1:])
		b.events[len(b.events)-1] = event
	} else {
		b.events = append(b.events, event)
	}
}

func (b *Buffer) Snapshot() []ports.LogEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return append([]ports.LogEvent(nil), b.events...)
}
