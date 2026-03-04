package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

func keyPress(char rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{
		Code: char,
		Text: string(char),
	}
}

func TestTabController(t *testing.T) {
	tests := []struct {
		name     string
		titles   []string
		msgs     []tea.Msg
		expected int
	}{
		{
			name:     "initial tab is 0",
			titles:   []string{"A", "B", "C"},
			msgs:     nil,
			expected: 0,
		},
		{
			name:     "] goes to next tab",
			titles:   []string{"A", "B", "C"},
			msgs:     []tea.Msg{keyPress(']')},
			expected: 1,
		},
		{
			name:     "] wraps at end",
			titles:   []string{"A", "B", "C"},
			msgs:     []tea.Msg{keyPress(']'), keyPress(']'), keyPress(']')},
			expected: 0,
		},
		{
			name:     "[ goes to previous tab",
			titles:   []string{"A", "B", "C"},
			msgs:     []tea.Msg{keyPress(']'), keyPress('[')},
			expected: 0,
		},
		{
			name:     "[ wraps at start",
			titles:   []string{"A", "B", "C"},
			msgs:     []tea.Msg{keyPress('[')},
			expected: 2,
		},
		{
			name:     "number key 1 jumps to index 0",
			titles:   []string{"A", "B", "C"},
			msgs:     []tea.Msg{keyPress(']'), keyPress('1')},
			expected: 0,
		},
		{
			name:     "number key 3 jumps to index 2",
			titles:   []string{"A", "B", "C"},
			msgs:     []tea.Msg{keyPress('3')},
			expected: 2,
		},
		{
			name:     "out-of-range number key is ignored",
			titles:   []string{"A", "B"},
			msgs:     []tea.Msg{keyPress('9')},
			expected: 0,
		},
		{
			name:     "number key 2 on 2-tab controller",
			titles:   []string{"A", "B"},
			msgs:     []tea.Msg{keyPress('2')},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := NewTabController(tt.titles)
			for _, msg := range tt.msgs {
				tc, _ = tc.Update(msg)
			}
			assert.Equal(t, tt.expected, tc.Active())
		})
	}
}

func TestTabControllerView(t *testing.T) {
	tc := NewTabController([]string{"Pods", "Services", "Nodes"})
	view := tc.View()
	assert.Contains(t, view, "Pods")
	assert.Contains(t, view, "Services")
	assert.Contains(t, view, "Nodes")
}
