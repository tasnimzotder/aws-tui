package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awsecs "tasnim.dev/aws-tui/internal/aws/ecs"
	awslogs "tasnim.dev/aws-tui/internal/aws/logs"
	"tasnim.dev/aws-tui/internal/tui/theme"
	"tasnim.dev/aws-tui/internal/utils"
)

type taskDetailMsg struct{ detail *awsecs.ECSTaskDetail }
type logEventsMsg struct {
	events       []awslogs.LogEvent
	forwardToken string
}

const (
	tabDetails = 0
	tabLogs    = 1
)

type TaskDetailView struct {
	client      *awsclient.ServiceClient
	clusterName string
	taskARN     string
	detail      *awsecs.ECSTaskDetail
	detailVP    viewport.Model
	spinner     spinner.Model
	loading     bool
	err         error
	ready       bool
	width       int
	height      int

	// Tab state
	activeTab int

	// Logs state
	logsVP       viewport.Model
	logLines     []string
	forwardToken string
	tailing      bool
	logsLoaded   bool
}

func NewTaskDetailView(client *awsclient.ServiceClient, clusterName, taskARN string) *TaskDetailView {
	return &TaskDetailView{
		client:      client,
		clusterName: clusterName,
		taskARN:     taskARN,
		spinner:     theme.NewSpinner(),
		loading:     true,
		width:       80,
		height:      20,
	}
}

func (v *TaskDetailView) Title() string {
	if v.detail != nil {
		return v.detail.TaskID
	}
	return "Task"
}

func (v *TaskDetailView) HelpContext() *HelpContext {
	ctx := HelpContextDetail
	return &ctx
}

func (v *TaskDetailView) Init() tea.Cmd {
	return tea.Batch(v.spinner.Tick, v.fetchData())
}

func (v *TaskDetailView) fetchData() tea.Cmd {
	return func() tea.Msg {
		detail, err := v.client.ECS.DescribeTask(context.Background(), v.clusterName, v.taskARN)
		if err != nil {
			return errViewMsg{err: err}
		}
		return taskDetailMsg{detail: detail}
	}
}

func (v *TaskDetailView) fetchLogs() tea.Cmd {
	logGroup, logStream := v.firstContainerLogInfo()
	if logGroup == "" || logStream == "" {
		return nil
	}
	return func() tea.Msg {
		events, token, err := v.client.Logs.GetLatestLogEvents(context.Background(), logGroup, logStream, 100)
		if err != nil {
			return errViewMsg{err: err}
		}
		return logEventsMsg{events: events, forwardToken: token}
	}
}

func (v *TaskDetailView) pollLogs() tea.Cmd {
	logGroup, logStream := v.firstContainerLogInfo()
	if logGroup == "" || logStream == "" {
		return nil
	}
	token := v.forwardToken
	return func() tea.Msg {
		events, newToken, err := v.client.Logs.GetLogEventsSince(context.Background(), logGroup, logStream, token)
		if err != nil {
			return errViewMsg{err: err}
		}
		return logEventsMsg{events: events, forwardToken: newToken}
	}
}

func (v *TaskDetailView) firstContainerLogInfo() (string, string) {
	if v.detail == nil {
		return "", ""
	}
	for _, c := range v.detail.Containers {
		if c.LogGroup != "" && c.LogStream != "" {
			return c.LogGroup, c.LogStream
		}
	}
	return "", ""
}

func (v *TaskDetailView) tickTail() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return tailTickMsg{}
	})
}

type tailTickMsg struct{}

func (v *TaskDetailView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case taskDetailMsg:
		v.detail = msg.detail
		v.loading = false
		v.initViewports()
		return v, nil

	case logEventsMsg:
		v.forwardToken = msg.forwardToken
		for _, e := range msg.events {
			line := fmt.Sprintf("%s  %s",
				theme.MutedStyle.Render(e.Timestamp.Format("15:04:05")),
				strings.TrimRight(e.Message, "\n"),
			)
			v.logLines = append(v.logLines, line)
		}
		v.logsLoaded = true
		v.logsVP.SetContent(strings.Join(v.logLines, "\n"))
		if v.tailing {
			v.logsVP.GotoBottom()
		}
		return v, nil

	case tailTickMsg:
		if v.tailing && v.forwardToken != "" {
			return v, tea.Batch(v.pollLogs(), v.tickTail())
		}
		return v, nil

	case errViewMsg:
		v.err = msg.err
		v.loading = false
		return v, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "r":
			v.loading = true
			v.err = nil
			return v, tea.Batch(v.spinner.Tick, v.fetchData())
		case "tab":
			v.activeTab = (v.activeTab + 1) % 2
			if v.activeTab == tabLogs && !v.logsLoaded {
				return v, v.fetchLogs()
			}
			return v, nil
		case "1":
			v.activeTab = tabDetails
			return v, nil
		case "2":
			v.activeTab = tabLogs
			if !v.logsLoaded {
				return v, v.fetchLogs()
			}
			return v, nil
		case "t":
			if v.activeTab == tabLogs {
				v.tailing = !v.tailing
				if v.tailing {
					if !v.logsLoaded {
						return v, tea.Batch(v.fetchLogs(), v.tickTail())
					}
					return v, v.tickTail()
				}
			}
			return v, nil
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		return v, cmd
	}

	if v.ready {
		var cmd tea.Cmd
		if v.activeTab == tabDetails {
			v.detailVP, cmd = v.detailVP.Update(msg)
		} else {
			v.logsVP, cmd = v.logsVP.Update(msg)
		}
		return v, cmd
	}

	return v, nil
}

