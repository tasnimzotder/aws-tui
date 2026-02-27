package services

import (
	"strings"

	"charm.land/lipgloss/v2"

	"tasnim.dev/aws-tui/internal/tui/theme"
)

// HelpContext determines which keybinding set to show.
type HelpContext int

const (
	HelpContextRoot       HelpContext = iota
	HelpContextTable
	HelpContextDetail
	HelpContextS3Objects
	HelpContextTextView
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
	case HelpContextS3Objects:
		title = "Keybindings — S3 Objects"
		bindings = []helpBinding{
			{"Enter", "Open folder"},
			{"v", "View content"},
			{"d", "Download"},
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
	case HelpContextTextView:
		title = "Keybindings — Text Viewer"
		bindings = []helpBinding{
			{"/", "Search"},
			{"n", "Next match"},
			{"N", "Prev match"},
			{"w", "Toggle word wrap"},
			{"j/k", "Scroll up/down"},
			{"Esc", "Go back / close search"},
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

// HelpContextProvider is implemented by views that provide a custom help context.
type HelpContextProvider interface {
	HelpContext() *HelpContext
}

// detectHelpContext determines the help context from the current view.
func detectHelpContext(v View) HelpContext {
	// Check if view provides its own help context
	if hcp, ok := v.(HelpContextProvider); ok {
		if ctx := hcp.HelpContext(); ctx != nil {
			return *ctx
		}
	}

	switch v.(type) {
	case *TextView:
		return HelpContextTextView
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
