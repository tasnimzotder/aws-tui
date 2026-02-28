package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/transport/spdy"

	awseks "tasnim.dev/aws-tui/internal/aws/eks"
	"tasnim.dev/aws-tui/internal/tui/theme"
)

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

type execDoneMsg struct {
	err error
}

type portForwardStartedMsg struct {
	pf  *PortForward
	err error
}

type portForwardStoppedMsg struct {
	index int
}

// ---------------------------------------------------------------------------
// Pod Exec — suspends TUI via tea.Exec
// ---------------------------------------------------------------------------

// k8sExecProcess implements tea.ExecCommand to run kubectl exec via SPDY.
type k8sExecProcess struct {
	k8s       *awseks.K8sClient
	pod       K8sPod
	container string
	command   []string
	stdin     io.Reader
	stdout    io.Writer
	stderr    io.Writer
}

func (p *k8sExecProcess) SetStdin(r io.Reader)  { p.stdin = r }
func (p *k8sExecProcess) SetStdout(w io.Writer) { p.stdout = w }
func (p *k8sExecProcess) SetStderr(w io.Writer) { p.stderr = w }

func (p *k8sExecProcess) Run() error {
	config := p.k8s.Config

	// Put local terminal in raw mode so escape sequences (arrow keys, tab,
	// ctrl-*) pass through to the remote shell instead of being interpreted.
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("setting terminal raw mode: %w", err)
	}
	defer term.Restore(fd, oldState)

	req := p.k8s.Clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(p.pod.Name).
		Namespace(p.pod.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: p.container,
			Command:   p.command,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("creating SPDY executor: %w", err)
	}

	// Set up terminal size queue with SIGWINCH handling for live resizes.
	sizeQueue := newTerminalSizeQueue(fd)
	defer sizeQueue.stop()

	return exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdin:             p.stdin,
		Stdout:            p.stdout,
		Stderr:            p.stderr,
		Tty:               true,
		TerminalSizeQueue: sizeQueue,
	})
}

// terminalSizeQueue sends the current terminal size, then listens for
// SIGWINCH signals to send updated sizes on window resize.
type terminalSizeQueue struct {
	fd      int
	sizeCh  chan remotecommand.TerminalSize
	sigCh   chan os.Signal
	stopCh  chan struct{}
}

func newTerminalSizeQueue(fd int) *terminalSizeQueue {
	q := &terminalSizeQueue{
		fd:     fd,
		sizeCh: make(chan remotecommand.TerminalSize, 1),
		sigCh:  make(chan os.Signal, 1),
		stopCh: make(chan struct{}),
	}

	// Send initial size.
	if w, h, err := term.GetSize(fd); err == nil {
		q.sizeCh <- remotecommand.TerminalSize{Width: uint16(w), Height: uint16(h)}
	}

	// Listen for SIGWINCH (terminal resize).
	signal.Notify(q.sigCh, unix.SIGWINCH)
	go func() {
		for {
			select {
			case <-q.sigCh:
				if w, h, err := term.GetSize(fd); err == nil {
					q.sizeCh <- remotecommand.TerminalSize{Width: uint16(w), Height: uint16(h)}
				}
			case <-q.stopCh:
				return
			}
		}
	}()

	return q
}

func (q *terminalSizeQueue) Next() *remotecommand.TerminalSize {
	select {
	case size := <-q.sizeCh:
		return &size
	case <-q.stopCh:
		return nil
	}
}

func (q *terminalSizeQueue) stop() {
	signal.Stop(q.sigCh)
	close(q.stopCh)
}

// execIntoPod returns a tea.Cmd that suspends the TUI and runs an interactive
// shell in the specified pod container with the given command.
func execIntoPod(k8s *awseks.K8sClient, pod K8sPod, container string, command []string) tea.Cmd {
	if container == "" && len(pod.Containers) == 1 {
		container = pod.Containers[0]
	}
	process := &k8sExecProcess{
		k8s:       k8s,
		pod:       pod,
		container: container,
		command:   command,
	}
	return tea.Exec(process, func(err error) tea.Msg {
		return execDoneMsg{err: err}
	})
}

// ---------------------------------------------------------------------------
// Exec Input View — prompts for command before exec
// ---------------------------------------------------------------------------

type execInputView struct {
	k8s       *awseks.K8sClient
	pod       K8sPod
	container string
	input     textinput.Model
}

func newExecInputView(k8s *awseks.K8sClient, pod K8sPod, container string) *execInputView {
	ti := textinput.New()
	ti.SetValue("/bin/sh")
	ti.CharLimit = 256
	ti.Focus()

	if container == "" && len(pod.Containers) == 1 {
		container = pod.Containers[0]
	}

	return &execInputView{
		k8s:       k8s,
		pod:       pod,
		container: container,
		input:     ti,
	}
}

