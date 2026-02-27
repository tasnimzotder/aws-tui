package services

import (
	"strings"

	"charm.land/lipgloss/v2"

	"tasnim.dev/aws-tui/internal/tui/theme"
)

// HelpContext determines which keybinding set to show.
type HelpContext int

const (
	HelpContextRoot   HelpContext = iota
	HelpContextTable
	HelpContextDetail
)

type helpBinding struct {
	key  string
	desc string
}

func renderHelp(ctx HelpContext, width, height int) string {
	var title string
	var bindings []helpBinding

	switch ctx {
	case HelpContextRoot:
		title = "Keybindings — Root"
		bindings = []helpBinding{
			{"Enter", "Select service"},
			{"j/k", "Navigate up/down"},
			{"?", "Toggle this help"},
			{"q", "Quit"},
		}
	case HelpContextTable:
		title = "Keybindings — Table"
		bindings = []helpBinding{
			{"Enter", "Drill down"},
			{"/", "Filter rows"},
			{"n", "Next page"},
			{"p", "Prev page"},
			{"L", "Load more"},
			{"r", "Refresh data"},
			{"c", "Copy ID"},
			{"C", "Copy ARN"},
			{"j/k", "Navigate up/down"},
			{"Esc", "Go back"},
			{"?", "Toggle this help"},
			{"q", "Quit"},
		}
	case HelpContextDetail:
		title = "Keybindings — Task Detail"
		bindings = []helpBinding{
			{"Tab/1/2", "Switch tabs"},
			{"t", "Toggle log tailing"},
			{"r", "Refresh data"},
			{"c", "Copy ID"},
			{"C", "Copy ARN"},
			{"j/k", "Scroll up/down"},
			{"Esc", "Go back"},
			{"?", "Toggle this help"},
			{"q", "Quit"},
		}
	}

	var b strings.Builder
	b.WriteString(theme.HelpTitleStyle.Render(title) + "\n")
	for _, binding := range bindings {
		b.WriteString(theme.HelpKeyStyle.Render(binding.key) + theme.HelpDescStyle.Render(binding.desc) + "\n")
	}

	box := theme.HelpBoxStyle.Render(b.String())
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

// detectHelpContext determines the help context from the current view.
func detectHelpContext(v View) HelpContext {
	switch v.(type) {
	case *TaskDetailView:
		return HelpContextDetail
	case *RootView:
		return HelpContextRoot
	case *VPCSubMenuView:
		return HelpContextRoot
	case *ECSServiceSubMenuView:
		return HelpContextRoot
	case *IAMSubMenuView:
		return HelpContextRoot
	case *IAMUserSubMenuView:
		return HelpContextRoot
	case *IAMRoleSubMenuView:
		return HelpContextRoot
	}
	if _, ok := v.(FilterableView); ok {
		return HelpContextTable
	}
	return HelpContextRoot
}
