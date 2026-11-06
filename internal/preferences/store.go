package preferences

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/fullsend-ai/fs-dash/internal/statepath"
)

const fileVersion = 1

type Store interface {
	Load() (bool, error)
	SaveEmojiMode(bool) error
}

type FileStore struct {
	path    string
	mu      sync.Mutex
	loaded  bool
	emoji   bool
	loadErr error
}

func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

func DefaultPath() (string, error) {
	return statepath.File("preferences.json")
}

func (s *FileStore) Load() (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loaded {
		return s.emoji, s.loadErr
	}
	s.loaded = true
	s.emoji = true

	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return s.emoji, nil
	}
	if err != nil {
		s.loadErr = fmt.Errorf("read local preferences: %w", err)
		return s.emoji, s.loadErr
	}
	var file struct {
		Version   int   `json:"version"`
		EmojiMode *bool `json:"emoji_mode"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		s.loadErr = fmt.Errorf("parse local preferences: %w", err)
		return s.emoji, s.loadErr
	}
	if file.Version != fileVersion {
		s.loadErr = fmt.Errorf("unsupported local preferences version %d", file.Version)
		return s.emoji, s.loadErr
	}
	if file.EmojiMode == nil {
		s.loadErr = errors.New("local preferences are missing emoji_mode")
		return s.emoji, s.loadErr
	}
	s.emoji = *file.EmojiMode
	return s.emoji, nil
}

func (s *FileStore) SaveEmojiMode(emojiMode bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loadErr != nil {
		return fmt.Errorf("local preferences are read-only after load error: %w", s.loadErr)
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return fmt.Errorf("create local preferences directory: %w", err)
	}
	data, err := json.MarshalIndent(struct {
		Version   int  `json:"version"`
		EmojiMode bool `json:"emoji_mode"`
	}{Version: fileVersion, EmojiMode: emojiMode}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode local preferences: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".preferences-*.tmp")
	if err != nil {
		return fmt.Errorf("create local preferences temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set local preferences permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write local preferences: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync local preferences: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close local preferences: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("replace local preferences: %w", err)
	}
	if dir, err := os.Open(filepath.Dir(s.path)); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	s.loaded = true
	s.emoji = emojiMode
	return nil
}
