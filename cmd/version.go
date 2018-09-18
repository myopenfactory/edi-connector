package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "0.0.0-DEV"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "shows version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version: %s", version)
	},
}
