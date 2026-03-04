package main

import (
	"os"

	"tasnim.dev/aws-tui/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
