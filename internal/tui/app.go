package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"investec.openbanking.tui/internal/api"
	"investec.openbanking.tui/internal/config"
)

// View states
type viewState int

const (
	viewCountry viewState = iota
	viewAccounts
	viewBalance
	viewTransactions
)

// Messages for async operations
type clientReadyMsg struct {
	client  *api.Client
	country config.Country
	err     error
}

type accountsLoadedMsg struct {
	accounts []api.Account
	err      error
}

type balanceLoadedMsg struct {
	balance *api.Balance
	err     error
}

type transactionsLoadedMsg struct {
	transactions []api.Transaction
	err          error
}

// Model is the root Bubble Tea model.
type Model struct {
	client       *api.Client
	country      config.Country
	state        viewState
	countryList  countryView
	accounts     accountsView
	balance      balanceView
	transactions transactionsView
	width        int
	height       int
}

// NewModel creates the initial app model, starting on the country landing page.
func NewModel(countries []config.Country) Model {
	return Model{
		state:       viewCountry,
		countryList: newCountryView(countries),
	}
}

// Init does nothing until a country has been chosen.
func (m Model) Init() tea.Cmd {
	return nil
}

func connectCountry(country config.Country) tea.Cmd {
	return func() tea.Msg {
		if !country.HasCredentials() {
			return clientReadyMsg{
				country: country,
				err: fmt.Errorf("missing credentials in .env: %s",
					strings.Join(country.MissingCredentials(), ", ")),
			}
		}

		client := api.NewClient(country.ClientID, country.ClientSecret, country.APIKey, country.Code)
		if err := client.Authenticate(); err != nil {
			return clientReadyMsg{country: country, err: err}
		}
		return clientReadyMsg{client: client, country: country}
	}
}

func (m Model) loadAccounts() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		accounts, err := client.GetAccounts()
		return accountsLoadedMsg{accounts: accounts, err: err}
	}
}

func (m Model) loadBalance(accountID string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		balance, err := client.GetBalance(accountID)
		return balanceLoadedMsg{balance: balance, err: err}
	}
}

func (m Model) loadTransactions(accountID, fromDate, toDate string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		txns, err := client.GetTransactions(accountID, fromDate, toDate)
		return transactionsLoadedMsg{transactions: txns, err: err}
	}
}

// Update handles messages and key events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case clientReadyMsg:
		m.countryList.connecting = false
		if msg.err != nil {
			m.countryList.err = msg.err
			return m, nil
		}
		m.countryList.err = nil
		m.client = msg.client
		m.country = msg.country
		m.accounts = newAccountsView()
		m.state = viewAccounts
		return m, m.loadAccounts()

	case accountsLoadedMsg:
		m.accounts.accounts = msg.accounts
		m.accounts.err = msg.err
		return m, nil

	case balanceLoadedMsg:
		m.balance.balance = msg.balance
		m.balance.err = msg.err
		m.balance.loading = false
		return m, nil

	case transactionsLoadedMsg:
		m.transactions.transactions = msg.transactions
		m.transactions.err = msg.err
		m.transactions.loading = false
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch m.state {

	// --- Country landing page ---
	case viewCountry:
		if m.countryList.connecting {
			if key == "ctrl+c" {
				return m, tea.Quit
			}
			return m, nil
		}

		switch key {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.countryList.cursor > 0 {
				m.countryList.cursor--
			}
		case "down", "j":
			if m.countryList.cursor < len(m.countryList.countries)-1 {
				m.countryList.cursor++
			}
		case "enter":
			country, ok := m.countryList.selected()
			if !ok {
				return m, nil
			}
			m.countryList.err = nil
			m.countryList.connecting = true
			return m, connectCountry(country)
		}

	// --- Accounts list ---
	case viewAccounts:
		switch key {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc", "backspace":
			m.state = viewCountry
		case "up", "k":
			if m.accounts.cursor > 0 {
				m.accounts.cursor--
			}
		case "down", "j":
			if m.accounts.cursor < len(m.accounts.accounts)-1 {
				m.accounts.cursor++
			}
		case "enter":
			if len(m.accounts.accounts) > 0 {
				acc := m.accounts.accounts[m.accounts.cursor]
				m.balance = newBalanceView(acc)
				m.state = viewBalance
				return m, m.loadBalance(acc.AccountID.String())
			}
		case "r":
			return m, m.loadAccounts()
		}

	// --- Balance view ---
	case viewBalance:
		switch key {
		case "esc", "backspace":
			m.state = viewAccounts
		case "t":
			acc := m.balance.account
			currency := acc.AccountCurrency
			if m.balance.balance != nil && m.balance.balance.Currency != "" {
				currency = m.balance.balance.Currency
			}
			fromDate, toDate := "", ""
			if m.client.RequiresDateRange() {
				fromDate, toDate = api.DefaultDateRange()
			}
			m.transactions = newTransactionsView(acc, currency, fromDate, toDate)
			m.state = viewTransactions
			return m, m.loadTransactions(acc.AccountID.String(), fromDate, toDate)
		case "r":
			m.balance.loading = true
			return m, m.loadBalance(m.balance.account.AccountID.String())
		case "q", "ctrl+c":
			m.state = viewAccounts
		}

	// --- Transactions view ---
	case viewTransactions:
		if m.transactions.editing {
			return m.handleTransactionEditing(msg)
		}

		switch key {
		case "esc", "backspace":
			m.state = viewBalance
		case "up", "k":
			if m.transactions.cursor > 0 {
				m.transactions.cursor--
				// Scroll up if needed
				if m.transactions.cursor < m.transactions.offset {
					m.transactions.offset = m.transactions.cursor
				}
			}
		case "down", "j":
			if m.transactions.cursor < len(m.transactions.transactions)-1 {
				m.transactions.cursor++
				// Scroll down if needed
				if m.transactions.cursor >= m.transactions.offset+m.transactions.pageSize {
					m.transactions.offset = m.transactions.cursor - m.transactions.pageSize + 1
				}
			}
		case "f":
			// Start editing from-date
			m.transactions.editing = true
			m.transactions.editField = 0
			m.transactions.editBuffer = m.transactions.fromDate
		case "r":
			m.transactions.loading = true
			acc := m.transactions.account
			return m, m.loadTransactions(acc.AccountID.String(), m.transactions.fromDate, m.transactions.toDate)
		case "q", "ctrl+c":
			m.state = viewBalance
		}
	}

	return m, nil
}

