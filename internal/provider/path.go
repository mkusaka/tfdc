package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const DefaultPathTemplate = "{out}/terraform/{provider}/{version}/docs/{category}/{slug}.{ext}"

var (
	reInvalidSegment = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	rePlaceholder    = regexp.MustCompile(`\{[a-zA-Z0-9_]+\}`)
)

func BuildOutputPath(template string, vars map[string]string, outDir string) (string, error) {
	result := template
	for key, value := range vars {
		result = strings.ReplaceAll(result, "{"+key+"}", value)
	}

	if unresolved := rePlaceholder.FindString(result); unresolved != "" {
		return "", fmt.Errorf("unresolved placeholder in path template: %s", unresolved)
	}

	cleaned := filepath.Clean(result)
	outAbs, err := filepath.Abs(outDir)
	if err != nil {
		return "", err
	}
	pathAbs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", err
	}

	if pathAbs != outAbs && !strings.HasPrefix(pathAbs, outAbs+string(os.PathSeparator)) {
		return "", fmt.Errorf("output path is outside --out-dir: %s", pathAbs)
	}

	return pathAbs, nil
}

func sanitizeSegment(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = reInvalidSegment.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-.")
	if s == "" {
		return "unknown"
	}
	return s
}

func extensionForFormat(format string) (string, error) {
	switch format {
	case "markdown":
		return "md", nil
	case "json":
		return "json", nil
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}
