package main

import (
	"strings"
	"testing"
	"time"

	"github.com/eurico-martins/lazytree/internal/config"
)

func TestOptionsFrom(t *testing.T) {
	cfg := config.Default()
	cfg.FetchInterval = 2 * time.Minute
	cfg.ShowAll = false
	cfg.HideTags = true
	cfg.HideRefs = []string{"origin/x/*"}
	cfg.Layout = "split"

	opts := optionsFrom(cfg)
	if opts.FetchInterval != 2*time.Minute || opts.ShowAll || !opts.HideTags || opts.HideRemotes {
		t.Errorf("settings not carried over: %+v", opts)
	}
	if opts.Layout != "split" || len(opts.HideRefs) != 1 {
		t.Errorf("layout/hide_refs not carried over: %+v", opts)
	}
}

func TestUsageMentionsPathAndVersion(t *testing.T) {
	for _, want := range []string{"lazytree [path]", "--version"} {
		if !strings.Contains(usage, want) {
			t.Errorf("usage text should mention %q", want)
		}
	}
}

func TestUsageDocumentsFlags(t *testing.T) {
	for _, want := range []string{"--version", "--help"} {
		if !strings.Contains(usage, want) {
			t.Errorf("usage should document %q (CLI is part of the 1.x contract)", want)
		}
	}
}
