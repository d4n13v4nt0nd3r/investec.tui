package tui

import (
	"os"
	"path/filepath"
)

// omarchyThemeFile finds the colours of the active Omarchy theme, or returns
// empty when Omarchy is not installed.
//
// Current Omarchy keeps the active theme under the XDG state folder; older
// releases kept it under ~/.config/omarchy.
func omarchyThemeFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	state := os.Getenv("XDG_STATE_HOME")
	if state == "" {
		state = filepath.Join(home, ".local", "state")
	}

	candidates := []string{
		filepath.Join(state, "omarchy", "current", "theme", "colors.toml"),
		filepath.Join(home, ".config", "omarchy", "current", "theme", "colors.toml"),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}
