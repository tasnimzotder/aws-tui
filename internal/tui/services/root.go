package services

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	awsclient "tasnim.dev/aws-tui/internal/aws"
)

type serviceItem struct {
	name string
	desc string
}

func (i serviceItem) Title() string       { return i.name }
func (i serviceItem) Description() string { return i.desc }
func (i serviceItem) FilterValue() string { return i.name }

type RootView struct {
	client *awsclient.ServiceClient
	list   list.Model
}

func NewRootView(client *awsclient.ServiceClient) *RootView {
	items := []list.Item{
		serviceItem{name: "EC2", desc: "Elastic Compute Cloud — Instances"},
		serviceItem{name: "ECR", desc: "Elastic Container Registry — Repositories, Images"},
		serviceItem{name: "ECS", desc: "Elastic Container Service — Clusters, Services, Tasks"},
		serviceItem{name: "ELB", desc: "Elastic Load Balancing — Load Balancers, Listeners, Target Groups"},
		serviceItem{name: "IAM", desc: "Identity & Access Management — Users, Roles, Policies"},
		serviceItem{name: "S3", desc: "Simple Storage Service — Buckets, Objects"},
		serviceItem{name: "VPC", desc: "Virtual Private Cloud — VPCs, Subnets, Security Groups"},
	}

	l := list.New(items, list.NewDefaultDelegate(), 60, 14)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	return &RootView{
		client: client,
		list:   l,
	}
}

func (v *RootView) Title() string { return "Services" }

func (v *RootView) Init() tea.Cmd { return nil }

func (v *RootView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			selected, ok := v.list.SelectedItem().(serviceItem)
			if !ok {
				return v, nil
			}
			return v, v.handleSelection(selected.name)
		}
	}

	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}

func (v *RootView) handleSelection(name string) tea.Cmd {
	switch name {
	case "EC2":
		return func() tea.Msg {
			return PushViewMsg{View: NewEC2View(v.client)}
		}
	case "ECS":
		return func() tea.Msg {
			return PushViewMsg{View: NewECSClustersView(v.client)}
		}
	case "VPC":
		return func() tea.Msg {
			return PushViewMsg{View: NewVPCListView(v.client)}
		}
	case "ECR":
		return func() tea.Msg {
			return PushViewMsg{View: NewECRReposView(v.client)}
		}
	case "ELB":
		return func() tea.Msg {
			return PushViewMsg{View: NewELBLoadBalancersView(v.client)}
		}
	case "S3":
		return func() tea.Msg {
			return PushViewMsg{View: NewS3BucketsView(v.client)}
		}
	case "IAM":
		return func() tea.Msg {
			return PushViewMsg{View: NewIAMSubMenuView(v.client)}
		}
	}
	return nil
}

func (v *RootView) View() string {
	return v.list.View()
}

func (v *RootView) SetSize(width, height int) {
	v.list.SetSize(width, height)
}
