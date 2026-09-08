// Package export builds CSV exports and writes downloaded files to disk.
package export

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"investec.openbanking.tui/internal/api"
)

// DefaultDir returns the user's Downloads folder when it exists, otherwise home.
func DefaultDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "."
	}
	downloads := filepath.Join(home, "Downloads")
	if st, err := os.Stat(downloads); err == nil && st.IsDir() {
		return downloads
	}
	return home
}

// SanitizeFilename strips path separators and control characters from a name.
// Returns "download" when nothing usable remains.
func SanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, name)
	name = strings.Trim(name, " .")
	if name == "" || name == "." || name == ".." {
		return "download"
	}
	return name
}

// EnsureExt appends ext (including the dot) when the base name does not already
// end with it, case-insensitively.
func EnsureExt(name, ext string) string {
	name = SanitizeFilename(name)
	ext = strings.ToLower(ext)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	if strings.HasSuffix(strings.ToLower(name), ext) {
		return name
	}
	if i := strings.LastIndex(name, "."); i > 0 {
		existing := strings.ToLower(name[i:])
		if existing == ".pdf" || existing == ".csv" {
			name = name[:i]
		}
	}
	return name + ext
}

// DefaultFilename returns accountNumber with the given extension (e.g. ".pdf").
func DefaultFilename(accountNumber, ext string) string {
	base := strings.TrimSpace(accountNumber)
	if base == "" {
		base = "download"
	}
	return EnsureExt(base, ext)
}

// JoinPath joins dir and a sanitized filename.
func JoinPath(dir, filename string) string {
	return filepath.Join(dir, SanitizeFilename(filename))
}

// TargetPath builds the full path from a directory, user-typed name, and required extension.
func TargetPath(dir, name, ext string) string {
	return filepath.Join(dir, EnsureExt(name, ext))
}

// FileExists reports whether path already exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// WriteFile writes data to path with mode 0600. Parent directories must exist.
func WriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o600)
}

// ListSubdirs returns ".." plus immediate subdirectory names of dir, sorted.
func ListSubdirs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := []string{".."}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		names = append(names, name)
	}
	return names, nil
}

// ResolveDir walks from current using the chosen entry (".." or a child name).
func ResolveDir(current, entry string) (string, error) {
	var next string
	if entry == ".." {
		next = filepath.Dir(current)
	} else {
		next = filepath.Join(current, entry)
	}
	st, err := os.Stat(next)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		return "", fmt.Errorf("not a directory: %s", next)
	}
	return next, nil
}

// TransactionsToCSV encodes transactions as CSV bytes with a header row.
// Amounts are raw decimals (no thousand separators) for spreadsheet use.
func TransactionsToCSV(txns []api.Transaction, currency string) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	header := []string{
		"Date",
		"Type",
		"Description",
		"Amount",
		"Currency",
		"RunningBalance",
		"Status",
		"TransactionType",
		"PostingDate",
		"ValueDate",
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}

	for _, tx := range txns {
		posting := tx.PostingDate
		if posting == "" {
			posting = tx.PostDate
		}
		row := []string{
			tx.Date(),
			tx.Kind(),
			tx.Detail(),
			formatRawAmount(tx.SignedAmount()),
			currency,
			formatRawAmount(tx.RunningBalance),
			tx.Status,
			tx.TransactionType,
			trimDate(posting),
			trimDate(tx.ValueDate),
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func formatRawAmount(v float64) string {
	return fmt.Sprintf("%.2f", v)
}

func trimDate(d string) string {
	if len(d) > 10 {
		return d[:10]
	}
	return d
}
