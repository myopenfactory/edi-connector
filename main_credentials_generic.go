//go:build !windows

package main

import "fmt"

func credentialsCommand(args []string) error {
	return fmt.Errorf("the credentials command is only available on windows")
}
