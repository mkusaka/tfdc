package policy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type fakePolicyClient struct{}

func (f *fakePolicyClient) GetJSON(_ context.Context, path string, dst any) error {
	if strings.HasPrefix(path, "/v2/policies?") {
		b, _ := json.Marshal(map[string]any{
			"data": []map[string]any{
				{
					"id": "1",
					"attributes": map[string]any{
						"name":      "CIS-Policy-Set-for-AWS-Terraform",
						"title":     "CIS Policy Set for AWS Terraform",
						"downloads": 1000,
					},
					"relationships": map[string]any{
						"latest-version": map[string]any{
							"links": map[string]any{
								"related": "/v2/policies/hashicorp/CIS-Policy-Set-for-AWS-Terraform/1.0.1",
							},
						},
					},
				},
				{
					"id": "2",
					"attributes": map[string]any{
						"name":      "GCP-Networking-Policy",
						"title":     "GCP Networking Policy",
						"downloads": 500,
					},
					"relationships": map[string]any{
						"latest-version": map[string]any{
							"links": map[string]any{
								"related": "/v2/policies/hashicorp/GCP-Networking-Policy/2.0.0",
							},
						},
					},
				},
			},
		})
		return json.Unmarshal(b, dst)
	}
	return fmt.Errorf("unexpected GetJSON path: %s", path)
}

func (f *fakePolicyClient) Get(_ context.Context, path string) ([]byte, error) {
	if strings.HasPrefix(path, "/v2/policies/hashicorp/CIS-Policy-Set-for-AWS-Terraform/1.0.1") {
		return json.Marshal(map[string]any{
			"data": map[string]any{
				"id": "policies/hashicorp/CIS-Policy-Set-for-AWS-Terraform/1.0.1",
				"attributes": map[string]any{
					"readme": "# CIS Policy Set\n\nThis policy set contains CIS benchmark rules.",
				},
			},
		})
	}
	return nil, fmt.Errorf("unexpected Get path: %s", path)
}

func TestSearchPolicies_Success(t *testing.T) {
	results, total, err := SearchPolicies(context.Background(), &fakePolicyClient{}, "cis")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total=1, got %d", total)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].TerraformPolicyID != "policies/hashicorp/CIS-Policy-Set-for-AWS-Terraform/1.0.1" {
		t.Errorf("unexpected policy id: %s", results[0].TerraformPolicyID)
	}
}

func TestSearchPolicies_NoMatch(t *testing.T) {
	results, total, err := SearchPolicies(context.Background(), &fakePolicyClient{}, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 || len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestSearchPolicies_EmptyQuery(t *testing.T) {
	_, _, err := SearchPolicies(context.Background(), &fakePolicyClient{}, "")
	if err == nil {
		t.Fatal("expected error for empty query")
	}
	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}

func TestGetPolicy_Success(t *testing.T) {
	result, err := GetPolicy(context.Background(), &fakePolicyClient{}, "policies/hashicorp/CIS-Policy-Set-for-AWS-Terraform/1.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "CIS Policy Set") {
		t.Errorf("expected readme content, got: %s", result.Content)
	}
}

func TestGetPolicy_EmptyID(t *testing.T) {
	_, err := GetPolicy(context.Background(), &fakePolicyClient{}, "")
	if err == nil {
		t.Fatal("expected error for empty id")
	}
}

func TestGetPolicy_InvalidPrefix(t *testing.T) {
	_, err := GetPolicy(context.Background(), &fakePolicyClient{}, "wrong/prefix")
	if err == nil {
		t.Fatal("expected error for invalid prefix")
	}
	if !strings.Contains(err.Error(), "policies/") {
		t.Errorf("expected prefix error, got: %v", err)
	}
}

func TestExtractPolicyID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/v2/policies/hashicorp/CIS/1.0.1", "policies/hashicorp/CIS/1.0.1"},
		{"policies/hashicorp/CIS/1.0.1", "policies/hashicorp/CIS/1.0.1"},
		{"", ""},
	}
	for _, tc := range tests {
		got := extractPolicyID(tc.input)
		if got != tc.want {
			t.Errorf("extractPolicyID(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
