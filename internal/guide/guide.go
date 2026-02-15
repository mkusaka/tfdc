package guide

import (
	"context"
	"fmt"
	"strings"
)

// APIClient is the interface needed for guide operations.
type APIClient interface {
	Get(ctx context.Context, path string) ([]byte, error)
}

const (
	styleURL      = "https://raw.githubusercontent.com/hashicorp/web-unified-docs/main/content/terraform/v1.12.x/docs/language/style.mdx"
	moduleDevBase = "https://raw.githubusercontent.com/hashicorp/web-unified-docs/main/content/terraform/v1.12.x/docs/language/modules/develop"
)

// ModuleDevSections lists the valid section names for module-dev guide.
var ModuleDevSections = []string{"index", "composition", "structure", "providers", "publish", "refactoring"}

// FetchStyleGuide fetches the Terraform style guide.
func FetchStyleGuide(ctx context.Context, client APIClient) (string, error) {
	b, err := client.Get(ctx, styleURL)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// FetchModuleDevGuide fetches the module development guide.
// section can be "all" or one of ModuleDevSections.
func FetchModuleDevGuide(ctx context.Context, client APIClient, section string) (string, error) {
	section = strings.ToLower(strings.TrimSpace(section))
	if section == "" || section == "all" {
		return fetchAllSections(ctx, client)
	}

	if !isValidSection(section) {
		return "", &ValidationError{Message: fmt.Sprintf("invalid -section: %s (valid: all, %s)", section, strings.Join(ModuleDevSections, ", "))}
	}

	url := fmt.Sprintf("%s/%s.mdx", moduleDevBase, section)
	b, err := client.Get(ctx, url)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func fetchAllSections(ctx context.Context, client APIClient) (string, error) {
	var parts []string
	for _, section := range ModuleDevSections {
		url := fmt.Sprintf("%s/%s.mdx", moduleDevBase, section)
		b, err := client.Get(ctx, url)
		if err != nil {
			return "", err
		}
		parts = append(parts, string(b))
	}
	return strings.Join(parts, "\n\n---\n\n"), nil
}

func isValidSection(section string) bool {
	for _, s := range ModuleDevSections {
		if s == section {
			return true
		}
	}
	return false
}

// ValidationError indicates invalid input.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }
