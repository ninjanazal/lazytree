from common import *


def build():
    s = start("vision", "The question lazytree answers, who it is for, the principles it follows, and what it will not do.")
    s.rect(0, 250, 1700, 140, "yellow", strong=False)
    s.text(40, 270, "“What happened in this repository,\n  and how did we get here?”", size=40, font=F_TITLE, color=PAL["yellow"][0])
    s.text(1000, 285, "Open lazytree in any repository and immediately\nsee the complete history: branches, merges,\ntags and diffs, without leaving the terminal.", size=18)

    section(s, 0, 430, "Where it comes from", "yellow", 1700)
    a = s.card(0, 510, 420, "gitk --all", "Shows topology beautifully\n…but it is a separate GUI window,\ndated, and slow on huge repos.", "gray", "🖼", min_h=150)
    b = s.card(560, 510, 420, "lazygit", "Fast, keyboard-driven terminal UX\n…but focused on doing things\n(staging, committing), not history.", "gray", "⌨", min_h=150)
    c = s.card(1180, 490, 520, "lazytree", "gitk's picture of the history\n+ lazygit's keyboard feel\n+ a commit inspector and search\nin a fast, read-only terminal UI.", "green", "🌳", min_h=190)
    s.arrow(a, c, "r", "b", color=PAL["gray"][0], via=[(470, 585), (470, 690), (1336, 690)], label="topology", tb=0.3)
    s.arrow(b, c, "r", "l", color=PAL["gray"][0], label="UX", tb=0.6)

    section(s, 0, 730, "Who it is for · typical moments", "yellow", 1700)
    cases = [
        ("🧭", "“Where did this branch come from?”", "See lanes fork and merge; follow\nparents with p, children with c.", "blue"),
        ("🕵", "“When did this line change?”", "s:text searches added/removed code\nacross the whole history (git -S).", "purple"),
        ("📝", "“What is in this release?”", "Tags are labels on the graph; open a\ncommit to read its message and diff.", "green"),
        ("👀", "“What did my team push?”", "Background fetch every 60 s; the view\nrefreshes when refs actually change.", "teal"),
    ]
    for i, (ic, t, b2, col) in enumerate(cases):
        s.card(i * 430, 810, 405, t, b2, col, ic, min_h=140, title_size=18)

    section(s, 0, 1000, "Principles", "yellow", 1700, "These rules explain most of the design. See Design Choices for the details.")
    pr = [("Read-only", "Never modifies a repository: no checkout,\nno commit, no reset. Safe to open anywhere.", "red", "🔒"),
          ("Never freeze", "History streams in pages; git always runs\noff the UI thread; the screen stays live.", "blue", "⚡"),
          ("Exact git semantics", "Uses the real git binary, so mailmap, config,\nworktrees and submodules just work.", "orange", "🧱"),
          ("Picture first", "The graph is the main view; everything else\n(inspector, search) serves understanding it.", "green", "🌿"),
          ("Looks right anywhere", "Light & dark terminals, narrow windows,\nmouse or keyboard, configurable keys.", "purple", "🎨"),
          ("Small & tested", "Five small packages, one-way dependencies,\na test for every bug that was fixed.", "yellow", "🧪")]
    for i, (t, b2, col, ic) in enumerate(pr):
        s.card((i % 3) * 575, 1090 + (i // 3) * 170, 550, t, b2, col, ic, min_h=145)

    section(s, 0, 1460, "Feature map", "yellow", 1700)
    cx, cy = 850, 1840
    hub = s.ellipse(cx - 130, cy - 55, 260, 110, "green", strong=True)
    s.text(cx - 70, cy - 22, "lazytree", size=30, font=F_TITLE, color=PAL["green"][0])
    feats = [("Graph", "lanes · merges · ref labels\n--all or current branch", "green", -560, -230),
             ("Inspect", "hash · author · parents\nstat · numbered diff", "orange", 0, -260),
             ("Search", "incremental · highlights\ng: s: p: across history", "purple", 560, -230),
             ("Navigate", "j/k · pages · H · p/c\nmouse wheel & click", "blue", -600, 40),
             ("Live", "background fetch\ncountdown · auto refresh", "teal", 600, 40),
             ("Actions", "y copy hash\no open in $EDITOR", "pink", -380, 250),
             ("Look & feel", "light/dark · split/popup\nzen mode", "yellow", 0, 270),
             ("Configure", "keys · colours · ref filters\nfetch interval · layout", "gray", 380, 250)]
    for t, b2, col, dx, dy in feats:
        bx, by = cx + dx - 160, cy + dy - 50
        card = s.card(bx, by, 320, t, b2, col, min_h=110, title_size=18, body_size=15)
        s.line([(cx + dx * 0.25, cy + dy * 0.25), (cx + dx * 0.72, cy + dy * 0.72)], color=PAL[col][0], sw=2)

    section(s, 0, 2140, "Deliberately out of scope", "red", 1700)
    s.note(0, 2210, 540, "No checkout, commit, rebase or reset.\nlazygit and git itself do that well.", "danger", title="Not a git client")
    s.note(580, 2210, 540, "Not built or tested there yet; would need\npath, clipboard and terminal work first.", "warn", title="No Windows (for now)")
    s.note(1160, 2210, 540, "Nothing auto-publishes: no hosted CI, no\nrelease bots. Everything runs locally.", "info", title="No automation that publishes")
    return s
