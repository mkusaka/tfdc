# terraform-docs-cli 仕様書

## 1. 目的

`terraform-docs-cli` は Terraform Registry および Terraform ドキュメントを参照し、Terraform の実装・レビューに必要なドキュメントを CLI から取得するためのツールである。

本仕様は以下を定義する。

- CLI の機能要件
- コマンドとフラグの入出力契約
- 終了コードとエラー挙動
- 実装時の内部構成方針
- テスト観点

## 2. スコープ

### 2.1 対象（In Scope）

- Public Terraform Registry からの取得
- Provider ドキュメントの検索と詳細取得
- Module の検索と詳細取得
- Policy の検索と詳細取得
- Terraform style guide / module development guide の取得
- `text|json|markdown` 出力
- 永続キャッシュ（ローカルディスク）
- 特定 provider/version の docs 一括エクスポートとディレクトリ永続化

### 2.2 非対象（Out of Scope）

- TFE/TFC private registry の認証付き操作
- Terraform 実行系操作（plan/apply/run 管理）

## 3. 用語

- `provider_doc_id`: provider-docs 詳細取得に使う数値 ID
- `module_id`: `namespace/name/provider/version` 形式 ID
- `terraform_policy_id`: `policies/<namespace>/<name>/<version>` 形式 ID

## 4. 全体方針

`hashicorp/terraform-mcp-server` の registry ツール実装に揃える。

- Provider: `search -> get`
- Module: `search -> get`
- Policy: `search -> get`
- Guide: raw markdown を直接取得

## 5. CLI 仕様

### 5.1 コマンドツリー

```text
terraform-docs-cli [global flags] <group> <command> [flags]

group:
  provider
  module
  policy
  guide
```

### 5.2 グローバルフラグ

```text
-chdir          作業ディレクトリ変更 (.terraform.lock.hcl自動検出)
-timeout        HTTPタイムアウト      (default: 10s)
-retry          リトライ回数          (default: 3)
-registry-url   RegistryベースURL     (default: https://registry.terraform.io)
-insecure       TLS検証スキップ
-user-agent     User-Agent上書き
-debug          デバッグログ出力
-cache-dir      キャッシュディレクトリ (default: ~/.cache/terraform-docs-cli)
-cache-ttl      キャッシュ有効期間     (default: 24h)
-no-cache       キャッシュを無効化
```

## 6. コマンド詳細

### 6.1 Provider

#### 6.1.1 `provider search`

Provider ドキュメント候補を検索し、`provider_doc_id` を返す。

```text
terraform-docs-cli provider search \
  -name aws \
  -namespace hashicorp \
  -service ec2 \
  -type resources \
  [-version latest] \
  [-limit 20]
```

入力:

- `-name` 必須
- `-namespace` 省略時 `hashicorp`
- `-service` 必須
- `-type` 必須
  `resources|data-sources|ephemeral-resources|functions|guides|overview|actions|list-resources`
- `-version` 省略時 `latest`
- `-limit` 省略時 `20`

出力項目:

- `provider_doc_id`
- `title`
- `category`
- `description`
- `provider`
- `namespace`
- `version`

#### 6.1.2 `provider get`

`provider_doc_id` を指定して全文取得。

```text
terraform-docs-cli provider get -doc-id 8894603
```

入力:

- `-doc-id` 必須（数値）

出力:

- text/markdown: 本文のみ
- json: `id`, `content`, `content_type`

#### 6.1.3 `provider doc`

検索と取得を1コマンドで実行する convenience command。

```text
terraform-docs-cli provider doc \
  -name aws -namespace hashicorp -service ec2 -type resources \
  [-version latest] [-select best]
```

入力:

- `-select`: `best|first|interactive`（default: `best`）

選択ルール:

- `best`: slug/title 類似スコア上位
- `first`: 先頭候補
- `interactive`: 候補表示後に番号選択

#### 6.1.4 `provider latest-version`

```text
terraform-docs-cli provider latest-version -namespace hashicorp -name aws
```

#### 6.1.5 `provider capabilities`

```text
terraform-docs-cli provider capabilities -namespace hashicorp -name aws [-version latest]
```

#### 6.1.6 `provider export`

特定 provider の特定 version の docs を全件取得し、指定ディレクトリ配下に保存する。

```text
terraform-docs-cli provider export \
  -namespace hashicorp \
  -name aws \
  -version 6.31.0 \
  -format markdown \
  -out-dir ./dir \
  [-categories all] \
  [-path-template "{out}/terraform/{namespace}/{provider}/{version}/docs/{category}/{slug}.{ext}"] \
  [-clean]
```

入力:

- `-namespace` 省略時 `hashicorp`
- `-name` 必須
- `-version` 必須（明示指定）
- `-format` `markdown|json`（default: `markdown`）
- `-out-dir` 必須
- `-categories` 省略時 `all`（`resources,data-sources,ephemeral-resources,functions,guides,overview,actions,list-resources`）
- `-path-template` 省略時デフォルトレイアウト
- `-clean` 指定時は既存 manifest ファイルを削除し、さらに導出されたテンプレートルートが namespace/provider/version でスコープされる場合のみそのルートを削除して再生成

デフォルト保存パス:

- `{out}/terraform/{namespace}/{provider}/{version}/docs/{category}/{slug}.{ext}`
- 例: `dir/terraform/hashicorp/aws/6.31.0/docs/guides/tag-policy-compliance.md`

