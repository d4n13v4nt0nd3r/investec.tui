package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const baseURL = "https://openapi.investec.com"

// defaultDateRangeDays is used for countries where the transactions endpoint
// requires an explicit date range.
const defaultDateRangeDays = 90

// Client handles authentication and API requests to Investec Open Banking.
type Client struct {
	clientID     string
	clientSecret string
	apiKey       string
	countryCode  string // ISO country code, e.g. "ZA" or "MU"
	accessToken  string
	tokenExpiry  time.Time
	httpClient   *http.Client
}

// NewClient creates a new Investec API client for the given country code.
func NewClient(clientID, clientSecret, apiKey, countryCode string) *Client {
	code := strings.ToUpper(strings.TrimSpace(countryCode))
	if code == "" {
		code = "ZA"
	}

	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		apiKey:       apiKey,
		countryCode:  code,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// CountryCode returns the country this client is bound to.
func (c *Client) CountryCode() string { return c.countryCode }

// path builds a country-scoped private banking path, e.g. /mu/pb/v1/accounts.
func (c *Client) path(suffix string) string {
	return fmt.Sprintf("/%s/pb/v1%s", strings.ToLower(c.countryCode), suffix)
}

// RequiresDateRange reports whether the transactions endpoint for this country
// requires explicit fromDate and toDate parameters. Only ZA defaults them
// server-side.
func (c *Client) RequiresDateRange() bool {
	return c.countryCode != "ZA"
}

// DefaultDateRange returns a sensible fromDate/toDate pair (YYYY-MM-DD) for
// countries that require an explicit date range.
func DefaultDateRange() (fromDate, toDate string) {
	now := time.Now()
	return now.AddDate(0, 0, -defaultDateRangeDays).Format("2006-01-02"), now.Format("2006-01-02")
}

// Authenticate obtains an access token using client credentials.
func (c *Client) Authenticate() error {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", baseURL+"/identity/v2/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("creating auth request: %w", err)
	}

	creds := base64.StdEncoding.EncodeToString([]byte(c.clientID + ":" + c.clientSecret))
	req.Header.Set("Authorization", "Basic "+creds)
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("auth failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("decoding auth response: %w", err)
	}

	c.accessToken = authResp.AccessToken
	// Refresh 60s before actual expiry to avoid edge cases
	c.tokenExpiry = time.Now().Add(time.Duration(authResp.ExpiresIn-60) * time.Second)
	return nil
}

// ensureAuth re-authenticates if the token is expired or about to expire.
func (c *Client) ensureAuth() error {
	if c.accessToken == "" || time.Now().After(c.tokenExpiry) {
		return c.Authenticate()
	}
	return nil
}

// doGet performs an authenticated GET request expecting a JSON body.
func (c *Client) doGet(path string) ([]byte, error) {
	return c.doGetWithAccept(path, "application/json")
}

// doGetBinary performs an authenticated GET expecting a binary body (PDF).
func (c *Client) doGetBinary(path string) ([]byte, error) {
	return c.doGetWithAccept(path, "application/pdf, application/octet-stream, */*")
}

func (c *Client) doGetWithAccept(path, accept string) ([]byte, error) {
	if err := c.ensureAuth(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", accept)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// GetAccounts returns all accounts for the authenticated user.
func (c *Client) GetAccounts() ([]Account, error) {
	body, err := c.doGet(c.path("/accounts"))
	if err != nil {
		return nil, fmt.Errorf("get accounts: %w", err)
	}
	return parseAccounts(body)
}

// GetBalance returns the balance for a specific account.
func (c *Client) GetBalance(accountID string) (*Balance, error) {
	body, err := c.doGet(c.path(fmt.Sprintf("/accounts/%s/balance", accountID)))
	if err != nil {
		return nil, fmt.Errorf("get balance: %w", err)
	}
	return parseBalance(body)
}

// GetPendingTransactions returns pending transactions for a specific account.
func (c *Client) GetPendingTransactions(accountID string) ([]Transaction, error) {
	body, err := c.doGet(c.path(fmt.Sprintf("/accounts/%s/pending-transactions", accountID)))
	if err != nil {
		return nil, fmt.Errorf("get pending transactions: %w", err)
	}
	return parseTransactions(body)
}

// GetTransactions returns transactions for a specific account within a date range.
// fromDate and toDate should be ISO 8601 format (YYYY-MM-DD). Pass empty strings
// for defaults; countries that require an explicit range get a default window.
func (c *Client) GetTransactions(accountID, fromDate, toDate string) ([]Transaction, error) {
	if c.RequiresDateRange() && (fromDate == "" || toDate == "") {
		defaultFrom, defaultTo := DefaultDateRange()
		if fromDate == "" {
			fromDate = defaultFrom
		}
		if toDate == "" {
			toDate = defaultTo
		}
	}

	path := c.path(fmt.Sprintf("/accounts/%s/transactions", accountID))

	params := url.Values{}
	if fromDate != "" {
		params.Set("fromDate", fromDate)
	}
	if toDate != "" {
		params.Set("toDate", toDate)
	}
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	body, err := c.doGet(path)
	if err != nil {
		return nil, fmt.Errorf("get transactions: %w", err)
	}
	return parseTransactions(body)
}

// defaultDocumentRangeDays is used when listing statements; monthly docs need
// a longer window than the transactions default.
const defaultDocumentRangeDays = 365

// DefaultDocumentDateRange returns fromDate/toDate (YYYY-MM-DD) covering about
// the last year of statements.
func DefaultDocumentDateRange() (fromDate, toDate string) {
	now := time.Now()
	return now.AddDate(0, 0, -defaultDocumentRangeDays).Format("2006-01-02"), now.Format("2006-01-02")
}

// GetDocuments lists available PDF documents (statements, tax certificates)
// for an account within a date range.
func (c *Client) GetDocuments(accountID, fromDate, toDate string) ([]Document, error) {
	if fromDate == "" || toDate == "" {
		defaultFrom, defaultTo := DefaultDocumentDateRange()
		if fromDate == "" {
			fromDate = defaultFrom
		}
		if toDate == "" {
			toDate = defaultTo
		}
	}

	path := c.path(fmt.Sprintf("/accounts/%s/documents", accountID))
	params := url.Values{}
	params.Set("fromDate", fromDate)
	params.Set("toDate", toDate)
	path += "?" + params.Encode()

	body, err := c.doGet(path)
	if err != nil {
		return nil, fmt.Errorf("get documents: %w", err)
	}
	docs, err := parseDocuments(body)
	if err != nil {
		return nil, err
	}
	sortDocumentsByDateDesc(docs)
	return docs, nil
}

// sortDocumentsByDateDesc orders statements newest-first. ISO dates sort as strings.
func sortDocumentsByDateDesc(docs []Document) {
	sort.SliceStable(docs, func(i, j int) bool {
		if docs[i].DocumentDate != docs[j].DocumentDate {
			return docs[i].DocumentDate > docs[j].DocumentDate
		}
		return docs[i].DocumentType < docs[j].DocumentType
	})
}

// GetDocument downloads a single document as raw bytes (PDF).
func (c *Client) GetDocument(accountID, documentType, documentDate string) ([]byte, error) {
	path := c.path(fmt.Sprintf("/accounts/%s/document/%s/%s",
		url.PathEscape(accountID),
		url.PathEscape(documentType),
		url.PathEscape(documentDate),
	))
	body, err := c.doGetBinary(path)
	if err != nil {
		return nil, fmt.Errorf("get document: %w", err)
	}
	return body, nil
}
