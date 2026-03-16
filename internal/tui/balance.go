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

	b.WriteString(subtitleStyle.Render(fmt.Sprintf("%s  (%s)", v.account.AccountName, v.account.AccountNumber)))
	b.WriteString("\n")
	b.WriteString(normalRowStyle.Render(fmt.Sprintf("Product: %s", v.account.ProductName)))
	b.WriteString("\n\n")

	cur := v.balance.Currency

	rows := []struct {
		label string
		value float64
	}{
		{"Available Balance", v.balance.AvailableBalance},
		{"Current Balance", v.balance.CurrentBalance},
		{"Budget Balance", v.balance.BudgetBalance},
		{"Straight Balance", v.balance.StraightBalance},
		{"Cash Balance", v.balance.CashBalance},
	}

	for _, r := range rows {
		label := labelStyle.Render(r.label + ":")
		val := FormatAmount(r.value, cur)
		b.WriteString(fmt.Sprintf("%s %s\n", label, valueStyle.Render(val)))
	}

	return b.String()
}
