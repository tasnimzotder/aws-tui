package cmd

import (
	"errors"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awscost "tasnim.dev/aws-tui/internal/aws/cost"
	"tasnim.dev/aws-tui/internal/config"
	"tasnim.dev/aws-tui/internal/tui"
)

func NewCostCmd() *cobra.Command {
	var profile string

	cmd := &cobra.Command{
		Use:   "cost",
		Short: "Show AWS cost dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			profile, _ = cfg.Merge(profile, "")

			awsCfg, err := awsclient.LoadConfig(ctx, profile, "")
			if err != nil {
				return fmt.Errorf("loading AWS config: %w", err)
			}

			accountID := awsclient.GetAccountID(ctx, awsCfg)
			client := awscost.NewClient(awsCfg)

			model := tui.NewModel(client, profile, accountID)
			p := tea.NewProgram(model, tea.WithContext(ctx))
			if _, err := p.Run(); err != nil {
				if errors.Is(err, tea.ErrInterrupted) || ctx.Err() != nil {
					return nil
				}
				return fmt.Errorf("running TUI: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&profile, "profile", "p", "", "AWS profile to use")

	return cmd
}
