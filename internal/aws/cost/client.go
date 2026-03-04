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
func aggregateUsage(out *costexplorer.GetCostAndUsageOutput, today, yesterday time.Time, metric string) usageAggregation {
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
			amount, _ := strconv.ParseFloat(aws.ToString(group.Metrics[metric].Amount), 64)
			unit := aws.ToString(group.Metrics[metric].Unit)
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

// aggregateByRegion aggregates cost data grouped by region.
func aggregateByRegion(out *costexplorer.GetCostAndUsageOutput, metric string) []RegionCost {
	regionMap := make(map[string]float64)
	for _, result := range out.ResultsByTime {
		for _, group := range result.Groups {
			region := group.Keys[0]
			amount, _ := strconv.ParseFloat(aws.ToString(group.Metrics[metric].Amount), 64)
			regionMap[region] += amount
		}
	}

	regions := make([]RegionCost, 0, len(regionMap))
	for region, cost := range regionMap {
		regions = append(regions, RegionCost{Region: region, Cost: cost})
	}
	sort.Slice(regions, func(i, j int) bool {
		return regions[i].Cost > regions[j].Cost
	})
	if len(regions) > 15 {
		regions = regions[:15]
	}
	return regions
}

// extractMoMSpend computes last-month MTD spend and month-over-month change percentage.
func extractMoMSpend(lastMonthRes usageResult, mtdSpend float64, metric string) (lastMonthMTDSpend, momChangePercent float64) {
	if lastMonthRes.err != nil || lastMonthRes.out == nil {
		return 0, 0
	}
	for _, result := range lastMonthRes.out.ResultsByTime {
		if mv, ok := result.Total[metric]; ok {
			amount, _ := strconv.ParseFloat(aws.ToString(mv.Amount), 64)
			lastMonthMTDSpend += amount
		}
	}
	if lastMonthMTDSpend > 0 {
		momChangePercent = ((mtdSpend - lastMonthMTDSpend) / lastMonthMTDSpend) * 100
	}
	return lastMonthMTDSpend, momChangePercent
}

// computeServiceChanges computes cost changes vs previous month for each service.
func computeServiceChanges(lastMonthRes usageResult, currentServices []ServiceCost, metric string) []ServiceCostChange {
	lastMonthMap := make(map[string]float64)
	if lastMonthRes.err == nil && lastMonthRes.out != nil {
		for _, result := range lastMonthRes.out.ResultsByTime {
			for _, group := range result.Groups {
				svcName := group.Keys[0]
				amount, _ := strconv.ParseFloat(aws.ToString(group.Metrics[metric].Amount), 64)
				lastMonthMap[svcName] += amount
			}
		}
	}

	var changes []ServiceCostChange
	for _, svc := range currentServices {
		lastCost := lastMonthMap[svc.Name]
		change := ServiceCostChange{
			Name:           svc.Name,
			CurrentCost:    svc.Cost,
			LastMonthCost:  lastCost,
			ChangeAbsolute: svc.Cost - lastCost,
		}
		if lastCost > 0 {
			change.ChangePercent = ((svc.Cost - lastCost) / lastCost) * 100
		}
		changes = append(changes, change)
	}
	return changes
}

func buildSortedServices(serviceMap map[string]float64, limit int) []ServiceCost {
	services := make([]ServiceCost, 0, len(serviceMap))
	for name, cost := range serviceMap {
		services = append(services, ServiceCost{Name: name, Cost: cost})
	}
	sort.Slice(services, func(i, j int) bool {
		return services[i].Cost > services[j].Cost
	})
	if limit > 0 && len(services) > limit {
		services = services[:limit]
	}
	return services
}

func buildSortedDaily(dailyMap map[string]float64) []DailySpendEntry {
	daily := make([]DailySpendEntry, 0, len(dailyMap))
	for date, spend := range dailyMap {
		daily = append(daily, DailySpendEntry{Date: date, Spend: spend})
	}
	sort.Slice(daily, func(i, j int) bool {
		return daily[i].Date < daily[j].Date
	})
	return daily
}

// FetchCostData retrieves cost and forecast data from AWS Cost Explorer.
func (c *Client) FetchCostData(ctx context.Context) (*CostData, error) {
	dr := computeDateRange(c.now())

	// Launch API calls concurrently: service usage, amortized usage, region usage,
	// forecast (unblended + amortized), last month MTD, last month service breakdown.
	usageCh := make(chan usageResult, 1)
	amortizedCh := make(chan usageResult, 1)
	regionCh := make(chan usageResult, 1)
	amortizedRegionCh := make(chan usageResult, 1)
	forecastCh := make(chan forecastResult, 1)
	amortizedForecastCh := make(chan forecastResult, 1)
	lastMonthCh := make(chan usageResult, 1)
	lastMonthServiceCh := make(chan usageResult, 1)

	timePeriod := &types.DateInterval{
		Start: aws.String(dr.monthStart.Format("2006-01-02")),
		End:   aws.String(dr.tomorrow.Format("2006-01-02")),
	}
	serviceGroupBy := []types.GroupDefinition{
		{Type: types.GroupDefinitionTypeDimension, Key: aws.String("SERVICE")},
	}
	regionGroupBy := []types.GroupDefinition{
		{Type: types.GroupDefinitionTypeDimension, Key: aws.String("REGION")},
	}

	// Unblended service usage
	go func() {
		out, err := c.ce.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
			TimePeriod:  timePeriod,
			Granularity: types.GranularityDaily,
			Metrics:     []string{"UnblendedCost"},
			GroupBy:     serviceGroupBy,
		})
		usageCh <- usageResult{out, err}
	}()

	// Amortized service usage
	go func() {
		out, err := c.ce.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
			TimePeriod:  timePeriod,
			Granularity: types.GranularityDaily,
			Metrics:     []string{"AmortizedCost"},
			GroupBy:     serviceGroupBy,
		})
		amortizedCh <- usageResult{out, err}
	}()

	// Region usage (unblended)
	go func() {
		out, err := c.ce.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
			TimePeriod:  timePeriod,
			Granularity: types.GranularityDaily,
			Metrics:     []string{"UnblendedCost"},
			GroupBy:     regionGroupBy,
		})
		regionCh <- usageResult{out, err}
	}()

	// Region usage (amortized)
	go func() {
		out, err := c.ce.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
			TimePeriod:  timePeriod,
			Granularity: types.GranularityDaily,
			Metrics:     []string{"AmortizedCost"},
			GroupBy:     regionGroupBy,
		})
		amortizedRegionCh <- usageResult{out, err}
	}()

	// Forecasts
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
		if !dr.tomorrow.Before(dr.monthEnd) {
			amortizedForecastCh <- forecastResult{}
			return
		}
		out, err := c.ce.GetCostForecast(ctx, &costexplorer.GetCostForecastInput{
			TimePeriod: &types.DateInterval{
				Start: aws.String(dr.tomorrow.Format("2006-01-02")),
				End:   aws.String(dr.monthEnd.Format("2006-01-02")),
			},
			Granularity: types.GranularityMonthly,
			Metric:      types.MetricAmortizedCost,
		})
		amortizedForecastCh <- forecastResult{out, err}
	}()

	// Last month MTD (for MoM comparison)
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

	// Last month service breakdown (for change tracking)
	go func() {
		lastMonthEnd := dr.lastMonthStart.AddDate(0, 1, 0)
		out, err := c.ce.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
			TimePeriod: &types.DateInterval{
				Start: aws.String(dr.lastMonthStart.Format("2006-01-02")),
				End:   aws.String(lastMonthEnd.Format("2006-01-02")),
			},
			Granularity: types.GranularityMonthly,
			Metrics:     []string{"UnblendedCost"},
			GroupBy:     serviceGroupBy,
		})
		lastMonthServiceCh <- usageResult{out, err}
	}()

	// Collect results
	usageRes := <-usageCh
	if usageRes.err != nil {
		return nil, fmt.Errorf("GetCostAndUsage: %w", usageRes.err)
	}

	forecastRes := <-forecastCh
	amortizedRes := <-amortizedCh
	regionRes := <-regionCh
	amortizedRegionRes := <-amortizedRegionCh
	amortizedForecastRes := <-amortizedForecastCh
	lastMonthRes := <-lastMonthCh
	lastMonthServiceRes := <-lastMonthServiceCh

	// Aggregate usage data
	yesterday := dr.today.AddDate(0, 0, -1)
	agg := aggregateUsage(usageRes.out, dr.today, yesterday, "UnblendedCost")

	services := buildSortedServices(agg.serviceMap, 10)
	dailySpend := buildSortedDaily(agg.dailyMap)

	// Detect anomalies and extract forecast
	todayStr := dr.today.Format("2006-01-02")
	anomalies := detectAnomalies(agg.serviceDailyMap, todayStr)

	var forecastSpend float64
	if forecastRes.err == nil && forecastRes.out != nil && forecastRes.out.Total != nil {
		forecastSpend, _ = strconv.ParseFloat(aws.ToString(forecastRes.out.Total.Amount), 64)
	}

	lastMonthMTDSpend, momChangePercent := extractMoMSpend(lastMonthRes, agg.mtdSpend, "UnblendedCost")
	serviceChanges := computeServiceChanges(lastMonthServiceRes, services, "UnblendedCost")

	// Amortized aggregation
	var amortizedServices []ServiceCost
	var amortizedDaily []DailySpendEntry
	var amortizedMTD, amortizedForecast float64
	if amortizedRes.err == nil && amortizedRes.out != nil {
		amortizedAgg := aggregateUsage(amortizedRes.out, dr.today, yesterday, "AmortizedCost")
		amortizedServices = buildSortedServices(amortizedAgg.serviceMap, 10)
		amortizedDaily = buildSortedDaily(amortizedAgg.dailyMap)
		amortizedMTD = amortizedAgg.mtdSpend
	}
	if amortizedForecastRes.err == nil && amortizedForecastRes.out != nil && amortizedForecastRes.out.Total != nil {
		amortizedForecast, _ = strconv.ParseFloat(aws.ToString(amortizedForecastRes.out.Total.Amount), 64)
	}

	// Region aggregation
	var regions, amortizedRegions []RegionCost
	if regionRes.err == nil && regionRes.out != nil {
		regions = aggregateByRegion(regionRes.out, "UnblendedCost")
	}
	if amortizedRegionRes.err == nil && amortizedRegionRes.out != nil {
		amortizedRegions = aggregateByRegion(amortizedRegionRes.out, "AmortizedCost")
	}

	return &CostData{
		TodaySpend:             agg.todaySpend,
		YesterdaySpend:         agg.yesterdaySpend,
		MTDSpend:               agg.mtdSpend,
		ForecastSpend:          forecastSpend,
		Currency:               agg.currency,
		TopServices:            services,
		DailySpend:             dailySpend,
		ServiceDailyMap:        agg.serviceDailyMap,
		Anomalies:              anomalies,
		LastMonthMTDSpend:      lastMonthMTDSpend,
		MoMChangePercent:       momChangePercent,
		LastUpdated:            dr.now,
		AmortizedMTDSpend:      amortizedMTD,
		AmortizedForecastSpend: amortizedForecast,
		AmortizedTopServices:   amortizedServices,
		AmortizedDailySpend:    amortizedDaily,
		TopRegions:             regions,
		AmortizedTopRegions:    amortizedRegions,
		ServiceChanges:         serviceChanges,
	}, nil
}

