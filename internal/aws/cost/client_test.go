package cost

import (
	"context"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

type mockCostExplorerAPI struct {
	getCostAndUsageFunc func(ctx context.Context, params *costexplorer.GetCostAndUsageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error)
	getCostForecastFunc func(ctx context.Context, params *costexplorer.GetCostForecastInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostForecastOutput, error)
}

func (m *mockCostExplorerAPI) GetCostAndUsage(ctx context.Context, params *costexplorer.GetCostAndUsageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error) {
	return m.getCostAndUsageFunc(ctx, params, optFns...)
}

func (m *mockCostExplorerAPI) GetCostForecast(ctx context.Context, params *costexplorer.GetCostForecastInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostForecastOutput, error) {
	return m.getCostForecastFunc(ctx, params, optFns...)
}

func TestFetchCostData_AggregatesCorrectly(t *testing.T) {
	mock := &mockCostExplorerAPI{
		getCostAndUsageFunc: func(ctx context.Context, params *costexplorer.GetCostAndUsageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error) {
			return &costexplorer.GetCostAndUsageOutput{
				ResultsByTime: []types.ResultByTime{
					{
						TimePeriod: &types.DateInterval{
							Start: awssdk.String("2026-02-01"),
							End:   awssdk.String("2026-02-02"),
						},
						Groups: []types.Group{
							{
								Keys: []string{"Amazon EC2"},
								Metrics: map[string]types.MetricValue{
									"UnblendedCost": {Amount: awssdk.String("50.00"), Unit: awssdk.String("USD")},
								},
							},
							{
								Keys: []string{"Amazon S3"},
								Metrics: map[string]types.MetricValue{
									"UnblendedCost": {Amount: awssdk.String("20.00"), Unit: awssdk.String("USD")},
								},
							},
						},
					},
				},
			}, nil
		},
		getCostForecastFunc: func(ctx context.Context, params *costexplorer.GetCostForecastInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostForecastOutput, error) {
			return &costexplorer.GetCostForecastOutput{
				Total: &types.MetricValue{
					Amount: awssdk.String("300.00"),
					Unit:   awssdk.String("USD"),
				},
			}, nil
		},
	}

	client := NewClientWithAPI(mock)
	client.now = func() time.Time { return time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC) }
	data, err := client.FetchCostData(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data.MTDSpend != 70.00 {
		t.Errorf("MTDSpend = %f, want 70.00", data.MTDSpend)
	}
	if data.ForecastSpend != 300.00 {
		t.Errorf("ForecastSpend = %f, want 300.00", data.ForecastSpend)
	}
	if data.Currency != "USD" {
		t.Errorf("Currency = %s, want USD", data.Currency)
	}
	if len(data.TopServices) != 2 {
		t.Fatalf("TopServices length = %d, want 2", len(data.TopServices))
	}
	if data.TopServices[0].Name != "Amazon EC2" {
		t.Errorf("TopServices[0].Name = %s, want Amazon EC2", data.TopServices[0].Name)
	}
	if data.TopServices[0].Cost != 50.00 {
		t.Errorf("TopServices[0].Cost = %f, want 50.00", data.TopServices[0].Cost)
	}
}

func TestFetchCostData_SortsServicesByDescendingCost(t *testing.T) {
	mock := &mockCostExplorerAPI{
		getCostAndUsageFunc: func(ctx context.Context, params *costexplorer.GetCostAndUsageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error) {
			return &costexplorer.GetCostAndUsageOutput{
				ResultsByTime: []types.ResultByTime{
					{
						TimePeriod: &types.DateInterval{
							Start: awssdk.String("2026-02-01"),
							End:   awssdk.String("2026-02-02"),
						},
						Groups: []types.Group{
							{
								Keys:    []string{"AWS Lambda"},
								Metrics: map[string]types.MetricValue{"UnblendedCost": {Amount: awssdk.String("5.00"), Unit: awssdk.String("USD")}},
							},
							{
								Keys:    []string{"Amazon EC2"},
								Metrics: map[string]types.MetricValue{"UnblendedCost": {Amount: awssdk.String("100.00"), Unit: awssdk.String("USD")}},
							},
							{
								Keys:    []string{"Amazon S3"},
								Metrics: map[string]types.MetricValue{"UnblendedCost": {Amount: awssdk.String("30.00"), Unit: awssdk.String("USD")}},
							},
						},
					},
				},
			}, nil
		},
		getCostForecastFunc: func(ctx context.Context, params *costexplorer.GetCostForecastInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostForecastOutput, error) {
			return &costexplorer.GetCostForecastOutput{
				Total: &types.MetricValue{Amount: awssdk.String("200.00"), Unit: awssdk.String("USD")},
			}, nil
		},
	}

	client := NewClientWithAPI(mock)
	client.now = func() time.Time { return time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC) }
	data, err := client.FetchCostData(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"Amazon EC2", "Amazon S3", "AWS Lambda"}
	for i, svc := range data.TopServices {
		if svc.Name != expected[i] {
			t.Errorf("TopServices[%d].Name = %s, want %s", i, svc.Name, expected[i])
		}
	}
}

