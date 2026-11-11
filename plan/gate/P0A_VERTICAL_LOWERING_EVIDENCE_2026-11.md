# P0-A 判定記録：旧 .tei 縦断の独立Expected再現

- **判定対象**: `tools/reference-application-binary`（PR [#4293](https://github.com/hat47x/tei-platform/pull/4293)）による Parse → Model → Generate → Build の縦断
- **設計正本**: [TEI_MD_DOMAIN_DIAGRAM_TO_BUILD_LOWERING_2026-11.md](../design/TEI_MD_DOMAIN_DIAGRAM_TO_BUILD_LOWERING_2026-11.md) P0-A
- **実行日**: 2026-11-11
- **実行HEAD**: `feat/issue-4218-step-code-adapter`（#4293 の内容を含む）
- **判定**: **P0-A 成立（旧 .tei 限定）**。ただし Execute 段階の実Provider適合は **unverified**（K-07）。

## 1. 独立Expected（Sourceから固定、実装から逆生成していない）

Source: `reference/workspace/databases/sales-core/migrations/20260826_001/migration.tei`

| 項目 | 独立Expected |
| --- | --- |
| Migration ID | `20260826_001` |
| Database | `sales_core` |
| Recovery | `forward-fix` |
| SchemaSQL | `CREATE INDEX orders_customer_idx\nON public.orders (customer_id);` |
| DataSQL | `UPDATE public.orders\nSET status = 'UNKNOWN'\nWHERE status IS NULL;` |
| VerifySQL | `SELECT COUNT(*)\nFROM public.orders\nWHERE status IS NULL;` |
| Dialect | `postgres`（schema/data/verify すべて） |

生成ファイル名集合（manifest.json を除くGo Source）:
`main.go, config.go, models.go, queries.go, commands.go, validations.go, errors.go, provenance.go, migrations.go`

Provenance fields: `applicationId=order_entry`, `generator=tei.support.application-binary`

## 2. 実行した検証（段階別・独立Expected照合）

検証実装: `tools/reference-application-binary/p0a_vertical_test.go`

| 段階 | テスト | 結果 | 照合内容 |
| --- | --- | --- | --- |
| (A)/(C) Parse | `TestP0AStageParse` | **PASS** | Migration ID/Database/Recovery/SchemaSQL/DataSQL/VerifySQL が独立Expectedと完全一致 |
| (C) Model | `TestP0AStageModel` | **PASS** | `ExtractModel` が `GenerationContext.Migrations` に同一内容を運ぶ |
| (D) Generate | `TestP0AStageGenerate` | **PASS** | 生成ファイル集合一致、`migrations.go` に migration ID・`applyMigrations`・`applyMigration20260826_001` 呼出・SchemaSQL 逐語・`DO NOT EDIT` provenance header |
| (E) Build | `TestP0AStageBuild` | **PASS** | 単一バイナリ生成（非空）、`provenance` 出力の `applicationId`/`generator` が独立Expectedと一致 |

実行ログ（要旨）:
```
=== RUN   TestP0AStageParse       --- PASS (0.01s)
=== RUN   TestP0AStageModel       --- PASS (0.01s)
=== RUN   TestP0AStageGenerate    --- PASS (0.20s)
=== RUN   TestP0AStageBuild       --- PASS (163.03s)
ok  github.com/hat47x/tei-platform/tools/reference-application-binary
```

## 3. 成立しなかったもの（自動継承しない）

| 項目 | 状態 | 理由 |
| --- | --- | --- |
| (F) Execute 実Provider適合 | **unverified** | workspace migration SQL は PostgreSQL 方言（`public.orders` 等）。SQLite では `near "."`: syntax error。方言非互換の実行を pass/fail とみなさない（K-07） |
| 新 `*.tei.md` Reader 接続 | **NOT RUN** | P0-A は旧 `.tei` 限定。Reader は #3030 M2/M3 の既存Owner。本PASSを新形式へ継承しない |
| Owner意味受理 | **NOT ACCEPTED** | 本記録は機械的独立Expected照合のみ。Shared Acceptance（#267）は別資格 |
| 実Host Grant / 業務Effect | **NOT RUN** | #1174/#2516 の既存Owner |

## 4. 段階分離の確認（DG-04）

Parse PASS・Model PASS・Generate PASS・Build PASS は**それぞれ独立のClaim**である。本記録では前段の成功を次段の合格として自動継承していない。Execute 段階は方言非互換のため `unverified` として明示し、P0-A の合格範囲に含めない。

## 5. 次の手順（P1-B）

P0-A 成立を受けて、P1-B（`*.tei.md` サンプルのread-only確認）へ進む。P1-B は Reader 未接続のまま、図表・表・説明文の共存と原文保全のみを確認する。実Source消費・Meaning受理・Build資格は読み替えない。
