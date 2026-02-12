package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mkusaka/terraform-docs-cli/internal/cache"
	"github.com/mkusaka/terraform-docs-cli/internal/provider"
	"github.com/mkusaka/terraform-docs-cli/internal/registry"
)

type globalFlags struct {
	output      string
	write       string
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

func Execute(args []string, stdout io.Writer, stderr io.Writer) int {
	g, rest, err := parseGlobalFlags(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
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
			summary, runErr := runProviderExport(ctx, g, subArgs)
			if runErr != nil {
				code := mapErrorToExitCode(runErr)
				fmt.Fprintln(stderr, runErr)
				return code
			}
			if err := writeSummary(g, summary, stdout); err != nil {
				fmt.Fprintln(stderr, err)
				return 4
			}
			return 0
		default:
			fmt.Fprintf(stderr, "unsupported provider command: %s\n", cmd)
			return 1
		}
	default:
		fmt.Fprintf(stderr, "unsupported command group: %s\n", group)
		printUsage(stderr)
		return 1
	}
}

func parseGlobalFlags(args []string) (globalFlags, []string, error) {
	g := globalFlags{}
	fs := flag.NewFlagSet("terraform-docs-cli", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&g.output, "output", "text", "output format: text|json|markdown")
	fs.StringVar(&g.output, "o", "text", "output format: text|json|markdown")
	fs.StringVar(&g.write, "write", "", "write output to file path")
	fs.DurationVar(&g.timeout, "timeout", 10*time.Second, "HTTP timeout")
	fs.IntVar(&g.retry, "retry", 3, "retry count")
	fs.StringVar(&g.registryURL, "registry-url", "https://registry.terraform.io", "registry base URL")
	fs.BoolVar(&g.insecure, "insecure", false, "skip TLS verification")
	fs.StringVar(&g.userAgent, "user-agent", "terraform-docs-cli/dev", "custom User-Agent")
	fs.BoolVar(&g.debug, "debug", false, "enable debug log")
	fs.StringVar(&g.cacheDir, "cache-dir", "~/.cache/terraform-docs-cli", "cache directory")
	fs.DurationVar(&g.cacheTTL, "cache-ttl", 24*time.Hour, "cache TTL")
	fs.BoolVar(&g.noCache, "no-cache", false, "disable cache")

	if err := fs.Parse(args); err != nil {
		return g, nil, err
	}

	g.output = strings.ToLower(strings.TrimSpace(g.output))
	if g.output != "text" && g.output != "json" && g.output != "markdown" {
		return g, nil, fmt.Errorf("unsupported --output: %s", g.output)
	}

	if g.retry < 0 {
		return g, nil, fmt.Errorf("--retry must be >= 0")
	}

	if !g.noCache {
		if g.cacheTTL <= 0 {
			return g, nil, fmt.Errorf("--cache-ttl must be positive")
		}
		expanded, err := expandHomeDir(g.cacheDir)
		if err != nil {
			return g, nil, err
		}
		g.cacheDir = expanded
	}

	return g, fs.Args(), nil
}

func runProviderExport(ctx context.Context, g globalFlags, args []string) (*provider.ExportSummary, error) {
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

	if strings.TrimSpace(outDir) == "" {
		return nil, &provider.ValidationError{Message: "--out-dir is required"}
	}

	outDirAbs, err := filepath.Abs(outDir)
	if err != nil {
		return nil, &provider.ValidationError{Message: fmt.Sprintf("invalid --out-dir: %v", err)}
	}

	cacheStore, err := cache.NewStore(g.cacheDir, g.cacheTTL, !g.noCache)
	if err != nil {
		return nil, err
	}

	client, err := registry.NewClient(registry.Config{
		BaseURL:   g.registryURL,
		Timeout:   g.timeout,
		Retry:     g.retry,
		Insecure:  g.insecure,
		UserAgent: g.userAgent,
		Debug:     g.debug,
	}, cacheStore)
	if err != nil {
		return nil, err
	}

	opts := provider.ExportOptions{
		Namespace:    namespace,
		Name:         name,
		Version:      version,
		Format:       strings.ToLower(format),
		OutDir:       outDirAbs,
		Categories:   []string{categories},
		PathTemplate: pathTemplate,
		Clean:        clean,
	}

	return provider.ExportDocs(ctx, client, opts)
}

func writeSummary(g globalFlags, summary *provider.ExportSummary, stdout io.Writer) error {
	var b []byte
	var err error

	switch g.output {
	case "json":
		b, err = json.MarshalIndent(summary, "", "  ")
		if err == nil {
			b = append(b, '\n')
		}
	case "markdown":
		b = []byte(fmt.Sprintf("- provider: `%s`\n- version: `%s`\n- written: `%d`\n- manifest: `%s`\n", summary.Provider, summary.Version, summary.Written, summary.Manifest))
	default:
		b = []byte(fmt.Sprintf("exported %d docs for %s@%s\nmanifest: %s\n", summary.Written, summary.Provider, summary.Version, summary.Manifest))
	}
	if err != nil {
		return err
	}

	if strings.TrimSpace(g.write) != "" {
		if err := os.MkdirAll(filepath.Dir(g.write), 0o755); err != nil {
			return err
		}
		return os.WriteFile(g.write, b, 0o644)
	}
	_, err = stdout.Write(b)
	return err
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

	return 3
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: terraform-docs-cli [global flags] provider export [flags]")
}

func expandHomeDir(path string) (string, error) {
	if path == "" || path[0] != '~' {
		return path, nil
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
