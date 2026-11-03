package reminder

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
)

const fileVersion = 1

type Entry struct {
	Starred      bool
	SnoozedUntil time.Time
}

type Store interface {
	Snapshot() map[domain.ItemID]Entry
	ToggleStar(domain.ItemID) error
	Snooze(domain.ItemID, time.Time) error
	Unsnooze(domain.ItemID) error
}

type MemoryStore struct {
	mu      sync.RWMutex
	entries map[domain.ItemID]Entry
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{entries: map[domain.ItemID]Entry{}}
}

func (s *MemoryStore) Snapshot() map[domain.ItemID]Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneEntries(s.entries)
}

func (s *MemoryStore) ToggleStar(id domain.ItemID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.entries[id]
	entry.Starred = !entry.Starred
	s.entries[id] = entry
	prune(&s.entries, id)
	return nil
}

func (s *MemoryStore) Snooze(id domain.ItemID, until time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.entries[id]
	entry.SnoozedUntil = until.UTC()
	s.entries[id] = entry
	prune(&s.entries, id)
	return nil
}

func (s *MemoryStore) Unsnooze(id domain.ItemID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.entries[id]
	entry.SnoozedUntil = time.Time{}
	s.entries[id] = entry
	prune(&s.entries, id)
	return nil
}

type FileStore struct {
	path    string
	mu      sync.RWMutex
	entries map[domain.ItemID]Entry
	loaded  bool
	loadErr error
}

func NewFileStore(path string) *FileStore {
	return &FileStore{path: path, entries: map[domain.ItemID]Entry{}}
}

func DefaultPath() (string, error) {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "state")
	} else if !filepath.IsAbs(base) {
		return "", fmt.Errorf("XDG_STATE_HOME must be an absolute path")
	}
	return filepath.Join(base, "fs-dash", "reminders.json"), nil
}

func (s *FileStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loaded {
		return s.loadErr
	}
	s.loaded = true
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		s.loadErr = fmt.Errorf("read local reminders: %w", err)
		return s.loadErr
	}
	var file diskFile
	if err := json.Unmarshal(data, &file); err != nil {
		s.loadErr = s.preserveCorrupt(data, fmt.Errorf("parse local reminders: %w", err))
		return s.loadErr
	}
	if file.Version != fileVersion {
		s.loadErr = s.preserveCorrupt(data, fmt.Errorf("unsupported local reminder version %d", file.Version))
		return s.loadErr
	}
	entries := make(map[domain.ItemID]Entry, len(file.Entries))
	for _, item := range file.Entries {
		if item.Source == "" || item.Host == "" || item.Repo == "" || item.Number < 1 || (!item.Starred && item.SnoozedUntil.IsZero()) {
			s.loadErr = s.preserveCorrupt(data, fmt.Errorf("invalid local reminder entry for %s/%s#%d", item.Host, item.Repo, item.Number))
			return s.loadErr
		}
		id := domain.ItemID{Source: item.Source, Host: item.Host, Repo: item.Repo, Number: item.Number}
		if _, exists := entries[id]; exists {
			s.loadErr = s.preserveCorrupt(data, fmt.Errorf("duplicate local reminder entry for %s", id.String()))
			return s.loadErr
		}
		entries[id] = Entry{Starred: item.Starred, SnoozedUntil: item.SnoozedUntil}
	}
	s.entries = entries
	return nil
}

func (s *FileStore) Snapshot() map[domain.ItemID]Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneEntries(s.entries)
}

func (s *FileStore) ToggleStar(id domain.ItemID) error {
	return s.update(id, func(entry Entry) Entry { entry.Starred = !entry.Starred; return entry })
}

func (s *FileStore) Snooze(id domain.ItemID, until time.Time) error {
	return s.update(id, func(entry Entry) Entry { entry.SnoozedUntil = until.UTC(); return entry })
}

func (s *FileStore) Unsnooze(id domain.ItemID) error {
	return s.update(id, func(entry Entry) Entry { entry.SnoozedUntil = time.Time{}; return entry })
}

func (s *FileStore) update(id domain.ItemID, change func(Entry) Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.loaded {
		return errors.New("local reminders have not been loaded")
	}
	if s.loadErr != nil {
		return fmt.Errorf("local reminders are read-only after load error: %w", s.loadErr)
	}
	next := cloneEntries(s.entries)
	next[id] = change(next[id])
	prune(&next, id)
	if err := s.save(next); err != nil {
		return err
	}
	s.entries = next
	return nil
}

type diskFile struct {
	Version int         `json:"version"`
	Entries []diskEntry `json:"entries"`
}

type diskEntry struct {
	Source       string    `json:"source"`
	Host         string    `json:"host"`
	Repo         string    `json:"repo"`
	Number       int       `json:"number"`
	Starred      bool      `json:"starred,omitempty"`
	SnoozedUntil time.Time `json:"snoozed_until,omitempty"`
}

func (s *FileStore) save(entries map[domain.ItemID]Entry) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return fmt.Errorf("create local reminder directory: %w", err)
	}
	ids := make([]domain.ItemID, 0, len(entries))
	for id := range entries {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return stableID(ids[i]) < stableID(ids[j]) })
	file := diskFile{Version: fileVersion, Entries: make([]diskEntry, 0, len(ids))}
	for _, id := range ids {
		entry := entries[id]
		file.Entries = append(file.Entries, diskEntry{Source: id.Source, Host: id.Host, Repo: id.Repo, Number: id.Number, Starred: entry.Starred, SnoozedUntil: entry.SnoozedUntil})
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("encode local reminders: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".reminders-*.tmp")
	if err != nil {
		return fmt.Errorf("create local reminder temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set local reminder permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write local reminders: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync local reminders: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close local reminders: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("replace local reminders: %w", err)
	}
	if dir, err := os.Open(filepath.Dir(s.path)); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	return nil
}

func (s *FileStore) preserveCorrupt(data []byte, cause error) error {
	backup := s.path + ".corrupt-" + time.Now().UTC().Format("20060102T150405.000000000Z")
	if err := os.WriteFile(backup, data, 0600); err != nil {
		return fmt.Errorf("%v (could not preserve backup: %w)", cause, err)
	}
	return fmt.Errorf("%v; preserved a copy at %s", cause, backup)
}

func cloneEntries(entries map[domain.ItemID]Entry) map[domain.ItemID]Entry {
	cloned := make(map[domain.ItemID]Entry, len(entries))
	for id, entry := range entries {
		cloned[id] = entry
	}
	return cloned
}

func prune(entries *map[domain.ItemID]Entry, id domain.ItemID) {
	entry := (*entries)[id]
	if !entry.Starred && entry.SnoozedUntil.IsZero() {
		delete(*entries, id)
	}
}

func stableID(id domain.ItemID) string {
	return id.Source + ":" + id.Host + "/" + id.Repo + "#" + fmt.Sprintf("%09d", id.Number)
}
