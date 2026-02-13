package registry

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mkusaka/terraform-docs-cli/internal/cache"
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

func TestNewClient_InvalidBaseURLWithoutSchemeOrHostReturnsConfigError(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		wantMsg string
	}{
		{name: "missing scheme", baseURL: "registry.terraform.io", wantMsg: "scheme and host are required"},
		{name: "missing host", baseURL: "https:///v2", wantMsg: "scheme and host are required"},
		{name: "unsupported scheme", baseURL: "ftp://registry.terraform.io", wantMsg: "scheme must be http or https"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(Config{BaseURL: tt.baseURL, Timeout: 5 * time.Second}, nil)
			if err == nil {
				t.Fatalf("expected error for base url %q", tt.baseURL)
			}

			var cfgErr *ConfigError
			if !errors.As(err, &cfgErr) {
				t.Fatalf("expected ConfigError, got %T (%v)", err, err)
			}
			if !strings.Contains(cfgErr.Error(), tt.wantMsg) {
				t.Fatalf("unexpected error message: %s", cfgErr.Error())
			}
		})
	}
}

func TestResolve_PreservesBasePathPrefixForAbsoluteAPIPaths(t *testing.T) {
	c, err := NewClient(Config{BaseURL: "https://example.com/registry", Timeout: 5 * time.Second}, nil)
	if err != nil {
		t.Fatal(err)
	}

	got, err := c.resolve("/v2/providers/hashicorp/aws?include=provider-versions")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://example.com/registry/v2/providers/hashicorp/aws?include=provider-versions"
	if got != want {
		t.Fatalf("unexpected resolved URL\nwant: %s\ngot:  %s", want, got)
	}
}

func TestResolve_RootBasePathStillUsesRoot(t *testing.T) {
	c, err := NewClient(Config{BaseURL: "https://registry.terraform.io", Timeout: 5 * time.Second}, nil)
	if err != nil {
		t.Fatal(err)
	}

	got, err := c.resolve("/v2/providers/hashicorp/aws")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://registry.terraform.io/v2/providers/hashicorp/aws"
	if got != want {
		t.Fatalf("unexpected resolved URL\nwant: %s\ngot:  %s", want, got)
	}
}

func TestGetJSON_RefetchesWhenCachedPayloadIsInvalidJSON(t *testing.T) {
	var requestCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	store, err := cache.NewStore(t.TempDir(), time.Hour, true)
	if err != nil {
		t.Fatal(err)
	}

	c, err := NewClient(Config{BaseURL: srv.URL, Timeout: 5 * time.Second}, store)
	if err != nil {
		t.Fatal(err)
	}

	path := "/v2/provider-docs/1"
	fullURL, err := c.resolve(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Set(http.MethodGet, fullURL, http.StatusOK, "application/json", []byte("not-json")); err != nil {
		t.Fatal(err)
	}

	var dst map[string]any
	if err := c.GetJSON(context.Background(), path, &dst); err != nil {
		t.Fatalf("expected cache decode fallback to succeed, got error: %v", err)
	}
	if requestCount.Load() != 1 {
		t.Fatalf("expected one network request for refetch, got %d", requestCount.Load())
	}
	if got, ok := dst["ok"].(bool); !ok || !got {
		t.Fatalf("unexpected decoded payload: %#v", dst)
	}

	// The refetched valid body should be cached and reused.
	dst = nil
	if err := c.GetJSON(context.Background(), path, &dst); err != nil {
		t.Fatalf("expected second call to succeed from refreshed cache, got error: %v", err)
	}
	if requestCount.Load() != 1 {
		t.Fatalf("expected no additional network request on second call, got %d", requestCount.Load())
	}
}
