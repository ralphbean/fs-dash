package preferences

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileStoreDefaultsToEmojiAndPersistsMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	store := NewFileStore(path)
	mode, err := store.Load()
	if err != nil || !mode {
		t.Fatalf("missing preference mode = %t, error = %v; want emoji mode", mode, err)
	}
	if err := store.SaveEmojiMode(false); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("preference file permissions = %o, want 600", info.Mode().Perm())
	}
	reloaded := NewFileStore(path)
	mode, err = reloaded.Load()
	if err != nil || mode {
		t.Fatalf("reloaded preference mode = %t, error = %v; want text mode", mode, err)
	}
	if err := reloaded.SaveEmojiMode(true); err != nil {
		t.Fatal(err)
	}
	mode, err = NewFileStore(path).Load()
	if err != nil || !mode {
		t.Fatalf("second reloaded mode = %t, error = %v; want emoji mode", mode, err)
	}
}

func TestFileStorePreservesUnreadablePreferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	original := []byte(`{"version": 99, "future": true}`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	store := NewFileStore(path)
	if _, err := store.Load(); err == nil {
		t.Fatal("unsupported preference version was accepted")
	}
	if err := store.SaveEmojiMode(false); err == nil {
		t.Fatal("save should be refused after an unsupported preference version")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("unsupported preference file was changed: %q", got)
	}
}

func TestDefaultPathUsesSharedStateDirectory(t *testing.T) {
	stateHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateHome)
	got, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(stateHome, "fs-dash", "preferences.json")
	if got != want {
		t.Fatalf("DefaultPath() = %q, want %q", got, want)
	}
}
