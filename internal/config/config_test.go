package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func write(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadFile_MissingFileGivesDefaults(t *testing.T) {
	cfg, err := LoadFile(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil || !reflect.DeepEqual(cfg, Default()) {
		t.Fatalf("got %+v, %v; want defaults", cfg, err)
	}
}

func TestLoadFile_Values(t *testing.T) {
	cfg, err := LoadFile(write(t, "fetch_interval = \"2m\"\nshow_all = false\ntheme = \"Light\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := Default()
	want.FetchInterval, want.ShowAll, want.Theme = 2*time.Minute, false, "light"
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("got %+v, want %+v", cfg, want)
	}
	off, err := LoadFile(write(t, "fetch_interval = \"off\"\n"))
	if err != nil || off.FetchInterval != 0 || !off.ShowAll {
		t.Errorf("off: got %+v, %v (other settings must keep their defaults)", off, err)
	}
}

func TestLoadFile_Errors(t *testing.T) {
	cases := map[string]string{
		"fetch_interval = \"soon\"": "not a duration",
		"fetch_interval = \"1s\"":   "too short",
		"theme = \"neon\"":          "must be auto, light or dark",
		"fetch_intervall = \"60s\"": "unknown key",
		"show_all = \"yes\"":        "config",
		"this is not toml":          "config",
	}
	for body, wantSub := range cases {
		_, err := LoadFile(write(t, body))
		if err == nil || !strings.Contains(err.Error(), wantSub) {
			t.Errorf("%q: got err %v, want it to contain %q", body, err, wantSub)
		}
	}
}

func TestLoadFile_FullConfig(t *testing.T) {
	cfg, err := LoadFile(write(t, `
layout = "split"
hide_tags = true
hide_remotes = true
hide_refs = ["origin/dependabot/*"]
lane_colors = ["#ff0000", "33"]

[keys]
zen = ["x"]
quit = ["q", "Q"]

[colors]
hash = "#aabbcc"
added = "34"
`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Layout != "split" || !cfg.HideTags || !cfg.HideRemotes {
		t.Errorf("flags not loaded: %+v", cfg)
	}
	if len(cfg.HideRefs) != 1 || len(cfg.LaneColors) != 2 {
		t.Errorf("lists not loaded: %+v", cfg)
	}
	if got := cfg.Keys["zen"]; len(got) != 1 || got[0] != "x" {
		t.Errorf("keys not loaded: %+v", cfg.Keys)
	}
	if cfg.Colors["hash"] != "#aabbcc" || cfg.Colors["added"] != "34" {
		t.Errorf("colors not loaded: %+v", cfg.Colors)
	}
	if _, err := LoadFile(write(t, "layout = \"diagonal\"")); err == nil {
		t.Error("an invalid layout must be rejected")
	}
}
