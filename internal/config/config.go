// Package config loads lazytree's optional TOML config file.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// Config holds user settings. The zero value is not usable; start from
// Default.
type Config struct {
	// FetchInterval is how often the background `git fetch --all` runs.
	// Zero disables background fetching.
	FetchInterval time.Duration
	// ShowAll starts with all refs shown (git log --all) rather than HEAD only.
	ShowAll bool
	// Theme is "auto" (detect the terminal background), "light" or "dark".
	Theme string
	// Layout is "auto" (split view on wide terminals), "split" or "popup".
	Layout string
	// HideTags and HideRemotes start with those ref pills hidden; HideRefs
	// lists ref-name globs ("*" matches anything) that are always hidden.
	HideTags    bool
	HideRemotes bool
	HideRefs    []string
	// Keys rebinds actions: action name -> keys. Names are validated by the
	// ui package (ui.ApplyKeys).
	Keys map[string][]string
	// Colors overrides named colors ("#rrggbb" or 0-255); LaneColors
	// replaces the graph lane palette. Validated by ui.ApplyColors.
	Colors     map[string]string
	LaneColors []string
}

// Default returns the settings used when no config file exists.
func Default() Config {
	return Config{FetchInterval: 60 * time.Second, ShowAll: true, Theme: "auto", Layout: "auto"}
}

// file mirrors the TOML layout. Pointers distinguish "unset" from zero values.
type file struct {
	FetchInterval *string  `toml:"fetch_interval"`
	ShowAll       *bool    `toml:"show_all"`
	Theme         *string  `toml:"theme"`
	Layout        *string  `toml:"layout"`
	HideTags      *bool    `toml:"hide_tags"`
	HideRemotes   *bool    `toml:"hide_remotes"`
	HideRefs      []string `toml:"hide_refs"`
	LaneColors    []string `toml:"lane_colors"`

	Keys   map[string][]string `toml:"keys"`
	Colors map[string]string   `toml:"colors"`
}

// Path returns the config file location: $XDG_CONFIG_HOME/lazytree/config.toml,
// or ~/.config/lazytree/config.toml.
func Path() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "lazytree", "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "lazytree", "config.toml"), nil
}

// Load reads the config file at Path. A missing file is not an error: the
// defaults are returned. Unknown keys and invalid values are errors, so a
// typo doesn't silently do nothing.
func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Default(), nil // no home directory: just use defaults
	}
	return LoadFile(path)
}

// LoadFile is Load for an explicit path.
func LoadFile(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("config %s: %w", path, err)
	}
	var f file
	md, err := toml.Decode(string(data), &f)
	if err != nil {
		return Default(), fmt.Errorf("config %s: %w", path, err)
	}
	if undecoded := md.Undecoded(); len(undecoded) > 0 {
		return Default(), fmt.Errorf("config %s: unknown key %q (valid: fetch_interval, show_all, theme, layout, hide_tags, hide_remotes, hide_refs, lane_colors, [keys], [colors])", path, undecoded[0].String())
	}
	if f.FetchInterval != nil {
		d, err := parseInterval(*f.FetchInterval)
		if err != nil {
			return Default(), fmt.Errorf("config %s: fetch_interval: %w", path, err)
		}
		cfg.FetchInterval = d
	}
	if f.ShowAll != nil {
		cfg.ShowAll = *f.ShowAll
	}
	if f.Theme != nil {
		theme := strings.ToLower(*f.Theme)
		if theme != "auto" && theme != "light" && theme != "dark" {
			return Default(), fmt.Errorf("config %s: theme %q must be auto, light or dark", path, *f.Theme)
		}
		cfg.Theme = theme
	}
	if f.Layout != nil {
		layout := strings.ToLower(*f.Layout)
		if layout != "auto" && layout != "split" && layout != "popup" {
			return Default(), fmt.Errorf("config %s: layout %q must be auto, split or popup", path, *f.Layout)
		}
		cfg.Layout = layout
	}
	if f.HideTags != nil {
		cfg.HideTags = *f.HideTags
	}
	if f.HideRemotes != nil {
		cfg.HideRemotes = *f.HideRemotes
	}
	cfg.HideRefs = f.HideRefs
	cfg.LaneColors = f.LaneColors
	cfg.Keys = f.Keys
	cfg.Colors = f.Colors
	return cfg, nil
}

// parseInterval accepts a Go duration ("90s", "5m"), or "off"/"0" to disable.
func parseInterval(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "off" || s == "0" || s == "never" {
		return 0, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("%q is not a duration like \"60s\" or \"off\"", s)
	}
	if d < 5*time.Second {
		return 0, fmt.Errorf("%q is too short (minimum 5s, or \"off\")", s)
	}
	return d, nil
}
