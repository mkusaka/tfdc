# tfdc Interface Design

## Goal
Build a standalone CLI that retrieves Terraform documentation from public registry APIs, based on the behavior and data flow used in `hashicorp/terraform-mcp-server`.

This CLI is optimized for two usage styles.

- Human usage from terminal (discover docs, inspect details, save markdown)
- Script usage in CI or shell pipelines (stable JSON output and exit codes)

## Design Inputs from terraform-mcp-server

Observed patterns in `terraform-mcp-server` that this CLI should preserve.

- Provider docs are a two-step flow.
- First resolve candidate doc IDs (`search_providers`), then fetch full content by `provider_doc_id` (`get_provider_details`).
- Module docs are a two-step flow.
- First search by query (`search_modules`), then fetch by exact `module_id` (`get_module_details`).
- Policy docs are a two-step flow.
- First search by query (`search_policies`), then fetch by exact `terraform_policy_id` (`get_policy_details`).
- Style guide and module development guide are static markdown resources fetched from HashiCorp docs repo.
- Provider version resolution defaults to latest when version is omitted or invalid.

## Command Tree

```text
tfdc [global flags] <group> <command> [flags]

Groups:
  provider   Provider documentation and metadata
  module     Module discovery and module docs
  policy     Policy set discovery and policy docs
  guide      Terraform style and module-development guides
```

## Global Flags

```text
-chdir             Switch to a different working directory (auto-detects .terraform.lock.hcl)
-timeout           HTTP timeout         (default: 10s)
-retry             Retry count          (default: 3)
-registry-url      Registry base URL    (default: https://registry.terraform.io)
-insecure          Skip TLS verification
-user-agent        Override User-Agent
-debug             Debug log to stderr
-cache-dir         Cache directory       (default: ~/.cache/tfdc)
-cache-ttl         Cache TTL             (default: 24h)
-no-cache          Disable cache
```

## Provider Commands

### `provider search`

Search candidate provider docs and return `provider_doc_id` list.

```text
tfdc provider search \
  -name aws \
  -namespace hashicorp \
  -service ec2 \
  -type resources \
  [-version latest] \
  [-limit 20]
```

Flags.

```text
-name         required
-namespace    default: hashicorp
-service      required; slug-like search token
-type         resources|data-sources|functions|guides|overview|actions|list-resources
               |ephemeral-resources
-version      semver or latest (default: latest)
-limit        max candidates in output (default: 20)
```

Output fields.

- `provider_doc_id`
- `title`
- `category`
- `description`
- `provider`
- `namespace`
- `version`

### `provider get`

Fetch full provider doc content by exact `provider_doc_id`.

```text
tfdc provider get -doc-id 8894603
```

Flags.

```text
-doc-id    required; numeric
```

### `provider doc`

Convenience command: search and fetch in one call.

```text
tfdc provider doc \
  -name aws -namespace hashicorp -service ec2 -type resources \
  [-version latest] [-select best]
```

Flags.

```text
-select    best|first|interactive  (default: best)
```

Selection policy.

- `best`: rank by slug/title match score, then fetch top candidate
- `first`: fetch first candidate
- `interactive`: print candidates and require explicit selection index

### `provider latest-version`

```text
tfdc provider latest-version -namespace hashicorp -name aws
```

### `provider capabilities`

```text
tfdc provider capabilities -namespace hashicorp -name aws [-version latest]
```

### `provider export`

Persist all docs of a specific provider version to a target directory.

```text
tfdc provider export \
  -namespace hashicorp \
  -name aws \
  -version 6.31.0 \
  -format markdown \
  -out-dir ./dir \
  [-categories all] \
  [-path-template "{out}/terraform/{namespace}/{provider}/{version}/docs/{category}/{slug}.{ext}"] \
  [-clean]
```

Default output layout.

- `{out}/terraform/{namespace}/{provider}/{version}/docs/{category}/{slug}.{ext}`
- Example: `dir/terraform/hashicorp/aws/6.31.0/docs/guides/tag-policy-compliance.md`

