package cli

import (
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
