package tui

import (
	"fmt"
	"strings"

	"investec.openbanking.tui/internal/api"
)

type documentsView struct {
	account   api.Account
	documents []api.Document
	cursor    int
	offset    int
	pageSize  int
	fromDate  string
	toDate    string
	err       error
	loading   bool
	editing   bool
	editField int
	editBuffer string
	status    string // last save path or message
	saving    bool
}

const documentsChrome = 13

func newDocumentsView(account api.Account, fromDate, toDate string, windowHeight int) documentsView {
	v := documentsView{
		account:  account,
		fromDate: fromDate,
		toDate:   toDate,
		loading:  true,
	}
	v.fitTo(windowHeight)
	return v
}

func (v *documentsView) fitTo(windowHeight int) {
	size := windowHeight - documentsChrome
	if size < minPageSize {
		size = minPageSize
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

func (v documentsView) selected() (api.Document, bool) {
	if v.cursor < 0 || v.cursor >= len(v.documents) {
		return api.Document{}, false
	}
	return v.documents[v.cursor], true
}

func (v documentsView) render() string {
	if v.err != nil && !v.loading {
		return errorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}

	var b strings.Builder

	fromVal := v.fromDate
	toVal := v.toDate
	if fromVal == "" {
		fromVal = "(default)"
	}
	if toVal == "" {
		toVal = "(default)"
	}
	if v.editing && v.editField == 0 {
		fromVal = v.editBuffer + "▎"
	}
	if v.editing && v.editField == 1 {
		toVal = v.editBuffer + "▎"
	}
	filterLine := fmt.Sprintf("From: %s  To: %s", fromVal, toVal)
	b.WriteString(normalRowStyle.Render(filterLine))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render(fmt.Sprintf("%s  (%s)", v.account.DisplayName(), v.account.AccountNumber)))
	b.WriteString("\n")

	if v.saving {
		b.WriteString(loadingStyle.Render("Saving document..."))
		b.WriteString("\n")
	}
	if v.status != "" {
		style := successStyle
		if strings.HasPrefix(v.status, "Save failed") {
			style = errorStyle
		}
		b.WriteString(style.Render(v.status))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if v.loading {
		b.WriteString(loadingStyle.Render("Loading documents..."))
		return b.String()
	}

	if len(v.documents) == 0 {
		b.WriteString(loadingStyle.Render("No documents found for this date range."))
		return b.String()
	}

	header := fmt.Sprintf("  %-16s  %s", "Date", "Type")
	b.WriteString(headerRowStyle.Render(header))
	b.WriteString("\n")

	end := v.offset + v.pageSize
	if end > len(v.documents) {
		end = len(v.documents)
	}
	for i, doc := range v.documents[v.offset:end] {
		row := fmt.Sprintf("  %-16s  %s", doc.DocumentDate, doc.DocumentType)
		globalIdx := v.offset + i
		if globalIdx == v.cursor {
			b.WriteString(selectedRowStyle.Render("> " + row[2:]))
		} else {
			b.WriteString(normalRowStyle.Render(row))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render(fmt.Sprintf("  Showing %d-%d of %d documents", v.offset+1, end, len(v.documents))))
	return b.String()
}
