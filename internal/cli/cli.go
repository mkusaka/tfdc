package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mkusaka/tfdc/internal/cache"
	"github.com/mkusaka/tfdc/internal/guide"
	"github.com/mkusaka/tfdc/internal/lockfile"
	"github.com/mkusaka/tfdc/internal/module"
	"github.com/mkusaka/tfdc/internal/output"
	"github.com/mkusaka/tfdc/internal/policy"
	"github.com/mkusaka/tfdc/internal/progress"
	"github.com/mkusaka/tfdc/internal/provider"
	"github.com/mkusaka/tfdc/internal/registry"
)

type globalFlags struct {
	chdir       string
	timeout     time.Duration
	retry       int
	registryURL string
	insecure    bool
	userAgent   string
	debug       bool
	cacheDir    string
	cacheTTL    time.Duration
	noCache     bool
}

type CacheInitError struct {
	Path string
	Err  error
}

func (e *CacheInitError) Error() string {
	return fmt.Sprintf("failed to initialize cache at %s: %v", e.Path, e.Err)
}

func (e *CacheInitError) Unwrap() error { return e.Err }

func Execute(args []string, stdout, stderr io.Writer) int {
	g, rest, err := parseGlobalFlags(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsage(stdout)
			return 0
		}
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	if len(rest) < 2 {
		printUsage(stderr)
		return 1
	}

	ctx := context.Background()
	group, cmd := rest[0], rest[1]
	subArgs := rest[2:]

	switch group {
	case "provider":
		return runProvider(ctx, g, cmd, subArgs, stdout, stderr)
	case "module":
		return runModule(ctx, g, cmd, subArgs, stdout, stderr)
	case "policy":
		return runPolicy(ctx, g, cmd, subArgs, stdout, stderr)
	case "guide":
		return runGuide(ctx, g, cmd, subArgs, stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unsupported command group: %s\n", group)
		printUsage(stderr)
		return 1
	}
}

// handleSubcmdResult maps the error returned by a subcommand to an exit code.
// flag.ErrHelp means help was already printed to stdout; exit 0.
func handleSubcmdResult(err error, stderr io.Writer) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	code := mapErrorToExitCode(err)
	_, _ = fmt.Fprintln(stderr, err)
	return code
}

func runProvider(ctx context.Context, g globalFlags, cmd string, subArgs []string, stdout, stderr io.Writer) int {
	switch cmd {
	case "--help", "-h":
		_, _ = fmt.Fprintln(stdout, "usage: tfdc [global flags] provider <command> [flags]\n\ncommands:\n  search   search provider documentation\n  get      fetch a provider doc by ID\n  export   export provider docs to files")
		return 0
	case "export":
		summaries, runErr := runProviderExport(ctx, g, subArgs, stdout, stderr)
		if runErr != nil {
			if errors.Is(runErr, flag.ErrHelp) {
				return 0
			}
			code := mapErrorToExitCode(runErr)
			_, _ = fmt.Fprintln(stderr, runErr)
			return code
		}
		printSummaries(summaries, stderr)
		return 0
	case "search":
		return handleSubcmdResult(runProviderSearch(ctx, g, subArgs, stdout, stderr), stderr)
	case "get":
		return handleSubcmdResult(runProviderGet(ctx, g, subArgs, stdout, stderr), stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unsupported provider command: %s\n", cmd)
		return 1
	}
}

func runProviderSearch(ctx context.Context, g globalFlags, args []string, stdout, _ io.Writer) error {
	var name, namespace, service, typ, version, format string
	var limit int

	fs := flag.NewFlagSet("provider search", flag.ContinueOnError)
	fs.SetOutput(stdout)
	fs.StringVar(&name, "name", "", "provider name")
	fs.StringVar(&namespace, "namespace", "hashicorp", "provider namespace")
	fs.StringVar(&service, "service", "", "slug-like search token")
	fs.StringVar(&typ, "type", "", "doc type: resources|data-sources|...")
	fs.StringVar(&version, "version", "latest", "provider version or latest")
	fs.IntVar(&limit, "limit", 20, "max results")
	fs.StringVar(&format, "format", "text", "output format: text|json|markdown")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return &provider.ValidationError{Message: err.Error()}
	}
	if extra := fs.Args(); len(extra) > 0 {
		return &provider.ValidationError{Message: fmt.Sprintf("unexpected positional arguments: %s", strings.Join(extra, ", "))}
	}

	client, err := buildRegistryClient(g)
	if err != nil {
		return err
	}

	results, err := provider.SearchDocs(ctx, client, provider.SearchOptions{
		Name:      name,
		Namespace: namespace,
		Service:   service,
		Type:      typ,
		Version:   version,
		Limit:     limit,
	})
	if err != nil {
		return err
	}

	items := make([]map[string]any, len(results))
	for i, r := range results {
		items[i] = map[string]any{
			"provider_doc_id": r.ProviderDocID,
			"title":           r.Title,
			"category":        r.Category,
			"description":     r.Slug,
			"provider":        r.Provider,
			"namespace":       r.Namespace,
			"version":         r.Version,
		}
	}
	columns := []string{"provider_doc_id", "title", "category", "description", "provider", "namespace", "version"}
	return output.WriteSearch(stdout, format, items, len(items), columns)
}

