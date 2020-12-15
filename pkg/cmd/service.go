package cmd

import (
	"github.com/spf13/cobra"
)

// Service represents the service command
var Service = &cobra.Command{
	Use:   "service",
	Short: "administrate windows service",
}
