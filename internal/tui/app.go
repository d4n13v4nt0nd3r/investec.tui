package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"investec.openbanking.tui/internal/api"
	"investec.openbanking.tui/internal/config"
	"investec.openbanking.tui/internal/export"
)

// View states
type viewState int

const (
	viewCountry viewState = iota
	viewAccounts
	viewBalance
	viewTransactions
	viewDocuments
	viewSetup
	viewSaveAs
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

// historyLoadedMsg carries the transactions behind the balance screen's
// sparkline, for the account they were fetched for.
type historyLoadedMsg struct {
	accountID    string
	transactions []api.Transaction
	err          error
}

type transactionsLoadedMsg struct {
	transactions []api.Transaction
	err          error
}

type documentsLoadedMsg struct {
	documents []api.Document
	err       error
}

type fileSavedMsg struct {
	path string
	kind saveAsKind
	err  error
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
	returnState  viewState // view under save-as overlay
	countryList  countryView
	accounts     accountsView
	balance      balanceView
	transactions transactionsView
	documents    documentsView
	saveAs       saveAsView
	setup        setupView
	lastSaveDir  string
	// Pending document download target while save-as is open.
	pendingDoc api.Document
	width      int
	height     int
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

// Init names the window and, when following a desktop theme, starts watching
// it. Nothing else happens until a country is chosen.
func (m Model) Init() tea.Cmd {
	if themeFile != "" {
		return tea.Batch(tea.SetWindowTitle(windowTitle), watchTheme(themeFile, themeModTime))
	}
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

// loadHistory fetches the last historyDays of transactions for the balance
// screen. Only the Omarchy look draws them, so elsewhere nothing is fetched.
func (m Model) loadHistory(accountID string) tea.Cmd {
	if !omarchyLook {
		return nil
	}
	client := m.client
	return func() tea.Msg {
		fromDate, toDate := api.DefaultDateRange()
		txns, err := client.GetTransactions(accountID, fromDate, toDate)
		return historyLoadedMsg{accountID: accountID, transactions: txns, err: err}
	}
}

// openBalance shows the balance screen for acc and starts loading it.
func (m Model) openBalance(acc api.Account) (tea.Model, tea.Cmd) {
	m.balance = newBalanceView(acc, m.country.Code)
	m.state = viewBalance
	id := acc.AccountID.String()
	return m, tea.Batch(m.loadBalance(id), m.loadHistory(id))
}

// layoutHeight is the window height the views size their tables against.
// Their chrome counts are for the classic layout, and the frame uses less.
func (m Model) layoutHeight() int {
	if omarchyLook {
		return m.height + frameRowsSaved
	}
	return m.height
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

func (m Model) loadDocuments(accountID, fromDate, toDate string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		docs, err := client.GetDocuments(accountID, fromDate, toDate)
		return documentsLoadedMsg{documents: docs, err: err}
	}
}

func (m Model) savePDF(accountID string, doc api.Document, path string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		data, err := client.GetDocument(accountID, doc.DocumentType, doc.DocumentDate)
		if err != nil {
			return fileSavedMsg{kind: saveAsPDF, err: err}
		}
		if err := export.WriteFile(path, data); err != nil {
			return fileSavedMsg{kind: saveAsPDF, path: path, err: err}
		}
		return fileSavedMsg{kind: saveAsPDF, path: path}
	}
}

func (m Model) saveCSV(path string) tea.Cmd {
	// Export what the user is looking at: the filtered list when searching.
	txns := m.transactions.visibleTransactions()
	currency := m.transactions.currency
	return func() tea.Msg {
		data, err := export.TransactionsToCSV(txns, currency)
		if err != nil {
			return fileSavedMsg{kind: saveAsCSV, err: err}
		}
		if err := export.WriteFile(path, data); err != nil {
			return fileSavedMsg{kind: saveAsCSV, path: path, err: err}
		}
		return fileSavedMsg{kind: saveAsCSV, path: path}
	}
}

func (m Model) openSaveAs(kind saveAsKind, accountNumber string) Model {
	m.returnState = m.state
	m.saveAs = newSaveAsView(kind, accountNumber, m.lastSaveDir, m.layoutHeight())
	m.state = viewSaveAs
	return m
}

// Update handles messages and key events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// The tables have to fit whatever the console gives us, which on
		// Windows the app cannot change.
		m.accounts.fitTo(m.layoutHeight())
		m.transactions.fitTo(m.layoutHeight())
		m.documents.fitTo(m.layoutHeight())
		m.saveAs.fitTo(m.layoutHeight())
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
		m.accounts = newAccountsView(msg.country.Code, m.layoutHeight())
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

	case historyLoadedMsg:
		// Ignore a slow reply for an account the user has already left.
		if msg.accountID == m.balance.account.AccountID.String() {
			m.balance.history = msg.transactions
			m.balance.historyErr = msg.err
			m.balance.historyLoading = false
		}
		return m, nil

	case transactionsLoadedMsg:
		m.transactions.transactions = msg.transactions
		m.transactions.err = msg.err
		m.transactions.loading = false
		return m, nil

	case documentsLoadedMsg:
		m.documents.documents = msg.documents
		m.documents.err = msg.err
		m.documents.loading = false
		m.documents.cursor = 0
		m.documents.offset = 0
		return m, nil

	case fileSavedMsg:
		if msg.kind == saveAsPDF {
			m.documents.saving = false
			if msg.err != nil {
				m.documents.status = fmt.Sprintf("Save failed: %v", msg.err)
			} else {
				m.documents.status = "Saved: " + msg.path
			}
			return m, nil
		}
		m.transactions.saving = false
		if msg.err != nil {
			m.transactions.status = fmt.Sprintf("Export failed: %v", msg.err)
		} else {
			m.transactions.status = "Saved: " + msg.path
		}
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

	case themeCheckedMsg:
		if msg.p != nil {
			applyPalette(*msg.p)
			styleInput(&m.setup.input)
			styleInput(&m.saveAs.input)
		}
		return m, watchTheme(themeFile, msg.modTime)

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
				m.accounts.fitTo(m.layoutHeight())
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
				return m.openBalance(accs[m.accounts.cursor])
			}
		case "r":
			return m, m.loadAccounts()
		case "s":
			m.accounts.searching = true
			// The search box takes two lines from the table.
			m.accounts.fitTo(m.layoutHeight())
		default:
			// Typing a digit jumps straight into account number search.
			if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
				m.accounts.searching = true
				m.accounts.searchQuery += key
				m.accounts.cursor = 0
				m.accounts.offset = 0
				m.accounts.fitTo(m.layoutHeight())
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
			m.transactions = newTransactionsView(acc, currency, fromDate, toDate, m.layoutHeight())
			m.state = viewTransactions
			return m, m.loadTransactions(acc.AccountID.String(), fromDate, toDate)
		case "p":
			acc := m.balance.account
			currency := acc.AccountCurrency
			if m.balance.balance != nil && m.balance.balance.Currency != "" {
				currency = m.balance.balance.Currency
			}
			m.transactions = newPendingTransactionsView(acc, currency, m.layoutHeight())
			m.state = viewTransactions
			return m, m.loadPendingTransactions(acc.AccountID.String())
		case "d":
			acc := m.balance.account
			fromDate, toDate := api.DefaultDocumentDateRange()
			m.documents = newDocumentsView(acc, fromDate, toDate, m.layoutHeight())
			m.state = viewDocuments
			return m, m.loadDocuments(acc.AccountID.String(), fromDate, toDate)
		case "r":
			m.balance.loading = true
			m.balance.historyLoading = omarchyLook
			id := m.balance.account.AccountID.String()
			return m, tea.Batch(m.loadBalance(id), m.loadHistory(id))
		case "q", "ctrl+c":
			m.state = viewAccounts
		}

	// --- Transactions view ---
	case viewTransactions:
		if m.transactions.editing {
			return m.handleTransactionEditing(msg)
		}
		if m.transactions.searching {
			return m.handleTransactionSearch(msg)
		}

		switch key {
		case "esc", "backspace":
			if m.transactions.searchQuery != "" {
				m.transactions.searchQuery = ""
				m.transactions.cursor = 0
				m.transactions.offset = 0
				m.transactions.fitTo(m.layoutHeight())
				return m, nil
			}
			m.state = viewBalance
		case "up", "k":
			if m.transactions.cursor > 0 {
				m.transactions.cursor--
				m.transactions.clampOffset()
			}
		case "down", "j":
			if m.transactions.cursor < len(m.transactions.visibleTransactions())-1 {
				m.transactions.cursor++
				m.transactions.clampOffset()
			}
		case "s":
			m.transactions.searching = true
			m.transactions.fitTo(m.layoutHeight())
		case "f":
			if m.transactions.pending {
				break
			}
			m.transactions.editing = true
			m.transactions.editField = 0
			m.transactions.editBuffer = m.transactions.fromDate
		case "e":
			visible := m.transactions.visibleTransactions()
			if m.transactions.pending || m.transactions.loading || len(visible) == 0 {
				m.transactions.status = "Nothing to export."
				break
			}
			m.transactions.status = ""
			m.transactions.err = nil
			m = m.openSaveAs(saveAsCSV, m.transactions.account.AccountNumber)
			return m, nil
		case "r":
			m.transactions.loading = true
			m.transactions.searchQuery = ""
			m.transactions.searching = false
			m.transactions.cursor = 0
			m.transactions.offset = 0
			acc := m.transactions.account
			if m.transactions.pending {
				return m, m.loadPendingTransactions(acc.AccountID.String())
			}
			return m, m.loadTransactions(acc.AccountID.String(), m.transactions.fromDate, m.transactions.toDate)
		case "q", "ctrl+c":
			m.state = viewBalance
		}

	// --- Documents view ---
	case viewDocuments:
		if m.documents.editing {
			return m.handleDocumentEditing(msg)
		}
		switch key {
		case "esc", "backspace":
			m.state = viewBalance
		case "up", "k":
			if m.documents.cursor > 0 {
				m.documents.cursor--
				if m.documents.cursor < m.documents.offset {
					m.documents.offset = m.documents.cursor
				}
			}
		case "down", "j":
			if m.documents.cursor < len(m.documents.documents)-1 {
				m.documents.cursor++
				if m.documents.cursor >= m.documents.offset+m.documents.pageSize {
					m.documents.offset = m.documents.cursor - m.documents.pageSize + 1
				}
			}
		case "f":
			m.documents.editing = true
			m.documents.editField = 0
			m.documents.editBuffer = m.documents.fromDate
		case "r":
			m.documents.loading = true
			m.documents.status = ""
			acc := m.documents.account
			return m, m.loadDocuments(acc.AccountID.String(), m.documents.fromDate, m.documents.toDate)
		case "enter":
			doc, ok := m.documents.selected()
			if !ok || m.documents.loading || m.documents.saving {
				return m, nil
			}
			m.pendingDoc = doc
			m.documents.status = ""
			m.documents.err = nil
			m = m.openSaveAs(saveAsPDF, m.documents.account.AccountNumber)
			return m, nil
		case "q", "ctrl+c":
			m.state = viewBalance
		}

	// --- Save-as overlay ---
	case viewSaveAs:
		updated, confirmed, cmd := m.saveAs.update(msg)
		m.saveAs = updated
		if !m.saveAs.active {
			m.state = m.returnState
			return m, nil
		}
		if confirmed {
			path := m.saveAs.targetPath()
			if m.saveAs.pending != "" {
				path = m.saveAs.pending
			}
			m.saveAs.active = false
			m.state = m.returnState
			m.lastSaveDir = m.saveAs.dir
			if m.saveAs.kind == saveAsPDF {
				m.documents.saving = true
				accID := m.documents.account.AccountID.String()
				return m, m.savePDF(accID, m.pendingDoc, path)
			}
			m.transactions.saving = true
			return m, m.saveCSV(path)
		}
		return m, cmd
	}

	return m, nil
}

