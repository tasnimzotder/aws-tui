package plugin

import (
	"context"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mock plugin ---

type mockView struct{}

func (m mockView) Init() tea.Cmd                         { return nil }
func (m mockView) Update(tea.Msg) (tea.Model, tea.Cmd)   { return m, nil }
func (m mockView) View() tea.View                        { return tea.View{} }
func (m mockView) Title() string                         { return "mock" }
func (m mockView) KeyHints() []KeyHint                   { return nil }

type mockPlugin struct {
	id   string
	name string
}

func (m *mockPlugin) ID() string   { return m.id }
func (m *mockPlugin) Name() string { return m.name }
func (m *mockPlugin) Icon() string { return ">" }
func (m *mockPlugin) Summary(_ context.Context) (ServiceSummary, error) {
	return ServiceSummary{Total: 1, Health: HealthHealthy, Label: m.name}, nil
}
func (m *mockPlugin) ListView(_ Router) View              { return mockView{} }
func (m *mockPlugin) DetailView(_ Router, _ string) View  { return mockView{} }
func (m *mockPlugin) Commands() []Command                 { return nil }
func (m *mockPlugin) PollConfig() PollConfig {
	return PollConfig{
		IdleInterval:   30 * time.Second,
		ActiveInterval: 5 * time.Second,
		IsActive:       func() bool { return false },
	}
}

// --- tests ---

func TestRegistry(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "Get registered plugin returns it",
			run: func(t *testing.T) {
				r := NewRegistry()
				p := &mockPlugin{id: "ec2", name: "EC2"}
				r.Add(p)

				got := r.Get("ec2")
				require.NotNil(t, got)
				assert.Equal(t, "ec2", got.ID())
				assert.Equal(t, "EC2", got.Name())
			},
		},
		{
			name: "Get unregistered ID returns nil",
			run: func(t *testing.T) {
				r := NewRegistry()
				got := r.Get("nonexistent")
				assert.Nil(t, got)
			},
		},
		{
			name: "All returns plugins in registration order",
			run: func(t *testing.T) {
				r := NewRegistry()
				r.Add(&mockPlugin{id: "s3", name: "S3"})
				r.Add(&mockPlugin{id: "ec2", name: "EC2"})
				r.Add(&mockPlugin{id: "lambda", name: "Lambda"})

				all := r.All()
				require.Len(t, all, 3)
				assert.Equal(t, "s3", all[0].ID())
				assert.Equal(t, "ec2", all[1].ID())
				assert.Equal(t, "lambda", all[2].ID())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
