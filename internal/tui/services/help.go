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
	HelpContextEKSDetail
	HelpContextK8sPods
	HelpContextK8sNodes
	HelpContextK8sLogs
	HelpContextEC2
	HelpContextEC2Detail
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
	case HelpContextEKSDetail:
		title = "Keybindings — EKS Cluster"
		bindings = []helpBinding{
			{"Tab", "Next tab"},
			{"Shift+Tab", "Prev tab"},
			{"1-8", "Jump to tab"},
			{"N", "Change namespace (K8s tabs)"},
			{"r", "Refresh data"},
			{"c", "Copy ID"},
			{"C", "Copy ARN"},
			{"j/k", "Navigate up/down"},
			{"Esc", "Go back"},
			{"?", "Toggle this help"},
			{"q", "Quit"},
		}
	case HelpContextK8sPods:
		title = "Keybindings — K8s Pods"
		bindings = []helpBinding{
			{"Enter", "Pod details"},
			{"e", "View YAML spec"},
			{"x", "Exec into pod (prompts cmd)"},
			{"l", "View pod logs"},
			{"f", "Port forward"},
			{"F", "List port forwards"},
			{"/", "Filter rows"},
			{"n/p", "Next/prev page"},
			{"r", "Refresh data"},
			{"c", "Copy ID"},
			{"j/k", "Navigate up/down"},
			{"Esc", "Go back"},
			{"?", "Toggle this help"},
			{"q", "Quit"},
		}
	case HelpContextK8sNodes:
		title = "Keybindings — K8s Nodes"
		bindings = []helpBinding{
			{"Enter", "Node details"},
			{"e", "View YAML spec"},
			{"x", "Debug exec into node"},
			{"/", "Filter rows"},
			{"n/p", "Next/prev page"},
			{"r", "Refresh data"},
			{"c", "Copy ID"},
			{"j/k", "Navigate up/down"},
			{"Esc", "Go back"},
			{"?", "Toggle this help"},
			{"q", "Quit"},
		}
	case HelpContextK8sLogs:
		title = "Keybindings — Pod Logs"
		bindings = []helpBinding{
			{"f", "Toggle follow"},
			{"w", "Toggle word wrap"},
			{"/", "Search"},
			{"n", "Next match"},
			{"N", "Prev match"},
			{"j/k", "Scroll up/down"},
			{"Esc", "Go back"},
			{"?", "Toggle this help"},
			{"q", "Quit"},
		}
	case HelpContextEC2:
		title = "Keybindings — EC2 Instances"
		bindings = []helpBinding{
			{"Enter", "Instance detail"},
			{"x", "SSM connect"},
			{"/", "Filter rows"},
			{"n/p", "Next/prev page"},
			{"r", "Refresh data"},
			{"c", "Copy ID"},
			{"C", "Copy ARN"},
			{"j/k", "Navigate up/down"},
			{"Esc", "Go back"},
			{"?", "Toggle this help"},
			{"q", "Quit"},
		}
	case HelpContextEC2Detail:
		title = "Keybindings — EC2 Instance"
		bindings = []helpBinding{
			{"Tab", "Next tab"},
			{"Shift+Tab", "Prev tab"},
			{"1-4", "Jump to tab"},
			{"x", "SSM connect"},
			{"v", "Navigate to VPC"},
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
	case *EKSClusterDetailView:
		return HelpContextEKSDetail
	case *EKSLogView:
		return HelpContextK8sLogs
	case *RootView:
		return HelpContextRoot
	case *VPCDetailView:
		return HelpContextDetail
	case *EC2DetailView:
		return HelpContextEC2Detail
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
