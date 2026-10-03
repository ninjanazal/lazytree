from common import *

STEPS = [
    ("M", "Merge feature", "[B, F]", "0 (new)", "B → 0 (inherits)\nF → 1 (new lane)", "{0:B, 1:F}", "—", "● \n╰ ╮"),
    ("F", "feature work", "[A]", "1 (reserved)", "A → 1 (inherits)", "{0:B, 1:A}", "—", "│ ●"),
    ("B", "main work", "[A]", "0 (reserved)", "A already on 1", "{1:A}", "lane 0", "● │\n╰ ╮"),
    ("A", "root", "[ ]", "1 (reserved)", "none", "{ }", "lane 1", "  ●"),
]


def build():
    s = start("graph", "How the coloured lanes are computed: one pass over the commits, newest first, with a little bookkeeping.")
    section(s, 0, 240, "1 · The idea in plain words", "green", 1700)
    tips = [("Rows are commits", "git log prints commits newest first, every\ncommit before its parents. Each row is one commit.", "blue", "📜"),
            ("Columns are lanes", "A lane is a vertical line reserved for a line\nof history. Lane 0 is the leftmost.", "green", "🛤"),
            ("Parents reserve lanes", "When a commit is placed, it books a lane for\neach parent that has none yet.", "orange", "📌"),
            ("First parent stays", "The first parent inherits the commit's own\nlane, so the main line stays straight.", "purple", "➡"),
            ("Merges open lanes", "Every extra parent gets a new lane: the lowest\nfree one, or a brand-new one on the right.", "teal", "🔀"),
            ("Lanes are recycled", "When nobody continues a lane, it goes back on\nthe free list, so the graph stays narrow.", "red", "♻")]
    for i, (t, b, col, ic) in enumerate(tips):
        s.card((i % 3) * 575, 320 + (i // 3) * 165, 550, t, b, col, ic, min_h=140)

    y = 680
    section(s, 0, y, "2 · Worked example", "green", 1700, "Four commits: a feature branch F merged into main by M. Read the table top to bottom, like the algorithm.")
    s.code(0, y + 90, 420, "A ── B ──────── M   (main)\n \\            /\n  └── F ─────┘     (feature)\n\ngit log order:  M, F, B, A", title="the history")
    rows = [["Row", "Commit", "Parents", "Own lane", "Parents get", "Active lanes after", "Freed", "Drawn"]]
    for st in STEPS:
        rows.append([st[0], st[1], st[2], st[3], st[4], st[5], st[6], st[7]])
    s.table(460, y + 90, [60, 150, 90, 130, 190, 170, 90, 120], rows, "green", size=15, mono_cols=(2, 5, 7))
    s.code(0, y + 330, 420, "●          M  Merge feature\n╰ ╮\n│ ●        F  feature work\n● │        B  main work\n╰ ╮\n  ●        A  root", title="result on screen")
    s.note(460, y + 450, 1010, "Row B is the subtle one: B's parent A was already booked on lane 1 by F, so B does not\npass its lane on. Lane 0 is freed, and a connector row draws B joining lane 1.", "idea", title="Why lane 0 is freed at B")

    y = 1300
    section(s, 0, y, "3 · The bookkeeping (Layouter)", "green", 1700)
    s.card(0, y + 80, 540, "laneOf", "map[hash] → lane\nwhich lane each commit is (or will be) drawn in.\nSet when a child books a lane for its parent.", "blue", "🗺", min_h=150)
    s.card(580, y + 80, 540, "freeLanes", "sorted list of recycled lanes\nallocLane takes the lowest; freeLane inserts\nin order, so low lanes are reused first.", "orange", "♻", min_h=150)
    s.card(1160, y + 80, 540, "activeLanes", "map[lane] → hash it is waiting for\nlanes still open below the current row;\nused for the graph width and drawing.", "purple", "📍", min_h=150)
    s.code(0, y + 270, 1700,
           "for each commit c (newest first):\n"
           "    lane = laneOf[c]  or  allocLane()          # step 1: where do I go?\n"
           "    for j, p in parents(c):                    # step 2: book my parents\n"
           "        if p has no lane: laneOf[p] = (j == 0 ? lane : allocLane())\n"
           "        activeLanes[laneOf[p]] = p\n"
           "    if lane not in parent lanes: freeLane(lane)  # step 3: nobody continues me\n"
           "    emit GraphNode{Lane, Parents: parent lanes, Color: lane % palette}", title="internal/graph/layout.go (simplified)")

    y = 1790
    section(s, 0, y, "4 · Drawing a row (render.go)", "green", 1700)
    gl = [("●", "commit dot, in its lane colour"), ("│", "a lane passing through this row"),
          ("╰ ╮", "connector row: a lane forks or merges"), ("─", "horizontal run when lanes are far apart")]
    for i, (g, d) in enumerate(gl):
        x = i * 425
        s.rect(x, y + 80, 400, 110, "green")
        s.text(x + 20, y + 92, g, size=40, color=PAL["green"][0], font=F_MONO)
        s.text(x + 20, y + 150, d, size=15)
    s.note(0, y + 220, 830, "A connector row is only added when the topology changes (a parent\nis in another lane). Straight history stays one line per commit.", "tip", title="Only spend a row when needed")
    s.note(870, y + 220, 830, "The log pane renders every row once into a cache; scrolling only\nreads it. A new page renders just the new rows.", "info", title="Cached")

    y = 2150
    section(s, 0, y, "5 · Why it is fast", "green", 1700)
    s.note(0, y + 80, 1700, "The Layouter keeps laneOf / freeLanes / activeLanes between calls, so each new page of 500 commits costs O(page), not\nO(everything so far). Measured: 108k commits laid out and rendered in ~1.2 s total. Preconditions: commits arrive newest first,\nchildren before parents, append only (exactly what git log streams).", "idea", title="Incremental by design")
    return s
