package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// tokyoNight is an Omarchy colors.toml, trimmed to what the app reads plus
// some it ignores.
const tokyoNight = `mode = "dark"

accent = "#7aa2f7"
selection = "#292e42"
muted = "#414868"

background = "#1a1b26"   # the terminal's own
foreground = "#a9b1d6"
dark_foreground = "#565f89"
bright_foreground = "#c0caf5"

red = "#f7768e"
green = "#9ece6a"

hyprland_active_border = "rgba(61afefff) rgba(56b6c2ff) 90deg"
`

// usePalette applies p for the rest of the test.
func usePalette(t *testing.T, p palette) {
	t.Helper()
	applyPalette(p)
	t.Cleanup(func() { applyPalette(investecPalette) })
}

func TestParseColors_ReadsQuotedValues(t *testing.T) {
	// Act
	colors := parseColors([]byte(tokyoNight))

	// Assert
	want := map[string]string{
		"mode":       "dark",
		"accent":     "#7aa2f7",
		"background": "#1a1b26",
		"red":        "#f7768e",
	}
	for key, value := range want {
		if colors[key] != value {
			t.Errorf("%s = %q, want %q", key, colors[key], value)
		}
	}
}

func TestOmarchyPalette_MapsThemeRoles(t *testing.T) {
	// Act
	p, ok := omarchyPalette(parseColors([]byte(tokyoNight)))

	// Assert
	if !ok {
		t.Fatal("theme was rejected")
	}
	tests := []struct {
		role string
		got  lipgloss.TerminalColor
		want lipgloss.TerminalColor
	}{
		{role: "primary", got: p.primary, want: lipgloss.Color("#7aa2f7")},
		{role: "text on primary", got: p.onPrimary, want: lipgloss.Color("#1a1b26")},
		{role: "text", got: p.text, want: lipgloss.Color("#a9b1d6")},
		{role: "header", got: p.header, want: lipgloss.Color("#c0caf5")},
		{role: "muted", got: p.muted, want: lipgloss.Color("#565f89")},
		{role: "selected background", got: p.selectedBg, want: lipgloss.Color("#292e42")},
		{role: "credit", got: p.credit, want: lipgloss.Color("#9ece6a")},
		{role: "error", got: p.errorColor, want: lipgloss.Color("#f7768e")},
		{role: "background", got: p.bg, want: lipgloss.NoColor{}},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.role, tt.got, tt.want)
		}
	}
}

func TestOmarchyPalette_RejectsIncompleteTheme(t *testing.T) {
	// A theme without an accent has nothing to colour the headings with.
	colors := parseColors([]byte(`foreground = "#ffffff"` + "\n" + `background = "#000000"`))

	if _, ok := omarchyPalette(colors); ok {
		t.Error("theme without an accent was accepted")
	}
}

func TestOmarchyPalette_IgnoresValuesThatAreNotColours(t *testing.T) {
	colors := parseColors([]byte(tokyoNight + `green = "rgba(0,255,0,1)"` + "\n"))

	p, _ := omarchyPalette(colors)

	if p.credit != investecPalette.credit {
		t.Errorf("credit = %v, want the app's own %v", p.credit, investecPalette.credit)
	}
}

func TestCheckTheme_ReloadsOnlyWhenTheFileChanges(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "colors.toml")
	if err := os.WriteFile(path, []byte(tokyoNight), 0o600); err != nil {
		t.Fatal(err)
	}

	// Act
	first := checkTheme(path, time.Time{})
	second := checkTheme(path, first.modTime)

	// Assert
	if first.p == nil {
		t.Fatal("first check did not load the theme")
	}
	if second.p != nil {
		t.Error("unchanged file was loaded again")
	}
	if !second.modTime.Equal(first.modTime) {
		t.Errorf("modTime moved from %v to %v with no change", first.modTime, second.modTime)
	}
}

