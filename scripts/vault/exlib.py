"""A small builder for Obsidian-Excalidraw pages.

The vault's pages are Excalidraw drawings (a project rule), written here as
uncompressed Excalidraw JSON inside the Obsidian-Excalidraw markdown wrapper.
Pages are generated from Python so they stay consistent and are easy to update
(`make vault`); see scripts/vault/README.md.

Design notes
- Text is always a *free* text element with explicit newlines (no bound text),
  so layout is deterministic and the plugin never re-wraps it.
- Widths are estimated from per-font average glyph widths; keep a little slack.
- Everything is drawn clean (roughness 0) on a white canvas, whatever the
  Obsidian theme, so the pages look the same for everyone.
"""
from __future__ import annotations

import json
import random
import re
from dataclasses import dataclass, field

# ---- palette ---------------------------------------------------------------
# name: (stroke, soft fill, strong fill)
PAL = {
    "blue":   ("#1971c2", "#e7f5ff", "#a5d8ff"),
    "green":  ("#2f9e44", "#ebfbee", "#b2f2bb"),
    "orange": ("#e8590c", "#fff4e6", "#ffd8a8"),
    "red":    ("#e03131", "#fff5f5", "#ffc9c9"),
    "purple": ("#9c36b5", "#f8f0fc", "#eebefa"),
    "teal":   ("#0c8599", "#e3fafc", "#99e9f2"),
    "yellow": ("#f08c00", "#fff9db", "#ffec99"),
    "pink":   ("#c2255c", "#fff0f6", "#fcc2d7"),
    "gray":   ("#495057", "#f8f9fa", "#dee2e6"),
    "ink":    ("#1e1e1e", "#ffffff", "#e9ecef"),
}
INK = "#1e1e1e"
MUTED = "#6c757d"

# font families (Excalidraw ids)
F_HAND, F_NUNITO, F_MONO, F_TITLE = 5, 6, 3, 7
LINE_H = {5: 1.25, 6: 1.35, 3: 1.2, 7: 1.15, 1: 1.25, 2: 1.15}
GLYPH = {5: 0.52, 6: 0.55, 3: 0.60, 7: 0.56, 1: 0.55, 2: 0.55}


def text_size(text: str, size: int, font: int = F_NUNITO) -> tuple[float, float]:
    lines = text.split("\n")
    longest = max((_visual_len(l) for l in lines), default=0)
    return longest * size * GLYPH[font], len(lines) * size * LINE_H[font]


def _visual_len(line: str) -> float:
    # wide glyphs (emoji, CJK-ish symbols) count double
    n = 0.0
    for ch in line:
        n += 1.7 if ord(ch) > 0x2600 and ord(ch) not in range(0x2500, 0x2600) else 1.0
    return n


