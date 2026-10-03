package ui

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
)

const flashDuration = 2 * time.Second

// MsgFlash shows a short status message in the footer; MsgFlashClear removes
// it, unless a newer flash has replaced it (Seq).
type MsgFlash struct{ Text string }
type MsgFlashClear struct{ Seq int }

// msgOpenEditor reports that the commit's patch is written to Path and the
// editor can be launched on it.
type msgOpenEditor struct {
	Path string
	Err  error
}

// MsgEditorDone reports that the editor exited.
type MsgEditorDone struct{ Err error }

// copyCmd puts text on the system clipboard. It tries the platform
// clipboard first, and falls back to the OSC 52 terminal escape (which also
// works over SSH in terminals that support it).
func copyCmd(text, label string) tea.Cmd {
	return func() tea.Msg {
		if err := clipboard.WriteAll(text); err != nil {
			termenv.NewOutput(os.Stdout).Copy(text)
		}
		return MsgFlash{Text: "copied " + label}
	}
}

// flash shows text in the footer for flashDuration.
func (m *AppModel) flash(text string) tea.Cmd {
	m.flashText = text
	m.flashSeq++
	seq := m.flashSeq
	return tea.Tick(flashDuration, func(time.Time) tea.Msg { return MsgFlashClear{Seq: seq} })
}

// editorCommand picks the editor from $VISUAL, then $EDITOR, then vi. The
// value may carry arguments ("code -w").
func editorCommand() []string {
	for _, env := range []string{"VISUAL", "EDITOR"} {
		if v := strings.Fields(os.Getenv(env)); len(v) > 0 {
			return v
		}
	}
	return []string{"vi"}
}

// writePatchCmd writes `git show <hash>` to a temp file for the editor.
func (m AppModel) writePatchCmd(hash string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		out, err := m.runner.Run(ctx, "show", "--stat", "--patch", hash)
		if err != nil {
			return msgOpenEditor{Err: err}
		}
		f, err := os.CreateTemp("", "lazytree-"+hash[:min(len(hash), 8)]+"-*.patch")
		if err != nil {
			return msgOpenEditor{Err: err}
		}
		defer f.Close()
		if _, err := f.Write(out); err != nil {
			os.Remove(f.Name())
			return msgOpenEditor{Err: err}
		}
		return msgOpenEditor{Path: f.Name()}
	}
}

// openEditorCmd suspends the TUI and runs the editor on path, deleting the
// temp file once it exits.
func openEditorCmd(path string) tea.Cmd {
	argv := editorCommand()
	cmd := exec.Command(argv[0], append(argv[1:], path)...)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		os.Remove(path)
		return MsgEditorDone{Err: err}
	})
}
