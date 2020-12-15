// +build !windows

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Bootstrap represents the bootstrap command
var Bootstrap = &cobra.Command{
	Use:   "bootstrap",
	Short: "bootstrap the client",
	Long:  "bootstrap the client.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("not implemented for your os")
		os.Exit(1)
	},
}
