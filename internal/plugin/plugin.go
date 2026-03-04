package plugin

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
)

type HealthLevel int

const (
	HealthHealthy  HealthLevel = iota
	HealthWarning
	HealthCritical
	HealthUnknown
)

type ToastLevel int

const (
	ToastInfo ToastLevel = iota
	ToastWarning
	ToastError
)

type ServiceSummary struct {
	Total  int
	Status map[string]int
	Health HealthLevel
	Label  string
}

type PollConfig struct {
	IdleInterval   time.Duration
	ActiveInterval time.Duration
	IsActive       func() bool
}

type Command struct {
	Title    string
	Keywords []string
	Action   func() tea.Cmd
}

type KeyHint struct {
	Key  string
	Desc string
}

type View interface {
	tea.Model
	Title() string
	KeyHints() []KeyHint
}

type Router interface {
	Push(view View)
	Pop()
	Navigate(pluginID string)
	NavigateDetail(pluginID, id string)
	Toast(level ToastLevel, msg string)
}

type ServicePlugin interface {
	ID() string
	Name() string
	Icon() string
	Summary(ctx context.Context) (ServiceSummary, error)
	ListView(router Router) View
	DetailView(router Router, id string) View
	Commands() []Command
	PollConfig() PollConfig
}
