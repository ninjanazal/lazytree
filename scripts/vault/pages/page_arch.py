from common import *


def build():
    s = start("arch", "How lazytree is put together: five small packages, a strict one-way dependency chain, and a UI layer that never blocks.")

    # ---------------------------------------------------------------- A
    section(s, 0, 240, "1 · The big picture", "blue", 1080,
            "Arrows read “depends on / calls”. Nothing below ever imports anything above it.")

    cfg = s.card(0, 330, 300, "internal/config", "Reads config.toml\nvalidates it\n→ Config struct", "teal", "⚙", min_h=150)
    cmd = s.card(380, 330, 480, "cmd/lazytree", "main.go · the entry point\nargs · config · repo · tea.Program\n(alt screen + mouse)", "purple", "🚀", min_h=150)
    ui = s.card(330, 520, 580, "internal/ui",
                "Bubble Tea model (Elm architecture)\nAppModel · log pane · diff pane · zen pane\nsearch · keys · styles · overlays\nruns git ONLY through async tea.Cmds",
                "blue", "🖥", min_h=200)
    gr = s.card(0, 820, 470, "internal/graph", "Lane assignment (Layouter)\nrow rendering (RenderCommitLine)\nlane colours\n⛔ no Bubble Tea dependency", "green", "🌿", min_h=200)
    gi = s.card(560, 820, 520, "internal/git", "Runner · ResolveRepo · log stream\nrefs · diff parser · search · fetch\nwraps git's stderr in errors", "orange", "🧰", min_h=200)
    mo = s.card(0, 1090, 470, "internal/model", "Commit · Ref · GraphNode · GraphLayout\nDiffFile · DiffHunk · DiffLine\n⛔ pure data: no I/O, no UI", "gray", "🧬", min_h=170)
    gc = s.card(700, 1090, 380, "git binary", "the real `git`, run as a\nchild process (exec)", "ink", "🧱", min_h=170)

    s.arrow(cfg, cmd, label="Config", color=PAL["teal"][0])
    s.arrow(cmd, ui, "b", "t", label="NewAppWithOptions", color=PAL["purple"][0], ta=0.5, tb=0.55)
    s.arrow(ui, gr, "b", "t", label="layout + draw rows", color=PAL["blue"][0], ta=0.2, tb=0.5,
            via=[(446, 790), (235, 790)])
    s.arrow(ui, gi, "b", "t", label="tea.Cmd → results as messages", color=PAL["blue"][0], ta=0.85, tb=0.5,
            via=[(821, 790), (820, 790)])
    s.arrow(gr, mo, "b", "t", label="reads / writes", color=PAL["green"][0], ta=0.5, tb=0.5)
    s.arrow(gi, mo, "l", "r", label="builds", color=PAL["orange"][0], ta=0.8, tb=0.5,
            via=[(520, 1000), (520, 1175)])
    s.arrow(gi, gc, "b", "t", label="exec git …", color=PAL["orange"][0], ta=0.75, tb=0.5)

    # rules panel
    s.rect(1160, 330, 540, 1000 - 0, "gray", dashed=True, fill="#fcfcfd")
    s.text(1184, 346, "📏  Ground rules", size=24, color=PAL["gray"][0], font=F_TITLE)
    s.note(1184, 400, 492, "Arrows only point down. A lower package never\nimports a higher one, so each layer can be\ntested without the ones above it.", "info", title="One-way dependencies")
    s.note(1184, 535, 492, "internal/graph has zero Bubble Tea imports, so\nlane layout and row rendering are unit-tested\nas plain functions (layout_test.go, render_test.go).", "tip", title="The graph is UI-agnostic")
    s.note(1184, 685, 492, "View() never touches the disk or runs git.\nEvery git call is a tea.Cmd that returns a\nmessage; the UI just reacts to messages.", "warn", title="Never block the UI")
    s.note(1184, 835, 492, "All git I/O goes through Runner.Run (buffered)\nor Runner.Start (streaming). Errors always carry\ngit's own stderr, never a bare “exit status 128”.", "idea", title="One door to git")
    s.note(1184, 985, 492, "internal/config only decodes and validates.\nApplying keys and colours happens in ui\n(ApplyKeys / ApplyColors) before the program starts.", "info", title="Config is dumb on purpose")

    # ---------------------------------------------------------------- B
    y0 = 1420
    section(s, 0, y0, "2 · Inside internal/ui", "blue", 1700,
            "One AppModel owns everything; the panes are plain structs it calls into (value-receiver Bubble Tea style).")
    app = s.card(560, y0 + 100, 580, "AppModel  (app.go)",
                 "mode: log | search | zen      layout: split | popup\n"
                 "flags: popupOpen(diff focus) · helpOpen\n"
                 "guards: reloadGen · debounceSeq · fetchSeq · deepSeq\n"
                 "services: git.Runner · diffCache (LRU) · logStream\n"
                 "Update(msg) → new model + Cmd      View() → string",
                 "blue", "🧠", min_h=240, body_size=15)
    lg = s.card(0, y0 + 100, 480, "logPane", "commits · cursor · offset\ncached graph lines + column widths\nsearch state · ref filter\nrow rendering, highlight, scrolling", "green", "📜", min_h=170)
    df = s.card(0, y0 + 330, 480, "diffPane", "viewport · inspector header\nstat summary · line gutter\nchroma syntax colours", "orange", "🔍", min_h=150)
    se = s.card(1220, y0 + 100, 480, "searchModel", "textinput · placeholder\nprefix hints g: s: p:", "purple", "⌨", min_h=110)
    zn = s.card(1220, y0 + 240, 480, "zenPane", "animated tree + title decrypt\nticks every 120 ms", "teal", "🧘", min_h=110)
    ex = s.card(1220, y0 + 380, 480, "keys · styles · footer · toolbar\noverlay · banner · palette · reffilter", "", "gray", "🧩", min_h=110)
    for t, tx, ty in [(lg, 480, y0 + 185), (df, 480, y0 + 405)]:
        pass
    s.arrow(app, lg, "l", "r", color=PAL["blue"][0], label="owns", ta=0.25)
    s.arrow(app, df, "l", "r", color=PAL["blue"][0], ta=0.8, tb=0.5)
    s.arrow(app, se, "r", "l", color=PAL["blue"][0], label="owns", ta=0.2)
    s.arrow(app, zn, "r", "l", color=PAL["blue"][0], ta=0.55)
    s.arrow(app, ex, "r", "l", color=PAL["blue"][0], ta=0.9)

    # ---------------------------------------------------------------- C
    y1 = y0 + 620
    section(s, 0, y1, "3 · File map", "blue", 1700, "Every Go file and what it is for.")
    rows = [["File", "What it does"],
            ["cmd/lazytree/main.go", "Parses args (--version, --help, path), loads config, applies keys/colours/theme, resolves the repo, starts Bubble Tea."],
            ["internal/config/config.go", "Finds and decodes config.toml, validates every value, rejects unknown keys."],
            ["internal/git/runner.go · error.go", "Run (buffered) / Start (streaming) wrappers around exec git; errors include git's stderr."],
            ["internal/git/repo.go", "ResolveRepo via git rev-parse: worktrees, submodules, bare repos, GIT_DIR."],
            ["internal/git/log.go", "FetchLog / ParseLog (NUL + SOH format) and StartLogStream / LogStream.Next(n) for paging."],
            ["internal/git/refs.go", "BuildRefsByHash, AttachRefs, classifyRef (incl. refs/stash), RefsFingerprint, CurrentBranch."],
            ["internal/git/diff.go · search.go · fetch.go", "git show --patch parser (line numbers, binary, renames); SearchHashes (--grep / -S / path); git fetch --all."],
            ["internal/model/*.go", "Commit, Ref, GraphNode/GraphLayout, DiffFile/DiffHunk/DiffLine."],
            ["internal/graph/layout.go · render.go · color.go", "Layouter (incremental lanes), RenderCommitLine + connectors, adaptive lane palette."],
            ["internal/ui/app.go", "AppModel: Update/View, modes, tick chains, mouse, split layout, git commands."],
            ["internal/ui/pane_log*.go", "Log rows + caches; search state and highlighting; HEAD / parent / child jumps."],
            ["internal/ui/pane_diff.go · highlight.go · diffcache.go", "Inspector header, numbered diff, chroma colours, 32-entry LRU of parsed diffs."],
            ["internal/ui/pane_zen.go", "Zen mode: animated tree and decrypting title."],
            ["internal/ui/keys.go · palette.go · styles.go", "Key bindings + ApplyKeys; colour overrides + ApplyColors; adaptive light/dark styles."],
            ["internal/ui/actions.go · reffilter.go · overlay.go", "Copy hash / open in editor / flash messages; hide tags & remotes; centred popups."],
            ]
    s.table(0, y1 + 90, [520, 1180], rows, "blue", size=15, mono_cols=(0,))
    return s