def _index(n: int) -> str:
    digits = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
    if n < 62:
        return "a" + digits[n]
    n -= 62
    if n < 62 * 62:
        return "b" + digits[n // 62] + digits[n % 62]
    n -= 62 * 62
    return "c" + digits[n // 3844] + digits[(n // 62) % 62] + digits[n % 62]


@dataclass
class Box:
    """Handle returned by shape helpers, for connecting arrows."""
    id: str
    x: float
    y: float
    w: float
    h: float

    @property
    def cx(self): return self.x + self.w / 2
    @property
    def cy(self): return self.y + self.h / 2
    @property
    def right(self): return self.x + self.w
    @property
    def bottom(self): return self.y + self.h

    def anchor(self, side: str, t: float = 0.5):
        if side == "r": return (self.right, self.y + self.h * t)
        if side == "l": return (self.x, self.y + self.h * t)
        if side == "t": return (self.x + self.w * t, self.y)
        if side == "b": return (self.x + self.w * t, self.bottom)
        raise ValueError(side)


class Scene:
    def __init__(self, name: str, prefix: str):
        self.name = name
        self.prefix = prefix
        self.els: list[dict] = []
        self.files: dict[str, str] = {}      # fileId -> vault file name
        self._n = 0
        self._rng = random.Random(name)
        self._group = 0

    # -- plumbing ----------------------------------------------------------
    def _id(self) -> str:
        self._n += 1
        return f"{self.prefix}{self._n:04d}"

    def _base(self, typ, x, y, w, h, **kw):
        stroke = kw.pop("stroke", INK)
        fill = kw.pop("fill", "transparent")
        e = {
            "id": self._id(), "type": typ,
            "x": round(x, 2), "y": round(y, 2), "width": round(w, 2), "height": round(h, 2),
            "angle": 0, "strokeColor": stroke, "backgroundColor": fill,
            "fillStyle": kw.pop("fillStyle", "solid"),
            "strokeWidth": kw.pop("sw", 2), "strokeStyle": kw.pop("style", "solid"),
            "roughness": 0, "opacity": kw.pop("opacity", 100),
            "groupIds": kw.pop("groups", []), "frameId": None,
            "roundness": kw.pop("roundness", None),
            "seed": self._rng.randrange(1, 2**31), "version": 2,
            "versionNonce": self._rng.randrange(1, 2**31),
            "isDeleted": False, "boundElements": [], "updated": 1790000000000,
            "link": kw.pop("link", None), "locked": False,
            "index": _index(len(self.els)),
            "hasTextLink": False, "created": None,
        }
        e.update(kw)
        self.els.append(e)
        return e

    def group(self, *items) -> str:
        """Group Box handles / ids so they move together."""
        self._group += 1
        gid = f"{self.prefix}g{self._group}"
        ids = {i.id if isinstance(i, Box) else i for i in items}
        for e in self.els:
            if e["id"] in ids:
                e["groupIds"] = [gid] + e["groupIds"]
        return gid

    # -- primitives --------------------------------------------------------
    def text(self, x, y, text, size=18, color=INK, font=F_NUNITO, align="left",
             link=None, w=None, opacity=100) -> Box:
        tw, th = text_size(text, size, font)
        if w is None:
            w = tw
        # for centered/right text, x is the left edge of a box of width w
        px = x
        if align == "center":
            px = x + (w - tw) / 2
        elif align == "right":
            px = x + w - tw
        e = self._base("text", px, y, tw, th, stroke=color, link=link, opacity=opacity)
        e.update({
            "text": text, "originalText": text, "fontSize": size, "fontFamily": font,
            "textAlign": "left", "verticalAlign": "top", "containerId": None,
            "lineHeight": LINE_H[font], "autoResize": True, "baseline": round(size * 0.9),
            "rawText": "", "labelPosition": None, "baseFontSize": None,
        })
        return Box(e["id"], px, y, tw, th)

    def rect(self, x, y, w, h, color="gray", strong=False, fill=None, stroke=None,
             radius=True, dashed=False, sw=2, link=None, opacity=100) -> Box:
        s, soft, hard = PAL[color]
        e = self._base("rectangle", x, y, w, h, stroke=stroke or s,
                       fill=fill if fill is not None else (hard if strong else soft),
                       style="dashed" if dashed else "solid", sw=sw, link=link,
                       roundness={"type": 3} if radius else None, opacity=opacity)
        return Box(e["id"], x, y, w, h)

    def ellipse(self, x, y, w, h, color="gray", strong=True, fill=None, stroke=None, sw=2) -> Box:
        s, soft, hard = PAL[color]
        e = self._base("ellipse", x, y, w, h, stroke=stroke or s,
                       fill=fill if fill is not None else (hard if strong else soft), sw=sw,
                       roundness={"type": 2})
        return Box(e["id"], x, y, w, h)

    def diamond(self, x, y, w, h, color="gray", strong=False, sw=2) -> Box:
        s, soft, hard = PAL[color]
        e = self._base("diamond", x, y, w, h, stroke=s, fill=hard if strong else soft, sw=sw,
                       roundness={"type": 2})
        return Box(e["id"], x, y, w, h)

    def line(self, pts, color=INK, sw=2, dashed=False, arrow=False, start_arrow=False,
             opacity=100) -> Box:
        """Polyline / arrow through absolute points."""
        x0, y0 = pts[0]
        rel = [[round(px - x0, 2), round(py - y0, 2)] for px, py in pts]
        xs = [p[0] for p in pts]; ys = [p[1] for p in pts]
        e = self._base("arrow" if arrow or start_arrow else "line", x0, y0,
                       max(xs) - min(xs), max(ys) - min(ys), stroke=color, sw=sw,
                       style="dashed" if dashed else "solid", opacity=opacity,
                       roundness={"type": 2} if len(pts) == 2 else None)
        e.update({
            "points": rel, "lastCommittedPoint": None,
            "startBinding": None, "endBinding": None,
            "startArrowhead": "arrow" if start_arrow else None,
            "endArrowhead": "arrow" if arrow else None,
            "elbowed": False,
        })
        return Box(e["id"], min(xs), min(ys), max(xs) - min(xs), max(ys) - min(ys))

    def image(self, x, y, w, h, vault_file: str, link=None) -> Box:
        fid = f"{abs(hash(vault_file)) % 10**12:012d}{self._n:04d}".ljust(40, "0")
        self.files[fid] = vault_file
        e = self._base("image", x, y, w, h, link=link)
        e.update({"fileId": fid, "status": "saved", "scale": [1, 1], "crop": None})
        return Box(e["id"], x, y, w, h)

    # -- composites --------------------------------------------------------
    def arrow(self, a: Box, b: Box, sa="r", sb="l", color=INK, label=None, dashed=False,
              sw=2, via=None, ta=0.5, tb=0.5, label_color=None, both=False,
              label_pos=0.5, label_dy=0) -> Box:
        p0 = a.anchor(sa, ta)
        p1 = b.anchor(sb, tb)
        pts = [p0] + (via or []) + [p1]
        ar = self.line(pts, color=color, sw=sw, dashed=dashed, arrow=True, start_arrow=both)
        if label:
            # label sits on the longest segment's midpoint
            segs = list(zip(pts, pts[1:]))
            i = max(range(len(segs)), key=lambda k: abs(segs[k][1][0] - segs[k][0][0]) + abs(segs[k][1][1] - segs[k][0][1]))
            (x0, y0), (x1, y1) = segs[i]
            mx = x0 + (x1 - x0) * label_pos
            my = y0 + (y1 - y0) * label_pos + label_dy
            self.pill(mx, my, label, color=label_color or color)
        return ar

    def pill(self, cx, cy, label, color=INK, size=14, bg="#ffffff"):
        tw, th = text_size(label, size, F_NUNITO)
        w, h = tw + 16, th + 8
        r = self._base("rectangle", cx - w / 2, cy - h / 2, w, h, stroke="transparent", fill=bg,
                       roundness={"type": 3}, sw=1)
        self.text(cx - tw / 2, cy - th / 2, label, size=size, color=color)
        return Box(r["id"], cx - w / 2, cy - h / 2, w, h)

    def chip(self, x, y, label, color="blue", size=15, strong=True, link=None, pad=12) -> Box:
        tw, th = text_size(label, size, F_NUNITO)
        w, h = tw + pad * 2, th + 10
        b = self.rect(x, y, w, h, color, strong=strong, link=link, radius=True, sw=1.5)
        t = self.text(x + pad, y + 5, label, size=size, color=PAL[color][0], link=link)
        self.group(b, t)
        return b

    def card(self, x, y, w, title, body="", color="blue", icon="", body_size=15,
             title_size=20, link=None, min_h=0, strong_head=True, mono=False, h=None) -> Box:
        """Titled card: tinted body, colored header strip, wrapped-by-hand body."""
        s, soft, hard = PAL[color]
        head_h = title_size * 1.35 + 18
        bfont = F_MONO if mono else F_NUNITO
        _, body_h = text_size(body, body_size, bfont) if body else (0, 0)
        total = max(min_h, head_h + (body_h + 24 if body else 6)) if h is None else h
        outer = self.rect(x, y, w, total, color, link=link)
        head = self.rect(x, y, w, head_h, color, strong=True, link=link)
        # square off the header's lower corners with a plain strip
        strip = self._base("rectangle", x + 1, y + head_h - 10, w - 2, 11, stroke="transparent",
                           fill=hard)
        t = self.text(x + 14, y + 9, f"{icon} {title}".strip(), size=title_size, color=s, font=F_NUNITO, link=link)
        items = [outer, head, Box(strip["id"], x, y, 0, 0), t]
        if body:
            b = self.text(x + 14, y + head_h + 12, body, size=body_size, color=INK, font=bfont)
            items.append(b)
        self.group(*items)
        return Box(outer.id, x, y, w, total)

    def note(self, x, y, w, text, kind="tip", size=15, title=None) -> Box:
        meta = {"tip": ("green", "💡"), "warn": ("orange", "⚠"), "info": ("blue", "ℹ"),
                "danger": ("red", "⛔"), "idea": ("purple", "✨")}[kind]
        color, icon = meta
        s, soft, _ = PAL[color]
        _, th = text_size(text, size)
        tt = size * 1.5 if title else 0
        h = th + 24 + tt
        r = self.rect(x, y, w, h, color, dashed=False)
        bar = self._base("rectangle", x, y, 8, h, stroke="transparent", fill=s)
        items = [r, Box(bar["id"], x, y, 8, h)]
        if title:
            items.append(self.text(x + 50, y + 10, title, size=size + 2, color=s, font=F_TITLE))
        items.append(self.text(x + 50, y + 12 + tt, text, size=size))
        items.append(self.text(x + 16, y + h / 2 - 14, icon, size=24, color=s))
        self.group(*items)
        return Box(r.id, x, y, w, h)

    def code(self, x, y, w, text, size=14, title=None) -> Box:
        _, th = text_size(text, size, F_MONO)
        top = 30 if title else 0
        h = th + 28 + top
        r = self.rect(x, y, w, h, "ink", fill="#1e1e2e", stroke="#1e1e2e")
        items = [r]
        if title:
            items.append(self.text(x + 16, y + 8, title, size=13, color="#9399b2", font=F_MONO))
            for i, c in enumerate(("#f38ba8", "#f9e2af", "#a6e3a1")):
                items.append(self.ellipse(x + w - 24 - i * 18, y + 10, 10, 10, fill=c, stroke=c))
        items.append(self.text(x + 16, y + 14 + top, text, size=size, color="#cdd6f4", font=F_MONO))
        self.group(*items)
        return Box(r.id, x, y, w, h)

    def step(self, cx, cy, n, color="blue", r=18) -> Box:
        e = self.ellipse(cx - r, cy - r, 2 * r, 2 * r, color, strong=True)
        label = str(n)
        tw, th = text_size(label, 18, F_NUNITO)
        t = self.text(cx - tw / 2, cy - th / 2, label, size=18, color=PAL[color][0])
        self.group(e, t)
        return e

    def table(self, x, y, col_w, rows, head_color="blue", size=15, row_h=None,
              zebra=True, mono_cols=()) -> Box:
        """rows[0] is the header. Cells may contain newlines; row height follows content."""
        total_w = sum(col_w)
        cy = y
        first = True
        for ri, row in enumerate(rows):
            heights = [text_size(c, size, F_MONO if ci in mono_cols and ri else F_NUNITO)[1] for ci, c in enumerate(row)]
            rh = max(row_h or 0, max(heights) + 16)
            s, soft, hard = PAL[head_color]
            if ri == 0:
                fill, stroke = hard, s
            else:
                fill = "#ffffff" if (ri % 2) or not zebra else "#f8f9fa"
                stroke = "#ced4da"
            cx = x
            for ci, (cell, cw) in enumerate(zip(row, col_w)):
                self.rect(cx, cy, cw, rh, "gray", fill=fill, stroke=stroke, radius=False, sw=1)
                font = F_MONO if (ci in mono_cols and ri) else F_NUNITO
                self.text(cx + 10, cy + 8, cell, size=size, color=(s if ri == 0 else INK), font=font)
                cx += cw
            cy += rh
        return Box("", x, y, total_w, cy - y)

    def header(self, title, subtitle, icon, color="blue", width=1500, x=0, y=0, links=None) -> Box:
        s, soft, hard = PAL[color]
        band = self.rect(x, y, width, 150, color, strong=True)
        ic = self.text(x + 34, y + 30, icon, size=64, color=s)
        t = self.text(x + 130, y + 24, title, size=52, color=s, font=F_TITLE)
        st = self.text(x + 132, y + 92, subtitle, size=20, color=INK)
        self.group(band, ic, t, st)
        return Box(band.id, x, y, width, 150)

    def nav(self, x, y, items):
        """items: [(label, page or None, color)] rendered as linked chips."""
        cx = x
        for label, page, color in items:
            c = self.chip(cx, y, label, color, strong=False, link=f"[[{page}]]" if page else None, size=15)
            cx += c.w + 10

    # -- output ------------------------------------------------------------
    def to_markdown(self) -> str:
        texts, links = [], []
        for e in self.els:
            if e["type"] == "text":
                texts.append(f"{e['text']} ^{e['id']}")
            if e.get("link"):
                links.append(f"{e['id']}: {e['link']}")
        scene = {
            "type": "excalidraw", "version": 2,
            "source": "https://github.com/zsviczian/obsidian-excalidraw-plugin/releases/tag/2.28.1",
            "elements": self.els,
            "appState": {"theme": "light", "viewBackgroundColor": "#ffffff",
                         "gridSize": None, "gridStep": 5, "gridModeEnabled": False},
            "files": {},
        }
        out = ["---", "", "excalidraw-plugin: parsed", "tags: [excalidraw, lazytree]", "", "---",
               "==⚠  Switch to EXCALIDRAW VIEW in the MORE OPTIONS menu of this document. ⚠== You can decompress Drawing data with the command palette: 'Decompress current Excalidraw file'. For more info check in plugin settings under 'Saving'",
               "", "", "# Excalidraw Data", "", "## Text Elements"]
        out.append("\n\n".join(texts))
        out.append("")
        if links:
            out += ["## Element Links", "\n\n".join(links), ""]
        if self.files:
            out += ["## Embedded Files", "\n\n".join(f"{fid}: [[{name}]]" for fid, name in self.files.items()), ""]
        out += ["%%", "## Drawing", "```json", json.dumps(scene, ensure_ascii=False), "```", "%%", ""]
        return "\n".join(out)

    def scene_json(self, asset_dir=None) -> dict:
        """The scene for previewing; images are inlined as data URLs."""
        import base64, mimetypes, os
        files = {}
        for fid, name in self.files.items():
            path = os.path.join(asset_dir or ".", name)
            mime = mimetypes.guess_type(path)[0] or "image/png"
            with open(path, "rb") as f:
                files[fid] = {"id": fid, "mimeType": mime, "created": 1790000000000,
                              "dataURL": f"data:{mime};base64," + base64.b64encode(f.read()).decode()}
        return {"type": "excalidraw", "version": 2, "elements": self.els, "files": files,
                "appState": {"theme": "light", "viewBackgroundColor": "#ffffff"}}

    def extent(self):
        xs, ys = [0], [0]
        for e in self.els:
            xs.append(e["x"] + e["width"]); ys.append(e["y"] + e["height"])
        return max(xs), max(ys)
