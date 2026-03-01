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

func TestRenderMonthHeader(t *testing.T) {
	tests := []struct {
		name          string
		selectedMonth time.Time
		wantContains  []string
	}{
		{
			name:          "zero time shows current month with current label",
			selectedMonth: time.Time{},
			wantContains:  []string{time.Now().Format("January 2006"), "(current)"},
		},
		{
			name:          "past month shows month name without current label",
			selectedMonth: time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
			wantContains:  []string{"December 2025"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "test", "")
			m.selectedMonth = tt.selectedMonth
			header := m.renderMonthHeader()
			for _, want := range tt.wantContains {
				if !strings.Contains(header, want) {
					t.Errorf("renderMonthHeader() missing %q", want)
				}
			}
			// Past months should NOT contain "(current)"
			if !tt.selectedMonth.IsZero() {
				if strings.Contains(header, "(current)") {
					t.Error("past month should not have (current) label")
				}
			}
		})
	}
}

func TestBuildServiceDrillDown(t *testing.T) {
	tests := []struct {
		name         string
		drillService string
		data         *awscost.CostData
		wantContains []string
		wantEmpty    bool
	}{
		{
			name:         "nil data shows no data message",
			drillService: "Amazon EC2",
			data:         nil,
			wantContains: []string{"No data for Amazon EC2"},
		},
		{
			name:         "missing service shows no daily data message",
			drillService: "Amazon RDS",
			data: &awscost.CostData{
				Currency:        "USD",
				ServiceDailyMap: map[string]map[string]float64{"Amazon EC2": {"2026-01-01": 10.0}},
			},
			wantContains: []string{"No daily data for Amazon RDS"},
		},
		{
			name:         "valid service shows chart and total",
			drillService: "Amazon EC2",
			data: &awscost.CostData{
				Currency: "USD",
				ServiceDailyMap: map[string]map[string]float64{
					"Amazon EC2": {
						"2026-01-01": 10.0,
						"2026-01-02": 15.0,
						"2026-01-03": 12.0,
					},
				},
			},
			wantContains: []string{"Amazon EC2", "$37.00", "Daily Spend"},
		},
		{
			name:         "single day service shows total but no chart",
			drillService: "Amazon S3",
			data: &awscost.CostData{
				Currency: "USD",
				ServiceDailyMap: map[string]map[string]float64{
					"Amazon S3": {"2026-01-01": 5.50},
				},
			},
			wantContains: []string{"Amazon S3", "$5.50"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "test", "")
			m.width = 100
			m.data = tt.data
			m.drillService = tt.drillService

			result := m.buildServiceDrillDown()

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("buildServiceDrillDown() missing %q", want)
				}
			}
		})
	}
}

func TestView_ServiceDrillDown(t *testing.T) {
	m := NewModel(nil, "test", "")
	m.loading = false
	m.drillService = "Amazon EC2"
	m.data = &awscost.CostData{
		Currency: "USD",
		ServiceDailyMap: map[string]map[string]float64{
			"Amazon EC2": {
				"2026-01-01": 10.0,
				"2026-01-02": 15.0,
			},
		},
		LastUpdated: time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC),
	}

	view := m.View().Content
	if !strings.Contains(view, "Amazon EC2") {
		t.Error("drill-down view should show service name")
	}
	if !strings.Contains(view, "Esc back") {
		t.Error("drill-down view should show Esc hint")
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

func TestUpdate_MonthNavigation(t *testing.T) {
	now := time.Now()
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	prevMonthStart := currentMonthStart.AddDate(0, -1, 0)
	minMonth := time.Date(now.Year()-1, now.Month(), 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		selectedMonth time.Time
		key           tea.KeyPressMsg
		wantMonth     time.Time
		wantLoading   bool
	}{
		{
			name:          "previous month from current",
			selectedMonth: time.Time{},
			key:           tea.KeyPressMsg{Code: '[', Text: "["},
			wantMonth:     prevMonthStart,
			wantLoading:   true,
		},
		{
			name:          "previous month cap at 12 months",
			selectedMonth: minMonth,
			key:           tea.KeyPressMsg{Code: '[', Text: "["},
			wantMonth:     minMonth,
			wantLoading:   false,
		},
		{
			name:          "next month from past",
			selectedMonth: currentMonthStart.AddDate(0, -2, 0),
			key:           tea.KeyPressMsg{Code: ']', Text: "]"},
			wantMonth:     currentMonthStart.AddDate(0, -1, 0),
			wantLoading:   true,
		},
		{
			name:          "next month already current",
			selectedMonth: time.Time{},
			key:           tea.KeyPressMsg{Code: ']', Text: "]"},
			wantMonth:     time.Time{},
			wantLoading:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "test", "")
			m.selectedMonth = tt.selectedMonth
			m.loading = false

			updated, _ := m.Update(tt.key)
			model := updated.(Model)

			if !model.selectedMonth.Equal(tt.wantMonth) {
				t.Errorf("selectedMonth = %v, want %v", model.selectedMonth, tt.wantMonth)
			}
			if model.loading != tt.wantLoading {
				t.Errorf("loading = %v, want %v", model.loading, tt.wantLoading)
			}
		})
	}
}

func TestUpdate_ServiceDrillDown(t *testing.T) {
	tests := []struct {
		name             string
		drillService     string
		key              tea.KeyPressMsg
		wantDrillService string
	}{
		{
			name:             "enter drills into selected service",
			drillService:     "",
			key:              tea.KeyPressMsg{Code: tea.KeyEnter},
			wantDrillService: "Amazon EC2",
		},
		{
			name:             "esc exits drill-down",
			drillService:     "Amazon EC2",
			key:              tea.KeyPressMsg{Code: tea.KeyEscape},
			wantDrillService: "",
		},
		{
			name:             "esc with no drill-down does nothing",
			drillService:     "",
			key:              tea.KeyPressMsg{Code: tea.KeyEscape},
			wantDrillService: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "test", "")
			m.loading = false
			m.drillService = tt.drillService
			m.data = &awscost.CostData{
				Currency: "USD",
				TopServices: []awscost.ServiceCost{
					{Name: "Amazon EC2", Cost: 89.12},
					{Name: "Amazon S3", Cost: 42.30},
				},
			}
			m.table.SetRows(m.buildRows())

			updated, _ := m.Update(tt.key)
			model := updated.(Model)

			if model.drillService != tt.wantDrillService {
				t.Errorf("drillService = %q, want %q", model.drillService, tt.wantDrillService)
			}
		})
	}
}

func TestUpdate_Refresh(t *testing.T) {
	tests := []struct {
		name        string
		key         tea.KeyPressMsg
		wantLoading bool
		wantErr     bool
	}{
		{
			name:        "refresh sets loading",
			key:         tea.KeyPressMsg{Code: 'r', Text: "r"},
			wantLoading: true,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "test", "")
			m.loading = false
			m.err = fmt.Errorf("old error")

			updated, _ := m.Update(tt.key)
			model := updated.(Model)

			if model.loading != tt.wantLoading {
				t.Errorf("loading = %v, want %v", model.loading, tt.wantLoading)
			}
			if tt.wantErr && model.err == nil {
				t.Errorf("err = nil, want non-nil")
			}
			if !tt.wantErr && model.err != nil {
				t.Errorf("err = %v, want nil", model.err)
			}
		})
	}
}
