package enrichmentcache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/fullsend-ai/fs-dash/internal/domain"
	"github.com/fullsend-ai/fs-dash/internal/statepath"
)

const (
	fileVersion    = 1
	DefaultMaxSize = 2000
	staleRetention = 24 * time.Hour
)

type Entry struct {
	Key        domain.FieldKey   `json:"key"`
	Value      json.RawMessage   `json:"value"`
	Provenance domain.Provenance `json:"provenance"`
	LastAccess time.Time         `json:"last_access"`
}

type Store interface {
	Load() ([]Entry, error)
	Save([]Entry) error
}

type FileStore struct {
	path    string
	maxSize int
	now     func() time.Time
	mu      sync.Mutex
}

func NewFileStore(path string) *FileStore {
	return &FileStore{path: path, maxSize: DefaultMaxSize, now: time.Now}
}

func DefaultPath() (string, error) {
	return statepath.File("github-cache.json")
}

func (s *FileStore) Load() ([]Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read GitHub enrichment cache: %w", err)
	}
	var file struct {
		Version int     `json:"version"`
		Entries []Entry `json:"entries"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse GitHub enrichment cache: %w", err)
	}
	if file.Version != fileVersion {
		return nil, fmt.Errorf("unsupported GitHub enrichment cache version %d", file.Version)
	}
	valid := make([]Entry, 0, len(file.Entries))
	for _, entry := range file.Entries {
		if validEntry(entry) {
			valid = append(valid, entry)
		}
	}
	return Prune(valid, s.currentTime(), s.maxEntries()), nil
}

func (s *FileStore) Save(entries []Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries = Prune(entries, s.currentTime(), s.maxEntries())
	data, err := json.MarshalIndent(struct {
		Version int     `json:"version"`
		Entries []Entry `json:"entries"`
	}{Version: fileVersion, Entries: entries}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode GitHub enrichment cache: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create GitHub enrichment cache directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".github-cache-*.tmp")
	if err != nil {
		return fmt.Errorf("create GitHub enrichment cache temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set GitHub enrichment cache permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write GitHub enrichment cache: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync GitHub enrichment cache: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close GitHub enrichment cache: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("replace GitHub enrichment cache: %w", err)
	}
	if dir, err := os.Open(filepath.Dir(s.path)); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	return nil
}

func Prune(entries []Entry, now time.Time, maxSize int) []Entry {
	if maxSize < 1 {
		maxSize = DefaultMaxSize
	}
	byKey := make(map[domain.FieldKey]Entry, len(entries))
	for _, entry := range entries {
		if !validEntry(entry) || (!entry.Provenance.FreshUntil.IsZero() && now.Sub(entry.Provenance.FreshUntil) > staleRetention) {
			continue
		}
		current, exists := byKey[entry.Key]
		if !exists || entry.Provenance.FetchedAt.After(current.Provenance.FetchedAt) {
			byKey[entry.Key] = entry
		}
	}
	pruned := make([]Entry, 0, len(byKey))
	for _, entry := range byKey {
		pruned = append(pruned, entry)
	}
	sort.Slice(pruned, func(i, j int) bool {
		iTime, jTime := pruned[i].LastAccess, pruned[j].LastAccess
		if iTime.IsZero() {
			iTime = pruned[i].Provenance.FetchedAt
		}
		if jTime.IsZero() {
			jTime = pruned[j].Provenance.FetchedAt
		}
		if iTime.Equal(jTime) {
			return pruned[i].Provenance.FetchedAt.After(pruned[j].Provenance.FetchedAt)
		}
		return iTime.After(jTime)
	})
	if len(pruned) > maxSize {
		pruned = pruned[:maxSize]
	}
	return pruned
}

func validEntry(entry Entry) bool {
	key := entry.Key
	if key.Item.Source != "github" || key.Item.Host == "" || key.Item.Repo == "" || key.Item.Number < 1 || key.Revision == "" || !cacheableField(key.Name) {
		return false
	}
	if len(entry.Value) == 0 || !json.Valid(entry.Value) || entry.Provenance.FetchedAt.IsZero() || entry.Provenance.FreshUntil.IsZero() {
		return false
	}
	return true
}

func cacheableField(name string) bool {
	switch name {
	case "checks", "reviews", "review_requests", "assignees", "review_threads", "activity", "last_touched":
		return true
	default:
		return false
	}
}

func (s *FileStore) currentTime() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

func (s *FileStore) maxEntries() int {
	if s.maxSize > 0 {
		return s.maxSize
	}
	return DefaultMaxSize
}
