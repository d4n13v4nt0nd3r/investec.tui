package tui

import (
	"fmt"
	"strings"

	"investec.openbanking.tui/internal/api"
)

type accountsView struct {
	accounts []api.Account
	cursor   int
	err      error
}

func newAccountsView() accountsView {
	return accountsView{}
}

func (v accountsView) renderTable() string {
	if v.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}

	if len(v.accounts) == 0 {
		return loadingStyle.Render("No accounts found.")
	}

	var b strings.Builder

	// Header
	header := fmt.Sprintf("  %-40s %-20s %-35s", "Account Name", "Account Number", "Product")
	b.WriteString(headerRowStyle.Render(header))
	b.WriteString("\n")

	// Rows
	for i, acc := range v.accounts {
		row := fmt.Sprintf("  %-40s %-20s %-35s",
			truncate(acc.DisplayName(), 38),
			acc.AccountNumber,
			truncate(acc.Product(), 33),
		)

		if i == v.cursor {
			b.WriteString(selectedRowStyle.Render("> " + row[2:]))
		} else {
			b.WriteString(normalRowStyle.Render(row))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
