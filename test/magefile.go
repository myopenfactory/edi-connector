// +build mage

package main

import (
	"os"
	"runtime"

	"github.com/magefile/mage/sh"
)

func Test() error {
	defer func() {
		sh.RunV("docker-compose", "-f", "docker-compose.linux.yml", "down", "-v")
		os.Remove("logs")
		os.Remove("inbound")
		os.Remove("outbound")
		os.Remove("success")
		os.Remove("error")
	}()
	os.Mkdir("logs", 0744)
	os.Mkdir("inbound", 0744)
	os.Mkdir("outbound", 0744)
	os.Mkdir("success", 0744)
	os.Mkdir("error", 0744)
	switch runtime.GOOS {
	case "linux":
		return sh.RunV("docker-compose", "-f", "docker-compose.linux.yml", "up", "--no-color", "--build", "--abort-on-container-exit", "--exit-code-from", "test")
	case "windows":
		return sh.RunV("docker-compose", "-f", "docker-compose.windows.yml", "up", "--no-color", "--build", "--abort-on-container-exit", "--exit-code-from", "test")
	}
	return nil
}
