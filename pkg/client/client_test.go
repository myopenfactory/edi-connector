// +build windows

package client

import (
	"bytes"
	"context"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"

	"testing"
	"time"

	"gotest.tools/fs"

	"github.com/pkg/errors"

	"gotest.tools/assert"

	"github.com/google/go-cmp/cmp/cmpopts"
	pb "github.com/myopenfactory/client/api"
)

var rsaKeyPEM = `-----BEGIN RSA PRIVATE KEY-----
MIIBOwIBAAJBANLJhPHhITqQbPklG3ibCVxwGMRfp/v4XqhfdQHdcVfHap6NQ5Wo
k/4xIA+ui35/MmNartNuC+BdZ1tMuVCPFZcCAwEAAQJAEJ2N+zsR0Xn8/Q6twa4G
6OB1M1WO+k+ztnX/1SvNeWu8D6GImtupLTYgjZcHufykj09jiHmjHx8u8ZZB/o1N
MQIhAPW+eyZo7ay3lMz1V01WVjNKK9QSn1MJlb06h/LuYv9FAiEA25WPedKgVyCW
SmUwbPw8fnTcpqDWE3yTO3vKcebqMSsCIBF3UmVue8YU3jybC3NxuXq3wNm34R8T
xVLHwDXh/6NJAiEAl2oHGGLz64BuAfjKrqwz7qMYr9HCLIe/YsoWq/olzScCIQDi
D2lWusoe2/nEqfDVVWGWlyJ7yOmqaVm/iNUN9B2N2g==
-----END RSA PRIVATE KEY-----
`

var rsaCertPEM = `-----BEGIN CERTIFICATE-----
MIIB0zCCAX2gAwIBAgIJAI/M7BYjwB+uMA0GCSqGSIb3DQEBBQUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwHhcNMTIwOTEyMjE1MjAyWhcNMTUwOTEyMjE1MjAyWjBF
MQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50
ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMFwwDQYJKoZIhvcNAQEBBQADSwAwSAJBANLJ
hPHhITqQbPklG3ibCVxwGMRfp/v4XqhfdQHdcVfHap6NQ5Wok/4xIA+ui35/MmNa
rtNuC+BdZ1tMuVCPFZcCAwEAAaNQME4wHQYDVR0OBBYEFJvKs8RfJaXTH08W+SGv
zQyKn0H8MB8GA1UdIwQYMBaAFJvKs8RfJaXTH08W+SGvzQyKn0H8MAwGA1UdEwQF
MAMBAf8wDQYJKoZIhvcNAQEFBQADQQBJlffJHybjDGxRMqaRmDhX0+6v02TUKZsW
r5QuVbpQhH6u+0UgcW0jp9QwpxoPTLTWGXEWBBBurxFwiCBhkQ+V
-----END CERTIFICATE-----
`
var rsaCAPem = `-----BEGIN CERTIFICATE-----
MIIB+zCCAWQCCQCr7saE51adXjANBgkqhkiG9w0BAQUFADBCMQswCQYDVQQGEwJY
WDEVMBMGA1UEBwwMRGVmYXVsdCBDaXR5MRwwGgYDVQQKDBNEZWZhdWx0IENvbXBh
bnkgTHRkMB4XDTE4MDQxNjExMzQyOFoXDTE5MDQxNjExMzQyOFowQjELMAkGA1UE
BhMCWFgxFTATBgNVBAcMDERlZmF1bHQgQ2l0eTEcMBoGA1UECgwTRGVmYXVsdCBD
b21wYW55IEx0ZDCBnzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEAwpT9KTqE1NbR
ZSW8Gp3vm6fULkRQyo/K8Ssyyv2W+ujKVrs3LtnJ7Gzds1QcccxQgGinN71naoMt
jBERPGBXD7s7PYTllYCYk2rTNlaURm3Rfa/TuK8hyc5aDnkDB6PFootuaLlEsz5w
nMSrltn1rSsfqVIV/8PlChmIS61voNcCAwEAATANBgkqhkiG9w0BAQUFAAOBgQCB
qI9aWrEJ68fjI6YItqpeUaAxNoI3r27Crcn7zYAVHVnXnPMjGyCT5aWEMvur7XID
AcjUPrdZG/dYpQBq+vl3+e8ojTBv+R1/m40NMjNJgVLs/oDG4zxDl+FQi8sN9uOi
RgRvuuo/3ot784iSQ6/rGW+5RSnAATcqoRF3DyJGPQ==
-----END CERTIFICATE-----`

