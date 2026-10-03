from common import *


def build():
    s = start("ui", "A guided tour of the screen: every region, the modes you can be in, the two layouts, and the full keyboard map.")
    section(s, 0, 240, "1 · Anatomy of the screen", "blue", 1700,
            "The real UI (split view, dark terminal). Numbers match the legend below.")
    X0, Y0, SC = 0, 320, 1700 / 1800
    s.image(X0, Y0, 1700, 850, "split-dark.png")
    pts = [  # (orig x, orig y, label, explanation)
        (60, 27, "Toolbar", "app + repo name on the left; current branch pill,\nhidden-ref notice and --all / current-branch mode right"),
        (60, 46, "Column header", "GRAPH · COMMIT · MESSAGE · AUTHOR · DATE;\nclipped to the pane width on narrow terminals"),
        (75, 330, "Graph lanes", "one colour per lane; ● is a commit, │ a lane passing\nthrough, ╰ ╮ the connector rows for merges and forks"),
        (215, 120, "Commit hash", "short hash, coloured like its lane so you can\ntrack a branch down the list"),
        (330, 100, "Ref labels", "green = local branch · blue = remote · purple = tag\norange = HEAD → branch.  t / r hide tags / remotes"),
        (40, 137, "Selected row", "tinted background; the row keeps its colours.\nj/k, wheel or click move it"),
        (800, 137, "Author · date", "relative date (\"1mo ago\"), fixed width so the\ncolumns never jump while the clock moves"),
        (1040, 63, "Inspector header", "short hash + hints, full hash, author <email>,\ncommitter if different, date, parents, refs, message"),
        (1020, 230, "File summary", "status letter (A/M/D/R/C) + per-file +/- counts\nand totals, computed from the parsed diff"),
        (1080, 330, "Numbered diff", "old/new line numbers in the gutter; chroma syntax\ncolours; binary files and pure renames are labelled"),
        (300, 833, "Footer", "key hints (generated from the key map) and, on the\nright, search status, flashes, position \"5/42 commits\""),
        (900, 852, "Fetch bar", "fills up until the next background git fetch;\nstays empty when fetching is turned off"),
    ]
    for i, (ox, oy, lab, _) in enumerate(pts, 1):
        cx, cy = X0 + ox * SC, Y0 + oy * SC
        s.step(cx, cy, i, "yellow", r=16)
    ly = Y0 + 880
    for i, (ox, oy, lab, exp) in enumerate(pts, 1):
        col = (i - 1) % 3
        row = (i - 1) // 3
        x, y = col * 575, ly + row * 120
        s.step(x + 20, y + 22, i, "yellow", r=16)
        s.text(x + 48, y + 8, lab, size=19, color=PAL["blue"][0], font=F_TITLE)
        s.text(x + 48, y + 38, exp, size=14)

    # ---------------------------------------------------------------- modes
    y = ly + 4 * 120 + 40
    section(s, 0, y, "2 · Modes and overlays", "blue", 1700,
            "lazytree is always in exactly one mode; overlays sit on top of the log mode.")
    lg = s.card(620, y + 120, 420, "Log mode", "the normal view: move, open,\ncopy, jump, toggle refs", "blue", "📜", min_h=130)
    sr = s.card(0, y + 120, 420, "Search mode", "typing into the / box; the log\njumps live to the first match", "purple", "🔎", min_h=130)
    zn = s.card(1280, y + 120, 420, "Zen mode", "animated tree; every key except\nz / esc / q is ignored", "teal", "🧘", min_h=130)
    dp = s.card(300, y + 400, 480, "Diff focused / popup", "enter: scroll the diff with j/k,\np / c follow parent / child, esc back", "orange", "🔍", min_h=130)
    hp = s.card(900, y + 400, 480, "Help overlay", "? shows every key binding;\nall other keys are swallowed", "gray", "❔", min_h=130)
    s.arrow(lg, sr, "l", "r", label="/", ta=0.35, tb=0.35, color=PAL["purple"][0])
    s.arrow(sr, lg, "r", "l", label="enter (keep) · esc (cancel)", ta=0.7, tb=0.7, color=PAL["purple"][0])
    s.arrow(lg, zn, "r", "l", label="z", ta=0.35, tb=0.35, color=PAL["teal"][0])
    s.arrow(zn, lg, "l", "r", label="z · esc", ta=0.7, tb=0.7, color=PAL["teal"][0])
    s.arrow(lg, dp, "b", "t", label="enter", ta=0.2, tb=0.75, color=PAL["orange"][0])
    s.arrow(dp, lg, "t", "b", label="esc", ta=0.95, tb=0.4, color=PAL["orange"][0], label_pos=0.6)
    s.arrow(lg, hp, "b", "t", label="?", ta=0.75, tb=0.25, color=PAL["gray"][0])
    s.arrow(hp, lg, "t", "b", label="? · esc", ta=0.05, tb=0.6, color=PAL["gray"][0], label_pos=0.6)

    # ---------------------------------------------------------------- layouts
    y2 = y + 620
    section(s, 0, y2, "3 · Two layouts", "blue", 1700,
            "Chosen by terminal width (layout = \"auto\"), forced in the config, or toggled any time with v.")
    s.image(0, y2 + 100, 820, 410, "split-dark.png")
    s.image(880, y2 + 100, 820 * 1100 / 1100, 596 * 820 / 820 * 0.5 * 2 * 410 / 596, "popup-dark.png")
    s.text(0, y2 + 525, "Split view  (≥ 140 columns)", size=22, color=PAL["blue"][0], font=F_TITLE)
    s.text(0, y2 + 560, "Log on the left (55 %), diff permanently on the right.\nThe diff follows the cursor (150 ms debounce, cached).\nenter or a click focuses the diff; esc returns.", size=15)
    s.text(880, y2 + 525, "Popup  (narrower terminals)", size=22, color=PAL["orange"][0], font=F_TITLE)
    s.text(880, y2 + 560, "The log uses the whole width; enter opens the inspector\nand diff as a centred popup (80 % × 80 %).\nesc closes it.", size=15)

    # ---------------------------------------------------------------- keyboard
    y3 = y2 + 680
    section(s, 0, y3, "4 · Keyboard map", "blue", 1700,
            "Coloured keys do something. Shift variants are listed under the key. Every binding can be remapped in the config.")
    groups = {"nav": ("blue", "Navigate"), "search": ("purple", "Search"), "act": ("orange", "Commit actions"),
              "view": ("teal", "View"), "app": ("red", "App")}
    keys_ = {
        "q": ("app", "quit"), "r": ("view", "remotes"), "t": ("view", "tags"), "y": ("act", "copy hash"),
        "o": ("act", "open $EDITOR"), "p": ("nav", "parent"), "a": ("view", "all refs"),
        "g": ("nav", "first\nG: last"), "h": ("nav", "H: HEAD"), "j": ("nav", "down"), "k": ("nav", "up"),
        "z": ("view", "zen"), "c": ("nav", "child"), "v": ("view", "split"), "n": ("search", "next\nN: prev"),
        "/": ("search", "search"),
    }
    rows = ["qwertyuiop", "asdfghjkl", "zxcvbnm/"]
    kw, kh, gap = 112, 96, 10
    for ri, row in enumerate(rows):
        for ci, ch in enumerate(row):
            x = ri * 40 + ci * (kw + gap)
            yy = y3 + 110 + ri * (kh + gap)
            if ch in keys_:
                g, lab = keys_[ch]
                col = groups[g][0]
                s.rect(x, yy, kw, kh, col, strong=True, sw=2)
                s.text(x + 12, yy + 6, ch, size=30, color=PAL[col][0], font=F_MONO)
                s.text(x + 12, yy + 48, lab, size=13)
            else:
                s.rect(x, yy, kw, kh, "gray", fill="#f8f9fa", stroke="#ced4da", sw=1)
                s.text(x + 12, yy + 6, ch, size=30, color="#ced4da", font=F_MONO)
    sx = 1270
    specials = [("esc", "back · clear search", "app"), ("enter", "open / focus diff", "act"),
                ("?", "help overlay", "view"), ("ctrl+d / ctrl+u", "page down / up", "nav"),
                ("↑ ↓ home end", "same as k j g G", "nav"), ("ctrl+c", "always quits", "app")]
    for i, (k, d, g) in enumerate(specials):
        col = groups[g][0]
        yy = y3 + 110 + i * 54
        b = s.chip(sx, yy, k, col)
        s.text(sx + b.w + 12, yy + 6, d, size=15)
    for i, (g, (col, lab)) in enumerate(groups.items()):
        s.chip(i * 200, y3 + 450, lab, col, strong=True)
    s.note(0, y3 + 510, 1700, "Mouse: the wheel scrolls the log (or the diff when the pointer is over it), a click selects a commit or focuses the diff.\nHold Shift while dragging to select text, since lazytree captures the mouse.", "info", title="Mouse")
    return s