func (m Model) handleTransactionSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.transactions.searching = false
		m.transactions.searchQuery = ""
		m.transactions.cursor = 0
		m.transactions.offset = 0
		m.transactions.fitTo(m.layoutHeight())
	case "enter":
		m.transactions.searching = false
		m.transactions.fitTo(m.layoutHeight())
	case "backspace":
		if len(m.transactions.searchQuery) > 0 {
			// Drop the last rune, not the last byte, so multi-byte input is safe.
			q := []rune(m.transactions.searchQuery)
			m.transactions.searchQuery = string(q[:len(q)-1])
			m.transactions.cursor = 0
			m.transactions.offset = 0
		} else {
			m.transactions.searching = false
			m.transactions.fitTo(m.layoutHeight())
		}
	case "up":
		if m.transactions.cursor > 0 {
			m.transactions.cursor--
			m.transactions.clampOffset()
		}
	case "down":
		if m.transactions.cursor < len(m.transactions.visibleTransactions())-1 {
			m.transactions.cursor++
			m.transactions.clampOffset()
		}
	default:
		// Accept printable single-rune keys (letters, digits, space, punctuation).
		if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
			m.transactions.searchQuery += key
			m.transactions.cursor = 0
			m.transactions.offset = 0
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
		m.accounts.fitTo(m.layoutHeight())
	case "enter":
		m.accounts.searching = false
		accs := m.accounts.visibleAccounts()
		if len(accs) > 0 {
			return m.openBalance(accs[m.accounts.cursor])
		}
	case "backspace":
		if len(m.accounts.searchQuery) > 0 {
			m.accounts.searchQuery = m.accounts.searchQuery[:len(m.accounts.searchQuery)-1]
			m.accounts.cursor = 0
			m.accounts.offset = 0
		} else {
			m.accounts.searching = false
			m.accounts.fitTo(m.layoutHeight())
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
			// Reload with new dates; drop any active search against the old set.
			m.transactions.loading = true
			m.transactions.searchQuery = ""
			m.transactions.searching = false
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

func (m Model) handleDocumentEditing(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "enter":
		if m.documents.editField == 0 {
			m.documents.fromDate = m.documents.editBuffer
			m.documents.editField = 1
			m.documents.editBuffer = m.documents.toDate
		} else {
			m.documents.toDate = m.documents.editBuffer
			m.documents.editing = false
			m.documents.loading = true
			m.documents.cursor = 0
			m.documents.offset = 0
			m.documents.status = ""
			acc := m.documents.account
			return m, m.loadDocuments(acc.AccountID.String(), m.documents.fromDate, m.documents.toDate)
		}
	case "esc":
		m.documents.editing = false
	case "backspace":
		if len(m.documents.editBuffer) > 0 {
			m.documents.editBuffer = m.documents.editBuffer[:len(m.documents.editBuffer)-1]
		}
	default:
		if len(key) == 1 && (key[0] >= '0' && key[0] <= '9' || key[0] == '-') {
			m.documents.editBuffer += key
		}
	}

	return m, nil
}

