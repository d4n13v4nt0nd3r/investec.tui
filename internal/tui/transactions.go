package tui

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"investec.openbanking.tui/internal/api"
)

type transactionsView struct {
	account      api.Account
	transactions []api.Transaction
	currency     string
	cursor       int
	offset       int // scroll offset for viewport
	pageSize     int
	fromDate     string
	toDate       string
	pending      bool // true when showing pending transactions
	err          error
	loading      bool
	editing      bool   // true when editing date filter
	editField    int    // 0 = fromDate, 1 = toDate
	editBuffer   string // current edit text
	status       string // last CSV export path or message
	saving       bool
	searching    bool   // true when the description/amount search box is active
	searchQuery  string // text typed to filter transactions
}

// Lines the view spends on everything that is not a transaction row: the app
// frame's padding, the title, the date filter, the table header and its rule,
// the footer and the help line. The pending view has no date filter. The
// search box costs two more.
const (
	transactionsChrome        = 13
	pendingTransactionsChrome = 11
	transactionsSearchExtra   = 2

	// minPageSize keeps the table usable in a window too short to fit a full
	// page, accepting overflow rather than showing nothing.
	minPageSize = 5
)

func newTransactionsView(account api.Account, currency, fromDate, toDate string, windowHeight int) transactionsView {
	v := transactionsView{
		account:  account,
		currency: currency,
		fromDate: fromDate,
		toDate:   toDate,
		loading:  true,
	}
	v.fitTo(windowHeight)
	return v
}

func newPendingTransactionsView(account api.Account, currency string, windowHeight int) transactionsView {
	v := transactionsView{
		account:  account,
		currency: currency,
		loading:  true,
		pending:  true,
	}
	v.fitTo(windowHeight)
	return v
}

// fitTo sizes the page to the window.
//
// Windows consoles cannot be resized by the app, so the table has to adapt to
// whatever the console opens at rather than assuming the 120x35 the macOS
// build asks Terminal for.
func (v *transactionsView) fitTo(windowHeight int) {
	chrome := transactionsChrome
	if v.pending {
		chrome = pendingTransactionsChrome
	}
	if v.searching || v.searchQuery != "" {
		chrome += transactionsSearchExtra
	}

	size := windowHeight - chrome
	if size < minPageSize {
		size = minPageSize
	}
	v.pageSize = size
	v.clampOffset()
}

// clampOffset keeps the cursor inside the visible page.
func (v *transactionsView) clampOffset() {
	if v.pageSize <= 0 {
		v.pageSize = minPageSize
	}
	if v.cursor < v.offset {
		v.offset = v.cursor
	}
	if v.cursor >= v.offset+v.pageSize {
		v.offset = v.cursor - v.pageSize + 1
	}
	if v.offset < 0 {
		v.offset = 0
	}
}

// visibleTransactions returns transactions matching the current search query.
// An empty query shows everything. Otherwise a transaction matches when the
// query fuzzy-matches its description/reference or its amount.
func (v transactionsView) visibleTransactions() []api.Transaction {
	query := strings.TrimSpace(v.searchQuery)
	if query == "" {
		return v.transactions
	}
	filtered := make([]api.Transaction, 0, len(v.transactions))
	for _, tx := range v.transactions {
		if transactionMatchesQuery(tx, query) {
			filtered = append(filtered, tx)
		}
	}
	return filtered
}

// transactionMatchesQuery is true when query fuzzy-matches the description
// (or bank reference) and/or the signed amount.
func transactionMatchesQuery(tx api.Transaction, query string) bool {
	if fuzzyMatch(query, tx.Detail()) {
		return true
	}
	return amountMatchesQuery(tx.SignedAmount(), query)
}

// fuzzyMatch is a case-insensitive match: exact substring first, then a
// subsequence match so typos like "wlmrt" still find "WALMART".
func fuzzyMatch(query, target string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true
	}
	t := strings.ToLower(target)
	if strings.Contains(t, q) {
		return true
	}
	return fuzzySubsequence(q, t)
}

