package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// splashBanner is the wordmark on the splash page.
var splashBanner = []string{
	"██╗███╗   ██╗██╗   ██╗███████╗███████╗████████╗███████╗ ██████╗",
	"██║████╗  ██║██║   ██║██╔════╝██╔════╝╚══██╔══╝██╔════╝██╔════╝",
	"██║██╔██╗ ██║██║   ██║█████╗  ███████╗   ██║   █████╗  ██║     ",
	"██║██║╚██╗██║╚██╗ ██╔╝██╔══╝  ╚════██║   ██║   ██╔══╝  ██║     ",
	"██║██║ ╚████║ ╚████╔╝ ███████╗███████║   ██║   ███████╗╚██████╗",
	"╚═╝╚═╝  ╚═══╝  ╚═══╝  ╚══════╝╚══════╝   ╚═╝   ╚══════╝ ╚═════╝",
}

// compactBanner tops the framed views, apart from the transactions list,
// which needs every row it can get.
var compactBanner = []string{
	"┳ ┳┓ ┓┏ ┏┓ ┏┓ ┏┳┓ ┏┓ ┏┓",
	"┃ ┃┃ ┃┃ ┣  ┗┓  ┃  ┣  ┃ ",
	"┻ ┛┗ ┗┛ ┗┛ ┗┛  ┻  ┗┛ ┗┛",
}

// bannerTagline sits under the compact banner, and under the splash one.
const bannerTagline = "open banking · tui"

// compactBannerRows is what the compact banner takes from a view's body: the
// art, the tagline and a blank row after them.
const compactBannerRows = 5

// minBannerBodyRows is the least a view keeps for itself before the banner
// is dropped to make room.
const minBannerBodyRows = 12

// bannerRoom is how many rows the compact banner takes from the body of a
// framed view, or 0 when the window is too short to spare them.
func (m Model) bannerRoom() int {
	if !framedLook || m.height-frameChromeRows-compactBannerRows < minBannerBodyRows {
		return 0
	}
	return compactBannerRows
}

// bodyHeight is layoutHeight less the banner, for the views that show it.
func (m Model) bodyHeight() int {
	return m.layoutHeight() - m.bannerRoom()
}

// renderCompactBanner draws the compact banner, lined up with the tables'
// two-space indent.
func renderCompactBanner() []string {
	lines := make([]string, 0, compactBannerRows)
	for _, row := range compactBanner {
		lines = append(lines, normalRowStyle.Render("  ")+bigNumberStyle.Render(row))
	}
	pad := lipgloss.Width(compactBanner[0]) - lipgloss.Width(bannerTagline)
	lines = append(lines, normalRowStyle.Render("  "+strings.Repeat(" ", pad))+hintStyle.Render(bannerTagline), "")
	return lines
}

// withBanner puts the compact banner above body, as long as both fit in
// rows. A body that would be pushed off the bottom goes without it.
func withBanner(body string, rows int) string {
	lines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	if len(lines)+compactBannerRows > rows {
		return body
	}
	return strings.Join(append(renderCompactBanner(), lines...), "\n")
}