func (m Model) handleTransactionEditing(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "enter":
		// Save current field
		if m.transactions.editField == 0 {
			m.transactions.fromDate = m.transactions.editBuffer
			// Move to toDate
			m.transactions.editField = 1
			m.transactions.editBuffer = m.transactions.toDate
		} else {
			m.transactions.toDate = m.transactions.editBuffer
			m.transactions.editing = false
			// Reload with new dates
			m.transactions.loading = true
			m.transactions.cursor = 0
			m.transactions.offset = 0
			acc := m.transactions.account
			return m, m.loadTransactions(acc.AccountID.String(), m.transactions.fromDate, m.transactions.toDate)
		}
	case "esc":
		m.transactions.editing = false
	case "backspace":
		if len(m.transactions.editBuffer) > 0 {
			m.transactions.editBuffer = m.transactions.editBuffer[:len(m.transactions.editBuffer)-1]
		}
	default:
		// Only allow date characters
		if len(key) == 1 && (key[0] >= '0' && key[0] <= '9' || key[0] == '-') {
			m.transactions.editBuffer += key
		}
	}

	return m, nil
}

// View renders the current state.
func (m Model) View() string {
	var content string

	switch m.state {
	case viewCountry:
		title := titleStyle.Render("Investec Open Banking")
		help := helpStyle.Render("↑/↓ navigate  •  enter select  •  q quit")
		content = fmt.Sprintf("%s\n%s\n%s", title, m.countryList.render(), help)

	case viewAccounts:
		title := titleStyle.Render(fmt.Sprintf("Investec Open Banking — %s", m.country.Name))
		help := helpStyle.Render("↑/↓ navigate  •  enter select  •  r refresh  •  esc change country  •  q quit")
		content = fmt.Sprintf("%s\n%s\n%s", title, m.accounts.renderTable(), help)

	case viewBalance:
		title := titleStyle.Render(fmt.Sprintf("Account Balance — %s", m.country.Name))
		help := helpStyle.Render("t transactions  •  r refresh  •  esc back")
		content = fmt.Sprintf("%s\n%s\n%s", title, m.balance.render(), help)

	case viewTransactions:
		title := titleStyle.Render(fmt.Sprintf("Transactions — %s", m.transactions.account.DisplayName()))
		help := helpStyle.Render("↑/↓ navigate  •  f filter dates  •  r refresh  •  esc back")
		if m.transactions.editing {
			help = helpStyle.Render("Type date (YYYY-MM-DD)  •  enter confirm  •  esc cancel")
		}
		content = fmt.Sprintf("%s\n%s\n%s", title, m.transactions.render(), help)
	}

	return appStyle.Render(content)
}
