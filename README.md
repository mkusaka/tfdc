# terraform-docs-cli

A Go CLI for exporting Terraform Registry documentation to local files with deterministic paths and persistent caching.

Inspired by <https://github.com/hashicorp/terraform-mcp-server>.

## Current Status

Implemented now:

- `provider export`

Planned in spec/interface docs (not implemented yet):

- `provider search|get|doc|latest-version|capabilities`
- `module search|get|latest-version`
- `policy search|get`
- `guide style|module-dev`

Current command tree:

```text
terraform-docs-cli [global flags] provider export [flags]
```

## Install

Install with Go:

```bash
go install github.com/mkusaka/terraform-docs-cli/cmd/terraform-docs-cli@latest
terraform-docs-cli --help
```

Build from source (for branch-specific changes):

```bash
git clone https://github.com/mkusaka/terraform-docs-cli.git
cd terraform-docs-cli
go build -o bin/terraform-docs-cli ./cmd/terraform-docs-cli
./bin/terraform-docs-cli --help
```

## Quick Start

Run without installation:

```bash
go run ./cmd/terraform-docs-cli \
  --output json \
  provider export \
  --namespace hashicorp \
  --name aws \
  --version 6.31.0 \
  --format markdown \
  --out-dir ./dir \
  --categories guides,resources
```

Build binary:

```bash
go build -o bin/terraform-docs-cli ./cmd/terraform-docs-cli
./bin/terraform-docs-cli provider export --name aws --version 6.31.0 --out-dir ./dir
```

## `provider export`

Fetches all docs for a specific provider version (filtered by categories), writes files to disk, and emits an export summary.

Required flags:

- `--name`
- `--version`
- `--out-dir`

Optional flags:

- `--namespace` (default: `hashicorp`)
- `--format` (`markdown|json`, default: `markdown`)
- `--categories` (default: `all`)
- `--path-template` (default below)
- `--clean` (remove previous export outputs for the same target before writing)

Default template:

```text
{out}/terraform/{namespace}/{provider}/{version}/docs/{category}/{slug}.{ext}
```

Default output example:

```text
dir/terraform/hashicorp/aws/6.31.0/docs/guides/tag-policy-compliance.md
```

Manifest path:

```text
dir/terraform/hashicorp/aws/6.31.0/docs/_manifest.json
```

`--categories all` expands to:

- `resources`
- `data-sources`
- `ephemeral-resources`
- `functions`
- `guides`
- `overview`
- `actions`
- `list-resources`

## Path Template Placeholders

Available placeholders:

- `{out}`
- `{namespace}`
- `{provider}`
- `{version}`
- `{category}`
- `{slug}`
- `{doc_id}`
- `{ext}`

Rules:

- Unknown placeholders are rejected.
- Malformed placeholder syntax (`{` / `}` mismatch) is rejected.
- Resolved paths must remain inside `--out-dir`.
- Path collisions are rejected (including collision with reserved manifest path).
- Safety checks reject symlink traversal outside `--out-dir` for both write and `--clean` deletion paths.
- `--clean` removes the existing manifest file for that provider version before rewriting.
- `--clean` removes a template root directory only when the derived root is scoped by namespace/provider/version path segments.

## Global Flags

- `--output, -o` (`text|json|markdown`, default: `text`)
- `--write` (write summary output to file instead of stdout)
- `--timeout` (default: `10s`)
- `--retry` (default: `3`)
- `--registry-url` (default: `https://registry.terraform.io`)
- `--insecure` (skip TLS verification)
- `--user-agent` (default: `terraform-docs-cli/dev`)
- `--debug`
- `--cache-dir` (default: `~/.cache/terraform-docs-cli`)
- `--cache-ttl` (default: `24h`)
- `--no-cache` (disable cache read/write)

## Persistent Cache

Cache is enabled by default and stores HTTP GET responses on disk.

Default structure:

```text
~/.cache/terraform-docs-cli/
  v1/
    meta.json
    entries/
      ab/
        <sha256>.json
    tmp/
      <sha256>.tmp
```

Notes:

- Cache key: `METHOD + URL` hash.
- TTL expiry is treated as cache miss.
- Corrupted entries are discarded and refetched.
- `--no-cache` disables both cache read and write.

## Exit Codes

- `0`: success
- `1`: invalid arguments / validation / config error
- `2`: not found
- `3`: remote API error
- `4`: local write/serialization/cache-init error

## Development

Run tests:

```bash
go test ./...
```

Run lint:

```bash
golangci-lint run ./...
```

CI (`.github/workflows/ci.yml`) runs:

- `go build -v ./cmd/terraform-docs-cli`
- `go test -v -race ./...`
- `golangci-lint` (binary install mode)

## Design and Full Spec

- Interface design: [`docs/cli-interface.md`](docs/cli-interface.md)
- Specification (EN): [`docs/spec.md`](docs/spec.md)
- 仕様書 (JA): [`docs/spec.ja.md`](docs/spec.ja.md)
