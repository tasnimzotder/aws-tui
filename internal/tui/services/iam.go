package services

import (
	"context"
	"encoding/json"
	"fmt"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awsiam "tasnim.dev/aws-tui/internal/aws/iam"
	"tasnim.dev/aws-tui/internal/utils"
)

// --- IAM Top-level Sub-menu ---

type iamMenuItem struct {
	name string
	desc string
}

func (i iamMenuItem) Title() string       { return i.name }
func (i iamMenuItem) Description() string { return i.desc }
func (i iamMenuItem) FilterValue() string { return i.name }

type IAMSubMenuView struct {
	client *awsclient.ServiceClient
	list   list.Model
}

func NewIAMSubMenuView(client *awsclient.ServiceClient) *IAMSubMenuView {
	items := []list.Item{
		iamMenuItem{name: "Users", desc: "IAM users and their policies/groups"},
		iamMenuItem{name: "Roles", desc: "IAM roles and trust policies"},
		iamMenuItem{name: "Policies", desc: "Customer-managed IAM policies"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 60, 10)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	return &IAMSubMenuView{client: client, list: l}
}

func (v *IAMSubMenuView) Title() string { return "IAM" }

func (v *IAMSubMenuView) HelpContext() *HelpContext {
	ctx := HelpContextRoot
	return &ctx
}

func (v *IAMSubMenuView) Init() tea.Cmd { return nil }
func (v *IAMSubMenuView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "enter" {
			selected, ok := v.list.SelectedItem().(iamMenuItem)
			if !ok {
				return v, nil
			}
			switch selected.name {
			case "Users":
				return v, pushView(NewIAMUsersView(v.client))
			case "Roles":
				return v, pushView(NewIAMRolesView(v.client))
			case "Policies":
				return v, pushView(NewIAMPoliciesView(v.client))
			}
		}
	}
	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}
func (v *IAMSubMenuView) View() string { return v.list.View() }
func (v *IAMSubMenuView) SetSize(width, height int) {
	v.list.SetSize(width, height)
}

// --- Users List ---

func NewIAMUsersView(client *awsclient.ServiceClient) *TableView[awsiam.IAMUser] {
	var nextMarker *string
	return NewTableView(TableViewConfig[awsiam.IAMUser]{
		Title:       "Users",
		LoadingText: "Loading users...",
		Columns: []table.Column{
			{Title: "Name", Width: 25},
			{Title: "User ID", Width: 22},
			{Title: "Created", Width: 20},
			{Title: "Path", Width: 20},
		},
		FetchFuncPaged: func(ctx context.Context) ([]awsiam.IAMUser, bool, error) {
			nextMarker = nil
			users, nm, err := client.IAM.ListUsersPage(ctx, nil)
			if err != nil {
				return nil, false, err
			}
			nextMarker = nm
			return users, nm != nil, nil
		},
		LoadMoreFunc: func(ctx context.Context) ([]awsiam.IAMUser, bool, error) {
			users, nm, err := client.IAM.ListUsersPage(ctx, nextMarker)
			if err != nil {
				return nil, false, err
			}
			nextMarker = nm
			return users, nm != nil, nil
		},
		RowMapper: func(u awsiam.IAMUser) table.Row {
			return table.Row{u.Name, u.UserID, utils.TimeOrDash(u.CreatedAt, utils.DateOnly), u.Path}
		},
		CopyIDFunc:  func(u awsiam.IAMUser) string { return u.Name },
		CopyARNFunc: func(u awsiam.IAMUser) string { return u.ARN },
		OnEnter: func(u awsiam.IAMUser) tea.Cmd {
			return pushView(NewIAMUserSubMenuView(client, u.Name))
		},
	})
}

// --- Roles List ---

