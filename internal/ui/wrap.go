package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// WrapText wraps text to fit within the given width, breaking on word
// boundaries when possible. If indent > 0, continuation lines are
// indented by that many spaces.
func WrapText(s string, width, indent int) string {
	if width <= 0 || len(s) <= width {
		return s
	}

	prefix := strings.Repeat(" ", indent)
	var b strings.Builder
	remaining := s

	first := true
	for len(remaining) > 0 {
		lineWidth := width
		if !first {
			b.WriteString("\n")
			b.WriteString(prefix)
			lineWidth = width - indent
		}
		if lineWidth <= 0 {
			lineWidth = 1
		}

		if len(remaining) <= lineWidth {
			b.WriteString(remaining)
			break
		}

		// Find last space within lineWidth to break on
		cut := lineWidth
		if idx := strings.LastIndex(remaining[:cut], " "); idx > 0 {
			cut = idx
		}

		b.WriteString(remaining[:cut])
		remaining = strings.TrimLeft(remaining[cut:], " ")
		first = false
	}

	return b.String()
}

var (
	kvLabelStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Width(20)
	kvValueStyle = lipgloss.NewStyle()
)

// RenderKV renders key-value rows with text wrapping on values.
// The labelWidth controls the label column width; valueWidth controls
// the max width for values (0 means 80).
func RenderKV(rows []KV, labelWidth, valueWidth int) string {
	if labelWidth <= 0 {
		labelWidth = 20
	}
	if valueWidth <= 0 {
		valueWidth = 80
	}

	ls := kvLabelStyle.Width(labelWidth)

	var b strings.Builder
	for _, r := range rows {
		b.WriteString(ls.Render(r.K))
		b.WriteString(kvValueStyle.Render(WrapText(r.V, valueWidth, labelWidth)))
		b.WriteString("\n")
	}
	return b.String()
}

// KV is a key-value pair for rendering in detail views.
type KV struct {
	K, V string
}
