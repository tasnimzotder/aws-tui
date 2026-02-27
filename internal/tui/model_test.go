package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	awscost "tasnim.dev/aws-tui/internal/aws/cost"
	"tasnim.dev/aws-tui/internal/utils"
)

func TestView_Loading(t *testing.T) {
	m := NewModel(nil, "test-profile", "123456789012")
	m.loading = true

	view := m.View().Content
	if !strings.Contains(view, "Fetching cost data") {
		t.Error("loading view should contain 'Fetching cost data'")
	}
	if !strings.Contains(view, "test-profile") {
		t.Error("loading view should show profile name")
	}
	if !strings.Contains(view, "123456789012") {
		t.Error("loading view should show account ID")
	}
}

func TestView_WithData(t *testing.T) {
	m := NewModel(nil, "prod", "111122223333")
	m.loading = false
	m.data = &awscost.CostData{
		TodaySpend:     12.34,
		YesterdaySpend: 15.67,
		MTDSpend:       187.52,
		ForecastSpend:  245.80,
		Currency:       "USD",
		TopServices: []awscost.ServiceCost{
			{Name: "Amazon EC2", Cost: 89.12},
			{Name: "Amazon S3", Cost: 42.30},
		},
		DailySpend: []awscost.DailySpendEntry{
			{Date: "2026-02-23", Spend: 8.50},
			{Date: "2026-02-24", Spend: 15.67},
			{Date: "2026-02-25", Spend: 12.34},
		},
		LastUpdated: time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC),
	}
	m.table.SetRows(m.buildRows())

	view := m.View().Content
	if !strings.Contains(view, "$12.34") {
		t.Error("view should show today's spend")
	}
	if !strings.Contains(view, "$15.67") {
		t.Error("view should show yesterday's spend")
	}
	if !strings.Contains(view, "$187.52") {
		t.Error("view should show MTD spend")
	}
	if !strings.Contains(view, "$245.80") {
		t.Error("view should show forecast")
	}
	if !strings.Contains(view, "Amazon EC2") {
		t.Error("view should show top service")
	}
	if !strings.Contains(view, "prod") {
		t.Error("view should show profile name")
	}
	if !strings.Contains(view, "Daily Spend") {
		t.Error("view should show daily spend chart")
	}
}

func TestBuildChart_WithData(t *testing.T) {
	m := NewModel(nil, "test", "")
	m.loading = false
	m.data = &awscost.CostData{
		DailySpend: []awscost.DailySpendEntry{
			{Date: "2026-02-01", Spend: 10.00},
			{Date: "2026-02-02", Spend: 15.00},
			{Date: "2026-02-03", Spend: 12.00},
			{Date: "2026-02-04", Spend: 20.00},
		},
	}

	chart := m.buildChart()
	if chart == "" {
		t.Error("chart should not be empty with 4 days of data")
	}
	if !strings.Contains(chart, "Daily Spend") {
		t.Error("chart should contain caption 'Daily Spend'")
	}
}

func TestBuildChart_TooFewDays(t *testing.T) {
	m := NewModel(nil, "test", "")
	m.loading = false
	m.data = &awscost.CostData{
		DailySpend: []awscost.DailySpendEntry{
			{Date: "2026-02-01", Spend: 10.00},
		},
	}

	chart := m.buildChart()
	if chart != "" {
		t.Error("chart should be empty with only 1 day of data")
	}
}

func TestView_Error(t *testing.T) {
	m := NewModel(nil, "broken", "")
	m.loading = false
	m.err = fmt.Errorf("access denied")

	view := m.View().Content
	if !strings.Contains(view, "access denied") {
		t.Error("error view should show error message")
	}
	if !strings.Contains(view, "retry") {
		t.Error("error view should mention retry")
	}
}

func TestUpdate_WindowSizeMsg(t *testing.T) {
	m := NewModel(nil, "test", "")

	// Simulate a window resize
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updated, _ := m.Update(msg)
	model := updated.(Model)

	if model.width != 120 {
		t.Errorf("width = %d, want 120", model.width)
	}
	if model.height != 40 {
		t.Errorf("height = %d, want 40", model.height)
	}
}

func TestResizeTable_ClampsDimensions(t *testing.T) {
	m := NewModel(nil, "test", "")

	// Very small terminal
	m.width = 50
	m.height = 15
	m = m.resizeTable()

	// Service col should be clamped at min 20
	cols := m.table.Columns()
	if cols[0].Width < 20 {
		t.Errorf("service col width = %d, want >= 20", cols[0].Width)
	}

	// Very large terminal
	m.width = 200
	m.height = 60
	m = m.resizeTable()

	cols = m.table.Columns()
	if cols[0].Width <= 20 {
		t.Errorf("service col width = %d, want > 20 for wide terminal", cols[0].Width)
	}
}

