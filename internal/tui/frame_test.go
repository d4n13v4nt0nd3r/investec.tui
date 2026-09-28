package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"investec.openbanking.tui/internal/api"
)

// useOmarchyLook switches to the framed layout in Tokyo Night for the rest of
// the test. Tests that call it must not run in parallel.
func useOmarchyLook(t *testing.T) {
	t.Helper()
	p, ok := omarchyPalette(parseColors([]byte(tokyoNight)))
	if !ok {
		t.Fatal("test theme was rejected")
	}
	usePalette(t, p)
	omarchyLook = true
	t.Cleanup(func() { omarchyLook = false })
}

// historyFixture is 90 days of one debit a day and a monthly salary.
func historyFixture(today time.Time) []api.Transaction {
	var txns []api.Transaction
	for i := 0; i < historyDays; i++ {
		date := today.AddDate(0, 0, -i).Format("2006-01-02")
		txns = append(txns, api.Transaction{
			Type: "DEBIT", Description: "CARD PURCHASE SOME MERCHANT NAME", TransactionDate: date, Amount: 186,
		})
		if i%30 == 0 {
			txns = append(txns, api.Transaction{
				Type: "CREDIT", Description: "SALARY", TransactionDate: date, Amount: 48000,
			})
		}
	}
	return txns
}

// balanceModel builds the balance screen with its history loaded.
func balanceModel(width, height int) Model {
	m := countryModel(width, height)
	m.state = viewBalance
	m.balance = newBalanceView(api.Account{
		AccountNumber: "10012616002",
		ProductName:   "Private Bank Account",
	}, "ZA")
	m.balance.loading = false
	m.balance.balance = &api.Balance{
		CurrentBalance:   1234567.89,
		AvailableBalance: 1204567.89,
		Currency:         "ZAR",
	}
	m.balance.history = historyFixture(time.Now())
	m.balance.historyLoading = false
	return m
}

func TestFramedView_FillsTheWindowExactly(t *testing.T) {
	useOmarchyLook(t)

	sizes := []struct {
		name          string
		width, height int
	}{
		{name: "omarchy floating window", width: 100, height: 30},
		{name: "macOS app window", width: 120, height: 35},
		{name: "legacy conhost default", width: 80, height: 25},
	}
	views := map[string]func(w, h int) Model{
		"country":           countryModel,
		"accounts":          func(w, h int) Model { return accountsModel(w, h, 60) },
		"transactions":      transactionsModel,
		"balance":           balanceModel,
		"setup intro":       func(w, h int) Model { return setupModel(setupIntro, w, h) },
		"setup credentials": func(w, h int) Model { return setupModel(setupFields, w, h) },
		"setup checking":    func(w, h int) Model { return setupModel(setupChecking, w, h) },
	}

	for name, build := range views {
		for _, s := range sizes {
			t.Run(name+"/"+s.name, func(t *testing.T) {
				// Arrange
				m := resize(build(s.width, s.height), s.width, s.height)

				// Act
				lines := strings.Split(m.View(), "\n")

				// Assert
				if len(lines) != s.height {
					t.Errorf("view is %d lines, want %d", len(lines), s.height)
				}
				for i, line := range lines {
					if got := lipgloss.Width(line); got != s.width {
						t.Errorf("line %d is %d cells wide, want %d: %q", i, got, s.width, line)
						break
					}
				}
			})
		}
	}
}

func TestFramedView_PutsTitleAndLegendOnTheBorder(t *testing.T) {
	useOmarchyLook(t)

	// Act
	lines := strings.Split(countryModel(100, 30).View(), "\n")
	top, bottom := ansiStrip(lines[0]), ansiStrip(lines[len(lines)-1])

	// Assert
	if !strings.HasPrefix(top, "╭─ "+brandGlyph+" investec · countries ") || !strings.HasSuffix(top, "─╮") {
		t.Errorf("top border is %q", top)
	}
	if !strings.HasPrefix(bottom, "╰─ ↑/↓ navigate  enter select") || !strings.HasSuffix(bottom, "─╯") {
		t.Errorf("bottom border is %q", bottom)
	}
	for _, line := range lines[1 : len(lines)-1] {
		plain := ansiStrip(line)
		if !strings.HasPrefix(plain, "│") || !strings.HasSuffix(plain, "│") {
			t.Fatalf("body line is not between the side borders: %q", plain)
		}
	}
}

// TestFramedView_TablesUseTheRoomTheFrameGives checks the tables are sized for
// the frame: a full page with the footer still showing, not cut off by the
// bottom border.
func TestFramedView_TablesUseTheRoomTheFrameGives(t *testing.T) {
	useOmarchyLook(t)

	views := map[string]Model{
		"accounts":     resize(accountsModel(100, 30, 60), 100, 30),
		"transactions": resize(transactionsModel(100, 30), 100, 30),
	}
	for name, m := range views {
		t.Run(name, func(t *testing.T) {
			view := ansiStrip(m.View())
			lines := strings.Split(view, "\n")

			if !strings.Contains(view, "Showing 1-") {
				t.Error("table footer is cut off by the frame")
			}
			// The last row inside the frame is padding, so the one before
			// it should be the footer: the page fills the space.
			if footer := lines[len(lines)-3]; !strings.Contains(footer, "Showing 1-") {
				t.Errorf("page does not reach the bottom of the frame; row above the padding is %q", footer)
			}
		})
	}
}

