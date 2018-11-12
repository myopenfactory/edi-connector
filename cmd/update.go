package cmd

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/spf13/cobra"
)

var certificatePEM = []byte(`-----BEGIN CERTIFICATE-----
MIICMDCCAdagAwIBAgIIaSFNcTwQ5fAwCgYIKoZIzj0EAwIwgY8xCzAJBgNVBAYT
AkRFMQwwCgYDVQQIEwNOUlcxJDAiBgNVBAoTG215T3BlbkZhY3RvcnkgU29mdHdh
cmUgR21iSDEiMCAGA1UEAxMZbXlPcGVuRmFjdG9yeSBEZXZlbG9wbWVudDEoMCYG
CSqGSIb3DQEJARYZc3VwcG9ydEBteW9wZW5mYWN0b3J5LmNvbTAeFw0xODExMDcw
OTE3MDBaFw0yMzExMDcwOTE3MDBaMIGPMQswCQYDVQQGEwJERTEMMAoGA1UECBMD
TlJXMSQwIgYDVQQKExtteU9wZW5GYWN0b3J5IFNvZnR3YXJlIEdtYkgxIjAgBgNV
BAMTGW15T3BlbkZhY3RvcnkgRGV2ZWxvcG1lbnQxKDAmBgkqhkiG9w0BCQEWGXN1
cHBvcnRAbXlvcGVuZmFjdG9yeS5jb20wWTATBgcqhkjOPQIBBggqhkjOPQMBBwNC
AAQogpx+sRI7Cchzt1YdGZ12DgGtnZQIr3/+tUi+OyixE7c2bVL6g2LEt3tmB/tC
M4Zee4LzHZkZUc2RA14hEmfAoxowGDAJBgNVHRMEAjAAMAsGA1UdDwQEAwIHgDAK
BggqhkjOPQQDAgNIADBFAiAfv00cEbCSz8R9p72pb7Qgad+LdtEWU84f4clYgze/
SgIhAOeU4LO4eLRsbPeDsc+uI8Em2Gmy2N6bQ/1vYFZdi0n2
-----END CERTIFICATE-----`)

func init() {
	rootCmd.AddCommand(updateCmd)
}

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "update the executable from github",
	Run: func(cmd *cobra.Command, args []string) {
		if err := preUpdate(); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		block, _ := pem.Decode(certificatePEM)
		if block == nil || block.Type != "CERTIFICATE" {
			fmt.Println("failed to decode PEM block containing certificate")
			os.Exit(1)
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			fmt.Println("failed to parse certificate")
			os.Exit(1)
		}

		pubKey, ok := cert.PublicKey.(*ecdsa.PublicKey)
		if !ok {
			fmt.Println("failed converting pubkey")
			os.Exit(1)
		}

		updater, err := selfupdate.NewUpdater(selfupdate.Config{
			Validator: &selfupdate.ECDSAValidator{PublicKey: pubKey},
		})

		latest, found, err := updater.DetectLatest("myopenfactory/client")
		if err != nil {
			fmt.Println("Error occurred while detecting version:", err)
			os.Exit(1)
		}

		if strings.HasPrefix(Version, "v") {
			Version = Version[1:]
		}

		v := semver.MustParse(Version)
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
		input := strings.ToLower(scanner.Text())
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

		if err != nil {
			fmt.Println("Error occured while creating updater", err)
			os.Exit(1)
		}
		if err := updater.UpdateTo(latest, os.Args[0]); err != nil {
			fmt.Println("Error occurred while updating binary:", err)
			os.Exit(1)
		}
		fmt.Println("Successfully updated to version:", latest.Version)

		if err := postUpdate(); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	},
}
