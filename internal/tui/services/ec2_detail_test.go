package services

import (
	"testing"

	awsec2 "tasnim.dev/aws-tui/internal/aws/ec2"
)

func TestEC2DetailView_TabNames(t *testing.T) {
	v := NewEC2DetailView(nil, awsec2.EC2Instance{}, "", "")
	expected := []string{"Details", "Security Groups", "Volumes", "Tags"}
	if len(v.tabs.TabNames) != len(expected) {
		t.Fatalf("expected %d tabs, got %d", len(expected), len(v.tabs.TabNames))
	}
	for i, name := range expected {
		if v.tabs.TabNames[i] != name {
			t.Errorf("tab %d: expected %q, got %q", i, name, v.tabs.TabNames[i])
		}
	}
}

func TestEC2DetailView_TabSwitch(t *testing.T) {
	v := NewEC2DetailView(nil, awsec2.EC2Instance{}, "", "")
	// Override InitTab to avoid needing real AWS client
	v.tabs.InitTab = func(idx int) View { return nil }

	v.tabs.SwitchTab(1)
	if v.tabs.ActiveTab != 1 {
		t.Errorf("expected tab 1, got %d", v.tabs.ActiveTab)
	}

	// Tab key cycles forward
	handled, _ := v.tabs.HandleKey("tab")
	if !handled {
		t.Error("expected tab key to be handled")
	}
	if v.tabs.ActiveTab != 2 {
		t.Errorf("expected tab 2, got %d", v.tabs.ActiveTab)
	}

	// Shift+tab cycles backward
	handled, _ = v.tabs.HandleKey("shift+tab")
	if !handled {
		t.Error("expected shift+tab to be handled")
	}
	if v.tabs.ActiveTab != 1 {
		t.Errorf("expected tab 1, got %d", v.tabs.ActiveTab)
	}
}

func TestEC2DetailView_NumberKeys(t *testing.T) {
	v := NewEC2DetailView(nil, awsec2.EC2Instance{}, "", "")
	v.tabs.InitTab = func(idx int) View { return nil }

	tests := []struct {
		key      string
		expected int
	}{
		{"1", 0},
		{"2", 1},
		{"3", 2},
		{"4", 3},
	}

	for _, tt := range tests {
		handled, _ := v.tabs.HandleKey(tt.key)
		if !handled {
			t.Errorf("key %q: expected handled", tt.key)
		}
		if v.tabs.ActiveTab != tt.expected {
			t.Errorf("key %q: expected tab %d, got %d", tt.key, tt.expected, v.tabs.ActiveTab)
		}
	}

	// Key "5" should not be handled (only 4 tabs)
	handled, _ := v.tabs.HandleKey("5")
	if handled {
		t.Error("key 5 should not be handled for 4-tab view")
	}
}
