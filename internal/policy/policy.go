package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// APIClient is the interface needed for policy operations.
type APIClient interface {
	GetJSON(ctx context.Context, path string, dst any) error
	Get(ctx context.Context, path string) ([]byte, error)
}

// SearchResult represents one matching policy.
type SearchResult struct {
	TerraformPolicyID string `json:"terraform_policy_id"`
	Name              string `json:"name"`
	Title             string `json:"title"`
	Downloads         int    `json:"downloads"`
}

// GetResult holds the result of fetching a policy.
type GetResult struct {
	ID      string
	Content string // readme content
	Raw     json.RawMessage
}

// v2PoliciesResponse is the response from GET /v2/policies.
type v2PoliciesResponse struct {
	Data []v2PolicyData `json:"data"`
}

type v2PolicyData struct {
	ID         string `json:"id"`
	Attributes struct {
		Name      string `json:"name"`
		Title     string `json:"title"`
		Downloads int    `json:"downloads"`
	} `json:"attributes"`
	Relationships struct {
		LatestVersion struct {
			Links struct {
				Related string `json:"related"`
			} `json:"links"`
		} `json:"latest-version"`
	} `json:"relationships"`
}

type v2PolicyDetailResponse struct {
	Data struct {
		ID         string `json:"id"`
		Attributes struct {
			Readme string `json:"readme"`
		} `json:"attributes"`
	} `json:"data"`
}

// SearchPolicies searches for policies matching the query.
// It fetches all policies (paginated) and filters client-side.
func SearchPolicies(ctx context.Context, client APIClient, query string) ([]SearchResult, int, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, 0, &ValidationError{Message: "-query is required"}
	}

	lowerQuery := strings.ToLower(query)
	var results []SearchResult
	for page := 1; ; page++ {
		path := fmt.Sprintf("/v2/policies?page[size]=100&page[number]=%d&include=latest-version", page)
		var resp v2PoliciesResponse
		if err := client.GetJSON(ctx, path, &resp); err != nil {
			return nil, 0, err
		}
		if len(resp.Data) == 0 {
			break
		}

		for _, p := range resp.Data {
			if !strings.Contains(strings.ToLower(p.Attributes.Name), lowerQuery) &&
				!strings.Contains(strings.ToLower(p.Attributes.Title), lowerQuery) {
				continue
			}

			policyID := extractPolicyID(p.Relationships.LatestVersion.Links.Related)
			if policyID == "" {
				policyID = p.ID
				if !strings.HasPrefix(policyID, "policies/") {
					policyID = "policies/" + policyID
				}
			}

			results = append(results, SearchResult{
				TerraformPolicyID: policyID,
				Name:              p.Attributes.Name,
				Title:             p.Attributes.Title,
				Downloads:         p.Attributes.Downloads,
			})
		}
	}
	return results, len(results), nil
}

// GetPolicy fetches details for a specific policy.
// id must start with "policies/".
func GetPolicy(ctx context.Context, client APIClient, id string) (*GetResult, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, &ValidationError{Message: "-id is required"}
	}
	if !strings.HasPrefix(id, "policies/") {
		return nil, &ValidationError{Message: fmt.Sprintf("-id must start with \"policies/\": %s", id)}
	}

	path := fmt.Sprintf("/v2/%s?include=policies,policy-modules,policy-library", id)
	raw, err := client.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var parsed v2PolicyDetailResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse policy response: %w", err)
	}

	return &GetResult{
		ID:      id,
		Content: parsed.Data.Attributes.Readme,
		Raw:     raw,
	}, nil
}

// extractPolicyID extracts the terraform_policy_id from a related link.
// Handles both relative paths ("/v2/policies/...") and full URLs
// ("https://registry.terraform.io/v2/policies/...").
func extractPolicyID(link string) string {
	link = strings.TrimSpace(link)
	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		u, err := url.Parse(link)
		if err != nil {
			return link
		}
		link = u.Path
	}
	if strings.HasPrefix(link, "/v2/") {
		return strings.TrimPrefix(link, "/v2/")
	}
	return link
}

// ValidationError indicates invalid input.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }
