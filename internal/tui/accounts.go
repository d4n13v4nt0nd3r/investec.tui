package tui

import (
	"fmt"
	"strings"

	"investec.openbanking.tui/internal/api"
)

type accountsView struct {
	accounts    []api.Account
	cursor      int
	err         error
	searching   bool   // true when the account number search box is active
	searchQuery string // digits typed to filter by account number
	countryCode string // ISO country code, used to select the column layout
}

func newAccountsView(countryCode string) accountsView {
	return accountsView{countryCode: countryCode}
}

// visibleAccounts returns the accounts matching the current search query.
// The query matches any substring of the account number, so the last few
// digits are enough to find an account.
func (v accountsView) visibleAccounts() []api.Account {
	if v.searchQuery == "" {
		return v.accounts
	}
	filtered := make([]api.Account, 0, len(v.accounts))
	for _, acc := range v.accounts {
		if strings.Contains(acc.AccountNumber, v.searchQuery) {
			filtered = append(filtered, acc)
		}
	}
	return filtered
}

// accountKey uniquely identifies an account by display name and account
// number, avoiding the delimiter-collision risk of a concatenated string key.
type accountKey struct {
	name   string
	number string
}

// dedupeAccounts removes accounts that share the same display name and
// account number, keeping the first occurrence of each.
func dedupeAccounts(accounts []api.Account) []api.Account {
	seen := make(map[accountKey]struct{}, len(accounts))
	unique := make([]api.Account, 0, len(accounts))
	for _, acc := range accounts {
		key := accountKey{name: acc.DisplayName(), number: acc.AccountNumber}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, acc)
	}
	return unique
}

func (v accountsView) renderTable() string {
	if v.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}

	if len(v.accounts) == 0 {
		return loadingStyle.Render("No accounts found.")
	}

	var b strings.Builder

	if v.searching || v.searchQuery != "" {
		query := v.searchQuery
		if v.searching {
			query += "▎"
		}
		b.WriteString(normalRowStyle.Render(fmt.Sprintf("Search (Account Number): %s", query)))
		b.WriteString("\n\n")
	}

	accounts := v.visibleAccounts()

	if len(accounts) == 0 {
		return b.String() + loadingStyle.Render("No matching accounts.")
	}

	if v.countryCode == "ZA" {
		// Header
		header := fmt.Sprintf("  %-30s %-18s %-40s %-12s", "Account Name", "Acc No", "Entity", "Type")
		b.WriteString(headerRowStyle.Render(header))
		b.WriteString("\n")

		// Rows
		for i, acc := range accounts {
			row := fmt.Sprintf("  %-30s %-18s %-40s %-12s",
				truncate(acc.ReferenceName, 28),
				acc.AccountNumber,
				truncate(acc.DisplayName(), 38),
				acc.MappedProductType(),
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

	// Header
	header := fmt.Sprintf("  %-40s %-20s %-35s", "Account Name", "Account Number", "Product")
	b.WriteString(headerRowStyle.Render(header))
	b.WriteString("\n")

	// Rows
	for i, acc := range accounts {
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
