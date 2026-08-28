package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// fileHeader tops a credentials file the app creates itself. It mirrors
// env.example, which stays the template for anyone writing the file by hand.
const fileHeader = `# Investec Open Banking TUI credentials
#
# Written by the app's setup screen. You can also edit it by hand: one
# KEY=value per line.
#
# Treat this file like a password. Do not share it, email it, or put it in a
# shared folder.`

// catalogue is the built-in list of countries the setup screen offers, and
// the same pair env.example ships with.
var catalogue = []Country{
	{Name: "South Africa", Code: "ZA"},
	{Name: "Mauritius", Code: "MU"},
}

// assignment matches a KEY=... line. The optional "export " prefix is
// accepted because some people write the file so a shell can source it too.
var assignment = regexp.MustCompile(`^\s*(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=`)

// setting is one KEY=value pair the app manages.
type setting struct {
	key   string
	value string
}

// section is a group of settings with the comment that introduces them.
type section struct {
	comment  string
	settings []setting
}

// KnownCountries returns the countries the setup screen offers: the built-in
// catalogue first, carrying over any credentials already configured, followed
// by any other country the user added to their file by hand.
func KnownCountries(configured []Country) []Country {
	known := make([]Country, 0, len(catalogue)+len(configured))
	seen := make(map[string]bool, len(catalogue)+len(configured))

	for _, c := range catalogue {
		if existing, ok := findCountry(configured, c.Code); ok {
			c.ClientID, c.ClientSecret, c.APIKey = existing.ClientID, existing.ClientSecret, existing.APIKey
		}
		known = append(known, c)
		seen[c.Code] = true
	}

	for _, c := range configured {
		if seen[c.Code] {
			continue
		}
		seen[c.Code] = true
		known = append(known, c)
	}

	return known
}

// findCountry returns the country with the given code.
func findCountry(countries []Country, code string) (Country, bool) {
	for _, c := range countries {
		if c.Code == code {
			return c, true
		}
	}
	return Country{}, false
}

// TargetEnvFile returns the path the setup screen should save to.
func TargetEnvFile() (string, error) {
	resolved, _ := ResolveEnvFile()

	dir, err := ConfigDir()
	if err != nil && resolved == "" && os.Getenv(EnvFileOverride) == "" {
		return "", err
	}

	return targetEnvFile(resolved, os.Getenv(EnvFileOverride), dir), nil
}

// targetEnvFile is the testable core of TargetEnvFile.
//
// A file that is already being loaded is edited in place, so a developer
// running from a clone keeps using the repo's own file instead of quietly
// gaining a second one in the per-user folder.
func targetEnvFile(resolved, override, configDir string) string {
	if resolved != "" {
		return resolved
	}
	if override != "" {
		return override
	}
	return filepath.Join(configDir, PreferredEnvFileName)
}

// SaveCountries writes COUNTRY_LIST and the per-country credentials to path.
//
// An existing file is edited rather than replaced: comments, blank lines and
// any setting the app does not manage are left exactly as they were, so a
// hand-written file survives a trip through the setup screen. Countries left
// out of the list keep their credentials in the file, ready for the day they
// are ticked again.
func SaveCountries(path string, countries []Country) error {
	sections := credentialSections(countries)

	for _, s := range sections {
		for _, set := range s.settings {
			if err := validateValue(set.key, set.value); err != nil {
				return err
			}
		}
	}

	existing, newline, err := readEnvFile(path)
	if err != nil {
		return err
	}

	lines := upsert(existing, sections, len(existing) == 0)

	return writeFileAtomically(path, []byte(strings.Join(lines, newline)+newline))
}

// credentialSections lays out everything the app manages, in the order it is
// written to a new file.
func credentialSections(countries []Country) []section {
	sections := []section{{
		comment:  "# Countries on the landing page, as {Name:CODE;Name:CODE}.",
		settings: []setting{{key: "COUNTRY_LIST", value: formatCountryList(countries)}},
	}}

	for _, c := range countries {
		sections = append(sections, section{
			comment: fmt.Sprintf("# %s (%s)", c.Name, c.Code),
			settings: []setting{
				{key: c.Code + "_CLIENT_ID", value: c.ClientID},
				{key: c.Code + "_CLIENT_SECRET", value: c.ClientSecret},
				{key: c.Code + "_API_KEY", value: c.APIKey},
			},
		})
	}

	return sections
}

