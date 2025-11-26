//go:build !windows

package main

import "fmt"

func serviceRun(configFile string, logLevel string) error {
	return fmt.Errorf("no windows service on linux")
}

func isWindowsService() bool {
	return false
}
