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
	"github.com/mkusaka/tfdc/internal/lockfile"
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

func Execute(args []string, stderr io.Writer) int {
	g, rest, err := parseGlobalFlags(args)
	if err != nil {
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
		switch cmd {
		case "export":
			summaries, runErr := runProviderExport(ctx, g, subArgs, stderr)
			if runErr != nil {
				code := mapErrorToExitCode(runErr)
				_, _ = fmt.Fprintln(stderr, runErr)
				return code
			}
			printSummaries(summaries, stderr)
			return 0
		default:
			_, _ = fmt.Fprintf(stderr, "unsupported provider command: %s\n", cmd)
			return 1
		}
	default:
		_, _ = fmt.Fprintf(stderr, "unsupported command group: %s\n", group)
		printUsage(stderr)
		return 1
	}
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

func runProviderExport(ctx context.Context, g globalFlags, args []string, stderr io.Writer) ([]provider.ExportSummary, error) {
	var namespace string
	var name string
	var version string
	var format string
	var outDir string
	var categories string
	var pathTemplate string
	var clean bool

	fs := flag.NewFlagSet("provider export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&namespace, "namespace", "hashicorp", "provider namespace")
	fs.StringVar(&name, "name", "", "provider name")
	fs.StringVar(&version, "version", "", "provider version")
	fs.StringVar(&format, "format", "markdown", "persist format: markdown|json")
	fs.StringVar(&outDir, "out-dir", "", "output directory")
	fs.StringVar(&categories, "categories", "all", "categories list or all")
	fs.StringVar(&pathTemplate, "path-template", provider.DefaultPathTemplate, "output path template")
	fs.BoolVar(&clean, "clean", false, "remove existing provider/version subtree before export")

	if err := fs.Parse(args); err != nil {
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
	_, _ = fmt.Fprintln(w, "usage: tfdc [global flags] provider export [flags]")
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
