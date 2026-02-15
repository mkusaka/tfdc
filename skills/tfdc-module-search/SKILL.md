---
name: tfdc-module-search
description: Search the Terraform module registry using tfdc. Use when you need to find Terraform modules by keyword, discover available modules for a specific purpose (e.g., VPC, EKS, S3), or get module IDs for use with tfdc module get.
---

# tfdc module search

Search the Terraform module registry.

## Usage

```bash
tfdc module search -query <keyword> [-offset 0] [-limit 20] [-format text]
```

## Flags

| Flag | Required | Default | Description |
|---|---|---|---|
| `-query` | Yes | | Search query (e.g., `vpc`, `eks`, `s3`) |
| `-offset` | No | `0` | Result offset for pagination |
| `-limit` | No | `20` | Max results |
| `-format` | No | `text` | Output format: `text`, `json`, `markdown` |

## Output fields

| Field | Description |
|---|---|
| `module_id` | Module ID in `namespace/name/provider/version` format |
| `name` | Module name |
| `description` | Module description |
| `downloads` | Download count |
| `verified` | Whether the module is verified |
| `published_at` | Publication timestamp |

## Examples

```bash
# Search for VPC modules
tfdc module search -query vpc

# Search with JSON output
tfdc module search -query eks -format json

# Paginated search
tfdc module search -query s3 -offset 20 -limit 10
```

## JSON output

```json
{
  "items": [
    {
      "module_id": "terraform-aws-modules/vpc/aws/6.6.0",
      "name": "vpc",
      "description": "Terraform module to create AWS VPC resources",
      "downloads": 168103660,
      "verified": false,
      "published_at": "2026-01-08T19:16:31.278629Z"
    }
  ],
  "total": 1
}
```

## Workflow

Use with `tfdc module get` to fetch full module details:

```bash
# Search
tfdc module search -query vpc -format json | jq '.items[].module_id'

# Get details
tfdc module get -id terraform-aws-modules/vpc/aws/6.6.0
```
