package services

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awsvpc "tasnim.dev/aws-tui/internal/aws/vpc"
	"tasnim.dev/aws-tui/internal/tui/theme"
)

// naclEntriesMsg carries fetched NACL entries.
type naclEntriesMsg struct {
	entries []awsvpc.NetworkACLEntry
}

// naclEntriesErrMsg signals an error fetching NACL entries.
type naclEntriesErrMsg struct {
	err error
}

// NACLEntriesView shows inbound and outbound entries for a Network ACL.
type NACLEntriesView struct {
	client        *awsclient.ServiceClient
	naclID        string
	title         string
	entries       []awsvpc.NetworkACLEntry
	loaded        bool
	err           error
	viewport      viewport.Model
	vpReady       bool
	spinner       spinner.Model
	width, height int
}

func NewNACLEntriesView(client *awsclient.ServiceClient, naclID, title string) *NACLEntriesView {
	return &NACLEntriesView{
		client:  client,
		naclID:  naclID,
		title:   title,
		spinner: theme.NewSpinner(),
	}
}

func (v *NACLEntriesView) Title() string { return "NACL: " + v.title }

func (v *NACLEntriesView) Init() tea.Cmd {
	return tea.Batch(v.spinner.Tick, v.fetchEntries())
}

func (v *NACLEntriesView) fetchEntries() tea.Cmd {
	client := v.client
	naclID := v.naclID
	return func() tea.Msg {
		entries, err := client.VPC.ListNetworkACLEntries(context.Background(), naclID)
		if err != nil {
			return naclEntriesErrMsg{err: err}
		}
		return naclEntriesMsg{entries: entries}
	}
}

func (v *NACLEntriesView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case naclEntriesMsg:
		v.entries = msg.entries
		v.loaded = true
		v.initViewport()
		return v, nil

	case naclEntriesErrMsg:
		v.err = msg.err
		v.loaded = true
		v.initViewport()
		return v, nil

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		if v.vpReady {
			v.viewport.SetWidth(v.width)
			h := v.height - 2
			if h < 1 {
				h = 1
			}
			v.viewport.SetHeight(h)
		}
		return v, nil

	case spinner.TickMsg:
		if !v.loaded {
			var cmd tea.Cmd
			v.spinner, cmd = v.spinner.Update(msg)
			return v, cmd
		}
	}

	if v.vpReady {
		var cmd tea.Cmd
		v.viewport, cmd = v.viewport.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v *NACLEntriesView) initViewport() {
	h := v.height - 2
	if h < 1 {
		h = 1
	}
	v.viewport = NewStyledViewport(v.width, h)
	v.viewport.SetContent(v.renderContent())
	v.vpReady = true
}

func (v *NACLEntriesView) renderContent() string {
	var b strings.Builder
	bold := lipgloss.NewStyle().Bold(true)

	b.WriteString(bold.Render("Network ACL: " + v.title))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 80))
	b.WriteString("\n")

	if v.err != nil {
		fmt.Fprintf(&b, "\nError loading entries: %s\n", v.err.Error())
		return b.String()
	}

	// Separate inbound and outbound
	var inbound, outbound []awsvpc.NetworkACLEntry
	for _, e := range v.entries {
		if e.Direction == "inbound" {
			inbound = append(inbound, e)
		} else {
			outbound = append(outbound, e)
		}
	}

	b.WriteString("\n")
	b.WriteString(bold.Render("Inbound Rules:"))
	b.WriteString("\n")
	if len(inbound) == 0 {
		b.WriteString("  (none)\n")
	} else {
		fmt.Fprintf(&b, "  %-8s %-10s %-12s %-20s %s\n", "RULE#", "PROTOCOL", "PORT", "CIDR", "ACTION")
		for _, e := range inbound {
			fmt.Fprintf(&b, "  %-8d %-10s %-12s %-20s %s\n", e.RuleNumber, e.Protocol, e.PortRange, e.CIDRBlock, e.Action)
		}
	}

	b.WriteString("\n")
	b.WriteString(bold.Render("Outbound Rules:"))
	b.WriteString("\n")
	if len(outbound) == 0 {
		b.WriteString("  (none)\n")
	} else {
		fmt.Fprintf(&b, "  %-8s %-10s %-12s %-20s %s\n", "RULE#", "PROTOCOL", "PORT", "CIDR", "ACTION")
		for _, e := range outbound {
			fmt.Fprintf(&b, "  %-8d %-10s %-12s %-20s %s\n", e.RuleNumber, e.Protocol, e.PortRange, e.CIDRBlock, e.Action)
		}
	}

	return b.String()
}

func (v *NACLEntriesView) View() string {
	if !v.loaded {
		return v.spinner.View() + " Loading NACL entries..."
	}
	if !v.vpReady {
		return ""
	}
	status := theme.MutedStyle.Render("  Esc back  ↑/↓ scroll")
	return v.viewport.View() + "\n" + status
}

func (v *NACLEntriesView) SetSize(width, height int) {
	v.width = width
	v.height = height
	if v.vpReady {
		v.viewport.SetWidth(width)
		h := height - 2
		if h < 1 {
			h = 1
		}
		v.viewport.SetHeight(h)
	}
}
