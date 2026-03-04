package ui

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Column defines a column in a TableView.
type Column[T any] struct {
	Title string
	Width int
	Field func(T) string
}

// TableView is a generic, navigable, sortable, filterable table component.
type TableView[T any] struct {
	columns []Column[T]
	allItems []T
	filtered []T
	idFunc  func(T) string

	cursor     int
	sortCol    int
	sortAsc    bool
	filtering  bool
	filterText string
	scrollX    int

	width  int
	height int
}

// NewTableView creates a TableView with the given columns, items, and ID function.
func NewTableView[T any](cols []Column[T], items []T, idFunc func(T) string) TableView[T] {
	tv := TableView[T]{
		columns:  cols,
		allItems: make([]T, len(items)),
		idFunc:   idFunc,
		sortAsc:  true,
	}
	copy(tv.allItems, items)
	tv.applyFilterAndSort()
	return tv
}

// Cursor returns the current cursor position.
func (tv TableView[T]) Cursor() int {
	return tv.cursor
}

// SortColumn returns the current sort column index.
func (tv TableView[T]) SortColumn() int {
	return tv.sortCol
}

// SortAsc returns true if sorting is ascending.
func (tv TableView[T]) SortAsc() bool {
	return tv.sortAsc
}

// Filtering returns true if filter mode is active.
func (tv TableView[T]) Filtering() bool {
	return tv.filtering
}

// FilteredCount returns the number of visible (filtered) items.
func (tv TableView[T]) FilteredCount() int {
	return len(tv.filtered)
}

// SelectedItem returns the item at the current cursor position.
// Returns the zero value of T if there are no items.
func (tv TableView[T]) SelectedItem() T {
	if len(tv.filtered) == 0 {
		var zero T
		return zero
	}
	return tv.filtered[tv.cursor]
}

// SelectedID returns the ID of the item at the current cursor position.
func (tv TableView[T]) SelectedID() string {
	if len(tv.filtered) == 0 {
		return ""
	}
	return tv.idFunc(tv.filtered[tv.cursor])
}

// SetItems replaces the item list and reapplies filter and sort.
func (tv *TableView[T]) SetItems(items []T) {
	tv.allItems = make([]T, len(items))
	copy(tv.allItems, items)
	tv.applyFilterAndSort()
}

// SetSize sets the viewport dimensions.
func (tv *TableView[T]) SetSize(w, h int) {
	tv.width = w
	tv.height = h
}

// Init satisfies the Bubble Tea Model interface.
func (tv TableView[T]) Init() tea.Cmd {
	return nil
}

// Update handles key messages for navigation, sorting, and filtering.
func (tv TableView[T]) Update(msg tea.Msg) (TableView[T], tea.Cmd) {
	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return tv, nil
	}

	if tv.filtering {
		return tv.updateFilterMode(km), nil
	}

	switch km.String() {
	case "j", "down":
		if len(tv.filtered) > 0 && tv.cursor < len(tv.filtered)-1 {
			tv.cursor++
		}
	case "k", "up":
		if tv.cursor > 0 {
			tv.cursor--
		}
	case "h":
		if tv.scrollX > 0 {
			tv.scrollX -= 4
			if tv.scrollX < 0 {
				tv.scrollX = 0
			}
		}
	case "l":
		maxScroll := tv.totalWidth() - tv.width
		if maxScroll < 0 {
			maxScroll = 0
		}
		tv.scrollX += 4
		if tv.scrollX > maxScroll {
			tv.scrollX = maxScroll
		}
	case "s":
		tv.sortCol = (tv.sortCol + 1) % len(tv.columns)
		tv.applyFilterAndSort()
	case "S":
		tv.sortAsc = !tv.sortAsc
		tv.applyFilterAndSort()
	case "/":
		tv.filtering = true
		tv.filterText = ""
	}

	return tv, nil
}

