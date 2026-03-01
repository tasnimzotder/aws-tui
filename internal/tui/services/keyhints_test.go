package services

import (
	"strings"
	"testing"
)

func TestRenderKeyHints_Root(t *testing.T) {
	hints := RenderKeyHints(HelpContextRoot, 120)
	if !strings.Contains(hints, "Enter") {
		t.Error("root hints should contain Enter")
	}
	if !strings.Contains(hints, "quit") {
		t.Error("root hints should contain quit")
	}
}

func TestRenderKeyHints_Table(t *testing.T) {
	hints := RenderKeyHints(HelpContextTable, 120)
	if !strings.Contains(hints, "Enter") {
		t.Error("table hints should contain Enter")
	}
	if !strings.Contains(hints, "filter") {
		t.Error("table hints should contain filter")
	}
	if !strings.Contains(hints, "refresh") {
		t.Error("table hints should contain refresh")
	}
}

func TestRenderKeyHints_Detail(t *testing.T) {
	hints := RenderKeyHints(HelpContextDetail, 120)
	if !strings.Contains(hints, "Tab") {
		t.Error("detail hints should contain Tab")
	}
	if !strings.Contains(hints, "back") {
		t.Error("detail hints should contain back")
	}
}

func TestRenderKeyHints_MaxWidth(t *testing.T) {
	// Very narrow width should still produce something
	hints := RenderKeyHints(HelpContextTable, 30)
	if hints == "" {
		t.Error("key hints should not be empty even at narrow width")
	}
}
