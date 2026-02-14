package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseGlobalFlags_NoCacheSkipsCachePathExpansion(t *testing.T) {
	g, rest, err := parseGlobalFlags([]string{"-no-cache", "-cache-ttl=-1s", "provider", "export"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rest) != 2 || rest[0] != "provider" || rest[1] != "export" {
		t.Fatalf("unexpected remaining args: %#v", rest)
	}
	if !strings.HasPrefix(g.cacheDir, "~") {
		t.Fatalf("expected cache dir to remain unexpanded in -no-cache mode, got %q", g.cacheDir)
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
	_, _, err := parseGlobalFlags([]string{"-cache-dir", "", "provider", "export"})
	if err == nil {
		t.Fatalf("expected error for empty -cache-dir")
	}
	if !strings.Contains(err.Error(), "-cache-dir must not be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseGlobalFlags_RejectsTildeUserCacheDirWhenCacheEnabled(t *testing.T) {
	_, _, err := parseGlobalFlags([]string{"-cache-dir", "~foo/cache", "provider", "export"})
	if err == nil {
		t.Fatalf("expected error for unsupported home path style")
	}
	if !strings.Contains(err.Error(), "unsupported home path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecute_UnknownProviderExportFlagReturnsExitCode1(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := Execute([]string{
		"provider", "export",
		"-unknown",
	}, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
}

func TestExecute_ProviderExportExtraArgsReturnsExitCode1(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := Execute([]string{
		"provider", "export",
		"-name", "aws",
		"-version", "6.31.0",
		"-out-dir", t.TempDir(),
		"extra",
	}, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stderr=%s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "unexpected positional arguments") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestExecute_InvalidRegistryURLReturnsExitCode1(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Execute([]string{
		"-registry-url", "://bad-url",
		"provider", "export",
		"-name", "aws",
		"-version", "6.31.0",
		"-out-dir", t.TempDir(),
	}, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stderr=%s", code, errOut.String())
	}
}

func TestExecute_UnsupportedRegistryURLSchemeReturnsExitCode1(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Execute([]string{
		"-registry-url", "ftp://registry.terraform.io",
		"provider", "export",
		"-name", "aws",
		"-version", "6.31.0",
		"-out-dir", t.TempDir(),
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
		"-cache-dir", cacheFile,
		"provider", "export",
		"-name", "aws",
		"-version", "6.31.0",
		"-out-dir", t.TempDir(),
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
		"-cache-dir", cacheFile,
		"provider", "export",
		"-version", "6.31.0",
		"-out-dir", t.TempDir(),
	}, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stderr=%s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "-name is required") {
		t.Fatalf("expected name validation error, got: %s", errOut.String())
	}
	if strings.Contains(errOut.String(), "failed to initialize cache") {
		t.Fatalf("cache init must not run before validation: %s", errOut.String())
	}
}

// --- chdir / lockfile tests ---

func TestParseGlobalFlags_ChdirIsParsed(t *testing.T) {
	g, rest, err := parseGlobalFlags([]string{"-chdir", "/tmp/proj", "provider", "export"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.chdir != "/tmp/proj" {
		t.Fatalf("expected chdir=/tmp/proj, got %q", g.chdir)
	}
	if len(rest) != 2 || rest[0] != "provider" || rest[1] != "export" {
		t.Fatalf("unexpected remaining args: %#v", rest)
	}
}

func TestResolveLockfilePath_ExplicitLockfile(t *testing.T) {
	got := resolveLockfilePath("/explicit/path.hcl", "/some/chdir")
	if got != "/explicit/path.hcl" {
		t.Fatalf("expected explicit path, got %q", got)
	}
}

func TestResolveLockfilePath_ChdirAutoDetect(t *testing.T) {
	got := resolveLockfilePath("", "/my/project")
	want := filepath.Join("/my/project", ".terraform.lock.hcl")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolveLockfilePath_NeitherSpecified(t *testing.T) {
	got := resolveLockfilePath("", "")
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestExecute_LockfileNotFoundReturnsError(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Execute([]string{
		"provider", "export",
		"-lockfile", "/nonexistent/.terraform.lock.hcl",
		"-out-dir", t.TempDir(),
	}, &out, &errOut)
	if code == 0 {
		t.Fatalf("expected non-zero exit code for missing lockfile")
	}
	if !strings.Contains(errOut.String(), "lockfile") {
		t.Fatalf("expected lockfile error in stderr, got: %s", errOut.String())
	}
}

func TestExecute_ChdirAutoDetectsLockfile(t *testing.T) {
	projDir := t.TempDir()
	lockContent := `
provider "registry.terraform.io/hashicorp/null" {
  version = "3.2.0"
}
`
	if err := os.WriteFile(filepath.Join(projDir, ".terraform.lock.hcl"), []byte(lockContent), 0o644); err != nil {
		t.Fatalf("failed to write lockfile: %v", err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	// This will fail at the registry call (no real server), but it should get past
	// lockfile parsing and validation. We verify that lockfile was found.
	code := Execute([]string{
		"-chdir", projDir,
		"provider", "export",
		"-out-dir", t.TempDir(),
	}, &out, &errOut)
	// Exit code should NOT be 1 (validation error) - it should be a network/registry error (code 3).
	// If lockfile wasn't found, we'd get a validation error about -name being required.
	if code == 1 && strings.Contains(errOut.String(), "-name is required") {
		t.Fatalf("lockfile auto-detection failed: got -name validation error instead of lockfile mode")
	}
}

func TestExecute_LockfileWithNameFilter_NotFound(t *testing.T) {
	lockContent := `
provider "registry.terraform.io/hashicorp/aws" {
  version = "5.31.0"
}
`
	lockPath := filepath.Join(t.TempDir(), ".terraform.lock.hcl")
	if err := os.WriteFile(lockPath, []byte(lockContent), 0o644); err != nil {
		t.Fatalf("failed to write lockfile: %v", err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Execute([]string{
		"provider", "export",
		"-lockfile", lockPath,
		"-name", "nonexistent",
		"-out-dir", t.TempDir(),
	}, &out, &errOut)
	if code != 2 {
		t.Fatalf("expected exit code 2 (not found), got %d; stderr=%s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "not found in lockfile") {
		t.Fatalf("expected not-found error, got: %s", errOut.String())
	}
}

func TestExecute_LockfileVersionWarning(t *testing.T) {
	lockContent := `
provider "registry.terraform.io/hashicorp/null" {
  version = "3.2.0"
}
`
	lockPath := filepath.Join(t.TempDir(), ".terraform.lock.hcl")
	if err := os.WriteFile(lockPath, []byte(lockContent), 0o644); err != nil {
		t.Fatalf("failed to write lockfile: %v", err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	// Will fail at registry call, but we check for the warning in stderr.
	_ = Execute([]string{
		"provider", "export",
		"-lockfile", lockPath,
		"-version", "ignored",
		"-out-dir", t.TempDir(),
	}, &out, &errOut)
	if !strings.Contains(errOut.String(), "-version is ignored") {
		t.Fatalf("expected -version warning, got stderr: %s", errOut.String())
	}
}

func TestExecute_LockfileEmptyReturnsError(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), ".terraform.lock.hcl")
	if err := os.WriteFile(lockPath, []byte(""), 0o644); err != nil {
		t.Fatalf("failed to write lockfile: %v", err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Execute([]string{
		"provider", "export",
		"-lockfile", lockPath,
		"-out-dir", t.TempDir(),
	}, &out, &errOut)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d; stderr=%s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "no providers found") {
		t.Fatalf("expected empty lockfile error, got: %s", errOut.String())
	}
}

func TestExecute_LegacyModeStillRequiresName(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Execute([]string{
		"provider", "export",
		"-version", "5.31.0",
		"-out-dir", t.TempDir(),
	}, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stderr=%s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "-name is required") {
		t.Fatalf("expected -name required error, got: %s", errOut.String())
	}
}
