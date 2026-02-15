---
name: tfdc-guide-style
description: Fetch the official Terraform style guide using tfdc. Use when you need Terraform coding conventions, formatting rules, naming standards, or best practices for writing HCL code.
---

# tfdc guide style

Fetch the official Terraform style guide from HashiCorp docs.

## Usage

```bash
tfdc guide style [-format text]
```

## Flags

| Flag | Required | Default | Description |
|---|---|---|---|
| `-format` | No | `text` | Output format: `text`, `json`, `markdown` |

## Examples

```bash
# Fetch style guide as plain text
tfdc guide style

# Fetch as JSON
tfdc guide style -format json
```

## JSON output

```json
{
  "id": "style-guide",
  "content": "# Terraform Style Guide\n\n...",
  "content_type": "text/markdown"
}
```

## Content

The style guide covers Terraform coding conventions including:

- File and directory structure
- Naming conventions
- Variable and output formatting
- Resource ordering
- Code organization best practices
