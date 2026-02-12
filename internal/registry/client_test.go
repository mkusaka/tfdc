package registry

import (
	"net/http"
	"os"
	"testing"
	"time"
)

func TestNewClient_UsesProxyFromEnvironment(t *testing.T) {
	proxyURL := "http://127.0.0.1:18080"
	oldHTTPProxy := os.Getenv("HTTP_PROXY")
	oldHTTPSProxy := os.Getenv("HTTPS_PROXY")
	defer func() {
		_ = os.Setenv("HTTP_PROXY", oldHTTPProxy)
		_ = os.Setenv("HTTPS_PROXY", oldHTTPSProxy)
	}()
	_ = os.Setenv("HTTP_PROXY", proxyURL)
	_ = os.Setenv("HTTPS_PROXY", proxyURL)

	c, err := NewClient(Config{BaseURL: "https://registry.terraform.io", Timeout: 5 * time.Second}, nil)
	if err != nil {
		t.Fatal(err)
	}

	transport, ok := c.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("unexpected transport type: %T", c.httpClient.Transport)
	}
	if transport.Proxy == nil {
		t.Fatalf("expected proxy function to be set")
	}

	req, err := http.NewRequest(http.MethodGet, "https://registry.terraform.io/v2/providers/hashicorp/aws", nil)
	if err != nil {
		t.Fatal(err)
	}
	gotProxy, err := transport.Proxy(req)
	if err != nil {
		t.Fatal(err)
	}
	if gotProxy == nil || gotProxy.String() != proxyURL {
		t.Fatalf("expected proxy %s, got %v", proxyURL, gotProxy)
	}
}
