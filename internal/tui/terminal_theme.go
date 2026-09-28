package tui

import "github.com/charmbracelet/lipgloss"

// The terminal's own sixteen colours, by index. Naming them by role rather
// than by number is what lets the app wear whatever theme the user has set:
// "green" is their green, not one the app chose.
const (
	ansiBlack       = "0"
	ansiRed         = "1"
	ansiGreen       = "2"
	ansiBlue        = "4"
	ansiSilver      = "7"
	ansiGrey        = "8"
	ansiBrightRed   = "9"
	ansiBrightGreen = "10"
	ansiBrightBlue  = "12"
	ansiWhite       = "15"
)

// terminalPalette follows the terminal's own theme, which is the nearest
// thing to a desktop theme that macOS and Windows offer a terminal program.
//
// Text and background are left to the terminal, so the app sits in the
// user's colours and their window keeps whatever transparency or blur it
// has. Everything else comes from the sixteen ANSI colours, picked light or
// dark so it stays readable either way.
func terminalPalette(dark bool) palette {
	p := palette{
		primary:    lipgloss.Color(ansiBlue),
		onPrimary:  lipgloss.Color(ansiWhite),
		credit:     lipgloss.Color(ansiGreen),
		debit:      lipgloss.Color(ansiRed),
		errorColor: lipgloss.Color(ansiRed),
		muted:      lipgloss.Color(ansiGrey),
		// header doubles as the selected row's text, so it has to contrast
		// with selectedBg rather than with the terminal's background.
		header:     lipgloss.Color(ansiBlack),
		selectedBg: lipgloss.Color(ansiSilver),
		text:       lipgloss.NoColor{},
		bg:         lipgloss.NoColor{},
	}

	if dark {
		p.primary = lipgloss.Color(ansiBrightBlue)
		p.onPrimary = lipgloss.Color(ansiBlack)
		p.credit = lipgloss.Color(ansiBrightGreen)
		p.debit = lipgloss.Color(ansiBrightRed)
		p.errorColor = lipgloss.Color(ansiBrightRed)
		p.header = lipgloss.Color(ansiWhite)
		p.selectedBg = lipgloss.Color(ansiGrey)
	}

	return p
}
