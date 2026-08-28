package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"investec.openbanking.tui/internal/api"
	"investec.openbanking.tui/internal/config"
)

// setupStep is a stage of the guided credentials screen.
type setupStep int

const (
	setupIntro setupStep = iota
	setupCountries
	setupFields
	setupChecking
	setupDone
)

// setupSteps is the number of stages the title counts through. The final
// confirmation is not one of them.
const setupSteps = 4

// The three values Investec issues per country, in the order the Open API
// tab lists them.
const (
	fieldClientID = iota
	fieldClientSecret
	fieldAPIKey
)

// credentialFields describes each value: the label the user sees, the
// suffix of the variable it is stored under, and where to find it.
var credentialFields = []struct {
	label  string
	suffix string
	hint   string
}{
	fieldClientID:     {label: "Client ID", suffix: "_CLIENT_ID", hint: "The long string of letters and numbers shown first."},
	fieldClientSecret: {label: "Client Secret", suffix: "_CLIENT_SECRET", hint: "The shorter secret shown next to it."},
	fieldAPIKey:       {label: "API Key", suffix: "_API_KEY", hint: "The long value listed as x-api-key."},
}

// Bounds for the input box. It follows the window, but stays wide enough to
// be usable on a narrow console and short enough to leave the labels room.
const (
	minInputWidth = 20
	maxInputWidth = 60

	// inputChrome is the prompt plus the app frame's own padding.
	inputChrome = 6
)

// setupChoice is a country the user can tick, with whatever credentials it
// already has.
type setupChoice struct {
	country  config.Country
	selected bool
}

// setupField is one credential being entered: which country it belongs to,
// which of the three values it is, and what has been typed so far.
type setupField struct {
	choice int
	index  int
	value  string
}

// setupCheck is the outcome of trying one country's credentials.
type setupCheck struct {
	name string
	code string
	err  error
}

// setupView is the guided screen that collects credentials and writes them
// to the credentials file.
type setupView struct {
	step     setupStep
	path     string // where the credentials file will be written
	choices  []setupChoice
	cursor   int
	fields   []setupField
	field    int
	input    textinput.Model
	reveal   bool
	checks   []setupCheck
	checking bool
	saving   bool
	err      error

	// canCancel is false on a first run: there is nothing behind this
	// screen to go back to until something has been saved.
	canCancel bool
}

// newSetupView builds the screen, ticking the countries already in use.
func newSetupView(countries []config.Country, path string, canCancel bool, windowWidth int) setupView {
	configured := make(map[string]bool, len(countries))
	for _, c := range countries {
		configured[c.Code] = true
	}

	known := config.KnownCountries(countries)
	choices := make([]setupChoice, len(known))
	for i, c := range known {
		choices[i] = setupChoice{country: c, selected: configured[c.Code]}
	}
	// A first run has nothing configured, so start with one ticked rather
	// than a list the user has to work out they must act on.
	if len(choices) > 0 && !anySelected(choices) {
		choices[0].selected = true
	}

	v := setupView{
		step:      setupIntro,
		path:      path,
		choices:   choices,
		canCancel: canCancel,
		input:     newCredentialInput(),
	}
	v.fitTo(windowWidth)
	return v
}

// newCredentialInput builds the masked field the credentials are typed into.
func newCredentialInput() textinput.Model {
	input := textinput.New()
	input.EchoMode = textinput.EchoPassword
	input.EchoCharacter = '•'
	input.Prompt = "  "

	// Every style has to carry both colours. The app paints its own
	// background, so a run left unstyled shows the terminal profile's
	// through instead -- including the blank space the input pads itself
	// out with.
	input.PromptStyle = normalRowStyle
	input.TextStyle = normalRowStyle
	input.PlaceholderStyle = hintStyle
	input.Cursor.Style = normalRowStyle
	input.Cursor.TextStyle = normalRowStyle

	// A blinking cursor would need its timer messages routed through the
	// root model on every tick, which buys nothing on a form.
	input.Cursor.SetMode(cursor.CursorStatic)
	input.Focus()

	return input
}

// fitTo sizes the input box to the window.
func (v *setupView) fitTo(windowWidth int) {
	width := windowWidth - 2*appHPadding - inputChrome
	if width > maxInputWidth {
		width = maxInputWidth
	}
	if width < minInputWidth {
		width = minInputWidth
	}
	v.input.Width = width
}

// anySelected reports whether at least one country is ticked.
func anySelected(choices []setupChoice) bool {
	for _, c := range choices {
		if c.selected {
			return true
		}
	}
	return false
}

