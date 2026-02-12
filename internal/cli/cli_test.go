package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseGlobalFlags_NoCacheSkipsCachePathExpansion(t *testing.T) {
	g, rest, err := parseGlobalFlags([]string{"--no-cache", "--cache-ttl=-1s", "provider", "export"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rest) != 2 || rest[0] != "provider" || rest[1] != "export" {
		t.Fatalf("unexpected remaining args: %#v", rest)
	}
	if !strings.HasPrefix(g.cacheDir, "~") {
		t.Fatalf("expected cache dir to remain unexpanded in --no-cache mode, got %q", g.cacheDir)
	}
}

func TestParseGlobalFlags_CacheEnabledExpandsCachePath(t *testing.T) {
	g, _, err := parseGlobalFlags([]string{"provider", "export"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.HasPrefix(g.cacheDir, "~") {
		t.Fatalf("expected cache dir to be expanded when cache is enabled, got %q", g.cacheDir)
	}
}

func TestParseGlobalFlags_RejectsEmptyCacheDirWhenCacheEnabled(t *testing.T) {
	_, _, err := parseGlobalFlags([]string{"--cache-dir", "", "provider", "export"})
	if err == nil {
		t.Fatalf("expected error for empty --cache-dir")
	}
	if !strings.Contains(err.Error(), "--cache-dir must not be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecute_UnknownProviderExportFlagReturnsExitCode1(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := Execute([]string{
		"provider", "export",
		"--unknown",
	}, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
}

func TestExecute_InvalidRegistryURLReturnsExitCode1(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Execute([]string{
		"--registry-url", "://bad-url",
		"provider", "export",
		"--name", "aws",
		"--version", "6.31.0",
		"--out-dir", t.TempDir(),
	}, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stderr=%s", code, errOut.String())
	}
}

func TestExecute_UnsupportedRegistryURLSchemeReturnsExitCode1(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Execute([]string{
		"--registry-url", "ftp://registry.terraform.io",
		"provider", "export",
		"--name", "aws",
		"--version", "6.31.0",
		"--out-dir", t.TempDir(),
	}, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stderr=%s", code, errOut.String())
	}
}

func TestExecute_CacheInitFailureReturnsExitCode4(t *testing.T) {
	cacheFile := filepath.Join(t.TempDir(), "cache-file")
	if err := os.WriteFile(cacheFile, []byte("not-a-dir"), 0o644); err != nil {
		t.Fatalf("failed to prepare cache file: %v", err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Execute([]string{
		"--cache-dir", cacheFile,
		"provider", "export",
		"--name", "aws",
		"--version", "6.31.0",
		"--out-dir", t.TempDir(),
	}, &out, &errOut)
	if code != 4 {
		t.Fatalf("expected exit code 4, got %d; stderr=%s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "failed to initialize cache") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestExecute_ValidationPrecedesCacheInit(t *testing.T) {
	cacheFile := filepath.Join(t.TempDir(), "cache-file")
	if err := os.WriteFile(cacheFile, []byte("not-a-dir"), 0o644); err != nil {
		t.Fatalf("failed to prepare cache file: %v", err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Execute([]string{
		"--cache-dir", cacheFile,
		"provider", "export",
		"--version", "6.31.0",
		"--out-dir", t.TempDir(),
	}, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stderr=%s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "--name is required") {
		t.Fatalf("expected name validation error, got: %s", errOut.String())
	}
	if strings.Contains(errOut.String(), "failed to initialize cache") {
		t.Fatalf("cache init must not run before validation: %s", errOut.String())
	}
}
