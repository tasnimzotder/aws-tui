package cost

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	awscost "tasnim.dev/aws-tui/internal/aws/cost"
	"tasnim.dev/aws-tui/internal/plugin"
	"tasnim.dev/aws-tui/internal/ui"
)

// detailCostMsg carries the result of loading cost detail data.
type detailCostMsg struct {
	data *awscost.CostData
	err  error
}

var (
	costTitleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	costDimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	costGreenStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	costRedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	costYellowStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	costBarFillStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	costBarBgStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	costMetricStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	costMonthStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
)

// DetailView shows detailed cost information with tabs.
type DetailView struct {
	client      CostClient
	router      plugin.Router
	serviceID   string
	data        *awscost.CostData
	tabs        ui.TabController
	servicesTbl ui.TableView[awscost.ServiceCost]
	dailyTbl    ui.TableView[awscost.DailySpendEntry]
	regionTbl   ui.TableView[awscost.RegionCost]
	changeTbl   ui.TableView[awscost.ServiceCostChange]
	loading     bool
	err         error
	amortized   bool // metric toggle: false=unblended, true=amortized
	month       time.Time // currently viewed month
}

// NewDetailView creates a DetailView for the given service ID.
func NewDetailView(client CostClient, router plugin.Router, serviceID string) *DetailView {
	return &DetailView{
		client:      client,
		router:      router,
		serviceID:   serviceID,
		tabs:        ui.NewTabController([]string{"Overview", "Services", "Regions", "Daily", "Changes"}),
		servicesTbl: ui.NewTableView(serviceDetailCols(), nil, func(sc awscost.ServiceCost) string { return sc.Name }),
		dailyTbl:    ui.NewTableView(dailyDetailCols(), nil, func(d awscost.DailySpendEntry) string { return d.Date }),
		regionTbl:   ui.NewTableView(regionDetailCols(), nil, func(r awscost.RegionCost) string { return r.Region }),
		changeTbl:   ui.NewTableView(changeCols(), nil, func(c awscost.ServiceCostChange) string { return c.Name }),
		loading:     true,
	}
}

func serviceDetailCols() []ui.Column[awscost.ServiceCost] {
	return []ui.Column[awscost.ServiceCost]{
		{Title: "Service", Width: 40, Field: func(sc awscost.ServiceCost) string { return sc.Name }},
		{Title: "Cost", Width: 14, Field: func(sc awscost.ServiceCost) string { return fmt.Sprintf("$%.2f", sc.Cost) }},
	}
}

func dailyDetailCols() []ui.Column[awscost.DailySpendEntry] {
	return []ui.Column[awscost.DailySpendEntry]{
		{Title: "Date", Width: 14, Field: func(d awscost.DailySpendEntry) string { return d.Date }},
		{Title: "Spend", Width: 14, Field: func(d awscost.DailySpendEntry) string { return fmt.Sprintf("$%.2f", d.Spend) }},
	}
}

func regionDetailCols() []ui.Column[awscost.RegionCost] {
	return []ui.Column[awscost.RegionCost]{
		{Title: "Region", Width: 24, Field: func(r awscost.RegionCost) string { return r.Region }},
		{Title: "Cost", Width: 14, Field: func(r awscost.RegionCost) string { return fmt.Sprintf("$%.2f", r.Cost) }},
	}
}

func changeCols() []ui.Column[awscost.ServiceCostChange] {
	return []ui.Column[awscost.ServiceCostChange]{
		{Title: "Service", Width: 36, Field: func(c awscost.ServiceCostChange) string { return c.Name }},
		{Title: "Current", Width: 12, Field: func(c awscost.ServiceCostChange) string { return fmt.Sprintf("$%.2f", c.CurrentCost) }},
		{Title: "Last Month", Width: 12, Field: func(c awscost.ServiceCostChange) string { return fmt.Sprintf("$%.2f", c.LastMonthCost) }},
		{Title: "Change", Width: 14, Field: func(c awscost.ServiceCostChange) string {
			if c.ChangePercent == 0 && c.ChangeAbsolute == 0 {
				return "—"
			}
			arrow := "▲"
			if c.ChangeAbsolute < 0 {
				arrow = "▼"
			}
			return fmt.Sprintf("%s %.1f%%", arrow, c.ChangePercent)
		}},
	}
}

func (dv *DetailView) loadCostData() tea.Cmd {
	client := dv.client
	month := dv.month
	return func() tea.Msg {
		var data *awscost.CostData
		var err error
		if month.IsZero() {
			data, err = client.FetchCostData(context.TODO())
		} else {
			data, err = client.FetchCostDataForMonth(context.TODO(), month)
		}
		return detailCostMsg{data: data, err: err}
	}
}

