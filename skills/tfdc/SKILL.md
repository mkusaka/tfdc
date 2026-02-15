---
name: tfdc
description: Retrieve Terraform documentation from the public registry using the tfdc CLI. Supports provider doc search/get/export, module search/get, policy search/get, and Terraform style/module-dev guides. Use when you need to find, read, or export Terraform provider docs, module docs, policy docs, or development guides.
---

# tfdc

tfdc retrieves Terraform documentation from the public Terraform Registry and official HashiCorp docs.

## Available commands

| Command | Description |
|---|---|
| `provider search` | Search provider docs by service slug, returns `provider_doc_id` list |
| `provider get` | Fetch full provider doc content by `provider_doc_id` |
| `provider export` | Bulk export all docs for a provider version to local files |
| `module search` | Search the Terraform module registry |
| `module get` | Fetch module details by module ID |
| `policy search` | Search Terraform policy sets |
| `policy get` | Fetch policy details by policy ID |
| `guide style` | Fetch Terraform style guide |
| `guide module-dev` | Fetch module development guide |

## Workflow

Provider, module, and policy docs use a two-step search-then-get flow:

```bash
# Step 1: Find doc IDs
tfdc provider search -name aws -service ec2 -type resources -format json

# Step 2: Fetch full content by ID
tfdc provider get -doc-id 10595066
```

## Output formats

All search/get/guide commands support `-format text|json|markdown` (default: `text`).

Search JSON: `{ "items": [...], "total": N }`
Detail JSON: `{ "id": "...", "content": "...", "content_type": "text/markdown" }`

## Global flags

| Flag | Default | Description |
|---|---|---|
| `-chdir` | none | Switch working directory; auto-detects `.terraform.lock.hcl` |
| `-timeout` | `10s` | HTTP timeout |
| `-retry` | `3` | Retry count |
| `-registry-url` | `https://registry.terraform.io` | Registry base URL |
| `-insecure` | off | Skip TLS verification |
| `-user-agent` | `tfdc/dev` | User-Agent header |
| `-debug` | off | Debug logs to stderr |
| `-cache-dir` | `~/.cache/tfdc` | Cache directory |
| `-cache-ttl` | `24h` | Cache TTL |
| `-no-cache` | off | Disable cache |

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | Invalid arguments or validation error |
| `2` | Not found |
| `3` | Remote API error |
| `4` | File write or serialization error |

## Individual skill details

For detailed usage of each command, see the individual skills: `tfdc-provider-search`, `tfdc-provider-get`, `tfdc-provider-export`, `tfdc-module-search`, `tfdc-module-get`, `tfdc-policy-search`, `tfdc-policy-get`, `tfdc-guide-style`, `tfdc-guide-module-dev`.
