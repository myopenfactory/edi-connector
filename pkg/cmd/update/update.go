package update

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/blang/semver"
	clientpkg "github.com/myopenfactory/client/pkg/client"
	"github.com/myopenfactory/client/pkg/config"
	log "github.com/myopenfactory/client/pkg/log"
	versionpkg "github.com/myopenfactory/client/pkg/version"
	"github.com/spf13/cobra"
)

// Command represents the update command
var Command = &cobra.Command{
	Use:      "update",
	Short:    "update the executable from github",
	PreRunE:  preUpdate,
	PostRunE: postUpdate,
	Run: func(cmd *cobra.Command, args []string) {
		logger := log.New(config.ParseLogOptions())

		release, err := clientpkg.Release()
		if err != nil {
			logger.Error(err)
			os.Exit(1)
		}

		version, err := semver.ParseTolerant(versionpkg.Version)
		if release.Version.Equals(version) {
			logger.Errorf("current version is the latest")
			os.Exit(0)
		}

		fmt.Print("Do you want to update to ", release.Version, "? (y/n): ")
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			logger.Error("failed to scan rune")
			os.Exit(1)
		}
		input := strings.ToLower(scanner.Text())
		if scanner.Err() != nil {
			logger.Errorf("failed to read input: %v", scanner.Err())
			os.Exit(1)
		}

		switch input {
		case "y":
			break
		case "n":
			os.Exit(0)
		default:
			logger.Errorf("Invalid input: %s", input)
			os.Exit(1)
		}

		if err := clientpkg.Update(release); err != nil {
			logger.Errorf("failed to update client: %v", err)
			os.Exit(1)
		}

		logger.Println("Successfully updated to version:", release.Version)

	},
}
