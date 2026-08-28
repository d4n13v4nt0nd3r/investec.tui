package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/joho/godotenv"
)

// za and mu are the two countries used throughout these tests.
func za(clientID, secret, apiKey string) Country {
	return Country{Name: "South Africa", Code: "ZA", ClientID: clientID, ClientSecret: secret, APIKey: apiKey}
}

func mu(clientID, secret, apiKey string) Country {
	return Country{Name: "Mauritius", Code: "MU", ClientID: clientID, ClientSecret: secret, APIKey: apiKey}
}

func TestSaveCountries_CreatesFileTheAppCanReadBack(t *testing.T) {
	// Arrange: a folder that does not exist yet, as on a first run.
	path := filepath.Join(t.TempDir(), "investec-tui", "investec.env")
	countries := []Country{za("za-id", "za-secret", "za-key"), mu("mu-id", "mu-secret", "mu-key")}

	// Act
	if err := SaveCountries(path, countries); err != nil {
		t.Fatalf("SaveCountries(): %v", err)
	}

	// Assert
	values, err := godotenv.Read(path)
	if err != nil {
		t.Fatalf("reading back %s: %v", path, err)
	}
	want := map[string]string{
		"COUNTRY_LIST":     "{South Africa:ZA;Mauritius:MU}",
		"ZA_CLIENT_ID":     "za-id",
		"ZA_CLIENT_SECRET": "za-secret",
		"ZA_API_KEY":       "za-key",
		"MU_CLIENT_ID":     "mu-id",
		"MU_CLIENT_SECRET": "mu-secret",
		"MU_API_KEY":       "mu-key",
	}
	for key, wantValue := range want {
		if values[key] != wantValue {
			t.Errorf("%s = %q, want %q", key, values[key], wantValue)
		}
	}
}

func TestSaveCountries_WritesCredentialsOnlyForTheOwner(t *testing.T) {
	// A credentials file is as sensitive as a password, so it must not be
	// readable by anyone else on the machine.
	if runtime.GOOS == "windows" {
		t.Skip("Unix file modes are not meaningful on Windows")
	}

	// Arrange
	dir := filepath.Join(t.TempDir(), "investec-tui")
	path := filepath.Join(dir, "investec.env")

	// Act
	if err := SaveCountries(path, []Country{za("id", "secret", "key")}); err != nil {
		t.Fatalf("SaveCountries(): %v", err)
	}

	// Assert
	file, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := file.Mode().Perm(); got != 0o600 {
		t.Errorf("file mode = %04o, want 0600", got)
	}

	folder, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat %s: %v", dir, err)
	}
	if got := folder.Mode().Perm(); got != 0o700 {
		t.Errorf("folder mode = %04o, want 0700", got)
	}
}

func TestSaveCountries_PreservesTheRestOfTheFile(t *testing.T) {
	// Arrange: a file the user has edited by hand, holding a comment, a
	// setting the app does not manage, and a country that is being removed
	// from the list but whose keys should survive in case it comes back.
	dir := t.TempDir()
	path := filepath.Join(dir, "investec.env")
	original := strings.Join([]string{
		"# My credentials -- do not share",
		"COUNTRY_LIST={South Africa:ZA;Mauritius:MU}",
		"",
		"ZA_CLIENT_ID=old-id",
		"ZA_CLIENT_SECRET=old-secret",
		"ZA_API_KEY=old-key",
		"",
		"# Mauritius",
		"MU_CLIENT_ID=mu-id",
		"MU_CLIENT_SECRET=mu-secret",
		"MU_API_KEY=mu-key",
		"",
		"SOME_OTHER_SETTING=keep me",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}

	// Act
	if err := SaveCountries(path, []Country{za("new-id", "new-secret", "new-key")}); err != nil {
		t.Fatalf("SaveCountries(): %v", err)
	}

	// Assert
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	contents := string(saved)

	for _, want := range []string{"# My credentials -- do not share", "# Mauritius"} {
		if !strings.Contains(contents, want) {
			t.Errorf("comment %q was lost", want)
		}
	}

	values, err := godotenv.Read(path)
	if err != nil {
		t.Fatalf("reading back %s: %v", path, err)
	}
	if values["SOME_OTHER_SETTING"] != "keep me" {
		t.Errorf("SOME_OTHER_SETTING = %q, want it left alone", values["SOME_OTHER_SETTING"])
	}
	if values["MU_CLIENT_ID"] != "mu-id" {
		t.Errorf("MU_CLIENT_ID = %q, want the untouched value", values["MU_CLIENT_ID"])
	}
	if values["ZA_CLIENT_ID"] != "new-id" {
		t.Errorf("ZA_CLIENT_ID = %q, want the new value", values["ZA_CLIENT_ID"])
	}
	if values["COUNTRY_LIST"] != "{South Africa:ZA}" {
		t.Errorf("COUNTRY_LIST = %q, want only the countries just chosen", values["COUNTRY_LIST"])
	}
	if got := strings.Count(contents, "ZA_CLIENT_ID"); got != 1 {
		t.Errorf("ZA_CLIENT_ID appears %d times, want exactly 1", got)
	}
}

func TestSaveCountries_ValuesSurviveARoundTrip(t *testing.T) {
	// Credentials are opaque strings, so every character a bank might hand
	// out has to come back exactly as it went in. Unquoted dotenv values
	// treat " #" as a comment and expand $NAME, which would corrupt them.
	tests := []struct {
		name  string
		value string
	}{
		{name: "plain", value: "abc123"},
		{name: "base64 padding", value: "YWJjMTIz+/=="},
		{name: "spaces", value: "two words"},
		{name: "hash", value: "abc # not a comment"},
		{name: "double quote", value: `abc"def`},
		{name: "single quote", value: "abc'def"},
		{name: "backslash", value: `abc\def`},
		{name: "dollar", value: "abc$HOME"},
		{name: "empty", value: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			path := filepath.Join(t.TempDir(), "investec.env")

			// Act
			if err := SaveCountries(path, []Country{za(tt.value, tt.value, tt.value)}); err != nil {
				t.Fatalf("SaveCountries(): %v", err)
			}

			// Assert
			values, err := godotenv.Read(path)
			if err != nil {
				t.Fatalf("reading back %s: %v", path, err)
			}
			for _, key := range []string{"ZA_CLIENT_ID", "ZA_CLIENT_SECRET", "ZA_API_KEY"} {
				if values[key] != tt.value {
					t.Errorf("%s = %q, want %q", key, values[key], tt.value)
				}
			}
		})
	}
}