// screen is what the current view puts in the window, before the layout
// arranges it.
type screen struct {
	title   string // the classic title bar
	section string // the view's name on the Omarchy frame
	context string // what the view is showing, at the frame's top right
	body    string
	help    string
}

// screen gathers the current view's parts.
func (m Model) screen() screen {
	switch m.state {
	case viewSetup:
		return screen{
			title:   m.setup.title(),
			section: "setup",
			context: m.setup.progress(),
			body:    m.setup.render(),
			help:    m.setup.help(),
		}

	case viewCountry:
		return screen{
			title:   "Investec Open Banking",
			section: "countries",
			body:    m.countryList.render(),
			help:    "↑/↓ navigate  •  enter select  •  c credentials  •  q quit",
		}

	case viewAccounts:
		help := "↑/↓ navigate  •  enter select  •  s search  •  r refresh  •  esc change country  •  q quit"
		if m.accounts.searching {
			help = "Type digits to filter account number  •  ↑/↓ navigate  •  enter select  •  esc cancel"
		}
		return screen{
			title:   fmt.Sprintf("Investec Open Banking — %s", m.country.Name),
			section: "accounts",
			context: m.country.Name,
			body:    m.accounts.renderTable(),
			help:    help,
		}

	case viewBalance:
		body := m.balance.render()
		if omarchyLook {
			body = m.balance.renderFramed(m.width-2*appHPadding, m.height-frameChromeRows, time.Now())
		}
		acc := m.balance.account
		return screen{
			title:   fmt.Sprintf("Account Balance — %s", m.country.Name),
			section: "balance",
			context: accountContext(acc),
			body:    body,
			help:    "t transactions  •  p pending  •  d documents  •  r refresh  •  esc back",
		}

	case viewTransactions:
		acc := m.transactions.account
		s := screen{
			title:   fmt.Sprintf("Transactions — %s", acc.DisplayName()),
			section: "transactions",
			context: accountContext(acc),
			body:    m.transactions.render(),
			help:    "↑/↓ navigate  •  s search  •  f filter dates  •  e export csv  •  r refresh  •  esc back",
		}
		if m.transactions.pending {
			s.title = fmt.Sprintf("Pending Transactions — %s", acc.DisplayName())
			s.section = "pending"
			s.help = "↑/↓ navigate  •  s search  •  r refresh  •  esc back"
		}
		if m.transactions.editing {
			s.help = "Type date (YYYY-MM-DD)  •  enter confirm  •  esc cancel"
		}
		if m.transactions.searching {
			s.help = "Type to filter description/amount  •  ↑/↓ navigate  •  enter done  •  esc clear"
		}
		return s

	case viewDocuments:
		acc := m.documents.account
		help := "↑/↓ navigate  •  enter download  •  f filter dates  •  r refresh  •  esc back"
		if m.documents.editing {
			help = "Type date (YYYY-MM-DD)  •  enter confirm  •  esc cancel"
		}
		return screen{
			title:   fmt.Sprintf("Documents — %s", acc.DisplayName()),
			section: "documents",
			context: accountContext(acc),
			body:    m.documents.render(),
			help:    help,
		}

	case viewSaveAs:
		return screen{
			title:   "Save As",
			section: "save as",
			body:    m.saveAs.render(),
			help:    m.saveAs.help(),
		}
	}
	return screen{}
}

// accountContext names an account for the frame's top right.
func accountContext(acc api.Account) string {
	var parts []string
	for _, p := range []string{acc.DisplayName(), acc.AccountNumber} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, " · ")
}

// View renders the current state.
func (m Model) View() string {
	s := m.screen()

	if omarchyLook && m.width > 0 && m.height > 0 {
		return renderFrame(frame{section: s.section, context: s.context, legend: s.help}, s.body, m.width, m.height)
	}

	content := fmt.Sprintf("%s\n%s\n%s", titleStyle.Render(s.title), s.body, helpStyle.Render(s.help))

	// Paint every cell of the window with the app's own background.
	// Terminal.app ignores requests to change the real window background, so
	// filling the viewport ourselves is the only way to look the same across
	// terminal profiles.
	window := appStyle
	if m.width > 0 {
		// Clip before padding. Width pads short lines but wraps long ones, and
		// a wrapped table row would destroy the column alignment.
		content = clipLines(content, m.width-2*appHPadding)
		window = window.Width(m.width)
	}
	if m.height > 0 {
		window = window.Height(m.height)
	}

	return window.Render(content)
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
