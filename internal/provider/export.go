package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string { return e.Message }

type WriteError struct {
	Path string
	Err  error
}

func (e *WriteError) Error() string { return fmt.Sprintf("failed to write file %s: %v", e.Path, e.Err) }
func (e *WriteError) Unwrap() error { return e.Err }

type APIClient interface {
	GetJSON(ctx context.Context, path string, dst any) error
	Get(ctx context.Context, path string) ([]byte, error)
}

type ExportOptions struct {
	Namespace    string
	Name         string
	Version      string
	Format       string
	OutDir       string
	Categories   []string
	PathTemplate string
	Clean        bool
}

type ExportSummary struct {
	Provider string `json:"provider"`
	Version  string `json:"version"`
	OutDir   string `json:"out_dir"`
	Written  int    `json:"written"`
	Manifest string `json:"manifest"`
}

type providerVersionsResponse struct {
	Included []struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			Version string `json:"version"`
		} `json:"attributes"`
	} `json:"included"`
}

type providerDocsListResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Category string `json:"category"`
			Slug     string `json:"slug"`
			Title    string `json:"title"`
		} `json:"attributes"`
	} `json:"data"`
}

type providerDocDetailResponse struct {
	Data struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Category string `json:"category"`
			Path     string `json:"path"`
			Slug     string `json:"slug"`
			Title    string `json:"title"`
			Content  string `json:"content"`
		} `json:"attributes"`
	} `json:"data"`
}

type manifest struct {
	Provider    string         `json:"provider"`
	Namespace   string         `json:"namespace"`
	Version     string         `json:"version"`
	Format      string         `json:"format"`
	GeneratedAt string         `json:"generated_at"`
	Total       int            `json:"total"`
	Docs        []manifestItem `json:"docs"`
}

type manifestItem struct {
	DocID    string `json:"doc_id"`
	Category string `json:"category"`
	Slug     string `json:"slug"`
	Title    string `json:"title"`
	Path     string `json:"path"`
}

type plannedFile struct {
	path    string
	content []byte
	item    manifestItem
}

const reservedManifestPathOwner = "_manifest"

var defaultCategories = []string{
	"resources",
	"data-sources",
	"ephemeral-resources",
	"functions",
	"guides",
	"overview",
	"actions",
	"list-resources",
}