// startFields lays out the questions for the ticked countries, keeping
// anything already entered.
func (v *setupView) startFields() {
	v.fields = nil
	for i, choice := range v.choices {
		if !choice.selected {
			continue
		}
		values := [...]string{
			fieldClientID:     choice.country.ClientID,
			fieldClientSecret: choice.country.ClientSecret,
			fieldAPIKey:       choice.country.APIKey,
		}
		for index := range credentialFields {
			v.fields = append(v.fields, setupField{choice: i, index: index, value: values[index]})
		}
	}

	v.field = 0
	v.step = setupFields
	v.loadField()
}

// loadField shows the current value, hidden again after a reveal.
func (v *setupView) loadField() {
	v.reveal = false
	v.input.EchoMode = textinput.EchoPassword
	v.input.SetValue(v.fields[v.field].value)
	v.input.CursorEnd()
}

// storeField keeps what was typed.
//
// Credentials are pasted far more often than typed, and a value copied out
// of a browser regularly brings a stray space or line break with it.
func (v *setupView) storeField() {
	v.fields[v.field].value = strings.TrimSpace(v.input.Value())
}

// toggleReveal shows or hides the value being entered.
func (v *setupView) toggleReveal() {
	v.reveal = !v.reveal
	v.input.EchoMode = textinput.EchoPassword
	if v.reveal {
		v.input.EchoMode = textinput.EchoNormal
	}
}

// countries returns the ticked countries, carrying the values entered.
func (v setupView) countries() []config.Country {
	countries := make([]config.Country, 0, len(v.choices))

	for i, choice := range v.choices {
		if !choice.selected {
			continue
		}

		country := choice.country
		for _, f := range v.fields {
			if f.choice != i {
				continue
			}
			switch f.index {
			case fieldClientID:
				country.ClientID = f.value
			case fieldClientSecret:
				country.ClientSecret = f.value
			case fieldAPIKey:
				country.APIKey = f.value
			}
		}
		countries = append(countries, country)
	}

	return countries
}

// checksPassed reports whether every country's credentials worked.
func (v setupView) checksPassed() bool {
	for _, check := range v.checks {
		if check.err != nil {
			return false
		}
	}
	return len(v.checks) > 0
}

// checkCredentials tries each country's credentials against Investec, so a
// mistyped key is caught here rather than on the landing page.
func checkCredentials(countries []config.Country) tea.Cmd {
	return func() tea.Msg {
		checks := make([]setupCheck, 0, len(countries))

		for _, c := range countries {
			check := setupCheck{name: c.Name, code: c.Code}
			switch {
			case !c.HasCredentials():
				check.err = fmt.Errorf("still empty: %s", strings.Join(c.MissingCredentials(), ", "))
			default:
				check.err = api.NewClient(c.ClientID, c.ClientSecret, c.APIKey, c.Code).Authenticate()
			}
			checks = append(checks, check)
		}

		return setupCheckedMsg{checks: checks}
	}
}

// saveCredentials writes the credentials file.
func saveCredentials(path string, countries []config.Country) tea.Cmd {
	return func() tea.Msg {
		return setupSavedMsg{err: config.SaveCountries(path, countries)}
	}
}

// --- Key handling ---

func (m Model) handleSetupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.setup.step {

	case setupIntro:
		switch key {
		case "enter":
			m.setup.step = setupCountries
		case "esc":
			return m.cancelSetup()
		}

	case setupCountries:
		switch key {
		case "up", "k":
			if m.setup.cursor > 0 {
				m.setup.cursor--
			}
		case "down", "j":
			if m.setup.cursor < len(m.setup.choices)-1 {
				m.setup.cursor++
			}
		case " ":
			if m.setup.cursor < len(m.setup.choices) {
				m.setup.choices[m.setup.cursor].selected = !m.setup.choices[m.setup.cursor].selected
				m.setup.err = nil
			}
		case "enter":
			if !anySelected(m.setup.choices) {
				m.setup.err = errors.New("tick at least one country to carry on")
				return m, nil
			}
			m.setup.err = nil
			m.setup.startFields()
		case "esc":
			m.setup.step = setupIntro
		}

	case setupFields:
		return m.handleSetupField(msg)

	case setupChecking:
		return m.handleSetupChecking(msg)

	case setupDone:
		if key == "enter" {
			return m.finishSetup()
		}
	}

	return m, nil
}

