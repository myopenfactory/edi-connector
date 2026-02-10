package main

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/myopenfactory/edi-connector/v2/config"
	"github.com/myopenfactory/edi-connector/v2/connector"
	"github.com/myopenfactory/edi-connector/v2/log"
	"github.com/myopenfactory/edi-connector/v2/version"
	"golang.org/x/sys/windows/svc"
)

func serviceRun(configFile string, logLevel string) error {
	err := svc.Run("EDI-Connector", &service{
		configFile: configFile,
		logLevel:   logLevel,
	})
	if err != nil {
		return err
	}
	return nil
}

func isWindowsService() bool {
	ok, err := svc.IsWindowsService()
	if err != nil {
		panic(fmt.Sprintf("failed to check for windows service: %v", err))
	}
	return ok
}

type service struct {
	configFile string
	logLevel   string
}

func (s *service) Execute(args []string, r <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending}

	cfg, configFile, err := config.ReadConfigFromFile(s.configFile)
	if err != nil {
		status <- svc.Status{State: svc.StopPending}
		fmt.Printf("failed to load configfile: %v\n", err)
		return false, 1
	}

	if s.logLevel != "" {
		cfg.Log.Level = s.logLevel
	}

	logger, err := log.NewFromConfig(cfg.Log)
	if err != nil {
		status <- svc.Status{State: svc.StopPending}
		fmt.Printf("failed to load log config: %v\n", err)
		return false, 1
	}

	logger.Info("client", "version", version.Version)
	logger.Info("Loaded config from", "path", configFile)

	connector, err := connector.New(logger, cfg)
	if err != nil {
		status <- svc.Status{State: svc.StopPending}
		logger.Error("failed to init connector: %w", "error", err)
		return false, 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				logger.Error("client panic", "value", r, "stack", stack)
			}
		}()
		if err := connector.Run(ctx); err != nil {
			logger.Error("error while running client", "error", err)
		}
		cancel()
	}()
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	deadline := 5 * time.Second
	status <- svc.Status{State: svc.Running, Accepts: cmdsAccepted, WaitHint: uint32(deadline.Seconds()) * 1000}
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				logger.Error("context with error closed: %w", "error", ctx.Err())
				return false, 1
			}
			return false, 0
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				status <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				status <- svc.Status{State: svc.StopPending}
				cancel()
			default:
				logger.Error("Unexpected service control request", "request", c)
			}
		}
	}
}
