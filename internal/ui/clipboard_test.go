package ui

import (
	"runtime"
	"testing"
)

func TestClipboardCommandsReturnsEntries(t *testing.T) {
	cmds := clipboardCommands()

	switch runtime.GOOS {
	case "darwin":
		if len(cmds) != 1 {
			t.Fatalf("expected 1 command on darwin, got %d", len(cmds))
		}
		if cmds[0].name != "pbcopy" {
			t.Errorf("expected pbcopy, got %s", cmds[0].name)
		}
	case "linux":
		if len(cmds) != 2 {
			t.Fatalf("expected 2 commands on linux, got %d", len(cmds))
		}
		if cmds[0].name != "xclip" {
			t.Errorf("expected xclip first, got %s", cmds[0].name)
		}
		if cmds[1].name != "xsel" {
			t.Errorf("expected xsel second, got %s", cmds[1].name)
		}
	default:
		if len(cmds) != 0 {
			t.Fatalf("expected 0 commands on %s, got %d", runtime.GOOS, len(cmds))
		}
	}
}

func TestDetectClipboardCmdFindsCommand(t *testing.T) {
	// On macOS and most Linux CI environments at least one clipboard tool is
	// available. On unsupported platforms we expect an error.
	cc, err := detectClipboardCmd()

	switch runtime.GOOS {
	case "darwin":
		if err != nil {
			t.Fatalf("expected pbcopy to be found on darwin: %v", err)
		}
		if cc.name != "pbcopy" {
			t.Errorf("expected pbcopy, got %s", cc.name)
		}
	case "linux":
		// xclip or xsel may not be installed in CI; just check consistency.
		if err != nil {
			t.Skipf("no clipboard command found on linux (expected in headless CI): %v", err)
		}
		if cc.name != "xclip" && cc.name != "xsel" {
			t.Errorf("unexpected command: %s", cc.name)
		}
	default:
		if err == nil {
			t.Fatalf("expected error on unsupported OS %s", runtime.GOOS)
		}
	}
}

func TestCopyToClipboardReturnsCmd(t *testing.T) {
	cmd := CopyToClipboard("hello")
	if cmd == nil {
		t.Fatal("expected non-nil tea.Cmd")
	}
}

func TestClipboardMsgFields(t *testing.T) {
	msg := ClipboardMsg{Text: "test", Err: nil}
	if msg.Text != "test" {
		t.Errorf("expected Text=test, got %s", msg.Text)
	}
	if msg.Err != nil {
		t.Errorf("expected nil Err, got %v", msg.Err)
	}
}
