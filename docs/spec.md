# terraform-docs-cli Specification

## 1. Purpose

`terraform-docs-cli` is a command-line tool that retrieves Terraform documentation from Terraform Registry and official Terraform documentation sources for implementation and review workflows.

This specification defines:

- Functional requirements
- Command and flag input/output contracts
- Exit code and error behavior
- Internal implementation guidelines
- Testing criteria

## 2. Scope

### 2.1 In Scope

- Retrieval from the public Terraform Registry
- Provider documentation search and detail retrieval
- Module search and detail retrieval
- Policy search and detail retrieval
- Retrieval of Terraform style guide and module development guide
- `text|json|markdown` output modes
- Persistent cache (local disk)
- Bulk export and directory persistence for all docs of a specific provider/version

### 2.2 Out of Scope

- Authenticated private registry operations for TFE/TFC
- Terraform execution operations (plan/apply/run management)

## 3. Terms

- `provider_doc_id`: Numeric ID used for provider-doc detail retrieval
- `module_id`: ID in `namespace/name/provider/version` format
- `terraform_policy_id`: ID in `policies/<namespace>/<name>/<version>` format

## 4. Design Principles

Align behavior with the registry-related tools in `hashicorp/terraform-mcp-server`.

- Provider: `search -> get`
- Module: `search -> get`
- Policy: `search -> get`
- Guide: fetch raw markdown directly

## 5. CLI Specification

### 5.1 Command Tree

```text
terraform-docs-cli [global flags] <group> <command> [flags]

group:
  provider
  module
  policy
  guide
```

### 5.2 Global Flags

```text
--output, -o     text|json|markdown   (default: text)
--write          Output file path      (default: stdout)
--timeout        HTTP timeout          (default: 10s)
--retry          Retry count           (default: 3)
--registry-url   Registry base URL     (default: https://registry.terraform.io)
--insecure       Skip TLS verification
--user-agent     Override User-Agent
--debug          Enable debug logs
--cache-dir      Cache directory        (default: ~/.cache/terraform-docs-cli)
--cache-ttl      Cache TTL              (default: 24h)
--no-cache       Disable cache
```

## 6. Command Details

### 6.1 Provider

#### 6.1.1 `provider search`

Searches candidate provider docs and returns `provider_doc_id`.

```text
terraform-docs-cli provider search \
  --name aws \
  --namespace hashicorp \
  --service ec2 \
  --type resources \
  [--version latest] \
  [--limit 20]
```

Inputs:

- `--name` required
- `--namespace` default `hashicorp`
- `--service` required
- `--type` required  
  `resources|data-sources|ephemeral-resources|functions|guides|overview|actions|list-resources`
- `--version` default `latest`
- `--limit` default `20`

Output fields:

- `provider_doc_id`
- `title`
- `category`
- `description`
- `provider`
- `namespace`
- `version`

#### 6.1.2 `provider get`

Fetches full content by `provider_doc_id`.

```text
terraform-docs-cli provider get --doc-id 8894603
```

Inputs:

- `--doc-id` required (numeric)

Output:

- text/markdown: raw content only
- json: `id`, `content`, `content_type`

#### 6.1.3 `provider doc`

Convenience command that performs search and get in a single call.

```text
terraform-docs-cli provider doc \
  --name aws --namespace hashicorp --service ec2 --type resources \
  [--version latest] [--select best]
```

Inputs:

- `--select`: `best|first|interactive` (default: `best`)

Selection rules:

- `best`: top match by slug/title similarity score
- `first`: first candidate
- `interactive`: show candidates and select by index

#### 6.1.4 `provider latest-version`

```text
terraform-docs-cli provider latest-version --namespace hashicorp --name aws
```

#### 6.1.5 `provider capabilities`

```text
terraform-docs-cli provider capabilities --namespace hashicorp --name aws [--version latest]
```

#### 6.1.6 `provider export`

Fetches all docs for a specific provider version and persists them under a target directory.

```text
terraform-docs-cli provider export \
  --namespace hashicorp \
  --name aws \
  --version 6.31.0 \
  --format markdown \
  --out-dir ./dir \
  [--categories all] \
  [--path-template "{out}/terraform/{namespace}/{provider}/{version}/docs/{category}/{slug}.{ext}"] \
  [--clean]
```

Inputs:

- `--namespace` default `hashicorp`
- `--name` required
- `--version` required (explicit)
- `--format` `markdown|json` (default: `markdown`)
- `--out-dir` required
- `--categories` default `all` (`resources,data-sources,ephemeral-resources,functions,guides,overview,actions,list-resources`)
- `--path-template` optional; uses default layout if omitted
- `--clean` removes the existing manifest file, and additionally removes a derived template root only when it is namespace/provider/version-scoped

Default persistence path:

- `{out}/terraform/{namespace}/{provider}/{version}/docs/{category}/{slug}.{ext}`
- Example: `dir/terraform/hashicorp/aws/6.31.0/docs/guides/tag-policy-compliance.md`

