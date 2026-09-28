package main

import (
	_ "embed"
	"errors"
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

// envTemplate is shown to someone with no credentials file who cannot be
// walked through the setup screen, because there is no terminal to draw it
// on.
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

	// Without a terminal the setup screen cannot be drawn, so fall back to
	// printing where to save the file by hand.
	if envFile == "" && !startup.IsTerminal() {
		startup.ShowSetup(os.Stdout, searched, envTemplate)
		startup.Pause()
		os.Exit(1)
	}

	if envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error: could not read %s: %v\n", envFile, err)
			startup.Pause()
			os.Exit(1)
		}
	}

	// Load the country list and per-country credentials. Authentication happens
	// once a country is chosen on the landing page.
	countries, err := config.LoadCountries()
	if err != nil && !errors.Is(err, config.ErrNoCredentials) {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Credentials file in use: %s\n", envFile)
		startup.Pause()
		os.Exit(1)
	}

	// Settle the layout, the colours and the icons before Bubble Tea takes
	// the terminal, because reading the terminal's own colours means asking
	// it and waiting for the reply.
	tui.UseSystemLook(startup.IsTerminal())

	// Nothing configured yet, so the app opens on the guided setup screen
	// rather than telling the user to go and write a file.
	model := tui.NewModel(countries)
	if errors.Is(err, config.ErrNoCredentials) {
		target, targetErr := config.TargetEnvFile()
		if targetErr != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", targetErr)
			startup.Pause()
			os.Exit(1)
		}
		model = tui.NewSetupModel(countries, target)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		startup.Pause()
		os.Exit(1)
	}
}
