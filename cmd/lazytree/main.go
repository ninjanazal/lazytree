package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eurico-martins/lazytree/internal/git"
	"github.com/eurico-martins/lazytree/internal/ui"
)

func main() {
	repoPath := "."
	if len(os.Args) > 1 {
		repoPath = os.Args[1]
	}

	root, err := findGitRoot(repoPath)
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

// findGitRoot walks up from dir looking for a .git directory.
func findGitRoot(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("cannot resolve path %q: %w", dir, err)
	}

	current := abs
	for {
		info, err := os.Stat(filepath.Join(current, ".git"))
		if err == nil && info.IsDir() {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", fmt.Errorf("%q is not inside a git repository", abs)
}
