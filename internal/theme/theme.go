package theme

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

// Theme defines the color palette for the entire application.
// All colors are adaptive, automatically switching between light and dark
// variants based on the terminal background.
type Theme struct {
	// Status indicators
	StatusHealthy  compat.AdaptiveColor
	StatusWarning  compat.AdaptiveColor
	StatusCritical compat.AdaptiveColor

	// Toast notifications
	ToastInfo    compat.AdaptiveColor
	ToastWarning compat.AdaptiveColor
	ToastError   compat.AdaptiveColor

	// UI elements
	Border      compat.AdaptiveColor
	Accent      compat.AdaptiveColor
	Muted       compat.AdaptiveColor
	Text        compat.AdaptiveColor
	TextInverse compat.AdaptiveColor

	// Table
	TableHeader   compat.AdaptiveColor
	TableSelected compat.AdaptiveColor
	TableRowAlt   compat.AdaptiveColor

	// Tabs
	TabActive   compat.AdaptiveColor
	TabInactive compat.AdaptiveColor

	// Breadcrumb
	BreadcrumbSep   compat.AdaptiveColor
	BreadcrumbText  compat.AdaptiveColor
	BreadcrumbStale compat.AdaptiveColor
}

// Default is the built-in theme with sensible adaptive colors.
var Default = Theme{
	// Status indicators
	StatusHealthy:  ac("#15803d", "#4ade80"),
	StatusWarning:  ac("#a16207", "#facc15"),
	StatusCritical: ac("#dc2626", "#f87171"),

	// Toast notifications
	ToastInfo:    ac("#2563eb", "#60a5fa"),
	ToastWarning: ac("#a16207", "#facc15"),
	ToastError:   ac("#dc2626", "#f87171"),

	// UI elements
	Border:      ac("#d4d4d4", "#525252"),
	Accent:      ac("#2563eb", "#60a5fa"),
	Muted:       ac("#a3a3a3", "#737373"),
	Text:        ac("#1c1917", "#e7e5e4"),
	TextInverse: ac("#e7e5e4", "#1c1917"),

	// Table
	TableHeader:   ac("#2563eb", "#60a5fa"),
	TableSelected: ac("#dbeafe", "#1e3a5f"),
	TableRowAlt:   ac("#f5f5f4", "#292524"),

	// Tabs
	TabActive:   ac("#2563eb", "#60a5fa"),
	TabInactive: ac("#a3a3a3", "#737373"),

	// Breadcrumb
	BreadcrumbSep:   ac("#a3a3a3", "#737373"),
	BreadcrumbText:  ac("#1c1917", "#e7e5e4"),
	BreadcrumbStale: ac("#a3a3a3", "#737373"),
}

// ac is a shorthand for creating an AdaptiveColor from light/dark hex strings.
func ac(light, dark string) compat.AdaptiveColor {
	return compat.AdaptiveColor{
		Light: lipgloss.Color(light),
		Dark:  lipgloss.Color(dark),
	}
}
