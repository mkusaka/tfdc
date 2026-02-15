---
name: tfdc-policy-search
description: Search Terraform policy sets using tfdc. Use when you need to find Sentinel or OPA policy sets by keyword, discover CIS benchmarks or compliance policies, or get policy IDs for use with tfdc policy get.
---

# tfdc policy search

Search Terraform policy sets.

## Usage

```bash
tfdc policy search -query <keyword> [-format text]
```

## Flags

| Flag | Required | Default | Description |
|---|---|---|---|
| `-query` | Yes | | Search query (e.g., `cis`, `aws`, `networking`) |
| `-format` | No | `text` | Output format: `text`, `json`, `markdown` |

## Output fields

| Field | Description |
|---|---|
| `terraform_policy_id` | Policy ID in `policies/namespace/name/version` format |
| `name` | Policy set name |
| `title` | Policy set title |
| `downloads` | Download count |

## Examples

```bash
# Search for CIS policies
tfdc policy search -query cis

# Search with JSON output
tfdc policy search -query aws -format json
```

## JSON output

```json
{
  "items": [
    {
      "terraform_policy_id": "policies/hashicorp/CIS-Policy-Set-for-AWS-Terraform/1.0.1",
      "name": "CIS-Policy-Set-for-AWS-Terraform",
      "title": "Pre-written Sentinel Policies for AWS CIS Foundations Benchmarking",
      "downloads": 647442
    }
  ],
  "total": 1
}
```

## Workflow

Use with `tfdc policy get` to fetch full policy details:

```bash
# Search
tfdc policy search -query cis -format json | jq '.items[].terraform_policy_id'

# Get details
tfdc policy get -id policies/hashicorp/CIS-Policy-Set-for-AWS-Terraform/1.0.1
```
