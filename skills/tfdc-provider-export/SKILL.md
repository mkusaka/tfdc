---
name: tfdc-provider-export
description: Bulk export all Terraform provider documentation for a specific version to local files using tfdc. Use when you need to download and persist all docs (resources, data sources, guides, etc.) for a provider to a local directory. Supports lockfile-based multi-provider export.
---

# tfdc provider export

Export all docs for a provider version to local files.

## Usage

### Single provider (legacy mode)

```bash
tfdc provider export \
  -name aws \
  -version 6.31.0 \
  -out-dir ./docs
```

Required: `-name`, `-version`, `-out-dir`

### Lockfile mode (multi-provider)

```bash
tfdc -chdir=./infra provider export -out-dir ./docs
```

Required: `-chdir` (global flag), `-out-dir`

Detects `.terraform.lock.hcl` and exports all listed providers. Filter with `-name`:

```bash
tfdc -chdir=./infra provider export -name aws -out-dir ./docs
```

## Flags

| Flag | Required | Default | Description |
|---|---|---|---|
| `-name` | Yes (legacy) | | Provider name |
| `-version` | Yes (legacy) | | Provider version (explicit semver) |
| `-out-dir` | Yes | | Output directory |
| `-namespace` | No | `hashicorp` | Provider namespace |
| `-format` | No | `markdown` | Persist format: `markdown` or `json` |
| `-categories` | No | `all` | Categories to export (comma-separated) |
| `-path-template` | No | See below | Output path template |
| `-clean` | No | off | Remove previous export before writing |

## Output layout

Default path template: `{out}/terraform/{namespace}/{provider}/{version}/docs/{category}/{slug}.{ext}`

Example: `docs/terraform/hashicorp/aws/6.31.0/docs/resources/instance.md`

Manifest: `docs/terraform/hashicorp/aws/6.31.0/docs/_manifest.json`

### Path template placeholders

`{out}`, `{namespace}`, `{provider}`, `{version}`, `{category}`, `{slug}`, `{doc_id}`, `{ext}`

## Categories

`-categories all` expands to: `resources`, `data-sources`, `ephemeral-resources`, `functions`, `guides`, `overview`, `actions`, `list-resources`

```bash
# Export only resources and data sources
tfdc provider export -name aws -version 6.31.0 -out-dir ./docs -categories resources,data-sources
```

## Examples

```bash
# Export AWS provider docs as markdown
tfdc provider export -name aws -version 6.31.0 -out-dir ./terraform-docs

# Export as JSON
tfdc provider export -name aws -version 6.31.0 -out-dir ./terraform-docs -format json

# Clean export (remove previous)
tfdc provider export -name aws -version 6.31.0 -out-dir ./terraform-docs -clean

# Export all providers from lockfile
tfdc -chdir=./my-project provider export -out-dir ./terraform-docs
```
