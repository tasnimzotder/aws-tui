package cost

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	awscost "tasnim.dev/aws-tui/internal/aws/cost"
	"tasnim.dev/aws-tui/internal/plugin"
	"tasnim.dev/aws-tui/internal/ui"
)

// costDataMsg carries the result of fetching cost data.
type costDataMsg struct {
	data *awscost.CostData
	err  error
}

var (
	headingStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	greenStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	redStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	yellowStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	sparkStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
)

// ListView displays a cost overview with current month total, forecast, and top services.
type ListView struct {
	client    CostClient
	router    plugin.Router
	table     ui.TableView[awscost.ServiceCost]
	data      *awscost.CostData
	loading   bool
	err       error
	amortized bool
	month     time.Time
}

// NewListView creates a new Cost Explorer ListView.
func NewListView(client CostClient, router plugin.Router) *ListView {
	cols := serviceColumns()
	tv := ui.NewTableView(cols, nil, func(sc awscost.ServiceCost) string {
		return sc.Name
	})
	return &ListView{
		client:  client,
		router:  router,
		table:   tv,
		loading: true,
	}
}

func serviceColumns() []ui.Column[awscost.ServiceCost] {
	return []ui.Column[awscost.ServiceCost]{
		{Title: "Service", Width: 40, Field: func(sc awscost.ServiceCost) string {
			return sc.Name
		}},
		{Title: "Cost", Width: 14, Field: func(sc awscost.ServiceCost) string {
			return fmt.Sprintf("$%.2f", sc.Cost)
		}},
	}
}

func (lv *ListView) fetchCostData() tea.Cmd {
	client := lv.client
	month := lv.month
	return func() tea.Msg {
		var data *awscost.CostData
		var err error
		if month.IsZero() {
			data, err = client.FetchCostData(context.TODO())
		} else {
			data, err = client.FetchCostDataForMonth(context.TODO(), month)
		}
		return costDataMsg{data: data, err: err}
	}
}