func (tv TableView[T]) updateFilterMode(km tea.KeyPressMsg) TableView[T] {
	switch km.String() {
	case "esc":
		tv.filtering = false
		tv.filterText = ""
		tv.applyFilterAndSort()
	case "enter":
		tv.filtering = false
	case "backspace":
		if len(tv.filterText) > 0 {
			tv.filterText = tv.filterText[:len(tv.filterText)-1]
			tv.applyFilterAndSort()
		}
	default:
		if km.Text != "" {
			tv.filterText += km.Text
			tv.applyFilterAndSort()
		}
	}
	return tv
}

func (tv *TableView[T]) applyFilterAndSort() {
	// Filter
	if tv.filterText == "" {
		tv.filtered = make([]T, len(tv.allItems))
		copy(tv.filtered, tv.allItems)
	} else {
		query := strings.ToLower(tv.filterText)
		tv.filtered = nil
		for _, item := range tv.allItems {
			if tv.matchesFilter(item, query) {
				tv.filtered = append(tv.filtered, item)
			}
		}
	}

	// Sort
	if len(tv.columns) > 0 && tv.sortCol < len(tv.columns) {
		field := tv.columns[tv.sortCol].Field
		asc := tv.sortAsc
		sort.SliceStable(tv.filtered, func(i, j int) bool {
			a, b := field(tv.filtered[i]), field(tv.filtered[j])
			if asc {
				return a < b
			}
			return a > b
		})
	}

	// Clamp cursor
	if tv.cursor >= len(tv.filtered) {
		if len(tv.filtered) > 0 {
			tv.cursor = len(tv.filtered) - 1
		} else {
			tv.cursor = 0
		}
	}
}

func (tv *TableView[T]) matchesFilter(item T, query string) bool {
	for _, col := range tv.columns {
		val := strings.ToLower(col.Field(item))
		if strings.Contains(val, query) {
			return true
		}
	}
	return false
}

// View renders the table with header, rows, and optional filter bar.
var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	selectedRowStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("236")).
				Foreground(lipgloss.Color("255"))

	normalRowStyle = lipgloss.NewStyle()

	filterBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	sortIndicator = " ▲"
	sortIndicatorDesc = " ▼"
)

// totalWidth returns the total character width of all columns plus separators.
func (tv TableView[T]) totalWidth() int {
	w := 0
	for _, col := range tv.columns {
		w += col.Width
	}
	// Add 1 space separator between each column
	if len(tv.columns) > 1 {
		w += len(tv.columns) - 1
	}
	return w
}

// hscroll applies horizontal scroll offset to a rendered line.
func hscroll(line string, offset int) string {
	runes := []rune(line)
	if offset >= len(runes) {
		return ""
	}
	return string(runes[offset:])
}

func (tv TableView[T]) View() string {
	var b strings.Builder

	// Header
	var headerParts []string
	for i, col := range tv.columns {
		title := col.Title
		if i == tv.sortCol {
			if tv.sortAsc {
				title += sortIndicator
			} else {
				title += sortIndicatorDesc
			}
		}
		headerParts = append(headerParts, padRight(title, col.Width))
	}
	headerLine := strings.Join(headerParts, " ")
	b.WriteString(headerStyle.Render(hscroll(headerLine, tv.scrollX)))
	b.WriteString("\n")

	// Rows
	for i, item := range tv.filtered {
		var rowParts []string
		for _, col := range tv.columns {
			rowParts = append(rowParts, padRight(col.Field(item), col.Width))
		}
		row := hscroll(strings.Join(rowParts, " "), tv.scrollX)
		if i == tv.cursor {
			row = selectedRowStyle.Render(row)
		} else {
			row = normalRowStyle.Render(row)
		}
		b.WriteString(row)
		b.WriteString("\n")
	}

	// Filter bar
	if tv.filtering {
		b.WriteString(filterBarStyle.Render(fmt.Sprintf("/%s", tv.filterText)))
	} else if tv.filterText != "" {
		b.WriteString(filterBarStyle.Render(fmt.Sprintf("[filter: %s]", tv.filterText)))
	}

	return b.String()
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}
