package iam

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	awsiam "tasnim.dev/aws-tui/internal/aws/iam"
	"tasnim.dev/aws-tui/internal/plugin"
	"tasnim.dev/aws-tui/internal/ui"
)

// Detail load messages.
type userDetailMsg struct {
	user           awsiam.IAMUser
	policies       []awsiam.IAMAttachedPolicy
	inlinePolicies []awsiam.IAMInlinePolicy
	groups         []awsiam.IAMGroup
	err            error
}

type roleDetailMsg struct {
	role           awsiam.IAMRole
	policies       []awsiam.IAMAttachedPolicy
	inlinePolicies []awsiam.IAMInlinePolicy
	err            error
}

type policyDetailMsg struct {
	policy   awsiam.IAMPolicy
	document string
	entities []awsiam.IAMPolicyEntity
	err      error
}

// DetailView shows details for a single IAM user, role, or policy.
type DetailView struct {
	client  IAMClient
	router  plugin.Router
	id      string
	kind    string // "user", "role", or "policy"
	name    string
	tabs    ui.TabController
	loading bool
	err     error

	// User detail
	user               *awsiam.IAMUser
	userPolicies       []awsiam.IAMAttachedPolicy
	userInlinePolicies []awsiam.IAMInlinePolicy
	userGroups         []awsiam.IAMGroup

	// Role detail
	role               *awsiam.IAMRole
	rolePolicies       []awsiam.IAMAttachedPolicy
	roleInlinePolicies []awsiam.IAMInlinePolicy

	// Policy detail
	policy         *awsiam.IAMPolicy
	policyDocument string
	policyEntities []awsiam.IAMPolicyEntity
}

// NewDetailView creates a DetailView. The id format determines the resource kind:
//   - "user:<name>" for users
//   - "role:<name>" for roles
//   - anything else (ARN) for policies
func NewDetailView(client IAMClient, router plugin.Router, id string) *DetailView {
	dv := &DetailView{
		client:  client,
		router:  router,
		id:      id,
		loading: true,
	}

	switch {
	case strings.HasPrefix(id, "user:"):
		dv.kind = "user"
		dv.name = strings.TrimPrefix(id, "user:")
		dv.tabs = ui.NewTabController([]string{"Overview", "Policies", "Inline Policies", "Groups"})
	case strings.HasPrefix(id, "role:"):
		dv.kind = "role"
		dv.name = strings.TrimPrefix(id, "role:")
		dv.tabs = ui.NewTabController([]string{"Overview", "Trust Policy", "Policies", "Inline Policies"})
	default:
		dv.kind = "policy"
		dv.name = id
		dv.tabs = ui.NewTabController([]string{"Overview", "Document", "Entities"})
	}

	return dv
}

func (dv *DetailView) Init() tea.Cmd {
	client := dv.client
	name := dv.name

	switch dv.kind {
	case "user":
		return func() tea.Msg {
			ctx := context.TODO()
			users, err := client.ListUsers(ctx)
			if err != nil {
				return userDetailMsg{err: err}
			}
			var found *awsiam.IAMUser
			for _, u := range users {
				if u.Name == name {
					found = &u
					break
				}
			}
			if found == nil {
				return userDetailMsg{err: fmt.Errorf("user %s not found", name)}
			}
			policies, _ := client.ListAttachedUserPolicies(ctx, name)
			inlinePolicies, _ := client.ListInlineUserPolicies(ctx, name)
			groups, _ := client.ListGroupsForUser(ctx, name)
			return userDetailMsg{user: *found, policies: policies, inlinePolicies: inlinePolicies, groups: groups}
		}
	case "role":
		return func() tea.Msg {
			ctx := context.TODO()
			roles, err := client.ListRoles(ctx)
			if err != nil {
				return roleDetailMsg{err: err}
			}
			var found *awsiam.IAMRole
			for _, r := range roles {
				if r.Name == name {
					found = &r
					break
				}
			}
			if found == nil {
				return roleDetailMsg{err: fmt.Errorf("role %s not found", name)}
			}
			policies, _ := client.ListAttachedRolePolicies(ctx, name)
			inlinePolicies, _ := client.ListInlineRolePolicies(ctx, name)
			return roleDetailMsg{role: *found, policies: policies, inlinePolicies: inlinePolicies}
		}
	default: // policy
		return func() tea.Msg {
			ctx := context.TODO()
			policies, err := client.ListPolicies(ctx)
			if err != nil {
				return policyDetailMsg{err: err}
			}
			var found *awsiam.IAMPolicy
			for _, p := range policies {
				if p.ARN == name {
					found = &p
					break
				}
			}
			if found == nil {
				return policyDetailMsg{err: fmt.Errorf("policy %s not found", name)}
			}
			var doc string
			if found.DefaultVersionID != "" {
				doc, _ = client.GetPolicyDocument(ctx, found.ARN, found.DefaultVersionID)
			}
			entities, _ := client.ListEntitiesForPolicy(ctx, name)
			return policyDetailMsg{policy: *found, document: doc, entities: entities}
		}
	}
}

