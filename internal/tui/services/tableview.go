package services

import (
	"context"
	"fmt"
	"unsafe"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"tasnim.dev/aws-tui/internal/tui/theme"
)

// tableDataMsg carries async-fetched data back to the correct TableView instance.
type tableDataMsg struct {
	viewID uintptr
	items  any
}

// TableViewConfig defines all the customizable parts of a table-based view.
type TableViewConfig[T any] struct {
	Title        string
	LoadingText  string
	Columns      []table.Column
	FetchFunc    func(ctx context.Context) ([]T, error)
	RowMapper    func(item T) table.Row
	CopyIDFunc   func(item T) string
	CopyARNFunc  func(item T) string
	SummaryFunc  func(items []T) string // optional, rendered above table
	OnEnter      func(item T) tea.Cmd   // optional, nil = no drill-down
	HeightOffset int                     // lines consumed by summary
}

// TableView is a generic, reusable table-based view.
type TableView[T any] struct {
	config  TableViewConfig[T]
	items   []T
	table   table.Model
	spinner spinner.Model
	loading bool
	err     error
	allRows []table.Row
}

// NewTableView creates a new TableView from the given config.
func NewTableView[T any](cfg TableViewConfig[T]) *TableView[T] {
	t := table.New(
		table.WithColumns(cfg.Columns),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(12),
	)
	t.SetStyles(theme.DefaultTableStyles())

	return &TableView[T]{
		config:  cfg,
		table:   t,
		spinner: theme.NewSpinner(),
		loading: true,
	}
}

func (v *TableView[T]) viewID() uintptr {
	return uintptr(unsafe.Pointer(v))
}

func (v *TableView[T]) Title() string { return v.config.Title }

func (v *TableView[T]) Init() tea.Cmd {
	return tea.Batch(v.spinner.Tick, v.fetchData())
}

func (v *TableView[T]) fetchData() tea.Cmd {
	id := v.viewID()
	fetch := v.config.FetchFunc
	return func() tea.Msg {
		items, err := fetch(context.Background())
		if err != nil {
			return errViewMsg{err: err}
		}
		return tableDataMsg{viewID: id, items: items}
	}
}

func (v *TableView[T]) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tableDataMsg:
		if msg.viewID != v.viewID() {
			return v, nil
		}
		items, ok := msg.items.([]T)
		if !ok {
			return v, nil
		}
		v.items = items
		v.loading = false
		rows := make([]table.Row, len(items))
		for i, item := range items {
			rows[i] = v.config.RowMapper(item)
		}
		v.allRows = rows
		v.table.SetRows(rows)
		return v, nil

	case errViewMsg:
		v.err = msg.err
		v.loading = false
		return v, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			v.loading = true
			v.err = nil
			return v, tea.Batch(v.spinner.Tick, v.fetchData())
		case "enter":
			if v.config.OnEnter != nil {
				idx := v.table.Cursor()
				if idx >= 0 && idx < len(v.items) {
					return v, v.config.OnEnter(v.items[idx])
				}
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		return v, cmd
	}

	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	return v, cmd
}

func (v *TableView[T]) View() string {
	if v.loading {
		return v.spinner.View() + " " + v.config.LoadingText
	}
	if v.err != nil {
		return theme.ErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	if v.config.SummaryFunc != nil {
		return v.config.SummaryFunc(v.items) + "\n\n" + v.table.View()
	}
	return v.table.View()
}

// FilterableView implementation
func (v *TableView[T]) AllRows() []table.Row    { return v.allRows }
func (v *TableView[T]) SetRows(rows []table.Row) { v.table.SetRows(rows) }

// ResizableView implementation
func (v *TableView[T]) SetSize(width, height int) {
	v.table.SetWidth(width)
	v.table.SetHeight(height - v.config.HeightOffset)
}

// CopyableView implementation
func (v *TableView[T]) CopyID() string {
	if v.config.CopyIDFunc == nil {
		return ""
	}
	idx := v.table.Cursor()
	if idx >= 0 && idx < len(v.items) {
		return v.config.CopyIDFunc(v.items[idx])
	}
	return ""
}

func (v *TableView[T]) CopyARN() string {
	if v.config.CopyARNFunc == nil {
		return v.CopyID()
	}
	idx := v.table.Cursor()
	if idx >= 0 && idx < len(v.items) {
		return v.config.CopyARNFunc(v.items[idx])
	}
	return ""
}
