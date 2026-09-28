package tui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"investec.openbanking.tui/internal/api"
)

type balanceView struct {
	account     api.Account
	balance     *api.Balance
	err         error
	loading     bool
	countryCode string // ISO country code, e.g. "ZA"

	// The framed look also shows the last historyDays of transactions, as a
	// sparkline of the balance and a list of the most recent.
	history        []api.Transaction
	historyErr     error
	historyLoading bool
}

// historyDays matches the date range api.DefaultDateRange asks for.
const historyDays = 90

func newBalanceView(account api.Account, countryCode string) balanceView {
	return balanceView{
		account:        account,
		loading:        true,
		countryCode:    countryCode,
		historyLoading: framedLook,
	}
}

func (v balanceView) render() string {
	if v.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}

	if v.loading || v.balance == nil {
		return loadingStyle.Render("Loading balance...")
	}

	var b strings.Builder

	b.WriteString(subtitleStyle.Render(fmt.Sprintf("%s  (%s)", v.account.DisplayName(), v.account.AccountNumber)))
	b.WriteString("\n")

	if v.countryCode == "ZA" {
		b.WriteString(normalRowStyle.Render(fmt.Sprintf("AccountID: %s", v.account.AccountID.String())))
		b.WriteString("\n")
	}

	product := v.account.Product()
	if product == "" {
		product = v.balance.AccountType
	}
	if product != "" {
		b.WriteString(normalRowStyle.Render(fmt.Sprintf("Product: %s", product)))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	cur := v.balance.Currency
	if cur == "" {
		cur = v.account.AccountCurrency
	}

	for _, r := range v.balance.Rows() {
		label := labelStyle.Render(r.Label + ":")
		val := FormatAmount(r.Value, cur)
		b.WriteString(fmt.Sprintf("%s %s\n", label, valueStyle.Render(val)))
	}

	// Interest rates are percentages, not amounts.
	rates := []struct {
		label string
		value float64
	}{
		{"Credit Interest Rate", v.balance.CreditInterestRate},
		{"Debit Interest Rate", v.balance.DebitInterestRate},
	}
	for _, r := range rates {
		if r.value == 0 {
			continue
		}
		label := labelStyle.Render(r.label + ":")
		b.WriteString(fmt.Sprintf("%s %s\n", label, valueStyle.Render(FormatRate(r.value))))
	}

	return b.String()
}

// renderFramed is the balance screen in the framed look: the headline
// balance in big digits, its 90-day trend, the other figures, and as many of
// the latest transactions as fit the rows left under a divider.
func (v balanceView) renderFramed(width, rows int, today time.Time) string {
	if v.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	if v.loading || v.balance == nil {
		return loadingStyle.Render("Loading balance...")
	}

	cur := v.balance.Currency
	if cur == "" {
		cur = v.account.AccountCurrency
	}

	// Rows always starts with the available balance, then the current one.
	// Where the available figure is not reported, the current one leads.
	figures := v.balance.Rows()
	current := figures[1].Value
	headline, others := figures[0], figures[1:]
	if headline.Value == 0 && current != 0 {
		headline = figures[1]
		others = append([]api.AmountRow{figures[0]}, figures[2:]...)
	}

	var lines []string

	var details []string
	if product := v.account.Product(); product != "" {
		details = append(details, product)
	} else if v.balance.AccountType != "" {
		details = append(details, v.balance.AccountType)
	}
	if id := v.account.AccountID.String(); v.countryCode == "ZA" && id != "" {
		details = append(details, "id "+id)
	}
	if len(details) > 0 {
		lines = append(lines, hintStyle.Render("  "+strings.Join(details, " · ")))
	}
	lines = append(lines, "")

	big := bigNumber(FormatAmount(headline.Value, ""))
	if 2+len([]rune(big[0]))+2+len(cur) <= width {
		for i, row := range big {
			line := normalRowStyle.Render("  ") + bigNumberStyle.Render(row)
			if i == len(big)-1 {
				line += hintStyle.Render("  " + cur)
			}
			lines = append(lines, line)
		}
	} else {
		lines = append(lines, normalRowStyle.Render("  ")+valueStyle.Render(FormatAmount(headline.Value, cur)))
	}
	lines = append(lines, hintStyle.Render("  "+strings.ToLower(headline.Label)), "")

	switch {
	case v.historyLoading:
		lines = append(lines, loadingStyle.Render(fmt.Sprintf("  Loading %d days of history...", historyDays)))
	case v.historyErr != nil:
		lines = append(lines, hintStyle.Render(fmt.Sprintf("  %d-day history unavailable", historyDays)))
	default:
		values := dailyBalances(current, v.history, historyDays, today)
		low, high := values[0], values[0]
		for _, b := range values {
			low = min(low, b)
			high = max(high, b)
		}
		spark := sparkline(values, min(60, width-50))
		lines = append(lines, normalRowStyle.Render("  ")+frameBorderStyle.Render(spark)+
			hintStyle.Render(fmt.Sprintf("  %dd  low %s · high %s", historyDays, FormatAmount(low, ""), FormatAmount(high, ""))))
	}
	lines = append(lines, "")

	for _, r := range others {
		lines = append(lines, normalRowStyle.Render("  ")+labelStyle.Render(r.Label)+" "+valueStyle.Render(FormatAmount(r.Value, cur)))
	}
	rates := []struct {
		label string
		value float64
	}{
		{"Credit Interest Rate", v.balance.CreditInterestRate},
		{"Debit Interest Rate", v.balance.DebitInterestRate},
	}
	for _, r := range rates {
		if r.value != 0 {
			lines = append(lines, normalRowStyle.Render("  ")+labelStyle.Render(r.label)+" "+valueStyle.Render(FormatRate(r.value)))
		}
	}

	// The divider takes a row of its own.
	if recent := latestTransactions(v.history, rows-len(lines)-1); len(recent) > 0 {
		lines = append(lines, frameDivider+"recent")
		for _, tx := range recent {
			lines = append(lines, recentRow(tx))
		}
	}

	return strings.Join(lines, "\n")
}

// latestTransactions returns up to n transactions, newest first.
func latestTransactions(txns []api.Transaction, n int) []api.Transaction {
	sorted := slices.Clone(txns)
	slices.SortStableFunc(sorted, func(a, b api.Transaction) int {
		return strings.Compare(b.Date(), a.Date())
	})
	return sorted[:max(0, min(n, len(sorted)))]
}

// recentRow is one line of the balance screen's recent list.
func recentRow(tx api.Transaction) string {
	date := tx.Date()
	if len(date) == 10 {
		date = date[5:] // the year is plain from context
	}
	amount := tx.SignedAmount()
	flow := flowStyle(normalRowStyle, amount)
	return normalRowStyle.Render(fmt.Sprintf("  %-5s  ", date)) +
		flow.Render(fmt.Sprintf("%-5s", flowLabel(tx))) +
		normalRowStyle.Render(fmt.Sprintf(" %-40s ", truncate(tx.Detail(), 38))) +
		flow.Render(fmt.Sprintf("%15s", signedAmount(amount)))
}
