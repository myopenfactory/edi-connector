// +build windows

package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	bootstrapURL        string
	bootstrapUsername   string
	bootstrapClientcert string
	bootstrapProxy      string
	bootstrapFolder     string
	serviceInstall      bool
)

func init() {
	rootCmd.AddCommand(bootstrapCmd)

	bootstrapCmd.Flags().StringVar(&bootstrapURL, "url", "https://myopenfactory.net", "portal url")
	setCmd.MarkFlagRequired("url")

	bootstrapCmd.Flags().StringVar(&bootstrapUsername, "username", "", "username")
	setCmd.MarkFlagRequired("username")

	bootstrapCmd.Flags().StringVar(&bootstrapClientcert, "clientcert", "certificate.pem", "file for client certificate")
	setCmd.MarkFlagRequired("clientcert")

	bootstrapCmd.Flags().StringVar(&bootstrapProxy, "proxy", "", "proxy url")
	bootstrapCmd.Flags().StringVar(&bootstrapFolder, "log", os.Getenv("ProgramData")+"/myOpenFactory/client/logs", "folder for logs")
	setCmd.MarkFlagRequired("log")

	bootstrapCmd.Flags().BoolVar(&serviceInstall, "service", true, "install as service?")

	setCmd.MarkFlagRequired("key")
}

// bootstrapCmd represents the bootstrap command
var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "shows version",
	Run: func(cmd *cobra.Command, args []string) {
		folder := os.Getenv("ProgramFiles") + "\\myOpenFactory\\Client\\"
		if err := os.MkdirAll(folder, 0644); err != nil {
			fmt.Printf("failed to create folder: %v", err)
			os.Exit(1)
		}
		command := exec.Command("setx", "/M", "PATH", fmt.Sprintf("\"%%PATH%%;%s\"", folder))
		if err := command.Run(); err != nil {
			fmt.Printf("failed to update PATH var: %v", err)
			os.Exit(1)
		}

		destinationFile := folder + "myof-client.exe"
		input, err := ioutil.ReadFile(os.Args[0])
		if err != nil {
			fmt.Printf("failed to read source file")
			os.Exit(1)
		}
		err = ioutil.WriteFile(destinationFile, input, 0644)
		if err != nil {
			fmt.Printf("failed to create target file: %q", destinationFile)
			os.Exit(1)
		}

		destCertFile := folder + "cert.pem"
		input, err = ioutil.ReadFile(bootstrapClientcert)
		if err != nil {
			fmt.Printf("failed to read client certificate file: %q", bootstrapClientcert)
			os.Exit(1)
		}
		err = ioutil.WriteFile(destCertFile, input, 0644)
		if err != nil {
			fmt.Printf("failed to create target file: %q", destCertFile)
			os.Exit(1)
		}

		input, err = ioutil.ReadFile("ca.pem")
		if err != nil {
			fmt.Printf("failed to read ca file: %q", bootstrapClientcert)
			os.Exit(1)
		}
		cafile := fmt.Sprintf("%s\\%s", filepath.Base(cfgFile), "ca.pem")
		err = ioutil.WriteFile(cafile, input, 0644) //TODO input load
		if err != nil {
			fmt.Printf("failed to create target file: %q", cafile)
			os.Exit(1)
		}

		f, err := os.OpenFile(cfgFile, os.O_RDWR, 0644)
		if err != nil {
			fmt.Printf("failed to create target file: %q", cfgFile)
			os.Exit(1)
		}
		defer f.Close()

		fmt.Fprintf(f, "cafile = %s\n", cafile)
		fmt.Fprintf(f, "url = %s\n", bootstrapURL)
		fmt.Fprintf(f, "username = %s\n", bootstrapUsername)
		fmt.Fprintf(f, "password = %s\n", readLine("Password: "))
		fmt.Fprintf(f, "clientcert = %s\n", destCertFile)
		if bootstrapProxy != "" {
			fmt.Fprintf(f, "proxy = %s\n", bootstrapProxy)
		}
		fmt.Fprintf(f, "log.folder = %s\n", bootstrapFolder)

		if serviceInstall {
			command = exec.Command(destinationFile, "service", "install", "--config", cfgFile)
			if err := command.Run(); err != nil {
				fmt.Printf("failed to install service: %v", err)
				os.Exit(1)
			}
		}
	},
}

func readLine(text string) string {
	fmt.Print(text)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return scanner.Text()
	}
	if scanner.Err() != nil {
		fmt.Printf("failed to read line: %v", scanner.Err())
		os.Exit(1)
	}
	return ""
}
