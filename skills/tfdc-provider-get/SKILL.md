---
name: tfdc-provider-get
description: Fetch full Terraform provider documentation content by provider_doc_id using tfdc. Use when you have a provider_doc_id from a previous search and need the complete documentation content for a specific resource, data source, or other provider doc.
---

# tfdc provider get

Fetch full provider doc content by `provider_doc_id`.

## Usage

```bash
tfdc provider get -doc-id <id> [-format text]
```

## Flags

| Flag | Required | Default | Description |
|---|---|---|---|
| `-doc-id` | Yes | | Numeric provider doc ID (from `provider search`) |
| `-format` | No | `text` | Output format: `text`, `json`, `markdown` |

## Examples

```bash
# Fetch doc as plain text (default)
tfdc provider get -doc-id 10595066

# Fetch as structured JSON
tfdc provider get -doc-id 10595066 -format json

# Fetch as markdown
tfdc provider get -doc-id 10595066 -format markdown
```

## JSON output

```json
{
  "id": "10595066",
  "content": "# Resource: aws_instance\n\nProvides an EC2 instance resource...",
  "content_type": "text/markdown"
}
```

## Workflow

Typically used after `tfdc provider search`:

```bash
# Search for the resource
tfdc provider search -name aws -service instance -type resources -format json

# Get the full doc by ID
tfdc provider get -doc-id 10595066
```
