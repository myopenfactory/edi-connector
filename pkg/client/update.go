package client

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"os"

	"github.com/pkg/errors"
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

func Release() (*selfupdate.Release, error) {
	updater := selfupdate.DefaultUpdater()

	release, _, err := updater.DetectLatest("myopenfactory/client")
	if err != nil {
		return nil, errors.Wrap(err, "failed to detect latest release")
	}

	return release, nil
}

func Update(release *selfupdate.Release) error {
	block, _ := pem.Decode([]byte(certificatePEM))
	if block == nil || block.Type != "CERTIFICATE" {
		return errors.New("failed to decode PEM block containing certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return errors.Wrap(err, "failed to parse certificate")
	}

	pubKey, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return errors.New("failed converting pubkey")
	}

	var updater *selfupdate.Updater
	updater, err = selfupdate.NewUpdater(selfupdate.Config{
		Validator: &selfupdate.ECDSAValidator{PublicKey: pubKey},
	})

	if err := updater.UpdateTo(release, os.Args[0]); err != nil {
		return errors.Wrapf(err, "failed updating client to version %s", release.Version)
	}

	return nil
}
