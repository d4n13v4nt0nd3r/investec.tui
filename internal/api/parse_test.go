package api

import "testing"

// Sample payloads below are taken from the published Investec ZA and MU
// OpenAPI documentation.

const zaAccountsBody = `{
  "data": {
    "accounts": [
      {
        "accountId": "172878438321",
        "accountNumber": "10010206147",
        "accountName": "Mr John Doe",
        "referenceName": "My Account",
        "productName": "Private Bank Account",
        "kycCompliant": true,
        "profileId": "10",
        "profileName": "Personal"
      }
    ]
  }
}`

const muAccountsBody = `{
  "data": {
    "accounts": {
      "accounts": [
        {
          "profileId": 9633,
          "profileName": "XYZ PTY LTD",
          "accountId": 5331,
          "accountName": "XYZ PTY LTD USD",
          "accountNumber": "10101010101",
          "accountCurrency": "USD"
        }
      ]
    }
  }
}`

func TestParseAccounts(t *testing.T) {
	za, err := parseAccounts([]byte(zaAccountsBody))
	if err != nil {
		t.Fatalf("ZA accounts: %v", err)
	}
	if len(za) != 1 || za[0].AccountID.String() != "172878438321" {
		t.Fatalf("ZA accounts: unexpected result %+v", za)
	}
	if za[0].Product() != "Private Bank Account" {
		t.Fatalf("ZA product: got %q", za[0].Product())
	}

	mu, err := parseAccounts([]byte(muAccountsBody))
	if err != nil {
		t.Fatalf("MU accounts: %v", err)
	}
	if len(mu) != 1 || mu[0].AccountID.String() != "5331" {
		t.Fatalf("MU accounts: unexpected result %+v", mu)
	}
	if mu[0].AccountCurrency != "USD" {
		t.Fatalf("MU currency: got %q", mu[0].AccountCurrency)
	}
}

func TestMappedProductType(t *testing.T) {
	cases := []struct {
		productName string
		want        string
	}{
		{"Investec Private Business Account", "CHQ"},
		{"Private Bank Account", "CHQ"},
		{"Daily Call Deposit", "CALL"},
		{"Cash Management Account", "CASHMNGMT"},
		{"MoneyFund Tracker", "MMRKT"},
		{"Some Unmapped Product", "Some Unmapped Product"},
		{"", ""},
	}

	for _, c := range cases {
		acc := Account{ProductName: c.productName}
		if got := acc.MappedProductType(); got != c.want {
			t.Errorf("MappedProductType(%q): got %q, want %q", c.productName, got, c.want)
		}
	}
}

const zaBalanceBody = `{
  "data": {
    "accountId": "172878438321",
    "currentBalance": 28857.76,
    "availableBalance": 98857.76,
    "currency": "ZAR"
  }
}`

const muBalanceBody = `{
  "data": {
    "accounts": {
      "balance": {
        "accountNumber": "10101010101",
        "accountType": "CALL DEPOSIT",
        "balance": 1043966.3,
        "availableBalance": 635506.65,
        "encumbrances": 152139.76,
        "currency": "USD",
        "creditInterestRate": 1.1375
      }
    }
  }
}`

func TestParseBalance(t *testing.T) {
	za, err := parseBalance([]byte(zaBalanceBody))
	if err != nil {
		t.Fatalf("ZA balance: %v", err)
	}
	if za.CurrentBalance != 28857.76 || za.Currency != "ZAR" {
		t.Fatalf("ZA balance: unexpected result %+v", za)
	}

	mu, err := parseBalance([]byte(muBalanceBody))
	if err != nil {
		t.Fatalf("MU balance: %v", err)
	}
	if mu.Balance != 1043966.3 || mu.Currency != "USD" {
		t.Fatalf("MU balance: unexpected result %+v", mu)
	}

	rows := mu.Rows()
	if len(rows) < 3 || rows[1].Label != "Current Balance" || rows[1].Value != 1043966.3 {
		t.Fatalf("MU balance rows: unexpected result %+v", rows)
	}
}

const zaTransactionsBody = `{
  "data": {
    "transactions": [
      {
        "accountId": "172878438321",
        "type": "DEBIT",
        "status": "POSTED",
        "description": "MONTHLY SERVICE CHARGE",
        "postingDate": "2024-02-12",
        "transactionDate": "2024-02-11",
        "amount": 535,
        "runningBalance": 9000.76
      }
    ]
  }
}`

const muTransactionsBody = `{
  "data": {
    "accounts": {
      "transactions": [
        {
          "description": "Wages",
          "creditAmount": 0,
          "debitAmount": 535,
          "transactionDate": "2024-02-12",
          "postDate": "2024-02-12",
          "bankReference": "Payment to XYZ LTD",
          "amount": 535,
          "runningBalance": 9000.76
        }
      ]
    }
  }
}`