func (dv *DetailView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case userDetailMsg:
		dv.loading = false
		if msg.err != nil {
			dv.err = msg.err
			return dv, nil
		}
		dv.user = &msg.user
		dv.userPolicies = msg.policies
		dv.userInlinePolicies = msg.inlinePolicies
		dv.userGroups = msg.groups
		return dv, nil

	case roleDetailMsg:
		dv.loading = false
		if msg.err != nil {
			dv.err = msg.err
			return dv, nil
		}
		dv.role = &msg.role
		dv.rolePolicies = msg.policies
		dv.roleInlinePolicies = msg.inlinePolicies
		return dv, nil

	case policyDetailMsg:
		dv.loading = false
		if msg.err != nil {
			dv.err = msg.err
			return dv, nil
		}
		dv.policy = &msg.policy
		dv.policyDocument = msg.document
		dv.policyEntities = msg.entities
		return dv, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "backspace":
			dv.router.Pop()
			return dv, nil
		}
	}

	var cmd tea.Cmd
	dv.tabs, cmd = dv.tabs.Update(msg)
	return dv, cmd
}

func (dv *DetailView) View() tea.View {
	if dv.loading {
		skel := ui.NewSkeleton(60, 8)
		return tea.NewView(skel.View())
	}
	if dv.err != nil {
		return tea.NewView("Error: " + dv.err.Error())
	}

	var b strings.Builder
	b.WriteString(dv.tabs.View())
	b.WriteString("\n\n")

	switch dv.kind {
	case "user":
		b.WriteString(dv.renderUser())
	case "role":
		b.WriteString(dv.renderRole())
	default:
		b.WriteString(dv.renderPolicy())
	}

	return tea.NewView(b.String())
}

func (dv *DetailView) renderUser() string {
	switch dv.tabs.Active() {
	case 0:
		return dv.renderUserOverview()
	case 1:
		return dv.renderAttachedPolicies(dv.userPolicies)
	case 2:
		return dv.renderInlinePolicies(dv.userInlinePolicies)
	case 3:
		return dv.renderGroups()
	}
	return ""
}

func (dv *DetailView) renderRole() string {
	switch dv.tabs.Active() {
	case 0:
		return dv.renderRoleOverview()
	case 1:
		return renderJSON(dv.role.AssumeRolePolicyDocument, "No trust policy document.")
	case 2:
		return dv.renderAttachedPolicies(dv.rolePolicies)
	case 3:
		return dv.renderInlinePolicies(dv.roleInlinePolicies)
	}
	return ""
}

func (dv *DetailView) renderPolicy() string {
	switch dv.tabs.Active() {
	case 0:
		return dv.renderPolicyOverview()
	case 1:
		return renderJSON(dv.policyDocument, "No policy document.")
	case 2:
		return dv.renderEntities()
	}
	return ""
}

func (dv *DetailView) renderUserOverview() string {
	u := dv.user
	rows := []ui.KV{
		{K: "Name", V: u.Name},
		{K: "User ID", V: u.UserID},
		{K: "ARN", V: u.ARN},
		{K: "Path", V: u.Path},
		{K: "Created", V: u.CreatedAt.Format("2006-01-02 15:04:05 UTC")},
	}
	return ui.RenderKV(rows, 20, 0)
}

func (dv *DetailView) renderRoleOverview() string {
	r := dv.role
	rows := []ui.KV{
		{K: "Name", V: r.Name},
		{K: "Role ID", V: r.RoleID},
		{K: "ARN", V: r.ARN},
		{K: "Path", V: r.Path},
		{K: "Description", V: r.Description},
		{K: "Created", V: r.CreatedAt.Format("2006-01-02 15:04:05 UTC")},
	}
	return ui.RenderKV(rows, 20, 0)
}

