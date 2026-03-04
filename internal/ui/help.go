package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"tasnim.dev/aws-tui/internal/plugin"
)

var (
	helpBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(1, 2)

	helpTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	helpKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			Width(12)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	helpFooterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			MarginTop(1).
			Italic(true)
)

// globalKeys are always shown at the bottom of the help overlay.
var globalKeys = []plugin.KeyHint{
	{Key: "Esc", Desc: "Go back"},
	{Key: "Ctrl+K", Desc: "Command palette"},
	{Key: "q", Desc: "Quit"},
	{Key: "?", Desc: "Toggle help"},
}

// HelpOverlay renders a toggleable overlay that shows keyboard shortcuts.
type HelpOverlay struct {
	hints   []plugin.KeyHint
	visible bool
}

// NewHelpOverlay creates a HelpOverlay with the given context-specific hints.
func NewHelpOverlay(hints []plugin.KeyHint) *HelpOverlay {
	return &HelpOverlay{hints: hints}
}

// Toggle flips the visibility of the overlay.
func (h *HelpOverlay) Toggle() {
	h.visible = !h.visible
}

// Visible returns whether the overlay is currently shown.
func (h *HelpOverlay) Visible() bool {
	return h.visible
}

// SetHints replaces the context-specific hints.
func (h *HelpOverlay) SetHints(hints []plugin.KeyHint) {
	h.hints = hints
}

// View renders the help overlay. Returns an empty string when hidden.
func (h *HelpOverlay) View() string {
	if !h.visible {
		return ""
	}

	var b strings.Builder

	b.WriteString(helpTitleStyle.Render("Keyboard Shortcuts"))
	b.WriteString("\n")

	// Context-specific hints.
	for _, hint := range h.hints {
		b.WriteString(formatHint(hint))
		b.WriteString("\n")
	}

	// Separator between context and global hints.
	if len(h.hints) > 0 {
		b.WriteString("\n")
	}

	// Global hints.
	for _, hint := range globalKeys {
		b.WriteString(formatHint(hint))
		b.WriteString("\n")
	}

	b.WriteString(helpFooterStyle.Render("Press ? or Esc to close"))

	return helpBoxStyle.Render(b.String())
}

func formatHint(hint plugin.KeyHint) string {
	return fmt.Sprintf("%s  %s",
		helpKeyStyle.Render(hint.Key),
		helpDescStyle.Render(hint.Desc),
	)
}
