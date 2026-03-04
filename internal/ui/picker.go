package ui

import (
	"fmt"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	pickerTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				MarginBottom(1)

	pickerCursorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205"))

	pickerItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	pickerFilterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))
)

// PickerResult is returned as a tea.Msg when the user selects an item or cancels.
type PickerResult struct {
	Selected string
	Canceled bool
}

// Picker is a filterable overlay for selecting from a list of items.
type Picker struct {
	title    string
	items    []string
	filtered []string
	cursor   int
	filter   string
}

// NewPicker creates a new Picker with the given title and items.
func NewPicker(title string, items []string) Picker {
	cp := make([]string, len(items))
	copy(cp, items)
	return Picker{
		title:    title,
		items:    cp,
		filtered: cp,
		cursor:   0,
	}
}

// Cursor returns the current cursor position.
func (p Picker) Cursor() int {
	return p.cursor
}

// FilteredCount returns the number of items matching the current filter.
func (p Picker) FilteredCount() int {
	return len(p.filtered)
}

// Init satisfies the Bubble Tea model interface. Returns nil.
func (p Picker) Init() tea.Cmd {
	return nil
}

// Update handles key events for the picker.
func (p Picker) Update(msg tea.Msg) (Picker, tea.Cmd) {
	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return p, nil
	}

	switch km.String() {
	case "j", "down":
		if len(p.filtered) > 0 && p.cursor < len(p.filtered)-1 {
			p.cursor++
		}
		return p, nil

	case "k", "up":
		if p.cursor > 0 {
			p.cursor--
		}
		return p, nil

	case "enter":
		if len(p.filtered) == 0 {
			return p, nil
		}
		selected := p.filtered[p.cursor]
		return p, func() tea.Msg {
			return PickerResult{Selected: selected}
		}

	case "esc":
		return p, func() tea.Msg {
			return PickerResult{Canceled: true}
		}

	case "backspace":
		if len(p.filter) > 0 {
			p.filter = p.filter[:len(p.filter)-1]
			p.applyFilter()
		}
		return p, nil

	default:
		// Only add printable characters to the filter
		if len(km.Text) > 0 {
			for _, r := range km.Text {
				if unicode.IsPrint(r) {
					p.filter += km.Text
					p.applyFilter()
					return p, nil
				}
			}
		}
		return p, nil
	}
}

func (p *Picker) applyFilter() {
	if p.filter == "" {
		p.filtered = p.items
	} else {
		lower := strings.ToLower(p.filter)
		var result []string
		for _, item := range p.items {
			if strings.Contains(strings.ToLower(item), lower) {
				result = append(result, item)
			}
		}
		p.filtered = result
	}
	p.cursor = 0
}

// View renders the picker overlay.
func (p Picker) View() string {
	var b strings.Builder

	b.WriteString(pickerTitleStyle.Render(p.title))
	b.WriteByte('\n')

	if p.filter != "" {
		b.WriteString(pickerFilterStyle.Render(fmt.Sprintf("Filter: %s", p.filter)))
	} else {
		b.WriteString(pickerFilterStyle.Render("Filter: (type to filter)"))
	}
	b.WriteByte('\n')
	b.WriteByte('\n')

	for i, item := range p.filtered {
		if i == p.cursor {
			b.WriteString(pickerCursorStyle.Render(fmt.Sprintf("> %s", item)))
		} else {
			b.WriteString(pickerItemStyle.Render(fmt.Sprintf("  %s", item)))
		}
		b.WriteByte('\n')
	}

	return b.String()
}
