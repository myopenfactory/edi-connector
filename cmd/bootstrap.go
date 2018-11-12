// +build windows

package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var rootCA = []byte(`-----BEGIN CERTIFICATE-----
MIIExjCCA66gAwIBAgIJAJUA5AJG0Ys6MA0GCSqGSIb3DQEBCwUAMIGJMQswCQYD
VQQGEwJERTEMMAoGA1UECAwDTlJXMSQwIgYDVQQKDBtteU9wZW5GYWN0b3J5IFNv
ZnR3YXJlIEdtYkgxHjAcBgNVBAMMFW15T3BlbkZhY3RvcnkgUm9vdCBDQTEmMCQG
CSqGSIb3DQEJARYXYWRtaW5AbXlvcGVuZmFjdG9yeS5jb20wHhcNMTYwNTE2MDk1
NDM3WhcNNDEwNTEwMDk1NDM3WjCBiTELMAkGA1UEBhMCREUxDDAKBgNVBAgMA05S
VzEkMCIGA1UECgwbbXlPcGVuRmFjdG9yeSBTb2Z0d2FyZSBHbWJIMR4wHAYDVQQD
DBVteU9wZW5GYWN0b3J5IFJvb3QgQ0ExJjAkBgkqhkiG9w0BCQEWF2FkbWluQG15
b3BlbmZhY3RvcnkuY29tMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA
04B+3jJP/8gpw3c7SEBHXKDCkxOFH6NjkEw0C1dHHK67WRA47yxJow2nte31JePv
rtJtJrWL+7e858i0KkVxZ39Nd+T83TB+7swNBrKmlVDXytFVy+Fd3yhnR3piJAjV
I+Wm1M0axr6bQFHuZR9Uyv6W7a5nz+HVRmwwCyeeCiKSGYW7+4wrKnv2LAy+gS6d
82IPGSU13pF7Sj0Y+WmZb6J8es2I6pWEhVDAmQCFBcGPmmuOSjXP2mcI7x7Wgv5u
gu3fwhtH4tIUbsvLMOF4GrIB4vYBfCsLrTViFkx7dW3e/hlFHX5F+BmSoLxMd8OX
qzmn8b2z9PxQyEQonwvFCQIDAQABo4IBLTCCASkwHQYDVR0OBBYEFPNz6mb+EFE8
LuP1OaFdAtWNwkOjMIG+BgNVHSMEgbYwgbOAFPNz6mb+EFE8LuP1OaFdAtWNwkOj
oYGPpIGMMIGJMQswCQYDVQQGEwJERTEMMAoGA1UECAwDTlJXMSQwIgYDVQQKDBtt
eU9wZW5GYWN0b3J5IFNvZnR3YXJlIEdtYkgxHjAcBgNVBAMMFW15T3BlbkZhY3Rv
cnkgUm9vdCBDQTEmMCQGCSqGSIb3DQEJARYXYWRtaW5AbXlvcGVuZmFjdG9yeS5j
b22CCQCVAOQCRtGLOjAMBgNVHRMEBTADAQH/MDkGA1UdHwQyMDAwLqAsoCqGKGh0
dHBzOi8vY2EubXlvcGVuZmFjdG9yeS5jb20vcm9vdF9jYS5jcmwwDQYJKoZIhvcN
AQELBQADggEBAFcO8nf4BRoJl3h00O83FHibnACdQ1i8LKRp2Hy3GMcduCZ5i2BD
D4bUMIMFVg9H8S3wI5adX/XeI0wcRINYB2/MVzFuJIT7xvM8YFNCarMunrLuA8au
Je13FzJSVemxTrF9b3pjkY2RbEMk+PlPWhXn9hknyxPtv0qUFyphrbC7hCbBS26x
2cH6ghwInFw/NWuHbb9aWPlaUOe2/p0IltyVv0fIFIZWLoZi1cto7n+N6C0dQXBp
TuDwjJf1lUs36S0W1vxqfdryRBlnWDHevtfYVOloXpkDVnsZEiB8F5viH2l4h9+b
pn/VbRGrrKzMkF97nfiquISpJ+HwTBAU1TQ=
-----END CERTIFICATE-----`)

func init() {
	rootCmd.AddCommand(bootstrapCmd)
}

// bootstrapCmd represents the bootstrap command
var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "bootstrap the client [EXPERIMENTAL]",
	Long:  "bootstrap the client.\n\nDO NOT USE IN PRODUCTION",
	Run: func(cmd *cobra.Command, args []string) {
		defaultInstallPath := filepath.ToSlash(filepath.Join(os.Getenv("ProgramFiles"), "myOpenFactory", "Client"))
		defaultConfigPath := filepath.ToSlash(filepath.Join(os.Getenv("ProgramData"), "myOpenFactory", "Client"))
		installPath := promptUser("Installation Folder", defaultInstallPath)
		configPath := promptUser("Configuration Folder", defaultConfigPath)

		if err := os.MkdirAll(installPath, 0644); err != nil {
			fmt.Printf("failed to create install folder: %v", err)
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

		caFile := filepath.Join(configPath, "myOpenFactoryCA.crt")
		if err := ioutil.WriteFile(caFile, rootCA, 0644); err != nil {
			fmt.Printf("failed to create target CA file: %v", err)
			os.Exit(1)
		}

		url := promptUser("URL", "https://myopenfactory.net")
		username := promptUser("Username", "")
		password := promptUser("Password", "")
		clientCert := promptUserMultiline("Client Certificate")
		logLevel := promptUser("Log Level", "INFO")
		logFolder := promptUser("Log Folder", filepath.ToSlash(filepath.Join(configPath, "logs")))
		eventlogName := promptUser("Eventlog Name", "")

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
		properties["url"] = url
		properties["username"] = username
		properties["password"] = password
		properties["cafile"] = filepath.ToSlash(caFile)
		properties["clientcert"] = filepath.ToSlash(certFile)
		properties["log.level"] = logLevel
		properties["log.folder"] = filepath.ToSlash(logFolder)
		if eventlogName != "" {
			eventlogName = "myof-" + eventlogName
			properties["log.eventlog"] = eventlogName
		}

		cfgFile := filepath.Join(configPath, "config.properties")
		f, err := os.OpenFile(cfgFile, os.O_RDWR|os.O_TRUNC, 0)
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

		serviceInstall := strings.ToLower(promptUser("Install Service", "y")) == "y"
		if serviceInstall {
			serviceName := promptUser("Service Name", "client")
			serviceName = "myof-" + serviceName
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