func ExportDocs(ctx context.Context, client APIClient, opts ExportOptions) (*ExportSummary, error) {
	ext, err := prepareExportOptions(&opts)
	if err != nil {
		return nil, err
	}

	providerVersionID, err := resolveProviderVersionID(ctx, client, opts.Namespace, opts.Name, opts.Version)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	planned := make([]plannedFile, 0)
	pathOwners := make(map[string]string)
	pathOwners[manifestPathForOptions(opts)] = reservedManifestPathOwner

	for _, category := range opts.Categories {
		for page := 1; ; page++ {
			docs, err := listProviderDocs(ctx, client, providerVersionID, category, page)
			if err != nil {
				return nil, err
			}
			if len(docs) == 0 {
				break
			}

			for _, doc := range docs {
				if _, exists := seen[doc.ID]; exists {
					continue
				}
				seen[doc.ID] = struct{}{}

				detail, raw, err := getProviderDocDetail(ctx, client, doc.ID)
				if err != nil {
					return nil, err
				}

				slug := detail.Data.Attributes.Slug
				if slug == "" {
					slug = doc.Attributes.Slug
				}
				if slug == "" {
					slug = detail.Data.ID
				}

				vars := map[string]string{
					"out":       opts.OutDir,
					"namespace": sanitizeSegment(opts.Namespace),
					"provider":  sanitizeSegment(opts.Name),
					"version":   sanitizeSegment(opts.Version),
					"category":  sanitizeSegment(detail.Data.Attributes.Category),
					"slug":      sanitizeSegment(slug),
					"doc_id":    sanitizeSegment(detail.Data.ID),
					"ext":       ext,
				}
				if vars["category"] == "unknown" {
					vars["category"] = sanitizeSegment(category)
				}

				filePath, err := BuildOutputPath(opts.PathTemplate, vars, opts.OutDir)
				if err != nil {
					return nil, &ValidationError{Message: err.Error()}
				}
				if existing, exists := pathOwners[filePath]; exists {
					if existing == reservedManifestPathOwner {
						return nil, &ValidationError{Message: fmt.Sprintf("path collision detected in --path-template: %s conflicts with reserved manifest path", filePath)}
					}
					return nil, &ValidationError{Message: fmt.Sprintf("path collision detected in --path-template: %s (doc_id=%s conflicts with doc_id=%s)", filePath, existing, detail.Data.ID)}
				}
				pathOwners[filePath] = detail.Data.ID

				content, err := renderContent(opts.Format, detail, raw)
				if err != nil {
					return nil, err
				}

				relPath, err := filepath.Rel(opts.OutDir, filePath)
				if err != nil {
					relPath = filePath
				}

				planned = append(planned, plannedFile{
					path:    filePath,
					content: content,
					item: manifestItem{
						DocID:    detail.Data.ID,
						Category: detail.Data.Attributes.Category,
						Slug:     slug,
						Title:    detail.Data.Attributes.Title,
						Path:     filepath.ToSlash(relPath),
					},
				})
			}
		}
	}

	sort.Slice(planned, func(i, j int) bool {
		return planned[i].item.Path < planned[j].item.Path
	})

	if opts.Clean {
		cleanTargets, err := deriveCleanTargets(opts, ext)
		if err != nil {
			return nil, err
		}
		for _, target := range cleanTargets {
			if err := ensureNoSymlinkTraversal(opts.OutDir, target); err != nil {
				return nil, &ValidationError{Message: fmt.Sprintf("unsafe --clean target %s: %v", target, err)}
			}
			if err := os.RemoveAll(target); err != nil {
				return nil, &WriteError{Path: target, Err: err}
			}
		}
	}

	manifestDocs := make([]manifestItem, 0, len(planned))
	for _, pf := range planned {
		if err := ensureNoSymlinkTraversal(opts.OutDir, pf.path); err != nil {
			return nil, &ValidationError{Message: fmt.Sprintf("unsafe output path %s: %v", pf.path, err)}
		}
		if err := os.MkdirAll(filepath.Dir(pf.path), 0o755); err != nil {
			return nil, &WriteError{Path: pf.path, Err: err}
		}
		if err := os.WriteFile(pf.path, pf.content, 0o644); err != nil {
			return nil, &WriteError{Path: pf.path, Err: err}
		}
		manifestDocs = append(manifestDocs, pf.item)
	}

	manifestPath, err := writeManifest(opts, manifestDocs)
	if err != nil {
		return nil, err
	}

	relManifestPath, err := filepath.Rel(opts.OutDir, manifestPath)
	if err != nil {
		relManifestPath = manifestPath
	}

	return &ExportSummary{
		Provider: sanitizeSegment(opts.Name),
		Version:  opts.Version,
		OutDir:   opts.OutDir,
		Written:  len(planned),
		Manifest: filepath.ToSlash(filepath.Join(opts.OutDir, relManifestPath)),
	}, nil
}

func PreflightExportOptions(opts *ExportOptions) error {
	_, err := prepareExportOptions(opts)
	return err
}

func validateExportOptions(opts *ExportOptions) error {
	opts.Namespace = strings.ToLower(strings.TrimSpace(opts.Namespace))
	opts.Name = strings.ToLower(strings.TrimSpace(opts.Name))
	opts.Version = strings.TrimSpace(opts.Version)
	opts.Format = strings.ToLower(strings.TrimSpace(opts.Format))
	opts.OutDir = strings.TrimSpace(opts.OutDir)
	opts.PathTemplate = strings.TrimSpace(opts.PathTemplate)

	if opts.Namespace == "" {
		opts.Namespace = "hashicorp"
	}
	if opts.Name == "" {
		return &ValidationError{Message: "--name is required"}
	}
	if opts.Version == "" {
		return &ValidationError{Message: "--version is required"}
	}
	if opts.Format == "" {
		opts.Format = "markdown"
	}
	if opts.OutDir == "" {
		return &ValidationError{Message: "--out-dir is required"}
	}
	if opts.PathTemplate == "" {
		opts.PathTemplate = DefaultPathTemplate
	}

	outAbs, err := filepath.Abs(opts.OutDir)
	if err != nil {
		return &ValidationError{Message: fmt.Sprintf("invalid --out-dir: %v", err)}
	}
	opts.OutDir = outAbs

	cats, err := normalizeCategories(opts.Categories)
	if err != nil {
		return err
	}
	opts.Categories = cats

	if _, err := extensionForFormat(opts.Format); err != nil {
		return &ValidationError{Message: err.Error()}
	}
	return nil
}

