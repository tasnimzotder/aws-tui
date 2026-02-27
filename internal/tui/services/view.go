package services

import (
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"tasnim.dev/aws-tui/internal/tui/theme"
)

// View represents a navigable screen in the services browser.
type View interface {
	Title() string
	View() string
	Update(msg tea.Msg) (View, tea.Cmd)
	Init() tea.Cmd
}

// PushViewMsg signals the model to push a new view onto the stack.
type PushViewMsg struct{ View View }

// PopViewMsg signals the model to pop the current view.
type PopViewMsg struct{}

// FilterableView is implemented by views that support text filtering.
type FilterableView interface {
	View
	AllRows() []table.Row
	SetRows(rows []table.Row)
}

// CopyableView is implemented by views that support clipboard copy.
type CopyableView interface {
	View
	CopyID() string
	CopyARN() string
}

// ResizableView is implemented by views that adapt to window size.
type ResizableView interface {
	View
	SetSize(width, height int)
}

// errViewMsg is a shared message for async error reporting.
type errViewMsg struct{ err error }

func renderBreadcrumb(titles []string) string {
	parts := make([]string, len(titles))
	for i, t := range titles {
		parts[i] = theme.BreadcrumbStyle.Render(t)
	}
	return strings.Join(parts, theme.BreadcrumbSepStyle.Render(" › "))
}
