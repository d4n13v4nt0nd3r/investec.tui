package api

// --- Auth ---

type AuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

// --- Accounts ---

type AccountsResponse struct {
	Data  AccountsData `json:"data"`
	Links Links        `json:"links"`
	Meta  Meta         `json:"meta"`
}

type AccountsData struct {
	Accounts []Account `json:"accounts"`
}

type Account struct {
	AccountID    string `json:"accountId"`
	AccountNumber string `json:"accountNumber"`
	AccountName  string `json:"accountName"`
	ReferenceName string `json:"referenceName"`
	ProductName  string `json:"productName"`
	KYCCompliant bool   `json:"kycCompliant"`
	ProfileID    string `json:"profileId"`
	ProfileName  string `json:"profileName"`
}

// --- Balance ---

type BalanceResponse struct {
	Data  Balance `json:"data"`
	Links Links   `json:"links"`
	Meta  Meta    `json:"meta"`
}

type Balance struct {
	AccountID        string  `json:"accountId"`
	CurrentBalance   float64 `json:"currentBalance"`
	AvailableBalance float64 `json:"availableBalance"`
	BudgetBalance    float64 `json:"budgetBalance"`
	StraightBalance  float64 `json:"straightBalance"`
	CashBalance      float64 `json:"cashBalance"`
	Currency         string  `json:"currency"`
}

// --- Transactions ---

type TransactionsResponse struct {
	Data  TransactionsData `json:"data"`
	Links Links            `json:"links"`
	Meta  Meta             `json:"meta"`
}

type TransactionsData struct {
	Transactions []Transaction `json:"transactions"`
}

type Transaction struct {
	AccountID       string  `json:"accountId"`
	Type            string  `json:"type"`
	TransactionType string  `json:"transactionType"`
	Status          string  `json:"status"`
	Description     string  `json:"description"`
	CardNumber      string  `json:"cardNumber"`
	PostedOrder     float64 `json:"postedOrder"`
	PostingDate     string  `json:"postingDate"`
	ValueDate       string  `json:"valueDate"`
	ActionDate      string  `json:"actionDate"`
	TransactionDate string  `json:"transactionDate"`
	Amount          float64 `json:"amount"`
	RunningBalance  float64 `json:"runningBalance"`
	UUID            string  `json:"uuid"`
}

// --- Common ---

type Links struct {
	Self string `json:"self"`
}

type Meta struct {
	TotalPages int `json:"totalPages"`
}
