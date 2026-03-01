package utils

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestDetailBuilder_Row_TableDriven(t *testing.T) {
	tests := []struct {
		name       string
		label      string
		value      string
		wantLabel  string
		wantValue  string
	}{
		{"simple row", "Name", "my-service", "Name", "my-service"},
		{"empty value", "Status", "", "Status", ""},
		{"long value", "ARN", "arn:aws:ecs:us-east-1:123456789:cluster/prod", "ARN", "arn:aws:ecs:us-east-1:123456789:cluster/prod"},
		{"special chars", "Tags", "env=prod, team=infra", "Tags", "env=prod, team=infra"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			style := lipgloss.NewStyle()
			db := NewDetailBuilder(16, style)
			db.Row(tt.label, tt.value)
			got := db.String()
			if !strings.Contains(got, tt.wantLabel) {
				t.Errorf("Row should contain label %q", tt.wantLabel)
			}
			if tt.wantValue != "" && !strings.Contains(got, tt.wantValue) {
				t.Errorf("Row should contain value %q", tt.wantValue)
			}
		})
	}
}

func TestDetailBuilder_Section_TableDriven(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		wantTitle string
		wantDash  bool
	}{
		{"normal section", "Containers", "── Containers", true},
		{"short title", "Info", "── Info", true},
		{"long title pads at least 4 dashes", "A Very Long Section Title Here!!!!!", "── A Very Long Section Title Here!!!!!", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			style := lipgloss.NewStyle()
			db := NewDetailBuilder(16, style)
			db.Section(tt.title)
			got := db.String()
			if !strings.Contains(got, tt.wantTitle) {
				t.Errorf("Section should contain %q, got %q", tt.wantTitle, got)
			}
			if tt.wantDash && !strings.Contains(got, "───") {
				t.Error("Section should contain padding dashes")
			}
		})
	}
}

func TestDetailBuilder_WriteString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", "custom line\n", "custom line"},
		{"empty string", "", ""},
		{"multiple writes", "first", "first"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			style := lipgloss.NewStyle()
			db := NewDetailBuilder(16, style)
			db.WriteString(tt.input)
			got := db.String()
			if tt.want != "" && !strings.Contains(got, tt.want) {
				t.Errorf("WriteString should contain %q, got %q", tt.want, got)
			}
			if tt.want == "" && got != "" {
				t.Errorf("WriteString of empty should produce empty, got %q", got)
			}
		})
	}
}

func TestDetailBuilder_Combined(t *testing.T) {
	tests := []struct {
		name         string
		build        func(db *DetailBuilder)
		wantContains []string
	}{
		{
			name: "section then rows",
			build: func(db *DetailBuilder) {
				db.Section("Info")
				db.Row("Name", "test")
				db.Row("Status", "active")
			},
			wantContains: []string{"Info", "Name", "test", "Status", "active"},
		},
		{
			name: "multiple sections with blank",
			build: func(db *DetailBuilder) {
				db.Section("Info")
				db.Row("Name", "test")
				db.Blank()
				db.Section("Details")
				db.Row("Region", "us-east-1")
			},
			wantContains: []string{"Info", "Details", "test", "us-east-1"},
		},
		{
			name: "write string mixed with rows",
			build: func(db *DetailBuilder) {
				db.Row("Name", "svc")
				db.WriteString("  custom: data\n")
			},
			wantContains: []string{"Name", "svc", "custom: data"},
		},
		{
			name: "blank inserts empty line",
			build: func(db *DetailBuilder) {
				db.Row("A", "1")
				db.Blank()
				db.Row("B", "2")
			},
			wantContains: []string{"\n\n", "A", "B"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			style := lipgloss.NewStyle()
			db := NewDetailBuilder(16, style)
			tt.build(db)
			got := db.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("should contain %q in output %q", want, got)
				}
			}
		})
	}
}
