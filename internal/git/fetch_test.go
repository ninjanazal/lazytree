package git

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestFetchEnvNonInteractive(t *testing.T) {
	t.Setenv("GIT_SSH_COMMAND", "")
	env := fetchEnv()
	for _, want := range []string{"GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=true", "GIT_SSH_COMMAND=ssh -o BatchMode=yes"} {
		if !slices.Contains(env, want) {
			t.Errorf("fetchEnv() missing %q: %v", want, env)
		}
	}
}

func TestFetchEnvKeepsUserSSHCommand(t *testing.T) {
	t.Setenv("GIT_SSH_COMMAND", "ssh -i mykey")
	for _, e := range fetchEnv() {
		if strings.HasPrefix(e, "GIT_SSH_COMMAND=") {
			t.Errorf("must not override user's GIT_SSH_COMMAND, got %q", e)
		}
	}
}

// A remote that demands credentials must make Fetch fail fast instead of
// prompting. The user's askpass helpers are stand-ins that hang (like a
// GUI password dialog nobody answers), so the test means something even
// without a terminal: Fetch must bypass them and fail on its own.
func TestFetchDoesNotPrompt(t *testing.T) {
	hang := filepath.Join(t.TempDir(), "askpass")
	if err := os.WriteFile(hang, []byte("#!/bin/sh\nsleep 60\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_ASKPASS", hang)
	t.Setenv("SSH_ASKPASS", hang)
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull) // no user credential helpers
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("WWW-Authenticate", `Basic realm="x"`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	r := &Runner{RepoPath: t.TempDir()}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for _, args := range [][]string{{"init", "-q"}, {"remote", "add", "origin", srv.URL + "/r.git"}} {
		if _, err := r.Run(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}
	err := Fetch(ctx, r)
	if ctx.Err() != nil {
		t.Fatal("Fetch blocked on a credential prompt")
	}
	var gerr *Error
	if !errors.As(err, &gerr) || !strings.Contains(gerr.Stderr, "Authentication failed") {
		t.Fatalf("want an authentication error, got %v", err)
	}
}
