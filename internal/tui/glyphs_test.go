package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"investec.openbanking.tui/internal/api"
)

// useGlyphs draws with set for the rest of the test, so what a test sees
// never depends on the fonts the machine running it happens to have.
func useGlyphs(t *testing.T, set glyphSet) {
	t.Helper()
	previous := glyphs
	glyphs = set
	t.Cleanup(func() { glyphs = previous })
}

func TestNerdFontInstalled_RecognisesPatchedFontNames(t *testing.T) {
	tests := []struct {
		name string
		file string
		want bool
	}{
		{name: "nerd fonts naming", file: "JetBrainsMonoNerdFont-Regular.ttf", want: true},
		{name: "spaced naming", file: "Hack Nerd Font Mono.otf", want: true},
		{name: "nf marker", file: "MesloLGS NF Regular.ttf", want: true},
		{name: "stock font", file: "Menlo.ttc", want: false},
		{name: "not a font at all", file: "nerd-fonts-readme.txt", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, tt.file), nil, 0o600); err != nil {
				t.Fatal(err)
			}

			// Act
			got := nerdFontInstalled([]string{dir})

			// Assert
			if got != tt.want {
				t.Errorf("%s: found a Nerd Font = %v, want %v", tt.file, got, tt.want)
			}
		})
	}
}

func TestNerdFontInstalled_SearchesInsideFontFolders(t *testing.T) {
	// Linux files its fonts a couple of folders down, by format and family.
	dir := t.TempDir()
	nested := filepath.Join(dir, "truetype", "jetbrains")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "JetBrainsMonoNerdFont-Regular.ttf"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	// Act: a folder that does not exist must not stop the search either.
	got := nerdFontInstalled([]string{filepath.Join(dir, "nothing-here"), dir})

	// Assert
	if !got {
		t.Error("a font two folders down was not found")
	}
}

func TestChooseGlyphs_TheOverrideWins(t *testing.T) {
	tests := []struct {
		name       string
		pref       string
		assumeNerd bool
		want       glyphSet
	}{
		{name: "plain over a desktop that has the font", pref: "plain", assumeNerd: true, want: plainGlyphs},
		{name: "nerd without looking for one", pref: " NERD ", assumeNerd: false, want: nerdGlyphs},
		{name: "desktop that ships the font", pref: "", assumeNerd: true, want: nerdGlyphs},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := chooseGlyphs(tt.pref, tt.assumeNerd); got != tt.want {
				t.Errorf("glyphs = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestGlyphSets_KeepTheColumnsLinedUp guards the tables. The plain arrows
// stand in for Nerd Font ones, so they have to take the same number of cells
// or every column after them shifts.
func TestGlyphSets_KeepTheColumnsLinedUp(t *testing.T) {
	useFramedLook(t)
	txns := map[string]api.Transaction{
		"credit": {Type: "CREDIT", Description: "SALARY", TransactionDate: "2026-09-25", Amount: 48000},
		"debit":  {Type: "DEBIT", Description: "UBER", TransactionDate: "2026-09-24", Amount: 186},
	}

	for name, tx := range txns {
		t.Run(name, func(t *testing.T) {
			// Act
			useGlyphs(t, nerdGlyphs)
			nerdRow, nerdRecent := transactionRow(tx, "98 765.43", false), recentRow(tx)
			useGlyphs(t, plainGlyphs)
			plainRow, plainRecent := transactionRow(tx, "98 765.43", false), recentRow(tx)

			// Assert
			if got, want := lipgloss.Width(plainRow), lipgloss.Width(nerdRow); got != want {
				t.Errorf("plain transaction row is %d cells, the Nerd Font one %d", got, want)
			}
			if got, want := lipgloss.Width(plainRecent), lipgloss.Width(nerdRecent); got != want {
				t.Errorf("plain recent row is %d cells, the Nerd Font one %d", got, want)
			}
		})
	}
}
