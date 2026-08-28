//go:build darwin

package startup

import (
	"fmt"
	"time"
)

// settleDelay gives Terminal a moment to apply the resize and deliver the
// resulting SIGWINCH, so the first frame is painted at the final size instead
// of being drawn small and then reflowed.
const settleDelay = 75 * time.Millisecond

// SizeWindow asks the terminal to resize itself to cols by rows.
//
// It only does so when the app owns the window, meaning Terminal was opened by
// double-clicking the .app. A developer running the binary in their own shell
// keeps whatever window size they chose.
//
// This is best-effort. If the user has macOS set to prefer tabs when opening
// documents, Terminal may run the app in a tab of an existing window, where
// the resize would affect its siblings and is ignored or clamped. The app
// simply uses whatever size it ends up with in that case.
func SizeWindow(cols, rows int) {
	if !IsTerminal() || bundlePath() == "" {
		return
	}

	// CSI 8 ; rows ; cols t is the standard window manipulation sequence, and
	// Terminal.app honours it.
	fmt.Printf("\033[8;%d;%dt", rows, cols)
	time.Sleep(settleDelay)
}
