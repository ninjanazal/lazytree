from common import *


def build():
    s = start("release", "Shipping without automation: one command prepares everything locally, you decide what gets published.")
    section(s, 0, 240, "1 · make release VERSION=v1.0.0", "pink", 1700, "scripts/release.sh. Nothing is pushed or uploaded by it.")
    st = [("Guards", "clean tree? tag free?\non main?", "red", "🛡"), ("Checks", "make check\nmake docker-matrix", "yellow", "🧪"),
          ("Tag", "git tag -a v1.0.0\n(local only)", "blue", "🏷"), ("Build", "goreleaser → dist/\n4 archives + checksums", "green", "📦"),
          ("Notes", "commits since last tag\n→ RELEASE_NOTES.md", "purple", "📝"), ("You", "push tag · upload ·\npaste notes", "gray", "🙋")]
    prev = None
    for i, (t, d, col, ic) in enumerate(st):
        b = s.card(i * 285, 320, 255, t, d, col, ic, min_h=130)
        s.step(i * 285 + 230, 320, i + 1, col, r=16)
        if prev: s.arrow(prev, b)
        prev = b
    s.note(0, 480, 820, "make release-dry VERSION=v1.0.0 shows the notes and files without\ncreating a tag. FAST=1 skips the Docker matrix.", "tip", title="Try it first")
    s.note(880, 480, 820, "Undo before pushing: git tag -d v1.0.0. After pushing, a tag is public:\ndelete it on GitHub too.", "warn", title="Undo")
    section(s, 0, 620, "2 · How the release text is written", "pink", 1700)
    rows = [["Commit starts with", "Goes to", "Example"],
            ["feat:", "### Features", "**ui:** add split view"],
            ["fix:", "### Fixes", "**git:** handle submodule paths"],
            ["perf:", "### Performance", "stream the history"],
            ["refactor:", "### Internal improvements", "**graph:** rework lanes"],
            ["docs: chore: test: ci: build: style:", "left out", "—"],
            ["anything else", "### Other changes", "Initial import"]]
    s.table(0, 700, [420, 380, 500], rows, "pink", size=15, mono_cols=(0, 2))
    s.code(1340, 700, 360, "## lazytree v1.0.0\n\n### Features\n- **ui:** …\n### Fixes\n- …\n### Install\n…", title="RELEASE_NOTES.md")
    section(s, 0, 1040, "3 · What ends up in dist/", "pink", 1700)
    for i, f in enumerate(["lazytree_1.0.0_linux_amd64.tar.gz", "lazytree_1.0.0_linux_arm64.tar.gz", "lazytree_1.0.0_darwin_amd64.tar.gz", "lazytree_1.0.0_darwin_arm64.tar.gz", "checksums.txt", "RELEASE_NOTES.md"]):
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
    s.note(0, 1520, 820, "Re-run make screenshots after UI changes, then make vault so the pages embed the new images.", "info", title="Keep images fresh")
    s.note(880, 1520, 820, "Rename [Unreleased] in CHANGELOG.md to the version + date before make release.\nThe script does not edit it.", "warn", title="Changelog first")
    section(s, 0, 1650, "5 · Makefile at a glance (make help shows these groups in colour)", "pink", 1700)
    groups = [
        ("Build & run", "blue", "🔨", "build            ./lazytree"),
        ("Quality", "green", "✅", "test             unit tests\nlint             vet + fmt-check\nvet · fmt        housekeeping\nfmt-check        CI formatting"),
        ("Run & clean", "orange", "▶", "run              build + open\ndemo             branchy test repo\nperf             100k-commit timings\nclean            remove binary"),
        ("Verification", "yellow", "🧪", "check            lint+test+cross\ncross            5 targets\ndocker-test      clean Debian\ndocker-matrix    Debian x2 + Alpine\ndocker-perf      perf in Docker"),
        ("Docs & vault", "purple", "📚", "screenshots      README images\nvault            these pages\nvault-preview    PNG previews"),
        ("Release", "pink", "📦", "dist             archives only\nrelease          tag + archives + notes\nrelease-dry      preview, no tag"),
        ("Help", "gray", "❓", "help             grouped, coloured\nNO_COLOR=1       plain output"),
    ]
    for i, (t, col, ic, body_) in enumerate(groups):
        x, y = (i % 4) * 430, 1740 + (i // 4) * 210
        s.card(x, y, 405, t, body_, col, ic, min_h=190 if i < 4 else 150, body_size=14, mono=True, title_size=19)
    s.code(1290, 1950, 410, "##@ Quality\n\ntest: ## Run all tests\n\n##@ Release\n\ndist: ## Build archives", title="Makefile: add a group", size=14)
    return s
