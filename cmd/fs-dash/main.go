package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fullsend-ai/fs-dash/internal/config"
	"github.com/fullsend-ai/fs-dash/internal/github"
	"github.com/fullsend-ai/fs-dash/internal/localcmd"
	"github.com/fullsend-ai/fs-dash/internal/ui"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	configPath := flag.String("config", "", "config TOML path (default: XDG config home)")
	repository := flag.String("repo", "", "GitHub owner/name to show for this run")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "fs-dash: Fullsend attention dashboard\n\nUsage: fs-dash [options]\n\n")
		flag.PrintDefaults()
		fmt.Fprintln(flag.CommandLine.Output(), "\nIn the TUI: ? help, q quit")
	}
	flag.Parse()
	if *showVersion {
		fmt.Println("fs-dash", version)
		return
	}
	cfg, err := (config.Loader{Path: *configPath}).Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "fs-dash:", err)
		os.Exit(2)
	}
	if *repository != "" {
		cfg.Repositories = []string{*repository}
		if err := cfg.Validate(); err != nil {
			fmt.Fprintln(os.Stderr, "fs-dash:", err)
			os.Exit(2)
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "fs-dash:", err)
		os.Exit(2)
	}
	repos, err := cfg.Scope(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fs-dash:", err)
		os.Exit(2)
	}
	options := ui.Options{Config: cfg, Repos: repos, Lister: github.PRLister{}, Runner: localcmd.Runner{}}
	if _, err := tea.NewProgram(ui.New(options)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "fs-dash:", err)
		os.Exit(1)
	}
}
