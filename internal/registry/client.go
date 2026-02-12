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
	b, err := c.Get(ctx, path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, dst); err != nil {
		return fmt.Errorf("failed to decode json response: %w", err)
	}
	return nil
}

func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	fullURL, err := c.resolve(path)
	if err != nil {
		return nil, err
	}

	if c.cache != nil {
		if b, ok, err := c.cache.Get(http.MethodGet, fullURL); err == nil && ok {
			if c.debug {
				fmt.Fprintf(os.Stderr, "cache hit: %s\n", fullURL)
			}
			return b, nil
		}
	}

	var lastErr error
	for attempt := 0; attempt <= c.retry; attempt++ {
		if c.debug {
			fmt.Fprintf(os.Stderr, "http get attempt=%d url=%s\n", attempt+1, fullURL)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", c.userAgent)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < c.retry {
				continue
			}
			return nil, err
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			if attempt < c.retry {
				continue
			}
			return nil, readErr
		}

		if resp.StatusCode != http.StatusOK {
			apiErr := &APIError{StatusCode: resp.StatusCode, URL: fullURL, Body: string(body)}
			lastErr = apiErr
			if (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError) && attempt < c.retry {
				continue
			}
			return nil, apiErr
		}

		if c.cache != nil {
			_ = c.cache.Set(http.MethodGet, fullURL, resp.StatusCode, resp.Header.Get("Content-Type"), body)
		}

		return body, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("unexpected error in get request")
}

func (c *Client) resolve(path string) (string, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path, nil
	}
	ref, err := url.Parse(path)
	if err != nil {
		return "", err
	}
	return c.baseURL.ResolveReference(ref).String(), nil
}
