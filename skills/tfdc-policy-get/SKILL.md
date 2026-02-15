---
name: tfdc-policy-get
description: Fetch Terraform policy set details by policy ID using tfdc. Use when you have a terraform_policy_id from a previous search and need the full policy documentation including readme content.
---

# tfdc policy get

Fetch policy details by exact policy ID.

## Usage

```bash
tfdc policy get -id <policy_id> [-format text]
```

## Flags

| Flag | Required | Default | Description |
|---|---|---|---|
| `-id` | Yes | | Policy ID in `policies/namespace/name/version` format |
| `-format` | No | `text` | Output format: `text`, `json`, `markdown` |

## ID format

The policy ID must start with `policies/` and follow the format: `policies/namespace/name/version`

Example: `policies/hashicorp/CIS-Policy-Set-for-AWS-Terraform/1.0.1`

## Examples

```bash
# Fetch policy readme as plain text
tfdc policy get -id policies/hashicorp/CIS-Policy-Set-for-AWS-Terraform/1.0.1

# Fetch as structured JSON
tfdc policy get -id policies/hashicorp/CIS-Policy-Set-for-AWS-Terraform/1.0.1 -format json
```

## JSON output

```json
{
  "id": "policies/hashicorp/CIS-Policy-Set-for-AWS-Terraform/1.0.1",
  "content": "# CIS Policy Set\n\nThis policy set contains CIS benchmark rules...",
  "content_type": "text/markdown"
}
```

## Workflow

Typically used after `tfdc policy search`:

```bash
# Search for policies
tfdc policy search -query cis -format json

# Get full details
tfdc policy get -id policies/hashicorp/CIS-Policy-Set-for-AWS-Terraform/1.0.1
```
