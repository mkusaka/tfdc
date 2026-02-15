package module

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// APIClient is the interface needed for module operations.
type APIClient interface {
	GetJSON(ctx context.Context, path string, dst any) error
	Get(ctx context.Context, path string) ([]byte, error)
}

// SearchOptions holds parameters for module search.
type SearchOptions struct {
	Query  string
	Offset int
	Limit  int
}

// SearchResult represents one matching module.
type SearchResult struct {
	ModuleID    string `json:"module_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Downloads   int    `json:"downloads"`
	Verified    bool   `json:"verified"`
	PublishedAt string `json:"published_at"`
}

// GetResult holds the result of fetching a module.
type GetResult struct {
	ID      string
	Content string // readme content for text/markdown
	Raw     json.RawMessage
}

type v1ModuleSearchResponse struct {
	Modules []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Downloads   int    `json:"downloads"`
		Verified    bool   `json:"verified"`
		PublishedAt string `json:"published_at"`
	} `json:"modules"`
	Meta struct {
		Limit         int `json:"limit"`
		CurrentOffset int `json:"current_offset"`
	} `json:"meta"`
}

type v1ModuleGetResponse struct {
	Root struct {
		Readme string `json:"readme"`
	} `json:"root"`
}

// SearchModules searches the Terraform module registry.
func SearchModules(ctx context.Context, client APIClient, opts SearchOptions) ([]SearchResult, int, error) {
	if strings.TrimSpace(opts.Query) == "" {
		return nil, 0, &ValidationError{Message: "-query is required"}
	}
	if opts.Limit <= 0 {
		opts.Limit = 20
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}

	q := url.Values{}
	q.Set("q", opts.Query)
	q.Set("offset", fmt.Sprintf("%d", opts.Offset))
	q.Set("limit", fmt.Sprintf("%d", opts.Limit))

	path := "/v1/modules/search?" + q.Encode()
	var resp v1ModuleSearchResponse
	if err := client.GetJSON(ctx, path, &resp); err != nil {
		return nil, 0, err
	}

	results := make([]SearchResult, len(resp.Modules))
	for i, m := range resp.Modules {
		results[i] = SearchResult{
			ModuleID:    m.ID,
			Name:        m.Name,
			Description: m.Description,
			Downloads:   m.Downloads,
			Verified:    m.Verified,
			PublishedAt: m.PublishedAt,
		}
	}
	return results, len(results), nil
}

// GetModule fetches details for a specific module.
// id must be in namespace/name/provider/version format (4 segments).
func GetModule(ctx context.Context, client APIClient, id string) (*GetResult, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, &ValidationError{Message: "-id is required"}
	}

	parts := strings.Split(id, "/")
	if len(parts) != 4 {
		return nil, &ValidationError{Message: fmt.Sprintf("-id must have 4 segments (namespace/name/provider/version), got %d", len(parts))}
	}

	path := fmt.Sprintf("/v1/modules/%s/%s/%s/%s",
		url.PathEscape(parts[0]), url.PathEscape(parts[1]),
		url.PathEscape(parts[2]), url.PathEscape(parts[3]))

	raw, err := client.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var parsed v1ModuleGetResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse module response: %w", err)
	}

	return &GetResult{
		ID:      id,
		Content: parsed.Root.Readme,
		Raw:     raw,
	}, nil
}

// ValidationError indicates invalid input.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }
