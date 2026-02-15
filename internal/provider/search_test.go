package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
)

type fakeSearchClient struct{}

func (f *fakeSearchClient) GetJSON(_ context.Context, path string, dst any) error {
	// GET /v1/providers/hashicorp/aws → latest version
	if path == "/v1/providers/hashicorp/aws" {
		b, _ := json.Marshal(map[string]any{"version": "6.31.0"})
		return json.Unmarshal(b, dst)
	}

	// GET /v1/providers/hashicorp/aws/6.31.0 → docs list (v1)
	if path == "/v1/providers/hashicorp/aws/6.31.0" {
		b, _ := json.Marshal(map[string]any{
			"docs": []map[string]any{
				{"id": 100, "title": "aws_ec2_instance", "category": "resources", "slug": "aws_ec2_instance", "language": "hcl"},
				{"id": 101, "title": "aws_s3_bucket", "category": "resources", "slug": "aws_s3_bucket", "language": "hcl"},
				{"id": 102, "title": "aws_ec2_network_interface", "category": "resources", "slug": "aws_ec2_network_interface", "language": "hcl"},
				{"id": 200, "title": "aws_ec2_instance", "category": "data-sources", "slug": "aws_ec2_instance", "language": "hcl"},
			},
		})
		return json.Unmarshal(b, dst)
	}

	// GET /v2/providers/hashicorp/aws?include=provider-versions → version resolution
	if strings.HasPrefix(path, "/v2/providers/hashicorp/aws") {
		b, _ := json.Marshal(map[string]any{
			"included": []any{
				map[string]any{"type": "provider-versions", "id": "70800", "attributes": map[string]any{"version": "6.31.0"}},
			},
		})
		return json.Unmarshal(b, dst)
	}

	// GET /v2/provider-docs?filter[...] → v2 doc listing
	if strings.HasPrefix(path, "/v2/provider-docs?") {
		u, err := url.Parse(path)
		if err != nil {
			return err
		}
		q := u.Query()
		cat := q.Get("filter[category]")
		page := q.Get("page[number]")

		var data []map[string]any
		if cat == "guides" && page == "1" {
			data = []map[string]any{
				{"id": "300", "attributes": map[string]any{"category": "guides", "slug": "ec2-guide", "title": "EC2 Guide"}},
				{"id": "301", "attributes": map[string]any{"category": "guides", "slug": "s3-guide", "title": "S3 Guide"}},
			}
		}
		b, _ := json.Marshal(map[string]any{"data": data})
		return json.Unmarshal(b, dst)
	}

	return fmt.Errorf("unexpected GetJSON path: %s", path)
}

func (f *fakeSearchClient) Get(_ context.Context, path string) ([]byte, error) {
	return nil, fmt.Errorf("unexpected Get call: %s", path)
}

func TestSearchDocs_V1_Resources(t *testing.T) {
	results, err := SearchDocs(context.Background(), &fakeSearchClient{}, SearchOptions{
		Name:      "aws",
		Namespace: "hashicorp",
		Service:   "ec2",
		Type:      "resources",
		Version:   "6.31.0",
		Limit:     20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should match aws_ec2_instance and aws_ec2_network_interface
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d: %+v", len(results), results)
	}
	for _, r := range results {
		if !strings.Contains(r.Slug, "ec2") {
			t.Errorf("expected slug to contain ec2, got %s", r.Slug)
		}
		if r.Category != "resources" {
			t.Errorf("expected category=resources, got %s", r.Category)
		}
	}
}

func TestSearchDocs_V1_DataSources(t *testing.T) {
	results, err := SearchDocs(context.Background(), &fakeSearchClient{}, SearchOptions{
		Name:    "aws",
		Service: "ec2",
		Type:    "data-sources",
		Version: "6.31.0",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ProviderDocID != "200" {
		t.Errorf("expected doc id 200, got %s", results[0].ProviderDocID)
	}
}

func TestSearchDocs_V2_Guides(t *testing.T) {
	results, err := SearchDocs(context.Background(), &fakeSearchClient{}, SearchOptions{
		Name:    "aws",
		Service: "ec2",
		Type:    "guides",
		Version: "6.31.0",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
	}
	if results[0].ProviderDocID != "300" {
		t.Errorf("expected doc id 300, got %s", results[0].ProviderDocID)
	}
}

func TestSearchDocs_LatestVersion(t *testing.T) {
	results, err := SearchDocs(context.Background(), &fakeSearchClient{}, SearchOptions{
		Name:    "aws",
		Service: "ec2",
		Type:    "resources",
		Version: "latest",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results for latest version")
	}
	if results[0].Version != "6.31.0" {
		t.Errorf("expected version 6.31.0, got %s", results[0].Version)
	}
}

func TestSearchDocs_Limit(t *testing.T) {
	results, err := SearchDocs(context.Background(), &fakeSearchClient{}, SearchOptions{
		Name:    "aws",
		Service: "ec2",
		Type:    "resources",
		Version: "6.31.0",
		Limit:   1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result (limit), got %d", len(results))
	}
}

func TestSearchDocs_ValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		opts SearchOptions
		want string
	}{
		{"missing name", SearchOptions{Service: "ec2", Type: "resources"}, "-name is required"},
		{"missing service", SearchOptions{Name: "aws", Type: "resources"}, "-service is required"},
		{"missing type", SearchOptions{Name: "aws", Service: "ec2"}, "-type is required"},
		{"invalid type", SearchOptions{Name: "aws", Service: "ec2", Type: "invalid"}, "unsupported -type"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := SearchDocs(context.Background(), &fakeSearchClient{}, tc.opts)
			if err == nil {
				t.Fatal("expected error")
			}
			var vErr *ValidationError
			if !errors.As(err, &vErr) {
				t.Fatalf("expected ValidationError, got %T: %v", err, err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("expected error containing %q, got: %v", tc.want, err)
			}
		})
	}
}
