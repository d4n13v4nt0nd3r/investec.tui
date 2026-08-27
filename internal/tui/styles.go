package tui

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Colours
	primaryColor = lipgloss.Color("#0066B2") // Investec-ish blue
	accentColor  = lipgloss.Color("#00A86B") // green for credits
	debitColor   = lipgloss.Color("#E05555") // red for debits
	mutedColor   = lipgloss.Color("#888888")
	headerColor  = lipgloss.Color("#FFFFFF")
	selectedBg   = lipgloss.Color("#1A3A5C")

	// textColor is the colour of ordinary body text.
	//
	// It must be set explicitly. Leaving a style with no foreground makes the
	// text fall through to the terminal profile's default, and the packaged
	// app opens in whichever Terminal profile the user happens to have -- a
	// profile like Homebrew defaults to green, which is where the green rows
	// came from. Adaptive so a light profile gets dark text rather than white
	// on white.
	textColor = lipgloss.AdaptiveColor{Light: "#1A1A1A", Dark: "#FFFFFF"}

	// Layout
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(headerColor).
			Background(primaryColor).
			Padding(0, 2).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			MarginBottom(1)

	// Table
	headerRowStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(mutedColor)

	selectedRowStyle = lipgloss.NewStyle().
				Background(selectedBg).
				Foreground(headerColor)

	normalRowStyle = lipgloss.NewStyle().
			Foreground(textColor)

	// Balance card
	labelStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Width(22)

	valueStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textColor)

	// Status bar
	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	loadingStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
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