func (v *TaskDetailView) View() string {
	if v.loading {
		return v.spinner.View() + " Loading task details..."
	}
	if v.err != nil {
		return theme.ErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	if !v.ready {
		return ""
	}

	// Tab bar
	tabs := []string{"Details", "Logs"}
	var renderedTabs []string
	for i, tab := range tabs {
		label := fmt.Sprintf("%d:%s", i+1, tab)
		if i == v.activeTab {
			renderedTabs = append(renderedTabs, theme.TabActiveStyle.Render(label))
		} else {
			renderedTabs = append(renderedTabs, theme.TabInactiveStyle.Render(label))
		}
	}
	tabBar := theme.TabBarStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...))

	// Status indicators for logs tab
	var statusLine string
	if v.activeTab == tabLogs {
		if v.tailing {
			statusLine = theme.SuccessStyle.Render("● tailing") + theme.MutedStyle.Render("  t to stop")
		} else {
			statusLine = theme.MutedStyle.Render("○ paused") + theme.MutedStyle.Render("  t to tail")
		}
		statusLine = "\n" + statusLine
	}

	// Content
	var content string
	if v.activeTab == tabDetails {
		content = v.detailVP.View()
	} else {
		if !v.logsLoaded {
			content = v.spinner.View() + " Loading logs..."
		} else if len(v.logLines) == 0 {
			content = theme.MutedStyle.Render("No log events found")
		} else {
			content = v.logsVP.View()
		}
	}

	return tabBar + "\n" + content + statusLine
}

func (v *TaskDetailView) initViewports() {
	vpHeight := v.height - 4 // account for tab bar + status line
	if vpHeight < 3 {
		vpHeight = 3
	}

	v.detailVP = viewport.New(viewport.WithWidth(v.width), viewport.WithHeight(vpHeight))
	v.detailVP.SetContent(v.renderDetail())

	v.logsVP = viewport.New(viewport.WithWidth(v.width), viewport.WithHeight(vpHeight))
	v.ready = true
}

func (v *TaskDetailView) renderDetail() string {
	d := v.detail
	db := utils.NewDetailBuilder(16, theme.MutedStyle)

	db.Row("Task ARN", d.TaskARN)
	db.Row("Status", d.Status)
	db.Row("Task Def", d.TaskDef)

	if started := utils.TimeOrDash(d.StartedAt, utils.DateTimeSec); started != "—" {
		db.Row("Started", started)
	}
	if stopped := utils.TimeOrDash(d.StoppedAt, utils.DateTimeSec); stopped != "—" {
		db.Row("Stopped", stopped)
	}
	if d.StopCode != "" {
		db.Row("Stop Code", d.StopCode)
	}
	if d.StopReason != "" {
		db.Row("Stop Reason", d.StopReason)
	}

	db.Row("CPU", d.CPU)
	db.Row("Memory", d.Memory)

	if d.NetworkMode != "" {
		db.Row("Network", d.NetworkMode)
	}
	if d.PrivateIP != "" {
		db.Row("Private IP", d.PrivateIP)
	}
	if d.SubnetID != "" {
		db.Row("Subnet", d.SubnetID)
	}

	// Containers
	if len(d.Containers) > 0 {
		db.Blank()
		db.Section("Containers")
		for _, c := range d.Containers {
			db.WriteString(fmt.Sprintf("\n  %s\n", lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("[%s]", c.Name))))
			db.Row("  Image", c.Image)
			db.Row("  Status", c.Status)
			if c.ExitCode != nil {
				db.Row("  Exit Code", fmt.Sprintf("%d", *c.ExitCode))
			}
			if c.HealthStatus != "" {
				db.Row("  Health", c.HealthStatus)
			}
			if c.CPU > 0 {
				db.Row("  CPU", fmt.Sprintf("%d", c.CPU))
			}
			if c.Memory > 0 {
				db.Row("  Memory", fmt.Sprintf("%d", c.Memory))
			}
			if c.LogGroup != "" {
				db.Row("  Log Group", c.LogGroup)
			}
			if c.LogStream != "" {
				db.Row("  Log Stream", c.LogStream)
			}
			if len(c.Environment) > 0 {
				db.WriteString(theme.MutedStyle.Render("    ── Environment ──") + "\n")
				for _, env := range c.Environment {
					db.Row("    "+env.Name, env.Value)
				}
			}
		}
	}

	return db.String()
}

// CopyableView implementation
func (v *TaskDetailView) CopyID() string {
	if v.detail != nil {
		return v.detail.TaskID
	}
	return ""
}

func (v *TaskDetailView) CopyARN() string {
	if v.detail != nil {
		return v.detail.TaskARN
	}
	return ""
}

// ResizableView implementation
func (v *TaskDetailView) SetSize(width, height int) {
	v.width = width
	v.height = height
	if v.ready {
		vpHeight := height - 4
		if vpHeight < 3 {
			vpHeight = 3
		}
		v.detailVP.SetWidth(width)
		v.detailVP.SetHeight(vpHeight)
		v.logsVP.SetWidth(width)
		v.logsVP.SetHeight(vpHeight)
	}
}
