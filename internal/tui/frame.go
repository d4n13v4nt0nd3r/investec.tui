package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// frameRowsSaved is how many more rows the frame leaves for the body than the
// classic layout.
//
// The views count their chrome against the classic layout: a title and a help
// line, each with a margin, inside a row of padding top and bottom. The frame
// puts the title and help on its border with a blank row inside each, which
// gives two rows back.
const frameRowsSaved = 2

// frameChromeRows is the frame's own rows: the top and bottom borders, and a
// blank row inside each.
const frameChromeRows = 4

// frameDivider starts a body line the frame draws as a divider across its
// full width. The rest of the line is the divider's title.
const frameDivider = "\x1f"

// frame is what goes on the border around a view.
type frame struct {
	section string // the view's name, after the brand on the top border
	context string // right-hand end of the top border
	legend  string // key help for the bottom border, as "key action  •  key action"
}

// renderFrame draws body inside a rounded border filling the window, with the
// title set into the top edge and the key legend into the bottom one.
//
// Body lines are clipped to the frame rather than wrapped, which would break
// the tables' columns, and rows past the bottom are dropped.
func renderFrame(f frame, body string, width, height int) string {
	inner := width - 2*appHPadding
	rows := height - frameChromeRows
	if inner < 1 || rows < 0 {
		return body
	}

	// The brand mark brings its own trailing space, so a glyph set without
	// one leaves no gap behind.
	title := frameTitleStyle.Render(glyphs.brand+"investec") + hintStyle.Render(" · ") + frameTitleStyle.Render(f.section)
	var context string
	if f.context != "" {
		context = hintStyle.Render(f.context)
	}

	lines := make([]string, 0, height)
	lines = append(lines, borderLine("╭", "╮", title, context, width))
	lines = append(lines, frameRow("", inner))

	bodyLines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	for i := 0; i < rows; i++ {
		switch {
		case i >= len(bodyLines):
			lines = append(lines, frameRow("", inner))
		case strings.HasPrefix(bodyLines[i], frameDivider):
			name := strings.TrimPrefix(bodyLines[i], frameDivider)
			lines = append(lines, borderLine("├", "┤", frameTitleStyle.Render(name), "", width))
		default:
			lines = append(lines, frameRow(bodyLines[i], inner))
		}
	}

	lines = append(lines, frameRow("", inner))
	lines = append(lines, borderLine("╰", "╯", formatLegend(f.legend), "", width))

	return strings.Join(lines, "\n")
}

// frameRow puts one body line between the side borders, clipped or padded to
// the inside width.
func frameRow(line string, inner int) string {
	line = ansi.Truncate(line, inner, "")
	pad := inner - lipgloss.Width(line)
	side := frameBorderStyle.Render("│")
	return side + normalRowStyle.Render(" ") + line + normalRowStyle.Render(strings.Repeat(" ", pad+1)) + side
}

// borderLine draws a horizontal edge between two corners, with optional text
// set into it at each end. When both do not fit, the right-hand text goes
// first, then the left is shortened.
func borderLine(leftCorner, rightCorner, left, right string, width int) string {
	// Corners, and the dash after the left corner and before the right one.
	room := width - 4

	label := func(s string) int {
		if s == "" {
			return 0
		}
		return lipgloss.Width(s) + 2 // a space either side
	}

	if label(left)+label(right) > room {
		right = ""
	}
	if label(left) > room {
		left = ansi.Truncate(left, room-2, "…")
		if room < 3 {
			left = ""
		}
	}

	var b strings.Builder
	b.WriteString(frameBorderStyle.Render(leftCorner + "─"))
	if left != "" {
		b.WriteString(normalRowStyle.Render(" ") + left + normalRowStyle.Render(" "))
	}
	fill := room - label(left) - label(right)
	if fill < 0 {
		fill = 0
	}
	b.WriteString(frameBorderStyle.Render(strings.Repeat("─", fill)))
	if right != "" {
		b.WriteString(normalRowStyle.Render(" ") + right + normalRowStyle.Render(" "))
	}
	b.WriteString(frameBorderStyle.Render("─" + rightCorner))
	return b.String()
}

// formatLegend turns a classic help line into the bottom border's legend: the
// key in the accent, what it does muted.
//
// A part only counts as "key action" when its first word looks like a key --
// short, with no capitals. Prompts such as "Type date (YYYY-MM-DD)" stay as
// plain text.
func formatLegend(help string) string {
	if help == "" {
		return ""
	}
	parts := strings.Split(help, "  •  ")
	out := make([]string, len(parts))
	for i, part := range parts {
		key, action, ok := strings.Cut(part, " ")
		if ok && len([]rune(key)) <= 9 && strings.ToLower(key) == key {
			out[i] = legendKeyStyle.Render(key) + hintStyle.Render(" "+action)
		} else {
			out[i] = hintStyle.Render(part)
		}
	}
	return strings.Join(out, hintStyle.Render("  "))
}
