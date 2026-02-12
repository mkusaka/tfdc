package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildOutputPath_DefaultTemplate(t *testing.T) {
	outDir := t.TempDir()
	tpl := "{out}/terraform/{namespace}/{provider}/{version}/docs/{category}/{slug}.{ext}"
	vars := map[string]string{
		"out":       outDir,
		"namespace": "hashicorp",
		"provider":  "aws",
		"version":   "6.31.0",
		"category":  "guides",
		"slug":      "tag-policy-compliance",
		"ext":       "md",
	}

	got, err := BuildOutputPath(tpl, vars, outDir)
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(outDir, "terraform", "hashicorp", "aws", "6.31.0", "docs", "guides", "tag-policy-compliance.md")
	if got != want {
		t.Fatalf("unexpected path\nwant: %s\ngot:  %s", want, got)
	}
}

func TestBuildOutputPath_RejectsOutsideOutDir(t *testing.T) {
	outDir := t.TempDir()
	tpl := "{out}/../outside/{slug}.md"
	vars := map[string]string{
		"out":  outDir,
		"slug": "x",
	}
	_, err := BuildOutputPath(tpl, vars, outDir)
	if err == nil {
		t.Fatalf("expected error for path outside out-dir")
	}
}

func TestBuildOutputPath_RejectsSymlinkTraversalOutsideOutDir(t *testing.T) {
	outDir := t.TempDir()
	externalDir := t.TempDir()

	guidesDir := filepath.Join(outDir, "terraform", "hashicorp", "aws", "6.31.0", "docs")
	if err := os.MkdirAll(guidesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(externalDir, filepath.Join(guidesDir, "guides")); err != nil {
		t.Skipf("symlink is not supported on this platform: %v", err)
	}

	tpl := "{out}/terraform/{namespace}/{provider}/{version}/docs/{category}/{slug}.{ext}"
	vars := map[string]string{
		"out":       outDir,
		"namespace": "hashicorp",
		"provider":  "aws",
		"version":   "6.31.0",
		"category":  "guides",
		"slug":      "tag-policy-compliance",
		"ext":       "md",
	}
	_, err := BuildOutputPath(tpl, vars, outDir)
	if err == nil {
		t.Fatalf("expected symlink traversal error")
	}
	if !strings.Contains(err.Error(), "crosses symlink") {
		t.Fatalf("unexpected error: %v", err)
	}
}