func (dv *DetailView) updateTables() {
	if dv.data == nil {
		return
	}
	if dv.amortized {
		dv.servicesTbl.SetItems(dv.data.AmortizedTopServices)
		dv.dailyTbl.SetItems(dv.data.AmortizedDailySpend)
		dv.regionTbl.SetItems(dv.data.AmortizedTopRegions)
	} else {
		dv.servicesTbl.SetItems(dv.data.TopServices)
		dv.dailyTbl.SetItems(dv.data.DailySpend)
		dv.regionTbl.SetItems(dv.data.TopRegions)
	}
	dv.changeTbl.SetItems(dv.data.ServiceChanges)
}

func (dv *DetailView) Init() tea.Cmd {
	return dv.loadCostData()
}

func (dv *DetailView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case detailCostMsg:
		dv.loading = false
		if msg.err != nil {
			dv.err = msg.err
			return dv, nil
		}
		dv.data = msg.data
		dv.updateTables()
		return dv, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "backspace":
			dv.router.Pop()
			return dv, nil
		case "m":
			dv.amortized = !dv.amortized
			dv.updateTables()
			return dv, nil
		case "<", "h":
			// Navigate to previous month (up to 6 months back)
			target := dv.currentMonth().AddDate(0, -1, 0)
			sixMonthsAgo := time.Now().AddDate(0, -6, 0)
			earliest := time.Date(sixMonthsAgo.Year(), sixMonthsAgo.Month(), 1, 0, 0, 0, 0, time.UTC)
			if !target.Before(earliest) {
				dv.month = target
				dv.loading = true
				return dv, dv.loadCostData()
			}
			return dv, nil
		case ">", "l":
			// Navigate to next month (up to current)
			now := time.Now()
			currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
			target := dv.currentMonth().AddDate(0, 1, 0)
			if !target.After(currentMonthStart) {
				dv.month = target
				dv.loading = true
				return dv, dv.loadCostData()
			} else if !dv.month.IsZero() {
				// Go back to current month
				dv.month = time.Time{}
				dv.loading = true
				return dv, dv.loadCostData()
			}
			return dv, nil
		case "r":
			dv.loading = true
			return dv, dv.loadCostData()
		}

		// Delegate to the active tab's table
		switch dv.tabs.Active() {
		case 1:
			var cmd tea.Cmd
			dv.servicesTbl, cmd = dv.servicesTbl.Update(msg)
			if cmd != nil {
				return dv, cmd
			}
		case 2:
			var cmd tea.Cmd
			dv.regionTbl, cmd = dv.regionTbl.Update(msg)
			if cmd != nil {
				return dv, cmd
			}
		case 3:
			var cmd tea.Cmd
			dv.dailyTbl, cmd = dv.dailyTbl.Update(msg)
			if cmd != nil {
				return dv, cmd
			}
		case 4:
			var cmd tea.Cmd
			dv.changeTbl, cmd = dv.changeTbl.Update(msg)
			if cmd != nil {
				return dv, cmd
			}
		}
	}

	var cmd tea.Cmd
	dv.tabs, cmd = dv.tabs.Update(msg)
	return dv, cmd
}

