package module

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type fakeModuleClient struct{}

func (f *fakeModuleClient) GetJSON(_ context.Context, path string, dst any) error {
	if strings.HasPrefix(path, "/v1/modules/search?") {
		b, _ := json.Marshal(map[string]any{
			"modules": []map[string]any{
				{
					"id":           "terraform-aws-modules/vpc/aws/6.0.1",
					"name":         "vpc",
					"description":  "Terraform module for AWS VPC",
					"downloads":    50000,
					"verified":     true,
					"published_at": "2024-01-15T00:00:00Z",
				},
				{
					"id":           "terraform-aws-modules/vpc/aws/5.0.0",
					"name":         "vpc",
					"description":  "Terraform module for AWS VPC (older)",
					"downloads":    30000,
					"verified":     true,
					"published_at": "2023-06-01T00:00:00Z",
				},
			},
			"meta": map[string]any{"limit": 20, "current_offset": 0},
		})
		return json.Unmarshal(b, dst)
	}
	return fmt.Errorf("unexpected GetJSON path: %s", path)
}

func (f *fakeModuleClient) Get(_ context.Context, path string) ([]byte, error) {
	if path == "/v1/modules/terraform-aws-modules/vpc/aws/6.0.1" {
		return json.Marshal(map[string]any{
			"root": map[string]any{
				"readme": "# VPC Module\n\nThis module creates a VPC.",
			},
		})
	}
	return nil, fmt.Errorf("unexpected Get path: %s", path)
}

func TestSearchModules_Success(t *testing.T) {
	results, total, err := SearchModules(context.Background(), &fakeModuleClient{}, SearchOptions{
		Query: "vpc",
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total=2, got %d", total)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Name != "vpc" {
		t.Errorf("expected name=vpc, got %s", results[0].Name)
	}
}

func TestSearchModules_EmptyQuery(t *testing.T) {
	_, _, err := SearchModules(context.Background(), &fakeModuleClient{}, SearchOptions{Query: ""})
	if err == nil {
		t.Fatal("expected error for empty query")
	}
	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}

func TestGetModule_Success(t *testing.T) {
	result, err := GetModule(context.Background(), &fakeModuleClient{}, "terraform-aws-modules/vpc/aws/6.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "terraform-aws-modules/vpc/aws/6.0.1" {
		t.Errorf("expected id match, got %s", result.ID)
	}
	if !strings.Contains(result.Content, "VPC Module") {
		t.Errorf("expected readme content, got: %s", result.Content)
	}
}

func TestGetModule_EmptyID(t *testing.T) {
	_, err := GetModule(context.Background(), &fakeModuleClient{}, "")
	if err == nil {
		t.Fatal("expected error for empty id")
	}
}

func TestGetModule_InvalidSegments(t *testing.T) {
	_, err := GetModule(context.Background(), &fakeModuleClient{}, "too/few/segments")
	if err == nil {
		t.Fatal("expected error for wrong segment count")
	}
	if !strings.Contains(err.Error(), "4 segments") {
		t.Errorf("expected segment count error, got: %v", err)
	}
}