func (lv *ListView) currentMonth() time.Time {
	if lv.month.IsZero() {
		now := time.Now()
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	return time.Date(lv.month.Year(), lv.month.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func (lv *ListView) updateTable() {
	if lv.data == nil {
		return
	}
	if lv.amortized && len(lv.data.AmortizedTopServices) > 0 {
		lv.table.SetItems(lv.data.AmortizedTopServices)
	} else {
		lv.table.SetItems(lv.data.TopServices)
	}
}

func (lv *ListView) Init() tea.Cmd {
	return lv.fetchCostData()
}

func (lv *ListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case costDataMsg:
		lv.loading = false
		if msg.err != nil {
			lv.err = msg.err
			return lv, nil
		}
		lv.data = msg.data
		lv.updateTable()
		return lv, nil

	case tea.KeyPressMsg:
		if lv.loading {
			return lv, nil
		}

		switch msg.String() {
		case "enter":
			if lv.data != nil {
				view := NewDetailView(lv.client, lv.router, "")
				lv.router.Push(view)
				return lv, view.Init()
			}
			return lv, nil
		case "esc", "backspace":
			lv.router.Pop()
			return lv, nil
		case "r":
			lv.loading = true
			return lv, lv.fetchCostData()
		case "m":
			lv.amortized = !lv.amortized
			lv.updateTable()
			return lv, nil
		case "<", "h":
			target := lv.currentMonth().AddDate(0, -1, 0)
			sixMonthsAgo := time.Now().AddDate(0, -6, 0)
			earliest := time.Date(sixMonthsAgo.Year(), sixMonthsAgo.Month(), 1, 0, 0, 0, 0, time.UTC)
			if !target.Before(earliest) {
				lv.month = target
				lv.loading = true
				return lv, lv.fetchCostData()
			}
			return lv, nil
		case ">", "l":
			now := time.Now()
			currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
			target := lv.currentMonth().AddDate(0, 1, 0)
			if !target.After(currentMonthStart) {
				lv.month = target
				lv.loading = true
				return lv, lv.fetchCostData()
			} else if !lv.month.IsZero() {
				lv.month = time.Time{}
				lv.loading = true
				return lv, lv.fetchCostData()
			}
			return lv, nil
		}
	}

	var cmd tea.Cmd
	lv.table, cmd = lv.table.Update(msg)
	return lv, cmd
}

func (lv *ListView) View() tea.View {
	if lv.loading {
		skel := ui.NewSkeleton(60, 6)
		return tea.NewView(skel.View())
	}
	if lv.err != nil {
		return tea.NewView("Error: " + lv.err.Error())
	}

	var b strings.Builder

	// Month navigation
	cm := lv.currentMonth()
	monthLabel := cm.Format("January 2006")
	now := time.Now()
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	if cm.Equal(currentMonthStart) {
		monthLabel += " (current)"
	}
	b.WriteString("  ")
	b.WriteString(dimStyle.Render("◀ <"))
	b.WriteString("  ")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")).Render(monthLabel))
	b.WriteString("  ")
	b.WriteString(dimStyle.Render("> ▶"))
	b.WriteString("\n\n")

	// Summary header
	b.WriteString(headingStyle.Render("Cost Overview"))

	// Metric badge
	metricLabel := "Unblended"
	if lv.amortized {
		metricLabel = "Amortized"
	}
	b.WriteString("  ")
	b.WriteString(dimStyle.Render("["))
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render(metricLabel))
	b.WriteString(dimStyle.Render("]"))
	b.WriteString("\n\n")

	mtd := lv.data.MTDSpend
	forecast := lv.data.ForecastSpend
	if lv.amortized {
		mtd = lv.data.AmortizedMTDSpend
		if lv.data.AmortizedForecastSpend > 0 {
			forecast = lv.data.AmortizedForecastSpend
		}
	}

	// Main metrics
	b.WriteString(fmt.Sprintf("  Month-to-Date:   $%.2f", mtd))

	// Sparkline inline
	daily := lv.data.DailySpend
	if lv.amortized && len(lv.data.AmortizedDailySpend) > 0 {
		daily = lv.data.AmortizedDailySpend
	}
	if len(daily) > 1 {
		vals := make([]float64, len(daily))
		for i, d := range daily {
			vals[i] = d.Spend
		}
		b.WriteString("  ")
		b.WriteString(sparkStyle.Render(renderSparkline(vals)))
	}
	b.WriteString("\n")

	if forecast > 0 {
		b.WriteString(fmt.Sprintf("  Forecasted:      $%.2f\n", forecast))
	}
	if lv.data.MoMChangePercent != 0 {
		momStr := fmt.Sprintf("%.1f%%", lv.data.MoMChangePercent)
		style := greenStyle
		arrow := "▼"
		if lv.data.MoMChangePercent > 0 {
			style = redStyle
			arrow = "▲"
		}
		b.WriteString(fmt.Sprintf("  MoM Change:      %s\n", style.Render(arrow+" "+momStr)))
	}
	if lv.data.TodaySpend > 0 {
		b.WriteString(fmt.Sprintf("  Today:           $%.2f\n", lv.data.TodaySpend))
	}

	// Anomaly alerts
	if len(lv.data.Anomalies) > 0 {
		b.WriteString("\n")
		b.WriteString(redStyle.Bold(true).Render("  ⚠ Cost Alerts"))
		b.WriteString("\n")
		limit := 3
		if len(lv.data.Anomalies) < limit {
			limit = len(lv.data.Anomalies)
		}
		for _, a := range lv.data.Anomalies[:limit] {
			b.WriteString(fmt.Sprintf("    %s %.1fx avg: %s ($%.2f)\n",
				redStyle.Render("▲"), a.Ratio, a.ServiceName, a.TodaySpend))
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Top Services"))
	b.WriteString("\n\n")
	b.WriteString(lv.table.View())

	return tea.NewView(b.String())
}

func (lv *ListView) Title() string { return "Cost Explorer" }

func (lv *ListView) KeyHints() []plugin.KeyHint {
	return []plugin.KeyHint{
		{Key: "enter", Desc: "view details"},
		{Key: "r", Desc: "refresh"},
		{Key: "m", Desc: "toggle metric"},
		{Key: "</>", Desc: "prev/next month"},
		{Key: "/", Desc: "filter"},
		{Key: "s", Desc: "sort"},
	}
}