Export side effects.

- Write one file per provider doc
- Write namespace-scoped `_manifest.json`
  (`{out}/terraform/{namespace}/{provider}/{version}/docs/_manifest.json`)
- Return export summary (`written`, `manifest`) in JSON mode

## Module Commands

### `module search`

```text
tfdc module search -query vpc [-offset 0] [-limit 20]
```

Output fields.

- `module_id`
- `name`
- `description`
- `downloads`
- `verified`
- `published_at`

### `module get`

```text
tfdc module get -id terraform-aws-modules/vpc/aws/6.0.1
```

Validation.

- `module_id` must be `namespace/name/provider/version` (4 segments)

### `module latest-version`

```text
tfdc module latest-version \
  -publisher terraform-aws-modules \
  -name vpc \
  -provider aws
```

## Policy Commands

### `policy search`

```text
tfdc policy search -query cis
```

Output fields.

- `terraform_policy_id`
- `name`
- `title`
- `downloads`

### `policy get`

```text
tfdc policy get -id policies/hashicorp/CIS-Policy-Set-for-AWS-Terraform/1.0.1
```

## Guide Commands

### `guide style`

Fetch Terraform style guide markdown.

```text
tfdc guide style
```

### `guide module-dev`

Fetch module development guide markdown.

```text
tfdc guide module-dev [-section all]
```

Flags.

```text
-section    all|index|composition|structure|providers|publish|refactoring
```

## Exit Codes

```text
0  success
1  invalid arguments or validation failure
2  not found (no matching docs/resources)
3  remote API error
4  output serialization or file write error
```

## Persistent Cache (MVP)

- Enabled by default for registry and guide retrieval commands
- Cache key is request-based
- TTL is controlled by `-cache-ttl`
- `-no-cache` disables both read/write cache behavior
- Corrupted cache entry is ignored and replaced by fresh response

## Output Contract

Search commands should return machine-readable arrays in JSON mode.

Example.

```json
{
  "items": [
    {
      "provider_doc_id": "8894603",
      "title": "aws_instance",
      "category": "resources",
      "description": "Provides an EC2 instance resource..."
    }
  ],
  "total": 1
}
```

Detail commands should return full markdown/text body in text and markdown modes, and structured wrapper in JSON mode.

```json
{
  "id": "8894603",
  "content": "...",
  "content_type": "text/markdown"
}
```

## Mapping to terraform-mcp-server

| CLI command | MCP tool/resource | Registry endpoint family |
|---|---|---|
| `provider search` | `search_providers` | `v1/providers/...`, `v2/provider-docs...` |
| `provider get` | `get_provider_details` | `v2/provider-docs/{id}` |
| `provider export` | (composed flow) | `v2/providers/{namespace}/{name}?include=provider-versions`, `v2/provider-docs?...`, `v2/provider-docs/{id}` |
| `provider latest-version` | `get_latest_provider_version` | `v1/providers/{namespace}/{name}` |
| `provider capabilities` | `get_provider_capabilities` | `v1/providers/{namespace}/{name}/{version}` |
| `module search` | `search_modules` | `v1/modules/search` |
| `module get` | `get_module_details` | `v1/modules/{module_id}` |
| `module latest-version` | `get_latest_module_version` | `v1/modules/{publisher}/{name}/{provider}` |
| `policy search` | `search_policies` | `v2/policies?...` |
| `policy get` | `get_policy_details` | `v2/policies/...?...` |
| `guide style` | resource `/terraform/style-guide` | raw GitHub docs |
| `guide module-dev` | resource `/terraform/module-development` | raw GitHub docs |

## Implementation Phasing

MVP.

- Global flags and HTTP client wiring
- `provider search|get`
- `provider export`
- `module search|get`
- `policy search|get`
- `guide style|module-dev`
- JSON/text output and exit code contract
- Result caching with local cache directory

Phase 2.

- `provider doc` convenience workflow with ranking and interactive select
- `provider capabilities`
- `provider latest-version` and `module latest-version`
