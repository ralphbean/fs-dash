package enrichmentcache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fullsend-ai/fs-dash/internal/domain"
)

func TestFileStorePersistsAtomicallyWithBoundedFreshness(t *testing.T) {
	now := time.Date(2026, 11, 6, 17, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "github-cache.json")
	store := NewFileStore(path)
	store.maxSize = 2
	store.now = func() time.Time { return now }
	entries := []Entry{
		cacheEntry(1, "head-1", now, now.Add(time.Minute), now),
		cacheEntry(2, "head-2", now.Add(-time.Minute), now.Add(time.Minute), now.Add(-time.Minute)),
		cacheEntry(3, "head-3", now.Add(-2*time.Minute), now.Add(-25*time.Hour), now.Add(-2*time.Minute)),
	}
	if err := store.Save(entries); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("cache permissions = %o, want 600", info.Mode().Perm())
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 2 || loaded[0].Key.Item.Number != 1 || loaded[1].Key.Item.Number != 2 {
		t.Fatalf("cache pruning or LRU order = %+v", loaded)
	}
	files, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".github-cache-*.tmp"))
	if err != nil || len(files) != 0 {
		t.Fatalf("temporary cache files remain: %v, err=%v", files, err)
	}
}

func TestLoadDiscardsMalformedEntriesAndRejectsOldFileVersions(t *testing.T) {
	now := time.Now().UTC()
	valid, err := json.Marshal(cacheEntry(1, "head", now, now.Add(time.Minute), now))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "github-cache.json")
	data := append([]byte(`{"version":1,"entries":[`), valid...)
	data = append(data, []byte(`,{"key":{"item":{"source":"github"}},"value":{},"provenance":{}}]}`)...)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewFileStore(path)
	loaded, err := store.Load()
	if err != nil || len(loaded) != 1 {
		t.Fatalf("valid entries after malformed-entry filtering = %d, err=%v", len(loaded), err)
	}
	if err := os.WriteFile(path, []byte(`{"version":0,"entries":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); err == nil {
		t.Fatal("unsupported cache version was silently accepted")
	}
}

func TestPruneKeepsExpiredEntriesBrieflyForStaleDisplay(t *testing.T) {
	now := time.Now().UTC()
	fresh := cacheEntry(1, "head-1", now.Add(-5*time.Minute), now.Add(-4*time.Minute), now)
	tooOld := cacheEntry(2, "head-2", now.Add(-26*time.Hour), now.Add(-25*time.Hour), now.Add(-time.Hour))
	got := Prune([]Entry{fresh, tooOld}, now, 10)
	if len(got) != 1 || got[0].Key.Item.Number != 1 {
		t.Fatalf("stale retention pruning = %+v", got)
	}
}

func cacheEntry(number int, revision string, fetched, freshUntil, accessed time.Time) Entry {
	value, _ := json.Marshal(domain.CheckSummary{Failed: number, Complete: true})
	return Entry{
		Key:        domain.FieldKey{Item: domain.ItemID{Source: "github", Host: "github.com", Repo: "owner/repo", Number: number}, Name: "checks", Revision: revision},
		Value:      value,
		Provenance: domain.Provenance{Source: "github", FetchedAt: fetched, FreshUntil: freshUntil},
		LastAccess: accessed,
	}
}
