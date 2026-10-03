from common import *

ACT = [("You", "gray"), ("Update loop\n(AppModel)", "blue"), ("tea.Cmd\n(goroutine)", "purple"), ("git", "orange")]


def build():
    s = start("runtime", "lazytree follows the Elm architecture: messages in, a new model out. Here is every flow, step by step.")
    section(s, 0, 240, "1 · The loop in one picture", "teal", 1700)
    m = s.card(0, 320, 360, "Message", "a key press, a tick, a\nfinished git command…", "gray", "✉", min_h=130)
    u = s.card(470, 320, 360, "Update(msg)", "pure state change; returns\nthe new model + commands", "blue", "🧠", min_h=130)
    v = s.card(940, 320, 360, "View()", "turns the model into text;\nnever does I/O", "green", "🖼", min_h=130)
    c = s.card(470, 560, 360, "tea.Cmd", "runs off the UI thread\n(git, timers, clipboard)", "purple", "⚙", min_h=130)
    s.arrow(m, u, label="Bubble Tea delivers")
    s.arrow(u, v, label="render")
    s.arrow(u, c, "b", "t", label="returns")
    s.arrow(c, m, "l", "b", label="result becomes a message", via=[(180, 625)])
    s.note(1360, 320, 340, "If something feels slow, it\nmust be in a tea.Cmd, never\nin Update or View.", "warn", title="Golden rule")

    flows = [
        ("2 · Startup and streaming the history", [
            (0, 1, "run lazytree", "call"),
            (1, 2, "startReloadCmd(gen)", "call"),
            (2, 3, "git log --all -z (streaming)", "call"),
            (3, 2, "first 500 commits", "reply"),
            (2, 3, "git for-each-ref (once)", "call"),
            ("note", 2, "AttachRefs · Layouter.Append"),
            (2, 1, "MsgCommitsLoaded{Gen, Stream}", "reply"),
            ("note", 1, "draw the screen now"),
            (1, 2, "nextPageCmd(stream)", "call"),
            (2, 1, "MsgCommitsBatch  (repeat until Done)", "reply"),
        ]),
        ("3 · Moving the cursor loads the diff", [
            (0, 1, "j / k / click", "call"),
            ("note", 1, "debounceSeq++"),
            (1, 2, "tea.Tick 150 ms", "call"),
            (2, 1, "MsgDiffDebounce{Seq}", "reply"),
            ("note", 1, "stale Seq? drop · in LRU? show it"),
            (1, 2, "loadDiffCmd(hash)", "call"),
            (2, 3, "git show --patch <hash>", "call"),
            (3, 2, "unified diff text", "reply"),
            (2, 1, "MsgDiffLoaded → ParseDiff → cache", "reply"),
        ]),
    ]
    y = 760
    for title, msgs in flows:
        section(s, 0, y, title, "teal", 1700)
        bottom = sequence(s, 0, y + 80, ACT, msgs, col_w=420, row_h=58)
        y = bottom + 60

    flows2 = [
        ("4 · Background fetch (every 60 s)", [
            ("note", 1, "MsgFetchTick{Seq} → reschedule"),
            (1, 2, "fetchCmd (30 s timeout)", "call"),
            (2, 3, "git fetch --all --quiet", "call"),
            (2, 3, "git for-each-ref", "call"),
            ("note", 2, "same RefsFingerprint as before?"),
            (2, 1, "MsgFetchResult{Changed, Refs}", "reply"),
            ("note", 1, "changed → close stream, gen++, reload"),
        ]),
        ("5 · Git-side search (g: s: p:)", [
            (0, 1, "/ g:perf  enter", "call"),
            ("note", 1, "deepSeq++ · footer: searching…"),
            (1, 2, "deepSearchCmd", "call"),
            (2, 3, "git log --format=%H --grep=perf", "call"),
            (3, 2, "matching hashes", "reply"),
            (2, 1, "MsgDeepSearch{Seq, Hashes}", "reply"),
            ("note", 1, "mark rows · jump to first"),
        ]),
    ]
    for title, msgs in flows2:
        section(s, 0, y, title, "teal", 1700)
        bottom = sequence(s, 0, y + 80, ACT, msgs, col_w=420, row_h=58)
        y = bottom + 60

    section(s, 0, y, "6 · Guards against stale results", "teal", 1700,
            "Commands are never cancelled. Instead each chain carries a token; a message whose token is old is simply ignored.")
    rows = [["Token", "Bumped when", "Protects"],
            ["reloadGen", "a reload starts (a toggle, fetch found changes)", "MsgCommitsLoaded / Batch / MsgError from an older load"],
            ["debounceSeq", "the cursor moves", "MsgDiffDebounce for a commit you already left"],
            ["fetchSeq", "a fetch tick is scheduled", "duplicate fetch-tick chains"],
            ["deepSeq", "a git-side search starts or is cancelled (esc)", "MsgDeepSearch for an old query"],
            ["flashSeq", "a footer flash is shown", "an older flash clearing a newer one"],
            ["animSeq · tickSeq", "zen animation / countdown restarted", "double animation or countdown chains"]]
    s.table(0, y + 90, [260, 640, 800], rows, "teal", size=15, mono_cols=(0,))
    y += 90 + 7 * 40 + 40
    # timeline example
    s.text(0, y, "Example: you press  a  (toggle refs) while page 3 of the old load is still running", size=18, color=PAL["teal"][0], font=F_TITLE)
    s.line([(0, y + 90), (1600, y + 90)], color=MUTED, arrow=True)
    ev = [(60, "gen = 1\nload starts", "blue"), (380, "a pressed\ngen = 2, new load", "purple"),
          (760, "old page arrives\nGen 1 ≠ 2 → dropped", "red"), (1150, "new page arrives\nGen 2 → shown", "green")]
    for x, t, col in ev:
        s.ellipse(x - 9, y + 81, 18, 18, col)
        s.card(x - 110, y + 120, 260, "", t, col, min_h=80, title_size=4, body_size=15)
    s.note(0, y + 250, 1700, "Init() has a value receiver: changes made inside it are thrown away. That is why the first seq values are set in NewApp,\nand Init only builds commands from them.", "warn", title="A Bubble Tea gotcha this code works around")
    return s
