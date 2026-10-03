#!/usr/bin/env python3
"""Builds every vault page (Excalidraw drawings) into LazyTreeObsidian/.

  python3 scripts/vault/build.py                     # write all pages
  python3 scripts/vault/build.py arch ui             # only these page keys
  python3 scripts/vault/build.py --preview DIR [..]  # also render PNG previews
                                                     # (Docker + playwright image)
Nothing is uploaded anywhere. See scripts/vault/README.md.
"""
import importlib, json, os, shutil, subprocess, sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.abspath(os.path.join(HERE, "..", ".."))
VAULT = os.path.join(ROOT, "LazyTreeObsidian")
sys.path.insert(0, HERE)
sys.path.insert(0, os.path.join(HERE, "pages"))

from common import PAGES  # noqa: E402

IMAGES = ["demo.gif", "split-dark.png", "split-light.png", "popup-dark.png",
          "search-dark.png", "help-dark.png", "zen-dark.png"]


def main(argv):
    preview = None
    keys = []
    it = iter(argv)
    for a in it:
        if a == "--preview":
            preview = next(it)
        else:
            keys.append(a)
    keys = keys or [k for k in PAGES]

    # screenshots used by the pages live inside the vault so Obsidian can embed them
    assets = os.path.join(VAULT, "assets")
    os.makedirs(assets, exist_ok=True)
    for im in IMAGES:
        src = os.path.join(ROOT, "docs", "img", im)
        if os.path.exists(src):
            shutil.copy(src, os.path.join(assets, im))

    built = {}
    for key in keys:
        mod = importlib.import_module(f"page_{key}")
        scene = mod.build()
        name = PAGES[key][0]
        folder = VAULT if key == "hub" else os.path.join(VAULT, "Overview")
        os.makedirs(folder, exist_ok=True)
        path = os.path.join(folder, name + ".md")
        with open(path, "w", encoding="utf-8") as f:
            f.write(scene.to_markdown())
        w, h = scene.extent()
        print(f"wrote {os.path.relpath(path, ROOT)}  ({len(scene.els)} elements, {int(w)}x{int(h)})")
        built[key] = scene

    if preview:
        pin, pout = os.path.join(preview, "in"), os.path.join(preview, "out")
        os.makedirs(pin, exist_ok=True)
        os.makedirs(pout, exist_ok=True)
        for k, sc in built.items():
            with open(os.path.join(pin, f"{k}.json"), "w") as f:
                json.dump(sc.scene_json(assets), f)
        subprocess.run([
            "docker", "run", "--rm", "-v", f"{pin}:/in", "-v", f"{pout}:/out",
            "-v", f"{os.path.join(HERE, 'preview_inner.py')}:/p.py",
            "mcr.microsoft.com/playwright/python:v1.49.1-noble", "bash", "-c",
            "pip install -q --break-system-packages playwright==1.49.1 >/dev/null 2>&1; python3 /p.py",
        ], check=True)


if __name__ == "__main__":
    main(sys.argv[1:])
