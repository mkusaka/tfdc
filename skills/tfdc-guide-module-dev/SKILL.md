---
name: tfdc-guide-module-dev
description: Fetch the Terraform module development guide using tfdc. Use when you need guidance on building, structuring, publishing, or refactoring Terraform modules. Supports fetching specific sections like composition, structure, or providers.
---

# tfdc guide module-dev

Fetch the Terraform module development guide from HashiCorp docs.

## Usage

```bash
tfdc guide module-dev [-section all] [-format text]
```

## Flags

| Flag | Required | Default | Description |
|---|---|---|---|
| `-section` | No | `all` | Section to fetch (see below) |
| `-format` | No | `text` | Output format: `text`, `json`, `markdown` |

## Sections

| Section | Description |
|---|---|
| `all` | Fetch all sections concatenated |
| `index` | Overview and introduction |
| `composition` | Module composition patterns |
| `structure` | Directory and file structure |
| `providers` | Provider configuration in modules |
| `publish` | Publishing modules to the registry |
| `refactoring` | Refactoring and versioning modules |

## Examples

```bash
# Fetch entire module development guide
tfdc guide module-dev

# Fetch only the structure section
tfdc guide module-dev -section structure

# Fetch composition guide as JSON
tfdc guide module-dev -section composition -format json
```

## JSON output

```json
{
  "id": "module-dev/structure",
  "content": "# Module Structure\n\n...",
  "content_type": "text/markdown"
}
```

When `-section all` is used, the `id` field is `"module-dev"`.
