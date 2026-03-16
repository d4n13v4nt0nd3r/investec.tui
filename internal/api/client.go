package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const baseURL = "https://openapi.investec.com"

// Client handles authentication and API requests to Investec Open Banking.
type Client struct {
	clientID     string
	clientSecret string
	apiKey       string
	accessToken  string
	tokenExpiry  time.Time
	httpClient   *http.Client
}

// NewClient creates a new Investec API client.
func NewClient(clientID, clientSecret, apiKey string) *Client {
	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		apiKey:       apiKey,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
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

// doGet performs an authenticated GET request.
func (c *Client) doGet(path string) ([]byte, error) {
	if err := c.ensureAuth(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/json")

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
	body, err := c.doGet("/za/pb/v1/accounts")
	if err != nil {
		return nil, fmt.Errorf("get accounts: %w", err)
	}

	var resp AccountsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding accounts: %w", err)
	}
	return resp.Data.Accounts, nil
}

// GetBalance returns the balance for a specific account.
func (c *Client) GetBalance(accountID string) (*Balance, error) {
	path := fmt.Sprintf("/za/pb/v1/accounts/%s/balance", accountID)
	body, err := c.doGet(path)
	if err != nil {
		return nil, fmt.Errorf("get balance: %w", err)
	}

	var resp BalanceResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding balance: %w", err)
	}
	return &resp.Data, nil
}

// GetTransactions returns transactions for a specific account within a date range.
// fromDate and toDate should be ISO 8601 format (YYYY-MM-DD). Pass empty strings for defaults.
func (c *Client) GetTransactions(accountID, fromDate, toDate string) ([]Transaction, error) {
	path := fmt.Sprintf("/za/pb/v1/accounts/%s/transactions", accountID)

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

	var resp TransactionsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding transactions: %w", err)
	}
	return resp.Data.Transactions, nil
}
