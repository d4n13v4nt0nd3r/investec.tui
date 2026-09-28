package tui

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// appHPadding is the outer frame's horizontal padding. View needs it to work
// out how much width is left for content.
const appHPadding = 2

// palette is the set of colours every style is built from.
type palette struct {
	primary    lipgloss.TerminalColor // title bar, headings, table headers
	onPrimary  lipgloss.TerminalColor // text on the title bar
	credit     lipgloss.TerminalColor // money in, and success messages
	debit      lipgloss.TerminalColor // money out
	errorColor lipgloss.TerminalColor
	muted      lipgloss.TerminalColor
	header     lipgloss.TerminalColor // bright text, such as the selected row
	selectedBg lipgloss.TerminalColor
	text       lipgloss.TerminalColor
	bg         lipgloss.TerminalColor
}

// investecPalette is the app's own look, used everywhere a desktop theme is
// not available.
//
// text and bg have to be set on every style. A style with no colour falls
// through to the terminal profile's default, and the packaged app opens in
// whichever profile the user happens to have -- one like Homebrew defaults to
// green, which is where the green rows came from.
//
// They are fixed rather than adaptive because the app paints its own
// background. Terminal.app answers an OSC 11 query but ignores an OSC 11 set,
// so the real window background cannot be changed; the only option is to fill
// the cells we draw. Once we are choosing the background, the foreground has
// to suit it rather than the user's profile.
var investecPalette = palette{
	primary:    lipgloss.Color("#0066B2"), // Investec-ish blue
	onPrimary:  lipgloss.Color("#FFFFFF"),
	credit:     lipgloss.Color("#00A86B"),
	debit:      lipgloss.Color("#E05555"),
	errorColor: lipgloss.Color("#FF5555"),
	muted:      lipgloss.Color("#888888"),
	header:     lipgloss.Color("#FFFFFF"),
	selectedBg: lipgloss.Color("#1A3A5C"),
	text:       lipgloss.Color("#FFFFFF"),
	bg:         lipgloss.Color("#0D1117"),
}

// activePalette is the palette the styles were last built from, for the few
// places that colour part of a styled row.
var activePalette palette

var (
	appStyle         lipgloss.Style
	titleStyle       lipgloss.Style
	subtitleStyle    lipgloss.Style
	headerRowStyle   lipgloss.Style
	selectedRowStyle lipgloss.Style
	normalRowStyle   lipgloss.Style
	labelStyle       lipgloss.Style
	valueStyle       lipgloss.Style
	helpStyle        lipgloss.Style
	hintStyle        lipgloss.Style
	successStyle     lipgloss.Style
	errorStyle       lipgloss.Style
	loadingStyle     lipgloss.Style

	// The Omarchy frame
	frameBorderStyle  lipgloss.Style
	frameTitleStyle   lipgloss.Style
	legendKeyStyle    lipgloss.Style
	selectedMarkStyle lipgloss.Style
	bigNumberStyle    lipgloss.Style
)

func init() {
	applyPalette(investecPalette)
}

