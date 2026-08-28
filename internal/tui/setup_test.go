package tui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
	"investec.openbanking.tui/internal/config"
)

// firstRunModel is the app as a new user meets it: nothing configured, and
// the setup screen already on show.
func firstRunModel(t *testing.T, path string) Model {
	t.Helper()

	m := NewSetupModel(nil, path)
	return resize(m, 120, 35)
}

// resize gives the model a window, as Bubble Tea does on start-up.
func resize(m Model, width, height int) Model {
	next, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return next.(Model)
}

// press sends keys to the model in turn and returns what it became, along
// with the command the last key produced.
func press(t *testing.T, m Model, keys ...string) (Model, tea.Cmd) {
	t.Helper()

	var cmd tea.Cmd
	for _, key := range keys {
		var next tea.Model
		next, cmd = m.Update(keyMsg(key))
		m = next.(Model)
	}
	return m, cmd
}

// paste delivers a value the way a terminal delivers a real paste: every
// character in one message rather than one keystroke at a time.
func paste(t *testing.T, m Model, value string) Model {
	t.Helper()

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value), Paste: true})
	return next.(Model)
}

// keyMsg turns a key name into the message Bubble Tea would deliver for it.
func keyMsg(key string) tea.KeyMsg {
	switch key {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case " ":
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	case "ctrl+r":
		return tea.KeyMsg{Type: tea.KeyCtrlR}
	case "ctrl+s":
		return tea.KeyMsg{Type: tea.KeyCtrlS}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
}

func TestSetup_WalksFromTheIntroToTheFirstQuestion(t *testing.T) {
	// Arrange
	m := firstRunModel(t, "/tmp/investec.env")

	// Act: through the intro, then accept the pre-ticked country.
	m, _ = press(t, m, "enter", "enter")

	// Assert
	if m.setup.step != setupFields {
		t.Fatalf("step = %v, want the credential questions", m.setup.step)
	}
	if len(m.setup.fields) != len(credentialFields) {
		t.Errorf("asked for %d values, want %d for one country", len(m.setup.fields), len(credentialFields))
	}
	if got := m.View(); !strings.Contains(got, credentialFields[fieldClientID].label) {
		t.Error("the first question does not ask for the Client ID")
	}
}

func TestSetup_HidesWhatIsTypedUntilItIsRevealed(t *testing.T) {
	// The whole point of the screen is that a credential never appears on a
	// screen someone could be looking over, or in a screen recording.
	const secret = "za-client-id-9f3a"

	// Arrange
	m := firstRunModel(t, "/tmp/investec.env")
	m, _ = press(t, m, "enter", "enter")

	// Act
	m = paste(t, m, secret)

	// Assert
	hidden := m.View()
	if strings.Contains(hidden, secret) {
		t.Error("the value is on screen while it is supposed to be hidden")
	}
	if !strings.Contains(hidden, strings.Repeat("•", len(secret))) {
		t.Error("the value is not masked")
	}

	// Act: reveal it, which is a deliberate keypress.
	m, _ = press(t, m, "ctrl+r")

	// Assert
	if !strings.Contains(m.View(), secret) {
		t.Error("ctrl+r did not reveal the value")
	}
}

func TestSetup_KeepsAPastedValueWhole(t *testing.T) {
	// A credential is pasted, not typed, and a terminal delivers a paste as
	// one message holding every character.
	const clientID = "  za-client-id-with-surrounding-space  "

	// Arrange
	m := firstRunModel(t, "/tmp/investec.env")
	m, _ = press(t, m, "enter", "enter")

	// Act
	m = paste(t, m, clientID)
	m, _ = press(t, m, "enter")

	// Assert
	if got := m.setup.fields[fieldClientID].value; got != strings.TrimSpace(clientID) {
		t.Errorf("stored %q, want the pasted value without the stray spaces", got)
	}
}

func TestSetup_GoingBackKeepsWhatWasEntered(t *testing.T) {
	// Arrange
	m := firstRunModel(t, "/tmp/investec.env")
	m, _ = press(t, m, "enter", "enter")
	m = paste(t, m, "first-value")

	// Act: on to the secret, then back again.
	m, _ = press(t, m, "enter", "shift+tab")

	// Assert
	if m.setup.field != fieldClientID {
		t.Fatalf("field = %d, want to be back on the Client ID", m.setup.field)
	}
	if got := m.setup.input.Value(); got != "first-value" {
		t.Errorf("input holds %q, want the value entered earlier", got)
	}
}

func TestSetup_AsksForEveryTickedCountry(t *testing.T) {
	// Arrange
	m := firstRunModel(t, "/tmp/investec.env")
	m, _ = press(t, m, "enter")

	// Act: tick the second country as well, then answer all six questions.
	m, _ = press(t, m, "down", " ", "enter")

	values := []string{"za-id", "za-secret", "za-key", "mu-id", "mu-secret", "mu-key"}
	for i, value := range values {
		if m.setup.step != setupFields {
			t.Fatalf("left the questions after %d of %d values", i, len(values))
		}
		m = paste(t, m, value)
		m, _ = press(t, m, "enter")
	}

	// Assert
	countries := m.setup.countries()
	if len(countries) != 2 {
		t.Fatalf("collected %d countries, want 2", len(countries))
	}
	want := []config.Country{
		{Name: "South Africa", Code: "ZA", ClientID: "za-id", ClientSecret: "za-secret", APIKey: "za-key"},
		{Name: "Mauritius", Code: "MU", ClientID: "mu-id", ClientSecret: "mu-secret", APIKey: "mu-key"},
	}
	for i, wantCountry := range want {
		if countries[i] != wantCountry {
			t.Errorf("country %d = %+v, want %+v", i, countries[i], wantCountry)
		}
	}
}

func TestSetup_WritesTheCredentialsFile(t *testing.T) {
	// Arrange: answer the three questions for the pre-ticked country.
	path := filepath.Join(t.TempDir(), "investec.env")
	m := firstRunModel(t, path)
	m, _ = press(t, m, "enter", "enter")
	for _, value := range []string{"za-id", "za-secret", "za-key"} {
		m = paste(t, m, value)
		if value != "za-key" {
			m, _ = press(t, m, "enter")
		}
	}

	// Act: save without waiting for the check against the live API.
	m, cmd := press(t, m, "ctrl+s")
	if cmd == nil {
		t.Fatal("ctrl+s produced no command, so nothing was saved")
	}
	next, _ := m.Update(cmd())
	m = next.(Model)

	// Assert
	if m.setup.err != nil {
		t.Fatalf("saving failed: %v", m.setup.err)
	}
	if m.setup.step != setupDone {
		t.Errorf("step = %v, want the confirmation", m.setup.step)
	}

	saved, err := godotenv.Read(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if saved["ZA_CLIENT_ID"] != "za-id" || saved["ZA_API_KEY"] != "za-key" {
		t.Errorf("file holds %v, want the values just entered", saved)
	}

	// And the app carries on with those credentials, without a restart.
	m, _ = press(t, m, "enter")
	if m.state != viewCountry {
		t.Fatalf("state = %v, want the country landing page", m.state)
	}
	if len(m.countryList.countries) != 1 || !m.countryList.countries[0].HasCredentials() {
		t.Error("the landing page did not pick up the new credentials")
	}
}

func TestSetup_CannotBeAbandonedOnAFirstRun(t *testing.T) {
	// There is nothing behind the screen to go back to until something has
	// been saved, so esc must not strand the user on an empty app.

	// Arrange
	m := firstRunModel(t, "/tmp/investec.env")

	// Act
	m, _ = press(t, m, "esc")

	// Assert
	if m.state != viewSetup {
		t.Errorf("state = %v, want to still be in setup", m.state)
	}
}

func TestCountryPage_OpensSetup(t *testing.T) {
	tests := []struct {
		name      string
		countries []config.Country
		key       string
	}{
		{
			name:      "c changes credentials that are already there",
			countries: []config.Country{{Name: "South Africa", Code: "ZA", ClientID: "id", ClientSecret: "secret", APIKey: "key"}},
			key:       "c",
		},
		{
			name:      "choosing a country with nothing entered offers to fill it in",
			countries: []config.Country{{Name: "South Africa", Code: "ZA"}},
			key:       "enter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			m := resize(NewModel(tt.countries), 120, 35)

			// Act
			m, _ = press(t, m, tt.key)

			// Assert
			if m.state != viewSetup {
				t.Fatalf("state = %v, want the setup screen", m.state)
			}
			if !m.setup.canCancel {
				t.Error("setup opened from the app should be escapable")
			}

			// Act: back out again.
			m, _ = press(t, m, "esc")

			// Assert
			if m.state != viewCountry {
				t.Errorf("state = %v, want to be back on the country page", m.state)
			}
		})
	}
}

func TestSetup_SeedsTheCountriesAlreadyConfigured(t *testing.T) {
	// Arrange
	configured := []config.Country{
		{Name: "Mauritius", Code: "MU", ClientID: "id", ClientSecret: "secret", APIKey: "key"},
	}

	// Act
	v := newSetupView(configured, "/tmp/investec.env", true, 120)

	// Assert
	for _, choice := range v.choices {
		if choice.country.Code == "MU" && !choice.selected {
			t.Error("the country already in use is not ticked")
		}
		if choice.country.Code == "ZA" && choice.selected {
			t.Error("a country that was never configured is ticked")
		}
	}
}
