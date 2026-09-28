package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"investec.openbanking.tui/internal/export"
)

// saveFocus is which control is active in the save-as overlay.
type saveFocus int

const (
	saveFocusName saveFocus = iota
	saveFocusFolder
)

// saveAsKind identifies what will be written after the path is chosen.
type saveAsKind int

const (
	saveAsPDF saveAsKind = iota
	saveAsCSV
)

// saveAsView is an in-terminal save dialog with a filename field and folder browser.
type saveAsView struct {
	kind       saveAsKind
	ext        string
	dir        string
	entries    []string
	cursor     int
	offset     int
	pageSize   int
	focus      saveFocus
	input      textinput.Model
	confirming bool // overwrite confirm
	pending    string
	err        error
	active     bool
}

const (
	saveAsChrome = 12
	saveAsMinPage = 5
)

func newSaveAsView(kind saveAsKind, accountNumber, startDir string, windowHeight int) saveAsView {
	ext := ".pdf"
	if kind == saveAsCSV {
		ext = ".csv"
	}
	if startDir == "" {
		startDir = export.DefaultDir()
	}

	input := textinput.New()
	input.Prompt = "  "
	styleInput(&input)
	input.Cursor.SetMode(cursor.CursorStatic)
	input.SetValue(export.DefaultFilename(accountNumber, ext))
	input.Focus()
	input.Width = 40
	input.CharLimit = 200

	v := saveAsView{
		kind:     kind,
		ext:      ext,
		dir:      startDir,
		focus:    saveFocusName,
		input:    input,
		active:   true,
		pageSize: 8,
	}
	v.fitTo(windowHeight)
	v.reloadEntries()
	return v
}

func (v *saveAsView) fitTo(windowHeight int) {
	size := windowHeight - saveAsChrome
	if size < saveAsMinPage {
		size = saveAsMinPage
	}
	v.pageSize = size
	if v.cursor < v.offset {
		v.offset = v.cursor
	}
	if v.cursor >= v.offset+v.pageSize {
		v.offset = v.cursor - v.pageSize + 1
	}
	if v.offset < 0 {
		v.offset = 0
	}
}

func (v *saveAsView) reloadEntries() {
	entries, err := export.ListSubdirs(v.dir)
	if err != nil {
		v.err = err
		v.entries = []string{".."}
	} else {
		v.err = nil
		v.entries = entries
	}
	v.cursor = 0
	v.offset = 0
}

func (v saveAsView) targetPath() string {
	return export.TargetPath(v.dir, v.input.Value(), v.ext)
}

// update handles keys while the overlay is open. confirmed is set when the user
// accepts a final path (after optional overwrite confirm).
func (v saveAsView) update(msg tea.KeyMsg) (saveAsView, bool, tea.Cmd) {
	key := msg.String()

	if v.confirming {
		switch key {
		case "y", "Y":
			v.confirming = false
			return v, true, nil
		case "n", "N", "esc":
			v.confirming = false
			v.pending = ""
			return v, false, nil
		}
		return v, false, nil
	}

	switch key {
	case "esc":
		v.active = false
		return v, false, nil
	case "tab", "shift+tab":
		if v.focus == saveFocusName {
			v.focus = saveFocusFolder
			v.input.Blur()
		} else {
			v.focus = saveFocusName
			v.input.Focus()
		}
		return v, false, nil
	case "enter":
		if v.focus == saveFocusName {
			path := v.targetPath()
			if export.FileExists(path) {
				v.confirming = true
				v.pending = path
				return v, false, nil
			}
			return v, true, nil
		}
		// Folder focus: open selected directory.
		if len(v.entries) == 0 {
			return v, false, nil
		}
		entry := v.entries[v.cursor]
		next, err := export.ResolveDir(v.dir, entry)
		if err != nil {
			v.err = err
			return v, false, nil
		}
		v.dir = next
		v.reloadEntries()
		return v, false, nil
	}

	if v.focus == saveFocusFolder {
		switch key {
		case "up", "k":
			if v.cursor > 0 {
				v.cursor--
				if v.cursor < v.offset {
					v.offset = v.cursor
				}
			}
		case "down", "j":
			if v.cursor < len(v.entries)-1 {
				v.cursor++
				if v.cursor >= v.offset+v.pageSize {
					v.offset = v.cursor - v.pageSize + 1
				}
			}
		}
		return v, false, nil
	}

	// Filename field: feed the key to the text input.
	var cmd tea.Cmd
	v.input, cmd = v.input.Update(msg)
	return v, false, cmd
}

func (v saveAsView) render() string {
	var b strings.Builder

	title := "Save as"
	switch v.kind {
	case saveAsPDF:
		title = "Save PDF statement"
	case saveAsCSV:
		title = "Export transactions to CSV"
	}
	b.WriteString(subtitleStyle.Render(title))
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("Filename:"))
	b.WriteString(" ")
	if v.focus == saveFocusName {
		b.WriteString(v.input.View())
	} else {
		b.WriteString(normalRowStyle.Render("  " + v.input.Value()))
	}
	b.WriteString("\n")
	b.WriteString(hintStyle.Render(fmt.Sprintf("  Folder: %s", v.dir)))
	b.WriteString("\n\n")

	if v.confirming {
		b.WriteString(errorStyle.Render(fmt.Sprintf("File exists: %s", v.pending)))
		b.WriteString("\n")
		b.WriteString(normalRowStyle.Render("Overwrite? y / n"))
		return b.String()
	}

	if v.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", v.err)))
		b.WriteString("\n")
	}

	b.WriteString(headerRowStyle.Render("  Folders"))
	b.WriteString("\n")

	end := v.offset + v.pageSize
	if end > len(v.entries) {
		end = len(v.entries)
	}
	for i, name := range v.entries[v.offset:end] {
		globalIdx := v.offset + i
		label := "  " + name
		if name != ".." {
			label = "  " + name + "/"
		}
		if v.focus == saveFocusFolder && globalIdx == v.cursor {
			b.WriteString(selectedRow(label))
		} else {
			b.WriteString(normalRowStyle.Render(label))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (v saveAsView) help() string {
	if v.confirming {
		return "y overwrite  •  n cancel"
	}
	if v.focus == saveFocusName {
		return "type filename  •  enter save  •  tab folders  •  esc cancel"
	}
	return "↑/↓ browse  •  enter open folder  •  tab filename  •  esc cancel"
}
