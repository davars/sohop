package sohop

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"time"

	"encoding/json"

	"github.com/davars/sohop/auth"
	"github.com/stretchr/testify/require"
)

func dummyBackend(name string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, name)
	}))
}

func TestProxy(t *testing.T) {
	const upstreamName = "foo"
	server := dummyBackend(upstreamName)
	defer server.Close()

	sohop := Server{
		Config: &Config{
			Domain: "example.com",
			Cookie: CookieConfig{Secret: "3c0767ada2466a92a59c1214061441713aeafe6d115e29aa376c0f9758cdf0f5"},
			Upstreams: map[string]UpstreamConfig{
				upstreamName: {
					URL: server.URL,
				},
			},
			Auth: auth.Config{
				Type:   "mock",
				Config: json.RawMessage(`{}`),
			},
			TLS: TLSConfig{
				CertFile: "fixtures/cert.pem",
				CertKey:  "fixtures/key.pem",
			},
		},
		HTTPAddr:  "127.0.0.1:42080",
		HTTPSAddr: "127.0.0.1:42443",
	}
	go sohop.Run()
	time.Sleep(time.Second)

	req, err := http.NewRequest("GET", "https://127.0.0.1:42443", nil)
	require.NoError(t, err)
	req.Host = "foo.example.com"

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	require.NoError(t, err)

	b, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, upstreamName, string(b))
	require.NoError(t, fmt.Errorf("break a test"))
}
