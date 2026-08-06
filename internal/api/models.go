package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// --- Auth ---

type AuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

// FlexString accepts either a JSON string or a JSON number and stores it as a
// string. ZA returns string IDs, MU returns numeric IDs.
type FlexString string

func (f *FlexString) UnmarshalJSON(b []byte) error {
	trimmed := bytes.TrimSpace(b)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		*f = ""
		return nil
	}

	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return err
		}
		*f = FlexString(s)
		return nil
	}

	*f = FlexString(trimmed)
	return nil
}

func (f FlexString) String() string { return string(f) }

// --- Accounts ---

type Account struct {
	AccountID       FlexString `json:"accountId"`
	AccountNumber   string     `json:"accountNumber"`
	AccountName     string     `json:"accountName"`
	AccountCurrency string     `json:"accountCurrency"` // MU only
	ReferenceName   string     `json:"referenceName"`
	ProductName     string     `json:"productName"` // ZA only
	KYCCompliant    bool       `json:"kycCompliant"`
	ProfileID       FlexString `json:"profileId"`
	ProfileName     string     `json:"profileName"`
}

// DisplayName falls back to the profile name when the account has no name.
func (a Account) DisplayName() string {
	if a.AccountName != "" {
		return a.AccountName
	}
	return a.ProfileName
}

// Product returns the product name (ZA) or, failing that, the account currency
// and profile name (MU).
func (a Account) Product() string {
	if a.ProductName != "" {
		return a.ProductName
	}
	if a.AccountCurrency != "" && a.ProfileName != "" {
		return fmt.Sprintf("%s - %s", a.AccountCurrency, a.ProfileName)
	}
	if a.AccountCurrency != "" {
		return a.AccountCurrency
	}
	return a.ProfileName
}

