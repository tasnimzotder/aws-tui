package services

import (
	"context"
	"fmt"
	"testing"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
)

type testItem struct {
	id string
}

func newTestTableView(pageSize int) *TableView[testItem] {
	return NewTableView(TableViewConfig[testItem]{
		PageSize:  pageSize,
		Columns:   []table.Column{{Title: "ID", Width: 10}},
		FetchFunc: func(ctx context.Context) ([]testItem, error) { return nil, nil },
		RowMapper: func(item testItem) table.Row { return table.Row{item.id} },
	})
}

func newTestTableViewWithData(count, pageSize int) *TableView[testItem] {
	tv := newTestTableView(pageSize)
	items := make([]testItem, count)
	rows := make([]table.Row, count)
	for i := range count {
		items[i] = testItem{id: fmt.Sprintf("item-%d", i)}
		rows[i] = table.Row{fmt.Sprintf("item-%d", i)}
	}
	tv.items = items
	tv.allRows = rows
	tv.displayRows = rows
	tv.loading = false
	tv.currentPage = 0
	tv.applyPage()
	return tv
}

func TestDefaultPageSize(t *testing.T) {
	tv := NewTableView(TableViewConfig[testItem]{
		Columns:   []table.Column{{Title: "ID", Width: 10}},
		FetchFunc: func(ctx context.Context) ([]testItem, error) { return nil, nil },
		RowMapper: func(item testItem) table.Row { return table.Row{item.id} },
	})
	if tv.pageSize != 20 {
		t.Errorf("default pageSize = %d, want 20", tv.pageSize)
	}
}

func TestCustomPageSize(t *testing.T) {
	tv := newTestTableView(50)
	if tv.pageSize != 50 {
		t.Errorf("custom pageSize = %d, want 50", tv.pageSize)
	}
}

