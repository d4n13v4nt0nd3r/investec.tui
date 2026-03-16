package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
	"investec.openbanking.tui/internal/api"
	"investec.openbanking.tui/internal/tui"
)

func main() {
	// Load environment
	if err := godotenv.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load .env file: %v\n", err)
	}

	clientID := os.Getenv("INVESTEC_CLIENT_ID")
	clientSecret := os.Getenv("INVESTEC_CLIENT_SECRET")
	apiKey := os.Getenv("INVESTEC_API_KEY")

	if clientID == "" || clientSecret == "" || apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: INVESTEC_CLIENT_ID, INVESTEC_CLIENT_SECRET, and INVESTEC_API_KEY must be set in .env")
		os.Exit(1)
	}

	// Create API client and authenticate
	client := api.NewClient(clientID, clientSecret, apiKey)
	if err := client.Authenticate(); err != nil {
		fmt.Fprintf(os.Stderr, "Authentication failed: %v\n", err)
		os.Exit(1)
	}

	// Start TUI
	model := tui.NewModel(client)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
