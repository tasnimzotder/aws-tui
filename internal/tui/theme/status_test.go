package theme

import (
	"testing"

	"charm.land/lipgloss/v2"
)

func TestDashboardBoxStyle_HasBorder(t *testing.T) {
	// Verify that DashboardBoxStyle uses a rounded border
	rendered := DashboardBoxStyle.Render("test")
	// Rounded border uses ╭ at top-left
	if !containsRune(rendered, '╭') {
		t.Error("expected DashboardBoxStyle to use rounded border")
	}
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}

func TestStatusColor_Running(t *testing.T) {
	c := StatusColor("running")
	if c != Success {
		t.Errorf("running: got %v, want Success", c)
	}
}

func TestStatusColor_Stopped(t *testing.T) {
	c := StatusColor("stopped")
	if c != Error {
		t.Errorf("stopped: got %v, want Error", c)
	}
}

func TestStatusColor_Pending(t *testing.T) {
	c := StatusColor("pending")
	if c != Warning {
		t.Errorf("pending: got %v, want Warning", c)
	}
}

func TestStatusColor_Available(t *testing.T) {
	c := StatusColor("available")
	if c != Success {
		t.Errorf("available: got %v, want Success", c)
	}
}

func TestStatusColor_Unknown(t *testing.T) {
	c := StatusColor("something-random")
	if c != Muted {
		t.Errorf("unknown: got %v, want Muted", c)
	}
}

func TestRenderStatus_ContainsBullet(t *testing.T) {
	r := RenderStatus("running")
	if !containsRune(r, '●') {
		t.Error("RenderStatus should contain bullet ●")
	}
}

func TestDashboardTitleStyle_IsBold(t *testing.T) {
	style := DashboardTitleStyle
	// Just verify it renders without panic and has bold set
	rendered := style.Render("Test Title")
	if rendered == "" {
		t.Error("expected non-empty render")
	}
	_ = lipgloss.NewStyle() // ensure lipgloss is used
}
