package cmd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/blang/semver"
	"github.com/spf13/cobra"

	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

func init() {
	rootCmd.AddCommand(updateCmd)
}

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "update the executable from github",
	Run: func(cmd *cobra.Command, args []string) {
		latest, found, err := selfupdate.DetectLatest("myopenfactory/client")
		if err != nil {
			fmt.Println("Error occurred while detecting version:", err)
			os.Exit(1)
		}

		v := semver.MustParse(version)
		if !found || latest.Version.Equals(v) {
			fmt.Println("Current version is the latest")
			os.Exit(1)
		}

		fmt.Print("Do you want to update to ", latest.Version, "? (y/n): ")
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			fmt.Println("faild to scan rune")
			os.Exit(1)
		}
		input := scanner.Text()
		if scanner.Err() != nil {
			fmt.Println("Invalid input:", scanner.Err())
			os.Exit(1)
		}

		switch input {
		case "y":
			break
		case "n":
			return
		default:
			fmt.Println("Invalid input:", string(input))
			os.Exit(1)
		}

		if err := selfupdate.UpdateTo(latest.AssetURL, os.Args[0]); err != nil {
			fmt.Println("Error occurred while updating binary:", err)
			os.Exit(1)
		}
		fmt.Println("Successfully updated to version", latest.Version)
	},
}