func NewIAMRolesView(client *awsclient.ServiceClient) *TableView[awsiam.IAMRole] {
	var nextMarker *string
	return NewTableView(TableViewConfig[awsiam.IAMRole]{
		Title:       "Roles",
		LoadingText: "Loading roles...",
		Columns: []table.Column{
			{Title: "Name", Width: 30},
			{Title: "Description", Width: 30},
			{Title: "Created", Width: 20},
			{Title: "Path", Width: 20},
		},
		FetchFuncPaged: func(ctx context.Context) ([]awsiam.IAMRole, bool, error) {
			nextMarker = nil
			roles, nm, err := client.IAM.ListRolesPage(ctx, nil)
			if err != nil {
				return nil, false, err
			}
			nextMarker = nm
			return roles, nm != nil, nil
		},
		LoadMoreFunc: func(ctx context.Context) ([]awsiam.IAMRole, bool, error) {
			roles, nm, err := client.IAM.ListRolesPage(ctx, nextMarker)
			if err != nil {
				return nil, false, err
			}
			nextMarker = nm
			return roles, nm != nil, nil
		},
		RowMapper: func(r awsiam.IAMRole) table.Row {
			desc := r.Description
			if desc == "" {
				desc = "—"
			}
			return table.Row{r.Name, desc, utils.TimeOrDash(r.CreatedAt, utils.DateOnly), r.Path}
		},
		CopyIDFunc:  func(r awsiam.IAMRole) string { return r.Name },
		CopyARNFunc: func(r awsiam.IAMRole) string { return r.ARN },
		OnEnter: func(r awsiam.IAMRole) tea.Cmd {
			return pushView(NewIAMRoleSubMenuView(client, r.Name, r.AssumeRolePolicyDocument))
		},
	})
}

// --- Policies List ---

func NewIAMPoliciesView(client *awsclient.ServiceClient) *TableView[awsiam.IAMPolicy] {
	var nextMarker *string
	return NewTableView(TableViewConfig[awsiam.IAMPolicy]{
		Title:       "Policies",
		LoadingText: "Loading policies...",
		Columns: []table.Column{
			{Title: "Name", Width: 30},
			{Title: "Attachments", Width: 12},
			{Title: "Created", Width: 20},
			{Title: "Updated", Width: 20},
		},
		FetchFuncPaged: func(ctx context.Context) ([]awsiam.IAMPolicy, bool, error) {
			nextMarker = nil
			policies, nm, err := client.IAM.ListPoliciesPage(ctx, nil)
			if err != nil {
				return nil, false, err
			}
			nextMarker = nm
			return policies, nm != nil, nil
		},
		LoadMoreFunc: func(ctx context.Context) ([]awsiam.IAMPolicy, bool, error) {
			policies, nm, err := client.IAM.ListPoliciesPage(ctx, nextMarker)
			if err != nil {
				return nil, false, err
			}
			nextMarker = nm
			return policies, nm != nil, nil
		},
		RowMapper: func(p awsiam.IAMPolicy) table.Row {
			return table.Row{p.Name, fmt.Sprintf("%d", p.AttachmentCount), utils.TimeOrDash(p.CreatedAt, utils.DateOnly), utils.TimeOrDash(p.UpdatedAt, utils.DateOnly)}
		},
		CopyIDFunc:  func(p awsiam.IAMPolicy) string { return p.Name },
		CopyARNFunc: func(p awsiam.IAMPolicy) string { return p.ARN },
		OnEnter: func(p awsiam.IAMPolicy) tea.Cmd {
			return pushView(NewIAMPolicyEntitiesView(client, p.ARN, p.Name))
		},
	})
}

// --- User Sub-menu ---

type iamUserSubMenuItem struct {
	name string
	desc string
}

func (i iamUserSubMenuItem) Title() string       { return i.name }
func (i iamUserSubMenuItem) Description() string { return i.desc }
func (i iamUserSubMenuItem) FilterValue() string { return i.name }

type IAMUserSubMenuView struct {
	client   *awsclient.ServiceClient
	userName string
	list     list.Model
}

