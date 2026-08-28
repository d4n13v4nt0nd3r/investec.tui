//go:build darwin

package startup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// bundleMarker appears in the path of any binary that lives inside a macOS
// .app bundle.
const bundleMarker = ".app/Contents/MacOS/"

// bundlePath returns this binary's path when it lives inside a .app bundle,
// and an empty string otherwise.
func bundlePath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	if !strings.Contains(exe, bundleMarker) {
		return ""
	}
	return exe
}

// RelaunchInTerminal hands the app over to Terminal and reports whether it
// did so, in which case the caller should exit immediately.
//
// A .app launched from Finder has no terminal attached, so the TUI would have
// nothing to draw on. Terminal re-runs this same binary with a real terminal,
// and because the check below requires the absence of one, the relaunch
// cannot recurse.
func RelaunchInTerminal() bool {
	if IsTerminal() {
		return false
	}

	exe := bundlePath()
	if exe == "" {
		return false
	}

	// The bundle is deliberately named without spaces, because Terminal runs
	// the path it is handed through a shell.
	if err := exec.Command("/usr/bin/open", "-a", "Terminal", exe).Run(); err != nil {
		return false
	}
	return true
}
