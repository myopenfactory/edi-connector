package version

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/myopenfactory/client/pkg/version"
)

// Command represents the version command
var Command = &cobra.Command{
	Use:   "version",
	Short: "show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Version:", version.Version)
		fmt.Println("Date:", version.Date)
		fmt.Println("Commit:", version.Commit)
	},
}
