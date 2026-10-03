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
- Commit inspector: full hash, author/committer, parents, refs, message and a +/- file summary
- Syntax-highlighted diff of the selected commit with line numbers (popup on `enter`)
- Async, streaming log loading — never freezes on large repos
- Background `git fetch --all` every 60s with a countdown in the footer
- Incremental search over subject, message body, hash, author and ref names, with highlights and `n`/`N`; `g:text`, `s:text`, `p:path` search git history by message, code or path
- Follow parent/child commits, jump to HEAD, mouse wheel and click
- Side-by-side log and diff on wide terminals; configurable keys, colors and ref filters
- Zen mode: an animated view of the tree (`z`)
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

## Configuration

Optional `~/.config/lazytree/config.toml`. Every key is optional:

```toml
fetch_interval = "60s"   # "off" disables the background fetch (min 5s)
show_all = true          # start with all refs
theme = "auto"           # auto | light | dark
layout = "auto"          # auto (side-by-side diff on >=140 columns) | split | popup
hide_tags = false
hide_remotes = false
hide_refs = ["origin/dependabot/*"]   # ref-name globs to hide; "*" matches anything
lane_colors = ["#5f87ff", "208"]      # graph lane colors

[keys]                   # rebind actions, e.g. zen = ["x"]
zen = ["x"]

[colors]                 # override colors with "#rrggbb" or 0-255
hash = "#ff8700"
```

Key actions: `up down page_up page_down top bottom enter back search next_match prev_match parent child head copy edit toggle_refs toggle_tags toggle_remotes split zen help quit`.
Color names: `selected hash author date help border accent title error head branch remote tag added removed diff_file diff_hunk`.

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
| `enter` | Open diff popup |
| `/` | Search commits |
| `H` | Jump to HEAD |
| `p` / `c` | Parent / child commit (also in the diff popup) |
| `n` / `N` | Next / previous search match |
| `y` | Copy commit hash |
| `o` | Open the commit's patch in `$EDITOR` |
| `t` / `r` | Hide / show tags / remote branches |
| `v` | Toggle side-by-side diff (wide terminals) |
| `a` | Toggle all refs |
| `?` | Show all key bindings |
| `z` | Zen mode |
| `esc` | Go back / clear search |
| `q` / `ctrl+c` | Quit |

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

Requirements: Go 1.26.5+ (see `go.mod`) and `git`

## Screenshots

_Screenshots coming soon._

## License

MIT — see [LICENSE](LICENSE).
