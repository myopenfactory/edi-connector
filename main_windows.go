package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/myopenfactory/edi-connector/config"
	"github.com/myopenfactory/edi-connector/ediconnector"
	"github.com/spf13/viper"
	"golang.org/x/sys/windows/svc"
)

func init() {
	run := windowsRun
	serviceRun = &run
}

func windowsRun(ctx context.Context, logger *slog.Logger, cfg config.Config) error {
	cl, err := ediconnector.New(logger, cfg)
	if err != nil {
		return fmt.Errorf("failed to create edi-connector: %w", err)
	}

	serviceName := viper.GetString("service.name")
	if err := svc.Run(serviceName, &service{connector: cl}); err != nil {
		return fmt.Errorf("service %q failed to run: %w", serviceName, err)
	}
	return nil
}

type service struct {
	connector *ediconnector.Connector
}

func (m *service) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	deadline := 5 * time.Second
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted, WaitHint: uint32(deadline.Seconds()) * 1000}
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				return false, 0
			}
		}
	}
}
