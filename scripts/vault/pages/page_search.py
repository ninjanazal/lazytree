from common import *


def build():
    s = start("search", "Two kinds of search, how matches are shown and visited, and every way to jump around the graph.")
    section(s, 0, 240, "1 · What happens when you press /", "purple", 1700)
    a = s.card(0, 320, 300, "/ pressed", "remember cursor position\n(searchOrigin), open box", "purple", "⌨", min_h=120)
    d = s.diamond(400, 320, 260, 140, "yellow", strong=True)
    s.text(440, 366, "starts with\ng:  s:  or  p: ?", size=16)
    im = s.card(760, 250, 440, "In-memory search (as you type)", "every keystroke rescans loaded commits:\nsubject · body · hash · author · ref names\njump to first match at/after the origin", "blue", "⚡", min_h=150)
    dp = s.card(760, 450, 440, "Git-side search (on enter)", "git log --grep / -S / -- path\nover the WHOLE history, async\nfooter shows “searching…”", "orange", "🧰", min_h=150)
    res = s.card(1300, 330, 400, "Matches", "rows get a ▌ marker and\nhighlighted text; n / N cycle;\nfooter shows /query 2/7", "green", "✅", min_h=150)
    s.arrow(a, Box(d.id, d.x, d.y, d.w, d.h), "r", "l")
    s.arrow(Box(d.id, d.x, d.y, d.w, d.h), im, "t", "l", label="no", via=[(530, 325)])
    s.arrow(Box(d.id, d.x, d.y, d.w, d.h), dp, "b", "l", label="yes", via=[(530, 525)])
    s.arrow(im, res, "r", "l", ta=0.5, tb=0.3)
    s.arrow(dp, res, "r", "l", ta=0.5, tb=0.75)

    section(s, 0, 680, "2 · The three git prefixes", "purple", 1700)
    rows = [["Type", "Finds", "Runs", "Good for"],
            ["g:text", "commits whose message contains text (case-insensitive)", "git log --grep=text -i -F", "“which commit mentions the bug id?”"],
            ["s:text", "commits that added or removed text in the code", "git log -Stext", "“when did this function appear?”"],
            ["p:path", "commits that touched a file or folder", "git log -- path", "“history of internal/ui”"]]
    s.table(0, 760, [140, 560, 380, 620], rows, "purple", size=16, mono_cols=(0, 2))
    s.note(0, 960, 830, "Git-side results are stored as a set of hashes, so commits that\nstream in later are matched too. esc bumps deepSeq, so a late\nresult for a cancelled search is ignored.", "info", title="Works while history is still loading")
    s.note(870, 960, 830, "In-memory search is ~20 ms over 108k commits, so it can run on\nevery keystroke. Git-side search can take seconds on big repos,\nso it waits for enter.", "tip", title="Why two kinds")

    section(s, 0, 1160, "3 · Keys while searching", "purple", 1700)
    keys = [("/", "start; type to search live"), ("enter", "keep the search (git prefixes run now)"),
            ("esc  (in the box)", "cancel: clear and jump back to where you were"), ("n / N", "next / previous match, wraps around"),
            ("esc  (in the log)", "clear highlights and any running git search")]
    for i, (k, d2) in enumerate(keys):
        b = s.chip(0 + (i % 2) * 850, 1240 + (i // 2) * 60, k, "purple")
        s.text(b.right + 14, 1246 + (i // 2) * 60, d2, size=17)

    section(s, 0, 1460, "4 · Jumping around", "purple", 1700, "None of these change the repository; they only move the cursor.")
    jumps = [("H", "Jump to HEAD", "finds the commit carrying the HEAD ref", "blue"),
             ("p", "Parent", "first parent (straight down the main line);\nworks inside the diff too", "green"),
             ("c", "Child", "nearest commit above that lists this one\nas a parent", "teal"),
             ("g / G", "First / last", "top or bottom of the loaded history", "gray"),
             ("click", "Select", "the row under the pointer (connector\nrows belong to the commit above)", "orange"),
             ("wheel", "Scroll", "3 rows per notch; scrolls the diff\nwhen the pointer is over it", "yellow")]
    for i, (k, t, d2, col) in enumerate(jumps):
        x, yy = (i % 3) * 575, 1560 + (i // 3) * 170
        s.card(x, yy, 550, f"{k}  ·  {t}", d2, col, min_h=145)
    s.note(0, 1910, 1700, "Merges have several parents: p always follows the first one. To walk the merged-in branch, move onto it with j/k or search for it,\nthen use p from there.", "idea", title="Following a side branch")
    return s
