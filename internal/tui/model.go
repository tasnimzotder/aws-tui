package tui

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	return func() tea.Msg {
		data, err := m.client.FetchCostData(context.Background())
		if err != nil {
			return errMsg{err: err}
		}
		return costDataMsg{data: data}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
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

func (m Model) View() string {
	header := m.renderHeader()

	if m.loading {
		return dashboardStyle.Render(
			header + "\n\n" + m.spinner.View() + " Fetching cost data...\n",
		)
	}

	if m.err != nil {
		return dashboardStyle.Render(
			header + "\n\n" + errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) +
				"\n\n" + helpStyle.Render("Press r to retry • q to quit"),
		)
	}

	if m.data == nil {
		return dashboardStyle.Render(header + "\n\nNo data available.\n")
	}

	return dashboardStyle.Render(
		headerStyle.Render(header) + "\n\n" +
			m.renderMetrics() + "\n" +
			m.buildAnomalyView() +
			m.buildChart() +
			"\n" + metricLabelStyle.Render("Top Services") + "\n" + m.table.View() + "\n" +
			helpStyle.Render("Press q to quit • r to refresh"),
	)
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

