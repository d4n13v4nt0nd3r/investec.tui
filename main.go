package main

import (
	_ "embed"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
	"investec.openbanking.tui/internal/config"
	"investec.openbanking.tui/internal/startup"
	"investec.openbanking.tui/internal/tui"
)

// version is replaced at build time with -ldflags "-X main.version=<tag>".
var version = "dev"

// Window size the packaged app asks Terminal for. The transactions table is
// the widest view, so the width is set to fit it without wrapping.
const (
	windowCols = 120
	windowRows = 35
)

// envTemplate is shown to a first-time user who has no credentials file yet.
//
//go:embed env.example
var envTemplate []byte

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println(version)
		return
	}

	// A .app launched from Finder has no terminal to draw on, so hand over to
	// Terminal and let this copy exit.
	if startup.RelaunchInTerminal() {
		return
	}

	startup.SizeWindow(windowCols, windowRows)

	envFile, searched := config.ResolveEnvFile()
	if envFile == "" {
		startup.ShowSetup(os.Stdout, searched, envTemplate)
		startup.Pause()
		os.Exit(1)
	}

	if err := godotenv.Load(envFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not read %s: %v\n", envFile, err)
		startup.Pause()
		os.Exit(1)
	}

	// Load the country list and per-country credentials. Authentication happens
	// once a country is chosen on the landing page.
	countries, err := config.LoadCountries()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Credentials file in use: %s\n", envFile)
		startup.Pause()
		os.Exit(1)
	}

	// Start TUI
	model := tui.NewModel(countries)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		startup.Pause()
		os.Exit(1)
	}
}
