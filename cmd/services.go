package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	"tasnim.dev/aws-tui/internal/config"
	"tasnim.dev/aws-tui/internal/tui/services"
)

func NewServicesCmd() *cobra.Command {
	var profile string
	var region string

	cmd := &cobra.Command{
		Use:   "services",
		Short: "Browse AWS services (EC2, ECS, VPC, ECR)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Suppress all klog/client-go stderr output to prevent TUI corruption.
			// Must use SetLogger for the structured logger, plus SetOutput for legacy paths.
			klog.SetLogger(logr.Discard())
			klog.SetOutput(io.Discard)
			klog.LogToStderr(false)

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			profile, region = cfg.Merge(profile, region)

			client, err := awsclient.NewServiceClient(context.Background(), profile, region)
			if err != nil {
				return fmt.Errorf("initializing AWS client: %w", err)
			}

			model := services.NewModel(client, profile, region, cfg)
			p := tea.NewProgram(model)
			if _, err := p.Run(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&profile, "profile", "p", "", "AWS profile to use")
	cmd.Flags().StringVarP(&region, "region", "r", "", "AWS region to use")

	return cmd
}