Notes:

- `{ext}` is `md` for markdown and `json` for json format
- Writes namespace-scoped `_manifest.json` under the manifest root  
  (`{out}/terraform/{namespace}/{provider}/{version}/docs/_manifest.json`)

### 6.2 Module

#### 6.2.1 `module search`

```text
terraform-docs-cli module search --query vpc [--offset 0] [--limit 20]
```

Output fields:

- `module_id`
- `name`
- `description`
- `downloads`
- `verified`
- `published_at`

#### 6.2.2 `module get`

```text
terraform-docs-cli module get --id terraform-aws-modules/vpc/aws/6.0.1
```

Input constraints:

- `module_id` must have 4 segments: `namespace/name/provider/version`

#### 6.2.3 `module latest-version`

```text
terraform-docs-cli module latest-version \
  --publisher terraform-aws-modules \
  --name vpc \
  --provider aws
```

### 6.3 Policy

#### 6.3.1 `policy search`

```text
terraform-docs-cli policy search --query cis
```

Output fields:

- `terraform_policy_id`
- `name`
- `title`
- `downloads`

#### 6.3.2 `policy get`

```text
terraform-docs-cli policy get --id policies/hashicorp/CIS-Policy-Set-for-AWS-Terraform/1.0.1
```

### 6.4 Guide

#### 6.4.1 `guide style`

```text
terraform-docs-cli guide style [--output markdown]
```

#### 6.4.2 `guide module-dev`

```text
terraform-docs-cli guide module-dev [--section all]
```

`--section`:

- `all|index|composition|structure|providers|publish|refactoring`

## 7. Output Specification

### 7.1 text

- Human-friendly output
- Detail commands return raw content

### 7.2 markdown

- Preserve markdown as-is
- Detail/guide output is equivalent to text mode

### 7.3 json

- Stable fields for script use
- Search commands format:

```json
{
  "items": [],
  "total": 0
}
```

- Detail commands format:

```json
{
  "id": "string",
  "content": "string",
  "content_type": "text/markdown"
}
```

- `provider export` returns:

```json
{
  "provider": "aws",
  "version": "6.31.0",
  "out_dir": "./dir",
  "written": 123,
  "manifest": "dir/terraform/hashicorp/aws/6.31.0/docs/_manifest.json"
}
```

## 8. Exit Codes

```text
0 success
1 invalid arguments / validation failure
2 target not found
3 remote API error
4 serialization/write failure
```

## 9. Error Handling

- Error messages are written to stderr
- `--debug` adds HTTP status, URL, and retry details
- Map `404` to exit code `2`
- Map `429/5xx` (after retries exhausted) to exit code `3`

## 10. Internal Implementation Guidelines

### 10.1 Package Layout (proposed)

```text
cmd/terraform-docs-cli/
internal/cli/           # cobra command definitions
internal/client/        # HTTP client, retry, TLS
internal/registry/      # provider/module/policy API calls
internal/formatter/     # text/json/markdown formatting
internal/output/        # stdout/file writer
```

### 10.2 HTTP Client

- Inject timeout/retry/insecure from global flags
- User-Agent should be `terraform-docs-cli/<version>`

### 10.3 Validation

- Provider: numeric doc ID check
- Module: segment count check
- Policy: prefix check (`policies/`)

### 10.4 Persistent Cache

- Target commands:
  - `provider search|get`
  - `provider export`
  - `module search|get`
  - `policy search|get`
  - `guide style|module-dev`
- Cache key is composed from method + normalized URL + effective query + output-independent data
- `--output` is presentation-only and must not affect cache keys
- Expired entries (TTL exceeded) are treated as cache miss and replaced on successful refetch
- With `--no-cache`, both cache read and write are disabled
- On cache corruption, discard only the affected entry and continue by refetching

## 11. Test Specification

### 11.1 Unit Tests

- Argument validation
- Response parsing
- Output formatting (text/json/markdown)
- Exit code mapping

### 11.2 Integration Tests

- Happy path for major commands
- Not found scenarios
- API error scenarios (5xx / timeout)
- Cache hit / miss / TTL expiry
- No cache read/write when `--no-cache` is set
- `provider export` writes the expected directory structure
- `provider export --clean` recreates output subtree correctly

### 11.3 Snapshot Tests

- `provider search/get`
- `provider export`
- `module search/get`
- `policy search/get`
- `guide style/module-dev`

## 12. MVP Release Criteria

MVP is complete when all conditions below are satisfied.

- 6.1.1, 6.1.2, 6.1.6, 6.2.1, 6.2.2, 6.3.1, 6.3.2, 6.4.1, 6.4.2 are implemented
- `text/json/markdown` outputs are stable
- Exit code behavior matches this specification
- Persistent cache behavior matches this specification (TTL, no-cache, corruption recovery)
- `provider export` output layout matches this specification (path-template, manifest)
- README includes usage examples
