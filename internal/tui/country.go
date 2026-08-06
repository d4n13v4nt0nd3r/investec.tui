package tui

import (
	"fmt"
	"strings"

	"investec.openbanking.tui/internal/config"
)

type countryView struct {
	countries  []config.Country
	cursor     int
	err        error
	connecting bool
}

func newCountryView(countries []config.Country) countryView {
	return countryView{countries: countries}
}

func (v countryView) selected() (config.Country, bool) {
	if v.cursor < 0 || v.cursor >= len(v.countries) {
		return config.Country{}, false
	}
	return v.countries[v.cursor], true
}

func (v countryView) render() string {
	if len(v.countries) == 0 {
		return errorStyle.Render("No countries configured. Set COUNTRY_LIST in .env")
	}

	var b strings.Builder

	b.WriteString(subtitleStyle.Render("Select a country"))
	b.WriteString("\n")

	header := fmt.Sprintf("  %-30s %-8s %-20s", "Country", "Code", "Credentials")
	b.WriteString(headerRowStyle.Render(header))
	b.WriteString("\n")

	for i, c := range v.countries {
		status := "configured"
		if !c.HasCredentials() {
			status = "missing"
		}

		row := fmt.Sprintf("  %-30s %-8s %-20s", truncate(c.Name, 28), c.Code, status)

		if i == v.cursor {
			b.WriteString(selectedRowStyle.Render("> " + row[2:]))
		} else {
			b.WriteString(normalRowStyle.Render(row))
		}
		b.WriteString("\n")
	}

	if v.connecting {
		if c, ok := v.selected(); ok {
			b.WriteString("\n")
			b.WriteString(loadingStyle.Render(fmt.Sprintf("Connecting to %s...", c.Name)))
			b.WriteString("\n")
		}
	}

	if v.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", v.err)))
		b.WriteString("\n")
	}

	return b.String()
}
