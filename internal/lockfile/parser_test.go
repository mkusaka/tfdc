package lockfile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestParseFile_MultipleProviders(t *testing.T) {
	content := `
provider "registry.terraform.io/hashicorp/aws" {
  version     = "5.31.0"
  constraints = "~> 5.0"
  hashes = [
    "h1:abc123",
  ]
}

provider "registry.terraform.io/hashicorp/random" {
  version = "3.6.0"
}
`
	path := writeTempLockfile(t, content)
	locks, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(locks) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(locks))
	}

	assertLock(t, locks[0], "registry.terraform.io/hashicorp/aws", "hashicorp", "aws", "5.31.0")
	assertLock(t, locks[1], "registry.terraform.io/hashicorp/random", "hashicorp", "random", "3.6.0")
}

func TestParseFile_SingleProvider(t *testing.T) {
	content := `
provider "registry.terraform.io/integrations/github" {
  version = "6.0.0"
}
`
	path := writeTempLockfile(t, content)
	locks, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(locks) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(locks))
	}
	assertLock(t, locks[0], "registry.terraform.io/integrations/github", "integrations", "github", "6.0.0")
}

func TestParseFile_EmptyFile(t *testing.T) {
	path := writeTempLockfile(t, "")
	locks, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(locks) != 0 {
		t.Fatalf("expected 0 providers, got %d", len(locks))
	}
}

func TestParseFile_FileNotFound(t *testing.T) {
	_, err := ParseFile(filepath.Join(t.TempDir(), "nonexistent.hcl"))
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *ParseError, got %T: %v", err, err)
	}
}

func TestParseFile_MissingVersion(t *testing.T) {
	content := `
provider "registry.terraform.io/hashicorp/aws" {
  constraints = "~> 5.0"
}
`
	path := writeTempLockfile(t, content)
	_, err := ParseFile(path)
	if err == nil {
		t.Fatal("expected error for missing version")
	}
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *ParseError, got %T: %v", err, err)
	}
}

func TestParseFile_InvalidAddress_TooFewParts(t *testing.T) {
	content := `
provider "hashicorp/aws" {
  version = "5.31.0"
}
`
	path := writeTempLockfile(t, content)
	_, err := ParseFile(path)
	if err == nil {
		t.Fatal("expected error for invalid address")
	}
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *ParseError, got %T: %v", err, err)
	}
}

func TestParseFile_InvalidHCL(t *testing.T) {
	content := `this is not valid HCL {{{`
	path := writeTempLockfile(t, content)
	_, err := ParseFile(path)
	if err == nil {
		t.Fatal("expected error for invalid HCL")
	}
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *ParseError, got %T: %v", err, err)
	}
}

func TestParseFile_CustomRegistry(t *testing.T) {
	content := `
provider "custom.example.com/myorg/myprovider" {
  version = "1.0.0"
}
`
	path := writeTempLockfile(t, content)
	locks, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(locks) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(locks))
	}
	assertLock(t, locks[0], "custom.example.com/myorg/myprovider", "myorg", "myprovider", "1.0.0")
}

func TestParseProviderAddress(t *testing.T) {
	tests := []struct {
		addr      string
		namespace string
		name      string
		wantErr   bool
	}{
		{"registry.terraform.io/hashicorp/aws", "hashicorp", "aws", false},
		{"registry.terraform.io/integrations/github", "integrations", "github", false},
		{"custom.example.com/myorg/myprovider", "myorg", "myprovider", false},
		{"hashicorp/aws", "", "", true},
		{"aws", "", "", true},
		{"", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			ns, name, err := parseProviderAddress(tt.addr)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tt.addr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.addr, err)
			}
			if ns != tt.namespace {
				t.Errorf("namespace: got %q, want %q", ns, tt.namespace)
			}
			if name != tt.name {
				t.Errorf("name: got %q, want %q", name, tt.name)
			}
		})
	}
}

func writeTempLockfile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".terraform.lock.hcl")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp lockfile: %v", err)
	}
	return path
}

func assertLock(t *testing.T, got ProviderLock, wantAddr, wantNS, wantName, wantVersion string) {
	t.Helper()
	if got.Address != wantAddr {
		t.Errorf("Address: got %q, want %q", got.Address, wantAddr)
	}
	if got.Namespace != wantNS {
		t.Errorf("Namespace: got %q, want %q", got.Namespace, wantNS)
	}
	if got.Name != wantName {
		t.Errorf("Name: got %q, want %q", got.Name, wantName)
	}
	if got.Version != wantVersion {
		t.Errorf("Version: got %q, want %q", got.Version, wantVersion)
	}
}
