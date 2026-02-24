package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	"tasnim.dev/aws-tui/internal/tui/theme"
)

type clearCopiedMsg struct{}

// Model is the root Bubble Tea model for the services browser.
type Model struct {
	client  *awsclient.ServiceClient
	profile string
	region  string
	stack   []View

	// Window size
	width  int
	height int

	// Filter state
	filtering   bool
	filterInput textinput.Model
	filterQuery string

	// Copy status
	copiedText string

	// Help overlay
	showHelp bool
}

// NewModel creates a new services browser model.
func NewModel(client *awsclient.ServiceClient, profile, region string) Model {
	root := NewRootView(client)

	ti := textinput.New()
	ti.Placeholder = "filter..."
	ti.CharLimit = 64

	return Model{
		client:      client,
		profile:     profile,
		region:      region,
		stack:       []View{root},
		filterInput: ti,
	}
}

func (m Model) Init() tea.Cmd {
	if len(m.stack) > 0 {
		return m.stack[len(m.stack)-1].Init()
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clearCopiedMsg:
		m.copiedText = ""
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Chrome takes ~8 lines: header(2) + filter(1) + padding(3) + help(2)
		contentHeight := msg.Height - 8
		if contentHeight < 3 {
			contentHeight = 3
		}
		// Resize all views in the stack so back-navigation uses correct size
		for _, v := range m.stack {
			if rv, ok := v.(ResizableView); ok {
				rv.SetSize(msg.Width-6, contentHeight)
			}
		}
		return m, nil

	case tea.KeyMsg:
		// Help overlay: ? toggles, Esc dismisses
		if m.showHelp {
			if msg.String() == "?" || msg.String() == "esc" || msg.String() == "q" {
				m.showHelp = false
			}
			return m, nil
		}

		if m.filtering {
			return m.updateFilterMode(msg)
		}

		return m.updateNormalKey(msg)

	case PushViewMsg:
		m.stack = append(m.stack, msg.View)
		// Apply current window size to new view
		if m.width > 0 && m.height > 0 {
			contentHeight := m.height - 8
			if contentHeight < 3 {
				contentHeight = 3
			}
			if rv, ok := msg.View.(ResizableView); ok {
				rv.SetSize(m.width-6, contentHeight)
			}
		}
		return m, msg.View.Init()

	case PopViewMsg:
		if len(m.stack) > 1 {
			m.stack = m.stack[:len(m.stack)-1]
		}
		return m, nil
	}

	// Delegate to current view
	if len(m.stack) > 0 {
		current := m.stack[len(m.stack)-1]
		updated, cmd := current.Update(msg)
		m.stack[len(m.stack)-1] = updated
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	// Build breadcrumb
	titles := make([]string, len(m.stack))
	for i, v := range m.stack {
		titles[i] = v.Title()
	}
	breadcrumb := renderBreadcrumb(titles)

	// Profile and region info
	profileText := "default"
	if m.profile != "" {
		profileText = m.profile
	}
	regionText := "default"
	if m.region != "" {
		regionText = m.region
	}
	info := theme.ProfileStyle.Render(fmt.Sprintf("profile: %s  region: %s", profileText, regionText))

	header := lipgloss.JoinHorizontal(lipgloss.Top, breadcrumb, "   ", info)

	// Filter bar
	filterBar := ""
	if m.filtering {
		filterBar = theme.FilterStyle.Render("/ ") + m.filterInput.View() + "\n"
	} else if m.filterQuery != "" {
		filterBar = theme.FilterStyle.Render(fmt.Sprintf("filter: %s", m.filterQuery)) + "\n"
	}

	// Current view content
	content := ""
	if len(m.stack) > 0 {
		content = m.stack[len(m.stack)-1].View()
	}

	// Help / copy status
	var help string
	if m.copiedText != "" {
		help = theme.CopiedStyle.Render(fmt.Sprintf("Copied: %s", m.copiedText))
	} else if m.filtering {
		help = theme.HelpStyle.Render("Enter to lock filter • Esc to clear")
	} else if len(m.stack) <= 1 {
		help = theme.HelpStyle.Render("Enter to select • ? for help • q to quit")
	} else {
		help = theme.HelpStyle.Render("Esc back • r refresh • / filter • c copy • ? help • q quit")
	}

	base := theme.DashboardStyle.Render(
		theme.HeaderStyle.Render(header) + "\n\n" +
			filterBar +
			content + "\n" +
			help,
	)

	// Help overlay
	if m.showHelp && len(m.stack) > 0 {
		ctx := detectHelpContext(m.stack[len(m.stack)-1])
		overlay := renderHelp(ctx, m.width, m.height)
		return overlay
	}

	return base
}

func (m Model) updateFilterMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filtering = false
		m.filterQuery = ""
		m.filterInput.SetValue("")
		if fv, ok := m.currentFilterable(); ok {
			fv.SetRows(fv.AllRows())
		}
		return m, nil
	case "enter":
		m.filtering = false
		return m, nil
	default:
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		m.filterQuery = m.filterInput.Value()
		if fv, ok := m.currentFilterable(); ok {
			m.applyFilter(fv)
		}
		return m, cmd
	}
}

func (m Model) updateNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		if len(m.stack) > 1 {
			m.stack = m.stack[:len(m.stack)-1]
			return m, nil
		}
		return m, tea.Quit
	case "backspace":
		if len(m.stack) > 1 {
			m.stack = m.stack[:len(m.stack)-1]
			return m, nil
		}
	case "/":
		if _, ok := m.currentFilterable(); ok {
			m.filtering = true
			m.filterInput.SetValue("")
			m.filterInput.Focus()
			return m, textinput.Blink
		}
	case "?":
		m.showHelp = true
		return m, nil
	case "c":
		if cv, ok := m.currentCopyable(); ok {
			id := cv.CopyID()
			if id != "" {
				clipboard.WriteAll(id)
				m.copiedText = id
				return m, m.clearCopiedAfter()
			}
		}
	case "C":
		if cv, ok := m.currentCopyable(); ok {
			arn := cv.CopyARN()
			if arn != "" {
				clipboard.WriteAll(arn)
				m.copiedText = arn
				return m, m.clearCopiedAfter()
			}
		}
	}

	// Delegate to current view for unhandled keys
	if len(m.stack) > 0 {
		current := m.stack[len(m.stack)-1]
		updated, cmd := current.Update(msg)
		m.stack[len(m.stack)-1] = updated
		return m, cmd
	}
	return m, nil
}

func (m Model) currentFilterable() (FilterableView, bool) {
	if len(m.stack) == 0 {
		return nil, false
	}
	fv, ok := m.stack[len(m.stack)-1].(FilterableView)
	return fv, ok
}

func (m Model) currentCopyable() (CopyableView, bool) {
	if len(m.stack) == 0 {
		return nil, false
	}
	cv, ok := m.stack[len(m.stack)-1].(CopyableView)
	return cv, ok
}

func (m Model) applyFilter(fv FilterableView) {
	if m.filterQuery == "" {
		fv.SetRows(fv.AllRows())
		return
	}
	query := strings.ToLower(m.filterQuery)
	var filtered []table.Row
	for _, row := range fv.AllRows() {
		for _, cell := range row {
			if strings.Contains(strings.ToLower(cell), query) {
				filtered = append(filtered, row)
				break
			}
		}
	}
	fv.SetRows(filtered)
}

func (m Model) clearCopiedAfter() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return clearCopiedMsg{}
	})
}
