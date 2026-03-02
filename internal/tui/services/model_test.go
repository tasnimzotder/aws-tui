package services

import (
	"context"
	"testing"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"tasnim.dev/aws-tui/internal/config"
)

// newTestModel creates a Model with a refreshable TableView on the stack.
func newTestModel() Model {
	cfg := &config.Config{AutoRefreshInterval: 5}
	m := Model{
		profile:         "test",
		region:          "us-east-1",
		refreshInterval: cfg.RefreshInterval(),
		width:           120,
		height:          40,
	}

	// Push a simple TableView that implements RefreshableView
	tv := NewTableView(TableViewConfig[testItem]{
		PageSize:  20,
		Columns:   []table.Column{{Title: "ID", Width: 10}},
		FetchFunc: func(ctx context.Context) ([]testItem, error) { return nil, nil },
		RowMapper: func(item testItem) table.Row { return table.Row{item.id} },
	})
	tv.loading = false
	m.stack = []View{tv}
	return m
}

func TestAutoRefresh_ToggleOn(t *testing.T) {
	m := newTestModel()

	// Press 'a' to toggle auto-refresh on
	result, cmd := m.updateNormalKey(tea.KeyPressMsg{Code: 'a', Text: "a"})
	model := result.(Model)

	if !model.autoRefresh {
		t.Error("autoRefresh should be true after pressing 'a'")
	}
	if model.autoRefreshGen != 1 {
		t.Errorf("autoRefreshGen = %d, want 1", model.autoRefreshGen)
	}
	if cmd == nil {
		t.Error("should return a tick cmd")
	}
	if model.nextRefreshAt.IsZero() {
		t.Error("nextRefreshAt should be set")
	}
}

func TestAutoRefresh_ToggleOff(t *testing.T) {
	m := newTestModel()
	m.autoRefresh = true
	m.autoRefreshGen = 1

	// Press 'a' to toggle off
	result, _ := m.updateNormalKey(tea.KeyPressMsg{Code: 'a', Text: "a"})
	model := result.(Model)

	if model.autoRefresh {
		t.Error("autoRefresh should be false after toggling off")
	}
}

func TestAutoRefresh_TickBeforeExpiry(t *testing.T) {
	m := newTestModel()
	m.autoRefresh = true
	m.autoRefreshGen = 1
	m.nextRefreshAt = time.Now().Add(10 * time.Second) // 10s in the future

	result, cmd := m.Update(autoRefreshTickMsg{gen: 1})
	model := result.(Model)

	// Should NOT trigger refresh, just reschedule tick
	if cmd == nil {
		t.Error("should reschedule tick")
	}
	// autoRefresh should still be on
	if !model.autoRefresh {
		t.Error("autoRefresh should remain true")
	}
}

func TestAutoRefresh_TickAtExpiry(t *testing.T) {
	m := newTestModel()
	m.autoRefresh = true
	m.autoRefreshGen = 1
	m.nextRefreshAt = time.Now().Add(-1 * time.Millisecond) // slightly in the past

	result, cmd := m.Update(autoRefreshTickMsg{gen: 1})
	model := result.(Model)

	// Should trigger refresh — nextRefreshAt should be reset to future
	if model.nextRefreshAt.Before(time.Now()) {
		t.Error("nextRefreshAt should be reset to future after trigger")
	}
	if cmd == nil {
		t.Error("should return batch cmd with tick + trigger")
	}
}

func TestAutoRefresh_TickWrongGen(t *testing.T) {
	m := newTestModel()
	m.autoRefresh = true
	m.autoRefreshGen = 2

	// Tick with old gen should be ignored
	_, cmd := m.Update(autoRefreshTickMsg{gen: 1})
	if cmd != nil {
		t.Error("tick with wrong gen should return nil cmd")
	}
}

func TestAutoRefresh_TickWhenDisabled(t *testing.T) {
	m := newTestModel()
	m.autoRefresh = false

	_, cmd := m.Update(autoRefreshTickMsg{gen: 0})
	if cmd != nil {
		t.Error("tick when disabled should return nil cmd")
	}
}

func TestAutoRefresh_TriggerCallsRefresh(t *testing.T) {
	m := newTestModel()
	m.autoRefresh = true

	// The top of stack is a TableView which implements RefreshableView
	_, cmd := m.Update(autoRefreshTriggerMsg{})

	if cmd == nil {
		t.Error("trigger should return Refresh() cmd from TableView")
	}

	// Verify the TableView is now in loading state
	tv := m.stack[0].(*TableView[testItem])
	if !tv.loading {
		t.Error("TableView should be in loading state after Refresh()")
	}
}

func TestAutoRefresh_TriggerNonRefreshableView(t *testing.T) {
	m := newTestModel()
	m.autoRefresh = true
	// Replace stack with a non-refreshable view (RootView)
	m.stack = []View{&mockPlainView{}}

	_, cmd := m.Update(autoRefreshTriggerMsg{})

	if cmd != nil {
		t.Error("trigger on non-refreshable view should return nil cmd")
	}
}

func TestAutoRefresh_TriggerWhenDisabled(t *testing.T) {
	m := newTestModel()
	m.autoRefresh = false

	_, cmd := m.Update(autoRefreshTriggerMsg{})

	if cmd != nil {
		t.Error("trigger when disabled should return nil cmd")
	}
}

func TestAutoRefresh_TriggerEmptyStack(t *testing.T) {
	m := newTestModel()
	m.autoRefresh = true
	m.stack = nil

	_, cmd := m.Update(autoRefreshTriggerMsg{})

	if cmd != nil {
		t.Error("trigger with empty stack should return nil cmd")
	}
}
