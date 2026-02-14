package lockfile

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// ProviderLock represents a single provider entry in .terraform.lock.hcl.
type ProviderLock struct {
	Address   string // e.g. "registry.terraform.io/hashicorp/aws"
	Namespace string // e.g. "hashicorp"
	Name      string // e.g. "aws"
	Version   string // e.g. "5.31.0"
}

// ParseError indicates a failure to parse a lock file.
type ParseError struct {
	Path string
	Err  error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("failed to parse lockfile %s: %v", e.Path, e.Err)
}

func (e *ParseError) Unwrap() error { return e.Err }

var rootSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "provider", LabelNames: []string{"source_addr"}},
	},
}

var providerBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "version", Required: true},
	},
}

// ParseFile reads a .terraform.lock.hcl file and returns the provider locks it contains.
func ParseFile(path string) ([]ProviderLock, error) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCLFile(path)
	if diags.HasErrors() {
		return nil, &ParseError{Path: path, Err: diags}
	}

	content, _, diags := file.Body.PartialContent(rootSchema)
	if diags.HasErrors() {
		return nil, &ParseError{Path: path, Err: diags}
	}

	var locks []ProviderLock
	for _, block := range content.Blocks {
		if block.Type != "provider" {
			continue
		}

		addr := block.Labels[0]
		namespace, name, err := parseProviderAddress(addr)
		if err != nil {
			return nil, &ParseError{Path: path, Err: fmt.Errorf("provider %q: %w", addr, err)}
		}

		attrs, _, diags := block.Body.PartialContent(providerBlockSchema)
		if diags.HasErrors() {
			return nil, &ParseError{Path: path, Err: diags}
		}

		versionAttr, ok := attrs.Attributes["version"]
		if !ok {
			return nil, &ParseError{Path: path, Err: fmt.Errorf("provider %q: missing version attribute", addr)}
		}

		var version string
		diags = gohcl.DecodeExpression(versionAttr.Expr, nil, &version)
		if diags.HasErrors() {
			return nil, &ParseError{Path: path, Err: diags}
		}

		locks = append(locks, ProviderLock{
			Address:   addr,
			Namespace: namespace,
			Name:      name,
			Version:   version,
		})
	}

	return locks, nil
}

// parseProviderAddress extracts namespace and name from a provider address like
// "registry.terraform.io/hashicorp/aws" → ("hashicorp", "aws").
func parseProviderAddress(addr string) (namespace, name string, err error) {
	parts := strings.Split(addr, "/")
	if len(parts) < 3 {
		return "", "", fmt.Errorf("invalid provider address: expected hostname/namespace/name, got %q", addr)
	}
	// Take the last two segments as namespace and name, allowing for hostnames
	// with multiple parts (though uncommon).
	namespace = parts[len(parts)-2]
	name = parts[len(parts)-1]
	if namespace == "" || name == "" {
		return "", "", fmt.Errorf("invalid provider address: empty namespace or name in %q", addr)
	}
	return namespace, name, nil
}
