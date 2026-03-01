package services

import (
	"testing"

	awselb "tasnim.dev/aws-tui/internal/aws/elb"
)

func TestELBDetailView_TabNames(t *testing.T) {
	v := NewELBDetailView(nil, awselb.ELBLoadBalancer{})
	expected := []string{"Listeners", "Target Groups", "Rules", "Attributes", "Tags"}
	if len(v.tabs.TabNames) != len(expected) {
		t.Fatalf("expected %d tabs, got %d", len(expected), len(v.tabs.TabNames))
	}
	for i, name := range expected {
		if v.tabs.TabNames[i] != name {
			t.Errorf("tab %d: expected %q, got %q", i, name, v.tabs.TabNames[i])
		}
	}
}

func TestELBDetailView_TabSwitch(t *testing.T) {
	v := NewELBDetailView(nil, awselb.ELBLoadBalancer{})
	v.tabs.InitTab = func(idx int) View { return nil }

	v.tabs.SwitchTab(2)
	if v.tabs.ActiveTab != 2 {
		t.Errorf("expected tab 2, got %d", v.tabs.ActiveTab)
	}

	// Tab wraps from last to first
	v.tabs.SwitchTab(4)
	handled, _ := v.tabs.HandleKey("tab")
	if !handled {
		t.Error("expected tab key to be handled")
	}
	if v.tabs.ActiveTab != 0 {
		t.Errorf("expected tab 0 after wrapping, got %d", v.tabs.ActiveTab)
	}
}

func TestELBDetailView_NumberKeys(t *testing.T) {
	v := NewELBDetailView(nil, awselb.ELBLoadBalancer{})
	v.tabs.InitTab = func(idx int) View { return nil }

	for i := 1; i <= 5; i++ {
		key := string(rune('0' + i))
		handled, _ := v.tabs.HandleKey(key)
		if !handled {
			t.Errorf("key %q: expected handled", key)
		}
		if v.tabs.ActiveTab != i-1 {
			t.Errorf("key %q: expected tab %d, got %d", key, i-1, v.tabs.ActiveTab)
		}
	}
}
