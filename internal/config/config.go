package config

import (
	"errors"
	"os"
	"strings"
)

// ErrNoCredentials reports that no country has a full set of credentials
// yet. That is the state a first-time user starts in rather than a failure,
// so callers open the setup screen instead of giving up.
var ErrNoCredentials = errors.New("no credentials have been entered yet")

// Country holds a selectable country and its Investec API credentials.
type Country struct {
	Name         string // e.g. "South Africa"
	Code         string // e.g. "ZA"
	ClientID     string
	ClientSecret string
	APIKey       string
}

// HasCredentials reports whether all three credentials are present.
func (c Country) HasCredentials() bool {
	return c.ClientID != "" && c.ClientSecret != "" && c.APIKey != ""
}

// MissingCredentials returns the names of the env vars that still need values.
func (c Country) MissingCredentials() []string {
	var missing []string
	if c.ClientID == "" {
		missing = append(missing, c.Code+"_CLIENT_ID")
	}
	if c.ClientSecret == "" {
		missing = append(missing, c.Code+"_CLIENT_SECRET")
	}
	if c.APIKey == "" {
		missing = append(missing, c.Code+"_API_KEY")
	}
	return missing
}

// LoadCountries reads COUNTRY_LIST and the per-country credentials from the
// environment.
//
// COUNTRY_LIST format: {South Africa:ZA;Mauritius:MU}
// Credentials per country: <CODE>_CLIENT_ID, <CODE>_CLIENT_SECRET, <CODE>_API_KEY
//
// If COUNTRY_LIST is absent, a single South Africa entry is assumed. Legacy
// INVESTEC_* variables are used as a fallback when a country has no
// country-specific credentials of its own.
func LoadCountries() ([]Country, error) {
	entries := parseCountryList(os.Getenv("COUNTRY_LIST"))
	if len(entries) == 0 {
		entries = []Country{{Name: "South Africa", Code: "ZA"}}
	}

	countries := make([]Country, 0, len(entries))
	for _, e := range entries {
		e.ClientID = os.Getenv(e.Code + "_CLIENT_ID")
		e.ClientSecret = os.Getenv(e.Code + "_CLIENT_SECRET")
		e.APIKey = os.Getenv(e.Code + "_API_KEY")

		// Legacy fallback: INVESTEC_* was the single (South African) credential set.
		if e.ClientID == "" && e.ClientSecret == "" && e.APIKey == "" {
			e.ClientID = os.Getenv("INVESTEC_CLIENT_ID")
			e.ClientSecret = os.Getenv("INVESTEC_CLIENT_SECRET")
			e.APIKey = os.Getenv("INVESTEC_API_KEY")
		}

		countries = append(countries, e)
	}

	configured := false
	for _, c := range countries {
		if c.HasCredentials() {
			configured = true
			break
		}
	}
	if !configured {
		return countries, ErrNoCredentials
	}

	return countries, nil
}

// parseCountryList parses "{South Africa:ZA;Mauritius:MU}" into countries.
func parseCountryList(raw string) []Country {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "{")
	raw = strings.TrimSuffix(raw, "}")
	if raw == "" {
		return nil
	}

	var countries []Country
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		name, code, found := strings.Cut(part, ":")
		if !found {
			continue
		}

		name = strings.TrimSpace(name)
		code = strings.ToUpper(strings.TrimSpace(code))
		if name == "" || code == "" {
			continue
		}

		countries = append(countries, Country{Name: name, Code: code})
	}

	return countries
}
