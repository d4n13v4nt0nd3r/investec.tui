package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
	"investec.openbanking.tui/internal/config"
	"investec.openbanking.tui/internal/tui"
)

func main() {
	// Load environment
	if err := godotenv.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load .env file: %v\n", err)
	}

	// Load the country list and per-country credentials. Authentication happens
	// once a country is chosen on the landing page.
	countries, err := config.LoadCountries()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Start TUI
	model := tui.NewModel(countries)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
