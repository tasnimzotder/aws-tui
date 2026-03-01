package services

import (
	"testing"

	awsvpc "tasnim.dev/aws-tui/internal/aws/vpc"
)

func TestVPCDetailView_TabNames(t *testing.T) {
	v := NewVPCDetailView(nil, awsvpc.VPCInfo{})
	expected := []string{
		"Subnets", "Security Groups", "Route Tables", "Internet Gateways", "NAT Gateways",
		"Endpoints", "Peering", "NACLs", "Flow Logs", "Tags",
	}
	if len(v.tabs.TabNames) != len(expected) {
		t.Fatalf("expected %d tabs, got %d", len(expected), len(v.tabs.TabNames))
	}
	for i, name := range expected {
		if v.tabs.TabNames[i] != name {
			t.Errorf("tab %d: expected %q, got %q", i, name, v.tabs.TabNames[i])
		}
	}
}

func TestVPCDetailView_TabSwitch(t *testing.T) {
	v := NewVPCDetailView(nil, awsvpc.VPCInfo{})
	v.tabs.InitTab = func(idx int) View { return nil }

	v.tabs.SwitchTab(5)
	if v.tabs.ActiveTab != 5 {
		t.Errorf("expected tab 5, got %d", v.tabs.ActiveTab)
	}
}

func TestVPCDetailView_NumberKeys(t *testing.T) {
	v := NewVPCDetailView(nil, awsvpc.VPCInfo{})
	v.tabs.InitTab = func(idx int) View { return nil }

	// Keys 1-9 map to tabs 0-8
	for i := 1; i <= 9; i++ {
		key := string(rune('0' + i))
		handled, _ := v.tabs.HandleKey(key)
		if !handled {
			t.Errorf("key %q: expected handled", key)
		}
		if v.tabs.ActiveTab != i-1 {
			t.Errorf("key %q: expected tab %d, got %d", key, i-1, v.tabs.ActiveTab)
		}
	}

	// Key "0" maps to tab 9 (10th tab)
	handled, _ := v.tabs.HandleKey("0")
	if !handled {
		t.Error("key 0: expected handled")
	}
	if v.tabs.ActiveTab != 9 {
		t.Errorf("key 0: expected tab 9, got %d", v.tabs.ActiveTab)
	}
}
