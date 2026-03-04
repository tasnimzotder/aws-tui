package app

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

var (
	breadcrumbStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Bold(true)

	breadcrumbTimestampStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))
)

// Breadcrumb renders a navigation breadcrumb trail with an optional timestamp.
type Breadcrumb struct{}

// NewBreadcrumb creates a new Breadcrumb.
func NewBreadcrumb() Breadcrumb {
	return Breadcrumb{}
}

// View renders the breadcrumb trail and optional "Updated Xs ago" timestamp.
func (b Breadcrumb) View(crumbs []string, lastUpdated time.Time, width int) string {
	left := breadcrumbStyle.Render(strings.Join(crumbs, " > "))

	if lastUpdated.IsZero() {
		return left
	}

	elapsed := time.Since(lastUpdated).Truncate(time.Second)
	right := breadcrumbTimestampStyle.Render(fmt.Sprintf("Updated %s ago", elapsed))

	// Calculate padding between left and right.
	// Use a rough visible-length estimate (strip ANSI would be better,
	// but for now use the raw string lengths of crumbs + timestamp).
	leftLen := len(strings.Join(crumbs, " > "))
	rightText := fmt.Sprintf("Updated %s ago", elapsed)
	rightLen := len(rightText)

	pad := width - leftLen - rightLen
	if pad < 1 {
		pad = 1
	}

	return left + strings.Repeat(" ", pad) + right
}