// applyPalette rebuilds every style from p.
func applyPalette(p palette) {
	activePalette = p

	// Layout
	appStyle = lipgloss.NewStyle().
		Padding(1, appHPadding).
		Foreground(p.text).
		Background(p.bg)

	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(p.onPrimary).
		Background(p.primary).
		Padding(0, 2).
		MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
		Foreground(p.primary).
		Background(p.bg).
		Bold(true).
		MarginBottom(1)

	// Table
	headerRowStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(p.primary).
		Background(p.bg).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(p.muted).
		BorderBackground(p.bg)

	selectedRowStyle = lipgloss.NewStyle().
		Background(p.selectedBg).
		Foreground(p.header)

	normalRowStyle = lipgloss.NewStyle().
		Foreground(p.text).
		Background(p.bg)

	// Balance card
	labelStyle = lipgloss.NewStyle().
		Foreground(p.muted).
		Background(p.bg).
		Width(22)

	valueStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(p.text).
		Background(p.bg)

	// Status bar
	helpStyle = lipgloss.NewStyle().
		Foreground(p.muted).
		Background(p.bg).
		MarginTop(1)

	// hintStyle is quiet explanatory text sitting inside a view, where
	// helpStyle's top margin would push everything else down a line.
	hintStyle = lipgloss.NewStyle().
		Foreground(p.muted).
		Background(p.bg)

	successStyle = lipgloss.NewStyle().
		Foreground(p.credit).
		Background(p.bg).
		Bold(true)

	errorStyle = lipgloss.NewStyle().
		Foreground(p.errorColor).
		Background(p.bg).
		Bold(true)

	loadingStyle = lipgloss.NewStyle().
		Foreground(p.muted).
		Background(p.bg).
		Italic(true)

	// The Omarchy frame. The border takes the accent, as Hyprland does for
	// the active window.
	frameBorderStyle = lipgloss.NewStyle().
		Foreground(p.primary).
		Background(p.bg)

	frameTitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(p.header).
		Background(p.bg)

	legendKeyStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(p.primary).
		Background(p.bg)

	selectedMarkStyle = selectedRowStyle.
		Foreground(p.primary)

	bigNumberStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(p.primary).
		Background(p.bg)
}

// selectedRow draws the row under the cursor. Rows are laid out with two
// leading spaces for the cursor mark to replace.
func selectedRow(row string) string {
	row = strings.TrimPrefix(row, "  ")
	if omarchyLook {
		return selectedMarkStyle.Render("▌") + selectedRowStyle.Render(" "+row)
	}
	return selectedRowStyle.Render("> " + row)
}

// styleInput gives a text input the app's colours. Inputs copy the styles
// when they are set, so this has to run again after the palette changes.
//
// Every style has to carry both colours. The app paints its own background,
// so a run left unstyled shows the terminal profile's through instead --
// including the blank space the input pads itself out with.
func styleInput(input *textinput.Model) {
	input.PromptStyle = normalRowStyle
	input.TextStyle = normalRowStyle
	input.PlaceholderStyle = hintStyle
	input.Cursor.Style = normalRowStyle
	input.Cursor.TextStyle = normalRowStyle
}

// FormatAmount formats a float with ≥2 decimal places, space as thousand separator,
// and the given currency prefix. Does not assume "$".
func FormatAmount(amount float64, currency string) string {
	negative := amount < 0
	abs := math.Abs(amount)

	// Format with 2 decimal places
	formatted := fmt.Sprintf("%.2f", abs)

	parts := strings.Split(formatted, ".")
	intPart := parts[0]
	decPart := parts[1]

	// Add space as thousand separator
	var result []byte
	for i, j := 0, len(intPart); i < len(intPart); i++ {
		if i > 0 && (j-i)%3 == 0 {
			result = append(result, ' ')
		}
		result = append(result, intPart[i])
	}

	sign := ""
	if negative {
		sign = "-"
	}

	if currency != "" {
		return fmt.Sprintf("%s%s %s.%s", sign, currency, string(result), decPart)
	}
	return fmt.Sprintf("%s%s.%s", sign, string(result), decPart)
}

// FormatRate formats an interest rate as a percentage with at least 2 and at
// most 7 decimal places.
func FormatRate(rate float64) string {
	s := strconv.FormatFloat(rate, 'f', -1, 64)

	intPart, decPart, hasDec := strings.Cut(s, ".")
	if !hasDec {
		decPart = ""
	}
	if len(decPart) > 7 {
		s = strconv.FormatFloat(rate, 'f', 7, 64)
		intPart, decPart, _ = strings.Cut(s, ".")
		decPart = strings.TrimRight(decPart, "0")
	}
	for len(decPart) < 2 {
		decPart += "0"
	}

	return fmt.Sprintf("%s.%s %%", intPart, decPart)
}
