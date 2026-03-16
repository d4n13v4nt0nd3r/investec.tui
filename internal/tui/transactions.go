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
	err          error
	loading      bool
	editing      bool   // true when editing date filter
	editField    int    // 0 = fromDate, 1 = toDate
	editBuffer   string // current edit text
}

func newTransactionsView(account api.Account, currency string) transactionsView {
	return transactionsView{
		account:  account,
		currency: currency,
		pageSize: 20,
		loading:  true,
	}
}

func (v transactionsView) render() string {
	if v.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}

	var b strings.Builder

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

	if v.loading {
		b.WriteString(loadingStyle.Render("Loading transactions..."))
		return b.String()
	}

	if len(v.transactions) == 0 {
		b.WriteString(loadingStyle.Render("No transactions found."))
		return b.String()
	}

	// Header
	header := fmt.Sprintf("  %-12s %-7s %-30s %15s %15s",
		"Date", "Type", "Description", "Amount", "Balance")
	b.WriteString(headerRowStyle.Render(header))
	b.WriteString("\n")

	// Determine visible range
	end := v.offset + v.pageSize
	if end > len(v.transactions) {
		end = len(v.transactions)
	}

	visible := v.transactions[v.offset:end]
	for i, tx := range visible {
		date := tx.TransactionDate
		if date == "" {
			date = tx.PostingDate
		}
		if len(date) > 10 {
			date = date[:10]
		}

		amtStr := FormatAmount(tx.Amount, "")
		balStr := FormatAmount(tx.RunningBalance, "")

		row := fmt.Sprintf("  %-12s %-7s %-30s %15s %15s",
			date,
			tx.Type,
			truncate(tx.Description, 28),
			amtStr,
			balStr,
		)

		globalIdx := v.offset + i
		if globalIdx == v.cursor {
			// Colour by type
			style := selectedRowStyle
			b.WriteString(style.Render("> " + row[2:]))
		} else {
			// Colour amount by type
			if tx.Type == "CREDIT" {
				row = fmt.Sprintf("  %-12s %-7s %-30s %15s %15s",
					date,
					tx.Type,
					truncate(tx.Description, 28),
					amtStr,
					balStr,
				)
			}
			b.WriteString(normalRowStyle.Render(row))
		}
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(mutedStyle(fmt.Sprintf("  Showing %d-%d of %d transactions", v.offset+1, end, len(v.transactions))))

	return b.String()
}

func mutedStyle(s string) string {
	return helpStyle.Render(s)
}
