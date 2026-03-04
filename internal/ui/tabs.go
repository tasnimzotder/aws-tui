package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Padding(0, 2)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Padding(0, 2)

	tabSeparator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238")).
			Render("|")
)

// TabController manages a horizontal tab bar with keyboard navigation.
type TabController struct {
	titles []string
	active int
}

// NewTabController creates a TabController with the given tab titles.
// The initial active tab is 0.
func NewTabController(titles []string) TabController {
	return TabController{
		titles: titles,
		active: 0,
	}
}

// Active returns the index of the currently active tab.
func (tc TabController) Active() int {
	return tc.active
}

// Count returns the number of tabs.
func (tc TabController) Count() int {
	return len(tc.titles)
}

// Update handles key events for tab navigation:
//   - ] next tab (wraps around)
//   - [ previous tab (wraps around)
//   - 1-9 jump to tab N-1 (ignored if out of range)
func (tc TabController) Update(msg tea.Msg) (TabController, tea.Cmd) {
	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return tc, nil
	}

	n := len(tc.titles)
	if n == 0 {
		return tc, nil
	}

	switch km.String() {
	case "]", "right":
		tc.active = (tc.active + 1) % n
	case "[", "left":
		tc.active = (tc.active - 1 + n) % n
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx := int(km.Code-'0') - 1
		if idx < n {
			tc.active = idx
		}
	}

	return tc, nil
}

// View renders the tab bar with active/inactive styling.
func (tc TabController) View() string {
	if len(tc.titles) == 0 {
		return ""
	}

	var tabs []string
	for i, title := range tc.titles {
		if i == tc.active {
			tabs = append(tabs, activeTabStyle.Render(title))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(title))
		}
	}

	return strings.Join(tabs, tabSeparator)
}
