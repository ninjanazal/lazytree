from common import *


def build():
    s = start("risks", "An honest list: what was fixed, what is still open, the known limits, and what is out of scope on purpose.")
    section(s, 0, 240, "Health dashboard", "red", 1700)
    tiles = [("Tests", "23 test files\nall green", "green"), ("Platforms", "Linux ✓  macOS ✓\nWindows ✗", "yellow"),
             ("Speed", "108k commits\n1.2 s full load", "green"), ("Git versions", "2.39 → 2.54\nin Docker", "green"),
             ("Release", "v1.0.0 planned\n(roadmap M7)", "yellow"), ("Automation", "none that\npublishes", "blue")]
    for i, (t, v, col) in enumerate(tiles):
        x = i * 284
        s.rect(x, 320, 264, 150, col, strong=True)
        s.text(x + 18, 334, t, size=18, color=PAL[col][0], font=F_TITLE)
        s.text(x + 18, 372, v, size=20)
    section(s, 0, 520, "✅ Fixed (used to be critical)", "red", 1700)
    fixed = ["Repo detection failed in worktrees, submodules and bare repos → ResolveRepo (git rev-parse)",
             "Errors lost git's stderr (“exit status 128”) → errors wrap stderr",
             "Any error replaced the whole UI → non-fatal footer banner",
             "Background fetch could hang on a credential prompt → non-interactive env + 2 min timeout",
             "Whole graph re-rendered every frame → cached rows",
             "O(n²) paging with --skip → one streaming git log",
             "Cursor could leave the screen on branchy history → line-based scrolling",
             "Rows overflowed narrow terminals → everything clipped to width; below 24×10 a “too small” message",
             "macOS-only symlinked temp dir broke a test → caught by the Docker symlink re-run"]
    for i, f in enumerate(fixed):
        s.text(0 + (i % 2) * 860, 600 + (i // 2) * 40, "✔ " + f, size=15, color=INK)
    section(s, 0, 800, "⚠ Open / known limits", "red", 1700)
    rows = [["Area", "Limit", "Impact", "Possible fix"],
            ["Diff view", "No horizontal scrolling; long lines are cut", "low", "viewport with x-offset"],
            ["Syntax colours", "Tokenised per line, so block comments may mis-colour", "low", "highlight whole hunks"],
            ["Background fetch", "Uses the network; auth failures are silent (non-interactive)", "low", "fetch_interval = \"off\""],
            ["Theme detection", "Some terminals don't report their background", "low", "theme = light | dark"],
            ["macOS", "Emulated (Docker symlink re-run), not run natively", "medium", "M7: native smoke test"],
            ["Search", "In-memory search only covers loaded commits", "low", "use g: / s: / p: for everything"]]
    s.table(0, 880, [220, 640, 140, 700], rows, "red", size=15)
    section(s, 0, 1270, "⛔ Out of scope on purpose", "red", 1700)
    oos = [("Changing the repo", "checkout, commit, rebase, reset: use git or lazygit"), ("Windows", "not supported until someone needs it"),
           ("Hosted CI / auto-publishing", "everything runs and ships locally"), ("Editing in place", "o opens the patch in your editor instead")]
    for i, (t, d) in enumerate(oos):
        s.note((i % 2) * 860, 1350 + (i // 2) * 110, 840, d, "danger", title=t)
    return s
