package startup

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"investec.openbanking.tui/internal/config"
)

// exampleFileName is the template dropped next to where the real credentials
// file belongs, so the user can see the expected format.
const exampleFileName = "env.example"

// ShowSetup explains where to save the credentials file when none was found.
// It creates the per-user config folder, leaves a copy of the template there
// and opens the folder, so the user only has to drop in the file they were
// given.
//
// Every step is best-effort: a failure to create or open the folder still
// leaves the user with the printed instructions.
func ShowSetup(w io.Writer, searched []string, template []byte) {
	fmt.Fprintln(w, "Investec Open Banking TUI -- setup needed")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "No credentials file was found.")
	fmt.Fprintln(w)

	dir, err := config.ConfigDir()
	if err != nil {
		fmt.Fprintf(w, "Could not work out where your settings folder is: %v\n", err)
		writeSearchedList(w, searched)
		return
	}

	target := filepath.Join(dir, "investec.env")
	fmt.Fprintln(w, "Save the file you were given as:")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "    %s\n", target)
	fmt.Fprintln(w)

	if err := os.MkdirAll(dir, 0o700); err != nil {
		fmt.Fprintf(w, "That folder could not be created automatically: %v\n", err)
		fmt.Fprintln(w, "Create it yourself, then put the file inside it.")
		writeSearchedList(w, searched)
		return
	}

	if len(template) > 0 {
		example := filepath.Join(dir, exampleFileName)
		if err := os.WriteFile(example, template, 0o600); err == nil {
			fmt.Fprintf(w, "An example of the expected format is in %s.\n", exampleFileName)
		}
	}

	if err := reveal(dir); err == nil {
		fmt.Fprintln(w, "That folder has been opened for you.")
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "You can also point the app at any file by setting the")
	fmt.Fprintf(w, "%s environment variable to its full path.\n", config.EnvFileOverride)
	writeSearchedList(w, searched)
}

// writeSearchedList prints every location that was checked, which is the
// quickest way to diagnose a file saved a folder too high or with the wrong
// name.
func writeSearchedList(w io.Writer, searched []string) {
	if len(searched) == 0 {
		return
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Looked in:")
	for _, path := range searched {
		fmt.Fprintf(w, "  - %s\n", path)
	}
}

// reveal opens dir in the platform's file manager.
func reveal(dir string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("/usr/bin/open", dir).Run()
	case "windows":
		// explorer.exe reports a non-zero exit code even when it succeeds, so
		// its error is deliberately discarded.
		_ = exec.Command("explorer.exe", dir).Run()
		return nil
	default:
		return fmt.Errorf("opening a folder is not supported on %s", runtime.GOOS)
	}
}
