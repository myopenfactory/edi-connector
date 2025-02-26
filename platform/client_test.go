package platform_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"github.com/myopenfactory/edi-connector/platform"
	"github.com/myopenfactory/edi-connector/version"
)

func TestUsernamePassword(t *testing.T) {
	testUsername := "user"
	testPassword := "password"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			t.Error("Expected basic auth headers")
		}
		if username != testUsername {
			t.Errorf("Expected username: %s, got: %s", testUsername, username)
		}
		if password != testPassword {
			t.Errorf("Expected password: %s, got: %s", testPassword, password)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Expected Accept: application/json header, got: %s", r.Header.Get("Accept"))
		}
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(platform.Transmission{}); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cl, err := platform.NewClient(server.URL, testUsername, testPassword, "", "", "")
	if err != nil {
		t.Errorf("failed to create edi client: %v", err)
	}

	_, err = cl.ListTransmissions(context.TODO(), "1")
	if err != nil {
		t.Errorf("failed to verify username password: %v", err)
	}
}

func TestUserAgent(t *testing.T) {
	testUsername := "user"
	testPassword := "password"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgentEntries := strings.Split(r.Header.Get("User-Agent"), " ")
		product := strings.Split(userAgentEntries[0], "/")
		if product[0] != "EDI-Connector" {
			t.Errorf("Expected product: EDI-Connector, got: %s", product[0])
		}
		if product[1] != version.Version {
			t.Errorf("Expected version: %s, got: %s", version.Version, product[1])
		}
		if userAgentEntries[1] != runtime.GOOS {
			t.Errorf("Expected os: %s, got: %s", runtime.GOOS, userAgentEntries[1])
		}
		if userAgentEntries[2] != runtime.GOARCH {
			t.Errorf("Expected arch: %s, got: %s", runtime.GOARCH, userAgentEntries[2])
		}
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(platform.Transmission{}); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cl, err := platform.NewClient(server.URL, testUsername, testPassword, "", "", "")
	if err != nil {
		t.Errorf("failed to create edi client: %v", err)
	}

	_, err = cl.ListTransmissions(context.TODO(), "1")
	if err != nil {
		t.Errorf("failed to verify user agent: %v", err)
	}
}

func TestAccept(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Expected Accept: application/json header, got: %s", r.Header.Get("Accept"))
		}
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(platform.Transmission{}); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cl, err := platform.NewClient(server.URL, "", "", "", "", "")
	if err != nil {
		t.Errorf("failed to create edi client: %v", err)
	}

	_, err = cl.ListTransmissions(context.TODO(), "1")
	if err != nil {
		t.Errorf("failed to verify accept header: %v", err)
	}
}

func TestDownloadTransmission(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data"))
	}))
	defer server.Close()

	cl, err := platform.NewClient("", "", "", "", "", "")
	if err != nil {
		t.Errorf("failed to create edi client: %v", err)
	}

	data, err := cl.DownloadTransmission(platform.Transmission{
		Id:  "1",
		Url: server.URL,
	})
	if err != nil {
		t.Errorf("failed to download transmission: %v", err)
	}

	expectedData := []byte("data")
	if !bytes.Equal(expectedData, data) {
		t.Errorf("Expected response data: %s, got: %s", string(expectedData), string(data))
	}
}

func TestListTransmissions(t *testing.T) {
	configId := "xaz43I"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/rest/v2/transmissions"
		gotPath := r.URL.Path
		if expectedPath != gotPath {
			t.Errorf("Expected request path: %s, got: %s", expectedPath, gotPath)
		}
		gotConfigId := r.URL.Query().Get("configID")
		if configId != gotConfigId {
			t.Errorf("Expected config id: %s, got: %s", configId, gotConfigId)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"transmissions": [{
				"id": "1"
			}]
		}`))
	}))
	defer server.Close()

	cl, err := platform.NewClient(server.URL, "", "", "", "", "")
	if err != nil {
		t.Errorf("failed to create edi client: %v", err)
	}

	transmissions, err := cl.ListTransmissions(t.Context(), configId)
	if err != nil {
		t.Errorf("failed to list transmission: %v", err)
	}

	if len(transmissions) != 1 {
		t.Errorf("Exepected transmissions: %d, got: %d", 1, len(transmissions))
		return
	}

	transmission := transmissions[0]
	id := "1"
	if transmission.Id != id {
		t.Errorf("Expected transmission id: %s, got: %s", id, transmission.Id)
	}
}

func TestAddTransmission(t *testing.T) {
	configId := "xaz43I"
	testData := []byte("test1235")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/rest/v2/transmissions"
		gotPath := r.URL.Path
		if expectedPath != gotPath {
			t.Errorf("Expected request path: %s, got: %s", expectedPath, gotPath)
		}
		gotConfigId := r.URL.Query().Get("configID")
		if configId != gotConfigId {
			t.Errorf("Expected config id: %s, got: %s", configId, gotConfigId)
		}
		gotMethod := r.Method
		expectedMethod := "POST"
		if expectedMethod != gotMethod {
			t.Errorf("Expected method: %s, got: %s", expectedMethod, gotMethod)
		}
		defer r.Body.Close()
		gotData, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
		}

		if !bytes.Equal(testData, gotData) {
			t.Errorf("Expected request data: %x, got: %x", testData, gotData)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cl, err := platform.NewClient(server.URL, "", "", "", "", "")
	if err != nil {
		t.Errorf("failed to create edi client: %v", err)
	}

	err = cl.AddTransmission(t.Context(), configId, testData)
	if err != nil {
		t.Errorf("failed to add transmission: %v", err)
	}
}