func runProviderGet(ctx context.Context, g globalFlags, args []string, stdout, _ io.Writer) error {
	var docID, format string

	fs := flag.NewFlagSet("provider get", flag.ContinueOnError)
	fs.SetOutput(stdout)
	fs.StringVar(&docID, "doc-id", "", "numeric provider doc ID")
	fs.StringVar(&format, "format", "text", "output format: text|json|markdown")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return &provider.ValidationError{Message: err.Error()}
	}
	if extra := fs.Args(); len(extra) > 0 {
		return &provider.ValidationError{Message: fmt.Sprintf("unexpected positional arguments: %s", strings.Join(extra, ", "))}
	}

	client, err := buildRegistryClient(g)
	if err != nil {
		return err
	}

	result, err := provider.GetDoc(ctx, client, docID)
	if err != nil {
		return err
	}

	return output.WriteDetail(stdout, format, result.ID, result.Content, result.ContentType)
}

func runModule(ctx context.Context, g globalFlags, cmd string, subArgs []string, stdout, stderr io.Writer) int {
	switch cmd {
	case "--help", "-h":
		_, _ = fmt.Fprintln(stdout, "usage: tfdc [global flags] module <command> [flags]\n\ncommands:\n  search   search modules\n  get      fetch a module by ID")
		return 0
	case "search":
		return handleSubcmdResult(runModuleSearch(ctx, g, subArgs, stdout, stderr), stderr)
	case "get":
		return handleSubcmdResult(runModuleGet(ctx, g, subArgs, stdout, stderr), stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unsupported module command: %s\n", cmd)
		return 1
	}
}

func runModuleSearch(ctx context.Context, g globalFlags, args []string, stdout, _ io.Writer) error {
	var query, format string
	var offset, limit int

	fs := flag.NewFlagSet("module search", flag.ContinueOnError)
	fs.SetOutput(stdout)
	fs.StringVar(&query, "query", "", "search query")
	fs.IntVar(&offset, "offset", 0, "result offset")
	fs.IntVar(&limit, "limit", 20, "max results")
	fs.StringVar(&format, "format", "text", "output format: text|json|markdown")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return &provider.ValidationError{Message: err.Error()}
	}
	if extra := fs.Args(); len(extra) > 0 {
		return &provider.ValidationError{Message: fmt.Sprintf("unexpected positional arguments: %s", strings.Join(extra, ", "))}
	}

	client, err := buildRegistryClient(g)
	if err != nil {
		return err
	}

	results, total, err := module.SearchModules(ctx, client, module.SearchOptions{
		Query:  query,
		Offset: offset,
		Limit:  limit,
	})
	if err != nil {
		return wrapModuleError(err)
	}

	items := make([]map[string]any, len(results))
	for i, r := range results {
		items[i] = map[string]any{
			"module_id":    r.ModuleID,
			"name":         r.Name,
			"description":  r.Description,
			"downloads":    r.Downloads,
			"verified":     r.Verified,
			"published_at": r.PublishedAt,
		}
	}
	columns := []string{"module_id", "name", "description", "downloads", "verified", "published_at"}
	return output.WriteSearch(stdout, format, items, total, columns)
}

