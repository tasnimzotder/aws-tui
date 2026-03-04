package app

import (
	"fmt"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"tasnim.dev/aws-tui/internal/ui"
)

var (
	paletteInputStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)

	paletteCursorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205"))

	paletteItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	paletteHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))
)

// PaletteEntry represents a single command in the palette.
type PaletteEntry struct {
	Title    string
	Keywords []string
	Action   func() tea.Cmd
}

// PaletteSelectMsg is sent when the user selects an entry from the palette.
type PaletteSelectMsg struct {
	Entry PaletteEntry
}

// CommandPalette is a fuzzy-searchable command overlay.
type CommandPalette struct {
	entries  []PaletteEntry
	filtered []PaletteEntry
	// searchStrings maps 1:1 with entries — combined Title + Keywords for filtering.
	searchStrings []string
	query         string
	cursor        int
	active        bool
}

// NewCommandPalette creates a new command palette with the given entries.
func NewCommandPalette(entries []PaletteEntry) CommandPalette {
	cp := CommandPalette{}
	cp.setEntriesInternal(entries)
	return cp
}

// SetEntries replaces the palette entries and reapplies the current filter.
func (p *CommandPalette) SetEntries(entries []PaletteEntry) {
	p.setEntriesInternal(entries)
	p.applyFilter()
}

func (p *CommandPalette) setEntriesInternal(entries []PaletteEntry) {
	p.entries = make([]PaletteEntry, len(entries))
	copy(p.entries, entries)

	p.searchStrings = make([]string, len(entries))
	for i, e := range entries {
		p.searchStrings[i] = strings.ToLower(e.Title + " " + strings.Join(e.Keywords, " "))
	}

	p.filtered = p.entries
}

// Active returns whether the palette is currently open.
func (p CommandPalette) Active() bool {
	return p.active
}

// ResultCount returns the number of entries matching the current query.
func (p CommandPalette) ResultCount() int {
	return len(p.filtered)
}

// Open activates the palette and resets query and cursor.
func (p *CommandPalette) Open() {
	p.active = true
	p.query = ""
	p.cursor = 0
	p.filtered = p.entries
}

// Close deactivates the palette.
func (p *CommandPalette) Close() {
	p.active = false
}

// Update handles key events for the command palette.
func (p CommandPalette) Update(msg tea.Msg) (CommandPalette, tea.Cmd) {
	if !p.active {
		return p, nil
	}

	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return p, nil
	}

	switch km.String() {
	case "esc":
		p.Close()
		return p, nil

	case "enter":
		if len(p.filtered) == 0 {
			return p, nil
		}
		selected := p.filtered[p.cursor]
		p.Close()
		return p, func() tea.Msg {
			return PaletteSelectMsg{Entry: selected}
		}

	case "backspace":
		if len(p.query) > 0 {
			p.query = p.query[:len(p.query)-1]
			p.applyFilter()
		}
		return p, nil

	case "up":
		if p.cursor > 0 {
			p.cursor--
		}
		return p, nil

	case "down":
		if len(p.filtered) > 0 && p.cursor < len(p.filtered)-1 {
			p.cursor++
		}
		return p, nil

	default:
		if len(km.Text) > 0 {
			for _, r := range km.Text {
				if unicode.IsPrint(r) {
					p.query += km.Text
					p.applyFilter()
					return p, nil
				}
			}
		}
		return p, nil
	}
}

func (p *CommandPalette) applyFilter() {
	if p.query == "" {
		p.filtered = p.entries
		p.cursor = 0
		return
	}

	matched := ui.FuzzyFilter(p.searchStrings, p.query)
	if matched == nil {
		p.filtered = nil
		p.cursor = 0
		return
	}

	// Build a set of matched search strings for lookup.
	// FuzzyFilter returns the matched strings in ranked order.
	// Map them back to entries.
	matchedSet := make(map[string]int, len(matched))
	for i, m := range matched {
		matchedSet[m] = i
	}

	result := make([]PaletteEntry, len(matched))
	for i, ss := range p.searchStrings {
		if rank, ok := matchedSet[ss]; ok {
			result[rank] = p.entries[i]
		}
	}

	p.filtered = result
	p.cursor = 0
}

// View renders the command palette.
func (p CommandPalette) View() string {
	if !p.active {
		return ""
	}

	var b strings.Builder

	b.WriteString(paletteInputStyle.Render("Command Palette"))
	b.WriteByte('\n')

	if p.query != "" {
		b.WriteString(paletteHintStyle.Render(fmt.Sprintf("> %s", p.query)))
	} else {
		b.WriteString(paletteHintStyle.Render("> (type to search)"))
	}
	b.WriteByte('\n')
	b.WriteByte('\n')

	for i, entry := range p.filtered {
		if i == p.cursor {
			b.WriteString(paletteCursorStyle.Render(fmt.Sprintf("  > %s", entry.Title)))
		} else {
			b.WriteString(paletteItemStyle.Render(fmt.Sprintf("    %s", entry.Title)))
		}
		b.WriteByte('\n')
	}

	return b.String()
}
