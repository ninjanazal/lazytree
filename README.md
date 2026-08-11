# lazytree

A terminal UI for exploring Git history — fast, keyboard-driven, and built for understanding repository topology.

```
● 8a31c2f  feat: add authentication            [HEAD -> main]  Alice    2h ago
● 71bd920  refactor: simplify middleware        [origin/main]   Bob      4h ago
╲
 ● 4fc821a  fix: token expiration              [feature/auth]  Alice    yesterday
```

## Vision

`lazytree` answers the question: **"What happened in this repository, and how did we get here?"**

Open it in any Git repo and immediately get a polished TUI showing complete history with branches, merges, and diffs — all without leaving the terminal.

Inspired by `gitk --all` for history topology and `lazygit` for keyboard-driven UX.

## Features

- `--all`-style history by default: all branches, remotes, tags
- Custom commit graph with lanes, branches, and merge visualisation
- Commit inspector: metadata, changed files, parents, refs
- Diff viewer with syntax coloring
- Async loading — never freezes on large repos
- Responsive layout: side-by-side on wide terminals, single-pane on narrow
- Search by subject, hash, or author
- Ref display: HEAD, local/remote branches, tags

## Installation

```bash
go install github.com/eurico-martins/lazytree/cmd/lazytree@latest
```

Or build from source:

```bash
git clone https://github.com/eurico-martins/lazytree
cd lazytree
make build
```

## Usage

```bash
# Open current repository
lazytree

# Open a specific repository
lazytree /path/to/repo
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `g` | First commit |
| `G` | Last commit |
| `ctrl+d` | Page down |
| `ctrl+u` | Page up |
| `enter` | Inspect commit |
| `d` | View diff |
| `/` | Search commits |
| `a` | Toggle all refs |
| `esc` | Go back |
| `q` / `ctrl+c` | Quit |

In diff view:

| Key | Action |
|-----|--------|
| `n` | Next file |
| `p` | Previous file |

## Architecture

```
Git CLI
   ↓
internal/git    — git command runner + output parsers
   ↓
internal/model  — Commit, Ref, GraphLayout, DiffFile types
   ↓
internal/graph  — lane assignment algorithm + row renderer (no Bubble Tea dependency)
   ↓
internal/ui     — Bubble Tea models + Lip Gloss styled views
   ↓
cmd/lazytree    — entry point: repo detection, program launch
```

The graph/layout engine has zero dependency on Bubble Tea and is fully unit-tested independently.

## Development

```bash
# Run tests
make test

# Vet and format
make lint

# Build binary
make build

# Run against a repo
./lazytree /path/to/repo
```

Requirements: Go 1.22+

## Screenshots

_Screenshots coming soon._

## License

MIT — see [LICENSE](LICENSE).
