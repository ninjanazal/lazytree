from common import *

MS = [
    ("M1", "Robustness", "Works everywhere", "red", True, ["ResolveRepo via rev-parse (worktrees, submodules, bare)", "git's stderr in every error", "non-fatal errors as a footer banner", "tests: ParseDiff, classifyRef, repo detection"]),
    ("M2", "Performance", "Large repos", "orange", True, ["cached graph lines + column widths", "incremental Layouter", "refs once per load; reload only if changed", "streaming git log instead of --skip", "measured on 108k commits (make perf)"]),
    ("M3", "Understand a commit", "Inspect", "yellow", True, ["inspector header: hash, people, parents, refs, body", "file summary with +/- counts", "line numbers, binary files, renames", "syntax highlighting (chroma)", "32-entry LRU diff cache", "p / c follow parent / child"]),
    ("M4", "Navigate & search", "Find things", "green", True, ["? help overlay from the key map", "highlights + n / N; bodies & refs searched", "g: s: p: git-side search", "H jump to HEAD", "mouse wheel + click"]),
    ("M5", "Actions & config", "Light touch", "blue", True, ["y copy hash · o open in $EDITOR", "config file: fetch, theme, layout", "[keys] remap · [colors] palette", "t / r ref filters + hide_refs", "split view"]),
    ("M6", "Ship it", "Release tooling", "purple", True, ["local checks: make check + docker-matrix", "goreleaser archives (make dist)", "make release: tag + files + notes", "README with screenshots + GIF"]),
]

# M7 — the first public release is v1.0.0 (the planned v0.1.0 was skipped).
# (title, why it matters for 1.0, done?)
MUST = [
    ("Safe background fetch", "Fetch runs with GIT_TERMINAL_PROMPT=0 and ssh BatchMode, so a credential prompt can never hang or corrupt the UI.", True),
    ("Jumpable parents", "The parent hashes in the inspector header can be selected and jumped to. This is the last open M3 item.", False),
    ("Too-small terminal message", "Below the minimum size, show “terminal too small” instead of a broken layout.", True),
    ("Stability contract", "Freeze the config keys, [keys] action names, [colors] names and CLI. Golden tests for ActionNames() and PaletteNames(), plus a README “Compatibility” section.", False),
    ("CHANGELOG.md", "A history people can read, seeded from the release notes. Release docs and examples use v1.0.0.", False),
    ("Native macOS smoke test", "Run the real binary once on a Mac: clipboard, $EDITOR, theme detection.", False),
]
NICE = [
    ("Diff horizontal scroll", "long lines are cut today", "blue"),
    ("Whole-hunk highlighting", "block comments mis-colour", "teal"),
    ("Homebrew tap / AUR", "after the tag is public", "pink"),
]
SHIP = ["make release-dry VERSION=v1.0.0", "make release VERSION=v1.0.0", "git push origin main v1.0.0", "GitHub release: upload dist/ + notes"]


def build():
    s = start("roadmap", "Six milestones are done. M7 is the road to the first public release, v1.0.0.")
    section(s, 0, 240, "Timeline", "green", 1700)
    s.line([(20, 360), (1680, 360)], color=PAL["gray"][0], sw=4)
    for i, (m, t, sub, col, done, tasks) in enumerate(MS):
        x = i * 284
        s.ellipse(x + 112, 340, 40, 40, "green" if done else "yellow")
        s.text(x + 120, 346, "✓" if done else "…", size=20, color=PAL["green" if done else "yellow"][0])
        body = "\n".join(("✔ " if done else "") + tk for tk in tasks)
        wrapped = []
        for ln in body.split("\n"):
            while len(ln) > 30:
                cut = ln.rfind(" ", 0, 30); wrapped.append(ln[:cut]); ln = "   " + ln[cut + 1:]
            wrapped.append(ln)
        s.card(x, 410, 268, f"{m} · {t}", f"{sub}\n\n" + "\n".join(wrapped), col, min_h=330, body_size=14, title_size=17)
        s.chip(x, 760, "done ✓", "green", strong=True)

    done = sum(1 for *_, d in MUST if d)
    section(s, 0, 840, f"M7 · v1.0.0 “Stable”  ({done}/{len(MUST)} must-haves)", "green", 1700,
            "1.0.0 is a promise: config, keys and CLI won't break without a 2.0. These items must be done before tagging.")
    for i, (t, why, d) in enumerate(MUST):
        x, y = (i % 3) * 575, 930 + (i // 3) * 190
        wrapped, ln = [], why
        while len(ln) > 52:
            cut = ln.rfind(" ", 0, 52); wrapped.append(ln[:cut]); ln = ln[cut + 1:]
        wrapped.append(ln)
        s.card(x, y, 555, f"{'☑' if d else '☐'} {t}", "\n".join(wrapped), "green" if d else "yellow", min_h=170, body_size=15, title_size=18)

    section(s, 0, 1340, "Nice to have (can slip to 1.1)", "green", 1700)
    for i, (t, d, col) in enumerate(NICE):
        s.card(i * 575, 1410, 555, t, d, col, "✨", min_h=100)

    section(s, 0, 1560, "Ship day", "green", 1700, "Nothing is pushed automatically; steps 3–4 are by hand.")
    for i, st in enumerate(SHIP):
        x = i * 430
        s.step(x + 18, 1668, i + 1, "green")
        s.text(x + 46, 1656, st, size=15, font=F_MONO)

    section(s, 0, 1730, "Ideas after v1.0.0 (not planned yet)", "green", 1700)
    ideas = [("Blame & file history", "per-file view of who changed what", "purple"), ("Compare two commits", "mark one, diff against another", "blue"),
             ("Windows support", "paths, clipboard, terminal checks", "orange"), ("Jump to any ref", "fuzzy picker over branches and tags", "pink")]
    for i, (t, d, col) in enumerate(ideas):
        s.card(i * 430, 1810, 405, t, d, col, "✨", min_h=110)
    s.note(0, 1960, 1700, "When a task is finished, tick it here (set its flag to True) and update Status & Risks in the same change (edit scripts/vault/pages, then make vault).", "tip", title="Keeping this page true")
    return s
