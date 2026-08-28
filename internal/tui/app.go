package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
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
	viewSetup
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

// setupCheckedMsg carries the result of trying each country's credentials
// during setup.
type setupCheckedMsg struct {
	checks []setupCheck
}

// setupSavedMsg reports whether the credentials file could be written.
type setupSavedMsg struct {
	err error
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
	setup        setupView
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

// NewSetupModel starts the app on the guided credentials screen, which is
// where a first-time user with nothing configured begins. path is the file
// the credentials will be written to.
func NewSetupModel(countries []config.Country, path string) Model {
	m := NewModel(countries)
	m.state = viewSetup
	m.setup = newSetupView(countries, path, false, 0)
	return m
}

// windowTitle labels the terminal window the app runs in.
const windowTitle = "Investec Open Banking"

// Init names the window; nothing else happens until a country is chosen.
func (m Model) Init() tea.Cmd {
	return tea.SetWindowTitle(windowTitle)
}

func connectCountry(country config.Country) tea.Cmd {
	return func() tea.Msg {
		if !country.HasCredentials() {
			return clientReadyMsg{
				country: country,
				err: fmt.Errorf("still to be entered: %s -- press c to set them up",
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

func (m Model) loadPendingTransactions(accountID string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		txns, err := client.GetPendingTransactions(accountID)
		return transactionsLoadedMsg{transactions: txns, err: err}
	}
}

// Update handles messages and key events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// The tables have to fit whatever the console gives us, which on
		// Windows the app cannot change.
		m.accounts.fitTo(msg.Height)
		m.transactions.fitTo(msg.Height)
		m.setup.fitTo(msg.Width)
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
		m.accounts = newAccountsView(msg.country.Code, m.height)
		m.state = viewAccounts
		return m, m.loadAccounts()

	case accountsLoadedMsg:
		m.accounts.accounts = dedupeAccounts(msg.accounts)
		m.accounts.err = msg.err
		m.accounts.cursor = 0
		m.accounts.offset = 0
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

	case setupCheckedMsg:
		m.setup.checking = false
		m.setup.checks = msg.checks
		// Nothing to decide when they all worked, so the file is written
		// without making the user confirm what they can already see.
		if m.setup.checksPassed() {
			return m.saveSetup()
		}
		return m, nil

	case setupSavedMsg:
		m.setup.saving = false
		if msg.err != nil {
			m.setup.err = msg.err
			return m, nil
		}
		m.setup.step = setupDone
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch m.state {

	// --- Guided credentials setup ---
	case viewSetup:
		return m.handleSetupKey(msg)

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
		case "c":
			return m.openSetup()
		case "enter":
			country, ok := m.countryList.selected()
			if !ok {
				return m, nil
			}
			// A country with nothing entered yet has nothing to connect
			// to, so selecting it offers to fill it in instead.
			if !country.HasCredentials() {
				return m.openSetup()
			}
			m.countryList.err = nil
			m.countryList.connecting = true
			return m, connectCountry(country)
		}

	// --- Accounts list ---
	case viewAccounts:
		if m.accounts.searching {
			return m.handleAccountSearch(msg)
		}

		switch key {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc", "backspace":
			if m.accounts.searchQuery != "" {
				m.accounts.searchQuery = ""
				m.accounts.cursor = 0
				m.accounts.offset = 0
				m.accounts.fitTo(m.height)
				return m, nil
			}
			m.state = viewCountry
		case "up", "k":
			if m.accounts.cursor > 0 {
				m.accounts.cursor--
				m.accounts.clampOffset()
			}
		case "down", "j":
			if m.accounts.cursor < len(m.accounts.visibleAccounts())-1 {
				m.accounts.cursor++
				m.accounts.clampOffset()
			}
		case "enter":
			accs := m.accounts.visibleAccounts()
			if len(accs) > 0 {
				acc := accs[m.accounts.cursor]
				m.balance = newBalanceView(acc, m.country.Code)
				m.state = viewBalance
				return m, m.loadBalance(acc.AccountID.String())
			}
		case "r":
			return m, m.loadAccounts()
		case "s":
			m.accounts.searching = true
			// The search box takes two lines from the table.
			m.accounts.fitTo(m.height)
		default:
			// Typing a digit jumps straight into account number search.
			if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
				m.accounts.searching = true
				m.accounts.searchQuery += key
				m.accounts.cursor = 0
				m.accounts.offset = 0
				m.accounts.fitTo(m.height)
			}
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
			m.transactions = newTransactionsView(acc, currency, fromDate, toDate, m.height)
			m.state = viewTransactions
			return m, m.loadTransactions(acc.AccountID.String(), fromDate, toDate)
		case "p":
			acc := m.balance.account
			currency := acc.AccountCurrency
			if m.balance.balance != nil && m.balance.balance.Currency != "" {
				currency = m.balance.balance.Currency
			}
			m.transactions = newPendingTransactionsView(acc, currency, m.height)
			m.state = viewTransactions
			return m, m.loadPendingTransactions(acc.AccountID.String())
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
			if m.transactions.pending {
				break
			}
			// Start editing from-date
			m.transactions.editing = true
			m.transactions.editField = 0
			m.transactions.editBuffer = m.transactions.fromDate
		case "r":
			m.transactions.loading = true
			acc := m.transactions.account
			if m.transactions.pending {
				return m, m.loadPendingTransactions(acc.AccountID.String())
			}
			return m, m.loadTransactions(acc.AccountID.String(), m.transactions.fromDate, m.transactions.toDate)
		case "q", "ctrl+c":
			m.state = viewBalance
		}
	}

	return m, nil
}

func (m Model) handleAccountSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.accounts.searching = false
		m.accounts.searchQuery = ""
		m.accounts.cursor = 0
		m.accounts.offset = 0
		m.accounts.fitTo(m.height)
	case "enter":
		m.accounts.searching = false
		accs := m.accounts.visibleAccounts()
		if len(accs) > 0 {
			acc := accs[m.accounts.cursor]
			m.balance = newBalanceView(acc, m.country.Code)
			m.state = viewBalance
			return m, m.loadBalance(acc.AccountID.String())
		}
	case "backspace":
		if len(m.accounts.searchQuery) > 0 {
			m.accounts.searchQuery = m.accounts.searchQuery[:len(m.accounts.searchQuery)-1]
			m.accounts.cursor = 0
			m.accounts.offset = 0
		} else {
			m.accounts.searching = false
			m.accounts.fitTo(m.height)
		}
	case "up":
		if m.accounts.cursor > 0 {
			m.accounts.cursor--
			m.accounts.clampOffset()
		}
	case "down":
		if m.accounts.cursor < len(m.accounts.visibleAccounts())-1 {
			m.accounts.cursor++
			m.accounts.clampOffset()
		}
	default:
		// Account numbers are numeric, so only accept digits.
		if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
			m.accounts.searchQuery += key
			m.accounts.cursor = 0
			m.accounts.offset = 0
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
	case viewSetup:
		title := titleStyle.Render(m.setup.title())
		help := helpStyle.Render(m.setup.help())
		content = fmt.Sprintf("%s\n%s\n%s", title, m.setup.render(), help)

	case viewCountry:
		title := titleStyle.Render("Investec Open Banking")
		help := helpStyle.Render("↑/↓ navigate  •  enter select  •  c credentials  •  q quit")
		content = fmt.Sprintf("%s\n%s\n%s", title, m.countryList.render(), help)

	case viewAccounts:
		title := titleStyle.Render(fmt.Sprintf("Investec Open Banking — %s", m.country.Name))
		help := helpStyle.Render("↑/↓ navigate  •  enter select  •  s search  •  r refresh  •  esc change country  •  q quit")
		if m.accounts.searching {
			help = helpStyle.Render("Type digits to filter account number  •  ↑/↓ navigate  •  enter select  •  esc cancel")
		}
		content = fmt.Sprintf("%s\n%s\n%s", title, m.accounts.renderTable(), help)

	case viewBalance:
		title := titleStyle.Render(fmt.Sprintf("Account Balance — %s", m.country.Name))
		help := helpStyle.Render("t transactions  •  p pending  •  r refresh  •  esc back")
		content = fmt.Sprintf("%s\n%s\n%s", title, m.balance.render(), help)

	case viewTransactions:
		titleText := fmt.Sprintf("Transactions — %s", m.transactions.account.DisplayName())
		help := helpStyle.Render("↑/↓ navigate  •  f filter dates  •  r refresh  •  esc back")
		if m.transactions.pending {
			titleText = fmt.Sprintf("Pending Transactions — %s", m.transactions.account.DisplayName())
			help = helpStyle.Render("↑/↓ navigate  •  r refresh  •  esc back")
		}
		if m.transactions.editing {
			help = helpStyle.Render("Type date (YYYY-MM-DD)  •  enter confirm  •  esc cancel")
		}
		title := titleStyle.Render(titleText)
		content = fmt.Sprintf("%s\n%s\n%s", title, m.transactions.render(), help)
	}

	// Paint every cell of the window with the app's own background.
	// Terminal.app ignores requests to change the real window background, so
	// filling the viewport ourselves is the only way to look the same across
	// terminal profiles.
	screen := appStyle
	if m.width > 0 {
		// Clip before padding. Width pads short lines but wraps long ones, and
		// a wrapped table row would destroy the column alignment.
		content = clipLines(content, m.width-2*appHPadding)
		screen = screen.Width(m.width)
	}
	if m.height > 0 {
		screen = screen.Height(m.height)
	}

	return screen.Render(content)
}

// clipLines truncates every line to width, leaving short lines untouched.
//
// lipgloss's MaxWidth cannot be used for this. Its render pass also pads every
// line out to the width of the longest one, and because a style with no
// colours produces no escape sequences, that padding arrives as bare spaces.
// Those spaces then show the terminal profile's own background instead of the
// app's, drawing bars across the window.
func clipLines(s string, width int) string {
	if width <= 0 {
		return s
	}

	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, width, "")
	}
	return strings.Join(lines, "\n")
}
