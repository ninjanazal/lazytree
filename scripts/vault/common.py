"""Helpers shared by the vault pages."""
from exlib import *

PAGES = {  # key -> (file name without .md, hub label, icon, color)
    "hub":      ("BaseOverview", "Hub", "🏠", "gray"),
    "vision":   ("01 Idea & Vision", "Idea & Vision", "💡", "yellow"),
    "arch":     ("02 Architecture", "Architecture", "🏗", "blue"),
    "runtime":  ("03 Runtime & Message Flow", "Runtime & Messages", "🔁", "teal"),
    "design":   ("04 Design Choices", "Design Choices", "🧭", "purple"),
    "risks":    ("05 Missing & Risks", "Status & Risks", "🚧", "red"),
    "roadmap":  ("06 Roadmap", "Roadmap", "🗺", "green"),
    "model":    ("07 Data Model", "Data Model", "🧬", "orange"),
    "graph":    ("08 Graph Algorithm", "Graph Algorithm", "🌿", "green"),
    "ui":       ("09 UI Guide", "UI Guide", "🖥", "blue"),
    "search":   ("10 Search & Navigation", "Search & Navigation", "🔎", "purple"),
    "config":   ("11 Configuration", "Configuration", "⚙", "teal"),
    "testing":  ("12 Testing & Quality", "Testing & Quality", "🧪", "yellow"),
    "release":  ("13 Release & Tooling", "Release & Tooling", "📦", "pink"),
    "glossary": ("14 Glossary", "Glossary", "📖", "gray"),
}
# the order a newcomer should read in
READING = ["vision", "ui", "arch", "model", "runtime", "graph", "search", "design",
           "config", "testing", "release", "roadmap", "risks", "glossary"]

W = 1700  # standard page width


def link(key):
    return f"[[{PAGES[key][0]}]]"


def start(key, subtitle, width=W):
    """Create a scene with the standard header and navigation row."""
    name, label, icon, color = PAGES[key]
    s = Scene(name, "p" + (key[:2] if key != "hub" else "hb"))
    s.header(label if key != "hub" else "lazytree · Project Map", subtitle, icon, color, width=width)
    y = 170
    items = [("🏠 Hub", "BaseOverview", "gray")]
    if key != "hub":
        i = READING.index(key)
        if i > 0:
            pk = READING[i - 1]
            items.append((f"← {PAGES[pk][1]}", PAGES[pk][0], PAGES[pk][3]))
        if i < len(READING) - 1:
            nk = READING[i + 1]
            items.append((f"{PAGES[nk][1]} →", PAGES[nk][0], PAGES[nk][3]))
    s.nav(0, y, items)
    return s


def section(s, x, y, title, color="blue", width=None, sub=None):
    st, soft, hard = PAL[color]
    t = s.text(x, y, title, size=30, color=st, font=F_TITLE)
    ln = s.line([(x, y + 42), (x + (width or t.w + 20), y + 42)], color=st, sw=3)
    if sub:
        s.text(x, y + 54, sub, size=16, color=MUTED)
    return t


def sequence(s, x, y, actors, msgs, col_w=250, row_h=62, notes=None, head_h=48):
    """Sequence/swimlane diagram.

    actors: [(label, color)]
    msgs:   [(from_idx, to_idx, label, kind)] kind in {"call","reply","async","self"}
            or ("note", idx, text) for a note box under an actor.
    Returns the bottom y.
    """
    heads = []
    for i, (label, color) in enumerate(actors):
        hx = x + i * col_w
        w = col_w - 30
        r = s.rect(hx, y, w, head_h, color, strong=True)
        tw, th = text_size(label, 16)
        t = s.text(hx + (w - tw) / 2, y + (head_h - th) / 2, label, size=16, color=PAL[color][0])
        s.group(r, t)
        heads.append((hx + w / 2, color))
    total = len(msgs) * row_h + 30
    for cx, color in heads:
        s.line([(cx, y + head_h), (cx, y + head_h + total)], color=PAL[color][0], sw=1.5, dashed=True, opacity=70)
    cy = y + head_h + row_h * 0.7
    for m in msgs:
        if m[0] == "note":
            _, idx, text = m
            tw, th = text_size(text, 14)
            bx = heads[idx][0] - tw / 2 - 12
            r = s.rect(bx, cy - th / 2 - 8, tw + 24, th + 16, "yellow", sw=1.5)
            t = s.text(bx + 12, cy - th / 2, text, size=14)
            s.group(r, t)
            cy += max(row_h, th + 28)
            continue
        a, b, label, kind = m
        xa, xb = heads[a][0], heads[b][0]
        color = PAL[actors[a][1]][0]
        if kind == "self":
            pts = [(xa, cy - 10), (xa + 46, cy - 10), (xa + 46, cy + 12), (xa + 4, cy + 12)]
            s.line(pts, color=color, sw=2, arrow=True)
            s.text(xa + 56, cy - 12, label, size=14, color=INK)
        else:
            dashed = kind == "reply"
            s.line([(xa + (6 if xb > xa else -6), cy), (xb - (6 if xb > xa else -6), cy)], color=color, sw=2,
                   dashed=dashed, arrow=True)
            lw, lh = text_size(label, 14)
            lx = min(xa, xb) + abs(xb - xa) / 2 - lw / 2
            s.text(lx, cy - lh - 6, label, size=14, color=INK)
        cy += row_h
    return y + head_h + total
