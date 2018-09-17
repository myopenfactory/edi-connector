package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "DEV"
)

func init() {
	rootCmd.AddCommand(runCmd)
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "shows version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version: %s", version)
	},
}