func TestSaveCountries_KeepsWindowsLineEndings(t *testing.T) {
	// A file created or edited in Notepad uses CRLF. Rewriting it with bare
	// newlines would leave a mixture that is awkward to read back in the
	// same editor.

	// Arrange
	path := filepath.Join(t.TempDir(), "investec.env")
	original := "# credentials\r\nZA_CLIENT_ID=old\r\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}

	// Act
	if err := SaveCountries(path, []Country{za("new", "secret", "key")}); err != nil {
		t.Fatalf("SaveCountries(): %v", err)
	}

	// Assert
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	contents := string(saved)
	if strings.Contains(strings.ReplaceAll(contents, "\r\n", ""), "\n") {
		t.Error("file has bare newlines mixed in with the CRLF ones")
	}
}

func TestSaveCountries_ReplacesAnExportedAssignment(t *testing.T) {
	// Some people write "export KEY=value" so the file can also be sourced
	// by a shell. That is still an assignment of the same key.

	// Arrange
	path := filepath.Join(t.TempDir(), "investec.env")
	if err := os.WriteFile(path, []byte("export ZA_CLIENT_ID=old\n"), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}

	// Act
	if err := SaveCountries(path, []Country{za("new", "secret", "key")}); err != nil {
		t.Fatalf("SaveCountries(): %v", err)
	}

	// Assert
	values, err := godotenv.Read(path)
	if err != nil {
		t.Fatalf("reading back %s: %v", path, err)
	}
	if values["ZA_CLIENT_ID"] != "new" {
		t.Errorf("ZA_CLIENT_ID = %q, want %q", values["ZA_CLIENT_ID"], "new")
	}
	if got := strings.Count(readFile(t, path), "ZA_CLIENT_ID"); got != 1 {
		t.Errorf("ZA_CLIENT_ID appears %d times, want exactly 1", got)
	}
}

// readFile returns the contents of path as a string.
func readFile(t *testing.T, path string) string {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(contents)
}

func TestTargetEnvFile_PrefersTheFileAlreadyInUse(t *testing.T) {
	tests := []struct {
		name      string
		resolved  string
		override  string
		configDir string
		want      string
	}{
		{
			name:      "an existing file is edited in place",
			resolved:  "/repo/.env",
			override:  "",
			configDir: "/home/user/config/investec-tui",
			want:      "/repo/.env",
		},
		{
			name:      "a pinned path is used even when the file is not there yet",
			resolved:  "",
			override:  "/pinned/credentials.env",
			configDir: "/home/user/config/investec-tui",
			want:      "/pinned/credentials.env",
		},
		{
			name:      "otherwise the per-user config folder",
			resolved:  "",
			override:  "",
			configDir: "/home/user/config/investec-tui",
			want:      filepath.Join("/home/user/config/investec-tui", PreferredEnvFileName),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Act
			got := targetEnvFile(tt.resolved, tt.override, tt.configDir)

			// Assert
			if got != tt.want {
				t.Errorf("targetEnvFile() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestKnownCountries_KeepsCodesTheUserAddedByHand(t *testing.T) {
	// Arrange: a configured list holding one country the app knows about
	// and one it does not.
	configured := []Country{
		{Name: "South Africa", Code: "ZA", ClientID: "id", ClientSecret: "secret", APIKey: "key"},
		{Name: "Guernsey", Code: "GG"},
	}

	// Act
	got := KnownCountries(configured)

	// Assert
	codes := make([]string, len(got))
	for i, c := range got {
		codes[i] = c.Code
	}
	if strings.Join(codes, ",") != "ZA,MU,GG" {
		t.Errorf("codes = %v, want the catalogue first then the hand-added code", codes)
	}
	if got[0].ClientID != "id" {
		t.Errorf("ZA lost its credentials: ClientID = %q", got[0].ClientID)
	}
}