// handleSetupField drives the credential questions. Anything that is not a
// navigation key belongs to the input box.
func (m Model) handleSetupField(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.setup.fields) == 0 {
		m.setup.step = setupCountries
		return m, nil
	}

	switch msg.String() {
	case "enter", "tab", "down":
		m.setup.storeField()
		if m.setup.field == len(m.setup.fields)-1 {
			return m.startChecking()
		}
		m.setup.field++
		m.setup.loadField()
		return m, nil

	case "shift+tab", "up":
		m.setup.storeField()
		if m.setup.field == 0 {
			m.setup.step = setupCountries
			return m, nil
		}
		m.setup.field--
		m.setup.loadField()
		return m, nil

	case "esc":
		m.setup.storeField()
		m.setup.step = setupCountries
		return m, nil

	case "ctrl+r":
		m.setup.toggleReveal()
		return m, nil

	case "ctrl+s":
		m.setup.storeField()
		return m.saveSetup()
	}

	var cmd tea.Cmd
	m.setup.input, cmd = m.setup.input.Update(msg)
	return m, cmd
}

// handleSetupChecking drives the verification screen. A failed check must
// not trap the user: the credentials can be saved anyway, since the API can
// also be unreachable for reasons that have nothing to do with them.
func (m Model) handleSetupChecking(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.setup.saving {
		return m, nil
	}

	switch msg.String() {
	case "s":
		return m.saveSetup()

	case "enter":
		if m.setup.checking {
			return m, nil
		}
		return m.saveSetup()

	case "r":
		if m.setup.checking {
			return m, nil
		}
		return m.startChecking()

	case "e", "esc":
		if m.setup.checking {
			return m, nil
		}
		m.setup.field = 0
		m.setup.step = setupFields
		m.setup.loadField()
		return m, nil
	}

	return m, nil
}

// startChecking asks Investec whether the credentials work before anything
// is written to disk.
func (m Model) startChecking() (tea.Model, tea.Cmd) {
	m.setup.step = setupChecking
	m.setup.checks = nil
	m.setup.checking = true
	m.setup.err = nil
	return m, checkCredentials(m.setup.countries())
}

// saveSetup writes the credentials file.
func (m Model) saveSetup() (tea.Model, tea.Cmd) {
	m.setup.step = setupChecking
	m.setup.checking = false
	m.setup.saving = true
	m.setup.err = nil
	return m, saveCredentials(m.setup.path, m.setup.countries())
}

// finishSetup hands the countries just entered to the rest of the app. The
// file has been written, but nothing is re-read: these are the same values.
func (m Model) finishSetup() (tea.Model, tea.Cmd) {
	countries := m.setup.countries()
	m.setup = setupView{}
	m.countryList = newCountryView(countries)
	m.state = viewCountry
	return m, nil
}

// cancelSetup leaves the screen without saving, which is only possible when
// there are already credentials to go back to.
func (m Model) cancelSetup() (tea.Model, tea.Cmd) {
	if !m.setup.canCancel {
		return m, nil
	}
	m.setup = setupView{}
	m.state = viewCountry
	return m, nil
}

// openSetup opens the guided screen over the country landing page.
func (m Model) openSetup() (tea.Model, tea.Cmd) {
	path, err := config.TargetEnvFile()
	if err != nil {
		m.countryList.err = err
		return m, nil
	}

	m.setup = newSetupView(m.countryList.countries, path, true, m.width)
	m.state = viewSetup
	return m, nil
}

// --- Rendering ---

// title labels the window, counting the steps so the user can see how much
// is left.
func (v setupView) title() string {
	if v.step == setupDone {
		return "Investec Open Banking — Setup complete"
	}
	return fmt.Sprintf("Investec Open Banking — Setup (step %d of %d)", int(v.step)+1, setupSteps)
}

// help is the key hint line for the current step.
func (v setupView) help() string {
	switch v.step {
	case setupIntro:
		if v.canCancel {
			return "enter start  •  esc cancel  •  ctrl+c quit"
		}
		return "enter start  •  ctrl+c quit"

	case setupCountries:
		return "↑/↓ move  •  space tick  •  enter continue  •  esc back"

	case setupFields:
		return "enter next  •  shift+tab previous  •  ctrl+r show/hide  •  ctrl+s save now  •  esc back"

	case setupChecking:
		if v.saving {
			return "saving..."
		}
		if v.checking {
			return "s skip the check and save  •  ctrl+c quit"
		}
		if v.checksPassed() {
			return "enter save  •  e change something  •  ctrl+c quit"
		}
		return "enter save anyway  •  e correct them  •  r try again  •  ctrl+c quit"

	case setupDone:
		return "enter continue"
	}

	return ""
}

