//+build wireinject

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/wire"
	"github.com/myopenfactory/client/pkg/client"
	"github.com/myopenfactory/client/pkg/errors"
	"github.com/myopenfactory/client/pkg/log"
	"github.com/myopenfactory/client/pkg/version"
	"github.com/spf13/viper"
)

var rootCA = `-----BEGIN CERTIFICATE-----
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
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIDdTCCAl2gAwIBAgILBAAAAAABFUtaw5QwDQYJKoZIhvcNAQEFBQAwVzELMAkG
A1UEBhMCQkUxGTAXBgNVBAoTEEdsb2JhbFNpZ24gbnYtc2ExEDAOBgNVBAsTB1Jv
b3QgQ0ExGzAZBgNVBAMTEkdsb2JhbFNpZ24gUm9vdCBDQTAeFw05ODA5MDExMjAw
MDBaFw0yODAxMjgxMjAwMDBaMFcxCzAJBgNVBAYTAkJFMRkwFwYDVQQKExBHbG9i
YWxTaWduIG52LXNhMRAwDgYDVQQLEwdSb290IENBMRswGQYDVQQDExJHbG9iYWxT
aWduIFJvb3QgQ0EwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDaDuaZ
jc6j40+Kfvvxi4Mla+pIH/EqsLmVEQS98GPR4mdmzxzdzxtIK+6NiY6arymAZavp
xy0Sy6scTHAHoT0KMM0VjU/43dSMUBUc71DuxC73/OlS8pF94G3VNTCOXkNz8kHp
1Wrjsok6Vjk4bwY8iGlbKk3Fp1S4bInMm/k8yuX9ifUSPJJ4ltbcdG6TRGHRjcdG
snUOhugZitVtbNV4FpWi6cgKOOvyJBNPc1STE4U6G7weNLWLBYy5d4ux2x8gkasJ
U26Qzns3dLlwR5EiUWMWea6xrkEmCMgZK9FGqkjWZCrXgzT/LCrBbBlDSgeF59N8
9iFo7+ryUp9/k5DPAgMBAAGjQjBAMA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMBAf8E
BTADAQH/MB0GA1UdDgQWBBRge2YaRQ2XyolQL30EzTSo//z9SzANBgkqhkiG9w0B
AQUFAAOCAQEA1nPnfE920I2/7LqivjTFKDK1fPxsnCwrvQmeU79rXqoRSLblCKOz
yj1hTdNGCbM+w6DjY1Ub8rrvrTnhQ7k4o+YviiY776BQVvnGCv04zcQLcFGUl5gE
38NflNUVyRRBnMRddWQVDf9VMOyGj/8N7yy5Y0b2qvzfvGn9LhJIZJrglfCm7ymP
AbEVtQwdpf5pLGkkeB6zpxxxYu7KyJesF12KwvhHhm4qxFYxldBniYUr+WymXUad
DKqC5JlR3XC321Y9YeRq4VzW9v493kHMB65jUr9TU/Qr6cf9tveCX4XSQRjbgbME
HMUfpIBvFSDJ3gyICh3WZlXi/EjJKSZp4A==
-----END CERTIFICATE-----`

func InitializeClient() (*client.Client, error) {
	wire.Build(InitializeLogger, provideClientID, provideOptions, client.New)
	return &client.Client{}, nil
}

func InitializeLogger() *log.Logger {
	wire.Build(provideLogOptions, log.New)
	return &log.Logger{}
}

func provideOptions() ([]client.Option, error) {
	const op errors.Op = "cmd.provideOptions"
	var err error

	runWaitTimeDuration := time.Minute
	runWaitTime := viper.GetString("runwaittime")
	if runWaitTime != "" {
		runWaitTimeDuration, err = time.ParseDuration(runWaitTime)
		if err != nil {
			return nil, err
		}
	}

	healthWaitTimeDuration := 15 * time.Minute
	healthWaitTime := viper.GetString("healthwaitttime")
	if healthWaitTime != "" {
		healthWaitTimeDuration, err = time.ParseDuration(healthWaitTime)
		if err != nil {
			return nil, err
		}
	}

	url := viper.GetString("url")
	if url == "" {
		url = "https://myopenfactory.net"
	}

	cafile := viper.GetString("cafile")
	if cafile == "" {
		cafile = rootCA
	}

	clientcert := viper.GetString("clientcert")
	if clientcert == "" {
		clientcert = filepath.Join(filepath.Dir(viper.ConfigFileUsed()), "client.crt")
	}
	if _, err := os.Stat(clientcert); os.IsNotExist(err) {
		return nil, errors.E(op, "client certificate does not exist", errors.KindUnexpected)
	}

	proxy := viper.GetString("proxy")
	if proxy == "" {
		proxy = os.Getenv("HTTP_PROXY")
	}

	return []client.Option{
		client.WithUsername(viper.GetString("username")),
		client.WithPassword(viper.GetString("password")),
		client.WithURL(url),
		client.WithCA(cafile),
		client.WithCert(clientcert),
		client.WithProxy(proxy),
		client.WithHealthWaitTime(healthWaitTimeDuration),
		client.WithRunWaitTime(runWaitTimeDuration),
	}, nil
}

func provideLogOptions() []log.Option {
	opts := []log.Option{}

	logLevel := viper.GetString("log.level")
	if logLevel == "" {
		logLevel = "INFO"
	}
	opts = append(opts, log.WithLevel(logLevel))

	logSyslog := viper.GetString("log.syslog")
	if logSyslog != "" {
		opts = append(opts, log.WithSyslog(logSyslog))
	}

	eventLog := viper.GetString("log.eventlog")
	if eventLog != "" {
		opts = append(opts, log.WithEventlog(eventLog))
	}

	logMailHost := viper.GetString("log.mail.host")
	if logMailHost != "" {
		addr := fmt.Sprintf("%s:%d", logMailHost, viper.GetInt("log.mail.port"))
		logMailFrom := viper.GetString("log.mail.from")
		logMailTo := viper.GetString("log.mail.to")
		logMailUsername := viper.GetString("log.mail.username")
		logMailPassword := viper.GetString("log.mail.password")
		opts = append(opts, log.WithMail("myOpenFactory Client", addr, logMailFrom, logMailTo, logMailUsername, logMailPassword))
	}

	logFolder := viper.GetString("log.folder")
	if logFolder != "" {
		opts = append(opts, log.WithFilesystem(logFolder))
	}

	return opts
}

func provideClientID() string {
	return fmt.Sprintf("Core_%s", version.Version)
}
