package tui

import (
	"strings"
	"testing"
	"time"

	"investec.openbanking.tui/internal/api"
)

func TestBalance_ShowsAccountID(t *testing.T) {
	// Account mappings verified against the Mauritius API.
	accounts := []api.Account{
		{AccountNumber: "0173198186505", AccountID: "22530"},
		{AccountNumber: "0173198186506", AccountID: "22531"},
	}
	for _, country := range []string{"ZA", "MU"} {
		for _, account := range accounts {
			t.Run(country+"/"+account.AccountID.String(), func(t *testing.T) {
				v := balanceView{account: account, balance: &api.Balance{}, countryCode: country}
				classic := ansiStrip(v.render())
				if !strings.Contains(classic, "AccountID: "+account.AccountID.String()) {
					t.Errorf("classic balance does not show account ID:\n%s", classic)
				}
				if !strings.Contains(classic, account.AccountNumber) {
					t.Errorf("classic balance no longer shows account number:\n%s", classic)
				}
				framed := ansiStrip(v.renderFramed(100, 30, time.Now()))
				if !strings.Contains(framed, "id "+account.AccountID.String()) {
					t.Errorf("framed balance does not show account ID:\n%s", framed)
				}
			})
		}
	}
}

func TestBalance_OmitsMissingAccountID(t *testing.T) {
	for _, country := range []string{"ZA", "MU"} {
		t.Run(country, func(t *testing.T) {
			v := balanceView{balance: &api.Balance{}, countryCode: country}
			if classic := ansiStrip(v.render()); strings.Contains(classic, "AccountID:") {
				t.Errorf("classic balance shows an empty account ID:\n%s", classic)
			}
			if framed := ansiStrip(v.renderFramed(100, 30, time.Now())); strings.Contains(framed, "id ") {
				t.Errorf("framed balance shows an empty account ID:\n%s", framed)
			}
		})
	}
}
