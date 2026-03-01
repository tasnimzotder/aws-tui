package services

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"tasnim.dev/aws-tui/internal/tui/theme"
)

// TabController manages a horizontal tab bar with lazy initialization of tab
// views, keyboard navigation, and resizing. It is designed to be embedded in
// detail views that follow the dashboard + tab pattern.
type TabController struct {
	ActiveTab    int
	TabNames     []string
	TabViews     []View
	InitTab      func(idx int) View
	BeforeSwitch func(idx int)
}

// NewTabController creates a TabController. initTab is called lazily the first
// time a tab is selected.
func NewTabController(names []string, initTab func(int) View) *TabController {
	return &TabController{
		TabNames: names,
		TabViews: make([]View, len(names)),
		InitTab:  initTab,
	}
}

// SwitchTab switches to the given tab index, lazily initializing if needed.
// Returns a tea.Cmd from the tab's Init if it was just created.
func (tc *TabController) SwitchTab(idx int) tea.Cmd {
	if tc.BeforeSwitch != nil {
		tc.BeforeSwitch(idx)
	}
	tc.ActiveTab = idx
	if tc.TabViews[idx] == nil && tc.InitTab != nil {
		tc.TabViews[idx] = tc.InitTab(idx)
	}
	if tc.TabViews[idx] != nil {
		return tc.TabViews[idx].Init()
	}
	return nil
}

// HandleKey handles tab/shift+tab/number key navigation. Returns handled=true
// if the key was consumed by the tab controller.
func (tc *TabController) HandleKey(key string) (handled bool, cmd tea.Cmd) {
	n := len(tc.TabNames)
	switch key {
	case "tab":
		return true, tc.SwitchTab((tc.ActiveTab + 1) % n)
	case "shift+tab":
		next := tc.ActiveTab - 1
		if next < 0 {
			next = n - 1
		}
		return true, tc.SwitchTab(next)
	}

	// Number keys: 1-9 map to tabs 0-8, 0 maps to tab 9
	if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
		var idx int
		if key == "0" {
			idx = 9
		} else {
			idx = int(key[0]-'0') - 1
		}
		if idx < n {
			return true, tc.SwitchTab(idx)
		}
	}

	return false, nil
}

// RenderTabBar renders the horizontal tab bar with numbered labels.
func (tc *TabController) RenderTabBar() string {
	var tabs []string
	for i, name := range tc.TabNames {
		key := fmt.Sprintf("%d", i+1)
		if i == 9 {
			key = "0"
		}
		label := key + ":" + name
		if i == tc.ActiveTab {
			tabs = append(tabs, theme.TabActiveStyle.Render(label))
		} else {
			tabs = append(tabs, theme.TabInactiveStyle.Render(label))
		}
	}
	return theme.TabBarStyle.Render(strings.Join(tabs, ""))
}

// ActiveView returns the current tab's view, or nil if not initialized.
func (tc *TabController) ActiveView() View {
	return tc.TabViews[tc.ActiveTab]
}

// ResizeActive resizes the currently active tab view if it implements
// ResizableView.
func (tc *TabController) ResizeActive(width, contentHeight int) {
	if v := tc.TabViews[tc.ActiveTab]; v != nil {
		if rv, ok := v.(ResizableView); ok {
			rv.SetSize(width, contentHeight)
		}
	}
}

// DelegateUpdate forwards a message to the active tab view and returns the
// updated view and command.
func (tc *TabController) DelegateUpdate(msg tea.Msg) tea.Cmd {
	if v := tc.TabViews[tc.ActiveTab]; v != nil {
		updated, cmd := v.Update(msg)
		tc.TabViews[tc.ActiveTab] = updated
		return cmd
	}
	return nil
}
