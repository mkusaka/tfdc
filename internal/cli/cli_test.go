package cli

import (
	"bytes"
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
