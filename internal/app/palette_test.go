package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func keyPress(char rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{
		Code: char,
		Text: string(char),
	}
}

func specialKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

func testEntries() []PaletteEntry {
	return []PaletteEntry{
		{Title: "Switch Region", Keywords: []string{"aws", "region"}},
		{Title: "Switch Profile", Keywords: []string{"aws", "credentials"}},
		{Title: "Refresh", Keywords: []string{"reload", "update"}},
		{Title: "Quit", Keywords: []string{"exit", "close"}},
	}
}

func TestCommandPaletteFiltering(t *testing.T) {
	tests := []struct {
		name      string
		query     []tea.Msg
		wantCount int
	}{
		{
			name:      "empty query shows all entries",
			query:     nil,
			wantCount: 4,
		},
		{
			name:      "typing filters by title",
			query:     []tea.Msg{keyPress('R'), keyPress('e'), keyPress('f')},
			wantCount: 1, // "Refresh"
		},
		{
			name:      "typing filters by keyword",
			query:     []tea.Msg{keyPress('r'), keyPress('e'), keyPress('l'), keyPress('o'), keyPress('a'), keyPress('d')},
			wantCount: 1, // "Refresh" via keyword "reload"
		},
		{
			name:      "partial match",
			query:     []tea.Msg{keyPress('s'), keyPress('w'), keyPress('i')},
			wantCount: 2, // "Switch Region" and "Switch Profile"
		},
		{
			name:      "no match returns 0",
			query:     []tea.Msg{keyPress('z'), keyPress('z'), keyPress('z')},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewCommandPalette(testEntries())
			p.Open()
			for _, msg := range tt.query {
				p, _ = p.Update(msg)
			}
			assert.Equal(t, tt.wantCount, p.ResultCount())
		})
	}
}

func TestCommandPaletteOpenClose(t *testing.T) {
	p := NewCommandPalette(testEntries())

	assert.False(t, p.Active(), "palette should start inactive")

	p.Open()
	assert.True(t, p.Active(), "palette should be active after Open")

	p.Close()
	assert.False(t, p.Active(), "palette should be inactive after Close")
}

func TestCommandPaletteOpenResetsState(t *testing.T) {
	p := NewCommandPalette(testEntries())
	p.Open()

	// Type a query and move cursor
	p, _ = p.Update(keyPress('r'))
	p, _ = p.Update(specialKey(tea.KeyDown))

	// Close and reopen
	p.Close()
	p.Open()

	assert.Equal(t, 4, p.ResultCount(), "results should be reset after reopen")
}

func TestCommandPaletteEscapeCloses(t *testing.T) {
	p := NewCommandPalette(testEntries())
	p.Open()

	p, _ = p.Update(specialKey(tea.KeyEscape))
	assert.False(t, p.Active(), "escape should close palette")
}

func TestCommandPaletteEnterSelectsEntry(t *testing.T) {
	called := false
	entries := []PaletteEntry{
		{Title: "Alpha", Action: func() tea.Cmd {
			called = true
			return nil
		}},
		{Title: "Beta", Action: nil},
	}

	p := NewCommandPalette(entries)
	p.Open()

	p, cmd := p.Update(specialKey(tea.KeyEnter))
	assert.False(t, p.Active(), "enter should close palette")
	require.NotNil(t, cmd)

	msg := cmd()
	sel, ok := msg.(PaletteSelectMsg)
	require.True(t, ok)
	assert.Equal(t, "Alpha", sel.Entry.Title)

	// Invoke the action
	sel.Entry.Action()
	assert.True(t, called)
}

func TestCommandPaletteEnterOnEmptyReturnsNil(t *testing.T) {
	p := NewCommandPalette(testEntries())
	p.Open()

	// Filter to no results
	p, _ = p.Update(keyPress('z'))
	p, _ = p.Update(keyPress('z'))
	p, _ = p.Update(keyPress('z'))

	_, cmd := p.Update(specialKey(tea.KeyEnter))
	assert.Nil(t, cmd, "enter on empty results should return nil cmd")
}

func TestCommandPaletteArrowNavigation(t *testing.T) {
	p := NewCommandPalette(testEntries())
	p.Open()

	// Move down
	p, _ = p.Update(specialKey(tea.KeyDown))
	p, _ = p.Update(specialKey(tea.KeyDown))

	// Select and verify it's the third entry
	p, cmd := p.Update(specialKey(tea.KeyEnter))
	require.NotNil(t, cmd)
	msg := cmd()
	sel, ok := msg.(PaletteSelectMsg)
	require.True(t, ok)
	assert.Equal(t, "Refresh", sel.Entry.Title)
}

func TestCommandPaletteArrowUpBounded(t *testing.T) {
	p := NewCommandPalette(testEntries())
	p.Open()

	// Try moving up from 0
	p, _ = p.Update(specialKey(tea.KeyUp))

	p, cmd := p.Update(specialKey(tea.KeyEnter))
	require.NotNil(t, cmd)
	msg := cmd()
	sel := msg.(PaletteSelectMsg)
	assert.Equal(t, "Switch Region", sel.Entry.Title)
}

func TestCommandPaletteBackspace(t *testing.T) {
	p := NewCommandPalette(testEntries())
	p.Open()

	// Type "zzz" -> 0 results, then backspace all -> 4 results
	p, _ = p.Update(keyPress('z'))
	p, _ = p.Update(keyPress('z'))
	p, _ = p.Update(keyPress('z'))
	assert.Equal(t, 0, p.ResultCount())

	p, _ = p.Update(specialKey(tea.KeyBackspace))
	p, _ = p.Update(specialKey(tea.KeyBackspace))
	p, _ = p.Update(specialKey(tea.KeyBackspace))
	assert.Equal(t, 4, p.ResultCount())
}

func TestCommandPaletteSetEntries(t *testing.T) {
	p := NewCommandPalette(testEntries())
	p.Open()
	assert.Equal(t, 4, p.ResultCount())

	newEntries := []PaletteEntry{
		{Title: "Only One"},
	}
	p.SetEntries(newEntries)
	assert.Equal(t, 1, p.ResultCount())
}

func TestCommandPaletteView(t *testing.T) {
	p := NewCommandPalette(testEntries())
	p.Open()

	view := p.View()
	assert.Contains(t, view, "Switch Region")
	assert.Contains(t, view, "Quit")
}

func TestCommandPaletteInactiveIgnoresInput(t *testing.T) {
	p := NewCommandPalette(testEntries())
	// Don't open — palette is inactive

	p, cmd := p.Update(keyPress('r'))
	assert.Nil(t, cmd)
	assert.Equal(t, 4, p.ResultCount(), "inactive palette should not filter")
}
