package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/magiconair/properties"
	"github.com/myopenfactory/client/api"
	"github.com/twitchtv/twirp"
)

var messageTpl = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Message xmlns="http://myopenfactory.net/myopenfactory50/">
    <Body>
        <Companies>
            <Company>
                <CompanyID>client.myopenfactory.test</CompanyID>
                <Name>myOpenFactory Client Test</Name>
            </Company>
            <Company>
                <CompanyID>myopenfactory.test</CompanyID>
                <Name>myOpenFactory DevOp Test</Name>
            </Company>
        </Companies>
        <Items>
            <Item>
                <Deliveries>
                    <Delivery>
                        <Quantity>10.0</Quantity>
                    </Delivery>
                </Deliveries>
                <ItemID>1</ItemID>
                <Unit>PCE</Unit>
            </Item>
        </Items>
	</Body>
	<Subject>MIRROR</Subject>
    <MessageID>%s</MessageID>
    <ReceiverID>myopenfactory.test</ReceiverID>
    <SenderID>client.myopenfactory.test</SenderID>
    <TypeID>ORDER</TypeID>
</Message>`

type httpClient struct {
	username string
	password string
	http     *http.Client
}

func (c *httpClient) Do(req *http.Request) (*http.Response, error) {
	data := base64.StdEncoding.EncodeToString([]byte(c.username + ":" + c.password))
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s", data))
	return c.http.Do(req)
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func main() {
	var cfgFile string
	flag.StringVar(&cfgFile, "config", "", "config file")
	flag.Parse()

	if cfgFile == "" {
		switch {
		case runtime.GOOS == "windows":
			cfgFile = os.Getenv("ProgramData") + "/myOpenFactory/client/config.properties"
		case runtime.GOOS == "linux":
			cfgFile = "/etc/myof-client/config.properties"
		}
	}

	p, err := properties.LoadFile(cfgFile, properties.UTF8)
	if err != nil {
		fmt.Printf("failed to load config file: %s", err)
		os.Exit(1)
	}

	tlsConfig := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	cert := p.GetString("clientcert", "")
	if strings.HasPrefix(cert, "./") {
		cert = filepath.Join(filepath.Dir(cfgFile), cert)
	}
	cc, err := tls.LoadX509KeyPair(cert, cert)
	if err != nil {
		fmt.Printf("error while loading client certificate: %v", err)
		os.Exit(1)
	}
	if len(cc.Certificate) > 0 {
		tlsConfig.Certificates = append(tlsConfig.Certificates, cc)
	}

	ca := p.GetString("cafile", "")
	if ca != "" {
		pem, err := ioutil.ReadFile(ca)
		if err != nil {
			fmt.Printf("error while loading ca certificates: %v", err)
			os.Exit(1)
		}

		certs := x509.NewCertPool()
		certs.AppendCertsFromPEM(pem)
		tlsConfig.RootCAs = certs
		tlsConfig.BuildNameToCertificate()
	}

	cl := &httpClient{
		http: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		},
		username: p.GetString("username", "username"),
		password: p.GetString("password", "password"),
	}
	client := api.NewClientServiceProtobufClient(p.GetString("url", "https://myopenfactory.net"), cl, twirp.WithClientPathPrefix("/v1"))
	res, err := client.ListConfigs(context.Background(), &api.Empty{})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	outboundPath := res.Outbound[0].Parameter["folder.first"]
	err = ioutil.WriteFile(filepath.Join(outboundPath, "message.xml"), []byte(fmt.Sprintf(messageTpl, time.Now().Format(time.RFC3339))), 0666)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	attachmentPath := res.Outbound[0].Parameter["attachmentfolder.first"]
	err = ioutil.WriteFile(filepath.Join(attachmentPath, "attachment.sample"), []byte(fmt.Sprintf("%s", time.Now().Format(time.RFC3339))), 0666)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	inboundPath := res.Inbound[0].Parameter["basefolder"]

	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			fmt.Println("Timed out!")
			return
		case <-ticker.C:
			files, err := ioutil.ReadDir(inboundPath)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if len(files) > 0 {
				attachments, err := ioutil.ReadDir(attachmentPath)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				if len(attachments) > 0 {
					fmt.Println("attachment not uploaded")
					os.Exit(1)
				}

				for _, file := range files {
					log.Println(file.Name())
				}
				return
			}
		}
	}
}