func runModuleGet(ctx context.Context, g globalFlags, args []string, stdout, _ io.Writer) error {
	var id, format string

	fs := flag.NewFlagSet("module get", flag.ContinueOnError)
	fs.SetOutput(stdout)
	fs.StringVar(&id, "id", "", "module ID (namespace/name/provider/version)")
	fs.StringVar(&format, "format", "text", "output format: text|json|markdown")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return &provider.ValidationError{Message: err.Error()}
	}
	if extra := fs.Args(); len(extra) > 0 {
		return &provider.ValidationError{Message: fmt.Sprintf("unexpected positional arguments: %s", strings.Join(extra, ", "))}
	}

	client, err := buildRegistryClient(g)
	if err != nil {
		return err
	}

	result, err := module.GetModule(ctx, client, id)
	if err != nil {
		return wrapModuleError(err)
	}

	return output.WriteDetail(stdout, format, result.ID, result.Content, "text/markdown")
}

// wrapModuleError converts module package errors to provider package errors
// so that mapErrorToExitCode works correctly.
func wrapModuleError(err error) error {
	var mvErr *module.ValidationError
	if errors.As(err, &mvErr) {
		return &provider.ValidationError{Message: mvErr.Message}
	}
	return err
}

func runPolicy(ctx context.Context, g globalFlags, cmd string, subArgs []string, stdout, stderr io.Writer) int {
	switch cmd {
	case "--help", "-h":
		_, _ = fmt.Fprintln(stdout, "usage: tfdc [global flags] policy <command> [flags]\n\ncommands:\n  search   search policy libraries\n  get      fetch a policy by ID")
		return 0
	case "search":
		return handleSubcmdResult(runPolicySearch(ctx, g, subArgs, stdout, stderr), stderr)
	case "get":
		return handleSubcmdResult(runPolicyGet(ctx, g, subArgs, stdout, stderr), stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unsupported policy command: %s\n", cmd)
		return 1
	}
}

func runPolicySearch(ctx context.Context, g globalFlags, args []string, stdout, _ io.Writer) error {
	var query, format string

	fs := flag.NewFlagSet("policy search", flag.ContinueOnError)
	fs.SetOutput(stdout)
	fs.StringVar(&query, "query", "", "search query")
	fs.StringVar(&format, "format", "text", "output format: text|json|markdown")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return &provider.ValidationError{Message: err.Error()}
	}
	if extra := fs.Args(); len(extra) > 0 {
		return &provider.ValidationError{Message: fmt.Sprintf("unexpected positional arguments: %s", strings.Join(extra, ", "))}
	}

	client, err := buildRegistryClient(g)
	if err != nil {
		return err
	}

	results, total, err := policy.SearchPolicies(ctx, client, query)
	if err != nil {
		return wrapPolicyError(err)
	}

	items := make([]map[string]any, len(results))
	for i, r := range results {
		items[i] = map[string]any{
			"terraform_policy_id": r.TerraformPolicyID,
			"name":                r.Name,
			"title":               r.Title,
			"downloads":           r.Downloads,
		}
	}
	columns := []string{"terraform_policy_id", "name", "title", "downloads"}
	return output.WriteSearch(stdout, format, items, total, columns)
}

func runPolicyGet(ctx context.Context, g globalFlags, args []string, stdout, _ io.Writer) error {
	var id, format string

	fs := flag.NewFlagSet("policy get", flag.ContinueOnError)
	fs.SetOutput(stdout)
	fs.StringVar(&id, "id", "", "policy ID (policies/namespace/name/version)")
	fs.StringVar(&format, "format", "text", "output format: text|json|markdown")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return &provider.ValidationError{Message: err.Error()}
	}
	if extra := fs.Args(); len(extra) > 0 {
		return &provider.ValidationError{Message: fmt.Sprintf("unexpected positional arguments: %s", strings.Join(extra, ", "))}
	}

	client, err := buildRegistryClient(g)
	if err != nil {
		return err
	}

	result, err := policy.GetPolicy(ctx, client, id)
	if err != nil {
		return wrapPolicyError(err)
	}

	return output.WriteDetail(stdout, format, result.ID, result.Content, "text/markdown")
}