func TestCheckTheme_KeepsCurrentThemeWhileFileIsMissing(t *testing.T) {
	// Switching theme replaces the file, so it can briefly not exist.
	since := time.Now()

	msg := checkTheme(filepath.Join(t.TempDir(), "colors.toml"), since)

	if msg.p != nil || !msg.modTime.Equal(since) {
		t.Errorf("missing file changed the theme: %+v", msg)
	}
}

func TestTerminalPalette_LeavesTextAndBackgroundToTheTerminal(t *testing.T) {
	for _, dark := range []bool{true, false} {
		// Act
		p := terminalPalette(dark)

		// Assert
		if _, unset := p.bg.(lipgloss.NoColor); !unset {
			t.Errorf("dark=%v: background is painted %v, want the terminal's own", dark, p.bg)
		}
		if _, unset := p.text.(lipgloss.NoColor); !unset {
			t.Errorf("dark=%v: body text is %v, want the terminal's own", dark, p.text)
		}
		// header is the selected row's text, so the two have to differ.
		if p.header == p.selectedBg {
			t.Errorf("dark=%v: selected row is %v on %v", dark, p.header, p.selectedBg)
		}
		if p.credit == p.debit {
			t.Errorf("dark=%v: money in and out are both %v", dark, p.credit)
		}
	}
}

// TestTerminalPalette_AdaptsToTheBackground checks the roles that have to
// change with the terminal's background to stay readable.
func TestTerminalPalette_AdaptsToTheBackground(t *testing.T) {
	dark, light := terminalPalette(true), terminalPalette(false)

	tests := []struct {
		role        string
		dark, light lipgloss.TerminalColor
	}{
		{role: "accent", dark: dark.primary, light: light.primary},
		{role: "bright text", dark: dark.header, light: light.header},
		{role: "selected background", dark: dark.selectedBg, light: light.selectedBg},
	}
	for _, tt := range tests {
		if tt.dark == tt.light {
			t.Errorf("%s is %v on both a dark and a light terminal", tt.role, tt.dark)
		}
	}
}

// TestView_TerminalThemePaintsNothingOfItsOwn is the macOS and Windows
// counterpart of the Omarchy check: following the terminal means leaving its
// background be, while still filling every cell of the window.
func TestView_TerminalThemePaintsNothingOfItsOwn(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })

	// Arrange
	usePalette(t, terminalPalette(true))
	m := countryModel(120, 35)

	// Act
	view := m.View()

	// Assert
	if strings.Contains(view, termenv.TrueColor.Color("#0D1117").Sequence(true)) {
		t.Error("the app's own #0D1117 background is still painted")
	}
	for i, line := range strings.Split(view, "\n") {
		if got := lipgloss.Width(line); got != 120 {
			t.Errorf("line %d is %d cells wide, want 120", i, got)
			break
		}
	}
}

// TestView_OmarchyThemeUsesItsColours checks the rendered screen, not just the
// palette: the title bar carries the theme's accent, and nothing paints over
// the terminal's own background.
func TestView_OmarchyThemeUsesItsColours(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })

	// Arrange
	p, _ := omarchyPalette(parseColors([]byte(tokyoNight)))
	usePalette(t, p)
	m := countryModel(120, 35)

	// Act
	view := m.View()

	// Assert
	background := func(hex string) string { return termenv.TrueColor.Color(hex).Sequence(true) }
	if !strings.Contains(view, background("#7aa2f7")) {
		t.Error("title bar is not drawn in the theme's accent #7aa2f7")
	}
	if strings.Contains(view, background("#0D1117")) {
		t.Error("the app's own #0D1117 background is still painted")
	}
	lines := strings.Split(view, "\n")
	if len(lines) != 35 {
		t.Errorf("view is %d lines, want 35", len(lines))
	}
	for i, line := range lines {
		if got := lipgloss.Width(line); got != 120 {
			t.Errorf("line %d is %d cells wide, want 120", i, got)
			break
		}
	}
}
