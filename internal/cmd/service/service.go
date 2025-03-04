//go:build !windows

package service

import "fmt"

func Run(args []string) error {
	return fmt.Errorf("service not supported on linux")
}
