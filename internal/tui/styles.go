package tui

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// appHPadding is the outer frame's horizontal padding. View needs it to work
// out how much width is left for content.
const appHPadding = 2

var (
	// Colours
	primaryColor = lipgloss.Color("#0066B2") // Investec-ish blue
	accentColor  = lipgloss.Color("#00A86B") // green for credits
	debitColor   = lipgloss.Color("#E05555") // red for debits
	mutedColor   = lipgloss.Color("#888888")
	headerColor  = lipgloss.Color("#FFFFFF")
	selectedBg   = lipgloss.Color("#1A3A5C")

	// textColor and bgColor are the app's own palette.
	//
	// Both have to be set on every style. A style with no colour falls
	// through to the terminal profile's default, and the packaged app opens
	// in whichever profile the user happens to have -- one like Homebrew
	// defaults to green, which is where the green rows came from.
	//
	// They are fixed rather than adaptive because the app paints its own
	// background. Terminal.app answers an OSC 11 query but ignores an OSC 11
	// set, so the real window background cannot be changed; the only option
	// is to fill the cells we draw. Once we are choosing the background, the
	// foreground has to suit it rather than the user's profile.
	textColor = lipgloss.Color("#FFFFFF")
	bgColor   = lipgloss.Color("#0D1117")

	// Layout
	appStyle = lipgloss.NewStyle().
			Padding(1, appHPadding).
			Foreground(textColor).
			Background(bgColor)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(headerColor).
			Background(primaryColor).
			Padding(0, 2).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Background(bgColor).
			Bold(true).
			MarginBottom(1)

	// Table
	headerRowStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Background(bgColor).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(mutedColor).
			BorderBackground(bgColor)

	selectedRowStyle = lipgloss.NewStyle().
				Background(selectedBg).
				Foreground(headerColor)

	normalRowStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Background(bgColor)

	// Balance card
	labelStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Background(bgColor).
			Width(22)

	valueStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textColor).
			Background(bgColor)

	// Status bar
	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Background(bgColor).
			MarginTop(1)

	// hintStyle is quiet explanatory text sitting inside a view, where
	// helpStyle's top margin would push everything else down a line.
	hintStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Background(bgColor)

	successStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Background(bgColor).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555")).
			Background(bgColor).
			Bold(true)

	loadingStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Background(bgColor).
			Italic(true)
)

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
