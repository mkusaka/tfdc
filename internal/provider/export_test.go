package provider

import (
	"context"
	"encoding/json"
	"errors"
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

type fakeVersionNotFoundClient struct{}

func (f *fakeVersionNotFoundClient) GetJSON(_ context.Context, path string, dst any) error {
	if strings.HasPrefix(path, "/v2/providers/hashicorp/aws") {
		data := map[string]any{
			"included": []any{
				map[string]any{
					"type": "provider-versions",
					"id":   "70800",
					"attributes": map[string]any{
						"version": "0.0.1",
					},
				},
			},
		}
		b, _ := json.Marshal(data)
		return json.Unmarshal(b, dst)
	}
	return fmt.Errorf("unexpected GetJSON path: %s", path)
}

func (f *fakeVersionNotFoundClient) Get(_ context.Context, path string) ([]byte, error) {
	return nil, fmt.Errorf("unexpected Get path: %s", path)
}

type fakeCollisionClient struct{}

func (f *fakeCollisionClient) GetJSON(_ context.Context, path string, dst any) error {
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
				"id": "100",
				"attributes": map[string]any{
					"category": "guides",
					"slug":     "duplicate",
					"title":    "Guide Duplicate",
				},
			}}
		case cat == "resources" && page == "1":
			data = []map[string]any{{
				"id": "101",
				"attributes": map[string]any{
					"category": "resources",
					"slug":     "duplicate",
					"title":    "Resource Duplicate",
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

func (f *fakeCollisionClient) Get(_ context.Context, path string) ([]byte, error) {
	switch path {
	case "/v2/provider-docs/100":
		return []byte(`{"data":{"id":"100","attributes":{"category":"guides","slug":"duplicate","title":"Guide Duplicate","content":"# g"}}}`), nil
	case "/v2/provider-docs/101":
		return []byte(`{"data":{"id":"101","attributes":{"category":"resources","slug":"duplicate","title":"Resource Duplicate","content":"# r"}}}`), nil
	default:
		return nil, fmt.Errorf("unexpected Get path: %s", path)
	}
}

type fakeDetailRecoverClient struct{}

func (f *fakeDetailRecoverClient) GetJSON(_ context.Context, path string, dst any) error {
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
		if cat == "guides" && page == "1" {
			b, _ := json.Marshal(map[string]any{
				"data": []map[string]any{{
					"id": "1",
					"attributes": map[string]any{
						"category": "guides",
						"slug":     "tag-policy-compliance",
						"title":    "Tag Policy Compliance",
					},
				}},
			})
			return json.Unmarshal(b, dst)
		}
		b, _ := json.Marshal(map[string]any{"data": []any{}})
		return json.Unmarshal(b, dst)
	}

	if strings.HasPrefix(path, "/v2/provider-docs/1") {
		b, _ := json.Marshal(map[string]any{
			"data": map[string]any{
				"id": "1",
				"attributes": map[string]any{
					"category": "guides",
					"slug":     "tag-policy-compliance",
					"title":    "Tag Policy Compliance",
					"content":  "# guide content",
				},
			},
		})
		return json.Unmarshal(b, dst)
	}

	return fmt.Errorf("unexpected GetJSON path: %s", path)
}

func (f *fakeDetailRecoverClient) Get(_ context.Context, path string) ([]byte, error) {
	if strings.HasPrefix(path, "/v2/provider-docs/1") {
		return []byte("not-json"), nil
	}
	return nil, fmt.Errorf("unexpected Get path: %s", path)
}

type fakeDetailRecoverRefetchErrorClient struct {
	refetchErr error
}

func (f *fakeDetailRecoverRefetchErrorClient) GetJSON(_ context.Context, path string, dst any) error {
	if strings.HasPrefix(path, "/v2/provider-docs/1") {
		return f.refetchErr
	}
	return fmt.Errorf("unexpected GetJSON path: %s", path)
}

func (f *fakeDetailRecoverRefetchErrorClient) Get(_ context.Context, path string) ([]byte, error) {
	if strings.HasPrefix(path, "/v2/provider-docs/1") {
		return []byte("not-json"), nil
	}
	return nil, fmt.Errorf("unexpected Get path: %s", path)
}

type fakeDetailRecoverRawPreserveClient struct {
	getDetailCalls int
}

func (f *fakeDetailRecoverRawPreserveClient) GetJSON(_ context.Context, path string, dst any) error {
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
		if cat == "guides" && page == "1" {
			b, _ := json.Marshal(map[string]any{
				"data": []map[string]any{{
					"id": "1",
					"attributes": map[string]any{
						"category": "guides",
						"slug":     "tag-policy-compliance",
						"title":    "Tag Policy Compliance",
					},
				}},
			})
			return json.Unmarshal(b, dst)
		}
		b, _ := json.Marshal(map[string]any{"data": []any{}})
		return json.Unmarshal(b, dst)
	}

	if strings.HasPrefix(path, "/v2/provider-docs/1") {
		const recoveredRaw = `{"data":{"id":"1","type":"provider-docs","links":{"self":"https://registry.terraform.io/v2/provider-docs/1"},"attributes":{"category":"guides","subcategory":"policy","language":"hcl","truncated":false,"slug":"tag-policy-compliance","title":"Tag Policy Compliance","content":"# guide content"}}}`
		return json.Unmarshal([]byte(recoveredRaw), dst)
	}

	return fmt.Errorf("unexpected GetJSON path: %s", path)
}

func (f *fakeDetailRecoverRawPreserveClient) Get(_ context.Context, path string) ([]byte, error) {
	if strings.HasPrefix(path, "/v2/provider-docs/1") {
		f.getDetailCalls++
		if f.getDetailCalls == 1 {
			return []byte("not-json"), nil
		}
		return []byte(`{"data":{"id":"1","type":"provider-docs","links":{"self":"https://registry.terraform.io/v2/provider-docs/1"},"attributes":{"category":"guides","subcategory":"policy","language":"hcl","truncated":false,"slug":"tag-policy-compliance","title":"Tag Policy Compliance","content":"# guide content"}}}`), nil
	}
	return nil, fmt.Errorf("unexpected Get path: %s", path)
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

	guidePath := filepath.Join(outDir, "terraform", "hashicorp", "aws", "6.31.0", "docs", "guides", "tag-policy-compliance.md")
	resourcePath := filepath.Join(outDir, "terraform", "hashicorp", "aws", "6.31.0", "docs", "resources", "aws_s3_bucket.md")
	manifestPath := filepath.Join(outDir, "terraform", "hashicorp", "aws", "6.31.0", "docs", "_manifest.json")

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
	if !strings.HasSuffix(summary.Manifest, "terraform/hashicorp/aws/6.31.0/docs/_manifest.json") {
		t.Fatalf("unexpected manifest path: %s", summary.Manifest)
	}
}

func TestExportDocs_RecoversFromInvalidDetailJSONViaGetJSON(t *testing.T) {
	outDir := t.TempDir()
	client := &fakeDetailRecoverClient{}

	summary, err := ExportDocs(context.Background(), client, ExportOptions{
		Namespace:  "hashicorp",
		Name:       "aws",
		Version:    "6.31.0",
		Format:     "markdown",
		OutDir:     outDir,
		Categories: []string{"guides"},
		Clean:      false,
	})
	if err != nil {
		t.Fatal(err)
	}

	guidePath := filepath.Join(outDir, "terraform", "hashicorp", "aws", "6.31.0", "docs", "guides", "tag-policy-compliance.md")
	if _, err := os.Stat(guidePath); err != nil {
		t.Fatalf("expected guide file to be written: %v", err)
	}
	if summary.Written != 1 {
		t.Fatalf("unexpected written count: %d", summary.Written)
	}
}

func TestGetProviderDocDetail_PropagatesRefetchError(t *testing.T) {
	wantErr := &NotFoundError{Message: "provider doc not found"}
	client := &fakeDetailRecoverRefetchErrorClient{refetchErr: wantErr}

	_, _, err := getProviderDocDetail(context.Background(), client, "1")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected refetch error to be propagated, got %T (%v)", err, err)
	}
}