func NewIAMUserSubMenuView(client *awsclient.ServiceClient, userName string) *IAMUserSubMenuView {
	items := []list.Item{
		iamUserSubMenuItem{name: "Attached Policies", desc: "Managed policies attached to this user"},
		iamUserSubMenuItem{name: "Groups", desc: "Groups this user belongs to"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 60, 8)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	return &IAMUserSubMenuView{client: client, userName: userName, list: l}
}

func (v *IAMUserSubMenuView) Title() string { return v.userName }

func (v *IAMUserSubMenuView) HelpContext() *HelpContext {
	ctx := HelpContextRoot
	return &ctx
}

func (v *IAMUserSubMenuView) Init() tea.Cmd { return nil }
func (v *IAMUserSubMenuView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "enter" {
			selected, ok := v.list.SelectedItem().(iamUserSubMenuItem)
			if !ok {
				return v, nil
			}
			switch selected.name {
			case "Attached Policies":
				return v, pushView(NewIAMUserPoliciesView(v.client, v.userName))
			case "Groups":
				return v, pushView(NewIAMUserGroupsView(v.client, v.userName))
			}
		}
	}
	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}
func (v *IAMUserSubMenuView) View() string { return v.list.View() }
func (v *IAMUserSubMenuView) SetSize(width, height int) {
	v.list.SetSize(width, height)
}

// --- Role Sub-menu ---

type iamRoleSubMenuItem struct {
	name string
	desc string
}

func (i iamRoleSubMenuItem) Title() string       { return i.name }
func (i iamRoleSubMenuItem) Description() string { return i.desc }
func (i iamRoleSubMenuItem) FilterValue() string { return i.name }

type IAMRoleSubMenuView struct {
	client   *awsclient.ServiceClient
	roleName string
	trustDoc string
	list     list.Model
}

func NewIAMRoleSubMenuView(client *awsclient.ServiceClient, roleName, trustDoc string) *IAMRoleSubMenuView {
	items := []list.Item{
		iamRoleSubMenuItem{name: "Attached Policies", desc: "Managed policies attached to this role"},
		iamRoleSubMenuItem{name: "Trust Policy", desc: "Who can assume this role"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 60, 8)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	return &IAMRoleSubMenuView{client: client, roleName: roleName, trustDoc: trustDoc, list: l}
}

func (v *IAMRoleSubMenuView) Title() string { return v.roleName }

func (v *IAMRoleSubMenuView) HelpContext() *HelpContext {
	ctx := HelpContextRoot
	return &ctx
}

func (v *IAMRoleSubMenuView) Init() tea.Cmd { return nil }
func (v *IAMRoleSubMenuView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "enter" {
			selected, ok := v.list.SelectedItem().(iamRoleSubMenuItem)
			if !ok {
				return v, nil
			}
			switch selected.name {
			case "Attached Policies":
				return v, pushView(NewIAMRolePoliciesView(v.client, v.roleName))
			case "Trust Policy":
				return v, pushView(NewIAMTrustPolicyView(v.roleName, v.trustDoc))
			}
		}
	}
	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}
func (v *IAMRoleSubMenuView) View() string { return v.list.View() }
func (v *IAMRoleSubMenuView) SetSize(width, height int) {
	v.list.SetSize(width, height)
}

// --- User Attached Policies ---

func NewIAMUserPoliciesView(client *awsclient.ServiceClient, userName string) *TableView[awsiam.IAMAttachedPolicy] {
	return NewTableView(TableViewConfig[awsiam.IAMAttachedPolicy]{
		Title:       "Policies",
		LoadingText: "Loading attached policies...",
		Columns: []table.Column{
			{Title: "Policy Name", Width: 35},
			{Title: "ARN", Width: 55},
		},
		FetchFunc: func(ctx context.Context) ([]awsiam.IAMAttachedPolicy, error) {
			return client.IAM.ListAttachedUserPolicies(ctx, userName)
		},
		RowMapper: func(p awsiam.IAMAttachedPolicy) table.Row {
			return table.Row{p.Name, p.ARN}
		},
		CopyIDFunc:  func(p awsiam.IAMAttachedPolicy) string { return p.Name },
		CopyARNFunc: func(p awsiam.IAMAttachedPolicy) string { return p.ARN },
	})
}

// --- User Groups ---

