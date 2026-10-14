from common import *

ACTIONS = ["back", "bottom", "child", "copy", "down", "edit", "enter", "head", "help", "next_match", "page_down",
           "page_up", "parent", "prev_match", "quit", "search", "toggle_refs", "toggle_remotes", "toggle_tags", "top", "up", "zen"]
COLORS = ["accent", "added", "author", "border", "branch", "date", "diff_file", "diff_hunk", "error", "hash", "head",
          "help", "remote", "removed", "selected", "stash", "tag", "title"]
CFG_KEYS = ["fetch_interval", "show_all", "theme", "layout", "hide_tags", "hide_remotes", "hide_refs", "lane_colors", "[keys]", "[colors]"]


def build():
    s = start("compat", "From 1.0.0 on, config, action names, colour names and the command line are a promise. Golden tests keep it; the changelog tells the story.")

    # 1 · the promise
    section(s, 0, 240, "1 · The promise (Semantic Versioning)", "teal", 1700,
            "Breaking the frozen surface needs a new major version. Adding to it is fine in a minor.")
    vs = [("PATCH", "1.0.x", "Bug fixes only.\nNothing a user configured\nstops working.", "green", "🩹"),
          ("MINOR", "1.x.0", "New features, new action\nor colour names, new config\nkeys. Old setups still load.", "blue", "✨"),
          ("MAJOR", "2.0.0", "May rename or remove a\nconfig key, action, colour\nor CLI flag.", "red", "💥")]
    prev = None
    for i, (t, v, d, col, ic) in enumerate(vs):
        b = s.card(i * 580, 350, 520, f"{t}  ·  {v}", d, col, ic, min_h=150, title_size=22, body_size=16)
        if prev:
            s.arrow(prev, b)
        prev = b

    # 2 · what is frozen
    section(s, 0, 560, "2 · What is frozen in 1.x", "teal", 1700, "Each box is pinned by a test, so a rename cannot slip through a review.")
    s.card(0, 650, 820, "Config file  ·  10 entries", "Location: $XDG_CONFIG_HOME/lazytree/config.toml\n"
           "or ~/.config/lazytree/config.toml. Unknown keys stay a fatal error.", "teal", "⚙", min_h=250, body_size=15)
    for i, k in enumerate(CFG_KEYS):
        s.chip((i % 5) * 164, 810 + (i // 5) * 46, k, "teal", strong=False, size=14)
    s.card(880, 650, 820, "Command line  ·  3 forms", "", "gray", "⌨", min_h=250)
    for i, c in enumerate(["lazytree [path]", "lazytree --version", "lazytree --help"]):
        s.chip(900, 705 + i * 38, c, "gray", strong=False, size=15)
    s.text(1160, 708, "Exit code 1 + message on stderr\nwhen the repo or config is invalid.", size=15, color=MUTED)

    s.card(0, 930, 1700, f"[keys] action names  ·  {len(ACTIONS)}", "", "purple", "🎹", min_h=195)
    for i, a in enumerate(ACTIONS):
        s.chip(14 + (i % 8) * 212, 990 + (i // 8) * 46, a, "purple", strong=False, size=14)
    s.card(0, 1140, 1700, f"[colors] names  ·  {len(COLORS)}", "", "pink", "🎨", min_h=150)
    for i, a in enumerate(COLORS):
        s.chip(14 + (i % 9) * 188, 1200 + (i // 9) * 46, a, "pink", strong=False, size=14)

    # 3 · may still change
    section(s, 0, 1340, "3 · What may still change in a minor release", "teal", 1700)
    rows = [["Not covered", "Why it is safe to change", "Where to look"],
            ["Default key bindings", "Users who care rebind them; the action names stay", "internal/ui/keys.go"],
            ["Layout, colours, glyphs", "Looks are not an API; override with [colors]", "styles.go · graph/color.go"],
            ["Message wording", "Banners and flashes are for humans, not scripts", "banner.go · actions.go"],
            ["Go packages", "Everything lives under internal/; there is no library API", "internal/*"],
            ["Fixed keys such as v", "No action name exists yet; adding one is a minor change", "keys.go (idea for 1.1)"]]
    s.table(0, 1420, [360, 800, 540], rows, "teal", size=15, mono_cols=(2,))

    # 4 · golden tests
    section(s, 0, 1700, "4 · How the golden tests guard it", "teal", 1700)
    st = [("Change a name", "edit keys.go,\npalette.go or config.go", "blue", "✏"), ("make test", "golden list no\nlonger matches", "yellow", "🧪"),
          ("Test fails", "“action X was removed\nor renamed (breaking)”", "red", "🚨")]
    prev = None
    for i, (t, d, col, ic) in enumerate(st):
        t_ = t
        b = s.card(i * 330, 1780, 290, t_, d, col, ic, min_h=110, title_size=17)
        if prev:
            s.arrow(prev, b)
        prev = b
    a = s.card(1010, 1760, 690, "Added a new name? (additive)", "Extend the golden list in the same change.\nMinor release, add a CHANGELOG line.", "green", "➕", min_h=90, title_size=17)
    r = s.card(1010, 1880, 690, "Removed or renamed one? (breaking)", "Only for a 2.0. Document the migration in CHANGELOG.", "red", "💥", min_h=90, title_size=17)
    s.arrow(prev, a, sa="r", sb="l", color=PAL["green"][0])
    s.arrow(prev, r, sa="r", sb="l", color=PAL["red"][0])
    rows = [["Test", "File", "Pins"],
            ["TestActionNamesAreStable", "internal/ui/contract_test.go", "the 22 [keys] action names"],
            ["TestPaletteNamesAreStable", "internal/ui/contract_test.go", "the 18 [colors] names"],
            ["TestConfigKeysAreStable", "internal/config/config_test.go", "the documented config.toml still loads"],
            ["TestUsageDocumentsFlags", "cmd/lazytree/main_test.go", "usage text lists --version and --help"]]
    s.table(0, 2050, [420, 560, 720], rows, "teal", size=15, mono_cols=(0, 1))

    # 5 · changelog
    section(s, 0, 2330, "5 · CHANGELOG.md vs the generated release notes", "teal", 1700)
    s.code(0, 2410, 560, "## [Unreleased]\n### Added\n- new feature…\n### Fixed\n- bug…\n\n## [1.0.0] - 2026-xx-xx\n### Added\n…", title="CHANGELOG.md (written by hand)", size=14)
    s.card(620, 2410, 520, "CHANGELOG.md", "Curated, user-facing, grouped as\nAdded · Changed · Fixed · Removed.\nLives in the repo, forever.", "teal", "📒", min_h=150)
    s.card(1180, 2410, 520, "RELEASE_NOTES.md", "Generated by make release from\nconventional commits. Pasted into\nthe GitHub release; not committed.", "pink", "📝", min_h=150)
    s.note(620, 2590, 1080, "Before make release: rename [Unreleased] to the version and date, add a fresh empty [Unreleased] above it.\nscripts/release.sh does not touch the changelog on purpose.", "warn", title="Manual step")

    # 6 · checklist
    section(s, 0, 2760, "6 · Adding a new action or colour", "teal", 1700)
    steps = [("Code", "add it to bindingTargets()\nor paletteTargets()"), ("Help", "add it to helpGroups()\n(a test enforces it)"),
             ("Golden", "extend the list in\ncontract_test.go"), ("Docs", "README + 11 Configuration\npage in this vault"),
             ("Changelog", "one line under\n[Unreleased] → Added")]
    prev = None
    for i, (t, d) in enumerate(steps):
        x = i * 342
        b = s.card(x, 2850, 310, t, d, "teal", min_h=110, title_size=18)
        s.step(x + 285, 2850, i + 1, "teal", r=16)
        if prev:
            s.arrow(prev, b)
        prev = b
    s.note(0, 3000, 1700, "The split view toggle (v) is a fixed key today. A `split` action would be additive, so it can ship in 1.1 without breaking anyone.", "idea", title="Idea for 1.1")
    return s
