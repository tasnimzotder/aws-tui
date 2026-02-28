package cost

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

// CostExplorerAPI is the subset of the AWS Cost Explorer client we use.
type CostExplorerAPI interface {
	GetCostAndUsage(ctx context.Context, params *costexplorer.GetCostAndUsageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error)
	GetCostForecast(ctx context.Context, params *costexplorer.GetCostForecastInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostForecastOutput, error)
}

// Client wraps the AWS Cost Explorer API.
type Client struct {
	ce  CostExplorerAPI
	now func() time.Time // injectable for testing; defaults to time.Now
}

// NewClient creates a new Cost Explorer client from an AWS config.
func NewClient(cfg aws.Config) *Client {
	return &Client{ce: costexplorer.NewFromConfig(cfg), now: time.Now}
}

// NewClientWithAPI creates a client with a custom API implementation (for testing).
func NewClientWithAPI(api CostExplorerAPI) *Client {
	return &Client{ce: api, now: time.Now}
}

type usageResult struct {
	out *costexplorer.GetCostAndUsageOutput
	err error
}

type forecastResult struct {
	out *costexplorer.GetCostForecastOutput
	err error
}

// dateRange holds computed date boundaries for cost queries.
type dateRange struct {
	now              time.Time
	monthStart       time.Time
	today            time.Time
	tomorrow         time.Time
	monthEnd         time.Time
	lastMonthStart   time.Time
	lastMonthSameDay time.Time
}

// computeDateRange calculates all date boundaries needed for cost queries.
func computeDateRange(now time.Time) dateRange {
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)
	monthEnd := monthStart.AddDate(0, 1, 0)

	lastMonthStart := monthStart.AddDate(0, -1, 0)
	lastMonthEnd := monthStart
	lastMonthLastDay := lastMonthEnd.AddDate(0, 0, -1).Day()
	sameDay := now.Day()
	if sameDay > lastMonthLastDay {
		sameDay = lastMonthLastDay
	}
	lastMonthSameDay := time.Date(lastMonthStart.Year(), lastMonthStart.Month(), sameDay+1, 0, 0, 0, 0, time.UTC)
	if lastMonthSameDay.After(lastMonthEnd) {
		lastMonthSameDay = lastMonthEnd
	}

	return dateRange{
		now:              now,
		monthStart:       monthStart,
		today:            today,
		tomorrow:         tomorrow,
		monthEnd:         monthEnd,
		lastMonthStart:   lastMonthStart,
		lastMonthSameDay: lastMonthSameDay,
	}
}

// usageAggregation holds aggregated results from cost usage data.
type usageAggregation struct {
	serviceMap      map[string]float64
	dailyMap        map[string]float64
	serviceDailyMap map[string]map[string]float64
	todaySpend      float64
	yesterdaySpend  float64
	mtdSpend        float64
	currency        string
}

// aggregateUsage processes API response into per-service and daily aggregations.
func aggregateUsage(out *costexplorer.GetCostAndUsageOutput, today, yesterday time.Time) usageAggregation {
	agg := usageAggregation{
		serviceMap:      make(map[string]float64),
		dailyMap:        make(map[string]float64),
		serviceDailyMap: make(map[string]map[string]float64),
	}

	todayStr := today.Format("2006-01-02")
	yesterdayStr := yesterday.Format("2006-01-02")

	for _, result := range out.ResultsByTime {
		dateStr := aws.ToString(result.TimePeriod.Start)
		for _, group := range result.Groups {
			svcName := group.Keys[0]
			amount, _ := strconv.ParseFloat(aws.ToString(group.Metrics["UnblendedCost"].Amount), 64)
			unit := aws.ToString(group.Metrics["UnblendedCost"].Unit)
			if agg.currency == "" {
				agg.currency = unit
			}
			agg.serviceMap[svcName] += amount
			agg.dailyMap[dateStr] += amount
			if agg.serviceDailyMap[svcName] == nil {
				agg.serviceDailyMap[svcName] = make(map[string]float64)
			}
			agg.serviceDailyMap[svcName][dateStr] += amount
			agg.mtdSpend += amount
			if dateStr == todayStr {
				agg.todaySpend += amount
			}
			if dateStr == yesterdayStr {
				agg.yesterdaySpend += amount
			}
		}
	}

	return agg
}

// extractMoMSpend computes last-month MTD spend and month-over-month change percentage.
func extractMoMSpend(lastMonthRes usageResult, mtdSpend float64) (lastMonthMTDSpend, momChangePercent float64) {
	if lastMonthRes.err != nil || lastMonthRes.out == nil {
		return 0, 0
	}
	for _, result := range lastMonthRes.out.ResultsByTime {
		if mv, ok := result.Total["UnblendedCost"]; ok {
			amount, _ := strconv.ParseFloat(aws.ToString(mv.Amount), 64)
			lastMonthMTDSpend += amount
		}
	}
	if lastMonthMTDSpend > 0 {
		momChangePercent = ((mtdSpend - lastMonthMTDSpend) / lastMonthMTDSpend) * 100
	}
	return lastMonthMTDSpend, momChangePercent
}

