package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// writeEnvFile creates a credentials file at dir/name and returns its path.
func writeEnvFile(t *testing.T, dir, name string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("COUNTRY_LIST={South Africa:ZA}\n"), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
	return path
}

// expand turns a "<dirIndex>/<filename>" reference into an absolute path.
func expand(t *testing.T, dirs []string, ref string) string {
	t.Helper()

	index, name, found := strings.Cut(ref, "/")
	if !found {
		t.Fatalf("malformed reference %q, want <dirIndex>/<filename>", ref)
	}
	i, err := strconv.Atoi(index)
	if err != nil {
		t.Fatalf("malformed directory index in %q: %v", ref, err)
	}
	return filepath.Join(dirs[i], name)
}

func TestResolveEnvFile_SearchOrder(t *testing.T) {
	tests := []struct {
		name string
		// files to create, each as "<dirIndex>/<filename>".
		files []string
		// want is the file expected to be returned, in the same notation.
		// An empty string means no file should be found.
		want string
	}{
		{
			name:  "nothing anywhere",
			files: nil,
			want:  "",
		},
		{
			name:  "visible name preferred over the hidden dot file",
			files: []string{"0/.env", "0/investec.env"},
			want:  "0/investec.env",
		},
		{
			name:  "dot file used when it is the only one",
			files: []string{"0/.env"},
			want:  "0/.env",
		},
		{
			name:  "earlier directory wins over a later one",
			files: []string{"0/.env", "1/investec.env"},
			want:  "0/.env",
		},
		{
			name:  "falls through directories with no credentials",
			files: []string{"2/investec.env"},
			want:  "2/investec.env",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			dirs := []string{t.TempDir(), t.TempDir(), t.TempDir()}
			for _, ref := range tt.files {
				path := expand(t, dirs, ref)
				writeEnvFile(t, filepath.Dir(path), filepath.Base(path))
			}

			// Act
			got, searched := resolveEnvFile("", dirs)

			// Assert
			want := ""
			if tt.want != "" {
				want = expand(t, dirs, tt.want)
			}
			if got != want {
				t.Errorf("resolveEnvFile() = %q, want %q", got, want)
			}
			if len(searched) != len(dirs)*len(envFileNames) {
				t.Errorf("searched %d locations, want %d", len(searched), len(dirs)*len(envFileNames))
			}
		})
	}
}

func TestResolveEnvFile_OverrideWins(t *testing.T) {
	// Arrange
	overrideDir, searchDir := t.TempDir(), t.TempDir()
	override := writeEnvFile(t, overrideDir, "custom-credentials.env")
	writeEnvFile(t, searchDir, "investec.env")

	// Act
	got, searched := resolveEnvFile(override, []string{searchDir})

	// Assert
	if got != override {
		t.Errorf("resolveEnvFile() = %q, want the override %q", got, override)
	}
	if len(searched) != 1 || searched[0] != override {
		t.Errorf("searched = %v, want only the override %q", searched, override)
	}
}

func TestResolveEnvFile_MissingOverrideDoesNotFallBack(t *testing.T) {
	// A typo in INVESTEC_TUI_ENV must not silently load a different account's
	// credentials from another location.

	// Arrange
	searchDir := t.TempDir()
	writeEnvFile(t, searchDir, "investec.env")
	override := filepath.Join(t.TempDir(), "does-not-exist.env")

	// Act
	got, searched := resolveEnvFile(override, []string{searchDir})

	// Assert
	if got != "" {
		t.Errorf("resolveEnvFile() = %q, want no result", got)
	}
	if len(searched) != 1 || searched[0] != override {
		t.Errorf("searched = %v, want only the override %q", searched, override)
	}
}

func TestResolveEnvFile_IgnoresDirectories(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "investec.env"), 0o755); err != nil {
		t.Fatalf("creating decoy directory: %v", err)
	}
	want := writeEnvFile(t, dir, ".env")

	// Act
	got, _ := resolveEnvFile("", []string{dir})

	// Assert
	if got != want {
		t.Errorf("resolveEnvFile() = %q, want %q", got, want)
	}
}

func TestSearchDirs_IncludesConfigDir(t *testing.T) {
	// Arrange
	configDir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir(): %v", err)
	}

	// Act
	dirs := SearchDirs()

	// Assert
	if len(dirs) == 0 {
		t.Fatal("SearchDirs() returned no directories")
	}
	if dirs[len(dirs)-1] != configDir {
		t.Errorf("SearchDirs() last entry = %q, want the config folder %q", dirs[len(dirs)-1], configDir)
	}
}
