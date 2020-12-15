package client

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

var certificatePEM = `-----BEGIN CERTIFICATE-----
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
-----END CERTIFICATE-----`

var updater *selfupdate.Updater

func init() {
	block, _ := pem.Decode([]byte(certificatePEM))
	if block == nil || block.Type != "CERTIFICATE" {
		panic("failed to decode PEM block containing certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		panic("failed to parse certificate")
	}

	publicKey, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		panic("failed converting to public key")
	}

	updater, err = selfupdate.NewUpdater(selfupdate.Config{
		Validator: &selfupdate.ECDSAValidator{PublicKey: publicKey},
	})
	if err != nil {
		panic("failed to create selfupdater")
	}
}

func Release() (*selfupdate.Release, error) {
	release, _, err := updater.DetectLatest("myopenfactory/client")
	if err != nil {
		return nil, fmt.Errorf("failed to detect latest release: %w", err)
	}

	return release, nil
}

func Update(release *selfupdate.Release) error {
	if err := updater.UpdateTo(release, os.Args[0]); err != nil {
		return fmt.Errorf("failed updating client to version %s: %w", release.Version, err)
	}

	return nil
}