// wrapPolicyError converts policy package errors to provider package errors.
func wrapPolicyError(err error) error {
	var pvErr *policy.ValidationError
	if errors.As(err, &pvErr) {
		return &provider.ValidationError{Message: pvErr.Message}
	}
	return err
}

func runGuide(ctx context.Context, g globalFlags, cmd string, subArgs []string, stdout, stderr io.Writer) int {
	switch cmd {
	case "--help", "-h":
		_, _ = fmt.Fprintln(stdout, "usage: tfdc [global flags] guide <command> [flags]\n\ncommands:\n  style       fetch the Terraform style guide\n  module-dev  fetch the module development guide")
		return 0
	case "style":
		return handleSubcmdResult(runGuideStyle(ctx, g, subArgs, stdout, stderr), stderr)
	case "module-dev":
		return handleSubcmdResult(runGuideModuleDev(ctx, g, subArgs, stdout, stderr), stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unsupported guide command: %s\n", cmd)
		return 1
	}
}

func runGuideStyle(ctx context.Context, g globalFlags, args []string, stdout, _ io.Writer) error {
	var format string

	fs := flag.NewFlagSet("guide style", flag.ContinueOnError)
	fs.SetOutput(stdout)
	fs.StringVar(&format, "format", "text", "output format: text|json|markdown")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return &provider.ValidationError{Message: err.Error()}
	}
	if extra := fs.Args(); len(extra) > 0 {
		return &provider.ValidationError{Message: fmt.Sprintf("unexpected positional arguments: %s", strings.Join(extra, ", "))}
	}

	client, err := buildRegistryClient(g)
	if err != nil {
		return err
	}

	content, err := guide.FetchStyleGuide(ctx, client)
	if err != nil {
		return err
	}

	return output.WriteDetail(stdout, format, "style-guide", content, "text/markdown")
}

func runGuideModuleDev(ctx context.Context, g globalFlags, args []string, stdout, _ io.Writer) error {
	var section, format string

	fs := flag.NewFlagSet("guide module-dev", flag.ContinueOnError)
	fs.SetOutput(stdout)
	fs.StringVar(&section, "section", "all", "section: all|index|composition|structure|providers|publish|refactoring")
	fs.StringVar(&format, "format", "text", "output format: text|json|markdown")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return &provider.ValidationError{Message: err.Error()}
	}
	if extra := fs.Args(); len(extra) > 0 {
		return &provider.ValidationError{Message: fmt.Sprintf("unexpected positional arguments: %s", strings.Join(extra, ", "))}
	}

	client, err := buildRegistryClient(g)
	if err != nil {
		return err
	}

	content, err := guide.FetchModuleDevGuide(ctx, client, section)
	if err != nil {
		return wrapGuideError(err)
	}

	id := "module-dev"
	if section != "all" && section != "" {
		id = "module-dev/" + section
	}
	return output.WriteDetail(stdout, format, id, content, "text/markdown")
}

// wrapGuideError converts guide package errors to provider package errors.
func wrapGuideError(err error) error {
	var gvErr *guide.ValidationError
	if errors.As(err, &gvErr) {
		return &provider.ValidationError{Message: gvErr.Message}
	}
	return err
}

func parseGlobalFlags(args []string) (globalFlags, []string, error) {
	g := globalFlags{}
	fs := flag.NewFlagSet("tfdc", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&g.chdir, "chdir", "", "switch to a different working directory before executing")
	fs.DurationVar(&g.timeout, "timeout", 10*time.Second, "HTTP timeout")
	fs.IntVar(&g.retry, "retry", 3, "retry count")
	fs.StringVar(&g.registryURL, "registry-url", "https://registry.terraform.io", "registry base URL")
	fs.BoolVar(&g.insecure, "insecure", false, "skip TLS verification")
	fs.StringVar(&g.userAgent, "user-agent", "tfdc/dev", "custom User-Agent")
	fs.BoolVar(&g.debug, "debug", false, "enable debug log")
	fs.StringVar(&g.cacheDir, "cache-dir", "~/.cache/tfdc", "cache directory")
	fs.DurationVar(&g.cacheTTL, "cache-ttl", 24*time.Hour, "cache TTL")
	fs.BoolVar(&g.noCache, "no-cache", false, "disable cache")

	if err := fs.Parse(args); err != nil {
		return g, nil, err
	}

	if g.retry < 0 {
		return g, nil, fmt.Errorf("-retry must be >= 0")
	}

	if !g.noCache {
		if g.cacheTTL <= 0 {
			return g, nil, fmt.Errorf("-cache-ttl must be positive")
		}
		expanded, err := expandHomeDir(g.cacheDir)
		if err != nil {
			return g, nil, err
		}
		if strings.TrimSpace(expanded) == "" {
			return g, nil, fmt.Errorf("-cache-dir must not be empty")
		}
		g.cacheDir = expanded
	}

	return g, fs.Args(), nil
}

