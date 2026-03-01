package theme

import (
	"image/color"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
)

// Colors
var (
	Primary   = lipgloss.Color("#33A8FF")
	Secondary = lipgloss.Color("#163047")
	Muted     = lipgloss.Color("#6B7280")
	Success   = lipgloss.Color("#10B981")
	Warning   = lipgloss.Color("#F59E0B")
	Error     = lipgloss.Color("#EF4444")
)

// Shared styles
var (
	HeaderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(Muted).
			Padding(0, 1)

	DashboardStyle = lipgloss.NewStyle().
			Padding(1, 2)

	HelpStyle = lipgloss.NewStyle().
			Foreground(Muted).
			Padding(1, 0, 0, 0)

	ProfileStyle = lipgloss.NewStyle().
			Foreground(Secondary)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(Error).
			Bold(true)

	MutedStyle = lipgloss.NewStyle().
			Foreground(Muted)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(Success).
			Bold(true)

	FilterStyle = lipgloss.NewStyle().
			Foreground(Primary)

	CopiedStyle = lipgloss.NewStyle().
			Foreground(Success).
			Bold(true)

	BreadcrumbStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	BreadcrumbSepStyle = lipgloss.NewStyle().
				Foreground(Muted)

	TabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			Padding(0, 1)

	TabInactiveStyle = lipgloss.NewStyle().
				Foreground(Muted).
				Padding(0, 1)

	TabBarStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(Muted)

	HelpBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(1, 3)

	HelpTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			MarginBottom(1)

	HelpKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Secondary).
			Width(12)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB"))

	DashboardBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Primary).
				Padding(0, 1)

	DashboardTitleStyle = lipgloss.NewStyle().
				Bold(true)

	StatusBarStyle = lipgloss.NewStyle().
				Padding(0, 1)

	LoadingStyle = lipgloss.NewStyle().
				Foreground(Muted).
				PaddingLeft(2)
)

// StatusColor maps common AWS resource statuses to theme colors.
func StatusColor(status string) color.Color {
	switch strings.ToLower(status) {
	case "running", "active", "available", "in-service", "healthy",
		"create_complete", "update_complete", "attached":
		return Success
	case "stopped", "terminated", "failed", "unhealthy", "error",
		"delete_failed", "create_failed", "detached":
		return Error
	case "pending", "stopping", "creating", "updating", "draining",
		"modifying", "delete_in_progress", "provisioning", "in-progress":
		return Warning
	default:
		return Muted
	}
}

// RenderStatus renders a status string with a colored bullet.
func RenderStatus(status string) string {
	c := StatusColor(status)
	bullet := lipgloss.NewStyle().Foreground(c).Render("●")
	return bullet + " " + status
}

// DefaultTableStyles returns styled table styles using theme colors.
func DefaultTableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(Muted).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	return s
}

// SpinnerStyle returns a spinner configured with the primary color.
func SpinnerStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Primary)
}

// NewSpinner returns a new spinner with the theme style.
func NewSpinner() spinner.Model {
	return spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(SpinnerStyle()),
	)
}