func TestParseTransactions(t *testing.T) {
	za, err := parseTransactions([]byte(zaTransactionsBody))
	if err != nil {
		t.Fatalf("ZA transactions: %v", err)
	}
	if len(za) != 1 || za[0].Kind() != "DEBIT" || za[0].Date() != "2024-02-11" {
		t.Fatalf("ZA transactions: unexpected result %+v", za)
	}
	if za[0].SignedAmount() != -535 {
		t.Fatalf("ZA signed amount: got %v", za[0].SignedAmount())
	}

	mu, err := parseTransactions([]byte(muTransactionsBody))
	if err != nil {
		t.Fatalf("MU transactions: %v", err)
	}
	if len(mu) != 1 || mu[0].Kind() != "DEBIT" || mu[0].Date() != "2024-02-12" {
		t.Fatalf("MU transactions: unexpected result %+v", mu)
	}
	if mu[0].SignedAmount() != -535 {
		t.Fatalf("MU signed amount: got %v", mu[0].SignedAmount())
	}
}

// The API returns merchant names HTML-escaped, e.g. "GINO&apos;S".
const escapedTransactionsBody = `{
  "data": {
    "transactions": [
      {
        "type": "DEBIT",
        "description": "GINO&apos;S RESTAURANT &amp; BAR STELLENBOSCH ZA",
        "transactionDate": "2026-09-04",
        "amount": 499,
        "runningBalance": 303069.73
      }
    ]
  }
}`

func TestParseTransactionsUnescapesDescription(t *testing.T) {
	txs, err := parseTransactions([]byte(escapedTransactionsBody))
	if err != nil {
		t.Fatalf("transactions: %v", err)
	}
	want := "GINO'S RESTAURANT & BAR STELLENBOSCH ZA"
	if len(txs) != 1 || txs[0].Detail() != want {
		t.Fatalf("description: got %q, want %q", txs[0].Detail(), want)
	}
}

func TestClientPaths(t *testing.T) {
	za := NewClient("id", "secret", "key", "ZA")
	if got := za.path("/accounts"); got != "/za/pb/v1/accounts" {
		t.Fatalf("ZA path: got %q", got)
	}
	if za.RequiresDateRange() {
		t.Fatal("ZA should not require a date range")
	}

	mu := NewClient("id", "secret", "key", "mu")
	if got := mu.path("/accounts/5331/balance"); got != "/mu/pb/v1/accounts/5331/balance" {
		t.Fatalf("MU path: got %q", got)
	}
	if !mu.RequiresDateRange() {
		t.Fatal("MU should require a date range")
	}
}

const zaDocumentsBody = `{
  "data": [
    {
      "documentType": "Statement",
      "documentDate": "2024-01-31"
    },
    {
      "documentType": "TaxCertificate",
      "documentDate": "2024-02-28"
    }
  ],
  "links": { "self": "https://openapi.investec.com/za/pb/v1/accounts/1/documents" },
  "meta": { "totalPages": 1 }
}`

const muDocumentsBody = `{
  "availableDocuments": {
    "accountNumber": "0173123456500",
    "documentInformation": [
      {
        "documentDate": "2025-01-31",
        "documentType": "Statement"
      }
    ]
  }
}`

func TestParseDocuments(t *testing.T) {
	za, err := parseDocuments([]byte(zaDocumentsBody))
	if err != nil {
		t.Fatalf("ZA documents: %v", err)
	}
	if len(za) != 2 || za[0].DocumentType != "Statement" || za[0].DocumentDate != "2024-01-31" {
		t.Fatalf("ZA documents: unexpected result %+v", za)
	}
	if za[1].DocumentType != "TaxCertificate" {
		t.Fatalf("ZA second doc: %+v", za[1])
	}

	mu, err := parseDocuments([]byte(muDocumentsBody))
	if err != nil {
		t.Fatalf("MU documents: %v", err)
	}
	if len(mu) != 1 || mu[0].DocumentType != "Statement" || mu[0].DocumentDate != "2025-01-31" {
		t.Fatalf("MU documents: unexpected result %+v", mu)
	}

	empty, err := parseDocuments([]byte(`{"data":[]}`))
	if err != nil {
		t.Fatalf("empty: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("empty: got %+v", empty)
	}
}

func TestSortDocumentsByDateDesc(t *testing.T) {
	docs := []Document{
		{DocumentType: "Statement", DocumentDate: "2025-10-01"},
		{DocumentType: "Statement", DocumentDate: "2026-09-01"},
		{DocumentType: "TaxCertificate", DocumentDate: "2026-09-01"},
		{DocumentType: "Statement", DocumentDate: "2026-01-01"},
	}
	sortDocumentsByDateDesc(docs)
	want := []string{"2026-09-01/Statement", "2026-09-01/TaxCertificate", "2026-01-01/Statement", "2025-10-01/Statement"}
	for i, d := range docs {
		got := d.DocumentDate + "/" + d.DocumentType
		if got != want[i] {
			t.Fatalf("index %d: got %s, want %s (full %+v)", i, got, want[i], docs)
		}
	}
}