func NewIAMUserGroupsView(client *awsclient.ServiceClient, userName string) *TableView[awsiam.IAMGroup] {
	return NewTableView(TableViewConfig[awsiam.IAMGroup]{
		Title:       "Groups",
		LoadingText: "Loading groups...",
		Columns: []table.Column{
			{Title: "Group Name", Width: 30},
			{Title: "ARN", Width: 55},
		},
		FetchFunc: func(ctx context.Context) ([]awsiam.IAMGroup, error) {
			return client.IAM.ListGroupsForUser(ctx, userName)
		},
		RowMapper: func(g awsiam.IAMGroup) table.Row {
			return table.Row{g.Name, g.ARN}
		},
		CopyIDFunc:  func(g awsiam.IAMGroup) string { return g.Name },
		CopyARNFunc: func(g awsiam.IAMGroup) string { return g.ARN },
	})
}

// --- Role Attached Policies ---

func NewIAMRolePoliciesView(client *awsclient.ServiceClient, roleName string) *TableView[awsiam.IAMAttachedPolicy] {
	return NewTableView(TableViewConfig[awsiam.IAMAttachedPolicy]{
		Title:       "Policies",
		LoadingText: "Loading attached policies...",
		Columns: []table.Column{
			{Title: "Policy Name", Width: 35},
			{Title: "ARN", Width: 55},
		},
		FetchFunc: func(ctx context.Context) ([]awsiam.IAMAttachedPolicy, error) {
			return client.IAM.ListAttachedRolePolicies(ctx, roleName)
		},
		RowMapper: func(p awsiam.IAMAttachedPolicy) table.Row {
			return table.Row{p.Name, p.ARN}
		},
		CopyIDFunc:  func(p awsiam.IAMAttachedPolicy) string { return p.Name },
		CopyARNFunc: func(p awsiam.IAMAttachedPolicy) string { return p.ARN },
	})
}

// --- Policy Attached Entities ---

func NewIAMPolicyEntitiesView(client *awsclient.ServiceClient, policyARN, policyName string) *TableView[awsiam.IAMPolicyEntity] {
	return NewTableView(TableViewConfig[awsiam.IAMPolicyEntity]{
		Title:       policyName,
		LoadingText: "Loading attached entities...",
		Columns: []table.Column{
			{Title: "Name", Width: 30},
			{Title: "Type", Width: 12},
		},
		FetchFunc: func(ctx context.Context) ([]awsiam.IAMPolicyEntity, error) {
			return client.IAM.ListEntitiesForPolicy(ctx, policyARN)
		},
		RowMapper: func(e awsiam.IAMPolicyEntity) table.Row {
			return table.Row{e.Name, e.Type}
		},
		CopyIDFunc: func(e awsiam.IAMPolicyEntity) string { return e.Name },
	})
}

// --- Trust Policy Text View (viewport-based) ---

type IAMTrustPolicyView struct {
	roleName string
	viewport viewport.Model
	content  string
	ready    bool
	width    int
	height   int
}

func NewIAMTrustPolicyView(roleName, trustDoc string) *IAMTrustPolicyView {
	content := trustDoc
	var parsed any
	if err := json.Unmarshal([]byte(trustDoc), &parsed); err == nil {
		if pretty, err := json.MarshalIndent(parsed, "", "  "); err == nil {
			content = string(pretty)
		}
	}
	return &IAMTrustPolicyView{
		roleName: roleName,
		content:  content,
		width:    80,
		height:   20,
	}
}

func (v *IAMTrustPolicyView) Title() string { return "Trust Policy" }
func (v *IAMTrustPolicyView) Init() tea.Cmd {
	v.viewport = viewport.New(viewport.WithWidth(v.width), viewport.WithHeight(v.height))
	v.viewport.SetContent(v.content)
	v.ready = true
	return nil
}
func (v *IAMTrustPolicyView) Update(msg tea.Msg) (View, tea.Cmd) {
	if v.ready {
		var cmd tea.Cmd
		v.viewport, cmd = v.viewport.Update(msg)
		return v, cmd
	}
	return v, nil
}
func (v *IAMTrustPolicyView) View() string {
	if v.ready {
		return v.viewport.View()
	}
	return ""
}
func (v *IAMTrustPolicyView) SetSize(width, height int) {
	v.width = width
	v.height = height
	if v.ready {
		v.viewport.SetWidth(width)
		v.viewport.SetHeight(height)
	}
}
