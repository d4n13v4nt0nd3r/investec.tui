package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestBodyStyles_SetExplicitForeground guards against text falling through to
// the terminal's default foreground colour.
//
// The packaged app opens in whatever Terminal profile the user happens to
// have. A style with no foreground inherits that profile's default, so on a
// profile like Homebrew the account rows rendered green instead of white.
func TestBodyStyles_SetExplicitForeground(t *testing.T) {
	tests := []struct {
		name  string
		style lipgloss.Style
	}{
		{name: "table row", style: normalRowStyle},
		{name: "balance value", style: valueStyle},
		{name: "table header", style: headerRowStyle},
		{name: "selected row", style: selectedRowStyle},
		{name: "muted help text", style: helpStyle},
		{name: "label", style: labelStyle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Act
			fg := tt.style.GetForeground()

			// Assert
			if _, unset := fg.(lipgloss.NoColor); unset {
				t.Errorf("%s has no foreground colour, so it inherits the terminal profile default", tt.name)
			}
		})
	}
}

func TestSelectedRow_ContrastsWithItsBackground(t *testing.T) {
	// The selected row paints its own background, so it must also pin the
	// foreground rather than relying on the profile.

	// Act
	fg := selectedRowStyle.GetForeground()
	bg := selectedRowStyle.GetBackground()

	// Assert
	if _, unset := bg.(lipgloss.NoColor); unset {
		t.Error("selected row has no background colour")
	}
	if fg == bg {
		t.Errorf("selected row foreground and background are both %v", fg)
	}
}