func runProviderExport(ctx context.Context, g globalFlags, args []string, stdout, stderr io.Writer) ([]provider.ExportSummary, error) {
	var namespace string
	var name string
	var version string
	var format string
	var outDir string
	var categories string
	var pathTemplate string
	var clean bool

	fs := flag.NewFlagSet("provider export", flag.ContinueOnError)
	fs.SetOutput(stdout)
	fs.StringVar(&namespace, "namespace", "hashicorp", "provider namespace")
	fs.StringVar(&name, "name", "", "provider name")
	fs.StringVar(&version, "version", "", "provider version")
	fs.StringVar(&format, "format", "markdown", "persist format: markdown|json")
	fs.StringVar(&outDir, "out-dir", "", "output directory")
	fs.StringVar(&categories, "categories", "all", "categories list or all")
	fs.StringVar(&pathTemplate, "path-template", provider.DefaultPathTemplate, "output path template")
	fs.BoolVar(&clean, "clean", false, "remove existing provider/version subtree before export")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, err
		}
		return nil, &provider.ValidationError{Message: err.Error()}
	}
	if extra := fs.Args(); len(extra) > 0 {
		return nil, &provider.ValidationError{Message: fmt.Sprintf("unexpected positional arguments: %s", strings.Join(extra, ", "))}
	}

	resolvedLockfile := resolveLockfilePath(g.chdir)

	spinner := progress.New(stderr)
	defer spinner.Stop()

	if resolvedLockfile != "" {
		return runLockfileExport(ctx, g, resolvedLockfile, name, version, stderr, spinner, provider.ExportOptions{
			Format:       strings.ToLower(format),
			OutDir:       outDir,
			Categories:   []string{categories},
			PathTemplate: pathTemplate,
			Clean:        clean,
		})
	}

	// Legacy mode: -name and -version required.
	opts := provider.ExportOptions{
		Namespace:    namespace,
		Name:         name,
		Version:      version,
		Format:       strings.ToLower(format),
		OutDir:       outDir,
		Categories:   []string{categories},
		PathTemplate: pathTemplate,
		Clean:        clean,
	}
	if err := provider.PreflightExportOptions(&opts); err != nil {
		return nil, err
	}

	client, err := buildRegistryClient(g)
	if err != nil {
		return nil, err
	}

	spinner.Start(fmt.Sprintf("Exporting %s/%s@%s", namespace, name, version))
	opts.OnProgress = func(msg string) { spinner.Update(msg) }

	summary, err := provider.ExportDocs(ctx, client, opts)
	if err != nil {
		return nil, err
	}
	return []provider.ExportSummary{*summary}, nil
}

func resolveLockfilePath(chdir string) string {
	if strings.TrimSpace(chdir) != "" {
		return filepath.Join(chdir, ".terraform.lock.hcl")
	}
	return ""
}

