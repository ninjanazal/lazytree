from common import *

D = [
    ("Graph decoupled from the TUI", "internal/graph knows nothing about Bubble Tea.", "Layout and rendering are tested as plain functions.", "Renderer returns coloured strings, so colour is not fully separated yet.", "green", "🌿"),
    ("Shell out to git (no go-git)", "Every operation runs the real git binary.", "Exact git behaviour: config, mailmap, worktrees, submodules.", "Process start-up cost per call; git must be installed.", "orange", "🧱"),
    ("NUL / SOH log format", "%x00 between commits, %x01 between fields, body last.", "Multi-line commit messages can never break the parser.", "Format string must stay in sync with ParseLog.", "gray", "🧾"),
    ("Stream the history", "One long-lived git log, read 500 commits at a time.", "First screen in ~14 ms; no O(n²) --skip paging.", "A process to close when a reload supersedes it.", "blue", "⚡"),
    ("Incremental layout", "Layouter keeps its state between pages.", "Each page costs O(page); 108k commits in ~1.2 s.", "Commits must arrive newest-first, append-only.", "green", "📈"),
    ("Debounced, cached diffs", "Load a diff 150 ms after the cursor stops; keep 32 in an LRU.", "Holding j doesn't spawn a git show per row; going back is instant.", "Small delay before the diff appears.", "orange", "⏱"),
    ("Silent live fetch", "git fetch --all every 60 s; reload only if refs moved.", "The view stays current without you doing anything.", "Network use and credential prompts; configurable / off.", "teal", "🔄"),
    ("Tokens, not cancellation", "Each async chain carries a Seq / Gen; stale messages are dropped.", "Simple and race-free in the Elm loop.", "Abandoned work still runs to completion (except the log stream).", "purple", "🎟"),
    ("--all by default", "Show every branch, remote and tag; a toggles to HEAD only.", "Topology is the point of the tool.", "Busy repos look busy: hence t / r / hide_refs.", "blue", "🌐"),
    ("Jump-style search", "Highlight and jump; never hide rows.", "The graph stays intact while you search.", "Non-matching rows still take space.", "purple", "🔎"),
    ("Two search engines", "Memory for loaded commits; git for g: s: p:.", "Instant as-you-type plus whole-history power.", "Two code paths to keep consistent.", "pink", "🧰"),
    ("Split view on wide terminals", "Diff pane next to the log at ≥ 140 columns.", "See history and changes together, no popups.", "Narrower message column.", "blue", "🪟"),
    ("Adaptive colours", "Every colour has a light and a dark variant.", "Readable in both terminal themes.", "Auto-detection can be wrong: theme = light | dark.", "yellow", "🎨"),
    ("Read-only", "No checkout, commit or reset.", "Safe to open anywhere, simple to reason about.", "You switch to git / lazygit to act.", "red", "🔒"),
    ("Local-only quality & release", "No hosted CI, nothing auto-publishes.", "Full control; checks run on your machine and Docker.", "Discipline needed: run make check before pushing.", "gray", "🏠"),
]


def build():
    s = start("design", "The decisions that shape lazytree. Each card: what was decided, why, what it costs, and the result.")
    section(s, 0, 240, "Decision cards", "purple", 1700, "Legend:  🎯 decision   ✅ why it helps   ⚖ the price we pay")
    for i, (t, what, why, cost, col, ic) in enumerate(D):
        x, y = (i % 3) * 575, 320 + (i // 3) * 250
        body = f"🎯 {what}\n✅ {why}\n⚖ {cost}"
        # wrap long lines at ~58 chars
        out = []
        for ln in body.split("\n"):
            while len(ln) > 58:
                cut = ln.rfind(" ", 0, 58)
                out.append(ln[:cut]); ln = "    " + ln[cut + 1:]
            out.append(ln)
        s.card(x, y, 550, f"{i + 1}. {t}", "\n".join(out), col, ic, min_h=225, body_size=15)
    return s
