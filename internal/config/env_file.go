package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// EnvFileOverride is the environment variable that pins the credentials file
// to an exact path, bypassing the search below.
const EnvFileOverride = "INVESTEC_TUI_ENV"

// ConfigDirName is the per-user folder the credentials file lives in.
const ConfigDirName = "investec-tui"

// PreferredEnvFileName is the name a credentials file gets when the app
// creates one itself.
const PreferredEnvFileName = "investec.env"

// envFileNames are the accepted credentials file names, in order of
// preference. "investec.env" comes first because a file literally named
// ".env" is hidden in Finder and in Explorer's default view, which makes it
// very hard for a non-technical user to put one in place.
var envFileNames = []string{PreferredEnvFileName, ".env"}

// ConfigDir returns the per-user folder where the credentials file belongs:
// ~/Library/Application Support/investec-tui on macOS and
// %APPDATA%\investec-tui on Windows.
func ConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not determine the user config folder: %w", err)
	}
	return filepath.Join(base, ConfigDirName), nil
}

// SearchDirs returns the directories searched for a credentials file, in the
// order they are searched: the current directory (so a developer running
// ./run from the repo keeps using the repo's own file), then the folder
// holding the executable (the natural spot for a downloaded .exe), then the
// per-user config folder.
func SearchDirs() []string {
	var dirs []string

	if wd, err := os.Getwd(); err == nil {
		dirs = append(dirs, wd)
	}

	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		if dir := filepath.Dir(exe); !contains(dirs, dir) {
			dirs = append(dirs, dir)
		}
	}

	if dir, err := ConfigDir(); err == nil && !contains(dirs, dir) {
		dirs = append(dirs, dir)
	}

	return dirs
}

// ResolveEnvFile returns the path of the credentials file to load, along with
// every location that was considered. The path is empty when no file exists,
// in which case the caller should show the setup instructions.
func ResolveEnvFile() (path string, searched []string) {
	return resolveEnvFile(os.Getenv(EnvFileOverride), SearchDirs())
}

// resolveEnvFile is the testable core of ResolveEnvFile.
func resolveEnvFile(override string, dirs []string) (path string, searched []string) {
	// An explicit override is deliberately not allowed to fall back to the
	// normal search: a typo should surface as "not found" rather than
	// silently loading a different set of credentials.
	if override != "" {
		if isRegularFile(override) {
			return override, []string{override}
		}
		return "", []string{override}
	}

	searched = make([]string, 0, len(dirs)*len(envFileNames))
	for _, dir := range dirs {
		for _, name := range envFileNames {
			candidate := filepath.Join(dir, name)
			searched = append(searched, candidate)
			if path == "" && isRegularFile(candidate) {
				path = candidate
			}
		}
	}

	return path, searched
}

// isRegularFile reports whether path exists and is a file rather than a
// directory.
func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// contains reports whether values already holds value.
func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