func TestTotalPages(t *testing.T) {
	tests := []struct {
		name      string
		itemCount int
		pageSize  int
		want      int
	}{
		{"empty", 0, 20, 0},
		{"exact fit", 20, 20, 1},
		{"partial last page", 21, 20, 2},
		{"single item", 1, 20, 1},
		{"large set", 100, 20, 5},
		{"page size 1", 5, 1, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tv := newTestTableView(tt.pageSize)
			tv.displayRows = make([]table.Row, tt.itemCount)
			got := tv.totalPages()
			if got != tt.want {
				t.Errorf("totalPages() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestApplyPage(t *testing.T) {
	tv := newTestTableViewWithData(5, 3)

	// Page 0: first 3 rows
	rows := tv.table.Rows()
	if len(rows) != 3 {
		t.Fatalf("page 0: got %d rows, want 3", len(rows))
	}
	if rows[0][0] != "item-0" || rows[2][0] != "item-2" {
		t.Errorf("page 0: unexpected rows: %v", rows)
	}

	// Page 1: last 2 rows
	tv.currentPage = 1
	tv.applyPage()
	rows = tv.table.Rows()
	if len(rows) != 2 {
		t.Fatalf("page 1: got %d rows, want 2", len(rows))
	}
	if rows[0][0] != "item-3" || rows[1][0] != "item-4" {
		t.Errorf("page 1: unexpected rows: %v", rows)
	}
}

func TestApplyPageEmpty(t *testing.T) {
	tv := newTestTableViewWithData(0, 5)
	rows := tv.table.Rows()
	if len(rows) != 0 {
		t.Errorf("empty: got %d rows, want 0", len(rows))
	}
}

func TestNextPrevPage(t *testing.T) {
	tv := newTestTableViewWithData(5, 2) // 3 pages: [0,1], [2,3], [4]

	// Start at page 0
	if tv.currentPage != 0 {
		t.Fatalf("initial: currentPage = %d, want 0", tv.currentPage)
	}

	// Next → page 1
	tv.nextPage()
	if tv.currentPage != 1 {
		t.Errorf("after nextPage: currentPage = %d, want 1", tv.currentPage)
	}
	if tv.table.Rows()[0][0] != "item-2" {
		t.Errorf("page 1 first row = %s, want item-2", tv.table.Rows()[0][0])
	}

	// Next → page 2
	tv.nextPage()
	if tv.currentPage != 2 {
		t.Errorf("after nextPage x2: currentPage = %d, want 2", tv.currentPage)
	}

	// Next past last page: stays at 2
	tv.nextPage()
	if tv.currentPage != 2 {
		t.Errorf("past end: currentPage = %d, want 2", tv.currentPage)
	}

	// Prev → page 1
	tv.prevPage()
	if tv.currentPage != 1 {
		t.Errorf("after prevPage: currentPage = %d, want 1", tv.currentPage)
	}

	// Prev → page 0
	tv.prevPage()
	if tv.currentPage != 0 {
		t.Errorf("after prevPage x2: currentPage = %d, want 0", tv.currentPage)
	}

	// Prev past first: stays at 0
	tv.prevPage()
	if tv.currentPage != 0 {
		t.Errorf("past start: currentPage = %d, want 0", tv.currentPage)
	}
}

func TestKeyBindingsPagination(t *testing.T) {
	tv := newTestTableViewWithData(5, 2)

	// Press 'n' for next page
	updated, _ := tv.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	tvUp := updated.(*TableView[testItem])
	if tvUp.currentPage != 1 {
		t.Errorf("after 'n': currentPage = %d, want 1", tvUp.currentPage)
	}

	// Press 'p' for prev page
	updated, _ = tvUp.Update(tea.KeyPressMsg{Code: 'p', Text: "p"})
	tvUp = updated.(*TableView[testItem])
	if tvUp.currentPage != 0 {
		t.Errorf("after 'p': currentPage = %d, want 0", tvUp.currentPage)
	}
}

func TestDataLoadInitializesPagination(t *testing.T) {
	tv := newTestTableView(3)

	// Simulate data arriving via tableDataMsg
	items := make([]testItem, 5)
	for i := range 5 {
		items[i] = testItem{id: fmt.Sprintf("item-%d", i)}
	}
	msg := tableDataMsg{viewID: tv.viewID(), items: items}
	tv.Update(msg)

	if tv.currentPage != 0 {
		t.Errorf("currentPage = %d, want 0", tv.currentPage)
	}
	if len(tv.displayRows) != 5 {
		t.Errorf("displayRows len = %d, want 5", len(tv.displayRows))
	}
	// Table should show only first page (3 items)
	tableRows := tv.table.Rows()
	if len(tableRows) != 3 {
		t.Fatalf("table rows = %d, want 3 (page 0)", len(tableRows))
	}
	if tableRows[0][0] != "item-0" {
		t.Errorf("first row = %s, want item-0", tableRows[0][0])
	}
}

func TestSetRowsResetsPagination(t *testing.T) {
	tv := newTestTableViewWithData(10, 3)

	// Navigate to page 2
	tv.nextPage()
	tv.nextPage()
	if tv.currentPage != 2 {
		t.Fatalf("setup: currentPage = %d, want 2", tv.currentPage)
	}

	// Simulate filter applying a subset via SetRows
	tv.SetRows(tv.allRows[:4])

	if tv.currentPage != 0 {
		t.Errorf("after SetRows: currentPage = %d, want 0", tv.currentPage)
	}
	tableRows := tv.table.Rows()
	if len(tableRows) != 3 {
		t.Errorf("after SetRows: table rows = %d, want 3 (page 0 of 4)", len(tableRows))
	}
}

func TestOnEnterUsesPageItems(t *testing.T) {
	var entered string
	tv := newTestTableView(2)
	tv.config.OnEnter = func(item testItem) tea.Cmd {
		entered = item.id
		return nil
	}

	// Load 5 items
	items := make([]testItem, 5)
	rows := make([]table.Row, 5)
	for i := range 5 {
		items[i] = testItem{id: fmt.Sprintf("item-%d", i)}
		rows[i] = table.Row{fmt.Sprintf("item-%d", i)}
	}
	tv.items = items
	tv.allRows = rows
	tv.displayRows = rows
	tv.loading = false
	tv.applyPage()

	// Go to page 1
	tv.nextPage()

	// Press enter on first row of page 1 (should be item-2)
	tv.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if entered != "item-2" {
		t.Errorf("OnEnter got item %q, want item-2", entered)
	}
}

func TestCopyIDUsesPageItems(t *testing.T) {
	tv := newTestTableView(2)
	tv.config.CopyIDFunc = func(item testItem) string { return item.id }

	items := make([]testItem, 5)
	rows := make([]table.Row, 5)
	for i := range 5 {
		items[i] = testItem{id: fmt.Sprintf("item-%d", i)}
		rows[i] = table.Row{fmt.Sprintf("item-%d", i)}
	}
	tv.items = items
	tv.allRows = rows
	tv.displayRows = rows
	tv.loading = false
	tv.applyPage()

	// Go to page 1, cursor at 0 → should be item-2
	tv.nextPage()
	got := tv.CopyID()
	if got != "item-2" {
		t.Errorf("CopyID = %q, want item-2", got)
	}
}

func TestPaginationStatus(t *testing.T) {
	tests := []struct {
		name        string
		total       int
		pageSize    int
		currentPage int
		want        string
	}{
		{"page 1 of 3", 50, 20, 0, "Page 1/3 (50 items)"},
		{"page 2 of 3", 50, 20, 1, "Page 2/3 (50 items)"},
		{"single page", 5, 20, 0, ""},
		{"empty", 0, 20, 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tv := newTestTableView(tt.pageSize)
			tv.displayRows = make([]table.Row, tt.total)
			tv.currentPage = tt.currentPage
			got := tv.paginationStatus()
			if got != tt.want {
				t.Errorf("paginationStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestViewIncludesPaginationStatus(t *testing.T) {
	tv := newTestTableViewWithData(25, 10)
	output := tv.View()
	if !contains(output, "Page 1/3") {
		t.Errorf("View() should contain 'Page 1/3', got: %s", output)
	}
	if !contains(output, "25 items") {
		t.Errorf("View() should contain '25 items', got: %s", output)
	}
}

func TestViewNoPaginationStatusForSinglePage(t *testing.T) {
	tv := newTestTableViewWithData(3, 20)
	output := tv.View()
	if contains(output, "Page") {
		t.Errorf("View() should not show pagination for single page, got: %s", output)
	}
}

func TestHelpIncludesPaginationKeys(t *testing.T) {
	output := renderHelp(HelpContextTable, 80, 24)
	if !contains(output, "Next page") {
		t.Errorf("help should show 'Next page', got: %s", output)
	}
	if !contains(output, "Prev page") {
		t.Errorf("help should show 'Prev page', got: %s", output)
	}
}

func TestLoadMore(t *testing.T) {
	loadCalls := 0
	tv := NewTableView(TableViewConfig[testItem]{
		PageSize: 3,
		Columns:  []table.Column{{Title: "ID", Width: 10}},
		FetchFunc: func(ctx context.Context) ([]testItem, error) {
			return []testItem{{id: "a"}, {id: "b"}}, nil
		},
		RowMapper: func(item testItem) table.Row { return table.Row{item.id} },
		LoadMoreFunc: func(ctx context.Context) ([]testItem, bool, error) {
			loadCalls++
			return []testItem{{id: "c"}, {id: "d"}}, false, nil
		},
	})

	// Simulate initial data load
	msg := tableDataMsg{viewID: tv.viewID(), items: []testItem{{id: "a"}, {id: "b"}}}
	tv.Update(msg)

	if len(tv.items) != 2 {
		t.Fatalf("initial items = %d, want 2", len(tv.items))
	}
	if !tv.hasMore {
		t.Fatal("hasMore should be true after initial load with LoadMoreFunc")
	}

	// Press 'L' to load more — triggers async, returns a Cmd
	_, cmd := tv.Update(tea.KeyPressMsg{Code: 'L', Text: "L"})
	if cmd == nil {
		t.Fatal("pressing L should return a cmd")
	}

	// Simulate the response
	moreMsg := tableMoreDataMsg{viewID: tv.viewID(), items: []testItem{{id: "c"}, {id: "d"}}, hasMore: false}
	tv.Update(moreMsg)

	if len(tv.items) != 4 {
		t.Fatalf("after load more: items = %d, want 4", len(tv.items))
	}
	if len(tv.allRows) != 4 {
		t.Errorf("after load more: allRows = %d, want 4", len(tv.allRows))
	}
	if tv.hasMore {
		t.Error("hasMore should be false after load returned hasMore=false")
	}
}

func TestLoadMoreStatus(t *testing.T) {
	tv := newTestTableViewWithData(5, 3)
	tv.hasMore = true
	status := tv.paginationStatus()
	if !contains(status, "5+") {
		t.Errorf("status should show '5+' when hasMore, got: %s", status)
	}
	if !contains(status, "L to load more") {
		t.Errorf("status should show 'L to load more', got: %s", status)
	}
}

func TestLoadMoreNoFuncNoop(t *testing.T) {
	tv := newTestTableViewWithData(5, 3)
	// No LoadMoreFunc set
	_, cmd := tv.Update(tea.KeyPressMsg{Code: 'L', Text: "L"})
	if cmd != nil {
		t.Error("L should be noop when LoadMoreFunc is nil")
	}
}

func TestHelpIncludesLoadMore(t *testing.T) {
	output := renderHelp(HelpContextTable, 80, 24)
	if !contains(output, "Load more") {
		t.Errorf("help should show 'Load more', got: %s", output)
	}
}

func TestKeyHandlers(t *testing.T) {
	var handled string
	tv := NewTableView(TableViewConfig[testItem]{
		PageSize:  20,
		Columns:   []table.Column{{Title: "ID", Width: 10}},
		FetchFunc: func(ctx context.Context) ([]testItem, error) { return nil, nil },
		RowMapper: func(item testItem) table.Row { return table.Row{item.id} },
		KeyHandlers: map[string]func(testItem) tea.Cmd{
			"v": func(item testItem) tea.Cmd {
				handled = item.id
				return nil
			},
		},
	})

	// Load items
	items := []testItem{{id: "file-a"}, {id: "file-b"}}
	rows := []table.Row{{"file-a"}, {"file-b"}}
	tv.items = items
	tv.allRows = rows
	tv.displayRows = rows
	tv.loading = false
	tv.applyPage()

	// Press 'v' should trigger handler for first item
	tv.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	if handled != "file-a" {
		t.Errorf("KeyHandler got item %q, want file-a", handled)
	}
}

func TestKeyHandlersWithPagination(t *testing.T) {
	var handled string
	tv := NewTableView(TableViewConfig[testItem]{
		PageSize:  2,
		Columns:   []table.Column{{Title: "ID", Width: 10}},
		FetchFunc: func(ctx context.Context) ([]testItem, error) { return nil, nil },
		RowMapper: func(item testItem) table.Row { return table.Row{item.id} },
		KeyHandlers: map[string]func(testItem) tea.Cmd{
			"d": func(item testItem) tea.Cmd {
				handled = item.id
				return nil
			},
		},
	})

	items := make([]testItem, 5)
	rows := make([]table.Row, 5)
	for i := range 5 {
		items[i] = testItem{id: fmt.Sprintf("item-%d", i)}
		rows[i] = table.Row{fmt.Sprintf("item-%d", i)}
	}
	tv.items = items
	tv.allRows = rows
	tv.displayRows = rows
	tv.loading = false
	tv.applyPage()

	// Go to page 1
	tv.nextPage()

	// Press 'd' should get item-2 (first item on page 1)
	tv.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if handled != "item-2" {
		t.Errorf("KeyHandler got item %q, want item-2", handled)
	}
}

func TestRefreshCancelsPreviousContext(t *testing.T) {
	var capturedCtx context.Context
	tv := NewTableView(TableViewConfig[testItem]{
		PageSize: 20,
		Columns:  []table.Column{{Title: "ID", Width: 10}},
		FetchFunc: func(ctx context.Context) ([]testItem, error) {
			capturedCtx = ctx
			return nil, nil
		},
		RowMapper: func(item testItem) table.Row { return table.Row{item.id} },
	})

	// First fetch via Init — run the returned cmd to capture context
	cmd := tv.fetchData()
	if cmd == nil {
		t.Fatal("fetchData should return a cmd")
	}
	cmd() // execute to capture ctx
	firstCtx := capturedCtx

	// Refresh creates a new context and cancels the old one
	cmd = tv.fetchData()
	cmd()

	// The first context should now be cancelled
	if firstCtx.Err() != context.Canceled {
		t.Error("first context should be cancelled after second fetchData")
	}
	// The second context should still be active
	if capturedCtx.Err() != nil {
		t.Error("second context should not be cancelled")
	}
}

func TestCancelMethod(t *testing.T) {
	tv := NewTableView(TableViewConfig[testItem]{
		PageSize:  20,
		Columns:   []table.Column{{Title: "ID", Width: 10}},
		FetchFunc: func(ctx context.Context) ([]testItem, error) { return nil, nil },
		RowMapper: func(item testItem) table.Row { return table.Row{item.id} },
	})

	// Cancel on nil should not panic
	tv.Cancel()

	// After fetchData, cancel should work
	tv.fetchData()
	if tv.cancel == nil {
		t.Fatal("cancel should be set")
	}
	tv.Cancel()
	// Should not panic on double cancel
	tv.Cancel()
}

func TestHelpContextS3Objects(t *testing.T) {
	output := renderHelp(HelpContextS3Objects, 80, 24)
	if !contains(output, "View content") {
		t.Errorf("S3 help should show 'View content', got: %s", output)
	}
	if !contains(output, "Download") {
		t.Errorf("S3 help should show 'Download', got: %s", output)
	}
}
