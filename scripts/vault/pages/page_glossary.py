from common import *

T = [
    ("Commit", "A saved snapshot of the project with a message, an author and one or more parents.", "orange"),
    ("Parent", "The commit(s) a commit was built on. A merge has two or more.", "orange"),
    ("Hash", "The commit's unique id (40 hex characters; lazytree shows 7).", "orange"),
    ("Branch", "A movable name pointing at a commit (refs/heads/*).", "green"),
    ("Remote branch", "Your copy of a branch on a server, e.g. origin/main (refs/remotes/*).", "blue"),
    ("Tag", "A fixed name for a commit, usually a release (refs/tags/*).", "purple"),
    ("HEAD", "The commit you have checked out; shown as HEAD -> branch.", "red"),
    ("Ref", "Any name for a commit: branch, remote branch, tag or HEAD.", "gray"),
    ("Topology", "The shape of history: how commits fork and merge.", "green"),
    ("Lane", "A column in the graph reserved for one line of history.", "green"),
    ("Connector row", "The extra ╰ ╮ row drawn when lanes fork or merge.", "green"),
    ("Worktree / submodule / bare", "Other repository layouts git supports; lazytree finds them all.", "gray"),
    ("Pickaxe (-S)", "git's search for commits that add or remove a piece of text.", "purple"),
    ("TUI", "Terminal User Interface: an app drawn with text in the terminal.", "blue"),
    ("Bubble Tea", "The Go TUI framework lazytree uses (Model · Update · View).", "blue"),
    ("Elm architecture", "State changes only in Update in reply to messages; View just draws.", "blue"),
    ("tea.Cmd", "A function Bubble Tea runs in the background; its result is a message.", "purple"),
    ("Lip Gloss", "Styling library: colours, borders, padding for terminal text.", "pink"),
    ("AdaptiveColor", "A colour with light and dark variants, picked from the terminal background.", "yellow"),
    ("Debounce", "Wait until input stops (150 ms) before doing expensive work.", "teal"),
    ("Seq / Gen token", "A counter attached to async work so stale results can be ignored.", "teal"),
    ("Streaming", "Reading git's output bit by bit instead of all at once.", "teal"),
    ("LRU cache", "Keeps the most recently used items (32 diffs), drops the oldest.", "teal"),
    ("OSC 52", "A terminal escape code that copies text to the clipboard, even over SSH.", "gray"),
    ("vhs", "A tool that records terminal sessions to GIF/PNG from a script (tape).", "pink"),
    ("goreleaser", "Builds release archives for every platform from one config.", "pink"),
]


def build():
    s = start("glossary", "Plain-language explanations of the git, terminal and project terms used across this vault.")
    for i, (t, d, col) in enumerate(T):
        x, y = (i % 3) * 575, 260 + (i // 3) * 140
        words, line, lines = d.split(), "", []
        for w in words:
            if len(line) + len(w) > 52:
                lines.append(line); line = w
            else:
                line = (line + " " + w).strip()
        lines.append(line)
        s.card(x, y, 550, t, "\n".join(lines), col, min_h=120)
    return s
