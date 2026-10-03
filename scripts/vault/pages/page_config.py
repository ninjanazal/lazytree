from common import *

PALETTE = [("selected", "#d7d7ff", "#444444"), ("hash", "#d75f00", "#ffaf5f"), ("author", "#00875f", "#87d7af"),
           ("date", "#626262", "#8a8a8a"), ("help", "#626262", "#8a8a8a"), ("border", "#bcbcbc", "#3a3a3a"),
           ("accent", "#87005f", "#af5fff"), ("title", "#87005f", "#af5fff"), ("error", "#d70000", "#ff5f5f"),
           ("head", "#d7875f", "#ffaf5f"), ("branch", "#87d787", "#87d787"), ("remote", "#87afff", "#87afff"),
           ("tag", "#d7afff", "#d7afff"), ("added", "#008700", "#87d787"), ("removed", "#d70000", "#ff5f5f"),
           ("diff_file", "#005faf", "#87afff"), ("diff_hunk", "#626262", "#8a8a8a")]


def build():
    s = start("config", "Everything is optional. One TOML file changes timing, layout, refs, keys and colours. Typos stop lazytree with a clear message.")
    section(s, 0, 240, "1 · Where the file lives", "teal", 1700)
    s.code(0, 320, 820, "$XDG_CONFIG_HOME/lazytree/config.toml\n# or, when XDG_CONFIG_HOME is unset:\n~/.config/lazytree/config.toml", title="path")
    s.note(880, 320, 820, "No file → built-in defaults. A present file only needs the keys\nyou want to change; everything else keeps its default.", "info", title="Missing is fine")
    section(s, 0, 480, "2 · A complete example, explained", "teal", 1700)
    s.code(0, 560, 760,
           'fetch_interval = "60s"\nshow_all = true\ntheme = "auto"\nlayout = "auto"\n\nhide_tags = false\nhide_remotes = false\nhide_refs = ["origin/dependabot/*"]\nlane_colors = ["#5f87ff", "208"]\n\n[keys]\nzen = ["x"]\nquit = ["q", "Q"]\n\n[colors]\nhash = "#ff8700"\nadded = "34"',
           title="config.toml", size=16)
    exp = [("background git fetch period; \"off\" disables it, minimum 5 s", 0),
           ("start with all refs (true) or the current branch (false)", 1),
           ("auto = detect terminal background; or force light / dark", 2),
           ("auto = split view at ≥ 140 cols; or split / popup", 3),
           ("start with tag labels hidden (t toggles)", 5),
           ("start with remote-branch labels hidden (r toggles)", 6),
           ("ref names to always hide; * matches any characters", 7),
           ("replace the graph lane colours (#rrggbb or 0-255)", 8),
           ("rebind actions; one key per action, ctrl+c always quits", 11),
           ("override named colours (see swatches below)", 15)]
    for t, line in exp:
        yy = 560 + 44 + line * 19.2 + 2
        s.line([(700, yy + 8), (800, yy + 8)], color=PAL["teal"][0], sw=1.5, arrow=True)
        s.text(810, yy - 2, t, size=15)
    section(s, 0, 1060, "3 · How it is applied at startup", "teal", 1700)
    st = [("config.toml", "gray"), ("config.Load\nparse + validate", "teal"), ("ui.ApplyKeys\nconflict check", "purple"),
          ("ui.ApplyColors\nrebuild styles", "pink"), ("theme →\nlipgloss", "yellow"), ("ui.Options →\nNewAppWithOptions", "blue")]
    prev = None
    for i, (t, col) in enumerate(st):
        b = s.card(i * 285, 1140, 250, t.split("\n")[0], "\n".join(t.split("\n")[1:]), col, min_h=100, title_size=17)
        if prev: s.arrow(prev, b)
        prev = b
    s.note(0, 1270, 1700, "lazytree: config ~/.config/lazytree/config.toml: unknown key \"fetch_intervall\" (valid: fetch_interval, show_all, …)\nlazytree: config [keys]: key \"q\" is bound to both \"quit\" and \"zen\"", "danger", title="Errors look like this (and lazytree exits before drawing)")
    section(s, 0, 1420, "4 · Key action names", "teal", 1700)
    acts = ["up", "down", "page_up", "page_down", "top", "bottom", "enter", "back", "search", "next_match", "prev_match", "parent",
            "child", "head", "copy", "edit", "toggle_refs", "toggle_tags", "toggle_remotes", "split", "zen", "help", "quit"]
    for i, a in enumerate(acts):
        s.chip((i % 8) * 212, 1500 + (i // 8) * 52, a, "purple", strong=False)
    section(s, 0, 1690, "5 · Colour names (default light / dark)", "teal", 1700, "Left half = light terminals, right half = dark. Your override applies to both.")
    for i, (n, l, d) in enumerate(PALETTE):
        x, y = (i % 6) * 283, 1780 + (i // 6) * 100
        s.rect(x, y, 60, 60, "gray", fill=l, stroke="#adb5bd", radius=False, sw=1)
        s.rect(x + 60, y, 60, 60, "gray", fill=d, stroke="#adb5bd", radius=False, sw=1)
        s.text(x + 132, y + 16, n, size=17, font=F_MONO)
    return s
