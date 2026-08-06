package tui

import (
	"fmt"
	"strings"

	"investec.openbanking.tui/internal/api"
)

type balanceView struct {
	account api.Account
	balance *api.Balance
	err     error
	loading bool
}

func newBalanceView(account api.Account) balanceView {
	return balanceView{
		account: account,
		loading: true,
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