func (dv *DetailView) currentMonth() time.Time {
	if dv.month.IsZero() {
		now := time.Now()
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	return time.Date(dv.month.Year(), dv.month.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func (dv *DetailView) View() tea.View {
	if dv.loading {
		skel := ui.NewSkeleton(60, 8)
		return tea.NewView(skel.View())
	}
	if dv.err != nil {
		return tea.NewView("Error: " + dv.err.Error())
	}

	var b strings.Builder

	// Month navigation bar
	b.WriteString(dv.renderMonthNav())
	b.WriteString("\n")

	// Metric indicator
	metricLabel := "Unblended"
	if dv.amortized {
		metricLabel = "Amortized"
	}
	b.WriteString(costDimStyle.Render("  Metric: "))
	b.WriteString(costMetricStyle.Render(metricLabel))
	b.WriteString(costDimStyle.Render("  (press m to toggle)"))
	b.WriteString("\n\n")

	b.WriteString(dv.tabs.View())
	b.WriteString("\n\n")

	switch dv.tabs.Active() {
	case 0:
		b.WriteString(dv.renderOverview())
	case 1:
		b.WriteString(dv.renderServices())
	case 2:
		b.WriteString(dv.renderRegions())
	case 3:
		b.WriteString(dv.renderDaily())
	case 4:
		b.WriteString(dv.renderChanges())
	}

	return tea.NewView(b.String())
}

func (dv *DetailView) renderMonthNav() string {
	cm := dv.currentMonth()
	label := cm.Format("January 2006")

	now := time.Now()
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	if cm.Equal(currentMonthStart) {
		label += " (current)"
	}

	return "  " + costDimStyle.Render("◀ <") + "  " + costMonthStyle.Render(label) + "  " + costDimStyle.Render("> ▶")
}

func (dv *DetailView) renderOverview() string {
	data := dv.data
	mtd := data.MTDSpend
	forecast := data.ForecastSpend
	if dv.amortized {
		mtd = data.AmortizedMTDSpend
		if data.AmortizedForecastSpend > 0 {
			forecast = data.AmortizedForecastSpend
		}
	}

	var b strings.Builder

	// Summary KV
	rows := []ui.KV{
		{K: "Month-to-Date", V: fmt.Sprintf("$%.2f", mtd)},
	}
	if data.TodaySpend > 0 {
		rows = append(rows, ui.KV{K: "Today", V: fmt.Sprintf("$%.2f", data.TodaySpend)})
	}
	if data.YesterdaySpend > 0 {
		rows = append(rows, ui.KV{K: "Yesterday", V: fmt.Sprintf("$%.2f", data.YesterdaySpend)})
	}
	if forecast > 0 {
		rows = append(rows, ui.KV{K: "Forecasted Total", V: fmt.Sprintf("$%.2f", forecast)})
	}
	if data.LastMonthMTDSpend > 0 {
		rows = append(rows, ui.KV{K: "Last Month (same period)", V: fmt.Sprintf("$%.2f", data.LastMonthMTDSpend)})
	}
	if data.MoMChangePercent != 0 {
		momStr := fmt.Sprintf("%.1f%%", data.MoMChangePercent)
		if data.MoMChangePercent > 0 {
			momStr = "▲ " + momStr
		} else {
			momStr = "▼ " + momStr
		}
		rows = append(rows, ui.KV{K: "MoM Change", V: momStr})
	}

	currency := data.Currency
	if currency == "" {
		currency = "USD"
	}
	rows = append(rows, ui.KV{K: "Currency", V: currency})
	b.WriteString(ui.RenderKV(rows, 26, 0))

	// Budget/Forecast progress bar
	if forecast > 0 && mtd > 0 {
		b.WriteString("\n")
		b.WriteString(costTitleStyle.Render("Forecast Progress"))
		b.WriteString("\n")
		pct := mtd / forecast
		b.WriteString(renderProgressBar(pct, 40))
		b.WriteString(fmt.Sprintf("  %.0f%% of $%.2f forecast\n", pct*100, forecast))
	}

	// Sparkline: daily spend trend
	daily := data.DailySpend
	if dv.amortized && len(data.AmortizedDailySpend) > 0 {
		daily = data.AmortizedDailySpend
	}
	if len(daily) > 1 {
		b.WriteString("\n")
		b.WriteString(costTitleStyle.Render("Daily Spend Trend"))
		b.WriteString("\n")
		vals := make([]float64, len(daily))
		for i, d := range daily {
			vals[i] = d.Spend
		}
		b.WriteString("  ")
		b.WriteString(renderSparkline(vals))
		b.WriteString("  ")
		b.WriteString(costDimStyle.Render(fmt.Sprintf("%s → %s", daily[0].Date[5:], daily[len(daily)-1].Date[5:])))
		b.WriteString("\n")
	}

	// Anomalies
	if len(data.Anomalies) > 0 {
		b.WriteString("\n")
		b.WriteString(costRedStyle.Bold(true).Render("Cost Alerts"))
		b.WriteString("\n")
		for _, a := range data.Anomalies {
			b.WriteString(fmt.Sprintf("  %s $%.2f today (%.1fx avg $%.2f)\n",
				costRedStyle.Render("▲"), a.TodaySpend, a.Ratio, a.AvgSpend))
			b.WriteString(fmt.Sprintf("    %s\n", costDimStyle.Render(a.ServiceName)))
		}
	}

	// Top 5 services mini-bar chart
	services := data.TopServices
	if dv.amortized && len(data.AmortizedTopServices) > 0 {
		services = data.AmortizedTopServices
	}
	if len(services) > 0 {
		b.WriteString("\n")
		b.WriteString(costTitleStyle.Render("Top Services"))
		b.WriteString("\n")
		limit := 5
		if len(services) < limit {
			limit = len(services)
		}
		maxCost := services[0].Cost
		for _, svc := range services[:limit] {
			name := svc.Name
			if len(name) > 30 {
				name = name[:27] + "..."
			}
			barWidth := 0
			if maxCost > 0 {
				barWidth = int((svc.Cost / maxCost) * 20)
			}
			if barWidth < 1 && svc.Cost > 0 {
				barWidth = 1
			}
			bar := costBarFillStyle.Render(strings.Repeat("█", barWidth))
			b.WriteString(fmt.Sprintf("  %-30s %s $%.2f\n", name, bar, svc.Cost))
		}
	}

	return b.String()
}

func (dv *DetailView) renderServices() string {
	return dv.servicesTbl.View()
}

func (dv *DetailView) renderRegions() string {
	return dv.regionTbl.View()
}

func (dv *DetailView) renderDaily() string {
	daily := dv.data.DailySpend
	if dv.amortized && len(dv.data.AmortizedDailySpend) > 0 {
		daily = dv.data.AmortizedDailySpend
	}

	if len(daily) == 0 {
		return "No daily data."
	}

	var b strings.Builder

	// ASCII bar chart
	b.WriteString(costTitleStyle.Render("Daily Cost Chart"))
	b.WriteString("\n\n")

	var maxSpend float64
	for _, d := range daily {
		if d.Spend > maxSpend {
			maxSpend = d.Spend
		}
	}

	barMaxWidth := 40
	for _, d := range daily {
		barWidth := 0
		if maxSpend > 0 {
			barWidth = int((d.Spend / maxSpend) * float64(barMaxWidth))
		}
		if barWidth < 1 && d.Spend > 0 {
			barWidth = 1
		}
		dateLabel := d.Date[5:] // MM-DD
		bar := costBarFillStyle.Render(strings.Repeat("█", barWidth))
		bg := costBarBgStyle.Render(strings.Repeat("░", barMaxWidth-barWidth))
		b.WriteString(fmt.Sprintf("  %s %s%s $%.2f\n", dateLabel, bar, bg, d.Spend))
	}

	b.WriteString("\n")
	b.WriteString(costDimStyle.Render("  Table view:"))
	b.WriteString("\n\n")
	b.WriteString(dv.dailyTbl.View())

	return b.String()
}

func (dv *DetailView) renderChanges() string {
	if len(dv.data.ServiceChanges) == 0 {
		return "No service change data available."
	}

	var b strings.Builder

	// Summary of significant changes
	var increases, decreases int
	for _, c := range dv.data.ServiceChanges {
		if c.ChangePercent > 10 {
			increases++
		} else if c.ChangePercent < -10 {
			decreases++
		}
	}

	if increases > 0 || decreases > 0 {
		b.WriteString(costTitleStyle.Render("Cost Change Summary"))
		b.WriteString("\n")
		if increases > 0 {
			b.WriteString(fmt.Sprintf("  %s %d service(s) with >10%% increase\n",
				costRedStyle.Render("▲"), increases))
		}
		if decreases > 0 {
			b.WriteString(fmt.Sprintf("  %s %d service(s) with >10%% decrease\n",
				costGreenStyle.Render("▼"), decreases))
		}
		b.WriteString("\n")
	}

	b.WriteString(dv.changeTbl.View())

	return b.String()
}

func (dv *DetailView) Title() string {
	return "Cost Details"
}

func (dv *DetailView) KeyHints() []plugin.KeyHint {
	return []plugin.KeyHint{
		{Key: "esc", Desc: "back"},
		{Key: "[/]", Desc: "switch tab"},
		{Key: "m", Desc: "toggle metric"},
		{Key: "</> ", Desc: "prev/next month"},
		{Key: "r", Desc: "refresh"},
	}
}

// renderSparkline renders a Unicode sparkline from a series of values.
func renderSparkline(values []float64) string {
	if len(values) == 0 {
		return ""
	}
	blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	minVal, maxVal := values[0], values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	spread := maxVal - minVal
	if spread == 0 {
		spread = 1
	}

	var b strings.Builder
	for _, v := range values {
		idx := int(((v - minVal) / spread) * float64(len(blocks)-1))
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		b.WriteRune(blocks[idx])
	}
	return b.String()
}

// renderProgressBar renders a progress bar with filled and empty segments.
func renderProgressBar(pct float64, width int) string {
	if pct < 0 {
		pct = 0
	}
	filled := int(math.Round(pct * float64(width)))
	if filled > width {
		filled = width
	}
	empty := width - filled

	bar := costBarFillStyle.Render(strings.Repeat("█", filled))
	bg := costBarBgStyle.Render(strings.Repeat("░", empty))
	return "  " + bar + bg
}
