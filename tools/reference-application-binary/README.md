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
- 入力は `--name=value` だけです。足りない・余分・重複・位置引数は、DBへ接続する前に終了コード2で拒否します。Queryが入力の型を宣言していればその型が権威で、`std.datetime` はRFC 3339として検査し、合わなければDBへ接続する前に終了コード2で拒否します。生成器が検査できない型の宣言は、生成前に `INPUT_TYPE_UNSUPPORTED` で止まります。
- `receipt.json` はmanifestのdigestとBinaryのdigestを対にします。`browsegen.Verify` が古いBinaryを検出します。

主張できる範囲: 「このSource版のQueryを忠実に実現した」まで。業務意味の受理、Environment/Provider束縛、実Grant、行の完全性（0行が不存在を意味するか）は主張しません。

既知の限界:

- `recent_orders` は `input from std.datetime` を宣言しています。宣言のない引数は `text-undeclared` として文字列のまま渡し、`manifest.json` の `inputDeclaration` に `undeclared` と残ります。
- `.sql` はSource Indexの対象外です。`SourceRevision` には `.sql` のSHA-256を別に加えています。
- 実PostgreSQL試験は `TEI_TASK9_PG_DSN`（データベース名 `tei_task9_probe`）を指定した場合だけ動きます。Seedは `examples/data-source-workbench/recent_orders_product_seed.sql`、独立Expectedは `testdata/browse_recent_orders_expected.json` です。

## `-function` なしの旧モード

`databases/` のMigrationを抽出して `migrations.go` を生成します。Query・Command・Validationは生成器内の固定内容で、Sourceに依存しません（Migration以外はSourceを変えても出力が変わらないことを2026-11-12に確認）。P0-Aの試験はこの旧モードを対象とします。

## `-validations` モード（Validationの規則の門）

Application が `uses validation` で宣言した Validation を、宣言順に評価して最初の不成立で止める Go CLI を生成します。DB・Authorization・効果は持ちません。

```bash
go run ./tools/reference-application-binary/cmd/tei-application-binary \
  -workspace reference/workspace -application order_entry -validations -out /tmp/order-entry-validate
/tmp/order-entry-validate/bin/validate validate order_entry < input.json
```

- 入力は標準入力のJSON（`numbers` と `relationKeys`。`reference/driver-scenarios/fixtures/` と同じ形）。
- 終了コード: 0 全て成立、1 不成立、2 引数不正またはJSONでない入力、3 評価を完了できない（値なし・数値でない・キーなし）。評価不能は業務上の不成立にしません。
- 対応する種類は `positive` と `relation-unique` だけです。それ以外の種類を宣言していれば、生成前に `VALIDATION_KIND_UNSUPPORTED` で止まります。
- 未生成の Pattern は `manifest.json` に列挙します。Command の効果は Source に本文がないため `effect-not-in-source` として残し、生成しません。
- 規則の評価は `validationeval` の第二実装です。生成物と既存の Evaluator を同じ入力に流す差分試験（`TestVGGeneratedGateMatchesFixedExpected`）と、手で固定した期待値（`testdata/validation_gate_expected.json`）で照合し、わずかに誤った生成器4種（0を許す、重複を見逃す、短絡しない、評価不能を拒否に畳む）が落ちることを `TestVGMutantsAreCaught` で確認しています。

主張できる範囲: 「宣言された規則を、宣言された順序と3値で評価する」まで。`submit_order` の成立、Authorization、効果、入力データの取得元は範囲外です。
