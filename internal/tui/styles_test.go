package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"investec.openbanking.tui/internal/api"
	"investec.openbanking.tui/internal/config"
)

// countryModel builds the landing page.
func countryModel(width, height int) Model {
	m := NewModel([]config.Country{
		{Name: "South Africa", Code: "ZA", ClientID: "id", ClientSecret: "secret", APIKey: "key"},
		{Name: "Mauritius", Code: "MU", ClientID: "id", ClientSecret: "secret", APIKey: "key"},
	})
	m.width, m.height = width, height
	return m
}

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
			bg := tt.style.GetBackground()

			// Assert
			if _, unset := fg.(lipgloss.NoColor); unset {
				t.Errorf("%s has no foreground colour, so it inherits the terminal profile default", tt.name)
			}
			if _, unset := bg.(lipgloss.NoColor); unset {
				t.Errorf("%s has no background colour, so the profile shows through behind it", tt.name)
			}
		})
	}
}

// TestView_FillsTheWindow checks that every cell of the terminal is painted
// by the app.
//
// Terminal.app ignores requests to change the real window background, so any
// line left short of the full width would show the user's profile colour
// through as a ragged edge down the side of the screen.
func TestView_FillsTheWindow(t *testing.T) {
	const width, height = 120, 35

	// Arrange
	m := NewModel([]config.Country{
		{Name: "South Africa", Code: "ZA", ClientID: "id", ClientSecret: "secret", APIKey: "key"},
	})
	m.width, m.height = width, height

	// Act
	lines := strings.Split(m.View(), "\n")

	// Assert
	if len(lines) != height {
		t.Errorf("view is %d lines, want %d", len(lines), height)
	}
	for i, line := range lines {
		if got := lipgloss.Width(line); got != width {
			t.Errorf("line %d is %d cells wide, want %d", i, got, width)
		}
	}
}

// TestView_DoesNotWrapWideContent guards the column alignment of the tables.
// Padding the view to the window width must not reflow a row that is wider
// than the space available.
func TestView_DoesNotWrapWideContent(t *testing.T) {
	// A window far too narrow for the country table.
	const width, height = 40, 12

	// Arrange
	m := NewModel([]config.Country{
		{Name: "South Africa", Code: "ZA", ClientID: "id", ClientSecret: "secret", APIKey: "key"},
	})
	m.width, m.height = width, height

	// Act
	lines := strings.Split(m.View(), "\n")

	// Assert
	if len(lines) != height {
		t.Errorf("view is %d lines, want %d -- content wrapped instead of being clipped", len(lines), height)
	}
	for i, line := range lines {
		if got := lipgloss.Width(line); got != width {
			t.Errorf("line %d is %d cells wide, want %d", i, got, width)
		}
	}
}

// transactionsModel builds the tallest view the app has, filled with enough
// rows to reach a full page.
func transactionsModel(width, height int) Model {
	txns := make([]api.Transaction, 50)
	for i := range txns {
		txns[i] = api.Transaction{
			Type:            "DEBIT",
			Description:     "CARD PURCHASE SOME MERCHANT NAME",
			TransactionDate: "2026-08-26",
			Amount:          1234.56,
			RunningBalance:  98765.43,
		}
	}

	m := NewModel([]config.Country{
		{Name: "South Africa", Code: "ZA", ClientID: "id", ClientSecret: "secret", APIKey: "key"},
	})
	m.state = viewTransactions
	m.transactions = newTransactionsView(api.Account{}, "ZAR", "", "", height)
	m.transactions.transactions = txns
	m.transactions.loading = false
	m.width, m.height = width, height
	return m
}

