# Vault generator

The Obsidian vault (`LazyTreeObsidian/`) is made of Excalidraw drawings that are
**generated** from these Python files, so edit here, not in Obsidian.

- `exlib.py`: tiny Excalidraw builder (text, cards, notes, tables, code blocks, arrows, images)
- `common.py`: page list, reading order, header/nav, sequence diagrams
- `pages/page_<key>.py`: one file per page, `build()` returns a `Scene`
- `build.py`: writes the pages (and copies `docs/img` screenshots into `LazyTreeObsidian/assets/`)

```bash
make vault           # rebuild every page
make vault-preview   # also render PNGs into ./vault-preview (Docker, playwright image)
python3 scripts/vault/build.py ui graph   # only some pages
```
