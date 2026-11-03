package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fullsend-ai/fs-dash/internal/domain"
)

func TestMissingAndInvalidConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	cfg, err := (Loader{Path: path}).Load()
	if err != nil || cfg.Version != 1 || cfg.Bindings["o"].Executable == "" {
		t.Fatalf("bad default: %+v %v", cfg, err)
	}
	if err := os.WriteFile(path, []byte("version = 9\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := (Loader{Path: path}).Load(); err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("expected actionable error, got %v", err)
	}
}

func TestLegacyConfigUsesVisualDefaultsAndRejectsInvalidPreferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("version = 1\nlog_limit = 500\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := (Loader{Path: path}).Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme != "dark" || cfg.Color != "auto" {
		t.Fatalf("legacy config did not keep visual defaults: theme=%q color=%q", cfg.Theme, cfg.Color)
	}

	for _, tc := range []struct {
		name  string
		field string
	}{
		{name: "theme", field: "theme = \"sepia\""},
		{name: "color", field: "color = \"always\""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte("version = 1\nlog_limit = 500\n"+tc.field+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := (Loader{Path: path}).Load(); err == nil {
				t.Fatalf("accepted invalid %s preference", tc.name)
			}
		})
	}
}

func TestBindingKeepsURLAsSingleArgument(t *testing.T) {
	pr := domain.PR{ID: domain.ItemID{Repo: "owner/repo", Number: 3}, URL: "https://github.com/owner/repo/pull/3", Title: "spaces and $(touch /tmp/never)"}
	dir := t.TempDir()
	b := Binding{Executable: "terminal", Args: []string{"codex", "Review {url}", "{title}"}, Dir: dir}
	cmd, err := b.Expand(pr)
	if err != nil {
		t.Fatal(err)
	}
	if len(cmd.Args) != 3 || cmd.Args[1] != "Review "+pr.URL || cmd.Args[2] != pr.Title {
		t.Fatalf("arguments changed: %#v", cmd.Args)
	}
	if cmd.Dir != dir {
		t.Fatalf("working directory changed: %q", cmd.Dir)
	}
}

func TestTriageHotkeysAreReservedFromCustomBindings(t *testing.T) {
	for _, key := range []string{"f", "s", "z", "S", "u"} {
		cfg := Default()
		cfg.Bindings[key] = Binding{Executable: "custom"}
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), key) {
			t.Errorf("binding %q was not rejected: %v", key, err)
		}
	}
}

func TestRemoteDetection(t *testing.T) {
	for _, remote := range []string{"git@github.com:owner/repo.git", "https://github.com/owner/repo.git"} {
		if got := ParseGitHubRemote(remote); got != "owner/repo" {
			t.Fatalf("%q => %q", remote, got)
		}
	}
	if ParseGitHubRemote("https://example.com/owner/repo.git") != "" {
		t.Fatal("accepted non-GitHub remote")
	}
}
