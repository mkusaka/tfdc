package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeAPIClient struct{}

func (f *fakeAPIClient) GetJSON(_ context.Context, path string, dst any) error {
	if strings.HasPrefix(path, "/v2/providers/hashicorp/aws") {
		data := map[string]any{
			"included": []any{
				map[string]any{
					"type": "provider-versions",
					"id":   "70800",
					"attributes": map[string]any{
						"version": "6.31.0",
					},
				},
			},
		}
		b, _ := json.Marshal(data)
		return json.Unmarshal(b, dst)
	}

	if strings.HasPrefix(path, "/v2/provider-docs?") {
		u, err := url.Parse(path)
		if err != nil {
			return err
		}
		q := u.Query()
		cat := q.Get("filter[category]")
		page := q.Get("page[number]")

		var data []map[string]any
		switch {
		case cat == "guides" && page == "1":
			data = []map[string]any{{
				"id": "1",
				"attributes": map[string]any{
					"category": "guides",
					"slug":     "tag-policy-compliance",
					"title":    "Tag Policy Compliance",
				},
			}}
		case cat == "resources" && page == "1":
			data = []map[string]any{{
				"id": "2",
				"attributes": map[string]any{
					"category": "resources",
					"slug":     "aws_s3_bucket",
					"title":    "aws_s3_bucket",
				},
			}}
		default:
			data = []map[string]any{}
		}

		b, _ := json.Marshal(map[string]any{"data": data})
		return json.Unmarshal(b, dst)
	}

	return fmt.Errorf("unexpected GetJSON path: %s", path)
}

func (f *fakeAPIClient) Get(_ context.Context, path string) ([]byte, error) {
	switch path {
	case "/v2/provider-docs/1":
		return []byte(`{"data":{"id":"1","attributes":{"category":"guides","slug":"tag-policy-compliance","title":"Tag Policy Compliance","content":"# guide content"}}}`), nil
	case "/v2/provider-docs/2":
		return []byte(`{"data":{"id":"2","attributes":{"category":"resources","slug":"aws_s3_bucket","title":"aws_s3_bucket","content":"# resource content"}}}`), nil
	default:
		return nil, fmt.Errorf("unexpected Get path: %s", path)
	}
}

func TestExportDocs_WritesLayoutAndManifest(t *testing.T) {
	outDir := t.TempDir()
	client := &fakeAPIClient{}

	summary, err := ExportDocs(context.Background(), client, ExportOptions{
		Namespace:  "hashicorp",
		Name:       "aws",
		Version:    "6.31.0",
		Format:     "markdown",
		OutDir:     outDir,
		Categories: []string{"guides", "resources"},
		Clean:      false,
	})
	if err != nil {
		t.Fatal(err)
	}

	guidePath := filepath.Join(outDir, "terraform", "aws", "6.31.0", "docs", "guides", "tag-policy-compliance.md")
	resourcePath := filepath.Join(outDir, "terraform", "aws", "6.31.0", "docs", "resources", "aws_s3_bucket.md")
	manifestPath := filepath.Join(outDir, "terraform", "aws", "6.31.0", "docs", "_manifest.json")

	for _, p := range []string{guidePath, resourcePath, manifestPath} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected file to exist: %s (%v)", p, err)
		}
	}

	manifestBody, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manifestBody), "tag-policy-compliance") {
		t.Fatalf("manifest does not contain expected guide slug")
	}
	if summary.Written != 2 {
		t.Fatalf("unexpected written count: %d", summary.Written)
	}
	if !strings.HasSuffix(summary.Manifest, "terraform/aws/6.31.0/docs/_manifest.json") {
		t.Fatalf("unexpected manifest path: %s", summary.Manifest)
	}
}

func TestExportDocs_CleanRemovesExistingSubtree(t *testing.T) {
	outDir := t.TempDir()
	stalePath := filepath.Join(outDir, "terraform", "aws", "6.31.0", "docs", "old", "stale.md")
	if err := os.MkdirAll(filepath.Dir(stalePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stalePath, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := &fakeAPIClient{}
	_, err := ExportDocs(context.Background(), client, ExportOptions{
		Namespace:  "hashicorp",
		Name:       "aws",
		Version:    "6.31.0",
		Format:     "markdown",
		OutDir:     outDir,
		Categories: []string{"guides"},
		Clean:      true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Fatalf("expected stale file to be removed by --clean")
	}
}
