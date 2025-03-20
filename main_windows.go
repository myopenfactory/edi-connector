package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"time"

	"github.com/myopenfactory/edi-connector/v2/config"
	"github.com/myopenfactory/edi-connector/v2/connector"
	"golang.org/x/sys/windows/svc"
)

func init() {
	run := windowsRun
	serviceRun = &run
}

func windowsRun(ctx context.Context, logger *slog.Logger, cfg config.Config, serviceName string) error {
	cl, err := connector.New(logger, cfg)
	if err != nil {
		return fmt.Errorf("failed to create edi-connector: %w", err)
	}

	if err := svc.Run(serviceName, &service{connector: cl}); err != nil {
		return fmt.Errorf("service %q failed to run: %w", serviceName, err)
	}
	return nil
}

type service struct {
	logger    *slog.Logger
	connector *connector.Connector
}

func (s *service) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("client panic", "value", r, "stack", debug.Stack())
			}
		}()
		if err := s.connector.Run(ctx); err != nil {
			s.logger.Error("error while running client", "error", err)
			os.Exit(1)
		}
	}()

	deadline := 5 * time.Second
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted, WaitHint: uint32(deadline.Seconds()) * 1000}
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				cancel()
				return false, 0
			}
		}
	}
}
