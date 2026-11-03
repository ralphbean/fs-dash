package diag

import (
	"strings"
	"testing"

	"github.com/fullsend-ai/fs-dash/internal/ports"
)

func TestBoundedRedactedLog(t *testing.T) {
	b := New(2)
	b.Log(ports.LogEvent{Message: "old"})
	b.Log(ports.LogEvent{Message: "Authorization: Bearer ghp_abcdefghijklmnopqrstuvwxyz"})
	b.Log(ports.LogEvent{Message: "new"})
	events := b.Snapshot()
	if len(events) != 2 || strings.Contains(events[0].Message, "ghp_") || events[1].Message != "new" {
		t.Fatalf("bad log contents: %+v", events)
	}
}
