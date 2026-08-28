//go:build !darwin

package startup

// RelaunchInTerminal is a no-op away from macOS. Double-clicking the .exe on
// Windows already opens a console window, so there is nothing to hand over to.
func RelaunchInTerminal() bool {
	return false
}
