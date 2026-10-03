from common import *


def build():
    s = start("release", "Shipping without automation: one command prepares everything locally, you decide what gets published.")
    section(s, 0, 240, "1 · make release VERSION=v0.1.0", "pink", 1700, "scripts/release.sh. Nothing is pushed or uploaded by it.")
    st = [("Guards", "clean tree? tag free?\non main?", "red", "🛡"), ("Checks", "make check\nmake docker-matrix", "yellow", "🧪"),
          ("Tag", "git tag -a v0.1.0\n(local only)", "blue", "🏷"), ("Build", "goreleaser → dist/\n4 archives + checksums", "green", "📦"),
          ("Notes", "commits since last tag\n→ RELEASE_NOTES.md", "purple", "📝"), ("You", "push tag · upload ·\npaste notes", "gray", "🙋")]
    prev = None
    for i, (t, d, col, ic) in enumerate(st):
        b = s.card(i * 285, 320, 255, t, d, col, ic, min_h=130)
        s.step(i * 285 + 230, 320, i + 1, col, r=16)
        if prev: s.arrow(prev, b)
        prev = b
    s.note(0, 480, 820, "make release-dry VERSION=v0.1.0 shows the notes and files without\ncreating a tag. FAST=1 skips the Docker matrix.", "tip", title="Try it first")
    s.note(880, 480, 820, "Undo before pushing: git tag -d v0.1.0. After pushing, a tag is public:\ndelete it on GitHub too.", "warn", title="Undo")
    section(s, 0, 620, "2 · How the release text is written", "pink", 1700)
    rows = [["Commit starts with", "Goes to", "Example"],
            ["feat:", "### Features", "**ui:** add split view"],
            ["fix:", "### Fixes", "**git:** handle submodule paths"],
            ["perf:", "### Performance", "stream the history"],
            ["refactor:", "### Internal improvements", "**graph:** rework lanes"],
            ["docs: chore: test: ci: build: style:", "left out", "—"],
            ["anything else", "### Other changes", "Initial import"]]
    s.table(0, 700, [420, 380, 500], rows, "pink", size=15, mono_cols=(0, 2))
    s.code(1340, 700, 360, "## lazytree v0.1.0\n\n### Features\n- **ui:** …\n### Fixes\n- …\n### Install\n…", title="RELEASE_NOTES.md")
    section(s, 0, 1040, "3 · What ends up in dist/", "pink", 1700)
    for i, f in enumerate(["lazytree_0.1.0_linux_amd64.tar.gz", "lazytree_0.1.0_linux_arm64.tar.gz", "lazytree_0.1.0_darwin_amd64.tar.gz", "lazytree_0.1.0_darwin_arm64.tar.gz", "checksums.txt", "RELEASE_NOTES.md"]):
        s.chip((i % 3) * 570, 1120 + (i // 3) * 56, "📄 " + f, "pink", strong=False)
    s.text(0, 1240, "Each archive: the lazytree binary (version injected with -ldflags, see lazytree --version) + LICENSE + README.", size=16)
    section(s, 0, 1310, "4 · Screenshots & demo GIF", "pink", 1700)
    p = [("make screenshots", "gray"), ("build static binary\n+ demo repo", "blue"), ("vhs in Docker\nplays docs/tapes/*.tape", "purple"),
         ("docs/img/*.png + demo.gif", "green"), ("README + this vault\n(assets/)", "pink")]
    prev = None
    for i, (t, col) in enumerate(p):
        b = s.card(i * 342, 1390, 310, t.split("\n")[0], "\n".join(t.split("\n")[1:]), col, min_h=100, title_size=16)
        if prev: s.arrow(prev, b)
        prev = b
    s.note(0, 1520, 1700, "Re-run make screenshots after UI changes, then make vault so the pages embed the new images.", "info", title="Keep images fresh")
    section(s, 0, 1650, "5 · Makefile at a glance", "pink", 1700)
    t = [("build", "./lazytree"), ("run / demo", "try it"), ("test / lint / check", "quality"), ("docker-test / docker-matrix", "clean Linux"),
         ("perf / docker-perf", "100k timings"), ("cross", "compile 5 targets"), ("dist", "archives only"), ("release / release-dry", "prepare a release"),
         ("screenshots", "README images"), ("vault", "rebuild these pages"), ("fmt / vet / clean", "housekeeping"), ("help", "list everything")]
    for i, (k, d) in enumerate(t):
        x, y = (i % 3) * 570, 1730 + (i // 3) * 60
        b = s.chip(x, y, k, "pink", strong=True)
        s.text(b.right + 12, y + 6, d, size=15)
    return s
