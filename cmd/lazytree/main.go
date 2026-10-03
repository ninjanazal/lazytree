package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eurico-martins/lazytree/internal/config"
	"github.com/eurico-martins/lazytree/internal/git"
	"github.com/eurico-martins/lazytree/internal/ui"
)

const usage = `lazytree - a terminal UI for exploring Git history

Usage:
  lazytree [path]      open the repository containing path (default: .)
  lazytree --version

Press ? inside lazytree for key bindings. Optional config:
~/.config/lazytree/config.toml (see the README).
`

// version is set at release time (-ldflags "-X main.version=v0.1.0").
var version = "dev"

func main() {
	repoPath := "."
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Println("lazytree", version)
			return
		case "--help", "-h", "help":
			fmt.Print(usage)
			return
		}
		repoPath = os.Args[1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	root, err := git.ResolveRepo(ctx, repoPath)
	cancel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "lazytree: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "lazytree: %v\n", err)
		os.Exit(1)
	}
	switch cfg.Theme {
	case "light":
		lipgloss.SetHasDarkBackground(false)
	case "dark":
		lipgloss.SetHasDarkBackground(true)
	}

	if err := ui.ApplyKeys(cfg.Keys); err != nil {
		fmt.Fprintf(os.Stderr, "lazytree: config [keys]: %v\n", err)
		os.Exit(1)
	}
	if err := ui.ApplyColors(cfg.Colors, cfg.LaneColors); err != nil {
		fmt.Fprintf(os.Stderr, "lazytree: config [colors]: %v\n", err)
		os.Exit(1)
	}

	runner := &git.Runner{RepoPath: root}
	model := ui.NewAppWithOptions(runner, optionsFrom(cfg))

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "lazytree: %v\n", err)
		os.Exit(1)
	}
}

// optionsFrom maps the loaded config onto the UI's options.
func optionsFrom(cfg config.Config) ui.Options {
	return ui.Options{
		FetchInterval: cfg.FetchInterval,
		ShowAll:       cfg.ShowAll,
		HideTags:      cfg.HideTags,
		HideRemotes:   cfg.HideRemotes,
		HideRefs:      cfg.HideRefs,
		Layout:        cfg.Layout,
	}
}
