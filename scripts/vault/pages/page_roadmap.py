from common import *

MS = [
    ("M1", "Robustness", "Works everywhere", "red", True, ["ResolveRepo via rev-parse (worktrees, submodules, bare)", "git's stderr in every error", "non-fatal errors as a footer banner", "tests: ParseDiff, classifyRef, repo detection"]),
    ("M2", "Performance", "Large repos", "orange", True, ["cached graph lines + column widths", "incremental Layouter", "refs once per load; reload only if changed", "streaming git log instead of --skip", "measured on 108k commits (make perf)"]),
    ("M3", "Understand a commit", "Inspect", "yellow", True, ["inspector header: hash, people, parents, refs, body", "file summary with +/- counts", "line numbers, binary files, renames", "syntax highlighting (chroma)", "32-entry LRU diff cache", "p / c follow parent / child"]),
    ("M4", "Navigate & search", "Find things", "green", True, ["? help overlay from the key map", "highlights + n / N; bodies & refs searched", "g: s: p: git-side search", "H jump to HEAD", "mouse wheel + click"]),
    ("M5", "Actions & config", "Light touch", "blue", True, ["y copy hash · o open in $EDITOR", "config file: fetch, theme, layout", "[keys] remap · [colors] palette", "t / r ref filters + hide_refs", "split view"]),
    ("M6", "Ship it", "Release", "purple", False, ["☑ local checks: make check + docker-matrix", "☑ goreleaser archives (make dist)", "☑ make release: tag + files + notes", "☑ README with screenshots + GIF", "☐ tag v0.1.0, push, upload by hand", "☐ Homebrew / AUR (optional)"]),
]


def build():
    s = start("roadmap", "Six milestones, each about one pull request, ordered by value for effort. Five are done.")
    section(s, 0, 240, "Timeline", "green", 1700)
    s.line([(20, 360), (1680, 360)], color=PAL["gray"][0], sw=4)
    for i, (m, t, sub, col, done, tasks) in enumerate(MS):
        x = i * 284
        s.ellipse(x + 112, 340, 40, 40, "green" if done else "yellow")
        s.text(x + 120, 346, "✓" if done else "…", size=20, color=PAL["green" if done else "yellow"][0])
        body = "\n".join(("✔ " if done else "") + tk if not tk.startswith(("☑", "☐")) else tk for tk in tasks)
        wrapped = []
        for ln in body.split("\n"):
            while len(ln) > 30:
                cut = ln.rfind(" ", 0, 30); wrapped.append(ln[:cut]); ln = "   " + ln[cut + 1:]
            wrapped.append(ln)
        s.card(x, 410, 268, f"{m} · {t}", f"{sub}\n\n" + "\n".join(wrapped), col, min_h=330, body_size=14, title_size=17)
        s.chip(x, 760, "done ✓" if done else "almost", "green" if done else "yellow", strong=True)
    section(s, 0, 1090, "Progress", "green", 1700)
    s.rect(0, 930, 1700, 50, "gray", fill="#f1f3f5")
    s.rect(0, 930, 1700 * 0.93, 50, "green", strong=True)
    s.text(20, 940, "≈ 93 %  ·  only the v0.1.0 tag and its upload are left", size=20, color=PAL["green"][0], font=F_TITLE)
    section(s, 0, 1030, "Ideas after v0.1.0 (not planned yet)", "green", 1700)
    ideas = [("Blame & file history", "per-file view of who changed what", "purple"), ("Compare two commits", "mark one, diff against another", "blue"),
             ("Windows support", "paths, clipboard, terminal checks", "orange"), ("Packages", "Homebrew tap, AUR", "pink")]
    for i, (t, d, col) in enumerate(ideas):
        s.card(i * 430, 1110, 405, t, d, col, "✨", min_h=110)
    s.note(0, 1260, 1700, "When a milestone task is finished, tick it here and update Status & Risks in the same change (edit scripts/vault/pages, then make vault).", "tip", title="Keeping this page true")
    return s
