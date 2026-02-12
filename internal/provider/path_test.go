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

func TestBuildOutputPath_RejectsUnresolvedPlaceholderWithHyphen(t *testing.T) {
	outDir := t.TempDir()
	tpl := "{out}/terraform/{namespace}/{provider}/{version}/docs/{unknown-placeholder}/{slug}.{ext}"
	vars := map[string]string{
		"out":       outDir,
		"namespace": "hashicorp",
		"provider":  "aws",
		"version":   "6.31.0",
		"slug":      "tag-policy-compliance",
		"ext":       "md",
	}
	_, err := BuildOutputPath(tpl, vars, outDir)
	if err == nil {
		t.Fatalf("expected unresolved placeholder error")
	}
	if !strings.Contains(err.Error(), "unresolved placeholder") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "{unknown-placeholder}") {
		t.Fatalf("unexpected placeholder in error: %v", err)
	}
}

func TestBuildOutputPath_RejectsOutDirAncestorSymlink(t *testing.T) {
	rootDir := t.TempDir()
	externalDir := t.TempDir()

	symlinkParent := filepath.Join(rootDir, "link")
	if err := os.Symlink(externalDir, symlinkParent); err != nil {
		t.Skipf("symlink is not supported on this platform: %v", err)
	}

	outDir := filepath.Join(symlinkParent, "out")
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
		t.Fatalf("expected symlink traversal error for out-dir ancestor")
	}
	if !strings.Contains(err.Error(), "crosses symlink") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildOutputPath_AllowsBracesInVariableValues(t *testing.T) {
	rootDir := t.TempDir()
	outDir := filepath.Join(rootDir, "a{b}")
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
		t.Fatalf("expected path to be valid, got error: %v", err)
	}

	want := filepath.Join(outDir, "terraform", "hashicorp", "aws", "6.31.0", "docs", "guides", "tag-policy-compliance.md")
	if got != want {
		t.Fatalf("unexpected path\nwant: %s\ngot:  %s", want, got)
	}
}

func TestBuildOutputPath_DoesNotExpandPlaceholderTokensInsideValues(t *testing.T) {
	rootDir := t.TempDir()
	outDir := filepath.Join(rootDir, "a{namespace}")
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

	want := filepath.Join(outDir, "terraform", "hashicorp", "aws", "6.31.0", "docs", "guides", "tag-policy-compliance.md")
	for i := 0; i < 128; i++ {
		got, err := BuildOutputPath(tpl, vars, outDir)
		if err != nil {
			t.Fatalf("iteration %d: expected path to be valid, got error: %v", i, err)
		}
		if got != want {
			t.Fatalf("iteration %d: unexpected path\nwant: %s\ngot:  %s", i, want, got)
		}
	}
}
