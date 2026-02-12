package provider

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const DefaultPathTemplate = "{out}/terraform/{namespace}/{provider}/{version}/docs/{category}/{slug}.{ext}"

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

	if !isPathWithinDir(outAbs, pathAbs) {
		return "", fmt.Errorf("output path is outside --out-dir: %s", pathAbs)
	}
	if err := ensureNoSymlinkTraversal(outAbs, pathAbs); err != nil {
		return "", fmt.Errorf("output path crosses symlink outside --out-dir: %v", err)
	}

	return pathAbs, nil
}

func isPathWithinDir(baseAbs, targetAbs string) bool {
	rel, err := filepath.Rel(baseAbs, targetAbs)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

func ensureNoSymlinkTraversal(baseAbs, targetAbs string) error {
	if !isPathWithinDir(baseAbs, targetAbs) {
		return fmt.Errorf("target is outside base dir: %s", targetAbs)
	}

	rel, err := filepath.Rel(baseAbs, targetAbs)
	if err != nil {
		return err
	}

	current := baseAbs
	if err := rejectSymlinkIfExists(current); err != nil {
		return err
	}
	if rel == "." {
		return nil
	}

	for _, segment := range strings.Split(rel, string(os.PathSeparator)) {
		if segment == "" || segment == "." {
			continue
		}
		current = filepath.Join(current, segment)
		if err := rejectSymlinkIfExists(current); err != nil {
			return err
		}
	}
	return nil
}

func rejectSymlinkIfExists(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symlink component detected: %s", path)
	}
	return nil
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
