package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func specialKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

func TestPicker(t *testing.T) {
	items := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}

	tests := []struct {
		name          string
		items         []string
		msgs          []tea.Msg
		wantCursor    int
		wantFiltered  int
	}{
		{
			name:         "initial cursor at 0",
			items:        items,
			msgs:         nil,
			wantCursor:   0,
			wantFiltered: 4,
		},
		{
			name:         "j moves cursor down",
			items:        items,
			msgs:         []tea.Msg{keyPress('j')},
			wantCursor:   1,
			wantFiltered: 4,
		},
		{
			name:         "j bounded at bottom",
			items:        items,
			msgs:         []tea.Msg{keyPress('j'), keyPress('j'), keyPress('j'), keyPress('j'), keyPress('j')},
			wantCursor:   3,
			wantFiltered: 4,
		},
		{
			name:         "k moves cursor up",
			items:        items,
			msgs:         []tea.Msg{keyPress('j'), keyPress('j'), keyPress('k')},
			wantCursor:   1,
			wantFiltered: 4,
		},
		{
			name:         "k bounded at top",
			items:        items,
			msgs:         []tea.Msg{keyPress('k')},
			wantCursor:   0,
			wantFiltered: 4,
		},
		{
			name:         "typing filters the list",
			items:        items,
			msgs:         []tea.Msg{keyPress('u'), keyPress('s')},
			wantCursor:   0,
			wantFiltered: 2,
		},
		{
			name:         "filter is case insensitive",
			items:        items,
			msgs:         []tea.Msg{keyPress('E'), keyPress('U')},
			wantCursor:   0,
			wantFiltered: 1,
		},
		{
			name:         "cursor resets when filter changes",
			items:        items,
			msgs:         []tea.Msg{keyPress('j'), keyPress('j'), keyPress('e'), keyPress('u')},
			wantCursor:   0,
			wantFiltered: 1, // only "eu-west-1" contains "eu"
		},
		{
			name:         "backspace removes last filter char",
			items:        items,
			msgs:         []tea.Msg{keyPress('e'), keyPress('u'), specialKey(tea.KeyBackspace)},
			wantCursor:   0,
			wantFiltered: 4, // "e" matches all items (east, west, eu, southeast)
		},
		{
			name:         "backspace on empty filter is no-op",
			items:        items,
			msgs:         []tea.Msg{specialKey(tea.KeyBackspace)},
			wantCursor:   0,
			wantFiltered: 4,
		},
		{
			name:         "filter with no matches",
			items:        items,
			msgs:         []tea.Msg{keyPress('z'), keyPress('z'), keyPress('z')},
			wantCursor:   0,
			wantFiltered: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPicker("Select Region", tt.items)
			for _, msg := range tt.msgs {
				p, _ = p.Update(msg)
			}
			assert.Equal(t, tt.wantCursor, p.Cursor())
			assert.Equal(t, tt.wantFiltered, p.FilteredCount())
		})
	}
}

func TestPickerEnterSelectsItem(t *testing.T) {
	p := NewPicker("Pick", []string{"alpha", "beta", "gamma"})
	p, _ = p.Update(keyPress('j')) // cursor at 1 -> "beta"
	p, cmd := p.Update(specialKey(tea.KeyEnter))

	require.NotNil(t, cmd)
	msg := cmd()
	result, ok := msg.(PickerResult)
	require.True(t, ok)
	assert.Equal(t, "beta", result.Selected)
	assert.False(t, result.Canceled)
}

func TestPickerEscapeCancels(t *testing.T) {
	p := NewPicker("Pick", []string{"alpha", "beta"})
	_, cmd := p.Update(specialKey(tea.KeyEscape))

	require.NotNil(t, cmd)
	msg := cmd()
	result, ok := msg.(PickerResult)
	require.True(t, ok)
	assert.True(t, result.Canceled)
	assert.Empty(t, result.Selected)
}

func TestPickerEnterWithFilteredList(t *testing.T) {
	p := NewPicker("Pick", []string{"alpha", "beta", "gamma"})
	// Type "al" to filter -> should get only "alpha"
	p, _ = p.Update(keyPress('a'))
	p, _ = p.Update(keyPress('l'))
	p, cmd := p.Update(specialKey(tea.KeyEnter))

	require.NotNil(t, cmd)
	msg := cmd()
	result, ok := msg.(PickerResult)
	require.True(t, ok)
	assert.Equal(t, "alpha", result.Selected)
}

func TestPickerEnterWithEmptyFilteredList(t *testing.T) {
	p := NewPicker("Pick", []string{"alpha", "beta"})
	p, _ = p.Update(keyPress('z'))
	p, _ = p.Update(keyPress('z'))
	_, cmd := p.Update(specialKey(tea.KeyEnter))
	assert.Nil(t, cmd)
}

func TestPickerInitReturnsNil(t *testing.T) {
	p := NewPicker("Pick", []string{"a", "b"})
	assert.Nil(t, p.Init())
}

func TestPickerView(t *testing.T) {
	p := NewPicker("Select Region", []string{"us-east-1", "us-west-2"})
	view := p.View()
	assert.Contains(t, view, "Select Region")
	assert.Contains(t, view, "us-east-1")
	assert.Contains(t, view, "us-west-2")
}