func Test_createHTTPClient(t *testing.T) {
	type args struct {
		clientpem string
		capem     string
		proxyURL  string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:    "NilInput",
			want:    true,
			wantErr: false,
		},
		{
			name: "ClientPemWrong",
			args: args{
				clientpem: "WRONGINPPUT",
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "ClientPemRight",
			args: args{
				clientpem: rsaKeyPEM + rsaCertPEM,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "CAPemWrongFile",
			args: args{
				capem: `WRONGPEM`,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "CAPemRight",
			args: args{
				capem: rsaCAPem,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Proxy",
			args: args{
				proxyURL: `http://localhost:1080`,
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createHTTPClient(tt.args.clientpem, tt.args.capem, tt.args.proxyURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("createHTTPClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (got != nil) != tt.want {
				t.Errorf("createHTTPClient() = %v, want %v", got != nil, tt.want)
			}
		})
	}
}

const RESPONSE_STRING = "OK"

func Test_createHTTPClientWithServer(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, RESPONSE_STRING)
	}))
	defer s.Close()
	cert := s.Certificate()
	certPem := bytes.NewBufferString("")
	pem.Encode(certPem, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	loadHttp(s, certPem.String(), t)
	file := fs.NewFile(t, "client", fs.WithBytes(certPem.Bytes()))
	defer file.Remove()
	loadHttp(s, file.Path(), t)
}

func loadHttp(s *httptest.Server, ca string, t *testing.T) {
	client, err := createHTTPClient("", ca, "")
	if err != nil {
		t.Fatalf("cannot create httpclient: %v", err)
	}
	resp, err := client.Get(s.URL)
	if err != nil {
		t.Fatalf("get is not working: %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if string(body) != RESPONSE_STRING {
		t.Fatalf("response wrong, want %q got %q", RESPONSE_STRING, string(body))
	}
}

func TestWithUsername(t *testing.T) {
	c := &Client{}
	WithUsername("test")(c)
	if c.Username != "test" {
		t.Error("username not set")
	}
}
func TestWithPassword(t *testing.T) {
	c := &Client{}
	WithPassword("test")(c)
	if c.Password != "test" {
		t.Error("password not set")
	}
}

func TestWithURL(t *testing.T) {
	c := &Client{}
	WithURL("test")(c)
	if c.URL != "test" {
		t.Error("url not set")
	}
}

func TestWithProxy(t *testing.T) {
	c := &Client{}
	WithProxy("test")(c)
	if c.ProxyURL != "test" {
		t.Error("proxy url not set")
	}
}

func TestWithCA(t *testing.T) {
	c := &Client{}
	WithCA("test")(c)
	if c.CA != "test" {
		t.Error("ca not set")
	}
}

func TestWithCert(t *testing.T) {
	c := &Client{}
	WithCert("test")(c)
	if c.ClientCert != "test" {
		t.Error("client cert not set")
	}
}

func TestWithRunWaitTime(t *testing.T) {
	c := &Client{}
	WithRunWaitTime(5 * time.Second)(c)
	if c.RunWaitTime != 5*time.Second {
		t.Error("run wait time not set")
	}
}

func TestWithHealthWaitTime(t *testing.T) {
	c := &Client{}
	WithHealthWaitTime(5 * time.Second)(c)
	if c.HealthWaitTime != 5*time.Second {
		t.Error("health wait time not set")
	}
}

func TestWithClient(t *testing.T) {
	c := &Client{}
	cl := &http.Client{}
	WithClient(cl)(c)
	if c.client != cl {
		t.Error("client not set")
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		opts    []Option
		want    *Client
		wantErr bool
	}{
		{
			name:    "NilInput",
			wantErr: false,
			want: &Client{
				RunWaitTime:    defaultRunWaitTime,
				HealthWaitTime: defaultHealthWaitTime,
			},
		},
		{
			name: "WithOption",
			id:   "id",
			opts: []Option{
				WithUsername("user"),
				WithPassword("password"),
				WithURL("url"),
			},
			want: &Client{
				ID:             "id",
				Username:       "user",
				Password:       "password",
				URL:            "url",
				RunWaitTime:    defaultRunWaitTime,
				HealthWaitTime: defaultHealthWaitTime,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.id, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.DeepEqual(t, got, tt.want, cmpopts.IgnoreUnexported(Client{}))
		})
	}
}

func Test_Client_Run(t *testing.T) {
	test := &testServer{}
	server := httptest.NewServer(pb.NewClientServiceServer(test, nil))

	log.Println(server.URL)

	dir := fs.NewDir(t, "client")
	defer dir.Remove()

	outDir := fs.NewDir(t, "client", fs.WithFile("test", "Hello World"))
	defer outDir.Remove()

	tests := []struct {
		name    string
		client  *Client
		ret     testReturns
		wantErr bool
	}{
		{
			name:    "NilInput",
			wantErr: true,
			client:  &Client{},
		},
		{
			name: "ServerNotAvailable",
			client: &Client{
				ID:       "id",
				Username: "username",
				Password: "password",
				URL:      "http://localhost",
			},
			wantErr: true,
		},
		{
			name: "FailListConfig",
			client: &Client{
				ID:       "id",
				Username: "username",
				Password: "password",
				URL:      server.URL,
			},
			wantErr: true,
		},
		{
			name: "NoBaseFolder",
			client: &Client{
				ID:          "id",
				Username:    "username",
				Password:    "password",
				URL:         server.URL,
				RunWaitTime: 500 * time.Millisecond,
			},
			ret: testReturns{
				configs: &pb.Configuration{
					Inbound: []*pb.ProcessConfig{
						&pb.ProcessConfig{
							ProcessId: "4900",
							Type:      "FILE",
							Parameter: map[string]string{},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "NoProcessId",
			client: &Client{
				ID:          "id",
				Username:    "username",
				Password:    "password",
				URL:         server.URL,
				RunWaitTime: 500 * time.Millisecond,
			},
			ret: testReturns{
				configs: &pb.Configuration{
					Outbound: []*pb.ProcessConfig{
						&pb.ProcessConfig{
							ProcessId: "",
							Type:      "FILE",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "ListMessagesError",
			client: &Client{
				ID:          "id",
				Username:    "username",
				Password:    "password",
				URL:         server.URL,
				client:      &http.Client{},
				ctx:         context.Background(),
				done:        make(chan struct{}, 1),
				RunWaitTime: 500 * time.Millisecond,
			},
			ret: testReturns{
				configs: &pb.Configuration{
					Inbound: []*pb.ProcessConfig{
						&pb.ProcessConfig{
							ProcessId: "4900",
							Type:      "FILE",
							Parameter: map[string]string{
								"basefolder": "/",
							},
						},
					},
				},
				Messages:      nil,
				MessagesError: errors.New("endpoint not available"),
			},
			wantErr: false,
		},
		{
			name: "InboundMessagesFailConfirm",
			client: &Client{
				ID:          "id",
				Username:    "username",
				Password:    "password",
				URL:         server.URL,
				client:      &http.Client{},
				ctx:         context.Background(),
				done:        make(chan struct{}, 1),
				RunWaitTime: 500 * time.Millisecond,
			},
			ret: testReturns{
				configs: &pb.Configuration{
					Inbound: []*pb.ProcessConfig{
						&pb.ProcessConfig{
							ProcessId: "4900",
							Type:      "FILE",
							Parameter: map[string]string{
								"basefolder": dir.Path(),
							},
						},
					},
				},
				Messages: &pb.Messages{
					Messages: []*pb.Message{
						&pb.Message{
							ProcessId: "4900",
							Test:      false,
							Id:        "4711",
							Content:   []byte("Hello World"),
							Properties: map[string]string{
								"filename": "Testfile.txt",
							},
						},
					},
				},
				confirmError: errors.New("expected error"),
			},
			wantErr: false,
		},
		{
			name: "InboundMessagesWithoutFilename",
			client: &Client{
				ID:          "id",
				Username:    "username",
				Password:    "password",
				URL:         server.URL,
				client:      &http.Client{},
				ctx:         context.Background(),
				done:        make(chan struct{}, 1),
				RunWaitTime: 500 * time.Millisecond,
			},
			ret: testReturns{
				configs: &pb.Configuration{
					Inbound: []*pb.ProcessConfig{
						&pb.ProcessConfig{
							ProcessId: "4900",
							Type:      "FILE",
							Parameter: map[string]string{
								"basefolder": dir.Path(),
							},
						},
					},
				},
				Messages: &pb.Messages{
					Messages: []*pb.Message{
						&pb.Message{
							ProcessId:  "4900",
							Test:       false,
							Id:         "4711",
							Content:    []byte("Hello World"),
							Properties: map[string]string{},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "PushMessage",
			client: &Client{
				ID:          "id",
				Username:    "username",
				Password:    "password",
				URL:         server.URL,
				client:      &http.Client{},
				ctx:         context.Background(),
				done:        make(chan struct{}, 1),
				RunWaitTime: 500 * time.Millisecond,
			},
			ret: testReturns{
				configs: &pb.Configuration{
					Outbound: []*pb.ProcessConfig{
						&pb.ProcessConfig{
							ProcessId: "4900",
							Type:      "FILE",
							Parameter: map[string]string{
								"waittime":     "1",
								"folder.first": outDir.Path(),
							},
						},
					},
				},
				addM: &pb.Confirm{},
			},
			wantErr: false,
		},
		{
			name: "PushAttachment",
			client: &Client{
				ID:          "id",
				Username:    "username",
				Password:    "password",
				URL:         server.URL,
				client:      &http.Client{},
				ctx:         context.Background(),
				done:        make(chan struct{}, 1),
				RunWaitTime: 500 * time.Millisecond,
			},
			ret: testReturns{
				configs: &pb.Configuration{
					Outbound: []*pb.ProcessConfig{
						&pb.ProcessConfig{
							ProcessId: "4900",
							Type:      "FILE",
							Parameter: map[string]string{
								"waittime":               "1",
								"attachmentfolder.first": outDir.Path(),
							},
						},
					},
				},
				addA: &pb.Confirm{},
			},
			wantErr: false,
		},

		{
			name: "HealthError",
			client: &Client{
				ID:             "id",
				Username:       "username",
				Password:       "password",
				URL:            server.URL,
				client:         &http.Client{},
				done:           make(chan struct{}, 1),
				HealthWaitTime: 100 * time.Millisecond,
			},
			ret: testReturns{
				configs:   &pb.Configuration{},
				addH:      nil,
				addHError: fmt.Errorf("error test"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			c.ctx, c.cancel = context.WithCancel(context.Background())
			go func() {
				<-time.After(1 * time.Second)
				ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
				c.Shutdown(ctx)
			}()

			test.ret = tt.ret
			err := c.Run()
			log.Println(err)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckParams(t *testing.T) {
	tests := []struct {
		name    string
		client  *Client
		wantErr bool
		want    *Client
	}{
		{
			name:    "NilInput",
			wantErr: true,
			want:    nil,
		},
		{
			name:    "NilID",
			client:  &Client{},
			wantErr: true,
			want:    nil,
		},
		{
			name: "NilUsername",
			client: &Client{
				ID: "id",
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "NilPassword",
			client: &Client{
				ID:       "id",
				Username: "username",
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "NilURL",
			client: &Client{
				ID:       "id",
				Username: "username",
				Password: "password",
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "NilWaitTime",
			client: &Client{
				ID:       "id",
				Username: "username",
				Password: "password",
				URL:      "http://localhost",
			},
			wantErr: false,
			want: &Client{
				ID:             "id",
				Username:       "username",
				Password:       "password",
				URL:            "http://localhost",
				RunWaitTime:    defaultRunWaitTime,
				HealthWaitTime: defaultHealthWaitTime,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkParams(tt.client); (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type testReturns struct {
	configs       *pb.Configuration
	configError   error
	Messages      *pb.Messages
	MessagesError error
	addM          *pb.Confirm
	addMError     error
	addA          *pb.Confirm
	addAError     error
	confirmM      *pb.Empty
	confirmError  error
	addH          *pb.Empty
	addHError     error
}

type testServer struct {
	ret testReturns
}

func (t *testServer) ListConfigs(context.Context, *pb.Empty) (*pb.Configuration, error) {
	return t.ret.configs, t.ret.configError
}

func (t *testServer) AddAttachment(context.Context, *pb.Attachment) (*pb.Confirm, error) {
	return t.ret.addA, t.ret.addAError
}

func (t *testServer) AddMessage(context.Context, *pb.Message) (*pb.Confirm, error) {
	return t.ret.addM, t.ret.addMError
}

func (t *testServer) ListMessages(context.Context, *pb.Empty) (*pb.Messages, error) {
	return t.ret.Messages, t.ret.MessagesError
}

func (t *testServer) ConfirmMessage(context.Context, *pb.Confirm) (*pb.Empty, error) {
	return t.ret.confirmM, t.ret.confirmError
}

func (t *testServer) AddHealth(context.Context, *pb.HealthInfo) (*pb.Empty, error) {
	return t.ret.addH, t.ret.addHError
}
