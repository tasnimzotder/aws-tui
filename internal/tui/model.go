package tui

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/guptarohit/asciigraph"

	awscost "tasnim.dev/aws-tui/internal/aws/cost"
	"tasnim.dev/aws-tui/internal/tui/theme"
	"tasnim.dev/aws-tui/internal/utils"
)

// Messages
type costDataMsg struct{ data *awscost.CostData }
type errMsg struct{ err error }

// Model holds the TUI state.
type Model struct {
	client    *awscost.Client
	profile   string
	accountID string
	data      *awscost.CostData
	err     error
	loading bool
	spinner spinner.Model
	table   table.Model
	width   int
	height  int

	// Month navigation
	selectedMonth time.Time // zero = current month

	// Service drill-down
	drillService string // non-empty = showing service detail
}

// NewModel creates a new TUI model.
func NewModel(client *awscost.Client, profile string, accountID string) Model {
	columns := []table.Column{
		{Title: "Service", Width: 45},
		{Title: "MTD Cost", Width: 20},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(10),
		table.WithWidth(80),
	)
	t.SetStyles(theme.DefaultTableStyles())

	return Model{
		client:    client,
		profile:   profile,
		accountID: accountID,
		loading:   true,
		spinner:   theme.NewSpinner(),
		table:     t,
		width:     80,
		height:    24,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchCost())
}

func (m Model) fetchCost() tea.Cmd {
	target := m.selectedMonth
	return func() tea.Msg {
		var data *awscost.CostData
		var err error
		if target.IsZero() {
			data, err = m.client.FetchCostData(context.Background())
		} else {
			data, err = m.client.FetchCostDataForMonth(context.Background(), target)
		}
		if err != nil {
			return errMsg{err: err}
		}
		return costDataMsg{data: data}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			m.loading = true
			m.err = nil
			return m, tea.Batch(m.spinner.Tick, m.fetchCost())
		case "esc":
			if m.drillService != "" {
				m.drillService = ""
				return m, nil
			}
		case "enter":
			if m.data != nil && m.drillService == "" {
				row := m.table.SelectedRow()
				if len(row) > 0 {
					m.drillService = row[0]
					return m, nil
				}
			}
		case "[":
			// Navigate to previous month (cap at 12 months back)
			now := time.Now()
			minMonth := time.Date(now.Year()-1, now.Month(), 1, 0, 0, 0, 0, time.UTC)
			current := m.selectedMonth
			if current.IsZero() {
				current = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
			}
			prev := current.AddDate(0, -1, 0)
			if !prev.Before(minMonth) {
				m.selectedMonth = prev
				m.drillService = ""
				m.loading = true
				m.err = nil
				return m, tea.Batch(m.spinner.Tick, m.fetchCost())
			}
		case "]":
			// Navigate to next month (cap at current month)
			now := time.Now()
			currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
			current := m.selectedMonth
			if current.IsZero() {
				return m, nil // already at current month
			}
			next := current.AddDate(0, 1, 0)
			if next.After(currentMonthStart) {
				m.selectedMonth = time.Time{} // zero = current
			} else {
				m.selectedMonth = next
			}
			m.drillService = ""
			m.loading = true
			m.err = nil
			return m, tea.Batch(m.spinner.Tick, m.fetchCost())
		}

	case costDataMsg:
		m.data = msg.data
		m.loading = false
		m.table.SetRows(m.buildRows())
		return m, nil

	case errMsg:
		m.err = msg.err
		m.loading = false
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.resizeTable()
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) renderHeader() string {
	profileText := "default"
	if m.profile != "" {
		profileText = m.profile
	}
	headerParts := []string{
		titleStyle.Render("AWS Cost Dashboard"),
		"   ",
	}
	if m.accountID != "" {
		headerParts = append(headerParts,
			metricLabelStyle.Render("account: ")+profileStyle.Render(m.accountID),
			"   ",
		)
	}
	headerParts = append(headerParts,
		metricLabelStyle.Render("profile: ")+profileStyle.Render(profileText),
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, headerParts...)
}

func (m Model) renderMetrics() string {
	monthLabel := m.data.LastUpdated.Format("Jan 2006")
	mtdText := metricLabelStyle.Render("MTD: ") + metricValueStyle.Render(utils.Currency(m.data.MTDSpend, m.data.Currency))
	if m.data.LastMonthMTDSpend > 0 {
		direction := "up"
		momStyle := errorStyle
		if m.data.MoMChangePercent < 0 {
			direction = "down"
			momStyle = metricValueStyle
		}
		mtdText += momStyle.Render(fmt.Sprintf("  vs %s last month (%s %.1f%%)",
			utils.Currency(m.data.LastMonthMTDSpend, m.data.Currency),
			direction,
			math.Abs(m.data.MoMChangePercent),
		))
	}

	metrics := lipgloss.JoinHorizontal(
		lipgloss.Top,
		metricLabelStyle.Render("Today: ")+metricValueStyle.Render(utils.Currency(m.data.TodaySpend, m.data.Currency)),
		"        ",
		metricLabelStyle.Render("Yesterday: ")+metricValueStyle.Render(utils.Currency(m.data.YesterdaySpend, m.data.Currency)),
	)
	metrics += "\n" + mtdText

	forecast := metricLabelStyle.Render("Forecast: ") +
		forecastValueStyle.Render(utils.Currency(m.data.ForecastSpend, m.data.Currency)) +
		metricLabelStyle.Render(fmt.Sprintf("    (%s)", monthLabel))

	return metrics + "\n" + forecast
}