func TestDetectAnomalies_SpikeDetected(t *testing.T) {
	today := "2026-02-08"
	serviceDailyMap := map[string]map[string]float64{
		"Amazon EC2": {
			"2026-02-01": 10.0, "2026-02-02": 10.0, "2026-02-03": 10.0, "2026-02-04": 10.0,
			"2026-02-05": 10.0, "2026-02-06": 10.0, "2026-02-07": 10.0, "2026-02-08": 30.0,
		},
	}

	anomalies := detectAnomalies(serviceDailyMap, today)
	if len(anomalies) != 1 {
		t.Fatalf("expected 1 anomaly, got %d", len(anomalies))
	}
	if anomalies[0].ServiceName != "Amazon EC2" {
		t.Errorf("ServiceName = %s, want Amazon EC2", anomalies[0].ServiceName)
	}
	if anomalies[0].Ratio < 2.0 {
		t.Errorf("Ratio = %f, want >= 2.0", anomalies[0].Ratio)
	}
}

func TestDetectAnomalies_SteadySpend(t *testing.T) {
	today := "2026-02-08"
	serviceDailyMap := map[string]map[string]float64{
		"Amazon EC2": {
			"2026-02-01": 10.0, "2026-02-02": 10.0, "2026-02-03": 10.0, "2026-02-04": 10.0,
			"2026-02-05": 10.0, "2026-02-06": 10.0, "2026-02-07": 10.0, "2026-02-08": 12.0,
		},
	}
	anomalies := detectAnomalies(serviceDailyMap, today)
	if len(anomalies) != 0 {
		t.Errorf("expected 0 anomalies for steady spend, got %d", len(anomalies))
	}
}

func TestDetectAnomalies_InsufficientHistory(t *testing.T) {
	today := "2026-02-03"
	serviceDailyMap := map[string]map[string]float64{
		"Amazon EC2": {"2026-02-02": 5.0, "2026-02-03": 50.0},
	}
	anomalies := detectAnomalies(serviceDailyMap, today)
	if len(anomalies) != 0 {
		t.Errorf("expected 0 anomalies with insufficient history, got %d", len(anomalies))
	}
}

func TestDetectAnomalies_BelowThresholds(t *testing.T) {
	today := "2026-02-08"
	serviceDailyMap := map[string]map[string]float64{
		"TinyService": {
			"2026-02-01": 0.10, "2026-02-02": 0.10, "2026-02-03": 0.10, "2026-02-04": 0.10,
			"2026-02-05": 0.10, "2026-02-06": 0.10, "2026-02-07": 0.10, "2026-02-08": 0.80,
		},
	}
	anomalies := detectAnomalies(serviceDailyMap, today)
	if len(anomalies) != 0 {
		t.Errorf("expected 0 anomalies for below-threshold spend, got %d", len(anomalies))
	}
}

