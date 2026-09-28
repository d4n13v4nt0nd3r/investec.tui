package tui

import (
	"fmt"
	"strings"

	"investec.openbanking.tui/internal/api"
)

type accountsView struct {
	accounts    []api.Account
	cursor      int
	offset      int // scroll offset for the viewport
	pageSize    int // rows that fit in the current window
	err         error
	searching   bool   // true when the account number search box is active
	searchQuery string // digits typed to filter by account number
	countryCode string // ISO country code, used to select the column layout
}

// Lines the view spends on everything that is not an account row: the app
// frame's padding, the title, the table header and its rule, the footer and
// the help line. The search box costs two more.
const (
	accountsChrome       = 11
	accountsSearchChrome = 13
)

func newAccountsView(countryCode string, windowHeight int) accountsView {
	v := accountsView{countryCode: countryCode}
	v.fitTo(windowHeight)
	return v
}

// fitTo sizes the page to the window.
//
// Without this the list renders every account, which overflows the window as
// soon as there are more accounts than rows -- on Windows, where the app
// cannot resize the console, and on macOS too for anyone with a long list.
func (v *accountsView) fitTo(windowHeight int) {
	chrome := accountsChrome
	if v.searching || v.searchQuery != "" {
		chrome = accountsSearchChrome
	}

	size := windowHeight - chrome
	if size < minPageSize {
		size = minPageSize
	}
	v.pageSize = size
	v.clampOffset()
}

// clampOffset keeps the cursor inside the visible page.
func (v *accountsView) clampOffset() {
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

	// Only the rows that fit are drawn; the rest are reached by scrolling.
	start := v.offset
	if start > len(accounts) {
		start = len(accounts)
	}
	end := start + v.pageSize
	if end > len(accounts) {
		end = len(accounts)
	}
	page := accounts[start:end]

	if v.countryCode == "ZA" {
		header := fmt.Sprintf("  %-30s %-18s %-40s %-12s", "Account Name", "Acc No", "Entity", "Type")
		b.WriteString(headerRowStyle.Render(header))
		b.WriteString("\n")

		for i, acc := range page {
			row := fmt.Sprintf("  %-30s %-18s %-40s %-12s",
				truncate(acc.ReferenceName, 28),
				acc.AccountNumber,
				truncate(acc.DisplayName(), 38),
				truncate(acc.MappedProductType(), 10),
			)

			if start+i == v.cursor {
				b.WriteString(selectedRow(row))
			} else {
				b.WriteString(normalRowStyle.Render(row))
			}
			b.WriteString("\n")
		}
	} else {
		header := fmt.Sprintf("  %-40s %-20s %-35s", "Account Name", "Account Number", "Product")
		b.WriteString(headerRowStyle.Render(header))
		b.WriteString("\n")

		for i, acc := range page {
			row := fmt.Sprintf("  %-40s %-20s %-35s",
				truncate(acc.DisplayName(), 38),
				acc.AccountNumber,
				truncate(acc.Product(), 33),
			)

			if start+i == v.cursor {
				b.WriteString(selectedRow(row))
			} else {
				b.WriteString(normalRowStyle.Render(row))
			}
			b.WriteString("\n")
		}
	}

	// Footer, so it is obvious when the list continues past the window.
	b.WriteString("\n")
	b.WriteString(mutedStyle(fmt.Sprintf("  Showing %d-%d of %d accounts", start+1, end, len(accounts))))

	return b.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