補足:

- `{ext}` は `markdown => md`, `json => json`
- エクスポート結果の一覧として namespace スコープの `_manifest.json` を出力する
  (`{out}/terraform/{namespace}/{provider}/{version}/docs/_manifest.json`)

### 6.2 Module

#### 6.2.1 `module search`

```text
terraform-docs-cli module search -query vpc [-offset 0] [-limit 20]
```

出力項目:

- `module_id`
- `name`
- `description`
- `downloads`
- `verified`
- `published_at`

#### 6.2.2 `module get`

```text
terraform-docs-cli module get -id terraform-aws-modules/vpc/aws/6.0.1
```

入力制約:

- `module_id` は `namespace/name/provider/version` の4セグメント必須

#### 6.2.3 `module latest-version`

```text
terraform-docs-cli module latest-version \
  -publisher terraform-aws-modules \
  -name vpc \
  -provider aws
```

### 6.3 Policy

#### 6.3.1 `policy search`

```text
terraform-docs-cli policy search -query cis
```

出力項目:

- `terraform_policy_id`
- `name`
- `title`
- `downloads`

#### 6.3.2 `policy get`

```text
terraform-docs-cli policy get -id policies/hashicorp/CIS-Policy-Set-for-AWS-Terraform/1.0.1
```

### 6.4 Guide

#### 6.4.1 `guide style`

```text
terraform-docs-cli guide style
```

#### 6.4.2 `guide module-dev`

```text
terraform-docs-cli guide module-dev [-section all]
```

`-section`:

- `all|index|composition|structure|providers|publish|refactoring`

## 7. 出力仕様

### 7.1 text

- 人間向け整形
- 詳細取得コマンドは本文をそのまま出力

### 7.2 markdown

- Markdown を壊さずそのまま出力
- 詳細系/guide では text と同等、search 系は markdown table 可

### 7.3 json

- スクリプト実行向けに安定フィールドを保証
- 検索系は以下フォーマット

```json
{
  "items": [],
  "total": 0
}
```

- 詳細系は以下フォーマット

```json
{
  "id": "string",
  "content": "string",
  "content_type": "text/markdown"
}
```

- `provider export` は以下フォーマットを返す

```json
{
  "provider": "aws",
  "version": "6.31.0",
  "out_dir": "./dir",
  "written": 123,
  "manifest": "dir/terraform/hashicorp/aws/6.31.0/docs/_manifest.json"
}
```

## 8. 終了コード

```text
0 成功
1 引数不正 / バリデーション失敗
2 対象なし（Not Found）
3 リモートAPIエラー
4 シリアライズ/書き込み失敗
```

## 9. エラーハンドリング

- エラーメッセージは標準エラー出力へ出す
- `-debug` 有効時は HTTP ステータス、URL、retry 情報を追加
- `404` は終了コード `2` にマップ
- `429/5xx` は retry 後に失敗した場合終了コード `3`

## 10. 内部実装方針

### 10.1 パッケージ構成（想定）

```text
cmd/terraform-docs-cli/
internal/cli/           # cobra command定義
internal/client/        # HTTP client, retry, tls
internal/registry/      # provider/module/policy API呼び出し
internal/formatter/     # text/json/markdown formatter
internal/output/        # stdout/file writer
```

### 10.2 HTTP クライアント

- timeout/retry/insecure をグローバルフラグから注入
- User-Agent は `terraform-docs-cli/<version>`

### 10.3 検証

- Provider: doc id 数値チェック
- Module: ID セグメント数チェック
- Policy: ID プレフィックス検証（`policies/`）

### 10.4 永続キャッシュ

- 対象コマンド:
  - `provider search|get`
  - `provider export`
  - `module search|get`
  - `policy search|get`
  - `guide style|module-dev`
- キャッシュキーは「HTTPメソッド + 正規化済みURL + 主要クエリ」で構成する
- TTL を超過したエントリはミスとして再取得し、取得成功後に上書きする
- `-no-cache` 指定時は read/write ともに無効化する
- キャッシュ破損時は当該エントリを破棄して再取得し、処理は継続する

## 11. テスト仕様

### 11.1 単体テスト

- 引数バリデーション
- レスポンスパース
- format 出力（text/json/markdown）
- 終了コードマッピング

### 11.2 結合テスト

- 主要コマンドのハッピーパス
- not found ケース
- API エラーケース（5xx / timeout）
- キャッシュ hit / miss / TTL 期限切れ
- `-no-cache` 時に read/write されないこと
- `provider export` が期待ディレクトリ構造で出力されること
- `provider export -clean` が再生成動作になること

### 11.3 スナップショットテスト

- `provider search/get`
- `provider export`
- `module search/get`
- `policy search/get`
- `guide style/module-dev`

## 12. リリース基準（MVP）

以下を満たしたら MVP 完了とする。

- 6.1.1, 6.1.2, 6.1.6, 6.2.1, 6.2.2, 6.3.1, 6.3.2, 6.4.1, 6.4.2 が動作
- `text/json/markdown` 出力が安定
- 終了コードが仕様通り
- 永続キャッシュが仕様通り（TTL, no-cache, 破損回復）
- `provider export` の出力レイアウトが仕様通り（path-template, manifest）
- README に使用例を記載
