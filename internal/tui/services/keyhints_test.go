package services

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
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

func TestEstimatePlainLen(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty string", "", 0},
		{"plain text", "hello world", 11},
		{"with ANSI escape", "\x1b[31mred\x1b[0m", 3},
		{"nested ANSI escapes", "\x1b[1m\x1b[34mbold blue\x1b[0m", 9},
		{"no escape content", "abc", 3},
		{"escape at end", "hi\x1b[0m", 2},
		{"multiple sequences", "\x1b[1mA\x1b[0m \x1b[2mB\x1b[0m", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimatePlainLen(tt.input)
			if got != tt.want {
				t.Errorf("estimatePlainLen(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestRenderKeyHints_AllContexts(t *testing.T) {
	contexts := []struct {
		name string
		ctx  HelpContext
		want string // at least one expected hint
	}{
		{"Root", HelpContextRoot, "select"},
		{"Table", HelpContextTable, "drill"},
		{"Detail", HelpContextDetail, "switch"},
		{"ECSTaskDetail", HelpContextECSTaskDetail, "exec"},
		{"EC2Detail", HelpContextEC2Detail, "SSM"},
		{"ELBDetail", HelpContextELBDetail, "VPC"},
		{"VPCDetail", HelpContextVPCDetail, "jump"},
		{"EKSDetail", HelpContextEKSDetail, "namespace"},
		{"K8sPods", HelpContextK8sPods, "logs"},
		{"K8sNodes", HelpContextK8sNodes, "YAML"},
		{"K8sLogs", HelpContextK8sLogs, "follow"},
		{"S3Objects", HelpContextS3Objects, "download"},
		{"TextView", HelpContextTextView, "search"},
	}

	for _, tt := range contexts {
		t.Run(tt.name, func(t *testing.T) {
			hints := RenderKeyHints(tt.ctx, 200)
			if hints == "" {
				t.Error("hints should not be empty")
			}
			if !strings.Contains(hints, tt.want) {
				t.Errorf("hints for %s missing %q, got: %s", tt.name, tt.want, hints)
			}
		})
	}
}

func TestDetectHelpContext(t *testing.T) {
	tests := []struct {
		name string
		view View
		want HelpContext
	}{
		{
			name: "view with HelpContext returns that context",
			view: &mockHelpContextView{ctx: HelpContextK8sPods},
			want: HelpContextK8sPods,
		},
		{
			name: "filterable view returns Table context",
			view: &mockFilterableView{},
			want: HelpContextTable,
		},
		{
			name: "plain view returns Root context",
			view: &mockPlainView{},
			want: HelpContextRoot,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectHelpContext(tt.view)
			if got != tt.want {
				t.Errorf("detectHelpContext() = %d, want %d", got, tt.want)
			}
		})
	}
}

// Mock views for detectHelpContext tests
type mockHelpContextView struct {
	mockPlainView
	ctx HelpContext
}

func (m *mockHelpContextView) HelpContext() *HelpContext { return &m.ctx }

type mockFilterableView struct {
	mockPlainView
}

func (m *mockFilterableView) AllRows() []table.Row    { return nil }
func (m *mockFilterableView) SetRows(rows []table.Row) {}

type mockPlainView struct{}

func (m *mockPlainView) Update(msg tea.Msg) (View, tea.Cmd) { return m, nil }
func (m *mockPlainView) View() string                       { return "" }
func (m *mockPlainView) Title() string                      { return "mock" }
func (m *mockPlainView) Init() tea.Cmd                      { return nil }