func TestFramedView_DrawsDividers(t *testing.T) {
	useOmarchyLook(t)

	view := ansiStrip(balanceModel(100, 35).View())

	if !strings.Contains(view, "├─ recent ─") {
		t.Errorf("balance screen has no recent divider:\n%s", view)
	}
}

func TestFramedBalance_ShowsBigDigitsAndTrend(t *testing.T) {
	useOmarchyLook(t)

	view := ansiStrip(balanceModel(100, 35).View())

	// Available balance 1 204 567.89 in the big font starts with a "1".
	if !strings.Contains(view, bigNumber("1 204 567.89")[0]) {
		t.Errorf("headline balance is not in big digits:\n%s", view)
	}
	if !strings.Contains(view, "90d  low ") {
		t.Errorf("no 90-day trend:\n%s", view)
	}
	if !strings.Contains(view, "+48 000.00") {
		t.Errorf("recent credits are not signed:\n%s", view)
	}
}

func TestFramedBalance_IgnoresHistoryForAnotherAccount(t *testing.T) {
	useOmarchyLook(t)

	// Arrange
	m := balanceModel(100, 30)
	m.balance.historyLoading = true

	// Act: a slow reply from the account the user looked at before.
	next, _ := m.Update(historyLoadedMsg{accountID: "someone-else"})

	// Assert
	if !next.(Model).balance.historyLoading {
		t.Error("history for another account was taken as this one's")
	}
}

func TestTransactionRow_ColoursByDirection(t *testing.T) {
	useOmarchyLook(t)
	credit := api.Transaction{Type: "CREDIT", Description: "SALARY", TransactionDate: "2026-09-25", Amount: 48000}
	debit := api.Transaction{Type: "DEBIT", Description: "UBER", TransactionDate: "2026-09-24", Amount: 186}

	creditRow, debitRow := transactionRow(credit, "", false), transactionRow(debit, "", false)

	if !strings.Contains(creditRow, flowStyle(normalRowStyle, 1).Render(signedAmount(48000))) ||
		!strings.Contains(ansiStrip(creditRow), "+48 000.00") {
		t.Errorf("credit row is not signed and green: %q", creditRow)
	}
	if !strings.Contains(debitRow, flowStyle(normalRowStyle, -1).Render(signedAmount(-186))) {
		t.Errorf("debit row is not red: %q", debitRow)
	}
	if got := lipgloss.Width(creditRow); got != lipgloss.Width(transactionRow(credit, "", true)) {
		t.Errorf("selecting a row changes its width from %d", got)
	}
}

func TestFormatLegend_KeepsPromptsAsText(t *testing.T) {
	legend := ansiStrip(formatLegend("Type date (YYYY-MM-DD)  •  enter confirm  •  esc cancel"))

	if legend != "Type date (YYYY-MM-DD)  enter confirm  esc cancel" {
		t.Errorf("legend is %q", legend)
	}
}

func TestBigNumber_RowsLineUp(t *testing.T) {
	rows := bigNumber("-1 234.50")

	for i, row := range rows {
		if lipgloss.Width(row) != lipgloss.Width(rows[0]) {
			t.Errorf("row %d is %d wide, row 0 is %d", i, lipgloss.Width(row), lipgloss.Width(rows[0]))
		}
	}
}

func TestDailyBalances_WorksBackFromToday(t *testing.T) {
	// Arrange
	today := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	txns := []api.Transaction{
		{Type: "CREDIT", TransactionDate: "2026-09-27", Amount: 100},
		{Type: "DEBIT", TransactionDate: "2026-09-26", Amount: 30},
	}

	// Act
	got := dailyBalances(1000, txns, 3, today)

	// Assert: 25th closes before both, 26th after the debit, today is now.
	want := []float64{930, 900, 1000}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("balances = %v, want %v", got, want)
		}
	}
}

func TestSparkline_ScalesToWidthAndRange(t *testing.T) {
	if got := sparkline([]float64{1, 2, 3, 4, 5, 6, 7, 8}, 8); got != "▁▂▃▄▅▆▇█" {
		t.Errorf("ramp = %q", got)
	}
	if got := sparkline([]float64{5, 5, 5, 5}, 10); got != "▁▁▁▁" {
		t.Errorf("flat = %q", got)
	}
	// Squeezed to two columns, each shows the last value of its half.
	if got := sparkline([]float64{1, 1, 9, 9, 9, 1}, 2); got != "█▁" {
		t.Errorf("resampled = %q", got)
	}
}

// ansiStrip removes styling, leaving what the user reads.
func ansiStrip(s string) string {
	return ansi.Strip(s)
}