func (dv *DetailView) renderPolicyOverview() string {
	p := dv.policy
	rows := []ui.KV{
		{K: "Name", V: p.Name},
		{K: "Policy ID", V: p.PolicyID},
		{K: "ARN", V: p.ARN},
		{K: "Path", V: p.Path},
		{K: "Attachment Count", V: fmt.Sprintf("%d", p.AttachmentCount)},
		{K: "Default Version", V: p.DefaultVersionID},
		{K: "Created", V: p.CreatedAt.Format("2006-01-02 15:04:05 UTC")},
		{K: "Updated", V: p.UpdatedAt.Format("2006-01-02 15:04:05 UTC")},
	}
	return ui.RenderKV(rows, 20, 0)
}

var jsonKeyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))

func renderJSON(doc string, emptyMsg string) string {
	if doc == "" {
		return emptyMsg
	}
	// Pretty-print the JSON.
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, []byte(doc), "", "  "); err != nil {
		// If it's not valid JSON, return as-is.
		return doc
	}

	// Syntax highlight: color the keys.
	var b strings.Builder
	for _, line := range strings.Split(pretty.String(), "\n") {
		// Find "key": pattern and colorize the key portion.
		if idx := strings.Index(line, `":`); idx >= 0 {
			keyStart := strings.Index(line, `"`)
			if keyStart >= 0 && keyStart < idx {
				b.WriteString(line[:keyStart])
				b.WriteString(jsonKeyStyle.Render(line[keyStart : idx+1]))
				b.WriteString(line[idx+1:])
				b.WriteByte('\n')
				continue
			}
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func (dv *DetailView) renderAttachedPolicies(policies []awsiam.IAMAttachedPolicy) string {
	if len(policies) == 0 {
		return "No attached policies."
	}

	cols := []ui.Column[awsiam.IAMAttachedPolicy]{
		{Title: "Policy Name", Width: 30, Field: func(p awsiam.IAMAttachedPolicy) string { return p.Name }},
		{Title: "ARN", Width: 60, Field: func(p awsiam.IAMAttachedPolicy) string { return p.ARN }},
	}
	tv := ui.NewTableView(cols, policies, func(p awsiam.IAMAttachedPolicy) string { return p.ARN })
	return tv.View()
}

func (dv *DetailView) renderInlinePolicies(policies []awsiam.IAMInlinePolicy) string {
	if len(policies) == 0 {
		return "No inline policies."
	}

	var b strings.Builder
	for i, p := range policies {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render(p.Name))
		b.WriteString("\n")
		b.WriteString(renderJSON(p.Document, "  (empty document)"))
	}
	return b.String()
}

func (dv *DetailView) renderGroups() string {
	if len(dv.userGroups) == 0 {
		return "No groups."
	}

	cols := []ui.Column[awsiam.IAMGroup]{
		{Title: "Group Name", Width: 24, Field: func(g awsiam.IAMGroup) string { return g.Name }},
		{Title: "ARN", Width: 60, Field: func(g awsiam.IAMGroup) string { return g.ARN }},
	}
	tv := ui.NewTableView(cols, dv.userGroups, func(g awsiam.IAMGroup) string { return g.Name })
	return tv.View()
}

func (dv *DetailView) renderEntities() string {
	if len(dv.policyEntities) == 0 {
		return "No attached entities."
	}

	cols := []ui.Column[awsiam.IAMPolicyEntity]{
		{Title: "Entity Name", Width: 30, Field: func(e awsiam.IAMPolicyEntity) string { return e.Name }},
		{Title: "Type", Width: 12, Field: func(e awsiam.IAMPolicyEntity) string { return e.Type }},
	}
	tv := ui.NewTableView(cols, dv.policyEntities, func(e awsiam.IAMPolicyEntity) string { return e.Name })
	return tv.View()
}

func (dv *DetailView) Title() string {
	return dv.name
}

func (dv *DetailView) KeyHints() []plugin.KeyHint {
	return []plugin.KeyHint{
		{Key: "esc", Desc: "back"},
		{Key: "[/]", Desc: "switch tab"},
	}
}
