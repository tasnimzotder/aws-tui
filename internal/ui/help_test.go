package ui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"tasnim.dev/aws-tui/internal/plugin"
)

func TestHelpOverlay(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "starts hidden",
			fn: func(t *testing.T) {
				h := NewHelpOverlay(nil)
				assert.False(t, h.Visible())
				assert.Empty(t, h.View())
			},
		},
		{
			name: "toggle makes it visible then hidden",
			fn: func(t *testing.T) {
				h := NewHelpOverlay(nil)
				h.Toggle()
				assert.True(t, h.Visible())
				h.Toggle()
				assert.False(t, h.Visible())
			},
		},
		{
			name: "view contains title when visible",
			fn: func(t *testing.T) {
				h := NewHelpOverlay(nil)
				h.Toggle()
				out := h.View()
				assert.Contains(t, out, "Keyboard Shortcuts")
			},
		},
		{
			name: "view contains context-specific hints",
			fn: func(t *testing.T) {
				hints := []plugin.KeyHint{
					{Key: "Enter", Desc: "Open detail"},
					{Key: "d", Desc: "Delete resource"},
				}
				h := NewHelpOverlay(hints)
				h.Toggle()
				out := h.View()
				assert.Contains(t, out, "Enter")
				assert.Contains(t, out, "Open detail")
				assert.Contains(t, out, "d")
				assert.Contains(t, out, "Delete resource")
			},
		},
		{
			name: "view contains global keys",
			fn: func(t *testing.T) {
				h := NewHelpOverlay(nil)
				h.Toggle()
				out := h.View()
				assert.Contains(t, out, "Esc")
				assert.Contains(t, out, "Go back")
				assert.Contains(t, out, "Ctrl+K")
				assert.Contains(t, out, "Command palette")
				assert.Contains(t, out, "q")
				assert.Contains(t, out, "Quit")
			},
		},
		{
			name: "view contains dismiss instruction",
			fn: func(t *testing.T) {
				h := NewHelpOverlay(nil)
				h.Toggle()
				out := h.View()
				assert.Contains(t, out, "Press ? or Esc to close")
			},
		},
		{
			name: "SetHints updates displayed hints",
			fn: func(t *testing.T) {
				h := NewHelpOverlay([]plugin.KeyHint{
					{Key: "a", Desc: "old action"},
				})
				h.SetHints([]plugin.KeyHint{
					{Key: "b", Desc: "new action"},
				})
				h.Toggle()
				out := h.View()
				assert.NotContains(t, out, "old action")
				assert.Contains(t, out, "new action")
			},
		},
		{
			name: "view has non-empty lines for each global key",
			fn: func(t *testing.T) {
				h := NewHelpOverlay(nil)
				h.Toggle()
				out := h.View()
				lines := strings.Split(out, "\n")
				require.True(t, len(lines) > 4, "expected multiple lines, got %d", len(lines))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}
