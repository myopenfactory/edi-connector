package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "1.1.0-DEV"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Version:", version)
	},
}
