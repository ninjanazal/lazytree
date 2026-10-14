# Changelog

All notable changes to lazytree. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project
follows [Semantic Versioning](https://semver.org/) from 1.0.0 on (see the
README "Compatibility" section for what is covered).

## [Unreleased]

## [1.0.0] - 2026-10-15

First public release. Highlights since the initial commits:

### Added
- Stability contract: golden tests pin the `[keys]` action names, `[colors]` names,
  config file keys and CLI flags; README "Compatibility" section.
- `make help` groups targets by purpose, in color (`NO_COLOR=1` disables it).
- Branch/merge graph with incremental lane layout, streaming `git log`, and
  a live diff pane (inspector header, line numbers, syntax highlighting).
- Search (`/`, with `g:`/`s:`/`p:` git-side prefixes), `n`/`N`, jump to HEAD,
  parent/child navigation, numbered merge parents (`1`-`9`), mouse support.
- Zen mode, split view on wide terminals, copy hash (`y`), open in `$EDITOR` (`o`).
- Config file with `[keys]`, `[colors]`, ref filters and fetch interval.
- Stash shown as a `stash` pill with git's helper commits folded away.
- Safe background fetch (non-interactive, never prompts for credentials).
- "Terminal too small" message below the minimum size.

### Fixed
- README listed a non-existent `split` action for `[keys]`; `v` is a fixed binding.
- Repository detection in worktrees, submodules and bare repos.
- Errors carry git's stderr; non-fatal errors show as a footer banner.
