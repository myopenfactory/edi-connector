package platform

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"

	"github.com/myopenfactory/edi-connector/v2/version"
)

type MessageAttachment struct {
	Url    string `json:"url"`
	ItemId string `json:"item_id"`
}

type Transmission struct {
	Id   string `json:"id"`
	Url  string `json:"url"`
	Hash struct {
		Method string `json:"method"`
		Sum    string `json:"sum"`
	} `json:"hash"`
	Test     bool `json:"test"`
	Metadata map[string]string
}

type Client struct {
	http    *http.Client
	baseUrl string
}

func NewClient(baseUrl string, username string, password string, certFile string, caFile string, proxy string) (*Client, error) {
	httpTransport := http.DefaultTransport
	if proxy != "" {
		url, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy setup: %w", err)
		}
		httpTransport.(*http.Transport).Proxy = http.ProxyURL(url)
	}
	tlsConfig := httpTransport.(*http.Transport).TLSClientConfig
	if tlsConfig == nil {
		tlsConfig = &tls.Config{}
		httpTransport.(*http.Transport).TLSClientConfig = tlsConfig
	}

	if certFile != "" {
		if _, err := os.Stat(certFile); !os.IsNotExist(err) {
			return nil, fmt.Errorf("client certFileificate %s does not exist", certFile)
		}
		certFile, err := tls.LoadX509KeyPair(certFile, certFile)
		if err != nil {
			return nil, fmt.Errorf("error loading client certFileificate: %w", err)
		}

		tlsConfig.Certificates = []tls.Certificate{certFile}
	}

	httpClient := &http.Client{
		Transport: &clientTransport{
			id:        fmt.Sprintf("EDI-Connector/%s %s %s", version.Version, runtime.GOOS, runtime.GOARCH),
			username:  username,
			password:  password,
			transport: httpTransport,
		},
	}

	c := &Client{
		http:    httpClient,
		baseUrl: baseUrl,
	}

	if caFile != "" {
		pem, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("error while loading ca certificates: %w", err)
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(pem)
		tlsConfig.RootCAs = pool
	}
	return c, nil
}

func (c *Client) DownloadTransmission(transmission Transmission) ([]byte, error) {
	url := transmission.Url

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download transmission request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error while loading transmission with url %q: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error bad response for transmission %q: %w", url, err)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error while writing response data for transmission %q: %w", url, err)
	}
	return data, nil
}

func (c *Client) ListTransmissions(ctx context.Context, configId string) ([]Transmission, error) {
	req, err := c.req("GET", fmt.Sprintf("/rest/v2/transmissions?configID=%s", configId), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list transmissions request: %w", err)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list transmisions: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		data, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response error: %w", err)
		}
		return nil, fmt.Errorf("received bad response: %s: %s", res.Status, string(data))
	}

	response := struct {
		Transmissions []Transmission
	}{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transmissions: %w", err)
	}

	return response.Transmissions, nil
}

func (c *Client) AddTransmission(ctx context.Context, configId string, data []byte) error {
	req, err := c.req("POST", fmt.Sprintf("/rest/v2/transmissions?configID=%s", configId), data)
	if err != nil {
		return fmt.Errorf("failed to create add transmission request: %w", err)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to add transmission: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}
		return fmt.Errorf("failed to add transmission got platform error: %s: %s", res.Status, data)
	}

	return nil
}

func (c *Client) ConfirmTransmission(ctx context.Context, id string) error {
	var confirmRequest struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}
	confirmRequest.Message = fmt.Sprintf("processed transmission %s", id)

	data, err := json.Marshal(confirmRequest)
	if err != nil {
		return fmt.Errorf("failed to confirm transmission: %w", err)
	}

	req, err := c.req("POST", fmt.Sprintf("/rest/v2/transmissions/%s/confirm", id), data)
	if err != nil {
		return fmt.Errorf("failed to create confirm request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to confirm transmission: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}
		return fmt.Errorf("failed to confirm transmission got platform error: %s: %s", res.Status, data)
	}

	return nil
}

func (c *Client) AddAttachment(ctx context.Context, data []byte, filename string) error {
	req, err := c.req("POST", "/rest/v2/attachments", data)
	if err != nil {
		return fmt.Errorf("failed to create attachment upload request: %w", err)
	}
	req.Header.Add("Content-Type", "application/octet-stream")
	req.Header.Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed issue to attachment upload request: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}
		return fmt.Errorf("failed to add attachment got platform error: %s: %s", res.Status, data)
	}

	return nil
}

func (c *Client) ListMessageAttachments(ctx context.Context, id string) ([]MessageAttachment, error) {
	req, err := c.req("GET", fmt.Sprintf("/rest/v2/messages/%s/attachments", id), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch attachments: %w", err)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create list message attachments request: %w", err)
	}

	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch attachments got platform error: %s: %s", res.Status, data)
	}

	attachments := make([]MessageAttachment, 0)
	if err := json.Unmarshal(data, &attachments); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attachment response: %w", err)
	}

	return attachments, nil
}

func (c *Client) req(method string, path string, data []byte) (*http.Request, error) {
	var req *http.Request
	var err error

	var reader io.Reader
	if data != nil {
		reader = bytes.NewReader(data)
	}
	req, err = http.NewRequest(method, fmt.Sprintf("%s%s", c.baseUrl, path), reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s:%s: %w", method, path, err)
	}
	return req, nil
}

type clientTransport struct {
	id       string
	username string
	password string

	transport http.RoundTripper
}

func (t *clientTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("User-Agent", t.id)
	r.Header.Set("Accept", "application/json")
	r.SetBasicAuth(t.username, t.password)
	return t.transport.RoundTrip(r)
}
