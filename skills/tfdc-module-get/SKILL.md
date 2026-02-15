---
name: tfdc-module-get
description: Fetch Terraform module details by module ID using tfdc. Use when you have a module ID (namespace/name/provider/version) from a previous search and need the full module documentation including readme content.
---

# tfdc module get

Fetch module details by exact module ID.

## Usage

```bash
tfdc module get -id <module_id> [-format text]
```

## Flags

| Flag | Required | Default | Description |
|---|---|---|---|
| `-id` | Yes | | Module ID in `namespace/name/provider/version` format |
| `-format` | No | `text` | Output format: `text`, `json`, `markdown` |

## ID format

The module ID must have exactly 4 segments: `namespace/name/provider/version`

Example: `terraform-aws-modules/vpc/aws/6.0.1`

## Examples

```bash
# Fetch module readme as plain text
tfdc module get -id terraform-aws-modules/vpc/aws/6.0.1

# Fetch as structured JSON
tfdc module get -id terraform-aws-modules/vpc/aws/6.0.1 -format json
```

## JSON output

```json
{
  "id": "terraform-aws-modules/vpc/aws/6.0.1",
  "content": "# AWS VPC Terraform module\n\n...",
  "content_type": "text/markdown"
}
```

## Workflow

Typically used after `tfdc module search`:

```bash
# Search for modules
tfdc module search -query vpc -format json

# Get full details
tfdc module get -id terraform-aws-modules/vpc/aws/6.0.1
```
