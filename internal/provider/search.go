package provider

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// SearchOptions holds parameters for provider doc search.
type SearchOptions struct {
	Name      string
	Namespace string
	Service   string // slug-like search token to match against doc slugs
	Type      string // category: resources, data-sources, etc.
	Version   string // semver or "latest"
	Limit     int
}

// SearchResult represents one matching provider doc.
type SearchResult struct {
	ProviderDocID string `json:"provider_doc_id"`
	Title         string `json:"title"`
	Category      string `json:"category"`
	Slug          string `json:"slug"`
	Provider      string `json:"provider"`
	Namespace     string `json:"namespace"`
	Version       string `json:"version"`
}

// v1ProviderLatestResponse is the response from GET /v1/providers/{ns}/{name}.
type v1ProviderLatestResponse struct {
	Version string `json:"version"`
}

// v1ProviderDocsResponse is the response from GET /v1/providers/{ns}/{name}/{ver}.
type v1ProviderDocsResponse struct {
	Docs []v1ProviderDoc `json:"docs"`
}

type v1ProviderDoc struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Category string `json:"category"`
	Slug     string `json:"slug"`
	Language string `json:"language"`
}

// v1DocCategories are categories served by the v1 provider docs endpoint.
var v1DocCategories = map[string]bool{
	"resources":    true,
	"data-sources": true,
}

// SearchDocs searches provider documentation by service slug.
func SearchDocs(ctx context.Context, client APIClient, opts SearchOptions) ([]SearchResult, error) {
	if err := validateSearchOptions(&opts); err != nil {
		return nil, err
	}

	version := opts.Version
	if strings.EqualFold(version, "latest") || version == "" {
		resolved, err := resolveLatestVersion(ctx, client, opts.Namespace, opts.Name)
		if err != nil {
			return nil, err
		}
		version = resolved
	}

	if v1DocCategories[opts.Type] {
		return searchV1(ctx, client, opts, version)
	}
	return searchV2(ctx, client, opts, version)
}

func validateSearchOptions(opts *SearchOptions) error {
	opts.Name = strings.ToLower(strings.TrimSpace(opts.Name))
	opts.Namespace = strings.ToLower(strings.TrimSpace(opts.Namespace))
	opts.Service = strings.ToLower(strings.TrimSpace(opts.Service))
	opts.Type = strings.ToLower(strings.TrimSpace(opts.Type))
	opts.Version = strings.TrimSpace(opts.Version)

	if opts.Namespace == "" {
		opts.Namespace = "hashicorp"
	}
	if opts.Name == "" {
		return &ValidationError{Message: "-name is required"}
	}
	if opts.Service == "" {
		return &ValidationError{Message: "-service is required"}
	}
	if opts.Type == "" {
		return &ValidationError{Message: "-type is required"}
	}

	allowed := make(map[string]struct{}, len(defaultCategories))
	for _, c := range defaultCategories {
		allowed[c] = struct{}{}
	}
	if _, ok := allowed[opts.Type]; !ok {
		return &ValidationError{Message: fmt.Sprintf("unsupported -type: %s", opts.Type)}
	}

	if opts.Version == "" {
		opts.Version = "latest"
	}
	if opts.Limit <= 0 {
		opts.Limit = 20
	}
	return nil
}

func resolveLatestVersion(ctx context.Context, client APIClient, namespace, name string) (string, error) {
	path := fmt.Sprintf("/v1/providers/%s/%s", url.PathEscape(namespace), url.PathEscape(name))
	var resp v1ProviderLatestResponse
	if err := client.GetJSON(ctx, path, &resp); err != nil {
		return "", err
	}
	if resp.Version == "" {
		return "", &NotFoundError{Message: fmt.Sprintf("no version found for %s/%s", namespace, name)}
	}
	return resp.Version, nil
}

// searchV1 uses the v1 provider docs endpoint for resources/data-sources.
func searchV1(ctx context.Context, client APIClient, opts SearchOptions, version string) ([]SearchResult, error) {
	path := fmt.Sprintf("/v1/providers/%s/%s/%s",
		url.PathEscape(opts.Namespace), url.PathEscape(opts.Name), url.PathEscape(version))
	var resp v1ProviderDocsResponse
	if err := client.GetJSON(ctx, path, &resp); err != nil {
		return nil, err
	}

	var results []SearchResult
	for _, doc := range resp.Docs {
		if !strings.EqualFold(doc.Language, "hcl") && doc.Language != "" {
			continue
		}
		if !strings.EqualFold(doc.Category, opts.Type) {
			continue
		}
		if !containsSlug(doc.Slug, opts.Service) {
			continue
		}
		results = append(results, SearchResult{
			ProviderDocID: doc.ID,
			Title:         doc.Title,
			Category:      doc.Category,
			Slug:          doc.Slug,
			Provider:      opts.Name,
			Namespace:     opts.Namespace,
			Version:       version,
		})
		if len(results) >= opts.Limit {
			break
		}
	}
	return results, nil
}

// searchV2 uses the v2 provider-docs endpoint for guides, functions, overview, etc.
func searchV2(ctx context.Context, client APIClient, opts SearchOptions, version string) ([]SearchResult, error) {
	providerVersionID, err := resolveProviderVersionID(ctx, client, opts.Namespace, opts.Name, version)
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	for page := 1; ; page++ {
		docs, listErr := listProviderDocs(ctx, client, providerVersionID, opts.Type, page)
		if listErr != nil {
			return nil, listErr
		}
		if len(docs) == 0 {
			break
		}

		for _, doc := range docs {
			if !containsSlug(doc.Attributes.Slug, opts.Service) {
				continue
			}
			results = append(results, SearchResult{
				ProviderDocID: doc.ID,
				Title:         doc.Attributes.Title,
				Category:      doc.Attributes.Category,
				Slug:          doc.Attributes.Slug,
				Provider:      opts.Name,
				Namespace:     opts.Namespace,
				Version:       version,
			})
			if len(results) >= opts.Limit {
				return results, nil
			}
		}
	}
	return results, nil
}

// containsSlug checks if the doc slug contains the service token.
func containsSlug(slug, service string) bool {
	return strings.Contains(strings.ToLower(slug), strings.ToLower(service))
}
