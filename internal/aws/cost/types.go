package cost

import "time"

// CostData holds all cost information for the dashboard.
type CostData struct {
	TodaySpend     float64
	YesterdaySpend float64
	MTDSpend       float64
	ForecastSpend  float64
	Currency       string
	TopServices    []ServiceCost
	DailySpend     []DailySpendEntry
	ServiceDailyMap    map[string]map[string]float64 // service -> date -> spend
	Anomalies          []ServiceAnomaly
	LastMonthMTDSpend  float64
	MoMChangePercent   float64
	LastUpdated        time.Time
	TargetMonth        time.Time // month being displayed (zero = current)

	// Amortized cost fields (populated when metric toggle is active).
	AmortizedMTDSpend      float64
	AmortizedForecastSpend float64
	AmortizedTopServices   []ServiceCost
	AmortizedDailySpend    []DailySpendEntry

	// Region breakdown.
	TopRegions          []RegionCost
	AmortizedTopRegions []RegionCost

	// Service cost changes vs last month.
	ServiceChanges []ServiceCostChange
}

// ServiceCost represents cost for a single AWS service.
type ServiceCost struct {
	Name string
	Cost float64
}

// RegionCost represents cost for a single AWS region.
type RegionCost struct {
	Region string
	Cost   float64
}

// DailySpendEntry represents total spend for a single day.
type DailySpendEntry struct {
	Date  string
	Spend float64
}

// ServiceAnomaly represents a service with abnormally high spend today.
type ServiceAnomaly struct {
	ServiceName string
	TodaySpend  float64
	AvgSpend    float64
	Ratio       float64
}

// ServiceCostChange tracks cost change for a service vs last month.
type ServiceCostChange struct {
	Name           string
	CurrentCost    float64
	LastMonthCost  float64
	ChangePercent  float64
	ChangeAbsolute float64
}
