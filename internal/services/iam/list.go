package iam

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	awsiam "tasnim.dev/aws-tui/internal/aws/iam"
	"tasnim.dev/aws-tui/internal/plugin"
	"tasnim.dev/aws-tui/internal/ui"
)

// Fetch result messages.
type usersMsg struct {
	users []awsiam.IAMUser
	err   error
}

type rolesMsg struct {
	roles []awsiam.IAMRole
	err   error
}

type policiesMsg struct {
	policies []awsiam.IAMPolicy
	err      error
}

// ListView displays IAM resources in a tabbed table view.
type ListView struct {
	client IAMClient
	router plugin.Router

	tabs     ui.TabController
	users    ui.TableView[awsiam.IAMUser]
	roles    ui.TableView[awsiam.IAMRole]
	policies ui.TableView[awsiam.IAMPolicy]

	loading bool
	err     error
}

// NewListView creates a new IAM ListView with tabs for Users, Roles, and Policies.
func NewListView(client IAMClient, router plugin.Router) *ListView {
	userCols := []ui.Column[awsiam.IAMUser]{
		{Title: "Name", Width: 24, Field: func(u awsiam.IAMUser) string { return u.Name }},
		{Title: "Created", Width: 12, Field: func(u awsiam.IAMUser) string {
			if u.CreatedAt.IsZero() {
				return "-"
			}
			return u.CreatedAt.Format("2006-01-02")
		}},
		{Title: "ARN", Width: 48, Field: func(u awsiam.IAMUser) string { return u.ARN }},
		{Title: "Path", Width: 16, Field: func(u awsiam.IAMUser) string { return u.Path }},
	}

	roleCols := []ui.Column[awsiam.IAMRole]{
		{Title: "Name", Width: 30, Field: func(r awsiam.IAMRole) string { return r.Name }},
		{Title: "Created", Width: 12, Field: func(r awsiam.IAMRole) string {
			if r.CreatedAt.IsZero() {
				return "-"
			}
			return r.CreatedAt.Format("2006-01-02")
		}},
		{Title: "Description", Width: 40, Field: func(r awsiam.IAMRole) string { return r.Description }},
	}

	policyCols := []ui.Column[awsiam.IAMPolicy]{
		{Title: "Name", Width: 30, Field: func(p awsiam.IAMPolicy) string { return p.Name }},
		{Title: "ARN", Width: 48, Field: func(p awsiam.IAMPolicy) string { return p.ARN }},
		{Title: "Attached", Width: 10, Field: func(p awsiam.IAMPolicy) string {
			return fmt.Sprintf("%d", p.AttachmentCount)
		}},
	}

	return &ListView{
		client:   client,
		router:   router,
		tabs:     ui.NewTabController([]string{"Users", "Roles", "Policies"}),
		users:    ui.NewTableView(userCols, nil, func(u awsiam.IAMUser) string { return "user:" + u.Name }),
		roles:    ui.NewTableView(roleCols, nil, func(r awsiam.IAMRole) string { return "role:" + r.Name }),
		policies: ui.NewTableView(policyCols, nil, func(p awsiam.IAMPolicy) string { return p.ARN }),
		loading:  true,
	}
}

func (lv *ListView) fetchAll() tea.Cmd {
	client := lv.client
	return tea.Batch(
		func() tea.Msg {
			users, err := client.ListUsers(context.TODO())
			return usersMsg{users: users, err: err}
		},
		func() tea.Msg {
			roles, err := client.ListRoles(context.TODO())
			return rolesMsg{roles: roles, err: err}
		},
		func() tea.Msg {
			policies, err := client.ListPolicies(context.TODO())
			return policiesMsg{policies: policies, err: err}
		},
	)
}

func (lv *ListView) Init() tea.Cmd {
	return lv.fetchAll()
}

func (lv *ListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case usersMsg:
		lv.loading = false
		if msg.err != nil {
			lv.err = msg.err
			return lv, nil
		}
		lv.users.SetItems(msg.users)
		return lv, nil

	case rolesMsg:
		lv.loading = false
		if msg.err != nil {
			lv.err = msg.err
			return lv, nil
		}
		lv.roles.SetItems(msg.roles)
		return lv, nil

	case policiesMsg:
		lv.loading = false
		if msg.err != nil {
			lv.err = msg.err
			return lv, nil
		}
		lv.policies.SetItems(msg.policies)
		return lv, nil

	case tea.KeyPressMsg:
		if lv.loading {
			return lv, nil
		}

		switch msg.String() {
		case "enter":
			id := lv.selectedID()
			if id != "" {
				view := NewDetailView(lv.client, lv.router, id)
				lv.router.Push(view)
				return lv, view.Init()
			}
			return lv, nil
		case "esc", "backspace":
			lv.router.Pop()
			return lv, nil
		case "r":
			lv.loading = true
			return lv, lv.fetchAll()
		}
	}

	// Forward to tab controller first.
	var cmd tea.Cmd
	lv.tabs, cmd = lv.tabs.Update(msg)

	// Forward to the active table.
	var tableCmd tea.Cmd
	switch lv.tabs.Active() {
	case 0:
		lv.users, tableCmd = lv.users.Update(msg)
	case 1:
		lv.roles, tableCmd = lv.roles.Update(msg)
	case 2:
		lv.policies, tableCmd = lv.policies.Update(msg)
	}

	return lv, tea.Batch(cmd, tableCmd)
}

func (lv *ListView) selectedID() string {
	switch lv.tabs.Active() {
	case 0:
		return lv.users.SelectedID()
	case 1:
		return lv.roles.SelectedID()
	case 2:
		return lv.policies.SelectedID()
	}
	return ""
}

func (lv *ListView) View() tea.View {
	if lv.loading {
		skel := ui.NewSkeleton(80, 6)
		return tea.NewView(skel.View())
	}
	if lv.err != nil {
		return tea.NewView("Error: " + lv.err.Error())
	}

	var b strings.Builder
	b.WriteString(lv.tabs.View())
	b.WriteString("\n\n")

	switch lv.tabs.Active() {
	case 0:
		b.WriteString(lv.users.View())
	case 1:
		b.WriteString(lv.roles.View())
	case 2:
		b.WriteString(lv.policies.View())
	}

	return tea.NewView(b.String())
}

func (lv *ListView) Title() string { return "IAM" }

func (lv *ListView) KeyHints() []plugin.KeyHint {
	return []plugin.KeyHint{
		{Key: "enter", Desc: "view details"},
		{Key: "r", Desc: "refresh"},
		{Key: "/", Desc: "filter"},
		{Key: "s", Desc: "sort"},
		{Key: "[/]", Desc: "switch tab"},
	}
}
