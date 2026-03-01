package cost

import (
	"context"
	"fmt"
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

func TestDetectAnomalies(t *testing.T) {
	tests := []struct {
		name            string
		serviceDailyMap map[string]map[string]float64
		today           string
		wantCount       int
		wantService     string
		wantMinRatio    float64
	}{
		{
			name: "spike detected",
			serviceDailyMap: map[string]map[string]float64{
				"Amazon EC2": {
					"2026-02-01": 10.0, "2026-02-02": 10.0, "2026-02-03": 10.0, "2026-02-04": 10.0,
					"2026-02-05": 10.0, "2026-02-06": 10.0, "2026-02-07": 10.0, "2026-02-08": 30.0,
				},
			},
			today:        "2026-02-08",
			wantCount:    1,
			wantService:  "Amazon EC2",
			wantMinRatio: 2.0,
		},
		{
			name: "steady spend",
			serviceDailyMap: map[string]map[string]float64{
				"Amazon EC2": {
					"2026-02-01": 10.0, "2026-02-02": 10.0, "2026-02-03": 10.0, "2026-02-04": 10.0,
					"2026-02-05": 10.0, "2026-02-06": 10.0, "2026-02-07": 10.0, "2026-02-08": 12.0,
				},
			},
			today:     "2026-02-08",
			wantCount: 0,
		},
		{
			name: "insufficient history",
			serviceDailyMap: map[string]map[string]float64{
				"Amazon EC2": {"2026-02-02": 5.0, "2026-02-03": 50.0},
			},
			today:     "2026-02-03",
			wantCount: 0,
		},
		{
			name: "below thresholds",
			serviceDailyMap: map[string]map[string]float64{
				"TinyService": {
					"2026-02-01": 0.10, "2026-02-02": 0.10, "2026-02-03": 0.10, "2026-02-04": 0.10,
					"2026-02-05": 0.10, "2026-02-06": 0.10, "2026-02-07": 0.10, "2026-02-08": 0.80,
				},
			},
			today:     "2026-02-08",
			wantCount: 0,
		},
		{
			name: "no today entry",
			serviceDailyMap: map[string]map[string]float64{
				"Amazon EC2": {
					"2026-02-01": 10.0, "2026-02-02": 10.0, "2026-02-03": 10.0, "2026-02-04": 10.0,
					"2026-02-05": 10.0, "2026-02-06": 10.0, "2026-02-07": 10.0,
				},
			},
			today:     "2026-02-08",
			wantCount: 0,
		},
		{
			name: "multiple services mixed",
			serviceDailyMap: map[string]map[string]float64{
				"Amazon EC2": {
					"2026-02-01": 10.0, "2026-02-02": 10.0, "2026-02-03": 10.0, "2026-02-04": 10.0,
					"2026-02-05": 10.0, "2026-02-06": 10.0, "2026-02-07": 10.0, "2026-02-08": 30.0,
				},
				"Amazon S3": {
					"2026-02-01": 5.0, "2026-02-02": 5.0, "2026-02-03": 5.0, "2026-02-04": 5.0,
					"2026-02-05": 5.0, "2026-02-06": 5.0, "2026-02-07": 5.0, "2026-02-08": 6.0,
				},
			},
			today:        "2026-02-08",
			wantCount:    1,
			wantService:  "Amazon EC2",
			wantMinRatio: 2.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			anomalies := detectAnomalies(tt.serviceDailyMap, tt.today)
			if len(anomalies) != tt.wantCount {
				t.Fatalf("expected %d anomalies, got %d", tt.wantCount, len(anomalies))
			}
			if tt.wantCount > 0 {
				if anomalies[0].ServiceName != tt.wantService {
					t.Errorf("ServiceName = %s, want %s", anomalies[0].ServiceName, tt.wantService)
				}
				if anomalies[0].Ratio < tt.wantMinRatio {
					t.Errorf("Ratio = %f, want >= %f", anomalies[0].Ratio, tt.wantMinRatio)
				}
			}
		})
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

func TestFetchCostDataForMonth(t *testing.T) {
	tests := []struct {
		name            string
		targetMonth     time.Time
		nowTime         time.Time
		wantMTD         float64
		wantForecast    float64
		wantServices    int
		wantFirstSvc    string
		wantDailyDays   int
		wantMoM         float64
		wantTargetMonth time.Time
	}{
		{
			name:            "past month returns full month data without forecast",
			targetMonth:     time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
			nowTime:         time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
			wantMTD:         150.00,
			wantForecast:    0,
			wantServices:    2,
			wantFirstSvc:    "Amazon EC2",
			wantDailyDays:   2,
			wantMoM:         50.0, // (150 - 100) / 100 * 100
			wantTargetMonth: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:            "current month delegates to FetchCostData",
			targetMonth:     time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC),
			nowTime:         time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
			wantMTD:         70.00,
			wantForecast:    300.00,
			wantServices:    2,
			wantFirstSvc:    "Amazon EC2",
			wantDailyDays:   1,
			wantMoM:         -30.0, // MoM from last-month comparison: (70-100)/100*100
			wantTargetMonth: time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC),
		},
		{
			name:            "past month with empty results",
			targetMonth:     time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
			nowTime:         time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
			wantMTD:         0,
			wantForecast:    0,
			wantServices:    0,
			wantDailyDays:   0,
			wantMoM:         0,
			wantTargetMonth: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCostExplorerAPI{
				getCostAndUsageFunc: func(ctx context.Context, params *costexplorer.GetCostAndUsageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error) {
					startDate := awssdk.ToString(params.TimePeriod.Start)

					// Current month usage (for delegation path)
					if startDate == tt.nowTime.Format("2006-01")+"-01" && params.Granularity == types.GranularityDaily {
						return &costexplorer.GetCostAndUsageOutput{
							ResultsByTime: []types.ResultByTime{
								{
									TimePeriod: &types.DateInterval{
										Start: awssdk.String(tt.nowTime.Format("2006-01")+"-01"),
										End:   awssdk.String(tt.nowTime.Format("2006-01")+"-02"),
									},
									Groups: []types.Group{
										{Keys: []string{"Amazon EC2"}, Metrics: map[string]types.MetricValue{"UnblendedCost": {Amount: awssdk.String("50.00"), Unit: awssdk.String("USD")}}},
										{Keys: []string{"Amazon S3"}, Metrics: map[string]types.MetricValue{"UnblendedCost": {Amount: awssdk.String("20.00"), Unit: awssdk.String("USD")}}},
									},
								},
							},
						}, nil
					}

					// Past month (target month) usage — daily granularity
					targetMonthStart := time.Date(tt.targetMonth.Year(), tt.targetMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
					if startDate == targetMonthStart.Format("2006-01-02") && params.Granularity == types.GranularityDaily {
						if tt.name == "past month with empty results" {
							return &costexplorer.GetCostAndUsageOutput{ResultsByTime: []types.ResultByTime{}}, nil
						}
						return &costexplorer.GetCostAndUsageOutput{
							ResultsByTime: []types.ResultByTime{
								{
									TimePeriod: &types.DateInterval{Start: awssdk.String("2026-01-01"), End: awssdk.String("2026-01-02")},
									Groups: []types.Group{
										{Keys: []string{"Amazon EC2"}, Metrics: map[string]types.MetricValue{"UnblendedCost": {Amount: awssdk.String("100.00"), Unit: awssdk.String("USD")}}},
										{Keys: []string{"Amazon S3"}, Metrics: map[string]types.MetricValue{"UnblendedCost": {Amount: awssdk.String("30.00"), Unit: awssdk.String("USD")}}},
									},
								},
								{
									TimePeriod: &types.DateInterval{Start: awssdk.String("2026-01-02"), End: awssdk.String("2026-01-03")},
									Groups: []types.Group{
										{Keys: []string{"Amazon EC2"}, Metrics: map[string]types.MetricValue{"UnblendedCost": {Amount: awssdk.String("20.00"), Unit: awssdk.String("USD")}}},
									},
								},
							},
						}, nil
					}

					// Previous month MoM (month before target) — monthly granularity
					prevMonthStart := targetMonthStart.AddDate(0, -1, 0)
					if startDate == prevMonthStart.Format("2006-01-02") && params.Granularity == types.GranularityMonthly {
						if tt.name == "past month with empty results" {
							return &costexplorer.GetCostAndUsageOutput{ResultsByTime: []types.ResultByTime{}}, nil
						}
						return &costexplorer.GetCostAndUsageOutput{
							ResultsByTime: []types.ResultByTime{
								{
									TimePeriod: &types.DateInterval{Start: awssdk.String(prevMonthStart.Format("2006-01-02")), End: awssdk.String(targetMonthStart.Format("2006-01-02"))},
									Total:      map[string]types.MetricValue{"UnblendedCost": {Amount: awssdk.String("100.00"), Unit: awssdk.String("USD")}},
								},
							},
						}, nil
					}

					// Last month MoM for current month delegation path
					return &costexplorer.GetCostAndUsageOutput{ResultsByTime: []types.ResultByTime{}}, nil
				},
				getCostForecastFunc: func(ctx context.Context, params *costexplorer.GetCostForecastInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostForecastOutput, error) {
					return &costexplorer.GetCostForecastOutput{
						Total: &types.MetricValue{Amount: awssdk.String("300.00"), Unit: awssdk.String("USD")},
					}, nil
				},
			}

			client := NewClientWithAPI(mock)
			client.now = func() time.Time { return tt.nowTime }
			data, err := client.FetchCostDataForMonth(context.Background(), tt.targetMonth)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if data.MTDSpend != tt.wantMTD {
				t.Errorf("MTDSpend = %f, want %f", data.MTDSpend, tt.wantMTD)
			}
			if data.ForecastSpend != tt.wantForecast {
				t.Errorf("ForecastSpend = %f, want %f", data.ForecastSpend, tt.wantForecast)
			}
			if len(data.TopServices) != tt.wantServices {
				t.Errorf("TopServices len = %d, want %d", len(data.TopServices), tt.wantServices)
			}
			if tt.wantServices > 0 && data.TopServices[0].Name != tt.wantFirstSvc {
				t.Errorf("TopServices[0].Name = %s, want %s", data.TopServices[0].Name, tt.wantFirstSvc)
			}
			if len(data.DailySpend) != tt.wantDailyDays {
				t.Errorf("DailySpend len = %d, want %d", len(data.DailySpend), tt.wantDailyDays)
			}
			if data.MoMChangePercent != tt.wantMoM {
				t.Errorf("MoMChangePercent = %f, want %f", data.MoMChangePercent, tt.wantMoM)
			}
			if data.TargetMonth != tt.wantTargetMonth {
				t.Errorf("TargetMonth = %v, want %v", data.TargetMonth, tt.wantTargetMonth)
			}
		})
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

func TestComputeDateRange(t *testing.T) {
	tests := []struct {
		name             string
		now              time.Time
		wantMonthStart   string
		wantToday        string
		wantTomorrow     string
		wantMonthEnd     string
		wantLastMStart   string
		wantLastMSameDay string
	}{
		{
			name:             "mid month",
			now:              time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
			wantMonthStart:   "2026-02-01",
			wantToday:        "2026-02-15",
			wantTomorrow:     "2026-02-16",
			wantMonthEnd:     "2026-03-01",
			wantLastMStart:   "2026-01-01",
			wantLastMSameDay: "2026-01-16",
		},
		{
			name:             "first of month",
			now:              time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			wantMonthStart:   "2026-03-01",
			wantToday:        "2026-03-01",
			wantTomorrow:     "2026-03-02",
			wantMonthEnd:     "2026-04-01",
			wantLastMStart:   "2026-02-01",
			wantLastMSameDay: "2026-02-02",
		},
		{
			name:             "end of month 31 to short month",
			now:              time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
			wantMonthStart:   "2026-03-01",
			wantToday:        "2026-03-31",
			wantTomorrow:     "2026-04-01",
			wantMonthEnd:     "2026-04-01",
			wantLastMStart:   "2026-02-01",
			wantLastMSameDay: "2026-03-01",
		},
		{
			name:             "leap year feb 29",
			now:              time.Date(2028, 2, 29, 0, 0, 0, 0, time.UTC),
			wantMonthStart:   "2028-02-01",
			wantToday:        "2028-02-29",
			wantTomorrow:     "2028-03-01",
			wantMonthEnd:     "2028-03-01",
			wantLastMStart:   "2028-01-01",
			wantLastMSameDay: "2028-01-30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dr := computeDateRange(tt.now)
			fmt := "2006-01-02"
			if got := dr.monthStart.Format(fmt); got != tt.wantMonthStart {
				t.Errorf("monthStart = %s, want %s", got, tt.wantMonthStart)
			}
			if got := dr.today.Format(fmt); got != tt.wantToday {
				t.Errorf("today = %s, want %s", got, tt.wantToday)
			}
			if got := dr.tomorrow.Format(fmt); got != tt.wantTomorrow {
				t.Errorf("tomorrow = %s, want %s", got, tt.wantTomorrow)
			}
			if got := dr.monthEnd.Format(fmt); got != tt.wantMonthEnd {
				t.Errorf("monthEnd = %s, want %s", got, tt.wantMonthEnd)
			}
			if got := dr.lastMonthStart.Format(fmt); got != tt.wantLastMStart {
				t.Errorf("lastMonthStart = %s, want %s", got, tt.wantLastMStart)
			}
			if got := dr.lastMonthSameDay.Format(fmt); got != tt.wantLastMSameDay {
				t.Errorf("lastMonthSameDay = %s, want %s", got, tt.wantLastMSameDay)
			}
		})
	}
}

func TestAggregateUsage(t *testing.T) {
	tests := []struct {
		name             string
		output           *costexplorer.GetCostAndUsageOutput
		today            time.Time
		yesterday        time.Time
		wantTodaySpend   float64
		wantYdaySpend    float64
		wantMTDSpend     float64
		wantCurrency     string
		wantServiceCount int
		wantDailyCount   int
	}{
		{
			name: "single day single service",
			output: &costexplorer.GetCostAndUsageOutput{
				ResultsByTime: []types.ResultByTime{
					{
						TimePeriod: &types.DateInterval{
							Start: awssdk.String("2026-02-15"),
							End:   awssdk.String("2026-02-16"),
						},
						Groups: []types.Group{
							{
								Keys: []string{"Amazon EC2"},
								Metrics: map[string]types.MetricValue{
									"UnblendedCost": {Amount: awssdk.String("50.00"), Unit: awssdk.String("USD")},
								},
							},
						},
					},
				},
			},
			today:            time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
			yesterday:        time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
			wantTodaySpend:   50.0,
			wantYdaySpend:    0.0,
			wantMTDSpend:     50.0,
			wantCurrency:     "USD",
			wantServiceCount: 1,
			wantDailyCount:   1,
		},
		{
			name: "multiple days multiple services",
			output: &costexplorer.GetCostAndUsageOutput{
				ResultsByTime: []types.ResultByTime{
					{
						TimePeriod: &types.DateInterval{
							Start: awssdk.String("2026-02-14"),
							End:   awssdk.String("2026-02-15"),
						},
						Groups: []types.Group{
							{
								Keys: []string{"Amazon EC2"},
								Metrics: map[string]types.MetricValue{
									"UnblendedCost": {Amount: awssdk.String("40.00"), Unit: awssdk.String("USD")},
								},
							},
							{
								Keys: []string{"Amazon S3"},
								Metrics: map[string]types.MetricValue{
									"UnblendedCost": {Amount: awssdk.String("10.00"), Unit: awssdk.String("USD")},
								},
							},
						},
					},
					{
						TimePeriod: &types.DateInterval{
							Start: awssdk.String("2026-02-15"),
							End:   awssdk.String("2026-02-16"),
						},
						Groups: []types.Group{
							{
								Keys: []string{"Amazon EC2"},
								Metrics: map[string]types.MetricValue{
									"UnblendedCost": {Amount: awssdk.String("30.00"), Unit: awssdk.String("USD")},
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
			},
			today:            time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
			yesterday:        time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
			wantTodaySpend:   50.0,
			wantYdaySpend:    50.0,
			wantMTDSpend:     100.0,
			wantCurrency:     "USD",
			wantServiceCount: 2,
			wantDailyCount:   2,
		},
		{
			name: "empty results",
			output: &costexplorer.GetCostAndUsageOutput{
				ResultsByTime: []types.ResultByTime{},
			},
			today:            time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
			yesterday:        time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
			wantTodaySpend:   0.0,
			wantYdaySpend:    0.0,
			wantMTDSpend:     0.0,
			wantCurrency:     "",
			wantServiceCount: 0,
			wantDailyCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := aggregateUsage(tt.output, tt.today, tt.yesterday)
			if agg.todaySpend != tt.wantTodaySpend {
				t.Errorf("todaySpend = %f, want %f", agg.todaySpend, tt.wantTodaySpend)
			}
			if agg.yesterdaySpend != tt.wantYdaySpend {
				t.Errorf("yesterdaySpend = %f, want %f", agg.yesterdaySpend, tt.wantYdaySpend)
			}
			if agg.mtdSpend != tt.wantMTDSpend {
				t.Errorf("mtdSpend = %f, want %f", agg.mtdSpend, tt.wantMTDSpend)
			}
			if agg.currency != tt.wantCurrency {
				t.Errorf("currency = %s, want %s", agg.currency, tt.wantCurrency)
			}
			if len(agg.serviceMap) != tt.wantServiceCount {
				t.Errorf("serviceMap count = %d, want %d", len(agg.serviceMap), tt.wantServiceCount)
			}
			if len(agg.dailyMap) != tt.wantDailyCount {
				t.Errorf("dailyMap count = %d, want %d", len(agg.dailyMap), tt.wantDailyCount)
			}
		})
	}
}

func TestExtractMoMSpend(t *testing.T) {
	tests := []struct {
		name              string
		lastMonthRes      usageResult
		mtdSpend          float64
		wantLastMonthSpend float64
		wantMoMPercent    float64
	}{
		{
			name: "normal comparison",
			lastMonthRes: usageResult{
				out: &costexplorer.GetCostAndUsageOutput{
					ResultsByTime: []types.ResultByTime{
						{
							Total: map[string]types.MetricValue{
								"UnblendedCost": {Amount: awssdk.String("200.00"), Unit: awssdk.String("USD")},
							},
						},
					},
				},
			},
			mtdSpend:           300.0,
			wantLastMonthSpend: 200.0,
			wantMoMPercent:     50.0,
		},
		{
			name: "last month error",
			lastMonthRes: usageResult{
				err: fmt.Errorf("api error"),
			},
			mtdSpend:           300.0,
			wantLastMonthSpend: 0.0,
			wantMoMPercent:     0.0,
		},
		{
			name: "last month nil output",
			lastMonthRes: usageResult{
				out: nil,
			},
			mtdSpend:           100.0,
			wantLastMonthSpend: 0.0,
			wantMoMPercent:     0.0,
		},
		{
			name: "last month zero spend",
			lastMonthRes: usageResult{
				out: &costexplorer.GetCostAndUsageOutput{
					ResultsByTime: []types.ResultByTime{
						{
							Total: map[string]types.MetricValue{
								"UnblendedCost": {Amount: awssdk.String("0.00"), Unit: awssdk.String("USD")},
							},
						},
					},
				},
			},
			mtdSpend:           100.0,
			wantLastMonthSpend: 0.0,
			wantMoMPercent:     0.0,
		},
		{
			name: "decrease",
			lastMonthRes: usageResult{
				out: &costexplorer.GetCostAndUsageOutput{
					ResultsByTime: []types.ResultByTime{
						{
							Total: map[string]types.MetricValue{
								"UnblendedCost": {Amount: awssdk.String("100.00"), Unit: awssdk.String("USD")},
							},
						},
					},
				},
			},
			mtdSpend:           80.0,
			wantLastMonthSpend: 100.0,
			wantMoMPercent:     -20.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lastMonthSpend, momPercent := extractMoMSpend(tt.lastMonthRes, tt.mtdSpend)
			if lastMonthSpend != tt.wantLastMonthSpend {
				t.Errorf("lastMonthMTDSpend = %f, want %f", lastMonthSpend, tt.wantLastMonthSpend)
			}
			if momPercent != tt.wantMoMPercent {
				t.Errorf("momChangePercent = %f, want %f", momPercent, tt.wantMoMPercent)
			}
		})
	}
}