func TestFetchCostData_YesterdaySpend(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)

	mock := &mockCostExplorerAPI{
		getCostAndUsageFunc: func(ctx context.Context, params *costexplorer.GetCostAndUsageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error) {
			return &costexplorer.GetCostAndUsageOutput{
				ResultsByTime: []types.ResultByTime{
					{
						TimePeriod: &types.DateInterval{Start: awssdk.String(yesterday.Format("2006-01-02")), End: awssdk.String(today.Format("2006-01-02"))},
						Groups: []types.Group{{
							Keys:    []string{"Amazon EC2"},
							Metrics: map[string]types.MetricValue{"UnblendedCost": {Amount: awssdk.String("45.00"), Unit: awssdk.String("USD")}},
						}},
					},
					{
						TimePeriod: &types.DateInterval{Start: awssdk.String(today.Format("2006-01-02")), End: awssdk.String(today.AddDate(0, 0, 1).Format("2006-01-02"))},
						Groups: []types.Group{{
							Keys:    []string{"Amazon EC2"},
							Metrics: map[string]types.MetricValue{"UnblendedCost": {Amount: awssdk.String("10.00"), Unit: awssdk.String("USD")}},
						}},
					},
				},
			}, nil
		},
		getCostForecastFunc: func(ctx context.Context, params *costexplorer.GetCostForecastInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostForecastOutput, error) {
			return &costexplorer.GetCostForecastOutput{
				Total: &types.MetricValue{Amount: awssdk.String("200.00"), Unit: awssdk.String("USD")},
			}, nil
		},
	}

	client := NewClientWithAPI(mock)
	data, err := client.FetchCostData(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.YesterdaySpend != 45.00 {
		t.Errorf("YesterdaySpend = %f, want 45.00", data.YesterdaySpend)
	}
	if data.TodaySpend != 10.00 {
		t.Errorf("TodaySpend = %f, want 10.00", data.TodaySpend)
	}
}

func TestFetchCostData_MonthOverMonth(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastMonthStart := monthStart.AddDate(0, -1, 0)

	mock := &mockCostExplorerAPI{
		getCostAndUsageFunc: func(ctx context.Context, params *costexplorer.GetCostAndUsageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error) {
			startDate := awssdk.ToString(params.TimePeriod.Start)
			if startDate == lastMonthStart.Format("2006-01-02") {
				return &costexplorer.GetCostAndUsageOutput{
					ResultsByTime: []types.ResultByTime{{
						TimePeriod: &types.DateInterval{Start: awssdk.String(lastMonthStart.Format("2006-01-02")), End: awssdk.String(monthStart.Format("2006-01-02"))},
						Total:      map[string]types.MetricValue{"UnblendedCost": {Amount: awssdk.String("200.00"), Unit: awssdk.String("USD")}},
					}},
				}, nil
			}
			return &costexplorer.GetCostAndUsageOutput{
				ResultsByTime: []types.ResultByTime{{
					TimePeriod: &types.DateInterval{Start: awssdk.String(today.Format("2006-01-02")), End: awssdk.String(today.AddDate(0, 0, 1).Format("2006-01-02"))},
					Groups: []types.Group{{
						Keys:    []string{"Amazon EC2"},
						Metrics: map[string]types.MetricValue{"UnblendedCost": {Amount: awssdk.String("100.00"), Unit: awssdk.String("USD")}},
					}},
				}},
			}, nil
		},
		getCostForecastFunc: func(ctx context.Context, params *costexplorer.GetCostForecastInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostForecastOutput, error) {
			return &costexplorer.GetCostForecastOutput{
				Total: &types.MetricValue{Amount: awssdk.String("300.00"), Unit: awssdk.String("USD")},
			}, nil
		},
	}

	client := NewClientWithAPI(mock)
	data, err := client.FetchCostData(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.LastMonthMTDSpend != 200.00 {
		t.Errorf("LastMonthMTDSpend = %f, want 200.00", data.LastMonthMTDSpend)
	}
	expectedMoM := -50.0
	if data.MoMChangePercent != expectedMoM {
		t.Errorf("MoMChangePercent = %f, want %f", data.MoMChangePercent, expectedMoM)
	}
}
