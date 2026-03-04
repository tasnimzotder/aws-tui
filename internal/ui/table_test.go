package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testItem struct {
	id     string
	name   string
	status string
}

func testColumns() []Column[testItem] {
	return []Column[testItem]{
		{Title: "Name", Width: 20, Field: func(t testItem) string { return t.name }},
		{Title: "Status", Width: 10, Field: func(t testItem) string { return t.status }},
	}
}

func testItems() []testItem {
	return []testItem{
		{id: "1", name: "alpha", status: "running"},
		{id: "2", name: "beta", status: "pending"},
		{id: "3", name: "gamma", status: "running"},
	}
}

func testIDFunc(t testItem) string { return t.id }

func newTestTable() TableView[testItem] {
	return NewTableView(testColumns(), testItems(), testIDFunc)
}

func specialKeyPress(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

func TestTableView(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "initial cursor is 0",
			fn: func(t *testing.T) {
				tv := newTestTable()
				assert.Equal(t, 0, tv.Cursor())
			},
		},
		{
			name: "j moves cursor down",
			fn: func(t *testing.T) {
				tv := newTestTable()
				tv, _ = tv.Update(keyPress('j'))
				assert.Equal(t, 1, tv.Cursor())
			},
		},
		{
			name: "k moves cursor up",
			fn: func(t *testing.T) {
				tv := newTestTable()
				tv, _ = tv.Update(keyPress('j'))
				tv, _ = tv.Update(keyPress('j'))
				tv, _ = tv.Update(keyPress('k'))
				assert.Equal(t, 1, tv.Cursor())
			},
		},
		{
			name: "cursor bounded at top",
			fn: func(t *testing.T) {
				tv := newTestTable()
				tv, _ = tv.Update(keyPress('k'))
				assert.Equal(t, 0, tv.Cursor())
			},
		},
		{
			name: "cursor bounded at bottom",
			fn: func(t *testing.T) {
				tv := newTestTable()
				tv, _ = tv.Update(keyPress('j'))
				tv, _ = tv.Update(keyPress('j'))
				tv, _ = tv.Update(keyPress('j'))
				tv, _ = tv.Update(keyPress('j'))
				assert.Equal(t, 2, tv.Cursor()) // 3 items, max index 2
			},
		},
		{
			name: "s cycles sort column forward",
			fn: func(t *testing.T) {
				tv := newTestTable()
				assert.Equal(t, 0, tv.SortColumn())
				tv, _ = tv.Update(keyPress('s'))
				assert.Equal(t, 1, tv.SortColumn())
				tv, _ = tv.Update(keyPress('s'))
				assert.Equal(t, 0, tv.SortColumn())
			},
		},
		{
			name: "S toggles sort direction",
			fn: func(t *testing.T) {
				tv := newTestTable()
				assert.True(t, tv.SortAsc())
				tv, _ = tv.Update(keyPress('S'))
				assert.False(t, tv.SortAsc())
				tv, _ = tv.Update(keyPress('S'))
				assert.True(t, tv.SortAsc())
			},
		},
		{
			name: "after sorting items are reordered",
			fn: func(t *testing.T) {
				tv := newTestTable()
				// Default sort col 0 (Name), asc => alpha, beta, gamma
				assert.Equal(t, "alpha", tv.SelectedItem().name)
				// Toggle to desc
				tv, _ = tv.Update(keyPress('S'))
				assert.Equal(t, "gamma", tv.SelectedItem().name)
			},
		},
		{
			name: "/ enters filter mode",
			fn: func(t *testing.T) {
				tv := newTestTable()
				tv, _ = tv.Update(keyPress('/'))
				assert.True(t, tv.Filtering())
			},
		},
		{
			name: "typing in filter mode filters rows across all columns",
			fn: func(t *testing.T) {
				tv := newTestTable()
				tv, _ = tv.Update(keyPress('/'))
				// Type "pend" to match "pending" status
				for _, ch := range "pend" {
					tv, _ = tv.Update(keyPress(ch))
				}
				assert.Equal(t, 1, tv.FilteredCount())
				assert.Equal(t, "beta", tv.SelectedItem().name)
			},
		},
		{
			name: "filter matches any column",
			fn: func(t *testing.T) {
				tv := newTestTable()
				tv, _ = tv.Update(keyPress('/'))
				for _, ch := range "gamma" {
					tv, _ = tv.Update(keyPress(ch))
				}
				assert.Equal(t, 1, tv.FilteredCount())
				assert.Equal(t, "gamma", tv.SelectedItem().name)
			},
		},
		{
			name: "escape in filter mode clears filter and exits",
			fn: func(t *testing.T) {
				tv := newTestTable()
				tv, _ = tv.Update(keyPress('/'))
				for _, ch := range "alpha" {
					tv, _ = tv.Update(keyPress(ch))
				}
				assert.Equal(t, 1, tv.FilteredCount())
				tv, _ = tv.Update(specialKeyPress(tea.KeyEscape))
				assert.False(t, tv.Filtering())
				assert.Equal(t, 3, tv.FilteredCount()) // all items visible
			},
		},
		{
			name: "enter exits filter mode but keeps filter active",
			fn: func(t *testing.T) {
				tv := newTestTable()
				tv, _ = tv.Update(keyPress('/'))
				for _, ch := range "run" {
					tv, _ = tv.Update(keyPress(ch))
				}
				assert.Equal(t, 2, tv.FilteredCount())
				tv, _ = tv.Update(specialKeyPress(tea.KeyEnter))
				assert.False(t, tv.Filtering())
				assert.Equal(t, 2, tv.FilteredCount()) // filter still applied
			},
		},
		{
			name: "SelectedItem returns item at cursor",
			fn: func(t *testing.T) {
				tv := newTestTable()
				item := tv.SelectedItem()
				assert.Equal(t, "alpha", item.name)
				tv, _ = tv.Update(keyPress('j'))
				item = tv.SelectedItem()
				assert.Equal(t, "beta", item.name)
			},
		},
		{
			name: "SelectedID returns id of item at cursor",
			fn: func(t *testing.T) {
				tv := newTestTable()
				assert.Equal(t, "1", tv.SelectedID())
				tv, _ = tv.Update(keyPress('j'))
				assert.Equal(t, "2", tv.SelectedID())
			},
		},
		{
			name: "FilteredCount reflects filter results",
			fn: func(t *testing.T) {
				tv := newTestTable()
				assert.Equal(t, 3, tv.FilteredCount())
				tv, _ = tv.Update(keyPress('/'))
				for _, ch := range "running" {
					tv, _ = tv.Update(keyPress(ch))
				}
				assert.Equal(t, 2, tv.FilteredCount())
			},
		},
		{
			name: "SetItems replaces items and reapplies filter and sort",
			fn: func(t *testing.T) {
				tv := newTestTable()
				// Set sort to desc
				tv, _ = tv.Update(keyPress('S'))
				// Set filter
				tv, _ = tv.Update(keyPress('/'))
				for _, ch := range "run" {
					tv, _ = tv.Update(keyPress(ch))
				}
				tv, _ = tv.Update(specialKeyPress(tea.KeyEnter))

				newItems := []testItem{
					{id: "4", name: "delta", status: "running"},
					{id: "5", name: "epsilon", status: "stopped"},
				}
				tv.SetItems(newItems)
				// Filter "run" should match only delta
				assert.Equal(t, 1, tv.FilteredCount())
				// Sort desc on name => delta is only result
				assert.Equal(t, "delta", tv.SelectedItem().name)
			},
		},
		{
			name: "backspace in filter mode removes last char",
			fn: func(t *testing.T) {
				tv := newTestTable()
				tv, _ = tv.Update(keyPress('/'))
				for _, ch := range "xyz" {
					tv, _ = tv.Update(keyPress(ch))
				}
				assert.Equal(t, 0, tv.FilteredCount())
				// Backspace 3 times to clear
				tv, _ = tv.Update(specialKeyPress(tea.KeyBackspace))
				tv, _ = tv.Update(specialKeyPress(tea.KeyBackspace))
				tv, _ = tv.Update(specialKeyPress(tea.KeyBackspace))
				assert.Equal(t, 3, tv.FilteredCount())
			},
		},
		{
			name: "View contains header and rows",
			fn: func(t *testing.T) {
				tv := newTestTable()
				tv.SetSize(80, 24)
				view := tv.View()
				assert.Contains(t, view, "Name")
				assert.Contains(t, view, "Status")
				assert.Contains(t, view, "alpha")
				assert.Contains(t, view, "running")
			},
		},
		{
			name: "Init returns nil",
			fn: func(t *testing.T) {
				tv := newTestTable()
				cmd := tv.Init()
				assert.Nil(t, cmd)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn(t)
		})
	}
}

func TestTableViewEmpty(t *testing.T) {
	tv := NewTableView(testColumns(), []testItem{}, testIDFunc)
	assert.Equal(t, 0, tv.Cursor())
	assert.Equal(t, 0, tv.FilteredCount())

	// SelectedItem on empty should return zero value
	item := tv.SelectedItem()
	assert.Equal(t, "", item.name)

	// SelectedID on empty should return empty
	assert.Equal(t, "", tv.SelectedID())

	// View should not panic
	require.NotPanics(t, func() {
		tv.View()
	})
}
