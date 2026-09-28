//go:build !linux

package tui

// omarchyThemeFile is Linux-only; other systems use the app's own palette.
func omarchyThemeFile() string { return "" }
