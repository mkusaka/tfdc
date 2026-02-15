---
name: tfdc-provider-search
description: Search Terraform provider documentation by service slug using tfdc. Use when you need to find provider_doc_id values for a specific resource, data source, or other provider doc type. Returns candidate doc IDs that can be passed to tfdc provider get.
---

# tfdc provider search

Search candidate provider docs and return `provider_doc_id` list.

## Usage

```bash
tfdc provider search \
  -name aws \
  -service ec2 \
  -type resources \
  [-namespace hashicorp] \
  [-version latest] \
  [-limit 20] \
  [-format text]
```

## Flags

| Flag | Required | Default | Description |
|---|---|---|---|
| `-name` | Yes | | Provider name (e.g., `aws`, `google`, `azurerm`) |
| `-service` | Yes | | Slug-like search token to match against doc slugs |
| `-type` | Yes | | Doc category (see below) |
| `-namespace` | No | `hashicorp` | Provider namespace |
| `-version` | No | `latest` | Provider version (semver or `latest`) |
| `-limit` | No | `20` | Max results |
| `-format` | No | `text` | Output format: `text`, `json`, `markdown` |

### `-type` values

`resources`, `data-sources`, `ephemeral-resources`, `functions`, `guides`, `overview`, `actions`, `list-resources`

## Output fields

| Field | Description |
|---|---|
| `provider_doc_id` | Numeric ID for use with `provider get` |
| `title` | Document title |
| `category` | Document category |
| `description` | Document slug/description |
| `provider` | Provider name |
| `namespace` | Provider namespace |
| `version` | Resolved provider version |

## Examples

```bash
# Search AWS EC2 resources
tfdc provider search -name aws -service ec2 -type resources

# Search data sources with JSON output
tfdc provider search -name aws -service ami -type data-sources -format json

# Search specific version
tfdc provider search -name google -service compute -type resources -version 5.0.0

# Search guides
tfdc provider search -name aws -service ec2 -type guides
```

## JSON output

```json
{
  "items": [
    {
      "provider_doc_id": "10595066",
      "title": "aws_instance",
      "category": "resources",
      "description": "aws_instance",
      "provider": "aws",
      "namespace": "hashicorp",
      "version": "6.32.1"
    }
  ],
  "total": 1
}
```

## Workflow

Use with `tfdc provider get` to fetch full doc content:

```bash
# Find doc IDs
tfdc provider search -name aws -service ec2 -type resources -format json | jq '.items[].provider_doc_id'

# Fetch content by ID
tfdc provider get -doc-id 10595066
```
