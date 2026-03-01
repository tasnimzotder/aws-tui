package services

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"golang.org/x/term"

	awsecs "tasnim.dev/aws-tui/internal/aws/ecs"
	"tasnim.dev/aws-tui/internal/tui/theme"
)

// ---------------------------------------------------------------------------
// ECS Exec Process — implements tea.ExecCommand
// ---------------------------------------------------------------------------

type ecsExecProcess struct {
	cluster   string
	taskARN   string
	container string
	command   string
	profile   string
	region    string
	stdin     io.Reader
	stdout    io.Writer
	stderr    io.Writer
}

func (p *ecsExecProcess) SetStdin(r io.Reader)  { p.stdin = r }
func (p *ecsExecProcess) SetStdout(w io.Writer) { p.stdout = w }
func (p *ecsExecProcess) SetStderr(w io.Writer) { p.stderr = w }

func (p *ecsExecProcess) Run() error {
	if _, err := exec.LookPath("session-manager-plugin"); err != nil {
		return fmt.Errorf("session-manager-plugin not found in PATH — install it: https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html")
	}

	p.printHeader()

	args := []string{"ecs", "execute-command",
		"--cluster", p.cluster,
		"--task", p.taskARN,
		"--container", p.container,
		"--interactive",
		"--command", p.command,
	}
	if p.profile != "" {
		args = append(args, "--profile", p.profile)
	}
	if p.region != "" {
		args = append(args, "--region", p.region)
	}

	cmd := exec.Command("aws", args...)
	cmd.Stdin = p.stdin
	cmd.Stdout = p.stdout
	cmd.Stderr = p.stderr

	return cmd.Run()
}

func (p *ecsExecProcess) printHeader() {
	w, _, _ := term.GetSize(int(os.Stdout.Fd()))
	if w <= 0 {
		w = 80
	}

	// Extract short task ID from ARN
	taskID := p.taskARN
	if parts := strings.Split(p.taskARN, "/"); len(parts) > 0 {
		taskID = parts[len(parts)-1]
	}

	label := " ECS Exec "
	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#1a1a2e")).
		Background(lipgloss.Color("#FF9F43"))
	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF9F43"))
	divStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))

	info := fmt.Sprintf(" %s/%s [%s]  cmd: %s", p.cluster, taskID, p.container, p.command)
	header := labelStyle.Render(label)
	headerInfo := infoStyle.Render(info)
	divider := divStyle.Render(strings.Repeat("─", w))

	fmt.Fprintf(p.stdout, "%s%s\n%s\n", header, headerInfo, divider)
	fmt.Fprintf(p.stdout, "\033]0;ecs-exec %s/%s\007", p.cluster, taskID)
}

// ---------------------------------------------------------------------------
// ECS Exec Input View — prompts for command before exec
// ---------------------------------------------------------------------------

type ecsExecInputView struct {
	cluster   string
	taskARN   string
	container string
	profile   string
	region    string
	input     textinput.Model
}

func newECSExecInputView(cluster, taskARN, container, profile, region string) *ecsExecInputView {
	ti := textinput.New()
	ti.SetValue("/bin/sh")
	ti.CharLimit = 256
	ti.Focus()

	return &ecsExecInputView{
		cluster:   cluster,
		taskARN:   taskARN,
		container: container,
		profile:   profile,
		region:    region,
		input:     ti,
	}
}

func (v *ecsExecInputView) Title() string {
	taskID := v.taskARN
	if parts := strings.Split(v.taskARN, "/"); len(parts) > 0 {
		taskID = parts[len(parts)-1]
	}
	return fmt.Sprintf("ECS Exec: %s", taskID)
}

func (v *ecsExecInputView) Init() tea.Cmd { return textinput.Blink }

func (v *ecsExecInputView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			cmd := strings.TrimSpace(v.input.Value())
			if cmd == "" {
				return v, nil
			}
			process := &ecsExecProcess{
				cluster:   v.cluster,
				taskARN:   v.taskARN,
				container: v.container,
				command:   cmd,
				profile:   v.profile,
				region:    v.region,
			}
			return v, tea.Exec(process, func(err error) tea.Msg {
				return execDoneMsg{err: err}
			})
		case "esc":
			return v, func() tea.Msg { return PopViewMsg{} }
		}
	case tea.WindowSizeMsg:
		return v, nil
	}

	var cmd tea.Cmd
	v.input, cmd = v.input.Update(msg)
	return v, cmd
}

func (v *ecsExecInputView) View() string {
	var b strings.Builder
	bold := lipgloss.NewStyle().Bold(true)

	taskID := v.taskARN
	if parts := strings.Split(v.taskARN, "/"); len(parts) > 0 {
		taskID = parts[len(parts)-1]
	}

	b.WriteString(bold.Render("ECS Exec"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Cluster:   %s\n", v.cluster))
	b.WriteString(fmt.Sprintf("Task:      %s\n", taskID))
	b.WriteString(fmt.Sprintf("Container: %s\n", v.container))
	b.WriteString("\nCommand:\n")
	b.WriteString(v.input.View())
	b.WriteString("\n\n")
	b.WriteString(theme.MutedStyle.Render("Enter to exec  Esc to cancel"))
	return b.String()
}

func (v *ecsExecInputView) SetSize(width, height int) {}
func (v *ecsExecInputView) CapturesInput() bool       { return true }

// ---------------------------------------------------------------------------
// ECS Container Picker — shown when a multi-container task needs selection
// ---------------------------------------------------------------------------

type ecsContainerItem struct {
	name string
}

func (c ecsContainerItem) Title() string       { return c.name }
func (c ecsContainerItem) Description() string { return "" }
func (c ecsContainerItem) FilterValue() string { return c.name }

type ecsContainerPickerView struct {
	cluster    string
	taskARN    string
	containers []awsecs.ECSContainerDetail
	profile    string
	region     string
	list       list.Model
}

func newECSContainerPickerView(cluster, taskARN, profile, region string, containers []awsecs.ECSContainerDetail) *ecsContainerPickerView {
	items := make([]list.Item, len(containers))
	for i, c := range containers {
		items[i] = ecsContainerItem{name: c.Name}
	}

	l := list.New(items, list.NewDefaultDelegate(), 40, 14)
	l.SetShowTitle(true)
	l.Title = "Select container (exec)"
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	return &ecsContainerPickerView{
		cluster:    cluster,
		taskARN:    taskARN,
		containers: containers,
		profile:    profile,
		region:     region,
		list:       l,
	}
}

func (v *ecsContainerPickerView) Title() string {
	return "Container (exec)"
}

func (v *ecsContainerPickerView) Init() tea.Cmd { return nil }

func (v *ecsContainerPickerView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			selected, ok := v.list.SelectedItem().(ecsContainerItem)
			if !ok {
				return v, nil
			}
			return v, pushView(newECSExecInputView(v.cluster, v.taskARN, selected.name, v.profile, v.region))
		case "esc":
			return v, func() tea.Msg { return PopViewMsg{} }
		}
	case tea.WindowSizeMsg:
		v.list.SetSize(msg.Width, msg.Height)
		return v, nil
	}

	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}

func (v *ecsContainerPickerView) View() string {
	return v.list.View()
}

func (v *ecsContainerPickerView) SetSize(width, height int) {
	v.list.SetSize(width, height)
}
