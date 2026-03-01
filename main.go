package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"tasnim.dev/aws-tui/cmd"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	rootCmd := &cobra.Command{
		Use:   "aws-tui",
		Short: "AWS utility tools",
	}

	rootCmd.AddCommand(cmd.NewCostCmd())
	rootCmd.AddCommand(cmd.NewServicesCmd())

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
