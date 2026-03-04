package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// ClipboardMsg is returned after a clipboard copy attempt.
type ClipboardMsg struct {
	Text string
	Err  error
}

// clipboardCmd describes a system command used to write to the clipboard.
type clipboardCmd struct {
	name string
	args []string
}

// clipboardCommands returns the ordered list of clipboard commands to try for
// the current OS. The first command whose binary is found on PATH is used.
func clipboardCommands() []clipboardCmd {
	switch runtime.GOOS {
	case "darwin":
		return []clipboardCmd{
			{name: "pbcopy"},
		}
	case "linux":
		return []clipboardCmd{
			{name: "xclip", args: []string{"-selection", "clipboard"}},
			{name: "xsel", args: []string{"--clipboard", "--input"}},
		}
	default:
		return nil
	}
}

// detectClipboardCmd returns the first available clipboard command on the
// system, or an error if none is found.
func detectClipboardCmd() (clipboardCmd, error) {
	for _, c := range clipboardCommands() {
		if _, err := exec.LookPath(c.name); err == nil {
			return c, nil
		}
	}
	return clipboardCmd{}, fmt.Errorf("no clipboard command available for %s", runtime.GOOS)
}

// CopyToClipboard returns a Bubble Tea command that copies text to the system
// clipboard. The resulting message is a ClipboardMsg indicating success or
// failure.
func CopyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		cc, err := detectClipboardCmd()
		if err != nil {
			return ClipboardMsg{Text: text, Err: err}
		}

		cmd := exec.Command(cc.name, cc.args...) // #nosec G204
		cmd.Stdin = strings.NewReader(text)

		if err := cmd.Run(); err != nil {
			return ClipboardMsg{Text: text, Err: fmt.Errorf("%s: %w", cc.name, err)}
		}

		return ClipboardMsg{Text: text}
	}
}
