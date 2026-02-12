# terraform-docs-cli

CLI version of <https://github.com/hashicorp/terraform-mcp-server>.

## Interface Design

See the proposed command interface and output contract here:

- [`docs/cli-interface.md`](docs/cli-interface.md)

## Specification

- English: [`docs/spec.md`](docs/spec.md)
- 日本語: [`docs/spec.ja.md`](docs/spec.ja.md)

## Planned MVP Scope

- Provider docs: search and fetch by doc ID
- Provider docs: bulk export for a fixed provider/version directory layout
- Module docs: search and fetch by module ID
- Policy docs: search and fetch by policy ID
- Terraform guides: style guide and module development guide
- Persistent local cache (TTL-based)

## Example (Provider Export)

```bash
go run ./cmd/terraform-docs-cli \
  --cache-ttl 24h \
  provider export \
  --namespace hashicorp \
  --name aws \
  --version 6.31.0 \
  --format markdown \
  --out-dir ./dir \
  --categories guides
```

Generated output example:

`dir/terraform/aws/6.31.0/docs/guides/tag-policy-compliance.md`
