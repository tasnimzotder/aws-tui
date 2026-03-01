package services

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"

	awsvpc "tasnim.dev/aws-tui/internal/aws/vpc"
)

// navigateVPCMsg is sent when a VPC lookup succeeds.
type navigateVPCMsg struct {
	vpc awsvpc.VPCInfo
}

// navigateVPCErrMsg is sent when a VPC lookup fails.
type navigateVPCErrMsg struct {
	err error
}

// NavigateToVPC looks up a VPC by ID and returns a message to navigate to it.
// Used by EC2 and ELB detail views to jump to the VPC detail.
func NavigateToVPC(vpcClient *awsvpc.Client, vpcID string) tea.Cmd {
	return func() tea.Msg {
		vpcs, err := vpcClient.ListVPCs(context.Background())
		if err != nil {
			return navigateVPCErrMsg{err: err}
		}
		for _, vpc := range vpcs {
			if vpc.VPCID == vpcID {
				return navigateVPCMsg{vpc: vpc}
			}
		}
		return navigateVPCErrMsg{err: fmt.Errorf("VPC %s not found", vpcID)}
	}
}