func (v setupView) render() string {
	switch v.step {
	case setupIntro:
		return v.renderIntro()
	case setupCountries:
		return v.renderCountries()
	case setupFields:
		return v.renderFields()
	case setupChecking:
		return v.renderChecking()
	case setupDone:
		return v.renderDone()
	}
	return ""
}

func (v setupView) renderIntro() string {
	lines := []string{
		subtitleStyle.Render("Let's get your Investec keys in place."),
		normalRowStyle.Render("You need three values, and only you can fetch them:"),
		"",
		normalRowStyle.Render("  1. Sign in at https://login.secure.investec.com"),
		normalRowStyle.Render("  2. Open Programmable Banking, then the Open API tab"),
		normalRowStyle.Render("  3. Enrol if you have not already, then copy the Client ID,"),
		normalRowStyle.Render("     the Client Secret and the API Key"),
		"",
		normalRowStyle.Render("You will paste them in on the next screens. They stay hidden,"),
		normalRowStyle.Render("go nowhere but Investec, and are saved only on this computer:"),
		hintStyle.Render("  " + v.path),
	}

	return strings.Join(lines, "\n")
}

func (v setupView) renderCountries() string {
	lines := []string{subtitleStyle.Render("Which countries do you bank in?")}

	for i, choice := range v.choices {
		box := "[ ]"
		if choice.selected {
			box = "[x]"
		}
		status := "not set up yet"
		if choice.country.HasCredentials() {
			status = "already set up"
		}

		row := fmt.Sprintf("  %s %-24s %-6s %s", box, truncate(choice.country.Name, 22), choice.country.Code, status)
		if i == v.cursor {
			lines = append(lines, selectedRowStyle.Render("> "+row[2:]))
		} else {
			lines = append(lines, normalRowStyle.Render(row))
		}
	}

	lines = append(lines,
		"",
		hintStyle.Render("Each country has its own keys, from that country's Investec profile."),
	)

	if v.err != nil {
		lines = append(lines, "", errorStyle.Render(v.err.Error()))
	}

	return strings.Join(lines, "\n")
}

func (v setupView) renderFields() string {
	if len(v.fields) == 0 {
		return errorStyle.Render("No countries were chosen.")
	}

	field := v.fields[v.field]
	country := v.choices[field.choice].country
	spec := credentialFields[field.index]

	reveal := "Hidden as you type. ctrl+r shows it."
	if v.reveal {
		reveal = "Showing the value. ctrl+r hides it again."
	}

	lines := []string{
		subtitleStyle.Render(fmt.Sprintf("%s (%s)", country.Name, country.Code)),
		normalRowStyle.Render(fmt.Sprintf("%s  (%d of %d)", spec.label, v.field+1, len(v.fields))),
		hintStyle.Render(spec.hint),
		"",
		hintStyle.Render("  " + country.Code + spec.suffix),
		v.input.View(),
		"",
		hintStyle.Render(reveal),
	}

	if v.err != nil {
		lines = append(lines, "", errorStyle.Render(fmt.Sprintf("Error: %v", v.err)))
	}

	return strings.Join(lines, "\n")
}

func (v setupView) renderChecking() string {
	lines := []string{subtitleStyle.Render("Checking your keys with Investec")}

	if v.checking {
		for _, country := range v.countries() {
			lines = append(lines, loadingStyle.Render(fmt.Sprintf("  %-30s checking...", label(country.Name, country.Code))))
		}
	}

	for _, check := range v.checks {
		row := fmt.Sprintf("  %-30s ", label(check.name, check.code))
		if check.err == nil {
			lines = append(lines, successStyle.Render(row+"works"))
			continue
		}
		lines = append(lines, errorStyle.Render(row+truncate(check.err.Error(), 60)))
	}

	if v.saving {
		lines = append(lines, "", loadingStyle.Render("Saving..."))
	}

	if v.err != nil {
		lines = append(lines, "", errorStyle.Render(fmt.Sprintf("Could not save: %v", v.err)))
	}

	return strings.Join(lines, "\n")
}

func (v setupView) renderDone() string {
	return strings.Join([]string{
		subtitleStyle.Render("You are all set."),
		normalRowStyle.Render("Your keys are saved, readable only by you, in:"),
		hintStyle.Render("  " + v.path),
		"",
		normalRowStyle.Render("Change them at any time with c on the country page."),
	}, "\n")
}

// label renders a country as "Name (CODE)".
func label(name, code string) string {
	return fmt.Sprintf("%s (%s)", name, code)
}