func runLockfileExport(ctx context.Context, g globalFlags, lockfilePath, nameFilter, versionFlag string, stderr io.Writer, spinner *progress.Spinner, baseOpts provider.ExportOptions) ([]provider.ExportSummary, error) {
	if strings.TrimSpace(versionFlag) != "" {
		_, _ = fmt.Fprintln(stderr, "warning: -version is ignored when using -chdir")
	}

	locks, err := lockfile.ParseFile(lockfilePath)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(nameFilter) != "" {
		filtered := make([]lockfile.ProviderLock, 0, 1)
		for _, lock := range locks {
			if strings.EqualFold(lock.Name, nameFilter) {
				filtered = append(filtered, lock)
			}
		}
		if len(filtered) == 0 {
			return nil, &provider.NotFoundError{Message: fmt.Sprintf("provider %q not found in lockfile %s", nameFilter, lockfilePath)}
		}
		locks = filtered
	}

	if len(locks) == 0 {
		return nil, &provider.NotFoundError{Message: fmt.Sprintf("no providers found in lockfile %s", lockfilePath)}
	}

	// Validate base options before starting exports.
	// Use the first lock for preflight since Name/Version/Namespace
	// will be overridden per provider anyway.
	preflightOpts := baseOpts
	preflightOpts.Namespace = locks[0].Namespace
	preflightOpts.Name = locks[0].Name
	preflightOpts.Version = locks[0].Version
	if err := provider.PreflightExportOptions(&preflightOpts); err != nil {
		return nil, err
	}

	client, err := buildRegistryClient(g)
	if err != nil {
		return nil, err
	}

	spinner.Start(fmt.Sprintf("Exporting %d providers from lockfile", len(locks)))

	summaries := make([]provider.ExportSummary, 0, len(locks))
	for i, lock := range locks {
		opts := baseOpts
		opts.Namespace = lock.Namespace
		opts.Name = lock.Name
		opts.Version = lock.Version
		prefix := fmt.Sprintf("[%d/%d] %s", i+1, len(locks), lock.Name)
		opts.OnProgress = func(msg string) {
			spinner.Update(fmt.Sprintf("%s: %s", prefix, msg))
		}

		summary, exportErr := provider.ExportDocs(ctx, client, opts)
		if exportErr != nil {
			return nil, exportErr
		}
		summaries = append(summaries, *summary)
	}

	return summaries, nil
}

func buildRegistryClient(g globalFlags) (*registry.Client, error) {
	cacheStore, err := cache.NewStore(g.cacheDir, g.cacheTTL, !g.noCache)
	if err != nil {
		return nil, &CacheInitError{Path: g.cacheDir, Err: err}
	}

	return registry.NewClient(registry.Config{
		BaseURL:   g.registryURL,
		Timeout:   g.timeout,
		Retry:     g.retry,
		Insecure:  g.insecure,
		UserAgent: g.userAgent,
		Debug:     g.debug,
	}, cacheStore)
}

func printSummaries(summaries []provider.ExportSummary, w io.Writer) {
	for _, s := range summaries {
		_, _ = fmt.Fprintf(w, "exported %d docs for %s@%s\nmanifest: %s\n", s.Written, s.Provider, s.Version, s.Manifest)
	}
}

func mapErrorToExitCode(err error) int {
	var vErr *provider.ValidationError
	if errors.As(err, &vErr) {
		return 1
	}

	var fErr *output.FormatError
	if errors.As(err, &fErr) {
		return 1
	}

	var nfErr *provider.NotFoundError
	if errors.As(err, &nfErr) {
		return 2
	}

	var apiErr *registry.APIError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode == 404 {
			return 2
		}
		return 3
	}

	var wErr *provider.WriteError
	if errors.As(err, &wErr) {
		return 4
	}

	var cfgErr *registry.ConfigError
	if errors.As(err, &cfgErr) {
		return 1
	}

	var cacheInitErr *CacheInitError
	if errors.As(err, &cacheInitErr) {
		return 4
	}

	return 3
}

func printUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, `usage: tfdc [global flags] <group> <command> [flags]

commands:
  provider  search | get | export
  module    search | get
  policy    search | get
  guide     style | module-dev

global flags:
  -chdir string
        switch to a different working directory before executing
  -timeout duration
        HTTP timeout (default 10s)
  -retry int
        retry count (default 3)
  -registry-url string
        registry base URL (default "https://registry.terraform.io")
  -insecure
        skip TLS verification
  -user-agent string
        custom User-Agent (default "tfdc/dev")
  -debug
        enable debug log
  -cache-dir string
        cache directory (default "~/.cache/tfdc")
  -cache-ttl duration
        cache TTL (default 24h0m0s)
  -no-cache
        disable cache`)
}

func expandHomeDir(path string) (string, error) {
	if path == "" || path[0] != '~' {
		return path, nil
	}
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return "", fmt.Errorf("unsupported home path: %s (use ~ or ~/...)", path)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
}