// parseAccounts handles both the ZA shape (data.accounts[]) and the MU shape
// (data.accounts.accounts[]).
func parseAccounts(body []byte) ([]Account, error) {
	var env struct {
		Data struct {
			Accounts json.RawMessage `json:"accounts"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("decoding accounts: %w", err)
	}

	raw := bytes.TrimSpace(env.Data.Accounts)
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	if raw[0] == '[' {
		var accounts []Account
		if err := json.Unmarshal(raw, &accounts); err != nil {
			return nil, fmt.Errorf("decoding accounts: %w", err)
		}
		return accounts, nil
	}

	var nested struct {
		Accounts []Account `json:"accounts"`
	}
	if err := json.Unmarshal(raw, &nested); err != nil {
		return nil, fmt.Errorf("decoding accounts: %w", err)
	}
	return nested.Accounts, nil
}

// --- Balance ---

type Balance struct {
	AccountID        FlexString `json:"accountId"`
	CurrentBalance   float64    `json:"currentBalance"`
	AvailableBalance float64    `json:"availableBalance"`
	BudgetBalance    float64    `json:"budgetBalance"`
	StraightBalance  float64    `json:"straightBalance"`
	CashBalance      float64    `json:"cashBalance"`
	Currency         string     `json:"currency"`

	// MU fields
	AccountNumber         string  `json:"accountNumber"`
	AccountType           string  `json:"accountType"`
	AccountShortName      string  `json:"accountShortName"`
	Balance               float64 `json:"balance"`
	Encumbrances          float64 `json:"encumbrances"`
	DebitInterestRate     float64 `json:"debitInterestRate"`
	CreditInterestRate    float64 `json:"creditInterestRate"`
	CreditInterestAccrued float64 `json:"creditInterestAccrued"`
	DebitInterestAccrued  float64 `json:"debitInterestAccrued"`
	OverdraftLimit        float64 `json:"overDraftLimit"`
}

// AmountRow is a single labelled monetary value on the balance card.
type AmountRow struct {
	Label string
	Value float64
}

// Rows returns the balance figures relevant to this response, covering both
// the ZA and MU field sets.
func (b Balance) Rows() []AmountRow {
	current := b.CurrentBalance
	if current == 0 {
		current = b.Balance
	}

	rows := []AmountRow{
		{"Available Balance", b.AvailableBalance},
		{"Current Balance", current},
	}

	optional := []AmountRow{
		{"Budget Balance", b.BudgetBalance},
		{"Straight Balance", b.StraightBalance},
		{"Cash Balance", b.CashBalance},
		{"Encumbrances", b.Encumbrances},
		{"Overdraft Limit", b.OverdraftLimit},
		{"Credit Interest Accrued", b.CreditInterestAccrued},
		{"Debit Interest Accrued", b.DebitInterestAccrued},
	}
	for _, r := range optional {
		if r.Value != 0 {
			rows = append(rows, r)
		}
	}

	return rows
}

// parseBalance handles both the ZA shape (data) and the MU shape
// (data.accounts.balance).
func parseBalance(body []byte) (*Balance, error) {
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("decoding balance: %w", err)
	}

	raw := bytes.TrimSpace(env.Data)
	if len(raw) == 0 || string(raw) == "null" {
		return nil, fmt.Errorf("decoding balance: empty response")
	}

	var nested struct {
		Accounts struct {
			Balance *Balance `json:"balance"`
		} `json:"accounts"`
	}
	if err := json.Unmarshal(raw, &nested); err == nil && nested.Accounts.Balance != nil {
		return nested.Accounts.Balance, nil
	}

	var balance Balance
	if err := json.Unmarshal(raw, &balance); err != nil {
		return nil, fmt.Errorf("decoding balance: %w", err)
	}
	return &balance, nil
}

// --- Transactions ---

type Transaction struct {
	AccountID       FlexString `json:"accountId"`
	Type            string     `json:"type"`
	TransactionType string     `json:"transactionType"`
	Status          string     `json:"status"`
	Description     string     `json:"description"`
	CardNumber      string     `json:"cardNumber"`
	PostedOrder     float64    `json:"postedOrder"`
	PostingDate     string     `json:"postingDate"`
	ValueDate       string     `json:"valueDate"`
	ActionDate      string     `json:"actionDate"`
	TransactionDate string     `json:"transactionDate"`
	Amount          float64    `json:"amount"`
	RunningBalance  float64    `json:"runningBalance"`
	UUID            string     `json:"uuid"`

	// MU fields
	PostDate      string  `json:"postDate"`
	BankReference string  `json:"bankReference"`
	CreditAmount  float64 `json:"creditAmount"`
	DebitAmount   float64 `json:"debitAmount"`
}

// Kind returns CREDIT or DEBIT. ZA returns it directly; MU is derived from the
// credit/debit amounts.
func (t Transaction) Kind() string {
	if t.Type != "" {
		return strings.ToUpper(t.Type)
	}
	if t.CreditAmount > 0 {
		return "CREDIT"
	}
	if t.DebitAmount > 0 {
		return "DEBIT"
	}
	return ""
}

// Date returns the best available date for the transaction (YYYY-MM-DD).
func (t Transaction) Date() string {
	for _, d := range []string{t.TransactionDate, t.PostingDate, t.PostDate, t.ActionDate} {
		if d == "" {
			continue
		}
		if len(d) > 10 {
			return d[:10]
		}
		return d
	}
	return ""
}

// SignedAmount returns the amount as a positive credit or a negative debit.
func (t Transaction) SignedAmount() float64 {
	if t.CreditAmount > 0 {
		return t.CreditAmount
	}
	if t.DebitAmount > 0 {
		return -t.DebitAmount
	}
	if t.Kind() == "DEBIT" && t.Amount > 0 {
		return -t.Amount
	}
	return t.Amount
}

// Detail returns the description, falling back to the bank reference.
func (t Transaction) Detail() string {
	if t.Description != "" {
		return t.Description
	}
	return t.BankReference
}

// parseTransactions handles both the ZA shape (data.transactions) and the MU
// shape (data.accounts.transactions).
func parseTransactions(body []byte) ([]Transaction, error) {
	var env struct {
		Data struct {
			Transactions []Transaction `json:"transactions"`
			Accounts     struct {
				Transactions []Transaction `json:"transactions"`
			} `json:"accounts"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("decoding transactions: %w", err)
	}

	if len(env.Data.Transactions) > 0 {
		return env.Data.Transactions, nil
	}
	return env.Data.Accounts.Transactions, nil
}

// --- Common ---

type Links struct {
	Self string `json:"self"`
}

type Meta struct {
	TotalPages int `json:"totalPages"`
}
