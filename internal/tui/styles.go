package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Colours
	primaryColor   = lipgloss.Color("#0066B2") // Investec-ish blue
	accentColor    = lipgloss.Color("#00A86B") // green for credits
	debitColor     = lipgloss.Color("#E05555") // red for debits
	mutedColor     = lipgloss.Color("#888888")
	headerColor    = lipgloss.Color("#FFFFFF")
	selectedBg     = lipgloss.Color("#1A3A5C")

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

	normalRowStyle = lipgloss.NewStyle()

	// Balance card
	labelStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Width(22)

	valueStyle = lipgloss.NewStyle().
			Bold(true)

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
