"""Runs INSIDE the playwright container: renders Excalidraw scene JSON files to PNG.
   /in/<name>.json  ->  /out/<name>.png
"""
import glob, json, os, sys
from playwright.sync_api import sync_playwright

HTML = """<!doctype html><html><body style="margin:0;background:#fff">
<div id="out"></div>
<script>window.EXCALIDRAW_ASSET_PATH = "https://esm.sh/@excalidraw/excalidraw@0.18.0/dist/prod/";</script>
<script type="module">
import { exportToSvg } from "https://esm.sh/@excalidraw/excalidraw@0.18.0?deps=react@18.3.1,react-dom@18.3.1";
window.render = async (scene) => {
  const svg = await exportToSvg({
    elements: scene.elements,
    appState: { ...scene.appState, exportBackground: true, exportWithDarkMode: false },
    files: scene.files || {},
    exportPadding: 30,
  });
  const out = document.getElementById("out");
  out.innerHTML = "";
  out.appendChild(svg);
  return [svg.getAttribute("width"), svg.getAttribute("height")];
};
window.ready = true;
</script></body></html>"""

scale = float(os.environ.get("SCALE", "1"))
with sync_playwright() as p:
    b = p.chromium.launch()
    for path in sorted(glob.glob("/in/*.json")):
        name = os.path.basename(path)[:-5]
        scene = json.load(open(path))
        page = b.new_page(device_scale_factor=scale)
        page.on("console", lambda m: print("console:", m.text, file=sys.stderr) if m.type in ("error",) else None)
        page.set_content(HTML)
        page.wait_for_function("window.ready === true", timeout=120000)
        w, h = page.evaluate("(s) => window.render(s)", scene)
        w, h = int(float(w)), int(float(h))
        page.set_viewport_size({"width": w, "height": h})
        page.wait_for_timeout(1500)  # fonts
        page.screenshot(path=f"/out/{name}.png", full_page=True)
        print(f"rendered {name}: {w}x{h}")
        page.close()
    b.close()
