package bootstrap

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// Command represents the bootstrap command
var Command = &cobra.Command{
	Use:   "bootstrap",
	Short: "bootstrap the client [EXPERIMENTAL]",
	Long:  "bootstrap the client.\n\nUSE WITH CARE",
	Run: func(cmd *cobra.Command, args []string) {
		defaultInstallPath := filepath.ToSlash(installPath())
		defaultConfigPath := filepath.ToSlash(configPath())
		installPath := promptUser("Installation Folder", defaultInstallPath)
		configPath := promptUser("Configuration Folder", defaultConfigPath)

		if err := os.MkdirAll(installPath, 0644); err != nil {
			fmt.Printf("failed to create install folder: %v", err)
			os.Exit(1)
		}

		if err := os.MkdirAll(configPath, 0644); err != nil {
			fmt.Printf("failed to create config folder: %v", err)
			os.Exit(1)
		}

		binary, err := ioutil.ReadFile(os.Args[0])
		if err != nil {
			fmt.Printf("failed to read source binary file: %v", err)
			os.Exit(1)
		}

		binaryFile := filepath.Join(installPath, "myof-client.exe")
		if err := ioutil.WriteFile(binaryFile, binary, 0644); err != nil {
			fmt.Printf("failed to create target binary file: %v", err)
			os.Exit(1)
		}

		username := promptUser("Username", "")
		password := promptUser("Password", "")
		clientCert := promptUserMultiline("Client Certificate")
		logFolder := promptUser("Log Folder", filepath.ToSlash(filepath.Join(configPath, "logs")))

		certFile := filepath.Join(configPath, "client.crt")
		if err := ioutil.WriteFile(certFile, []byte(clientCert), 0644); err != nil {
			fmt.Printf("failed to create client cert file: %v", err)
			os.Exit(1)
		}

		if err := os.MkdirAll(logFolder, 0655); err != nil {
			fmt.Printf("failed to create log folder: %v", err)
			os.Exit(1)
		}

		properties := make(map[string]string)
		properties["username"] = username
		properties["password"] = password
		properties["log.folder"] = filepath.ToSlash(logFolder)
		serviceInstall := strings.ToLower(promptUser("Install Service", "y")) == "y"
		var serviceName string
		if serviceInstall {
			serviceName = promptUser("Service Name", "client")
			serviceName = "myof-" + serviceName
		}

		cfgFile := filepath.Join(configPath, "config.properties")
		f, err := os.Create(cfgFile)
		if err != nil {
			fmt.Printf("failed to open config file: %v", err)
			os.Exit(1)
		}
		defer f.Close()

		var keys []string
		for k := range properties {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(f, "%s = %s\r\n", key, properties[key])
		}

		if serviceInstall {
			cmd := exec.Command(binaryFile, "service", "install", "--config", cfgFile, "--name", serviceName)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Printf("failed to install service: %v", err)
				os.Exit(1)
			}
		}
	},
}

func promptUser(value, defaultValue string) string {
	fmt.Printf("%s [%s]: ", value, defaultValue)
	scanner := bufio.NewScanner(os.Stdin)
	var val string
	if scanner.Scan() {
		val = scanner.Text()
	}
	if val == "" {
		return defaultValue
	}
	return val
}

func promptUserMultiline(value string) string {
	fmt.Printf("%s []: ", value)
	scanner := bufio.NewScanner(os.Stdin)
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func installPath() string {
	switch {
	case runtime.GOOS == "windows":
		return filepath.Join(os.Getenv("ProgramFiles"), "myOpenFactory", "Client")
	case runtime.GOOS == "linux":
		return filepath.Join("opt", "myopenfactory", "client")
	default:
		return ""
	}
}

func configPath() string {
	switch {
	case runtime.GOOS == "windows":
		return filepath.Join(os.Getenv("ProgramData"), "myOpenFactory", "Client")
	case runtime.GOOS == "linux":
		return filepath.Join("etc", "myopenfactory", "client")
	default:
		return ""
	}
}
