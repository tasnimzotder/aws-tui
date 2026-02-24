package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"tasnim.dev/aws-tui/cmd"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "aws-tui",
		Short: "AWS utility tools",
	}

	rootCmd.AddCommand(cmd.NewCostCmd())
	rootCmd.AddCommand(cmd.NewServicesCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