// fuzzySubsequence is true when every rune in query appears in order in target.
func fuzzySubsequence(query, target string) bool {
	qr := []rune(query)
	if len(qr) == 0 {
		return true
	}
	i := 0
	for _, r := range target {
		if r == qr[i] {
			i++
			if i == len(qr) {
				return true
			}
		}
	}
	return false
}

// amountMatchesQuery matches against the display amount and a digit-only form
// so "1234", "1 234", "234.56", and "-50" all work.
func amountMatchesQuery(amount float64, query string) bool {
	q := strings.TrimSpace(query)
	if q == "" {
		return true
	}
	display := FormatAmount(amount, "")
	if fuzzyMatch(q, display) {
		return true
	}
	// Also compare space-stripped and digit-normalized forms.
	qNorm := normalizeAmountQuery(q)
	if qNorm == "" {
		return false
	}
	displayNorm := normalizeAmountQuery(display)
	if strings.Contains(displayNorm, qNorm) {
		return true
	}
	// Absolute value without sign, for queries that omit the leading minus.
	absDisplay := normalizeAmountQuery(FormatAmount(absFloat(amount), ""))
	return strings.Contains(absDisplay, qNorm)
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// normalizeAmountQuery keeps digits, one leading minus, and a decimal point,
// dropping spaces and other separators the user might type or see on screen.
func normalizeAmountQuery(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	seenDot := false
	for _, r := range s {
		switch {
		case r == '-' && b.Len() == 0:
			b.WriteRune(r)
		case r == '.' && !seenDot:
			seenDot = true
			b.WriteRune(r)
		case unicode.IsDigit(r):
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (v transactionsView) render() string {
	if v.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}

	var b strings.Builder

	if !v.pending {
		// Date filter
		fromLabel := "From: "
		toLabel := "  To: "
		fromVal := v.fromDate
		toVal := v.toDate
		if fromVal == "" {
			fromVal = "(default)"
		}
		if toVal == "" {
			toVal = "(default)"
		}

		if v.editing && v.editField == 0 {
			fromVal = v.editBuffer + "▎"
		}
		if v.editing && v.editField == 1 {
			toVal = v.editBuffer + "▎"
		}

		filterLine := fmt.Sprintf("%s%s%s%s", fromLabel, fromVal, toLabel, toVal)
		b.WriteString(normalRowStyle.Render(filterLine))
		b.WriteString("\n\n")
	}

	if v.searching || v.searchQuery != "" {
		query := v.searchQuery
		if v.searching {
			query += "▎"
		}
		b.WriteString(normalRowStyle.Render(fmt.Sprintf("Search (description / amount): %s", query)))
		b.WriteString("\n\n")
	}

	if v.saving {
		b.WriteString(loadingStyle.Render("Exporting CSV..."))
		b.WriteString("\n")
	}
	if v.status != "" {
		style := successStyle
		if strings.HasPrefix(v.status, "Export failed") || strings.HasPrefix(v.status, "Nothing") {
			style = errorStyle
		}
		b.WriteString(style.Render(v.status))
		b.WriteString("\n")
	}

	if v.loading {
		msg := "Loading transactions..."
		if v.pending {
			msg = "Loading pending transactions..."
		}
		b.WriteString(loadingStyle.Render(msg))
		return b.String()
	}

	if len(v.transactions) == 0 {
		msg := "No transactions found."
		if v.pending {
			msg = "No pending transactions found."
		}
		b.WriteString(loadingStyle.Render(msg))
		return b.String()
	}

	txns := v.visibleTransactions()
	if len(txns) == 0 {
		b.WriteString(loadingStyle.Render("No matching transactions."))
		return b.String()
	}

	// Header
	amountLabel := "Amount"
	lastColLabel := "Balance"
	if v.currency != "" {
		amountLabel = fmt.Sprintf("Amount (%s)", v.currency)
		lastColLabel = fmt.Sprintf("Balance (%s)", v.currency)
	}
	if v.pending {
		lastColLabel = "Status"
	}
	header := fmt.Sprintf("  %-12s %-7s %-52s %15s %15s",
		"Date", "Type", "Description", amountLabel, lastColLabel)
	b.WriteString(headerRowStyle.Render(header))
	b.WriteString("\n")

	// Determine visible range over the filtered list
	start := v.offset
	if start > len(txns) {
		start = len(txns)
	}
	end := start + v.pageSize
	if end > len(txns) {
		end = len(txns)
	}

	// Amounts are always in the account's currency, so it's shown once in the
	// column heading rather than repeated on every row.
	page := txns[start:end]
	for i, tx := range page {
		amtStr := FormatAmount(tx.SignedAmount(), "")
		lastCol := FormatAmount(tx.RunningBalance, "")
		if v.pending {
			lastCol = tx.Status
		}

		row := fmt.Sprintf("  %-12s %-7s %-52s %15s %15s",
			tx.Date(),
			tx.Kind(),
			truncate(tx.Detail(), 50),
			amtStr,
			lastCol,
		)

		globalIdx := start + i
		if framedLook {
			b.WriteString(transactionRow(tx, lastCol, globalIdx == v.cursor))
		} else if globalIdx == v.cursor {
			b.WriteString(selectedRow(row))
		} else {
			b.WriteString(normalRowStyle.Render(row))
		}
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("\n")
	label := "transactions"
	if v.pending {
		label = "pending transactions"
	}
	footer := fmt.Sprintf("  Showing %d-%d of %d %s", start+1, end, len(txns), label)
	if v.searchQuery != "" && len(txns) != len(v.transactions) {
		footer = fmt.Sprintf("  Showing %d-%d of %d matching (%d total)", start+1, end, len(txns), len(v.transactions))
	}
	b.WriteString(mutedStyle(footer))

	return b.String()
}

func mutedStyle(s string) string {
	return helpStyle.Render(s)
}

// transactionRow is a table row in the framed look. It keeps the classic
// columns, but the type becomes an arrow and the amount carries a sign, both
// coloured by which way the money went.
func transactionRow(tx api.Transaction, lastCol string, selected bool) string {
	base, mark := normalRowStyle, normalRowStyle.Render("  ")
	if selected {
		base = selectedRowStyle
		mark = selectedMarkStyle.Render("▌") + base.Render(" ")
	}
	amount := tx.SignedAmount()
	flow := flowStyle(base, amount)

	return mark +
		base.Render(fmt.Sprintf("%-12s ", tx.Date())) +
		flow.Render(fmt.Sprintf("%-7s", flowLabel(tx))) +
		base.Render(fmt.Sprintf(" %-52s ", truncate(tx.Detail(), 50))) +
		flow.Render(fmt.Sprintf("%15s", signedAmount(amount))) +
		base.Render(fmt.Sprintf(" %15s", lastCol))
}

// flowLabel is the type column in the framed look. Both glyph sets use a
// single-cell arrow, so the column lines up either way.
func flowLabel(tx api.Transaction) string {
	switch tx.Kind() {
	case "CREDIT":
		return glyphs.in + " in"
	case "DEBIT":
		return glyphs.out + " out"
	}
	return ""
}

// flowStyle colours an amount by direction, on the row's own background.
func flowStyle(base lipgloss.Style, amount float64) lipgloss.Style {
	switch {
	case amount > 0:
		return base.Foreground(activePalette.credit)
	case amount < 0:
		return base.Foreground(activePalette.debit)
	}
	return base
}

// signedAmount formats an amount with a sign on credits as well as debits.
func signedAmount(amount float64) string {
	s := FormatAmount(amount, "")
	if amount > 0 {
		return "+" + s
	}
	return s
}