func normalizeCategories(input []string) ([]string, error) {
	if len(input) == 0 {
		return append([]string{}, defaultCategories...), nil
	}

	allowed := make(map[string]struct{}, len(defaultCategories))
	for _, c := range defaultCategories {
		allowed[c] = struct{}{}
	}

	set := make(map[string]struct{})
	for _, raw := range input {
		for _, token := range strings.Split(raw, ",") {
			cat := strings.ToLower(strings.TrimSpace(token))
			if cat == "" {
				continue
			}
			if cat == "all" {
				return append([]string{}, defaultCategories...), nil
			}
			if _, ok := allowed[cat]; !ok {
				return nil, &ValidationError{Message: fmt.Sprintf("unsupported category: %s", cat)}
			}
			set[cat] = struct{}{}
		}
	}

	if len(set) == 0 {
		return append([]string{}, defaultCategories...), nil
	}

	result := make([]string, 0, len(set))
	for cat := range set {
		result = append(result, cat)
	}
	sort.Strings(result)
	return result, nil
}

func resolveProviderVersionID(ctx context.Context, client APIClient, namespace, provider, version string) (string, error) {
	path := fmt.Sprintf("/v2/providers/%s/%s?include=provider-versions", url.PathEscape(namespace), url.PathEscape(provider))
	var resp providerVersionsResponse
	if err := client.GetJSON(ctx, path, &resp); err != nil {
		return "", err
	}

	for _, included := range resp.Included {
		if included.Type == "provider-versions" && included.Attributes.Version == version {
			return included.ID, nil
		}
	}

	return "", &NotFoundError{Message: fmt.Sprintf("provider version not found: %s/%s@%s", namespace, provider, version)}
}

