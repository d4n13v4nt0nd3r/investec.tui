package tui

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// glyphSet is the icons the framed look draws. There are two of them because
// the nicer icons are Nerd Font code points, which only draw in a patched
// font. Omarchy ships one in every terminal; a stock macOS or Windows
// terminal does not, and the glyphs would come out as empty boxes.
type glyphSet struct {
	// brand leads the frame's title. It carries its own trailing space, so
	// the plain set can leave it out altogether.
	brand string
	in    string // money coming into the account
	out   string // money going out of it
}

// nerdGlyphs uses nf-md-bank, nf-md-arrow_down and nf-md-arrow_up.
var nerdGlyphs = glyphSet{
	brand: "\U000F0070 ",
	in:    "\U000F0045",
	out:   "\U000F005D",
}

// plainGlyphs drops the brand mark and falls back to arrows the stock
// terminal fonts have. The app's help lines already use these two.
var plainGlyphs = glyphSet{
	in:  "↓",
	out: "↑",
}

// glyphs is the set in use. The plain one is the safe default; UseSystemLook
// swaps in the Nerd Font set when there is one to draw with.
var glyphs = plainGlyphs

// chooseGlyphs picks the set to draw with. pref is the user's override, and
// assumeNerd is set by a desktop that is known to ship a patched font.
func chooseGlyphs(pref string, assumeNerd bool) glyphSet {
	switch strings.ToLower(strings.TrimSpace(pref)) {
	case "nerd":
		return nerdGlyphs
	case "plain":
		return plainGlyphs
	}
	if assumeNerd || nerdFontInstalled(fontDirs()) {
		return nerdGlyphs
	}
	return plainGlyphs
}

// maxFontDepth bounds the search. macOS and Windows keep fonts in one flat
// folder; Linux nests them a couple deep under the family name.
const maxFontDepth = 3

// fontDirs are the folders a user's fonts are installed into, most likely
// first. Folders that do not exist are simply not walked.
func fontDirs() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	inHome := func(parts ...string) string {
		if home == "" {
			return ""
		}
		return filepath.Join(append([]string{home}, parts...)...)
	}

	switch runtime.GOOS {
	case "darwin":
		return []string{inHome("Library", "Fonts"), "/Library/Fonts"}
	case "windows":
		var dirs []string
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			dirs = append(dirs, filepath.Join(local, "Microsoft", "Windows", "Fonts"))
		}
		if win := os.Getenv("WINDIR"); win != "" {
			dirs = append(dirs, filepath.Join(win, "Fonts"))
		}
		return dirs
	default:
		data := os.Getenv("XDG_DATA_HOME")
		if data == "" {
			data = inHome(".local", "share")
		}
		var dirs []string
		if data != "" {
			dirs = append(dirs, filepath.Join(data, "fonts"))
		}
		return append(dirs, inHome(".fonts"), "/usr/share/fonts")
	}
}

// errFontFound stops the walk as soon as there is an answer.
var errFontFound = errors.New("nerd font found")

// nerdFontInstalled reports whether any of dirs holds a patched font.
//
// This is a guess: it says what is installed, not what the terminal is set
// to use. It is the best a program can do from inside the terminal, and
// INVESTEC_TUI_GLYPHS is there for when the guess is wrong.
func nerdFontInstalled(dirs []string) bool {
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		if walkFonts(dir) {
			return true
		}
	}
	return false
}

// walkFonts reports whether dir holds a patched font, searching no deeper
// than maxFontDepth.
func walkFonts(dir string) bool {
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		// An unreadable folder is skipped rather than failing the search.
		if err != nil {
			if entry != nil && entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			if depthBelow(dir, path) > maxFontDepth {
				return fs.SkipDir
			}
			return nil
		}
		if isFontFile(entry.Name()) && looksLikeNerdFont(entry.Name()) {
			return errFontFound
		}
		return nil
	})
	return errors.Is(err, errFontFound)
}

// depthBelow counts how many folders down path sits from root.
func depthBelow(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	return len(strings.Split(rel, string(filepath.Separator)))
}

func isFontFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".ttf", ".otf", ".ttc", ".otc":
		return true
	}
	return false
}

// looksLikeNerdFont matches the two ways patched fonts are named: the full
// "Nerd Font" in the file name, as the Nerd Fonts project ships them, and the
// "NF" marker of builds such as "MesloLGS NF Regular.ttf".
func looksLikeNerdFont(name string) bool {
	name = strings.ToLower(strings.TrimSuffix(name, filepath.Ext(name)))
	if strings.Contains(name, "nerd") {
		return true
	}
	for _, word := range strings.FieldsFunc(name, isNameSeparator) {
		if word == "nf" {
			return true
		}
	}
	return false
}

func isNameSeparator(r rune) bool {
	return r == ' ' || r == '-' || r == '_' || r == '.'
}
