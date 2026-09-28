package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// settleLook runs the real entry point with everything it changes put back
// afterwards. Tests that call it must not run in parallel.
//
// They all pin the theme, so the run never picks up the desktop theme or the
// fonts of whatever machine it happens to be on.
func settleLook(t *testing.T, isTerminal bool) {
	t.Helper()
	look, set, p := framedLook, glyphs, activePalette
	t.Cleanup(func() {
		framedLook, glyphs = look, set
		applyPalette(p)
	})
	UseSystemLook(isTerminal)
}

func TestUseSystemLook_FramesTheViewByDefault(t *testing.T) {
	// Arrange
	t.Setenv(themeEnv, "app")
	t.Setenv(glyphsEnv, "plain")

	// Act
	settleLook(t, false)

	// Assert
	if !framedLook {
		t.Fatal("the framed layout is not the default")
	}
	top := ansiStrip(strings.Split(countryModel(100, 30).View(), "\n")[0])
	if !strings.HasPrefix(top, "╭─ investec · countries ") {
		t.Errorf("top border is %q", top)
	}
}

func TestUseSystemLook_ClassicOnRequest(t *testing.T) {
	// Arrange
	t.Setenv(lookEnv, "classic")
	t.Setenv(themeEnv, "app")
	t.Setenv(glyphsEnv, "plain")

	// Act
	settleLook(t, false)

	// Assert
	if framedLook {
		t.Fatal("INVESTEC_TUI_LOOK=classic still framed the view")
	}
	view := ansiStrip(countryModel(120, 35).View())
	if strings.Contains(view, "╭─") {
		t.Error("the frame is still drawn")
	}
	if !strings.Contains(view, "Investec Open Banking") {
		t.Error("the classic title bar is missing")
	}
}

func TestUseSystemLook_AppThemeKeepsThePaintedPalette(t *testing.T) {
	// Arrange
	t.Setenv(themeEnv, "app")
	t.Setenv(glyphsEnv, "plain")

	// Act: even with a terminal to ask, the app's own colours are kept.
	settleLook(t, true)

	// Assert
	if activePalette.bg != investecPalette.bg {
		t.Errorf("background = %v, want the app's own %v", activePalette.bg, investecPalette.bg)
	}
}

func TestUseSystemLook_TerminalThemeNeedsATerminalToAsk(t *testing.T) {
	// Arrange
	t.Setenv(themeEnv, "terminal")
	t.Setenv(glyphsEnv, "plain")

	// Act: with nothing to ask, there is nothing to follow.
	settleLook(t, false)

	// Assert
	if _, unpainted := activePalette.bg.(lipgloss.NoColor); unpainted {
		t.Error("the app stopped painting its background without a terminal to follow")
	}
}

func TestUseSystemLook_GlyphOverrideIsObeyed(t *testing.T) {
	// Arrange
	t.Setenv(themeEnv, "app")
	t.Setenv(glyphsEnv, "nerd")

	// Act
	settleLook(t, false)

	// Assert
	if glyphs != nerdGlyphs {
		t.Errorf("glyphs = %+v, want the Nerd Font set", glyphs)
	}
}
