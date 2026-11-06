package reminder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fullsend-ai/fs-dash/internal/domain"
)

func TestFileStorePersistsStarsAndSnoozesBySourceIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "reminders.json")
	id := domain.ItemID{Source: "github", Host: "github.com", Repo: "owner/repo", Number: 42}
	store := NewFileStore(path)
	if err := store.Load(); err != nil {
		t.Fatal(err)
	}
	if err := store.ToggleStar(id); err != nil {
		t.Fatal(err)
	}
	wake := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	if err := store.Snooze(id, wake); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("reminder file permissions = %o, want 600", info.Mode().Perm())
	}
	reloaded := NewFileStore(path)
	if err := reloaded.Load(); err != nil {
		t.Fatal(err)
	}
	entry := reloaded.Snapshot()[id]
	if !entry.Starred || !entry.SnoozedUntil.Equal(wake) {
		t.Fatalf("persisted reminder = %+v", entry)
	}
	otherHost := domain.ItemID{Source: "github", Host: "ghe.example", Repo: "owner/repo", Number: 42}
	if _, exists := reloaded.Snapshot()[otherHost]; exists {
		t.Fatal("reminders from different hosts collided")
	}
	if err := reloaded.Unsnooze(id); err != nil {
		t.Fatal(err)
	}
	if !reloaded.Snapshot()[id].Starred || !reloaded.Snapshot()[id].SnoozedUntil.IsZero() {
		t.Fatal("unsnooze changed the star or retained the wake time")
	}
}

func TestCorruptFileIsBackedUpAndNeverOverwritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reminders.json")
	original := []byte(`{"version": 1, "entries": [`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	store := NewFileStore(path)
	if err := store.Load(); err == nil || !strings.Contains(err.Error(), "preserved a copy") {
		t.Fatalf("Load() error = %v, want preserved-corrupt-file error", err)
	}
	if err := store.ToggleStar(domain.ItemID{Source: "github", Host: "github.com", Repo: "owner/repo", Number: 1}); err == nil {
		t.Fatal("mutation should be disabled after corrupt state")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("corrupt source file was changed: %q", got)
	}
	backups, err := filepath.Glob(path + ".corrupt-*")
	if err != nil || len(backups) != 1 {
		t.Fatalf("corrupt backups = %v, error = %v", backups, err)
	}
	backup, err := os.ReadFile(backups[0])
	if err != nil || string(backup) != string(original) {
		t.Fatalf("backup contents = %q, error = %v", backup, err)
	}
}

func TestToggleStarRemovesEmptyEntry(t *testing.T) {
	store := NewMemoryStore()
	id := domain.ItemID{Source: "github", Host: "github.com", Repo: "owner/repo", Number: 9}
	if err := store.ToggleStar(id); err != nil {
		t.Fatal(err)
	}
	if len(store.Snapshot()) != 1 {
		t.Fatal("starred item not stored")
	}
	if err := store.ToggleStar(id); err != nil {
		t.Fatal(err)
	}
	if len(store.Snapshot()) != 0 {
		t.Fatal("empty reminder entry was retained")
	}
}

func TestDefaultPathRejectsRelativeXDGStateHome(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "relative/state")
	if _, err := DefaultPath(); err == nil {
		t.Fatal("relative XDG_STATE_HOME should be rejected")
	}
}

func TestDefaultPathUsesTriageFileName(t *testing.T) {
	stateHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateHome)
	got, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(stateHome, "fs-dash", "triage.json")
	if got != want {
		t.Fatalf("DefaultPath() = %q, want %q", got, want)
	}
}

func TestLoadMigratesLegacyReminderStateToTriageFile(t *testing.T) {
	dir := t.TempDir()
	legacyPath := filepath.Join(dir, "reminders.json")
	triagePath := filepath.Join(dir, "triage.json")
	id := domain.ItemID{Source: "github", Host: "github.com", Repo: "owner/repo", Number: 7}
	legacy := NewFileStore(legacyPath)
	if err := legacy.Load(); err != nil {
		t.Fatal(err)
	}
	if err := legacy.ToggleStar(id); err != nil {
		t.Fatal(err)
	}
	wake := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	if err := legacy.Snooze(id, wake); err != nil {
		t.Fatal(err)
	}
	legacyBytes, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatal(err)
	}

	migrated := NewFileStore(triagePath)
	if err := migrated.Load(); err != nil {
		t.Fatal(err)
	}
	entry := migrated.Snapshot()[id]
	if !entry.Starred || !entry.SnoozedUntil.Equal(wake) {
		t.Fatalf("migrated triage entry = %+v, want star and snooze", entry)
	}
	if _, err := os.Stat(triagePath); err != nil {
		t.Fatalf("new triage file was not written: %v", err)
	}
	legacyAfter, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(legacyAfter) != string(legacyBytes) {
		t.Fatal("legacy reminder file changed during migration")
	}
}
