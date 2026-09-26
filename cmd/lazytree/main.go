package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eurico-martins/lazytree/internal/git"
	"github.com/eurico-martins/lazytree/internal/ui"
)

func main() {
	repoPath := "."
	if len(os.Args) > 1 {
		repoPath = os.Args[1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	root, err := git.ResolveRepo(ctx, repoPath)
	cancel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "lazytree: %v\n", err)
		os.Exit(1)
	}

	runner := &git.Runner{RepoPath: root}
	model := ui.NewApp(runner)

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
