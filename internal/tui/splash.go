package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"investec.openbanking.tui/internal/config"
)

// The splash page's choices, in the order they are listed.
const (
	splashUseCurrent = iota
	splashChange
)

var splashOptions = []string{
	splashUseCurrent: "Use current credentials",
	splashChange:     "Change credentials",
}

// splashOptionWidth is the width of a highlighted choice, so both bars are
// the same length.
const splashOptionWidth = 30

// splashView is the welcome page the framed layout opens on when there are
// credentials already, offering to go on with them or change them.
type splashView struct {
	countries []config.Country
	cursor    int
}

func newSplashView(countries []config.Country) splashView {
	return splashView{countries: countries}
}

// render lays the page out centred in a body width cells wide and rows tall.
func (v splashView) render(width, rows int) string {
	art := splashBanner
	if lipgloss.Width(art[0]) > width {
		art = compactBanner
	}

	var lines []string
	for _, row := range art {
		lines = append(lines, centre(bigNumberStyle.Render(row), width))
	}
	lines = append(lines, centre(hintStyle.Render(bannerTagline), width), "")

	if names := v.configured(); len(names) > 0 {
		lines = append(lines, centre(hintStyle.Render("Credentials saved for "+strings.Join(names, ", ")), width), "")
	}

	for i, option := range splashOptions {
		row := fmt.Sprintf("  %-*s", splashOptionWidth-2, option)
		if i == v.cursor {
			row = selectedRow(row)
		} else {
			row = normalRowStyle.Render(row)
		}
		lines = append(lines, centre(row, width))
	}

	if top := (rows - len(lines)) / 2; top > 0 {
		lines = append(make([]string, top), lines...)
	}
	return strings.Join(lines, "\n")
}

// configured names the countries that have all three values entered.
func (v splashView) configured() []string {
	var names []string
	for _, c := range v.countries {
		if c.HasCredentials() {
			names = append(names, c.Name)
		}
	}
	return names
}

// centre pads s on the left to sit in the middle of width. The padding is
// painted, since bare spaces would show the terminal's own background.
func centre(s string, width int) string {
	pad := (width - lipgloss.Width(s)) / 2
	if pad <= 0 {
		return s
	}
	return normalRowStyle.Render(strings.Repeat(" ", pad)) + s
}

// handleSplashKey moves between the choices and acts on one.
func (m Model) handleSplashKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit
	case "up", "k":
		if m.splash.cursor > 0 {
			m.splash.cursor--
		}
	case "down", "j":
		if m.splash.cursor < len(splashOptions)-1 {
			m.splash.cursor++
		}
	case "u":
		m.state = viewCountry
	case "c":
		return m.openSetup()
	case "enter":
		if m.splash.cursor == splashChange {
			return m.openSetup()
		}
		m.state = viewCountry
	}
	return m, nil
}
