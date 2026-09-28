package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"investec.openbanking.tui/internal/config"
)

// splashModel opens the framed layout on its splash page.
func splashModel(t *testing.T, width, height int) Model {
	t.Helper()
	useOmarchyLook(t)
	return resize(NewModel([]config.Country{
		{Name: "South Africa", Code: "ZA", ClientID: "id", ClientSecret: "secret", APIKey: "key"},
	}), width, height)
}

func TestSplash_OnlyTheFramedLayoutOpensOnIt(t *testing.T) {
	// Act
	m := NewModel([]config.Country{{Name: "South Africa", Code: "ZA"}})

	// Assert
	if m.state != viewCountry {
		t.Errorf("classic layout opened on %v, want the country page", m.state)
	}
}

func TestSplash_UseCurrentGoesToTheCountryPage(t *testing.T) {
	for _, keys := range [][]string{{"enter"}, {"u"}} {
		// Arrange
		m := splashModel(t, 100, 30)
		if m.state != viewSplash {
			t.Fatalf("framed layout opened on %v, want the splash page", m.state)
		}

		// Act
		m, _ = press(t, m, keys...)

		// Assert
		if m.state != viewCountry {
			t.Errorf("%v: state = %v, want the country page", keys, m.state)
		}
	}
}

func TestSplash_ChangeOpensSetup(t *testing.T) {
	for _, keys := range [][]string{{"down", "enter"}, {"c"}} {
		// Arrange
		m := splashModel(t, 100, 30)

		// Act
		m, _ = press(t, m, keys...)

		// Assert
		if m.state != viewSetup {
			t.Fatalf("%v: state = %v, want the setup screen", keys, m.state)
		}
		if !m.setup.canCancel {
			t.Errorf("%v: setup opened from the splash page should be escapable", keys)
		}
	}
}

func TestSplash_FillsTheWindowWithTheRightBanner(t *testing.T) {
	sizes := []struct {
		name          string
		width, height int
		banner        []string
	}{
		{name: "wide", width: 100, height: 30, banner: splashBanner},
		{name: "legacy conhost default", width: 80, height: 25, banner: splashBanner},
		{name: "too narrow for the big one", width: 60, height: 25, banner: compactBanner},
	}

	for _, s := range sizes {
		t.Run(s.name, func(t *testing.T) {
			// Arrange
			m := splashModel(t, s.width, s.height)

			// Act
			view := m.View()
			lines := strings.Split(view, "\n")

			// Assert
			if len(lines) != s.height {
				t.Errorf("view is %d lines, want %d", len(lines), s.height)
			}
			for i, line := range lines {
				if got := lipgloss.Width(line); got != s.width {
					t.Errorf("line %d is %d cells wide, want %d", i, got, s.width)
					break
				}
			}
			if !strings.Contains(ansiStrip(view), strings.TrimRight(s.banner[0], " ")) {
				t.Error("banner missing")
			}
		})
	}
}

func TestBanner_TopsEveryViewButTransactions(t *testing.T) {
	useOmarchyLook(t)

	views := map[string]struct {
		build func(w, h int) Model
		want  bool
	}{
		"country":      {build: countryModel, want: true},
		"accounts":     {build: func(w, h int) Model { return accountsModel(w, h, 60) }, want: true},
		"balance":      {build: balanceModel, want: true},
		"setup intro":  {build: func(w, h int) Model { return setupModel(setupIntro, w, h) }, want: true},
		"transactions": {build: transactionsModel, want: false},
	}

	for name, v := range views {
		t.Run(name, func(t *testing.T) {
			// Arrange
			m := resize(v.build(120, 35), 120, 35)

			// Act
			view := ansiStrip(m.View())

			// Assert
			if got := strings.Contains(view, compactBanner[0]); got != v.want {
				t.Errorf("banner shown = %v, want %v", got, v.want)
			}
		})
	}
}

func TestBanner_DroppedWhenTheWindowIsShort(t *testing.T) {
	useOmarchyLook(t)

	// Arrange
	m := resize(countryModel(100, 16), 100, 16)

	// Act
	view := ansiStrip(m.View())

	// Assert
	if strings.Contains(view, compactBanner[0]) {
		t.Error("banner drawn in a window too short to spare the rows")
	}
}
