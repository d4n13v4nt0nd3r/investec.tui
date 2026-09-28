package tui

import (
	"math"
	"strings"
	"time"

	"investec.openbanking.tui/internal/api"
)

// bigGlyphs is a three-row font for amounts, drawn with heavy box-drawing
// lines like the Omarchy clock. Every glyph in a row has the same width.
var bigGlyphs = map[rune][3]string{
	'0': {"┏━┓", "┃ ┃", "┗━┛"},
	'1': {"╺┓ ", " ┃ ", "╺┻╸"},
	'2': {"┏━┓", "┏━┛", "┗━╸"},
	'3': {"┏━┓", "╺━┫", "┗━┛"},
	'4': {"╻ ╻", "┗━┫", "  ╹"},
	'5': {"┏━╸", "┗━┓", "┗━┛"},
	'6': {"┏━┓", "┣━┓", "┗━┛"},
	'7': {"┏━┓", "  ┃", "  ╹"},
	'8': {"┏━┓", "┣━┫", "┗━┛"},
	'9': {"┏━┓", "┗━┫", "┗━┛"},
	'-': {"   ", "╺━╸", "   "},
	'.': {" ", " ", "╹"},
	' ': {" ", " ", " "}, // the thousands separator
}

// bigNumber draws s in the big font, one string per row, with a column of
// space between glyphs. Characters the font lacks are left out.
func bigNumber(s string) [3]string {
	var rows [3]strings.Builder
	first := true
	for _, r := range s {
		glyph, ok := bigGlyphs[r]
		if !ok {
			continue
		}
		for i := range rows {
			if !first {
				rows[i].WriteByte(' ')
			}
			rows[i].WriteString(glyph[i])
		}
		first = false
	}
	return [3]string{rows[0].String(), rows[1].String(), rows[2].String()}
}

// dailyBalances works the closing balance for each of the last days days back
// from today's, oldest first, by taking away each day's transactions in turn.
//
// This needs nothing but the current balance and the posted transactions, so
// it works for every country whether or not the API sends a running balance.
func dailyBalances(current float64, txns []api.Transaction, days int, today time.Time) []float64 {
	if days <= 0 {
		return nil
	}
	moved := make(map[string]float64, len(txns))
	for _, tx := range txns {
		moved[tx.Date()] += tx.SignedAmount()
	}

	balances := make([]float64, days)
	balance := current
	for i := days - 1; i >= 0; i-- {
		balances[i] = balance
		day := today.AddDate(0, 0, i-(days-1)).Format("2006-01-02")
		balance -= moved[day]
	}
	return balances
}

var sparkLevels = []rune("▁▂▃▄▅▆▇█")

// sparkline draws values as a row of block characters at most width wide.
// When there are more values than columns, each column shows the last value
// of its share, so the line ends on today.
func sparkline(values []float64, width int) string {
	if width <= 0 || len(values) == 0 {
		return ""
	}
	if width > len(values) {
		width = len(values)
	}
	points := make([]float64, width)
	for i := range points {
		points[i] = values[(i+1)*len(values)/width-1]
	}

	low, high := math.Inf(1), math.Inf(-1)
	for _, v := range points {
		low = math.Min(low, v)
		high = math.Max(high, v)
	}

	var b strings.Builder
	for _, v := range points {
		level := 0
		if high > low {
			level = int(math.Round((v - low) / (high - low) * float64(len(sparkLevels)-1)))
		}
		b.WriteRune(sparkLevels[level])
	}
	return b.String()
}
