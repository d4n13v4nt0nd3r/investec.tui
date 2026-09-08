package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"investec.openbanking.tui/internal/api"
)

func TestDefaultDirUsesDownloads(t *testing.T) {
	dir := DefaultDir()
	if filepath.Base(dir) != "Downloads" && dir != "." {
		// Only "." is acceptable when no home can be resolved at all.
		t.Fatalf("DefaultDir base = %q (full %q), want Downloads", filepath.Base(dir), dir)
	}
	if dir != "." {
		st, err := os.Stat(dir)
		if err != nil || !st.IsDir() {
			t.Fatalf("DefaultDir %q is not a usable directory: %v", dir, err)
		}
	}
}

func TestSanitizeFilename(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"10010206147", "10010206147"},
		{"foo/bar", "foo_bar"},
		{"foo\\bar", "foo_bar"},
		{"  name  ", "name"},
		{"", "download"},
		{".", "download"},
		{"..", "download"},
		{"a\nb", "ab"},
	}
	for _, c := range cases {
		if got := SanitizeFilename(c.in); got != c.want {
			t.Errorf("SanitizeFilename(%q)=%q, want %q", c.in, got, c.want)
		}
	}
}

func TestEnsureExtAndDefaultFilename(t *testing.T) {
	if got := EnsureExt("1001", ".pdf"); got != "1001.pdf" {
		t.Fatalf("EnsureExt: got %q", got)
	}
	if got := EnsureExt("1001.PDF", ".pdf"); got != "1001.PDF" {
		t.Fatalf("EnsureExt keep: got %q", got)
	}
	if got := EnsureExt("1001.csv", ".pdf"); got != "1001.pdf" {
		t.Fatalf("EnsureExt replace: got %q", got)
	}
	if got := DefaultFilename("10010206147", ".csv"); got != "10010206147.csv" {
		t.Fatalf("DefaultFilename: got %q", got)
	}
	if got := DefaultFilename("", ".pdf"); got != "download.pdf" {
		t.Fatalf("DefaultFilename empty: got %q", got)
	}
}

func TestTransactionsToCSV(t *testing.T) {
	txns := []api.Transaction{
		{
			Type:            "DEBIT",
			Description:     `Coffee, "shop"`,
			TransactionDate: "2024-02-11",
			PostingDate:     "2024-02-12",
			ValueDate:       "2024-02-12",
			Amount:          12.5,
			RunningBalance:  1000.5,
			Status:          "POSTED",
			TransactionType: "CardPurchases",
		},
	}
	data, err := TransactionsToCSV(txns, "ZAR")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "Date,Type,Description,Amount,Currency") {
		t.Fatalf("missing header: %s", s)
	}
	if !strings.Contains(s, "2024-02-11,DEBIT,") {
		t.Fatalf("missing row: %s", s)
	}
	if !strings.Contains(s, "-12.50") {
		t.Fatalf("expected signed amount: %s", s)
	}
	if !strings.Contains(s, "ZAR") {
		t.Fatalf("expected currency: %s", s)
	}
	// Description must be quoted because of comma/quote.
	if !strings.Contains(s, `"Coffee, ""shop"""`) {
		t.Fatalf("expected quoted description: %s", s)
	}
}

func TestWriteFileAndListSubdirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.csv")
	if err := WriteFile(path, []byte("a,b\n")); err != nil {
		t.Fatal(err)
	}
	if !FileExists(path) {
		t.Fatal("expected file to exist")
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o700); err != nil {
		t.Fatal(err)
	}
	names, err := ListSubdirs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) < 2 || names[0] != ".." {
		t.Fatalf("ListSubdirs: %v", names)
	}
	found := false
	for _, n := range names {
		if n == "sub" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected sub in %v", names)
	}
	next, err := ResolveDir(dir, "sub")
	if err != nil {
		t.Fatal(err)
	}
	if next != filepath.Join(dir, "sub") {
		t.Fatalf("ResolveDir: %q", next)
	}
}
