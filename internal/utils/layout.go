package utils

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// DetailBuilder builds formatted key-value detail views for TUI viewports.
type DetailBuilder struct {
	b          strings.Builder
	labelStyle lipgloss.Style
	sectionStyle lipgloss.Style
}

// NewDetailBuilder creates a builder with a fixed-width label column.
// sectionStyle controls the rendering of section headings.
func NewDetailBuilder(labelWidth int, sectionStyle lipgloss.Style) *DetailBuilder {
	return &DetailBuilder{
		labelStyle:   sectionStyle.Width(labelWidth),
		sectionStyle: sectionStyle,
	}
}

// Row writes a labeled key-value row.
func (d *DetailBuilder) Row(label, value string) {
	fmt.Fprintf(&d.b, "  %s %s\n", d.labelStyle.Render(label), value)
}

// Section writes a section heading like "── title ──────...".
func (d *DetailBuilder) Section(title string) {
	pad := max(40-len(title), 4)
	heading := fmt.Sprintf("  ── %s %s", title, strings.Repeat("─", pad))
	d.b.WriteString(d.sectionStyle.Render(heading) + "\n")
}

// Blank writes an empty line.
func (d *DetailBuilder) Blank() {
	d.b.WriteString("\n")
}

// WriteString appends arbitrary text (for custom formatting not covered by Row/Section).
func (d *DetailBuilder) WriteString(s string) {
	d.b.WriteString(s)
}

// String returns the accumulated content.
func (d *DetailBuilder) String() string {
	return d.b.String()
}
