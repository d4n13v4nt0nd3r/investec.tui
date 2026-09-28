package tui

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// framedLook swaps the title bar and help line for a btop-style frame, with
// the splash page, the banner and the big-digit balance that go with it. It
// is the default everywhere; INVESTEC_TUI_LOOK=classic turns it off.
var framedLook bool

// Environment overrides, for the times the app guesses wrong.
const (
	// lookEnv picks the layout: "classic" for the title bar, "framed" or
	// unset for the frame.
	lookEnv = "INVESTEC_TUI_LOOK"
	// themeEnv picks the colours: "app" for the app's own painted palette,
	// "terminal" for the terminal's, unset to follow the desktop where
	// there is one to follow.
	themeEnv = "INVESTEC_TUI_THEME"
	// glyphsEnv picks the icons: "nerd" or "plain".
	glyphsEnv = "INVESTEC_TUI_GLYPHS"
)

// UseSystemLook settles how the app draws itself: framed or classic, in
// whose colours, and with which icons.
//
// main calls it once, before Bubble Tea takes the terminal, because working
// out the colours means asking the terminal for its own and reading the
// reply. isTerminal says whether there is one to ask.
func UseSystemLook(isTerminal bool) {
	framedLook = !equalsSetting(os.Getenv(lookEnv), "classic")

	desktop := false
	switch {
	case equalsSetting(os.Getenv(themeEnv), "app"):
		// The app's own palette is already applied.
	case equalsSetting(os.Getenv(themeEnv), "terminal"):
		useTerminalTheme(isTerminal)
	default:
		// A desktop theme is the richest source, where there is one.
		desktop = useDesktopTheme()
		if !desktop {
			useTerminalTheme(isTerminal)
		}
	}

	// A desktop that themes its terminals -- Omarchy, today -- ships a Nerd
	// Font in them too, so there is no need to go looking for one.
	glyphs = chooseGlyphs(os.Getenv(glyphsEnv), desktop)
}

// useTerminalTheme dresses the app in the terminal's own colours. With no
// terminal to ask there is nothing to follow, so the app keeps its own.
func useTerminalTheme(isTerminal bool) {
	if !isTerminal {
		return
	}
	applyPalette(terminalPalette(lipgloss.HasDarkBackground()))
}

// equalsSetting compares an environment value with a setting name, ignoring
// case and the spaces a shell script might leave behind.
func equalsSetting(value, setting string) bool {
	return strings.EqualFold(strings.TrimSpace(value), setting)
}