func (m Model) View() tea.View {
	header := m.renderHeader()

	var content string
	if m.loading {
		content = dashboardStyle.Render(
			header + "\n\n" + m.spinner.View() + " Fetching cost data...\n",
		)
	} else if m.err != nil {
		content = dashboardStyle.Render(
			header + "\n\n" + errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) +
				"\n\n" + helpStyle.Render("Press r to retry • q to quit"),
		)
	} else if m.data == nil {
		content = dashboardStyle.Render(header + "\n\nNo data available.\n")
	} else if m.drillService != "" {
		content = dashboardStyle.Render(
			headerStyle.Render(header) + "\n\n" +
				m.renderMonthHeader() +
				m.buildServiceDrillDown() +
				helpStyle.Render("Esc back • [ ] month • q quit"),
		)
	} else {
		content = dashboardStyle.Render(
			headerStyle.Render(header) + "\n\n" +
				m.renderMonthHeader() +
				m.renderMetrics() + "\n" +
				m.buildAnomalyView() +
				m.buildChart() +
				"\n" + metricLabelStyle.Render("Top Services") + "\n" + m.table.View() + "\n" +
				helpStyle.Render("Enter drill down • [ ] month • r refresh • q quit"),
		)
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m Model) buildAnomalyView() string {
	if m.data == nil || len(m.data.Anomalies) == 0 {
		return ""
	}

	view := "\n" + anomalyHeaderStyle.Render("▲ Anomalies") + "\n"
	for _, a := range m.data.Anomalies {
		line := fmt.Sprintf("  %s  %s today vs %s avg (%.1fx)",
			a.ServiceName,
			utils.Currency(a.TodaySpend, m.data.Currency),
			utils.Currency(a.AvgSpend, m.data.Currency),
			a.Ratio,
		)
		view += anomalyStyle.Render(line) + "\n"
	}
	return view
}

func (m Model) resizeTable() Model {
	contentWidth := m.width - 4 // dashboardStyle Padding(1,2)
	costColWidth := 20
	borderWidth := 4
	serviceColWidth := contentWidth - costColWidth - borderWidth
	if serviceColWidth < 20 {
		serviceColWidth = 20
	}

	m.table.SetColumns([]table.Column{
		{Title: "Service", Width: serviceColWidth},
		{Title: "MTD Cost", Width: costColWidth},
	})
	m.table.SetWidth(contentWidth)

	tableHeight := m.height - 19 // header+metrics+chart+help
	if tableHeight < 3 {
		tableHeight = 3
	}
	if tableHeight > 15 {
		tableHeight = 15
	}
	m.table.SetHeight(tableHeight)
	return m
}

func (m Model) buildChart() string {
	if m.data == nil || len(m.data.DailySpend) < 2 {
		return ""
	}

	values := make([]float64, len(m.data.DailySpend))
	for i, entry := range m.data.DailySpend {
		values[i] = entry.Spend
	}

	// Calculate proportional width against table width
	// Table columns: Service(35) + MTD Cost(15) + padding/borders(~4) = ~54
	now := m.data.LastUpdated
	daysInMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	currentDay := now.Day()

	tableWidth := m.width - 4    // content width (matches dashboardStyle padding)
	yAxisWidth := 10             // space for y-axis labels (e.g. "  15.67 ┤")
	chartWidth := ((tableWidth - yAxisWidth) * currentDay) / daysInMonth
	if chartWidth < 10 {
		chartWidth = 10
	}

	chart := asciigraph.Plot(values,
		asciigraph.Height(5),
		asciigraph.Width(chartWidth),
		asciigraph.Caption("Daily Spend"),
		asciigraph.Precision(2),
	)

	return "\n" + metricLabelStyle.Render(chart) + "\n"
}

func (m Model) renderMonthHeader() string {
	var month time.Time
	if m.selectedMonth.IsZero() {
		month = time.Now()
	} else {
		month = m.selectedMonth
	}
	now := time.Now()
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	label := month.Format("January 2006")
	if time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC).Equal(currentMonthStart) {
		label += " (current)"
	}
	return metricLabelStyle.Render("◀ "+label+" ▶") + "\n"
}

func (m Model) buildServiceDrillDown() string {
	if m.data == nil || m.data.ServiceDailyMap == nil {
		return metricLabelStyle.Render("No data for "+m.drillService) + "\n"
	}

	dailyCosts, ok := m.data.ServiceDailyMap[m.drillService]
	if !ok || len(dailyCosts) == 0 {
		return metricLabelStyle.Render("No daily data for "+m.drillService) + "\n"
	}

	// Build sorted daily entries
	type dayEntry struct {
		date  string
		spend float64
	}
	entries := make([]dayEntry, 0, len(dailyCosts))
	var total float64
	for date, spend := range dailyCosts {
		entries = append(entries, dayEntry{date, spend})
		total += spend
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].date < entries[j].date
	})

	// Header
	result := titleStyle.Render(m.drillService) + "\n"
	result += metricLabelStyle.Render("Total: ") + metricValueStyle.Render(utils.Currency(total, m.data.Currency)) + "\n\n"

	// Daily chart
	if len(entries) >= 2 {
		values := make([]float64, len(entries))
		for i, e := range entries {
			values[i] = e.spend
		}
		chartWidth := m.width - 16
		if chartWidth < 10 {
			chartWidth = 10
		}
		chart := asciigraph.Plot(values,
			asciigraph.Height(7),
			asciigraph.Width(chartWidth),
			asciigraph.Caption("Daily Spend — "+m.drillService),
			asciigraph.Precision(2),
		)
		result += metricLabelStyle.Render(chart) + "\n\n"
	}

	return result
}

func (m Model) buildRows() []table.Row {
	if m.data == nil {
		return nil
	}
	rows := make([]table.Row, len(m.data.TopServices))
	for i, svc := range m.data.TopServices {
		rows[i] = table.Row{svc.Name, utils.Currency(svc.Cost, m.data.Currency)}
	}
	return rows
}