// FetchCostData retrieves cost and forecast data from AWS Cost Explorer.
func (c *Client) FetchCostData(ctx context.Context) (*CostData, error) {
	dr := computeDateRange(c.now())

	// Launch API calls concurrently
	usageCh := make(chan usageResult, 1)
	forecastCh := make(chan forecastResult, 1)
	lastMonthCh := make(chan usageResult, 1)

	go func() {
		out, err := c.ce.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
			TimePeriod: &types.DateInterval{
				Start: aws.String(dr.monthStart.Format("2006-01-02")),
				End:   aws.String(dr.tomorrow.Format("2006-01-02")),
			},
			Granularity: types.GranularityDaily,
			Metrics:     []string{"UnblendedCost"},
			GroupBy: []types.GroupDefinition{
				{Type: types.GroupDefinitionTypeDimension, Key: aws.String("SERVICE")},
			},
		})
		usageCh <- usageResult{out, err}
	}()

	go func() {
		if !dr.tomorrow.Before(dr.monthEnd) {
			forecastCh <- forecastResult{}
			return
		}
		out, err := c.ce.GetCostForecast(ctx, &costexplorer.GetCostForecastInput{
			TimePeriod: &types.DateInterval{
				Start: aws.String(dr.tomorrow.Format("2006-01-02")),
				End:   aws.String(dr.monthEnd.Format("2006-01-02")),
			},
			Granularity: types.GranularityMonthly,
			Metric:      types.MetricUnblendedCost,
		})
		forecastCh <- forecastResult{out, err}
	}()

	go func() {
		out, err := c.ce.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
			TimePeriod: &types.DateInterval{
				Start: aws.String(dr.lastMonthStart.Format("2006-01-02")),
				End:   aws.String(dr.lastMonthSameDay.Format("2006-01-02")),
			},
			Granularity: types.GranularityMonthly,
			Metrics:     []string{"UnblendedCost"},
		})
		lastMonthCh <- usageResult{out, err}
	}()

	usageRes := <-usageCh
	if usageRes.err != nil {
		return nil, fmt.Errorf("GetCostAndUsage: %w", usageRes.err)
	}

	forecastRes := <-forecastCh
	if forecastRes.err != nil {
		return nil, fmt.Errorf("GetCostForecast: %w", forecastRes.err)
	}

	// Aggregate usage data
	yesterday := dr.today.AddDate(0, 0, -1)
	agg := aggregateUsage(usageRes.out, dr.today, yesterday)

	// Build sorted daily spend entries
	dailySpend := make([]DailySpendEntry, 0, len(agg.dailyMap))
	for date, spend := range agg.dailyMap {
		dailySpend = append(dailySpend, DailySpendEntry{Date: date, Spend: spend})
	}
	sort.Slice(dailySpend, func(i, j int) bool {
		return dailySpend[i].Date < dailySpend[j].Date
	})

	// Sort services by cost descending, cap at top 10
	services := make([]ServiceCost, 0, len(agg.serviceMap))
	for name, cost := range agg.serviceMap {
		services = append(services, ServiceCost{Name: name, Cost: cost})
	}
	sort.Slice(services, func(i, j int) bool {
		return services[i].Cost > services[j].Cost
	})
	if len(services) > 10 {
		services = services[:10]
	}

	// Detect anomalies and extract forecast
	todayStr := dr.today.Format("2006-01-02")
	anomalies := detectAnomalies(agg.serviceDailyMap, todayStr)

	var forecastSpend float64
	if forecastRes.out != nil && forecastRes.out.Total != nil {
		forecastSpend, _ = strconv.ParseFloat(aws.ToString(forecastRes.out.Total.Amount), 64)
	}

	// Extract MoM comparison
	lastMonthMTDSpend, momChangePercent := extractMoMSpend(<-lastMonthCh, agg.mtdSpend)

	return &CostData{
		TodaySpend:        agg.todaySpend,
		YesterdaySpend:    agg.yesterdaySpend,
		MTDSpend:          agg.mtdSpend,
		ForecastSpend:     forecastSpend,
		Currency:          agg.currency,
		TopServices:       services,
		DailySpend:        dailySpend,
		Anomalies:         anomalies,
		LastMonthMTDSpend: lastMonthMTDSpend,
		MoMChangePercent:  momChangePercent,
		LastUpdated:       dr.now,
	}, nil
}

// detectAnomalies compares each service's today spend against its trailing 7-day average.
// Flags services where ratio >= 2.0, today >= $1, avg >= $0.50, and >= 3 days history.
func detectAnomalies(serviceDailyMap map[string]map[string]float64, today string) []ServiceAnomaly {
	var anomalies []ServiceAnomaly

	for svc, dailyCosts := range serviceDailyMap {
		todaySpend, hasToday := dailyCosts[today]
		if !hasToday || todaySpend < 1.0 {
			continue
		}

		// Collect trailing days (excluding today)
		var trailingSum float64
		var trailingDays int
		for date, spend := range dailyCosts {
			if date != today {
				trailingSum += spend
				trailingDays++
			}
		}

		if trailingDays < 3 {
			continue
		}

		avg := trailingSum / float64(trailingDays)
		if avg < 0.50 {
			continue
		}

		ratio := todaySpend / avg
		if ratio >= 2.0 {
			anomalies = append(anomalies, ServiceAnomaly{
				ServiceName: svc,
				TodaySpend:  todaySpend,
				AvgSpend:    avg,
				Ratio:       ratio,
			})
		}
	}

	sort.Slice(anomalies, func(i, j int) bool {
		return anomalies[i].Ratio > anomalies[j].Ratio
	})

	return anomalies
}
