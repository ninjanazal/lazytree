from common import *


def fields(s, x, y, w, name, rows, color, note=None):
    s_, soft, hard = PAL[color]
    body = "\n".join(f"{a:<17}{b}" for a, b in rows)
    c = s.card(x, y, w, name, body, color, mono=True, body_size=14)
    if note:
        s.text(x, c.bottom + 8, note, size=13, color=MUTED)
    return c


def build():
    s = start("model", "The handful of plain data types everything else passes around. No methods with side effects, no I/O.")
    section(s, 0, 240, "1 · Types at a glance", "orange", 1700, "internal/model: what each type holds and how they point at each other.")
    cm = fields(s, 0, 320, 520, "Commit", [("Hash", "string  (40 hex)"), ("ShortHash", "string  (7)"),
                                           ("Parents", "[]string  hashes"), ("Author", "string"),
                                           ("AuthorEmail", "string"), ("Committer", "string"),
                                           ("CommitterEmail", "string"), ("Timestamp", "time.Time"),
                                           ("Subject", "string  first line"), ("Body", "string  rest"),
                                           ("Refs", "[]Ref")], "orange", "one per row of the log, from git log")
    rf = fields(s, 640, 320, 420, "Ref", [("Name", "string  e.g. main"), ("Kind", "RefKind"), ("IsHead", "bool")], "green",
                "attached to its commit by AttachRefs")
    rk = fields(s, 640, 580, 420, "RefKind (enum)", [("RefHead", "detached HEAD"), ("RefLocalBranch", "refs/heads/*"),
                                                     ("RefRemoteBranch", "refs/remotes/*"), ("RefTag", "refs/tags/*"), ("RefStash", "refs/stash"), ("RefStashHelper", "hidden stash parents")], "green")
    gn = fields(s, 1180, 320, 520, "GraphNode", [("CommitIndex", "int  row index"), ("Lane", "int  column"),
                                                 ("Parents", "[]int  parent lanes"), ("Color", "int  palette idx")], "teal",
                "one per commit, same order as the commits")
    gl = fields(s, 1180, 580, 520, "GraphLayout", [("Nodes", "[]GraphNode"), ("Width", "int  widest row")], "teal",
                "built by graph.Layouter")
    s.arrow(cm, rf, "r", "l", label="Refs  0..n", ta=0.85, tb=0.5, color=PAL["orange"][0])
    s.arrow(rf, rk, "b", "t", label="Kind", color=PAL["green"][0])
    s.arrow(gl, gn, "t", "b", label="Nodes 1..n", color=PAL["teal"][0])
    s.arrow(gn, cm, "l", "r", label="CommitIndex → row", ta=0.3, tb=0.15, color=PAL["teal"][0],
            via=[(1120, 380), (1120, 270), (560, 270), (560, 340)])

    y = 820
    df = fields(s, 0, y, 520, "DiffFile", [("OldPath", "string"), ("NewPath", "string"), ("Status", "A M D R C"),
                                          ("Binary", "bool"), ("Hunks", "[]DiffHunk")], "purple", "one per changed file")
    dh = fields(s, 640, y, 420, "DiffHunk", [("Header", "@@ -10,4 +20,4 @@"), ("Lines", "[]DiffLine")], "purple")
    dl = fields(s, 1180, y, 520, "DiffLine", [("Kind", "Context Added Removed"), ("Text", "\"+new line\""),
                                             ("OldN", "int  0 = none"), ("NewN", "int  0 = none")], "purple",
                "line numbers come from the hunk header")
    s.arrow(df, dh, "r", "l", label="Hunks 0..n", color=PAL["purple"][0])
    s.arrow(dh, dl, "r", "l", label="Lines 1..n", color=PAL["purple"][0])

    y = 1110
    section(s, 0, y, "2 · Where the data comes from", "orange", 1700)
    s.code(0, y + 80, 820,
           "git log --all -z --format=%H%x01%P%x01%an%x01%ae%x01\n"
           "        %cn%x01%ce%x01%at%x01%s%x01%b\n\n"
           "record ⟶ split on NUL (\\x00)\n"
           "field  ⟶ split on SOH (\\x01)\n"
           "body is last, so multi-line messages are safe", title="log.go · ParseLog")
    s.code(880, y + 80, 820,
           "diff --git a/x.go b/x.go      → new DiffFile (Status M)\n"
           "new file / deleted / rename  → Status A / D / R\n"
           "Binary files … differ        → Binary = true\n"
           "@@ -10,4 +20,4 @@            → new DiffHunk, OldN=10 NewN=20\n"
           "+ / - / space                → DiffLine, counters advance\n"
           "\\ No newline at end of file  → no line number", title="diff.go · ParseDiff state machine")
    s.note(0, y + 330, 1700,
           "Refs are fetched once per load (git for-each-ref, tags dereferenced to their commit) into map[fullHash][]Ref and attached to\n"
           "each page of commits as it streams in. RefsFingerprint turns that map into a string, so a background fetch can tell cheaply\n"
           "whether anything moved before paying for a reload.", "info", title="Refs: fetched once, attached per page")

    y = 1560
    section(s, 0, y, "3 · A worked example", "orange", 1700, "One merge commit, as each layer sees it.")
    s.code(0, y + 80, 540, 'Commit{\n  Hash:    "d3f1…",\n  ShortHash: "d3f1a2c",\n  Parents: ["b07e…", "9c1a…"],\n  Author:  "Jane Doe",\n  Subject: "Merge branch \'feature\'",\n  Refs:    [{main, Local, IsHead}],\n}', title="model.Commit")
    s.code(580, y + 80, 540, "GraphNode{\n  CommitIndex: 0,\n  Lane:        0,\n  Parents:     [0, 1],\n  Color:       0,\n}\n// lane 1 opens for the merged branch", title="model.GraphNode")
    s.code(1160, y + 80, 540, "●          d3f1a2c  [HEAD -> main] Merge…\n╰ ╮\n│ ●        9c1a0de  feature work\n● │        b07e4f1  main work", title="what you see")
    return s
