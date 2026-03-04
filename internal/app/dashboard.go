package app

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	internalaws "tasnim.dev/aws-tui/internal/aws"
	"tasnim.dev/aws-tui/internal/cache"
	"tasnim.dev/aws-tui/internal/plugin"
)

var (
	dashHeaderLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))

	dashHeaderValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Bold(true)

	dashSepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))

	dashSectionTitle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)

	dashAccentBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))

	dashServiceName = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)

	dashServiceNameActive = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)

	dashServiceDesc = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)

// serviceDescriptions maps plugin IDs to human-readable subtitles.
var serviceDescriptions = map[string]string{
	"ec2":  "Elastic Compute Cloud — Instances",
	"ecs":  "Elastic Container Service — Clusters, Services, Tasks",
	"eks":  "Elastic Kubernetes Service — Clusters, Pods, Services",
	"vpc":  "Virtual Private Cloud — VPCs, Subnets, Security Groups",
	"s3":   "Simple Storage Service — Buckets, Objects",
	"iam":  "Identity & Access Management — Users, Roles, Policies",
	"ecr":  "Elastic Container Registry — Repositories, Images",
	"elb":  "Elastic Load Balancing — Load Balancers, Listeners, Target Groups",
	"cost": "Cost Explorer — Spend Analysis, Forecasts",
}

type identityMsg struct {
	identity internalaws.Identity
	err      error
}

// Dashboard is the home screen showing service cards from registered plugins.
type Dashboard struct {
	registry *plugin.Registry
	router   plugin.Router
	session  *internalaws.Session
	cache    *cache.DB
	region   string
	profile  string
	identity *internalaws.Identity
	cursor   int
	width    int
	height   int
}

// NewDashboard creates a Dashboard wired to the given registry and router.
func NewDashboard(registry *plugin.Registry, router plugin.Router, sess *internalaws.Session, cacheDB *cache.DB, region, profile string) *Dashboard {
	return &Dashboard{
		registry: registry,
		router:   router,
		session:  sess,
		cache:    cacheDB,
		region:   region,
		profile:  profile,
	}
}

// Title returns the view title for the breadcrumb.
func (d *Dashboard) Title() string {
	return "Dashboard"
}

// KeyHints returns navigation hints for the dashboard.
func (d *Dashboard) KeyHints() []plugin.KeyHint {
	return []plugin.KeyHint{
		{Key: "j/k", Desc: "navigate"},
		{Key: "enter", Desc: "open"},
		{Key: "ctrl+k", Desc: "palette"},
		{Key: "q", Desc: "quit"},
	}
}

// Init fetches AWS identity asynchronously.
func (d *Dashboard) Init() tea.Cmd {
	if d.session == nil || d.identity != nil {
		return nil
	}
	sess := d.session
	return func() tea.Msg {
		id, err := sess.CallerIdentity(context.TODO())
		return identityMsg{identity: id, err: err}
	}
}

// Update handles messages for the dashboard.
func (d *Dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case identityMsg:
		if msg.err == nil {
			d.identity = &msg.identity
		}
		return d, nil

	case tea.KeyPressMsg:
		plugins := d.registry.All()
		switch msg.String() {
		case "j", "down":
			if len(plugins) > 0 && d.cursor < len(plugins)-1 {
				d.cursor++
			}
		case "k", "up":
			if d.cursor > 0 {
				d.cursor--
			}
		case "enter":
			if len(plugins) > 0 && d.cursor < len(plugins) {
				p := plugins[d.cursor]
				view := p.ListView(d.router)
				d.router.Push(view)
				return d, view.Init()
			}
		}
		return d, nil

	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height
		return d, nil
	}

	return d, nil
}

// View renders the dashboard as a static list of service names.
func (d *Dashboard) View() tea.View {
	plugins := d.registry.All()

	if len(plugins) == 0 {
		return tea.NewView("  No services registered.\n\n  Press Ctrl+K to open the command palette.")
	}

	var b strings.Builder

	// Header: AWS identity + tool info.
	b.WriteString(d.renderHeader())
	b.WriteByte('\n')

	// Separator line.
	lineWidth := d.width
	if lineWidth <= 0 {
		lineWidth = 80
	}
	b.WriteString(dashSepStyle.Render(strings.Repeat("─", lineWidth)))
	b.WriteString("\n\n")

	// Section title.
	b.WriteString("    ")
	b.WriteString(dashSectionTitle.Render("Services"))
	b.WriteString("\n\n")

	// Service cards with accent bar.
	bar := dashAccentBar.Render("│")
	for i, p := range plugins {
		nameStyle := dashServiceName
		if i == d.cursor {
			nameStyle = dashServiceNameActive
		}

		// Line 1: icon + name
		b.WriteString("    ")
		b.WriteString(bar)
		b.WriteString("  ")
		b.WriteString(p.Icon())
		b.WriteString("  ")
		b.WriteString(nameStyle.Render(p.Name()))
		b.WriteByte('\n')

		// Line 2: description
		desc := serviceDescriptions[p.ID()]
		if desc != "" {
			b.WriteString("    ")
			b.WriteString(bar)
			b.WriteString("  ")
			b.WriteString(dashServiceDesc.Render(desc))
			b.WriteByte('\n')
		}

		// Blank line between services.
		if i < len(plugins)-1 {
			b.WriteString("    ")
			b.WriteString(bar)
			b.WriteByte('\n')
		}
	}

	return tea.NewView(b.String())
}

func (d *Dashboard) renderHeader() string {
	sep := dashHeaderLabelStyle.Render(" | ")

	var parts []string

	// Identity
	if d.identity != nil {
		parts = append(parts,
			dashHeaderLabelStyle.Render("Account: ")+dashHeaderValueStyle.Render(d.identity.Account),
		)
		// Extract user/role from ARN (last segment after / or :)
		arn := d.identity.ARN
		if idx := strings.LastIndex(arn, "/"); idx >= 0 {
			arn = arn[idx+1:]
		}
		parts = append(parts,
			dashHeaderLabelStyle.Render("Identity: ")+dashHeaderValueStyle.Render(arn),
		)
	} else {
		parts = append(parts, dashHeaderLabelStyle.Render("Identity: ")+dashHeaderValueStyle.Render("loading..."))
	}

	// Region + Profile
	parts = append(parts,
		dashHeaderLabelStyle.Render("Region: ")+dashHeaderValueStyle.Render(d.region),
		dashHeaderLabelStyle.Render("Profile: ")+dashHeaderValueStyle.Render(d.profile),
	)

	// Cache status
	cacheStatus := "off"
	if d.cache != nil {
		cacheStatus = "on"
	}
	parts = append(parts,
		dashHeaderLabelStyle.Render("Cache: ")+dashHeaderValueStyle.Render(cacheStatus),
	)

	return "  " + strings.Join(parts, sep)
}
