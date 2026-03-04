package app

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

var (
	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	statusBarSepStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))
)

// StatusBar renders contextual information at the bottom of the screen.
type StatusBar struct {
	region      string
	profile     string
	autoRefresh bool
	nextRefresh time.Duration
	offline     bool
}

// NewStatusBar creates a StatusBar with the given region and profile.
func NewStatusBar(region, profile string) StatusBar {
	return StatusBar{
		region:  region,
		profile: profile,
	}
}

// SetRegion updates the displayed region.
func (s *StatusBar) SetRegion(region string) {
	s.region = region
}

// SetProfile updates the displayed profile.
func (s *StatusBar) SetProfile(profile string) {
	s.profile = profile
}

// SetAutoRefresh sets whether auto-refresh is enabled.
func (s *StatusBar) SetAutoRefresh(on bool) {
	s.autoRefresh = on
}

// SetNextRefresh sets the countdown until the next refresh.
func (s *StatusBar) SetNextRefresh(d time.Duration) {
	s.nextRefresh = d
}

// SetOffline sets the offline indicator.
func (s *StatusBar) SetOffline(offline bool) {
	s.offline = offline
}

// View renders the status bar to the given width.
func (s StatusBar) View(width int) string {
	sep := statusBarSepStyle.Render(" │ ")

	var segments []string
	segments = append(segments, statusBarStyle.Render(s.region))
	segments = append(segments, statusBarStyle.Render(s.profile))

	if s.autoRefresh {
		secs := int(s.nextRefresh.Seconds())
		segments = append(segments, statusBarStyle.Render(fmt.Sprintf("● %ds", secs)))
	}

	if s.offline {
		segments = append(segments, statusBarStyle.Render("offline"))
	}

	segments = append(segments, statusBarStyle.Render("? help"))

	return strings.Join(segments, sep)
}
