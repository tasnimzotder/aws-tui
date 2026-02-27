package services

import (
	"context"
	"fmt"
	"unsafe"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"tasnim.dev/aws-tui/internal/tui/theme"
)

// tableDataMsg carries async-fetched data back to the correct TableView instance.
type tableDataMsg struct {
	viewID uintptr
	items  any
}

type tableMoreDataMsg struct {
	viewID  uintptr
	items   any
	hasMore bool
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
	PageSize     int                                        // 0 = default (20)
	LoadMoreFunc func(ctx context.Context) ([]T, bool, error) // optional: returns items, hasMore, error
}

const defaultPageSize = 20

// TableView is a generic, reusable table-based view.
type TableView[T any] struct {
	config      TableViewConfig[T]
	items       []T
	table       table.Model
	spinner     spinner.Model
	loading     bool
	err         error
	allRows     []table.Row
	displayRows []table.Row
	pageItems   []T
	currentPage int
	pageSize    int
	hasMore     bool
	loadingMore bool
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

	ps := cfg.PageSize
	if ps <= 0 {
		ps = defaultPageSize
	}

	return &TableView[T]{
		config:   cfg,
		table:    t,
		spinner:  theme.NewSpinner(),
		loading:  true,
		pageSize: ps,
	}
}

func (v *TableView[T]) totalPages() int {
	n := len(v.displayRows)
	if n == 0 {
		return 0
	}
	return (n + v.pageSize - 1) / v.pageSize
}

func (v *TableView[T]) applyPage() {
	if len(v.displayRows) == 0 {
		v.table.SetRows(nil)
		v.pageItems = nil
		return
	}
	start := v.currentPage * v.pageSize
	end := min(start+v.pageSize, len(v.displayRows))
	v.table.SetRows(v.displayRows[start:end])
	// Build pageItems for correct cursor→item mapping
	if start < len(v.items) {
		v.pageItems = v.items[start:min(end, len(v.items))]
	}
	v.table.SetCursor(0)
}

func (v *TableView[T]) nextPage() {
	if v.currentPage < v.totalPages()-1 {
		v.currentPage++
		v.applyPage()
	}
}

func (v *TableView[T]) prevPage() {
	if v.currentPage > 0 {
		v.currentPage--
		v.applyPage()
	}
}

func (v *TableView[T]) paginationStatus() string {
	total := v.totalPages()
	itemCount := len(v.displayRows)
	if total <= 1 && !v.hasMore {
		return ""
	}
	countStr := fmt.Sprintf("%d", itemCount)
	if v.hasMore {
		countStr += "+"
	}
	if total <= 1 {
		return fmt.Sprintf("(%s items) L to load more", countStr)
	}
	status := fmt.Sprintf("Page %d/%d (%s items)", v.currentPage+1, total, countStr)
	if v.hasMore {
		status += "  L to load more"
	}
	return status
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

func (v *TableView[T]) fetchMore() tea.Cmd {
	id := v.viewID()
	loadMore := v.config.LoadMoreFunc
	return func() tea.Msg {
		items, hasMore, err := loadMore(context.Background())
		if err != nil {
			return errViewMsg{err: err}
		}
		return tableMoreDataMsg{viewID: id, items: items, hasMore: hasMore}
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
		v.displayRows = rows
		v.currentPage = 0
		v.applyPage()
		if v.config.LoadMoreFunc != nil {
			v.hasMore = true // assume more until first LoadMore returns hasMore=false
		}
		return v, nil

	case errViewMsg:
		v.err = msg.err
		v.loading = false
		return v, nil

	case tableMoreDataMsg:
		if msg.viewID != v.viewID() {
			return v, nil
		}
		newItems, ok := msg.items.([]T)
		if !ok {
			return v, nil
		}
		v.loadingMore = false
		v.hasMore = msg.hasMore
		v.items = append(v.items, newItems...)
		newRows := make([]table.Row, len(newItems))
		for i, item := range newItems {
			newRows[i] = v.config.RowMapper(item)
		}
		v.allRows = append(v.allRows, newRows...)
		v.displayRows = v.allRows
		v.applyPage()
		return v, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "r":
			v.loading = true
			v.err = nil
			return v, tea.Batch(v.spinner.Tick, v.fetchData())
		case "n":
			v.nextPage()
			return v, nil
		case "p":
			v.prevPage()
			return v, nil
		case "L":
			if v.config.LoadMoreFunc != nil && v.hasMore && !v.loadingMore {
				v.loadingMore = true
				return v, v.fetchMore()
			}
		case "enter":
			if v.config.OnEnter != nil {
				idx := v.table.Cursor()
				if idx >= 0 && idx < len(v.pageItems) {
					return v, v.config.OnEnter(v.pageItems[idx])
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
	out := ""
	if v.config.SummaryFunc != nil {
		out = v.config.SummaryFunc(v.items) + "\n\n"
	}
	out += v.table.View()
	if status := v.paginationStatus(); status != "" {
		out += "\n" + theme.MutedStyle.Render(status)
	}
	return out
}

// FilterableView implementation
func (v *TableView[T]) AllRows() []table.Row    { return v.allRows }
func (v *TableView[T]) SetRows(rows []table.Row) {
	v.displayRows = rows
	v.currentPage = 0
	v.applyPage()
}

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
	if idx >= 0 && idx < len(v.pageItems) {
		return v.config.CopyIDFunc(v.pageItems[idx])
	}
	return ""
}

func (v *TableView[T]) CopyARN() string {
	if v.config.CopyARNFunc == nil {
		return v.CopyID()
	}
	idx := v.table.Cursor()
	if idx >= 0 && idx < len(v.pageItems) {
		return v.config.CopyARNFunc(v.pageItems[idx])
	}
	return ""
}
