package services

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	corev1 "k8s.io/api/core/v1"

	awseks "tasnim.dev/aws-tui/internal/aws/eks"
	"tasnim.dev/aws-tui/internal/tui/theme"
)

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

type logLineMsg struct {
	line string
}

type logDoneMsg struct{}

type logErrorMsg struct {
	err error
}

// ---------------------------------------------------------------------------
// EKSLogView — streams pod logs into a scrollable viewport
// ---------------------------------------------------------------------------

// EKSLogView streams pod container logs in a viewport.
type EKSLogView struct {
	k8s       *awseks.K8sClient
	pod       K8sPod
	container string

	viewport  viewport.Model
	buffer    strings.Builder
	lineCount int
	ready     bool
	done      bool
	err       error
	cancel    context.CancelFunc
	logCh     chan string // channel for streaming lines from goroutine

	width, height int
}

// NewEKSLogView creates a log viewer for the given pod container.
// If container is empty and the pod has exactly one container, it uses that.
func NewEKSLogView(k8s *awseks.K8sClient, pod K8sPod, container string) *EKSLogView {
	if container == "" && len(pod.Containers) == 1 {
		container = pod.Containers[0]
	}
	return &EKSLogView{
		k8s:       k8s,
		pod:       pod,
		container: container,
		logCh:     make(chan string, 100),
		width:     80,
		height:    24,
	}
}

func (v *EKSLogView) Title() string {
	return fmt.Sprintf("Logs: %s/%s", v.pod.Name, v.container)
}

func (v *EKSLogView) Init() tea.Cmd {
	h := v.height - 4
	if h < 1 {
		h = 1
	}
	vp := viewport.New(
		viewport.WithWidth(v.width),
		viewport.WithHeight(h),
	)
	vp.MouseWheelEnabled = true
	vp.SoftWrap = true
	vp.Style = lipgloss.NewStyle().Padding(0, 1)
	v.viewport = vp
	v.ready = true

	return tea.Batch(v.startLogStream(), v.waitForLogLine())
}

// startLogStream launches a background goroutine that reads log lines and
// sends them to the logCh channel. It returns a Cmd that completes when
// the stream ends.
func (v *EKSLogView) startLogStream() tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	v.cancel = cancel

	k8s := v.k8s
	pod := v.pod
	container := v.container
	ch := v.logCh

	return func() tea.Msg {
		defer close(ch)

		tailLines := int64(1000)
		req := k8s.Clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
			Container: container,
			Follow:    true,
			TailLines: &tailLines,
		})
		stream, err := req.Stream(ctx)
		if err != nil {
			return logErrorMsg{err: err}
		}
		defer stream.Close()

		scanner := bufio.NewScanner(stream)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return logDoneMsg{}
			case ch <- scanner.Text():
			}
		}
		if err := scanner.Err(); err != nil {
			if ctx.Err() != nil {
				return logDoneMsg{}
			}
			return logErrorMsg{err: err}
		}
		return logDoneMsg{}
	}
}

// waitForLogLine returns a Cmd that reads one line from the channel and
// returns it as a logLineMsg. If the channel is closed, returns logDoneMsg.
func (v *EKSLogView) waitForLogLine() tea.Cmd {
	ch := v.logCh
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return logDoneMsg{}
		}
		return logLineMsg{line: line}
	}
}

func (v *EKSLogView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case logLineMsg:
		v.lineCount++
		if v.buffer.Len() > 0 {
			v.buffer.WriteByte('\n')
		}
		v.buffer.WriteString(msg.line)
		if v.ready {
			v.viewport.SetContent(v.buffer.String())
			v.viewport.GotoBottom()
		}
		// Wait for next line
		return v, v.waitForLogLine()

	case logDoneMsg:
		v.done = true
		return v, nil

	case logErrorMsg:
		v.err = msg.err
		v.done = true
		return v, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			if v.cancel != nil {
				v.cancel()
			}
			return v, func() tea.Msg { return PopViewMsg{} }
		}

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		if v.ready {
			v.viewport.SetWidth(v.width)
			h := v.height - 4
			if h < 1 {
				h = 1
			}
			v.viewport.SetHeight(h)
		}
		return v, nil
	}

	if v.ready {
		var cmd tea.Cmd
		v.viewport, cmd = v.viewport.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v *EKSLogView) View() string {
	if v.err != nil {
		return theme.ErrorStyle.Render(fmt.Sprintf("Error streaming logs: %v", v.err)) +
			"\n\n" + theme.MutedStyle.Render("Press Esc to go back")
	}
	if !v.ready {
		return theme.MutedStyle.Render("Initializing log stream...")
	}

	// Header
	status := "streaming..."
	if v.done {
		status = "ended"
	}
	header := fmt.Sprintf("Logs: %s (container: %s) — %s", v.pod.Name, v.container, status)
	headerLine := lipgloss.NewStyle().Bold(true).Render(header)
	separator := theme.MutedStyle.Render(strings.Repeat("─", v.width))
	helpLine := theme.MutedStyle.Render("Esc back  ↑/↓ scroll")

	return headerLine + "\n" + separator + "\n" + v.viewport.View() + "\n" + helpLine
}

func (v *EKSLogView) SetSize(width, height int) {
	v.width = width
	v.height = height
	if v.ready {
		v.viewport.SetWidth(width)
		h := height - 4
		if h < 1 {
			h = 1
		}
		v.viewport.SetHeight(h)
	}
}

// ---------------------------------------------------------------------------
// Container Picker — shown when a multi-container pod needs selection
// ---------------------------------------------------------------------------

type containerItem struct {
	name string
}

func (c containerItem) Title() string       { return c.name }
func (c containerItem) Description() string { return "" }
func (c containerItem) FilterValue() string { return c.name }

// ContainerPickerView lets the user choose a container from a multi-container pod.
type ContainerPickerView struct {
	k8s    *awseks.K8sClient
	pod    K8sPod
	list   list.Model
	action string // "logs", "exec", or "portforward"
}

// NewContainerPickerView creates a container picker for the given pod and action.
func NewContainerPickerView(k8s *awseks.K8sClient, pod K8sPod, action string) *ContainerPickerView {
	items := make([]list.Item, len(pod.Containers))
	for i, c := range pod.Containers {
		items[i] = containerItem{name: c}
	}

	l := list.New(items, list.NewDefaultDelegate(), 40, 14)
	l.SetShowTitle(true)
	l.Title = fmt.Sprintf("Select container (%s)", action)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	return &ContainerPickerView{
		k8s:    k8s,
		pod:    pod,
		list:   l,
		action: action,
	}
}

func (v *ContainerPickerView) Title() string {
	return fmt.Sprintf("Container (%s): %s", v.action, v.pod.Name)
}

func (v *ContainerPickerView) Init() tea.Cmd { return nil }

func (v *ContainerPickerView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			selected, ok := v.list.SelectedItem().(containerItem)
			if !ok {
				return v, nil
			}
			container := selected.name
			switch v.action {
			case "logs":
				return v, pushView(NewEKSLogView(v.k8s, v.pod, container))
			case "exec":
				return v, pushView(newExecInputView(v.k8s, v.pod, container))
			}
			return v, nil
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

func (v *ContainerPickerView) View() string {
	return v.list.View()
}

func (v *ContainerPickerView) SetSize(width, height int) {
	v.list.SetSize(width, height)
}
