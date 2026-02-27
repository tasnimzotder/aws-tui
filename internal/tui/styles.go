package tui

import (
	"charm.land/lipgloss/v2"

	"tasnim.dev/aws-tui/internal/tui/theme"
)

var (
	// Cost-specific styles that compose from the shared theme
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Primary)

	headerStyle = theme.HeaderStyle

	metricLabelStyle = theme.MutedStyle

	metricValueStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(theme.Success)

	forecastValueStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(theme.Warning)

	profileStyle = lipgloss.NewStyle().
			Foreground(theme.Secondary)

	helpStyle = theme.HelpStyle

	errorStyle = theme.ErrorStyle

	anomalyHeaderStyle = lipgloss.NewStyle().
				Foreground(theme.Error).
				Bold(true)

	anomalyStyle = lipgloss.NewStyle().
			Foreground(theme.Warning)

	dashboardStyle = theme.DashboardStyle
)
