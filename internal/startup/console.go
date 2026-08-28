// Package startup holds the small amount of platform glue needed to run the
// TUI as a downloaded, double-clickable application rather than from a
// developer's shell.
package startup

import (
	"bufio"
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
)

// NoPauseEnv disables the "press Enter" pause. The ./run script sets it so a
// developer's edit-run loop is not interrupted.
const NoPauseEnv = "INVESTEC_TUI_NO_PAUSE"

// IsTerminal reports whether stdin is attached to a real terminal. A process
// launched from Finder gets /dev/null, which is a character device but not a
// terminal, so this cannot be done with a plain os.Stat.
func IsTerminal() bool {
	fd := os.Stdin.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

// Pause waits for Enter so a message stays readable in a console window that
// would otherwise vanish the instant the process exits, which is what happens
// when a Windows user double-clicks the .exe.
func Pause() {
	if os.Getenv(NoPauseEnv) == "1" || !IsTerminal() {
		return
	}

	fmt.Print("\nPress Enter to close this window... ")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}
