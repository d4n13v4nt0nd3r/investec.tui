package tui

import (
	"testing"

	"investec.openbanking.tui/internal/api"
)

func TestFuzzyMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		query  string
		target string
		want   bool
	}{
		{name: "empty query", query: "", target: "anything", want: true},
		{name: "exact substring", query: "mart", target: "WALMART STORE", want: true},
		{name: "case insensitive", query: "WalMart", target: "walmart store", want: true},
		{name: "subsequence typo", query: "wlmrt", target: "WALMART", want: true},
		{name: "no match", query: "xyz", target: "WALMART", want: false},
		{name: "order matters", query: "tram", target: "WALMART", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := fuzzyMatch(tt.query, tt.target); got != tt.want {
				t.Errorf("fuzzyMatch(%q, %q) = %v, want %v", tt.query, tt.target, got, tt.want)
			}
		})
	}
}

func TestAmountMatchesQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		amount float64
		query  string
		want   bool
	}{
		{name: "empty query", amount: 12.34, query: "", want: true},
		{name: "plain digits", amount: 1234.56, query: "1234", want: true},
		{name: "with space thousands", amount: 1234.56, query: "1 234", want: true},
		{name: "decimal fragment", amount: 1234.56, query: "234.56", want: true},
		{name: "negative sign", amount: -50.00, query: "-50", want: true},
		{name: "absolute without sign", amount: -50.00, query: "50", want: true},
		{name: "no match", amount: 12.34, query: "999", want: false},
		{name: "letters only", amount: 12.34, query: "abc", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := amountMatchesQuery(tt.amount, tt.query); got != tt.want {
				t.Errorf("amountMatchesQuery(%v, %q) = %v, want %v", tt.amount, tt.query, got, tt.want)
			}
		})
	}
}

func TestVisibleTransactions_FiltersDescriptionAndAmount(t *testing.T) {
	t.Parallel()

	v := transactionsView{
		transactions: []api.Transaction{
			{Description: "CARD PURCHASE WALMART", Type: "DEBIT", Amount: 150.00},
			{Description: "SALARY ACME CORP", Type: "CREDIT", Amount: 25000.00},
			{BankReference: "SPOTIFY PREMIUM", DebitAmount: 79.99},
			{Description: "PETROL STATION", Type: "DEBIT", Amount: 850.50},
		},
	}

	t.Run("empty query returns all", func(t *testing.T) {
		t.Parallel()
		if got := len(v.visibleTransactions()); got != 4 {
			t.Fatalf("got %d, want 4", got)
		}
	})

	t.Run("description substring", func(t *testing.T) {
		t.Parallel()
		filtered := transactionsView{transactions: v.transactions, searchQuery: "walmart"}.visibleTransactions()
		if len(filtered) != 1 || filtered[0].Description != "CARD PURCHASE WALMART" {
			t.Fatalf("got %#v", filtered)
		}
	})

	t.Run("description fuzzy", func(t *testing.T) {
		t.Parallel()
		filtered := transactionsView{transactions: v.transactions, searchQuery: "sptfy"}.visibleTransactions()
		if len(filtered) != 1 || filtered[0].Detail() != "SPOTIFY PREMIUM" {
			t.Fatalf("got %#v", filtered)
		}
	})

	t.Run("amount match", func(t *testing.T) {
		t.Parallel()
		filtered := transactionsView{transactions: v.transactions, searchQuery: "850.50"}.visibleTransactions()
		if len(filtered) != 1 || filtered[0].Description != "PETROL STATION" {
			t.Fatalf("got %#v", filtered)
		}
	})

	t.Run("no matches", func(t *testing.T) {
		t.Parallel()
		filtered := transactionsView{transactions: v.transactions, searchQuery: "zzzz"}.visibleTransactions()
		if len(filtered) != 0 {
			t.Fatalf("got %d, want 0", len(filtered))
		}
	})
}
