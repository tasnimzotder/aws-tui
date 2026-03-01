package services

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"

	awsvpc "tasnim.dev/aws-tui/internal/aws/vpc"
)

func TestNavigateToVPC_Found(t *testing.T) {
	vpcs := []awsvpc.VPCInfo{
		{VPCID: "vpc-111", Name: "first"},
		{VPCID: "vpc-222", Name: "second"},
	}
	cmd := navigateToVPCFromList(vpcs, "vpc-222")
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	navMsg, ok := msg.(navigateVPCMsg)
	if !ok {
		t.Fatalf("expected navigateVPCMsg, got %T", msg)
	}
	if navMsg.vpc.VPCID != "vpc-222" {
		t.Errorf("vpc.VPCID = %s, want vpc-222", navMsg.vpc.VPCID)
	}
}

func TestNavigateToVPC_NotFound(t *testing.T) {
	vpcs := []awsvpc.VPCInfo{
		{VPCID: "vpc-111"},
	}
	cmd := navigateToVPCFromList(vpcs, "vpc-999")
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	errMsg, ok := msg.(navigateVPCErrMsg)
	if !ok {
		t.Fatalf("expected navigateVPCErrMsg, got %T", msg)
	}
	if errMsg.err == nil {
		t.Error("expected non-nil error")
	}
}

// navigateToVPCFromList is a test helper that simulates the lookup without
// needing a real VPC client.
func navigateToVPCFromList(vpcs []awsvpc.VPCInfo, vpcID string) tea.Cmd {
	return func() tea.Msg {
		for _, vpc := range vpcs {
			if vpc.VPCID == vpcID {
				return navigateVPCMsg{vpc: vpc}
			}
		}
		return navigateVPCErrMsg{err: fmt.Errorf("VPC %s not found", vpcID)}
	}
}
