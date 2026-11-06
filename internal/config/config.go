package config

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/fullsend-ai/fs-dash/internal/domain"
	"github.com/fullsend-ai/fs-dash/internal/ports"
)

type Binding struct {
	Executable string   `toml:"executable"`
	Args       []string `toml:"args"`
	Dir        string   `toml:"dir"`
	Detach     bool     `toml:"detach"`
}

type Config struct {
	Version      int                `toml:"version"`
	Repositories []string           `toml:"repositories"`
	Bindings     map[string]Binding `toml:"bindings"`
	LogLimit     int                `toml:"log_limit"`
	Theme        string             `toml:"theme"`
	Color        string             `toml:"color"`
}

type Loader struct{ Path string }

func Default() Config {
	opener := "xdg-open"
	if runtime.GOOS == "darwin" {
		opener = "open"
	}
	return Config{Version: 1, LogLimit: 500, Theme: "dark", Color: "auto", Bindings: map[string]Binding{"o": {Executable: opener, Args: []string{"{url}"}, Detach: true}}}
}

func DefaultPath() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		var err error
		base, err = os.UserConfigDir()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(base, "fs-dash", "config.toml"), nil
}

func (l Loader) Load() (Config, error) {
	cfg := Default()
	path := l.Path
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return cfg, err
		}
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	} else if err != nil {
		return cfg, fmt.Errorf("read config %s: %w", path, err)
	}
	meta, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("parse config %s: %w", path, err)
	}
	if len(meta.Undecoded()) > 0 {
		return cfg, fmt.Errorf("unknown config keys in %s: %v", path, meta.Undecoded())
	}
	if err := cfg.Validate(); err != nil {
		return cfg, fmt.Errorf("invalid config %s: %w", path, err)
	}
	return cfg, nil
}

var repoPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
var placeholderPattern = regexp.MustCompile(`\{[^{}]+\}`)
var allowedPlaceholders = map[string]bool{"{url}": true, "{repo}": true, "{number}": true, "{title}": true, "{author}": true}

func (c Config) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("version must be 1, got %d", c.Version)
	}
	if c.LogLimit < 1 || c.LogLimit > 10000 {
		return fmt.Errorf("log_limit must be between 1 and 10000")
	}
	if c.Theme != "dark" && c.Theme != "light" {
		return fmt.Errorf("theme must be \"dark\" or \"light\"")
	}
	if c.Color != "auto" && c.Color != "never" {
		return fmt.Errorf("color must be \"auto\" or \"never\"")
	}
	for _, repo := range c.Repositories {
		if !repoPattern.MatchString(repo) {
			return fmt.Errorf("repository %q must be owner/name", repo)
		}
	}
	for key, binding := range c.Bindings {
		if key == "" || strings.ContainsAny(key, " \t\n") {
			return fmt.Errorf("invalid binding key %q", key)
		}
		if key == "q" || key == "?" || key == "r" || key == "L" || key == "ctrl+c" || key == "h" || key == "l" || key == "s" || key == "z" || key == "u" || key == "e" {
			return fmt.Errorf("binding key %q is reserved", key)
		}
		if binding.Executable == "" {
			return fmt.Errorf("binding %q has no executable", key)
		}
		for _, part := range append(append([]string(nil), binding.Args...), binding.Dir) {
			for _, placeholder := range placeholderPattern.FindAllString(part, -1) {
				if !allowedPlaceholders[placeholder] {
					return fmt.Errorf("binding %q uses unknown placeholder %s", key, placeholder)
				}
			}
		}
	}
	return nil
}

func (c Config) Scope(cwd string) ([]string, error) {
	if len(c.Repositories) > 0 {
		return append([]string(nil), c.Repositories...), nil
	}
	if explicit := os.Getenv("GH_REPO"); repoPattern.MatchString(explicit) {
		return []string{explicit}, nil
	}
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = cwd
	output, err := cmd.Output()
	if err != nil {
		return nil, nil
	}
	repo := ParseGitHubRemote(strings.TrimSpace(string(output)))
	if repo == "" {
		return nil, nil
	}
	return []string{repo}, nil
}

func ParseGitHubRemote(remote string) string {
	remote = strings.TrimSuffix(remote, ".git")
	for _, prefix := range []string{"https://github.com/", "http://github.com/", "git@github.com:", "ssh://git@github.com/"} {
		if strings.HasPrefix(remote, prefix) {
			repo := strings.TrimPrefix(remote, prefix)
			if repoPattern.MatchString(repo) {
				return repo
			}
		}
	}
	return ""
}

func (b Binding) Expand(pr domain.PR) (ports.Command, error) {
	replace := strings.NewReplacer("{url}", pr.URL, "{repo}", pr.ID.Repo, "{number}", strconv.Itoa(pr.ID.Number), "{title}", pr.Title, "{author}", pr.Author)
	args := make([]string, len(b.Args))
	for i, arg := range b.Args {
		args[i] = replace.Replace(arg)
	}
	dir := replace.Replace(b.Dir)
	if dir != "" {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			return ports.Command{}, fmt.Errorf("command working directory %q is unavailable", dir)
		}
	}
	return ports.Command{Executable: b.Executable, Args: args, Dir: dir, Detach: b.Detach}, nil
}
