from common import *


def build():
    s = start("testing", "How we know it works: what each test covers, the commands to run, and the Docker matrix that replaces hosted CI.")
    section(s, 0, 240, "1 · Test pyramid", "yellow", 1700)
    lv = [("Unit: graph, parsers, config, panes", 1500, "green", "layout/render · ParseLog/ParseDiff · config · logPane · diffPane · highlight · LRU"),
          ("Model-level: AppModel.Update with fake messages", 1100, "blue", "keys · search · mouse · split view · flashes · narrow terminals"),
          ("Real git: temp repos", 700, "orange", "ResolveRepo · SearchHashes · submodule/worktree/bare"),
          ("Opt-in: 108k-commit perf", 380, "purple", "make perf")]
    for i, (t, w, col, d) in enumerate(lv):
        x = (1700 - w) / 2
        y = 320 + (3 - i) * 110
        s.rect(x, y, w, 96, col, strong=True)
        s.text(x + 20, y + 10, t, size=19, color=PAL[col][0], font=F_TITLE)
        s.text(x + 20, y + 48, d, size=14)
    section(s, 0, 800, "2 · What is tested where", "yellow", 1700)
    rows = [["Package", "Test files", "Covers"],
            ["internal/git", "diff · log · refs · repo · error · search", "parsers, stderr wrapping, repo kinds, git-side search on real repos"],
            ["internal/graph", "layout · layout_incremental · render", "lane assignment, incremental == full layout, glyphs"],
            ["internal/config", "config_test", "defaults, every option, every error"],
            ["internal/ui", "app · pane_log · pane_diff · pane_zen · banner\nhighlight · diffcache · reffilter · narrow · perf", "Update/View behaviour, scrolling, search, keys, split view, sizes 24×10 → 220×50"],
            ["cmd/lazytree", "main_test", "config → options mapping, usage text"]]
    s.table(0, 880, [240, 620, 840], rows, "yellow", size=15, mono_cols=(0,))
    section(s, 0, 1240, "3 · Commands", "yellow", 1700)
    cmds = [("make test", "all unit tests (seconds)"), ("make check", "lint + tests + compile for 5 targets (incl. Windows)"),
            ("make docker-test", "tests in a clean Debian container"), ("make docker-matrix", "Debian bookworm · trixie · Alpine"),
            ("make perf", "synthetic 108k-commit timings"), ("make demo", "try it on a branchy demo repo")]
    for i, (c, d) in enumerate(cmds):
        x, y = (i % 2) * 860, 1320 + (i // 2) * 70
        b = s.chip(x, y, c, "yellow", strong=True, size=17)
        s.text(b.right + 14, y + 7, d, size=16)
    section(s, 0, 1560, "4 · The Docker matrix (instead of hosted CI)", "yellow", 1700)
    host = s.card(0, 1640, 300, "Your machine", "make docker-matrix", "gray", "💻", min_h=110)
    imgs = [("bookworm", "git 2.39"), ("trixie", "git 2.47"), ("alpine", "git 2.54")]
    for i, (n, g) in enumerate(imgs):
        b = s.card(420, 1620 + i * 110, 340, f"golang:1.26-{n}", g, "blue", "🐳", min_h=90, title_size=17)
        s.arrow(host, b, tb=0.5, ta=0.2 + 0.3 * i)
        r1 = s.card(860, 1620 + i * 110, 380, "make lint test", "", "green", "✅", min_h=50, title_size=16)
        r2 = s.card(1300, 1620 + i * 110, 400, "re-run with TMPDIR → symlink", "", "purple", "🍎", min_h=50, title_size=16)
        s.arrow(b, r1, ta=0.4); s.arrow(r1, r2)
    s.note(860, 1960, 840, "macOS's temp dir is a symlink (/var → /private/var). Running the git tests\nfrom a symlinked TMPDIR reproduces the bug class that once broke on macOS.", "idea", title="🍎 macOS emulation")
    s.note(0, 1960, 820, "Rule of thumb: every fixed bug gets a test that failed before the fix\n(e.g. TestLogPane_CursorStaysVisible, TestNarrowTerminals).", "tip", title="Regression first")
    return s
