package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"investec.openbanking.tui/internal/api"
)

// View states
type viewState int

const (
	viewAccounts viewState = iota
	viewBalance
	viewTransactions
)

// Messages for async operations
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
	state        viewState
	accounts     accountsView
	balance      balanceView
	transactions transactionsView
	width        int
	height       int
}

// NewModel creates the initial app model.
func NewModel(client *api.Client) Model {
	return Model{
		client:   client,
		state:    viewAccounts,
		accounts: newAccountsView(),
	}
}

// Init starts by loading accounts.
func (m Model) Init() tea.Cmd {
	return m.loadAccounts()
}

func (m Model) loadAccounts() tea.Cmd {
	return func() tea.Msg {
		accounts, err := m.client.GetAccounts()
		return accountsLoadedMsg{accounts: accounts, err: err}
	}
}

func (m Model) loadBalance(accountID string) tea.Cmd {
	return func() tea.Msg {
		balance, err := m.client.GetBalance(accountID)
		return balanceLoadedMsg{balance: balance, err: err}
	}
}

func (m Model) loadTransactions(accountID, fromDate, toDate string) tea.Cmd {
	return func() tea.Msg {
		txns, err := m.client.GetTransactions(accountID, fromDate, toDate)
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
		// Global quit
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			if m.state == viewAccounts && !m.transactions.editing {
				return m, tea.Quit
			}
		}

		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch m.state {

	// --- Accounts list ---
	case viewAccounts:
		switch key {
		case "q", "ctrl+c":
			return m, tea.Quit
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
				return m, m.loadBalance(acc.AccountID)
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
			currency := ""
			if m.balance.balance != nil {
				currency = m.balance.balance.Currency
			}
			m.transactions = newTransactionsView(acc, currency)
			m.state = viewTransactions
			return m, m.loadTransactions(acc.AccountID, "", "")
		case "r":
			return m, m.loadBalance(m.balance.account.AccountID)
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
			return m, m.loadTransactions(acc.AccountID, m.transactions.fromDate, m.transactions.toDate)
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
			return m, m.loadTransactions(acc.AccountID, m.transactions.fromDate, m.transactions.toDate)
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
	case viewAccounts:
		title := titleStyle.Render("Investec Open Banking")
		help := helpStyle.Render("↑/↓ navigate  •  enter select  •  r refresh  •  q quit")
		content = fmt.Sprintf("%s\n%s\n%s", title, m.accounts.renderTable(), help)

	case viewBalance:
		title := titleStyle.Render("Account Balance")
		help := helpStyle.Render("t transactions  •  r refresh  •  esc back")
		content = fmt.Sprintf("%s\n%s\n%s", title, m.balance.render(), help)

	case viewTransactions:
		title := titleStyle.Render(fmt.Sprintf("Transactions — %s", m.transactions.account.AccountName))
		help := helpStyle.Render("↑/↓ navigate  •  f filter dates  •  r refresh  •  esc back")
		if m.transactions.editing {
			help = helpStyle.Render("Type date (YYYY-MM-DD)  •  enter confirm  •  esc cancel")
		}
		content = fmt.Sprintf("%s\n%s\n%s", title, m.transactions.render(), help)
	}

	return appStyle.Render(content)
}
