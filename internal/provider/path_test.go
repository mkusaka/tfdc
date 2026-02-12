package provider

import (
	"path/filepath"
	"testing"
)

func TestBuildOutputPath_DefaultTemplate(t *testing.T) {
	outDir := t.TempDir()
	tpl := "{out}/terraform/{provider}/{version}/docs/{category}/{slug}.{ext}"
	vars := map[string]string{
		"out":      outDir,
		"provider": "aws",
		"version":  "6.31.0",
		"category": "guides",
		"slug":     "tag-policy-compliance",
		"ext":      "md",
	}

	got, err := BuildOutputPath(tpl, vars, outDir)
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(outDir, "terraform", "aws", "6.31.0", "docs", "guides", "tag-policy-compliance.md")
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
