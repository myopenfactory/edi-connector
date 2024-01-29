package cmd

import (
	"github.com/myopenfactory/client/pkg/config"
	"github.com/myopenfactory/client/pkg/log"
	"github.com/spf13/cobra"
)

func init() {
	Check.AddCommand(mailCmd)
}

var Check = &cobra.Command{
	Use:   "check",
	Short: "check the configuration",
}

var mailCmd = &cobra.Command{
	Use:   "mail",
	Short: "send test mail",
	Run: func(cmd *cobra.Command, args []string) {
		logger := log.New(config.ParseLogOptions()...)
		logger.Error("test email")
	},
}