// formatCountryList renders the list in the {Name:CODE;Name:CODE} form
// parseCountryList reads back.
func formatCountryList(countries []Country) string {
	parts := make([]string, 0, len(countries))
	for _, c := range countries {
		parts = append(parts, c.Name+":"+c.Code)
	}
	return "{" + strings.Join(parts, ";") + "}"
}

// upsert rewrites the managed settings in lines and appends whichever ones
// were not already there.
func upsert(lines []string, sections []section, newFile bool) []string {
	replacement := make(map[string]string)
	for _, s := range sections {
		for _, set := range s.settings {
			replacement[set.key] = formatSetting(set)
		}
	}

	written := make(map[string]bool, len(replacement))
	updated := make([]string, 0, len(lines)+len(replacement)+2*len(sections))

	for _, line := range lines {
		match := assignment.FindStringSubmatch(line)
		if match == nil {
			updated = append(updated, line)
			continue
		}

		key := match[1]
		text, managed := replacement[key]
		if !managed {
			updated = append(updated, line)
			continue
		}

		// A second assignment of the same key would leave the file
		// contradicting itself, so only the first one is kept.
		if written[key] {
			continue
		}
		written[key] = true
		updated = append(updated, text)
	}

	if newFile {
		updated = append(updated, strings.Split(fileHeader, "\n")...)
	}

	for _, s := range sections {
		missing := make([]string, 0, len(s.settings))
		for _, set := range s.settings {
			if written[set.key] {
				continue
			}
			written[set.key] = true
			missing = append(missing, formatSetting(set))
		}
		if len(missing) == 0 {
			continue
		}

		if len(updated) > 0 && strings.TrimSpace(updated[len(updated)-1]) != "" {
			updated = append(updated, "")
		}
		updated = append(updated, s.comment)
		updated = append(updated, missing...)
	}

	return updated
}

// formatSetting renders one line of the file.
func formatSetting(s setting) string {
	return s.key + "=" + quote(s.value)
}

// quote renders a value as a double-quoted dotenv string.
//
// Everything is quoted and \, " and $ are escaped. An unquoted value ends at
// the first " #", and both quoted and unquoted values have $NAME replaced
// with whatever that variable holds. A credential is an opaque string, so
// none of that may be allowed to happen to it.
var quoteReplacer = strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `\$`)

func quote(value string) string {
	return `"` + quoteReplacer.Replace(value) + `"`
}

// validateValue rejects the two values that cannot be represented in a
// dotenv file. Neither can occur in an Investec credential, and refusing
// them is far better than writing a file the app can no longer read.
//
// The value itself is never included in the error: it is a secret.
func validateValue(key, value string) error {
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("%s cannot contain a line break", key)
	}
	if strings.HasSuffix(value, `\`) {
		return fmt.Errorf(`%s cannot end with a backslash`, key)
	}
	return nil
}

// readEnvFile returns the lines of path and the line ending it uses. A file
// that is not there yet reads as empty, which is the first-run case.
func readEnvFile(path string) (lines []string, newline string, err error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "\n", nil
		}
		return nil, "", fmt.Errorf("reading %s: %w", path, err)
	}

	text := string(contents)

	// Keep whichever line ending the file already uses, so a file written in
	// Notepad does not come back with a mixture of both.
	newline = "\n"
	if strings.Contains(text, "\r\n") {
		newline = "\r\n"
		text = strings.ReplaceAll(text, "\r\n", "\n")
	}

	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		return nil, newline, nil
	}

	return strings.Split(text, "\n"), newline, nil
}

// writeFileAtomically writes data to path through a temporary file in the
// same folder, so an interrupted save cannot leave a half-written
// credentials file behind. The folder and the file are readable only by the
// user who owns them.
func writeFileAtomically(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}

	temp, err := os.CreateTemp(dir, ".investec-env-*")
	if err != nil {
		return fmt.Errorf("creating a temporary file in %s: %w", dir, err)
	}
	tempPath := temp.Name()
	// Harmless once the rename below has succeeded, and the only cleanup
	// that matters on every path that fails before it.
	defer os.Remove(tempPath)

	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return fmt.Errorf("securing %s: %w", tempPath, err)
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return fmt.Errorf("writing %s: %w", tempPath, err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", tempPath, err)
	}

	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("saving %s: %w", path, err)
	}

	return nil
}
