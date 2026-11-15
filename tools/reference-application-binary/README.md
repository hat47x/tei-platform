# reference-application-binary

TEIワークスペースからGoコードを生成して単一バイナリを作る参照です。二つのモードがあります。製品の受入ではなく、意味保存の検査を試す参照実装です。

## `-function` モード（Source束縛のread-only Function）

browse Pattern 1つを、Sourceから決まる範囲だけ生成します。

```bash
go run ./tools/reference-application-binary/cmd/tei-application-binary \
  -workspace reference/workspace -application order_entry \
  -function order_entry.recent_orders -target postgres -out /tmp/recent-orders
```

- Sourceは既存の `workspace.Load` と `driver.Resolver` で読みます。別のParserは使いません。
- 生成物は `main.go`、`go.mod`、`manifest.json`、`bin/app`、`receipt.json` です。SQLはSourceのQuery本文を束縛した形で入り、READ ONLYトランザクションで実行します。
- Application が宣言していて生成しなかった Pattern・Validation・Database は、`manifest.json` の `plan.omissions` に理由つきで残ります。
- QueryのDialectが `-target` と違う場合は、ファイルを書く前に `TARGET_DIALECT_UNSUPPORTED` で止まります。
- 入力は `--name=value` だけです。足りない・余分・重複・位置引数は、DBへ接続する前に終了コード2で拒否します。
- `receipt.json` はmanifestのdigestとBinaryのdigestを対にします。`browsegen.Verify` が古いBinaryを検出します。

主張できる範囲: 「このSource版のQueryを忠実に実現した」まで。業務意味の受理、Environment/Provider束縛、実Grant、行の完全性（0行が不存在を意味するか）は主張しません。

既知の限界:

- `recent_orders` は入力宣言（`input from`）がありません。`manifest.json` の `inputDeclaration` は `undeclared` になり、SQL本文の `:from` を暫定の権威にします。
- `.sql` はSource Indexの対象外です。`SourceRevision` には `.sql` のSHA-256を別に加えています。
- 実PostgreSQL試験は `TEI_TASK9_PG_DSN`（データベース名 `tei_task9_probe`）を指定した場合だけ動きます。Seedは `examples/data-source-workbench/recent_orders_product_seed.sql`、独立Expectedは `testdata/browse_recent_orders_expected.json` です。

## `-function` なしの旧モード

`databases/` のMigrationを抽出して `migrations.go` を生成します。Query・Command・Validationは生成器内の固定内容で、Sourceに依存しません（Migration以外はSourceを変えても出力が変わらないことを2026-11-12に確認）。P0-Aの試験はこの旧モードを対象とします。
