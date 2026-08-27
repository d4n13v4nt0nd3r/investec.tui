package tui

import (
	"fmt"
	"strings"

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
}

func newTransactionsView(account api.Account, currency, fromDate, toDate string) transactionsView {
	return transactionsView{
		account:  account,
		currency: currency,
		fromDate: fromDate,
		toDate:   toDate,
		pageSize: 20,
		loading:  true,
	}
}

func newPendingTransactionsView(account api.Account, currency string) transactionsView {
	return transactionsView{
		account:  account,
		currency: currency,
		pageSize: 20,
		loading:  true,
		pending:  true,
	}
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

	// Determine visible range
	end := v.offset + v.pageSize
	if end > len(v.transactions) {
		end = len(v.transactions)
	}

	// Amounts are always in the account's currency, so it's shown once in the
	// column heading rather than repeated on every row.
	visible := v.transactions[v.offset:end]
	for i, tx := range visible {
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

		globalIdx := v.offset + i
		if globalIdx == v.cursor {
			b.WriteString(selectedRowStyle.Render("> " + row[2:]))
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
	b.WriteString(mutedStyle(fmt.Sprintf("  Showing %d-%d of %d %s", v.offset+1, end, len(v.transactions), label)))

	return b.String()
}

func mutedStyle(s string) string {
	return helpStyle.Render(s)
}
