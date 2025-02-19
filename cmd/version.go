package cmd

import (
	"fmt"

	"github.com/myopenfactory/edi-connector/version"
	"github.com/spf13/cobra"
)

// Version represents the version command
var Version = &cobra.Command{
	Use:   "version",
	Short: "show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Version:", version.Version)
		fmt.Println("Date:", version.Date)
		fmt.Println("Commit:", version.Commit)
	},
}
