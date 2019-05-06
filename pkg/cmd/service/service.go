package service

import (
	"github.com/spf13/cobra"
)

// serviceCmd represents the service command
var Command = &cobra.Command{
	Use:   "service",
	Short: "administrate windows service",
}
