//go:build !darwin

package startup

// SizeWindow is a no-op away from macOS.
//
// Windows console hosts do not agree on the resize escape sequence: the
// classic conhost ignores it and Windows Terminal only honours it when the
// user has not disabled window manipulation, so the app leaves the window at
// whatever size the user's console opens with.
func SizeWindow(cols, rows int) {}