func (v *execInputView) Title() string {
	return fmt.Sprintf("Exec: %s", v.pod.Name)
}

func (v *execInputView) Init() tea.Cmd { return textinput.Blink }

func (v *execInputView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			cmd := strings.TrimSpace(v.input.Value())
			if cmd == "" {
				return v, nil
			}
			return v, execIntoPod(v.k8s, v.pod, v.container, parseCommand(cmd))
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

func (v *execInputView) View() string {
	var b strings.Builder
	bold := lipgloss.NewStyle().Bold(true)

	b.WriteString(bold.Render(fmt.Sprintf("Exec into: %s/%s", v.pod.Namespace, v.pod.Name)))
	if v.container != "" {
		b.WriteString(bold.Render(fmt.Sprintf(" [%s]", v.container)))
	}
	b.WriteString("\n\n")
	b.WriteString("Command:\n")
	b.WriteString(v.input.View())
	b.WriteString("\n\n")
	b.WriteString(theme.MutedStyle.Render("Enter to exec  Esc to cancel"))
	return b.String()
}

func (v *execInputView) SetSize(width, height int) {}
func (v *execInputView) CapturesInput() bool        { return true }

// parseCommand splits a command string into args, respecting quoted strings.
func parseCommand(input string) []string {
	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(input); i++ {
		c := input[i]
		switch {
		case inQuote:
			if c == quoteChar {
				inQuote = false
			} else {
				current.WriteByte(c)
			}
		case c == '"' || c == '\'':
			inQuote = true
			quoteChar = c
		case c == ' ' || c == '\t':
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

// ---------------------------------------------------------------------------
// Port Forward — runs in background, TUI stays active
// ---------------------------------------------------------------------------

// PortForward represents an active port-forward session.
type PortForward struct {
	LocalPort  int
	RemotePort int
	PodName    string
	Namespace  string
	StopChan   chan struct{}
	ReadyChan  chan struct{}
	ErrChan    chan error
}

func (pf *PortForward) String() string {
	return fmt.Sprintf("%s/%s  localhost:%d -> :%d", pf.Namespace, pf.PodName, pf.LocalPort, pf.RemotePort)
}

// portForwardManager tracks active port-forward sessions.
type portForwardManager struct {
	forwards []*PortForward
	mu       sync.Mutex
}

func newPortForwardManager() *portForwardManager {
	return &portForwardManager{}
}

func (m *portForwardManager) Add(pf *PortForward) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.forwards = append(m.forwards, pf)
}

func (m *portForwardManager) Remove(index int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if index < 0 || index >= len(m.forwards) {
		return
	}
	pf := m.forwards[index]
	close(pf.StopChan)
	m.forwards = append(m.forwards[:index], m.forwards[index+1:]...)
}

func (m *portForwardManager) List() []*PortForward {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*PortForward, len(m.forwards))
	copy(result, m.forwards)
	return result
}

func (m *portForwardManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, pf := range m.forwards {
		close(pf.StopChan)
	}
	m.forwards = nil
}

// startPortForward creates and starts a port-forward in a background goroutine.
func startPortForward(k8s *awseks.K8sClient, pod K8sPod, localPort, remotePort int, pfManager *portForwardManager) tea.Cmd {
	return func() tea.Msg {
		config := k8s.Config

		req := k8s.Clientset.CoreV1().RESTClient().Post().
			Resource("pods").
			Name(pod.Name).
			Namespace(pod.Namespace).
			SubResource("portforward")

		transport, upgrader, err := spdy.RoundTripperFor(config)
		if err != nil {
			return portForwardStartedMsg{err: fmt.Errorf("creating SPDY transport: %w", err)}
		}

		dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())

		stopChan := make(chan struct{})
		readyChan := make(chan struct{})
		errChan := make(chan error, 1)

		portSpec := fmt.Sprintf("%d:%d", localPort, remotePort)
		fw, err := portforward.New(dialer, []string{portSpec}, stopChan, readyChan, io.Discard, io.Discard)
		if err != nil {
			return portForwardStartedMsg{err: fmt.Errorf("creating port forwarder: %w", err)}
		}

		pf := &PortForward{
			LocalPort:  localPort,
			RemotePort: remotePort,
			PodName:    pod.Name,
			Namespace:  pod.Namespace,
			StopChan:   stopChan,
			ReadyChan:  readyChan,
			ErrChan:    errChan,
		}

		// Run port forward in background
		go func() {
			if err := fw.ForwardPorts(); err != nil {
				errChan <- err
			}
		}()

		// Wait for ready or error
		select {
		case <-readyChan:
			pfManager.Add(pf)
			return portForwardStartedMsg{pf: pf}
		case err := <-errChan:
			return portForwardStartedMsg{err: fmt.Errorf("port forward failed: %w", err)}
		}
	}
}

// ---------------------------------------------------------------------------
// Port Forward Input View — prompts for localPort:remotePort
// ---------------------------------------------------------------------------

type portForwardInputView struct {
	k8s       *awseks.K8sClient
	pod       K8sPod
	input     textinput.Model
	pfManager *portForwardManager
	err       error
}

func newPortForwardInputView(k8s *awseks.K8sClient, pod K8sPod, pfManager *portForwardManager) *portForwardInputView {
	ti := textinput.New()
	ti.Placeholder = "8080:80"
	ti.CharLimit = 11
	ti.Focus()

	return &portForwardInputView{
		k8s:       k8s,
		pod:       pod,
		input:     ti,
		pfManager: pfManager,
	}
}

func (v *portForwardInputView) Title() string {
	return fmt.Sprintf("Port Forward: %s", v.pod.Name)
}

func (v *portForwardInputView) Init() tea.Cmd {
	return textinput.Blink
}

func (v *portForwardInputView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case portForwardStartedMsg:
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}
		// Success — pop back to the table view
		return v, func() tea.Msg { return PopViewMsg{} }

	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			local, remote, err := parsePortSpec(v.input.Value())
			if err != nil {
				v.err = err
				return v, nil
			}
			return v, startPortForward(v.k8s, v.pod, local, remote, v.pfManager)
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

func (v *portForwardInputView) View() string {
	var b strings.Builder
	bold := lipgloss.NewStyle().Bold(true)

	b.WriteString(bold.Render(fmt.Sprintf("Port Forward: %s/%s", v.pod.Namespace, v.pod.Name)))
	b.WriteString("\n\n")
	b.WriteString("Enter port mapping (localPort:remotePort):\n")
	b.WriteString(v.input.View())
	b.WriteString("\n\n")

	if v.err != nil {
		b.WriteString(theme.ErrorStyle.Render(fmt.Sprintf("Error: %v", v.err)))
		b.WriteString("\n\n")
	}

	b.WriteString(theme.MutedStyle.Render("Enter to start  Esc to cancel"))
	return b.String()
}

func (v *portForwardInputView) SetSize(width, height int) {}
func (v *portForwardInputView) CapturesInput() bool        { return true }

// parsePortSpec parses "localPort:remotePort" and returns the two port numbers.
func parsePortSpec(spec string) (int, int, error) {
	parts := strings.SplitN(spec, ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected format: localPort:remotePort (e.g., 8080:80)")
	}
	local, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || local < 1 || local > 65535 {
		return 0, 0, fmt.Errorf("invalid local port: %s", parts[0])
	}
	remote, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || remote < 1 || remote > 65535 {
		return 0, 0, fmt.Errorf("invalid remote port: %s", parts[1])
	}
	return local, remote, nil
}

// ---------------------------------------------------------------------------
// Port Forward List View — shows active forwards, allows stopping
// ---------------------------------------------------------------------------

type pfListItem struct {
	index int
	pf    *PortForward
}

func (i pfListItem) Title() string       { return i.pf.String() }
func (i pfListItem) Description() string { return "Press Enter or d to stop" }
func (i pfListItem) FilterValue() string { return i.pf.PodName }

type portForwardListView struct {
	pfManager *portForwardManager
	list      list.Model
}

func newPortForwardListView(pfManager *portForwardManager) *portForwardListView {
	forwards := pfManager.List()
	items := make([]list.Item, len(forwards))
	for i, pf := range forwards {
		items[i] = pfListItem{index: i, pf: pf}
	}

	l := list.New(items, list.NewDefaultDelegate(), 60, 14)
	l.SetShowTitle(true)
	l.Title = "Active Port Forwards"
	l.SetShowStatusBar(true)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	return &portForwardListView{
		pfManager: pfManager,
		list:      l,
	}
}

func (v *portForwardListView) Title() string { return "Port Forwards" }

func (v *portForwardListView) Init() tea.Cmd { return nil }

func (v *portForwardListView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter", "d":
			selected, ok := v.list.SelectedItem().(pfListItem)
			if !ok {
				return v, nil
			}
			v.pfManager.Remove(selected.index)
			// Refresh the list
			v.refreshList()
			if len(v.pfManager.List()) == 0 {
				return v, func() tea.Msg { return PopViewMsg{} }
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

func (v *portForwardListView) refreshList() {
	forwards := v.pfManager.List()
	items := make([]list.Item, len(forwards))
	for i, pf := range forwards {
		items[i] = pfListItem{index: i, pf: pf}
	}
	v.list.SetItems(items)
}

func (v *portForwardListView) View() string {
	if len(v.pfManager.List()) == 0 {
		return theme.MutedStyle.Render("No active port forwards") + "\n\n" +
			theme.MutedStyle.Render("Press Esc to go back")
	}
	return v.list.View()
}

func (v *portForwardListView) SetSize(width, height int) {
	v.list.SetSize(width, height)
}

