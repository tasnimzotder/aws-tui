package services

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// mockView is a minimal View implementation for testing.
type mockTabView struct {
	title         string
	initCalled    bool
	width, height int
}

func (m *mockTabView) Title() string                        { return m.title }
func (m *mockTabView) View() string                         { return m.title + " content" }
func (m *mockTabView) Init() tea.Cmd                        { m.initCalled = true; return nil }
func (m *mockTabView) Update(msg tea.Msg) (View, tea.Cmd)   { return m, nil }
func (m *mockTabView) SetSize(width, height int)            { m.width = width; m.height = height }

func newTestTabController() *TabController {
	return NewTabController(
		[]string{"Tab1", "Tab2", "Tab3"},
		func(idx int) View {
			return &mockTabView{title: "Tab" + string(rune('1'+idx))}
		},
	)
}

func TestTabController_SwitchTab(t *testing.T) {
	tc := newTestTabController()
	tc.SwitchTab(0) // init tab 0
	if tc.ActiveTab != 0 {
		t.Errorf("ActiveTab = %d, want 0", tc.ActiveTab)
	}
	if tc.TabViews[0] == nil {
		t.Error("expected tab 0 to be initialized")
	}
	// Tab 1 should still be nil (lazy)
	if tc.TabViews[1] != nil {
		t.Error("expected tab 1 to be nil (lazy)")
	}

	tc.SwitchTab(2)
	if tc.ActiveTab != 2 {
		t.Errorf("ActiveTab = %d, want 2", tc.ActiveTab)
	}
	if tc.TabViews[2] == nil {
		t.Error("expected tab 2 to be initialized")
	}
}

func TestTabController_HandleKey_Tab(t *testing.T) {
	tc := newTestTabController()
	tc.SwitchTab(0)

	handled, _ := tc.HandleKey("tab")
	if !handled {
		t.Error("expected tab key to be handled")
	}
	if tc.ActiveTab != 1 {
		t.Errorf("ActiveTab = %d, want 1", tc.ActiveTab)
	}

	// Wrap around
	tc.SwitchTab(2)
	tc.HandleKey("tab")
	if tc.ActiveTab != 0 {
		t.Errorf("ActiveTab = %d, want 0 (wrap)", tc.ActiveTab)
	}
}

func TestTabController_HandleKey_ShiftTab(t *testing.T) {
	tc := newTestTabController()
	tc.SwitchTab(0)

	handled, _ := tc.HandleKey("shift+tab")
	if !handled {
		t.Error("expected shift+tab to be handled")
	}
	if tc.ActiveTab != 2 {
		t.Errorf("ActiveTab = %d, want 2 (wrap backward)", tc.ActiveTab)
	}
}

func TestTabController_HandleKey_NumberKeys(t *testing.T) {
	tc := newTestTabController()
	tc.SwitchTab(0)

	handled, _ := tc.HandleKey("2")
	if !handled {
		t.Error("expected number key to be handled")
	}
	if tc.ActiveTab != 1 {
		t.Errorf("ActiveTab = %d, want 1", tc.ActiveTab)
	}

	handled, _ = tc.HandleKey("3")
	if !handled {
		t.Error("expected number key 3 to be handled")
	}
	if tc.ActiveTab != 2 {
		t.Errorf("ActiveTab = %d, want 2", tc.ActiveTab)
	}
}

func TestTabController_HandleKey_NumberKey0(t *testing.T) {
	names := make([]string, 10)
	for i := range names {
		names[i] = "Tab"
	}
	tc := NewTabController(names, func(idx int) View {
		return &mockTabView{title: "tab"}
	})
	tc.SwitchTab(0)

	handled, _ := tc.HandleKey("0")
	if !handled {
		t.Error("expected 0 key to be handled for 10-tab controller")
	}
	if tc.ActiveTab != 9 {
		t.Errorf("ActiveTab = %d, want 9", tc.ActiveTab)
	}
}

func TestTabController_HandleKey_Invalid(t *testing.T) {
	tc := newTestTabController()
	tc.SwitchTab(0)

	handled, _ := tc.HandleKey("x")
	if handled {
		t.Error("expected unknown key to not be handled")
	}
}

func TestTabController_RenderTabBar(t *testing.T) {
	tc := newTestTabController()
	tc.SwitchTab(0)
	bar := tc.RenderTabBar()
	for _, name := range tc.TabNames {
		if !strings.Contains(bar, name) {
			t.Errorf("tab bar missing tab name %q", name)
		}
	}
}

func TestTabController_BeforeSwitch(t *testing.T) {
	called := false
	calledIdx := -1
	tc := newTestTabController()
	tc.BeforeSwitch = func(idx int) {
		called = true
		calledIdx = idx
	}
	tc.SwitchTab(2)
	if !called {
		t.Error("expected BeforeSwitch to be called")
	}
	if calledIdx != 2 {
		t.Errorf("BeforeSwitch called with idx=%d, want 2", calledIdx)
	}
}

func TestTabController_ResizeActive(t *testing.T) {
	tc := newTestTabController()
	tc.SwitchTab(0)
	tc.ResizeActive(200, 50)

	mv := tc.TabViews[0].(*mockTabView)
	if mv.width != 200 || mv.height != 50 {
		t.Errorf("expected size 200x50, got %dx%d", mv.width, mv.height)
	}
}
