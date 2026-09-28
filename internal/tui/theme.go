package tui

import (
	"os"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	// themeFile is the desktop theme the styles were loaded from, or empty
	// when the app is using its own palette.
	themeFile string
	// themeModTime is when themeFile had last changed as of loading it, so
	// the first check does not reload what is already applied.
	themeModTime time.Time
)

// themePollInterval is how often the theme file is checked for a change. A
// stat every couple of seconds is cheap, and needs no hook installed.
const themePollInterval = 2 * time.Second

// themeCheckedMsg carries the result of looking at the theme file. p is nil
// when nothing changed.
type themeCheckedMsg struct {
	p       *palette
	modTime time.Time
}

// useDesktopTheme switches the styles to the desktop's colour theme when
// there is one, and starts following it. Today that is Omarchy's, on Linux;
// elsewhere there is no such thing and the caller falls back to the
// terminal's own colours. It reports whether a theme was found.
func useDesktopTheme() bool {
	path := omarchyThemeFile()
	if path == "" {
		return false
	}
	msg := checkTheme(path, time.Time{})
	if msg.p == nil {
		return false
	}
	applyPalette(*msg.p)
	themeFile = path
	themeModTime = msg.modTime
	return true
}

// watchTheme checks the theme file again after the poll interval.
func watchTheme(path string, since time.Time) tea.Cmd {
	return tea.Tick(themePollInterval, func(time.Time) tea.Msg {
		return checkTheme(path, since)
	})
}

// checkTheme reads the theme file if it has changed since the given time.
//
// Switching theme replaces the file, so it can briefly be missing or
// half-written. Either way the current palette is kept and the next check
// tries again.
func checkTheme(path string, since time.Time) themeCheckedMsg {
	info, err := os.Stat(path)
	if err != nil || info.ModTime().Equal(since) {
		return themeCheckedMsg{modTime: since}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return themeCheckedMsg{modTime: since}
	}
	p, ok := omarchyPalette(parseColors(data))
	if !ok {
		return themeCheckedMsg{modTime: since}
	}
	return themeCheckedMsg{p: &p, modTime: info.ModTime()}
}

// parseColors reads the `key = "value"` lines of an Omarchy colors.toml. The
// file is flat and only holds strings, so a full TOML parser is not needed.
func parseColors(data []byte) map[string]string {
	colors := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) < 2 || (value[0] != '"' && value[0] != '\'') {
			continue
		}
		end := strings.IndexByte(value[1:], value[0])
		if end < 0 {
			continue
		}
		colors[strings.TrimSpace(key)] = value[1 : end+1]
	}
	return colors
}

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// omarchyPalette maps an Omarchy theme onto the app's palette. It needs at
// least the accent, foreground and background; anything else missing falls
// back to the app's own colour for that role.
//
// The background is left to the terminal. Omarchy already sets the terminal
// to the theme's background, and leaving it unpainted keeps the window's
// transparency and blur.
func omarchyPalette(colors map[string]string) (palette, bool) {
	pick := func(fallback lipgloss.TerminalColor, keys ...string) lipgloss.TerminalColor {
		for _, k := range keys {
			if v := colors[k]; hexColor.MatchString(v) {
				return lipgloss.Color(v)
			}
		}
		return fallback
	}

	accent := pick(nil, "accent")
	text := pick(nil, "foreground")
	background := pick(nil, "background")
	if accent == nil || text == nil || background == nil {
		return palette{}, false
	}

	return palette{
		primary:    accent,
		onPrimary:  background,
		credit:     pick(investecPalette.credit, "green"),
		debit:      pick(investecPalette.debit, "red"),
		errorColor: pick(investecPalette.errorColor, "red"),
		// dark_foreground is meant for text; muted is often too dim to
		// read help lines in.
		muted:      pick(text, "dark_foreground", "muted"),
		header:     pick(text, "bright_foreground"),
		selectedBg: pick(investecPalette.selectedBg, "selection", "lighter_background"),
		text:       text,
		bg:         lipgloss.NoColor{},
	}, true
}
