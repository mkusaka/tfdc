package registry

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/mkusaka/terraform-docs-cli/internal/cache"
)

type APIError struct {
	StatusCode int
	URL        string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("registry API error: status=%d url=%s", e.StatusCode, e.URL)
}

type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string { return e.Message }

type Config struct {
	BaseURL   string
	Timeout   time.Duration
	Retry     int
	Insecure  bool
	UserAgent string
	Debug     bool
}

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	retry      int
	cache      *cache.Store
	userAgent  string
	debug      bool
}

func NewClient(cfg Config, cacheStore *cache.Store) (*Client, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://registry.terraform.io"
	}
	base, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, &ConfigError{Message: fmt.Sprintf("invalid base url: %v", err)}
	}
	if strings.TrimSpace(base.Scheme) == "" || strings.TrimSpace(base.Host) == "" {
		return nil, &ConfigError{Message: fmt.Sprintf("invalid base url: scheme and host are required (%s)", cfg.BaseURL)}
	}
	scheme := strings.ToLower(strings.TrimSpace(base.Scheme))
	if scheme != "http" && scheme != "https" {
		return nil, &ConfigError{Message: fmt.Sprintf("invalid base url: scheme must be http or https (%s)", cfg.BaseURL)}
	}

	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, &ConfigError{Message: "unexpected default transport type"}
	}
	transport = transport.Clone()
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	} else {
		transport.TLSClientConfig = transport.TLSClientConfig.Clone()
	}
	transport.TLSClientConfig.InsecureSkipVerify = cfg.Insecure
	transport.Proxy = http.ProxyFromEnvironment

	client := &http.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
	}

	userAgent := cfg.UserAgent
	if userAgent == "" {
		userAgent = "terraform-docs-cli/dev"
	}

	return &Client{
		baseURL:    base,
		httpClient: client,
		retry:      cfg.Retry,
		cache:      cacheStore,
		userAgent:  userAgent,
		debug:      cfg.Debug,
	}, nil
}

func (c *Client) GetJSON(ctx context.Context, path string, dst any) error {
	b, fromCache, err := c.get(ctx, path, true)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, dst); err != nil {
		if !fromCache {
			return fmt.Errorf("failed to decode json response: %w", err)
		}
		// If cached payload is undecodable, treat it as cache miss and refetch.
		fresh, _, refetchErr := c.get(ctx, path, false)
		if refetchErr != nil {
			return refetchErr
		}
		if err := json.Unmarshal(fresh, dst); err != nil {
			return fmt.Errorf("failed to decode json response: %w", err)
		}
		return nil
	}
	return nil
}

func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	b, _, err := c.get(ctx, path, true)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (c *Client) get(ctx context.Context, path string, readCache bool) ([]byte, bool, error) {
	fullURL, err := c.resolve(path)
	if err != nil {
		return nil, false, err
	}

	if readCache && c.cache != nil {
		if b, ok, err := c.cache.Get(http.MethodGet, fullURL); err == nil && ok {
			if c.debug {
				fmt.Fprintf(os.Stderr, "cache hit: %s\n", fullURL)
			}
			return b, true, nil
		}
	}

	var lastErr error
	for attempt := 0; attempt <= c.retry; attempt++ {
		if c.debug {
			fmt.Fprintf(os.Stderr, "http get attempt=%d url=%s\n", attempt+1, fullURL)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
		if err != nil {
			return nil, false, err
		}
		req.Header.Set("User-Agent", c.userAgent)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < c.retry {
				continue
			}
			return nil, false, err
		}

		body, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if readErr == nil && closeErr != nil {
			readErr = closeErr
		}
		if readErr != nil {
			lastErr = readErr
			if attempt < c.retry {
				continue
			}
			return nil, false, readErr
		}

		if resp.StatusCode != http.StatusOK {
			apiErr := &APIError{StatusCode: resp.StatusCode, URL: fullURL, Body: string(body)}
			lastErr = apiErr
			if (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError) && attempt < c.retry {
				continue
			}
			return nil, false, apiErr
		}

		if c.cache != nil {
			_ = c.cache.Set(http.MethodGet, fullURL, resp.StatusCode, resp.Header.Get("Content-Type"), body)
		}

		return body, false, nil
	}

	if lastErr != nil {
		return nil, false, lastErr
	}
	return nil, false, fmt.Errorf("unexpected error in get request")
}

func (c *Client) resolve(path string) (string, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path, nil
	}
	ref, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	// Keep a configured base path prefix (e.g. https://host/registry) for
	// API paths that start with "/" so reverse-proxy deployments work.
	if strings.HasPrefix(path, "/") && c.baseURL.Path != "" && c.baseURL.Path != "/" {
		basePath := "/" + strings.Trim(strings.TrimSpace(c.baseURL.Path), "/")
		ref.Path = basePath + "/" + strings.TrimLeft(ref.Path, "/")
		if ref.RawPath != "" {
			baseRawPath := "/" + strings.Trim(strings.TrimSpace(c.baseURL.EscapedPath()), "/")
			ref.RawPath = baseRawPath + "/" + strings.TrimLeft(ref.RawPath, "/")
		}
	}

	return c.baseURL.ResolveReference(ref).String(), nil
}