func TestExportDocs_JSONRecoveryPreservesRawFields(t *testing.T) {
	outDir := t.TempDir()
	client := &fakeDetailRecoverRawPreserveClient{}

	summary, err := ExportDocs(context.Background(), client, ExportOptions{
		Namespace:  "hashicorp",
		Name:       "aws",
		Version:    "6.31.0",
		Format:     "json",
		OutDir:     outDir,
		Categories: []string{"guides"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if summary.Written != 1 {
		t.Fatalf("unexpected written count: %d", summary.Written)
	}

	guidePath := filepath.Join(outDir, "terraform", "hashicorp", "aws", "6.31.0", "docs", "guides", "tag-policy-compliance.json")
	body, err := os.ReadFile(guidePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"links":`) {
		t.Fatalf("expected recovered raw json to keep data.links, got: %s", string(body))
	}
	if !strings.Contains(string(body), `"language": "hcl"`) {
		t.Fatalf("expected recovered raw json to keep attributes.language, got: %s", string(body))
	}
	if client.getDetailCalls != 2 {
		t.Fatalf("expected detail endpoint to be read twice (initial+recovered), got %d", client.getDetailCalls)
	}
}

func TestExportDocs_CleanRemovesExistingSubtree(t *testing.T) {
	outDir := t.TempDir()
	stalePath := filepath.Join(outDir, "terraform", "hashicorp", "aws", "6.31.0", "docs", "old", "stale.md")
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

func TestExportDocs_CleanDoesNotDeleteWhenVersionResolutionFails(t *testing.T) {
	outDir := t.TempDir()
	stalePath := filepath.Join(outDir, "terraform", "hashicorp", "aws", "6.31.0", "docs", "guides", "stale.md")
	if err := os.MkdirAll(filepath.Dir(stalePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stalePath, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := &fakeVersionNotFoundClient{}
	_, err := ExportDocs(context.Background(), client, ExportOptions{
		Namespace:  "hashicorp",
		Name:       "aws",
		Version:    "6.31.0",
		Format:     "markdown",
		OutDir:     outDir,
		Categories: []string{"guides"},
		Clean:      true,
	})
	if err == nil {
		t.Fatalf("expected error")
	}

	if _, err := os.Stat(stalePath); err != nil {
		t.Fatalf("expected stale file to remain when version resolution fails: %v", err)
	}
}

func TestExportDocs_CleanWithBracesInOutDir(t *testing.T) {
	rootDir := t.TempDir()
	outDir := filepath.Join(rootDir, "a{b}")

	stalePath := filepath.Join(outDir, "terraform", "hashicorp", "aws", "6.31.0", "docs", "old", "stale.md")
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

	guidePath := filepath.Join(outDir, "terraform", "hashicorp", "aws", "6.31.0", "docs", "guides", "tag-policy-compliance.md")
	if _, err := os.Stat(guidePath); err != nil {
		t.Fatalf("expected guide file to be written: %v", err)
	}
}

func TestExportDocs_CleanUsesPathTemplateRoot(t *testing.T) {
	outDir := t.TempDir()
	staleCustom := filepath.Join(outDir, "custom", "guides", "stale.md")
	if err := os.MkdirAll(filepath.Dir(staleCustom), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(staleCustom, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := &fakeAPIClient{}
	_, err := ExportDocs(context.Background(), client, ExportOptions{
		Namespace:    "hashicorp",
		Name:         "aws",
		Version:      "6.31.0",
		Format:       "markdown",
		OutDir:       outDir,
		Categories:   []string{"guides"},
		PathTemplate: "{out}/custom/{category}/{slug}.{ext}",
		Clean:        true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(staleCustom); !os.IsNotExist(err) {
		t.Fatalf("expected stale custom file to be removed by --clean with custom template")
	}

	newGuide := filepath.Join(outDir, "custom", "guides", "tag-policy-compliance.md")
	if _, err := os.Stat(newGuide); err != nil {
		t.Fatalf("expected exported guide in custom template path: %v", err)
	}
}

func TestExportDocs_CleanUsesRelativePathTemplateRoot(t *testing.T) {
	outDir := t.TempDir()
	staleCustom := filepath.Join(outDir, "custom", "guides", "stale.md")
	if err := os.MkdirAll(filepath.Dir(staleCustom), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(staleCustom, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := &fakeAPIClient{}
	_, err := ExportDocs(context.Background(), client, ExportOptions{
		Namespace:    "hashicorp",
		Name:         "aws",
		Version:      "6.31.0",
		Format:       "markdown",
		OutDir:       outDir,
		Categories:   []string{"guides"},
		PathTemplate: "custom/{category}/{slug}.{ext}",
		Clean:        true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(staleCustom); !os.IsNotExist(err) {
		t.Fatalf("expected stale custom file to be removed by --clean with relative template")
	}

	newGuide := filepath.Join(outDir, "custom", "guides", "tag-policy-compliance.md")
	if _, err := os.Stat(newGuide); err != nil {
		t.Fatalf("expected exported guide in relative template path: %v", err)
	}
}

func TestExportDocs_CleanRejectsSymlinkedTargetOutsideOutDir(t *testing.T) {
	outDir := t.TempDir()
	externalDir := t.TempDir()

	if err := os.Symlink(externalDir, filepath.Join(outDir, "terraform")); err != nil {
		t.Skipf("symlink is not supported on this platform: %v", err)
	}

	externalVictim := filepath.Join(externalDir, "hashicorp", "aws", "6.31.0", "docs", "victim.txt")
	if err := os.MkdirAll(filepath.Dir(externalVictim), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(externalVictim, []byte("do-not-delete"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := &fakeAPIClient{}
	_, err := ExportDocs(context.Background(), client, ExportOptions{
		Namespace:    "hashicorp",
		Name:         "aws",
		Version:      "6.31.0",
		Format:       "markdown",
		OutDir:       outDir,
		Categories:   []string{"guides"},
		PathTemplate: "{out}/custom/{category}/{slug}.{ext}",
		Clean:        true,
	})
	if err == nil {
		t.Fatalf("expected error for symlinked clean target")
	}
	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected validation error, got %T (%v)", err, err)
	}
	if !strings.Contains(vErr.Error(), "unsafe --clean target") {
		t.Fatalf("unexpected error message: %s", vErr.Error())
	}

	if _, err := os.Stat(externalVictim); err != nil {
		t.Fatalf("expected external file to remain untouched: %v", err)
	}
}

func TestExportDocs_CleanRejectsOutDirAncestorSymlink(t *testing.T) {
	rootDir := t.TempDir()
	externalDir := t.TempDir()

	symlinkParent := filepath.Join(rootDir, "link")
	if err := os.Symlink(externalDir, symlinkParent); err != nil {
		t.Skipf("symlink is not supported on this platform: %v", err)
	}
	outDir := filepath.Join(symlinkParent, "out")

	externalVictim := filepath.Join(externalDir, "out", "terraform", "hashicorp", "aws", "6.31.0", "docs", "old", "stale.md")
	if err := os.MkdirAll(filepath.Dir(externalVictim), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(externalVictim, []byte("do-not-delete"), 0o644); err != nil {
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
	if err == nil {
		t.Fatalf("expected validation error for out-dir ancestor symlink")
	}
	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected validation error, got %T (%v)", err, err)
	}
	if !strings.Contains(vErr.Error(), "crosses symlink") {
		t.Fatalf("unexpected error message: %s", vErr.Error())
	}

	if _, err := os.Stat(externalVictim); err != nil {
		t.Fatalf("expected external file to remain untouched: %v", err)
	}
}

func TestExportDocs_PathTemplateCollisionReturnsValidationError(t *testing.T) {
	outDir := t.TempDir()
	client := &fakeCollisionClient{}
	_, err := ExportDocs(context.Background(), client, ExportOptions{
		Namespace:    "hashicorp",
		Name:         "aws",
		Version:      "6.31.0",
		Format:       "markdown",
		OutDir:       outDir,
		Categories:   []string{"guides", "resources"},
		PathTemplate: "{out}/flat/{slug}.{ext}",
	})
	if err == nil {
		t.Fatalf("expected path collision error")
	}
	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected validation error, got %T (%v)", err, err)
	}
	if !strings.Contains(vErr.Error(), "path collision detected") {
		t.Fatalf("unexpected error message: %s", vErr.Error())
	}
}

func TestExportDocs_PathTemplateCollisionWithManifestReturnsValidationError(t *testing.T) {
	outDir := t.TempDir()
	client := &fakeAPIClient{}
	_, err := ExportDocs(context.Background(), client, ExportOptions{
		Namespace:    "hashicorp",
		Name:         "aws",
		Version:      "6.31.0",
		Format:       "markdown",
		OutDir:       outDir,
		Categories:   []string{"guides"},
		PathTemplate: "{out}/terraform/{namespace}/{provider}/{version}/docs/_manifest.json",
	})
	if err == nil {
		t.Fatalf("expected path collision with manifest")
	}
	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected validation error, got %T (%v)", err, err)
	}
	if !strings.Contains(vErr.Error(), "reserved manifest path") {
		t.Fatalf("unexpected error message: %s", vErr.Error())
	}
}

func TestExportDocs_PathTemplateCollisionWithManifestFailsWhenNoDocsFound(t *testing.T) {
	outDir := t.TempDir()
	client := &fakeAPIClient{}
	_, err := ExportDocs(context.Background(), client, ExportOptions{
		Namespace:    "hashicorp",
		Name:         "aws",
		Version:      "6.31.0",
		Format:       "markdown",
		OutDir:       outDir,
		Categories:   []string{"functions"},
		PathTemplate: "{out}/terraform/{namespace}/{provider}/{version}/docs/_manifest.json",
	})
	if err == nil {
		t.Fatalf("expected path collision with manifest")
	}
	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected validation error, got %T (%v)", err, err)
	}
	if !strings.Contains(vErr.Error(), "reserved manifest path") {
		t.Fatalf("unexpected error message: %s", vErr.Error())
	}
}

func TestExportDocs_InvalidPathTemplateFailsWhenNoDocsFound(t *testing.T) {
	outDir := t.TempDir()
	client := &fakeAPIClient{}
	_, err := ExportDocs(context.Background(), client, ExportOptions{
		Namespace:    "hashicorp",
		Name:         "aws",
		Version:      "6.31.0",
		Format:       "markdown",
		OutDir:       outDir,
		Categories:   []string{"functions"},
		PathTemplate: "{out}/custom/{unknown}/{slug}.{ext}",
	})
	if err == nil {
		t.Fatalf("expected validation error for unresolved placeholder")
	}
	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected validation error, got %T (%v)", err, err)
	}
	if !strings.Contains(vErr.Error(), "unresolved placeholder") {
		t.Fatalf("unexpected error message: %s", vErr.Error())
	}

	manifestPath := filepath.Join(outDir, "terraform", "hashicorp", "aws", "6.31.0", "docs", "_manifest.json")
	if _, statErr := os.Stat(manifestPath); !os.IsNotExist(statErr) {
		t.Fatalf("manifest must not be written on invalid template: %v", statErr)
	}
}

func TestExportDocs_PathTemplateOutsideOutDirFailsWhenNoDocsFound(t *testing.T) {
	outDir := t.TempDir()
	client := &fakeAPIClient{}
	_, err := ExportDocs(context.Background(), client, ExportOptions{
		Namespace:    "hashicorp",
		Name:         "aws",
		Version:      "6.31.0",
		Format:       "markdown",
		OutDir:       outDir,
		Categories:   []string{"functions"},
		PathTemplate: "{out}/../outside/{slug}.{ext}",
	})
	if err == nil {
		t.Fatalf("expected validation error for template outside out-dir")
	}
	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected validation error, got %T (%v)", err, err)
	}
	if !strings.Contains(vErr.Error(), "outside --out-dir") {
		t.Fatalf("unexpected error message: %s", vErr.Error())
	}
}

func TestNormalizeCategories_AllIncludesEphemeralResources(t *testing.T) {
	cats, err := normalizeCategories([]string{"all"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, cat := range cats {
		if cat == "ephemeral-resources" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected all categories to include ephemeral-resources, got: %v", cats)
	}
}

func TestNormalizeCategories_EphemeralResourcesAllowed(t *testing.T) {
	cats, err := normalizeCategories([]string{"ephemeral-resources"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cats) != 1 || cats[0] != "ephemeral-resources" {
		t.Fatalf("unexpected categories: %v", cats)
	}
}

func TestExportDocs_CleanKeepsLegacySharedManifestWhenNamespaceDiffers(t *testing.T) {
	outDir := t.TempDir()
	legacyManifestPath := filepath.Join(outDir, "terraform", "aws", "6.31.0", "docs", "_manifest.json")
	if err := os.MkdirAll(filepath.Dir(legacyManifestPath), 0o755); err != nil {
		t.Fatal(err)
	}
	const marker = `{"namespace":"legacy-other"}`
	if err := os.WriteFile(legacyManifestPath, []byte(marker), 0o644); err != nil {
		t.Fatal(err)
	}

	client := &fakeAPIClient{}
	_, err := ExportDocs(context.Background(), client, ExportOptions{
		Namespace:    "hashicorp",
		Name:         "aws",
		Version:      "6.31.0",
		Format:       "markdown",
		OutDir:       outDir,
		Categories:   []string{"guides"},
		PathTemplate: "{out}/custom/{namespace}/{category}/{slug}.{ext}",
		Clean:        true,
	})
	if err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(legacyManifestPath)
	if err != nil {
		t.Fatalf("expected legacy shared manifest to remain untouched: %v", err)
	}
	if string(b) != marker {
		t.Fatalf("legacy shared manifest was modified unexpectedly: %s", string(b))
	}

	namespacedManifestPath := filepath.Join(outDir, "terraform", "hashicorp", "aws", "6.31.0", "docs", "_manifest.json")
	if _, err := os.Stat(namespacedManifestPath); err != nil {
		t.Fatalf("expected namespaced manifest to be written: %v", err)
	}
}