// accountsModel builds the accounts list with more accounts than fit on one
// page, which is the normal case for a business profile.
func accountsModel(width, height, count int) Model {
	accs := make([]api.Account, count)
	for i := range accs {
		accs[i] = api.Account{
			AccountNumber: "10012616002",
			ReferenceName: "PTS_INV_OPS_ZAR",
			ProductName:   "Private Bank Account",
		}
	}

	m := NewModel([]config.Country{
		{Name: "South Africa", Code: "ZA", ClientID: "id", ClientSecret: "secret", APIKey: "key"},
	})
	m.state = viewAccounts
	m.accounts = newAccountsView("ZA", height)
	m.accounts.accounts = accs
	m.width, m.height = width, height
	return m
}

// TestView_FitsCommonConsoleSizes checks the layout against the window sizes
// users will actually get.
//
// macOS is resized to 120x35 by the app itself, but Windows consoles cannot be
// resized programmatically, so the app has to live within whatever the console
// opens at. Overflowing the window scrolls the alternate screen and breaks the
// display.
func TestView_FitsCommonConsoleSizes(t *testing.T) {
	sizes := []struct {
		name          string
		width, height int
	}{
		{name: "macOS app window", width: 120, height: 35},
		{name: "Windows Terminal default", width: 120, height: 30},
		{name: "legacy conhost default", width: 80, height: 25},
	}

	views := []struct {
		name  string
		build func(width, height int) Model
	}{
		{name: "transactions", build: transactionsModel},
		{name: "accounts", build: func(w, h int) Model { return accountsModel(w, h, 60) }},
	}

	for _, v := range views {
		for _, s := range sizes {
			t.Run(v.name+"/"+s.name, func(t *testing.T) {
				// Arrange
				m := v.build(s.width, s.height)

				// Act
				lines := strings.Split(m.View(), "\n")

				// Assert
				if len(lines) > s.height {
					t.Errorf("view is %d lines, overflowing a %d-line window by %d",
						len(lines), s.height, len(lines)-s.height)
				}
				for i, line := range lines {
					if got := lipgloss.Width(line); got != s.width {
						t.Errorf("line %d is %d cells wide, want %d", i, got, s.width)
						break
					}
				}
			})
		}
	}
}

// TestView_LeavesNoUnpaintedGaps guards against bars of the terminal's own
// background showing through the app.
//
// A run of spaces emitted straight after a reset carries no colour, so it
// renders in the terminal profile's background rather than the app's. That is
// what lipgloss's MaxWidth padding produced: a style with no colours generates
// no escape sequences, so its padding arrived bare.
func TestView_LeavesNoUnpaintedGaps(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })

	views := map[string]Model{
		"country":      countryModel(120, 35),
		"accounts":     accountsModel(120, 35, 60),
		"transactions": transactionsModel(120, 35),
	}

	for name, m := range views {
		t.Run(name, func(t *testing.T) {
			// Act
			lines := strings.Split(m.View(), "\n")

			// Assert
			for i, line := range lines {
				if strings.Contains(line, "\x1b[0m ") {
					t.Errorf("line %d has unpainted spaces after a reset: %s",
						i, strings.ReplaceAll(line, "\x1b", "<ESC>"))
					break
				}
			}
		})
	}
}

// TestAccounts_ScrollingKeepsCursorVisible checks that moving to the end of a
// long list scrolls the page rather than running off the bottom.
func TestAccounts_ScrollingKeepsCursorVisible(t *testing.T) {
	const count = 60

	// Arrange
	m := accountsModel(120, 30, count)

	// Act: walk the cursor to the last account.
	m.accounts.cursor = count - 1
	m.accounts.clampOffset()

	// Assert
	if m.accounts.cursor < m.accounts.offset ||
		m.accounts.cursor >= m.accounts.offset+m.accounts.pageSize {
		t.Errorf("cursor %d is outside the visible page [%d,%d)",
			m.accounts.cursor, m.accounts.offset, m.accounts.offset+m.accounts.pageSize)
	}
	if lines := strings.Split(m.View(), "\n"); len(lines) > 30 {
		t.Errorf("scrolled view is %d lines, overflowing a 30-line window", len(lines))
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