func listProviderDocs(ctx context.Context, client APIClient, providerVersionID, category string, page int) ([]struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Category string `json:"category"`
		Slug     string `json:"slug"`
		Title    string `json:"title"`
	} `json:"attributes"`
}, error) {
	q := url.Values{}
	q.Set("filter[provider-version]", providerVersionID)
	q.Set("filter[category]", category)
	q.Set("filter[language]", "hcl")
	q.Set("page[number]", fmt.Sprintf("%d", page))

	path := "/v2/provider-docs?" + q.Encode()
	var resp providerDocsListResponse
	if err := client.GetJSON(ctx, path, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func getProviderDocDetail(ctx context.Context, client APIClient, docID string) (providerDocDetailResponse, []byte, error) {
	var detail providerDocDetailResponse
	path := fmt.Sprintf("/v2/provider-docs/%s", url.PathEscape(docID))
	raw, err := client.Get(ctx, path)
	if err != nil {
		return detail, nil, err
	}
	if err := json.Unmarshal(raw, &detail); err != nil {
		// Recover from cached corrupt JSON by using GetJSON, which can bypass cache
		// and refetch when cached payload is undecodable.
		if jsonErr := client.GetJSON(ctx, path, &detail); jsonErr != nil {
			return detail, nil, jsonErr
		}
		// Re-read raw after successful recovery so --format json preserves
		// fields that are not represented in providerDocDetailResponse.
		recoveredRaw, getErr := client.Get(ctx, path)
		if getErr != nil {
			return detail, nil, getErr
		}
		return detail, recoveredRaw, nil
	}
	return detail, raw, nil
}

func renderContent(format string, detail providerDocDetailResponse, raw []byte) ([]byte, error) {
	switch format {
	case "markdown":
		return []byte(detail.Data.Attributes.Content), nil
	case "json":
		var anyDoc any
		if err := json.Unmarshal(raw, &anyDoc); err != nil {
			if len(raw) == 0 {
				return nil, &WriteError{Path: "", Err: errors.New("empty provider doc response")}
			}
			return raw, nil
		}
		formatted, err := json.MarshalIndent(anyDoc, "", "  ")
		if err != nil {
			return nil, &WriteError{Path: "", Err: err}
		}
		return append(formatted, '\n'), nil
	default:
		return nil, &ValidationError{Message: fmt.Sprintf("unsupported format: %s", format)}
	}
}

func writeManifest(opts ExportOptions, docs []manifestItem) (string, error) {
	manifestPath := manifestPathForOptions(opts)
	if err := ensureNoSymlinkTraversal(opts.OutDir, manifestPath); err != nil {
		return "", &ValidationError{Message: fmt.Sprintf("unsafe manifest path %s: %v", manifestPath, err)}
	}
	docsRoot := filepath.Dir(manifestPath)
	if err := os.MkdirAll(docsRoot, 0o755); err != nil {
		return "", &WriteError{Path: docsRoot, Err: err}
	}

	m := manifest{
		Provider:    sanitizeSegment(opts.Name),
		Namespace:   sanitizeSegment(opts.Namespace),
		Version:     opts.Version,
		Format:      opts.Format,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Total:       len(docs),
		Docs:        docs,
	}

	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", &WriteError{Path: filepath.Join(docsRoot, "_manifest.json"), Err: err}
	}

	if err := os.WriteFile(manifestPath, append(b, '\n'), 0o644); err != nil {
		return "", &WriteError{Path: manifestPath, Err: err}
	}
	return manifestPath, nil
}

func deriveCleanTargets(opts ExportOptions, ext string) ([]string, error) {
	templateRoot, err := deriveTemplateRoot(opts, ext)
	if err != nil {
		return nil, err
	}
	manifestRoot := manifestRootForOptions(opts)

	targetSet := map[string]struct{}{
		templateRoot: {},
		manifestRoot: {},
	}
	targets := make([]string, 0, len(targetSet))
	for target := range targetSet {
		if target == opts.OutDir {
			return nil, &ValidationError{Message: "--clean template resolves to --out-dir root, which is too broad"}
		}
		targets = append(targets, target)
	}

	// Remove deeper paths first to avoid broad parent deletes when roots overlap.
	sort.Slice(targets, func(i, j int) bool {
		return len(targets[i]) > len(targets[j])
	})
	return targets, nil
}

func deriveTemplateRoot(opts ExportOptions, ext string) (string, error) {
	outAbs, err := filepath.Abs(opts.OutDir)
	if err != nil {
		return "", &ValidationError{Message: fmt.Sprintf("invalid --out-dir: %v", err)}
	}

	known := map[string]string{
		"out":       outAbs,
		"namespace": sanitizeSegment(opts.Namespace),
		"provider":  sanitizeSegment(opts.Name),
		"version":   sanitizeSegment(opts.Version),
		"ext":       ext,
	}

	prefix, hasUnknown := substituteUntilUnknownPlaceholder(opts.PathTemplate, known)
	if !hasUnknown {
		prefix = filepath.Dir(prefix)
	} else if !strings.HasSuffix(prefix, "/") && !strings.HasSuffix(prefix, string(os.PathSeparator)) {
		prefix = filepath.Dir(prefix)
	}
	if strings.TrimSpace(prefix) == "" || prefix == "." {
		prefix = outAbs
	}

	rootAbs, err := resolvePathWithinBase(prefix, outAbs)
	if err != nil {
		return "", &ValidationError{Message: fmt.Sprintf("failed to derive clean root from template: %v", err)}
	}

	if !isPathWithinDir(outAbs, rootAbs) {
		return "", &ValidationError{Message: "derived clean root is outside --out-dir"}
	}
	return rootAbs, nil
}

func substituteUntilUnknownPlaceholder(template string, known map[string]string) (string, bool) {
	var b strings.Builder
	cursor := 0

	for _, loc := range rePlaceholder.FindAllStringIndex(template, -1) {
		b.WriteString(template[cursor:loc[0]])
		token := template[loc[0]:loc[1]]
		key := token[1 : len(token)-1]
		replacement, ok := known[key]
		if !ok {
			return b.String(), true
		}
		b.WriteString(replacement)
		cursor = loc[1]
	}
	b.WriteString(template[cursor:])
	return b.String(), false
}

func validatePathTemplate(opts ExportOptions, ext string) error {
	vars := map[string]string{
		"out":       opts.OutDir,
		"namespace": sanitizeSegment(opts.Namespace),
		"provider":  sanitizeSegment(opts.Name),
		"version":   sanitizeSegment(opts.Version),
		"category":  "validation",
		"slug":      "validation",
		"doc_id":    "validation",
		"ext":       ext,
	}
	filePath, err := BuildOutputPath(opts.PathTemplate, vars, opts.OutDir)
	if err != nil {
		return &ValidationError{Message: err.Error()}
	}
	if filePath == manifestPathForOptions(opts) {
		return &ValidationError{Message: fmt.Sprintf("path collision detected in --path-template: %s conflicts with reserved manifest path", filePath)}
	}
	return nil
}

func prepareExportOptions(opts *ExportOptions) (string, error) {
	if err := validateExportOptions(opts); err != nil {
		return "", err
	}

	ext, err := extensionForFormat(opts.Format)
	if err != nil {
		return "", &ValidationError{Message: err.Error()}
	}
	if err := validatePathTemplate(*opts, ext); err != nil {
		return "", err
	}
	return ext, nil
}

func manifestRootForOptions(opts ExportOptions) string {
	return filepath.Join(opts.OutDir, "terraform", sanitizeSegment(opts.Namespace), sanitizeSegment(opts.Name), sanitizeSegment(opts.Version), "docs")
}

func manifestPathForOptions(opts ExportOptions) string {
	return filepath.Join(manifestRootForOptions(opts), "_manifest.json")
}
