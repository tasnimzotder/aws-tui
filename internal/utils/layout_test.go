package utils

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestDetailBuilder_Row(t *testing.T) {
	style := lipgloss.NewStyle()
	db := NewDetailBuilder(16, style)
	db.Row("Name", "my-service")

	got := db.String()
	if !strings.Contains(got, "Name") {
		t.Error("Row should contain label")
	}
	if !strings.Contains(got, "my-service") {
		t.Error("Row should contain value")
	}
}

func TestDetailBuilder_Section(t *testing.T) {
	style := lipgloss.NewStyle()
	db := NewDetailBuilder(16, style)
	db.Section("Containers")

	got := db.String()
	if !strings.Contains(got, "── Containers") {
		t.Error("Section should contain heading")
	}
	if !strings.Contains(got, "───") {
		t.Error("Section should contain padding dashes")
	}
}

func TestDetailBuilder_Blank(t *testing.T) {
	style := lipgloss.NewStyle()
	db := NewDetailBuilder(16, style)
	db.Row("A", "1")
	db.Blank()
	db.Row("B", "2")

	got := db.String()
	if !strings.Contains(got, "\n\n") {
		t.Error("Blank should insert empty line")
	}
}

func TestDetailBuilder_Combined(t *testing.T) {
	style := lipgloss.NewStyle()
	db := NewDetailBuilder(16, style)
	db.Section("Info")
	db.Row("Name", "test")
	db.Blank()
	db.Section("Details")
	db.Row("Status", "active")

	got := db.String()
	if !strings.Contains(got, "Info") {
		t.Error("should contain first section")
	}
	if !strings.Contains(got, "Details") {
		t.Error("should contain second section")
	}
	if !strings.Contains(got, "test") {
		t.Error("should contain first value")
	}
	if !strings.Contains(got, "active") {
		t.Error("should contain second value")
	}
}
