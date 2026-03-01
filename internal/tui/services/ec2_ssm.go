package services

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"golang.org/x/term"

	awsec2 "tasnim.dev/aws-tui/internal/aws/ec2"
	"tasnim.dev/aws-tui/internal/tui/theme"
)

// ---------------------------------------------------------------------------
// SSM Process — implements tea.ExecCommand
// ---------------------------------------------------------------------------

type ssmProcess struct {
	instance awsec2.EC2Instance
	profile  string
	region   string
	stdin    io.Reader
	stdout   io.Writer
	stderr   io.Writer
}

func (p *ssmProcess) SetStdin(r io.Reader)  { p.stdin = r }
func (p *ssmProcess) SetStdout(w io.Writer) { p.stdout = w }
func (p *ssmProcess) SetStderr(w io.Writer) { p.stderr = w }

func (p *ssmProcess) Run() error {
	// Check for session-manager-plugin
	if _, err := exec.LookPath("session-manager-plugin"); err != nil {
		return fmt.Errorf("session-manager-plugin not found in PATH — install it: https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html")
	}

	p.printHeader()

	args := []string{"ssm", "start-session", "--target", p.instance.InstanceID}
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

func (p *ssmProcess) printHeader() {
	w, _, _ := term.GetSize(int(os.Stdout.Fd()))
	if w <= 0 {
		w = 80
	}

	name := p.instance.InstanceID
	if p.instance.Name != "" {
		name = p.instance.Name + " (" + p.instance.InstanceID + ")"
	}

	label := " SSM Connect "
	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#1a1a2e")).
		Background(lipgloss.Color("#4ADE80"))
	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4ADE80"))
	divStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))

	info := fmt.Sprintf(" %s", name)

	header := labelStyle.Render(label)
	headerInfo := infoStyle.Render(info)
	divider := divStyle.Render(strings.Repeat("─", w))

	fmt.Fprintf(p.stdout, "%s%s\n%s\n", header, headerInfo, divider)
	fmt.Fprintf(p.stdout, "\033]0;ssm %s\007", p.instance.InstanceID)
}

// ---------------------------------------------------------------------------
// SSM Input View — confirmation before connecting
// ---------------------------------------------------------------------------

type ssmInputView struct {
	instance awsec2.EC2Instance
	profile  string
	region   string
}

func newSSMInputView(instance awsec2.EC2Instance, profile, region string) *ssmInputView {
	return &ssmInputView{
		instance: instance,
		profile:  profile,
		region:   region,
	}
}

func (v *ssmInputView) Title() string {
	return fmt.Sprintf("SSM: %s", v.instance.InstanceID)
}

func (v *ssmInputView) Init() tea.Cmd { return nil }

func (v *ssmInputView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			process := &ssmProcess{
				instance: v.instance,
				profile:  v.profile,
				region:   v.region,
			}
			return v, tea.Exec(process, func(err error) tea.Msg {
				return execDoneMsg{err: err}
			})
		case "esc":
			return v, func() tea.Msg { return PopViewMsg{} }
		}
	}
	return v, nil
}

func (v *ssmInputView) View() string {
	var b strings.Builder
	bold := lipgloss.NewStyle().Bold(true)

	name := v.instance.InstanceID
	if v.instance.Name != "" {
		name = v.instance.Name + " (" + v.instance.InstanceID + ")"
	}

	b.WriteString(bold.Render("SSM Connect"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Instance: %s\n", name))
	b.WriteString(fmt.Sprintf("State:    %s\n", v.instance.State))
	b.WriteString("\n")
	b.WriteString(theme.MutedStyle.Render("Enter to connect  Esc to cancel"))
	return b.String()
}

func (v *ssmInputView) SetSize(width, height int) {}
