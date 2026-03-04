package cost

import (
	"context"
	"testing"
	"time"

	awscost "tasnim.dev/aws-tui/internal/aws/cost"
	"tasnim.dev/aws-tui/internal/plugin"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClient implements CostClient for testing.
type mockClient struct {
	data *awscost.CostData
	err  error
}

func (m *mockClient) FetchCostData(ctx context.Context) (*awscost.CostData, error) {
	return m.data, m.err
}

func (m *mockClient) FetchCostDataForMonth(ctx context.Context, target time.Time) (*awscost.CostData, error) {
	return m.data, m.err
}

func TestPluginMetadata(t *testing.T) {
	p := NewPlugin(&mockClient{})
	assert.Equal(t, "cost", p.ID())
	assert.Equal(t, "Cost Explorer", p.Name())
	assert.NotEmpty(t, p.Icon())
}

func TestSummary_Basic(t *testing.T) {
	mc := &mockClient{
		data: &awscost.CostData{
			MTDSpend: 1234.56,
			Currency: "USD",
			TopServices: []awscost.ServiceCost{
				{Name: "Amazon EC2", Cost: 800.00},
				{Name: "Amazon S3", Cost: 434.56},
			},
		},
	}
	p := NewPlugin(mc)
	summary, err := p.Summary(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 2, summary.Total)
	assert.Equal(t, "$1234.56 this month", summary.Label)
	assert.Equal(t, plugin.HealthHealthy, summary.Health)
}

func TestSummary_HighMoMChange(t *testing.T) {
	data := &awscost.CostData{
		MTDSpend:         500.00,
		MoMChangePercent: 25.0,
		TopServices: []awscost.ServiceCost{
			{Name: "Amazon EC2", Cost: 500.00},
		},
	}
	result := mapSummary(data)

	assert.Equal(t, plugin.HealthWarning, result.Health)
}

func TestSummary_CriticalMoMChange(t *testing.T) {
	data := &awscost.CostData{
		MTDSpend:         1000.00,
		MoMChangePercent: 55.0,
		TopServices: []awscost.ServiceCost{
			{Name: "Amazon EC2", Cost: 1000.00},
		},
	}
	result := mapSummary(data)

	assert.Equal(t, plugin.HealthCritical, result.Health)
}

func TestSummary_NoMoMChange(t *testing.T) {
	data := &awscost.CostData{
		MTDSpend:         100.00,
		MoMChangePercent: 0,
		TopServices:      []awscost.ServiceCost{},
	}
	result := mapSummary(data)

	assert.Equal(t, plugin.HealthHealthy, result.Health)
	assert.Equal(t, "$100.00 this month", result.Label)
}

func TestSummary_DefaultCurrency(t *testing.T) {
	data := &awscost.CostData{
		MTDSpend:    42.00,
		Currency:    "",
		TopServices: []awscost.ServiceCost{},
	}
	result := mapSummary(data)

	assert.Equal(t, "$42.00 this month", result.Label)
}

func TestSummary_Error(t *testing.T) {
	mc := &mockClient{
		err: assert.AnError,
	}
	p := NewPlugin(mc)
	_, err := p.Summary(context.Background())

	assert.Error(t, err)
}

func TestCommands(t *testing.T) {
	p := NewPlugin(&mockClient{})
	cmds := p.Commands()

	require.Len(t, cmds, 1)
	assert.Equal(t, "View Cost Explorer", cmds[0].Title)
	assert.Contains(t, cmds[0].Keywords, "cost")
	assert.Contains(t, cmds[0].Keywords, "billing")
	assert.Contains(t, cmds[0].Keywords, "spend")
}

func TestPollConfig(t *testing.T) {
	p := NewPlugin(&mockClient{})
	cfg := p.PollConfig()

	assert.Equal(t, 1*time.Hour, cfg.IdleInterval)
	assert.Equal(t, time.Duration(0), cfg.ActiveInterval)
	assert.NotNil(t, cfg.IsActive)
	assert.False(t, cfg.IsActive())
}