func TestBuildChart_UsesModelWidth(t *testing.T) {
	m := NewModel(nil, "test", "")
	m.width = 120
	m.data = &awscost.CostData{
		DailySpend: []awscost.DailySpendEntry{
			{Date: "2026-02-01", Spend: 10.00},
			{Date: "2026-02-02", Spend: 15.00},
			{Date: "2026-02-03", Spend: 12.00},
		},
		LastUpdated: time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC),
	}

	chart := m.buildChart()
	if chart == "" {
		t.Error("chart should not be empty")
	}
}

func TestView_WithMoMDecrease(t *testing.T) {
	m := NewModel(nil, "prod", "")
	m.loading = false
	m.data = &awscost.CostData{
		TodaySpend:        12.34,
		MTDSpend:          187.52,
		ForecastSpend:     245.80,
		Currency:          "USD",
		LastMonthMTDSpend: 210.00,
		MoMChangePercent:  -10.7,
		TopServices: []awscost.ServiceCost{
			{Name: "Amazon EC2", Cost: 89.12},
		},
		DailySpend: []awscost.DailySpendEntry{
			{Date: "2026-02-24", Spend: 15.67},
			{Date: "2026-02-25", Spend: 12.34},
		},
		LastUpdated: time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC),
	}
	m.table.SetRows(m.buildRows())

	view := m.View().Content
	if !strings.Contains(view, "$210.00") {
		t.Error("view should show last month MTD spend")
	}
	if !strings.Contains(view, "10.7%") {
		t.Error("view should show MoM percentage")
	}
	if !strings.Contains(view, "down") {
		t.Error("view should indicate decrease direction")
	}
}

func TestView_WithMoMIncrease(t *testing.T) {
	m := NewModel(nil, "prod", "")
	m.loading = false
	m.data = &awscost.CostData{
		TodaySpend:        12.34,
		MTDSpend:          250.00,
		ForecastSpend:     300.00,
		Currency:          "USD",
		LastMonthMTDSpend: 200.00,
		MoMChangePercent:  25.0,
		TopServices: []awscost.ServiceCost{
			{Name: "Amazon EC2", Cost: 89.12},
		},
		DailySpend: []awscost.DailySpendEntry{
			{Date: "2026-02-24", Spend: 15.67},
			{Date: "2026-02-25", Spend: 12.34},
		},
		LastUpdated: time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC),
	}
	m.table.SetRows(m.buildRows())

	view := m.View().Content
	if !strings.Contains(view, "up") {
		t.Error("view should indicate increase direction")
	}
	if !strings.Contains(view, "25.0%") {
		t.Error("view should show MoM percentage")
	}
}

func TestView_WithAnomalies(t *testing.T) {
	m := NewModel(nil, "prod", "")
	m.loading = false
	m.data = &awscost.CostData{
		TodaySpend:    45.00,
		MTDSpend:      187.52,
		ForecastSpend: 245.80,
		Currency:      "USD",
		TopServices: []awscost.ServiceCost{
			{Name: "Amazon EC2", Cost: 89.12},
		},
		DailySpend: []awscost.DailySpendEntry{
			{Date: "2026-02-24", Spend: 15.67},
			{Date: "2026-02-25", Spend: 45.00},
		},
		Anomalies: []awscost.ServiceAnomaly{
			{ServiceName: "Amazon EC2", TodaySpend: 45.00, AvgSpend: 18.00, Ratio: 2.5},
		},
		LastUpdated: time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC),
	}
	m.table.SetRows(m.buildRows())

	view := m.View().Content
	if !strings.Contains(view, "Anomalies") {
		t.Error("view should show anomalies header")
	}
	if !strings.Contains(view, "Amazon EC2") {
		t.Error("view should show anomalous service name")
	}
	if !strings.Contains(view, "2.5x") {
		t.Error("view should show anomaly ratio")
	}
}

func TestCurrency(t *testing.T) {
	tests := []struct {
		amount   float64
		currency string
		want     string
	}{
		{12.34, "USD", "$12.34"},
		{0.00, "USD", "$0.00"},
		{1234.56, "", "$1234.56"},
		{50.00, "EUR", "EUR 50.00"},
	}

	for _, tt := range tests {
		got := utils.Currency(tt.amount, tt.currency)
		if got != tt.want {
			t.Errorf("Currency(%f, %q) = %q, want %q", tt.amount, tt.currency, got, tt.want)
		}
	}
}