// FetchCostDataForMonth retrieves cost data for a specific past month.
func (c *Client) FetchCostDataForMonth(ctx context.Context, target time.Time) (*CostData, error) {
	monthStart := time.Date(target.Year(), target.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)

	now := c.now()
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	if monthStart.Equal(currentMonthStart) {
		data, err := c.FetchCostData(ctx)
		if err != nil {
			return nil, err
		}
		data.TargetMonth = target
		return data, nil
	}

	// Past month: fetch full month usage concurrently
	usageCh := make(chan usageResult, 1)
	amortizedCh := make(chan usageResult, 1)
	regionCh := make(chan usageResult, 1)
	amortizedRegionCh := make(chan usageResult, 1)
	prevMonthCh := make(chan usageResult, 1)
	prevMonthServiceCh := make(chan usageResult, 1)

	timePeriod := &types.DateInterval{
		Start: aws.String(monthStart.Format("2006-01-02")),
		End:   aws.String(monthEnd.Format("2006-01-02")),
	}
	serviceGroupBy := []types.GroupDefinition{
		{Type: types.GroupDefinitionTypeDimension, Key: aws.String("SERVICE")},
	}
	regionGroupBy := []types.GroupDefinition{
		{Type: types.GroupDefinitionTypeDimension, Key: aws.String("REGION")},
	}

	go func() {
		out, err := c.ce.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
			TimePeriod:  timePeriod,
			Granularity: types.GranularityDaily,
			Metrics:     []string{"UnblendedCost"},
			GroupBy:     serviceGroupBy,
		})
		usageCh <- usageResult{out, err}
	}()

	go func() {
		out, err := c.ce.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
			TimePeriod:  timePeriod,
			Granularity: types.GranularityDaily,
			Metrics:     []string{"AmortizedCost"},
			GroupBy:     serviceGroupBy,
		})
		amortizedCh <- usageResult{out, err}
	}()

	go func() {
		out, err := c.ce.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
			TimePeriod:  timePeriod,
			Granularity: types.GranularityDaily,
			Metrics:     []string{"UnblendedCost"},
			GroupBy:     regionGroupBy,
		})
		regionCh <- usageResult{out, err}
	}()

	go func() {
		out, err := c.ce.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
			TimePeriod:  timePeriod,
			Granularity: types.GranularityDaily,
			Metrics:     []string{"AmortizedCost"},
			GroupBy:     regionGroupBy,
		})
		amortizedRegionCh <- usageResult{out, err}
	}()

	prevMonthStart := monthStart.AddDate(0, -1, 0)
	prevMonthEnd := monthStart
	go func() {
		out, err := c.ce.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
			TimePeriod: &types.DateInterval{
				Start: aws.String(prevMonthStart.Format("2006-01-02")),
				End:   aws.String(prevMonthEnd.Format("2006-01-02")),
			},
			Granularity: types.GranularityMonthly,
			Metrics:     []string{"UnblendedCost"},
		})
		prevMonthCh <- usageResult{out, err}
	}()

	go func() {
		out, err := c.ce.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
			TimePeriod: &types.DateInterval{
				Start: aws.String(prevMonthStart.Format("2006-01-02")),
				End:   aws.String(prevMonthEnd.Format("2006-01-02")),
			},
			Granularity: types.GranularityMonthly,
			Metrics:     []string{"UnblendedCost"},
			GroupBy:     serviceGroupBy,
		})
		prevMonthServiceCh <- usageResult{out, err}
	}()

	usageRes := <-usageCh
	if usageRes.err != nil {
		return nil, fmt.Errorf("GetCostAndUsage: %w", usageRes.err)
	}

	amortizedRes := <-amortizedCh
	regionRes := <-regionCh
	amortizedRegionRes := <-amortizedRegionCh

	lastDay := monthEnd.AddDate(0, 0, -1)
	agg := aggregateUsage(usageRes.out, lastDay, lastDay.AddDate(0, 0, -1), "UnblendedCost")

	services := buildSortedServices(agg.serviceMap, 10)
	dailySpend := buildSortedDaily(agg.dailyMap)

	lastMonthMTDSpend, momChangePercent := extractMoMSpend(<-prevMonthCh, agg.mtdSpend, "UnblendedCost")
	prevMonthServiceRes := <-prevMonthServiceCh
	serviceChanges := computeServiceChanges(prevMonthServiceRes, services, "UnblendedCost")

	// Amortized
	var amortizedServices []ServiceCost
	var amortizedDaily []DailySpendEntry
	var amortizedMTD float64
	if amortizedRes.err == nil && amortizedRes.out != nil {
		amortizedAgg := aggregateUsage(amortizedRes.out, lastDay, lastDay.AddDate(0, 0, -1), "AmortizedCost")
		amortizedServices = buildSortedServices(amortizedAgg.serviceMap, 10)
		amortizedDaily = buildSortedDaily(amortizedAgg.dailyMap)
		amortizedMTD = amortizedAgg.mtdSpend
	}

	var regions, amortizedRegions []RegionCost
	if regionRes.err == nil && regionRes.out != nil {
		regions = aggregateByRegion(regionRes.out, "UnblendedCost")
	}
	if amortizedRegionRes.err == nil && amortizedRegionRes.out != nil {
		amortizedRegions = aggregateByRegion(amortizedRegionRes.out, "AmortizedCost")
	}

	return &CostData{
		MTDSpend:               agg.mtdSpend,
		Currency:               agg.currency,
		TopServices:            services,
		DailySpend:             dailySpend,
		ServiceDailyMap:        agg.serviceDailyMap,
		LastMonthMTDSpend:      lastMonthMTDSpend,
		MoMChangePercent:       momChangePercent,
		LastUpdated:            now,
		TargetMonth:            target,
		AmortizedMTDSpend:      amortizedMTD,
		AmortizedTopServices:   amortizedServices,
		AmortizedDailySpend:    amortizedDaily,
		TopRegions:             regions,
		AmortizedTopRegions:    amortizedRegions,
		ServiceChanges:         serviceChanges,
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
