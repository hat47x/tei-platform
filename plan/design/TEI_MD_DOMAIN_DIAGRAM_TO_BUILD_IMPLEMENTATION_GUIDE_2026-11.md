# *.tei.md ドメイン図表→Build 実装導線（P0-A成立後）

- **状態**: 実装導線。設計・受入の正本は [TEI_MD_DOMAIN_DIAGRAM_TO_BUILD_LOWERING_2026-11.md](TEI_MD_DOMAIN_DIAGRAM_TO_BUILD_LOWERING_2026-11.md)。本書は**後続実装者が迷わず最後まで作業するための運用手順**であり、設計の第二正本を作らない。
- **作成日**: 2026-11-11
- **前提**: P0-A 成立（[判定記録](https://github.com/hat47x/tei-platform/pull/4310)）。PR [#4293](https://github.com/hat47x/tei-platform/pull/4293)（実装）・[#4295](https://github.com/hat47x/tei-platform/pull/4295)（設計）。
- **対象読者**: P1-B〜P3-D を実装する担当者。

## 0. 現在地（P0-A で確定した事実）

| 事実 | 証拠 |
| --- | --- |
| 旧 `.tei` の Parse→Model→Generate→Build が独立Expectedで縦断成立（**Migrationに限る**。Query・Command・Validationは生成器内の固定内容でSourceに依存しない。2026-11-13追記、下の注記参照） | `tools/reference-application-binary/p0a_vertical_test.go`、[P0-A判定記録](https://github.com/hat47x/tei-platform/pull/4310) |
| migration SQL の抽出・生成・起動適用が動作 | #4293 |
| 生成物は外部runtimeなしの単一バイナリ | #4293（`modernc.org/sqlite` 埋め込み） |
| workspace migration SQL は PostgreSQL 方言で SQLite 非互換 | K-07。Execute は `unverified` |
| `*.tei.md` の実Readerは未接続 | #3030 M2/M3 が主担当。本書は受入照合のみ |

**P0-A のPASSが自動的に保証しないもの**: 新 `*.tei.md` Reader、Execute 実Provider適合、Owner意味受理、実Host Grant。

**2026-11-13 追記（範囲の訂正）**: P0-A が縦断で確認したのは `databases/` の Migration 抽出・生成だけである。旧モードの生成物のQuery・Command・Validationはテンプレートではなく生成器内の固定文字列で、Source のSQLやValidation宣言を変えても生成コードは変わらない（main `4b1a3973` の写しで実測）。また生成物はSQLite埋め込みで、`query` は起動時のPG方言Migrationで失敗する。Source由来の生成が成立しているのは `tools/reference-application-binary` の `-function` モード（browse Pattern 1つ、PostgreSQL、READ ONLY）だけで、[README](../../tools/reference-application-binary/README.md) に範囲と限界を書いた。これは「このSource版のQueryの忠実な実現」までで、業務意味の受理・Environment/Provider束縛・Host Grantは未成立。

## 1. 全体の道順と判定条件

```text
P0-A 旧 .tei 縦断再現          [DONE] 独立Expected PASS
  |
P1-B *.tei.md サンプル read-only [NOW] 本書 §2
  |
P2-C Reader接続・旧新比較         [#3030主担当] 本書 §3
  |
P3-D 複数Consumer一致             [#1124/#2497/#2516] 本書 §4
```

**共通の判定規則**: 各段階は「前段のPASSを次段の合格に自動継承しない」。Execute・Meaning・Grant は独立のClaim。1段階でも独立Expectedと不一致なら、その段階を `fails` とし、他段階のPASSで覆わない。

## 2. P1-B: `*.tei.md` サンプル read-only（現在の作業）

### 2.1 成果物
- `reference/candidates/domain-diagram-to-build/order_entry.tei.md`（作成済み、合成サンプル）

### 2.2 受入条件（すべて満たすこと）

| # | 受入 | 確認方法 |
| --- | --- | --- |
| B-1 | ファイルが Workspace 宣言Source Root **外** にある（`reference/candidates/`） | `reference/workspace/tei.workspace.yaml` の sources に対象パスが含まれない |
| B-2 | Markdownとして表示・保存可能（byte 不変） | `git diff` で byte 不変、Mermaid/表が rendering 可能 |
| B-3 | 図表のSubject集合が意味正本と対応 | §1 の Subject（`customer.customer`, `sales.order`, `sales.order_line`）が `reference/workspace/semantic/` に存在 |
| B-4 | 説明文と定義の境界が明示 | frontmatter・prose・Mermaid・表・fence の役割が文書内で説明されている |
| B-5 | 合成サンプルであることを明示 | frontmatter `status: draft`、冒頭の位置づけ注記 |

### 2.3 禁止事項
- サンプルの存在を「実Source消費」「Meaning受理」「Build資格」と読み替えない。
- サンプルを Workspace の宣言Source Root内へ移動しない（#3031 の `TEI_MD_PARSER_UNBOUND` がWorkspace Load全体を停止させ得る）。
- サンプルを `.tei` へ rename して旧Loaderに読ませない。

### 2.4 次の一手
P2-C のReaderが実装されるまで、このサンプルは**表示・原文保全の確認のみ**。§7 の Lowering map は「将来Readerが読むべき構造の候補」であり、現時点で実行しない。

## 3. P2-C: Reader接続・旧新比較（#3030主担当、本書は受入照合のみ）

### 3.1 依存
- #3030 M2/M3（Markdown span reader / Loader対応）が実装・独立試験されること。
- #988（Language Plugin/Parser選択）。
- #2823（Loader実読込）。

### 3.2 受入条件（Reader実装後の照合）

| # | 受入 | 独立Expected |
| --- | --- | --- |
| C-1 | §1 の Mermaid ER図から `Order`/`OrderLine`/`Customer` の Subject を抽出 | P0-A の `DataModel` 集合と一致 |
| C-2 | §6 のマイグレーション表から `20260826_001` を抽出 | P0-A の `Migration`（ID/Database/Recovery/SQL）と一致 |
| C-3 | §4 の業務ルール表から Rule closure を抽出 | `application.tei` の `uses validation` と一致 |
| C-4 | 旧 `.tei` と新 `*.tei.md` の同一Subjectで消失項目・重複・誤引用がない | 旧新比較で差分0、または差分は説明可能 |
| C-5 | 抽出→Generate→Build が縦断で成立 | P0-A と同様の独立Expected |

### 3.3 禁止事項
- 単純rename、同byte移動、`validate ok`、CI Green を新形式の合格としない。
- 旧 `.tei` と新 `*.tei.md` が同一Subjectのとき、先勝ち/後勝ちで解決しない。明示選択を要求する。
- Parse PASS を実Provider成立へ拡大しない。

## 4. P3-D: 複数Consumer一致（#1124/#2497/#2516主担当）

### 4.1 依存
- #1124（Diagram/Navigation）、#2497（投影意味保持）、#2516（Consumer資格）。

### 4.2 受入条件

| # | 受入 |
| --- | --- |
| D-1 | 図表・CLI・SHO が同じAccepted Meaning/Sourceから導出 |
| D-2 | 各ConsumerのScope/Coverage、未解決・非公開の集合が表示 |
| D-3 | 概観・現在地・近傍が生成AIにCLI/MCPから構造化テキストで渡せる |
| D-4 | 標準フロー／差分／合成後の実効フローが相互参照可能 |

## 5. 再利用Fixture一覧（新規作成禁止）

| Fixture | 所在 | 検証内容 |
| --- | --- | --- |
| P0-A独立Expected | `tools/reference-application-binary/p0a_vertical_test.go` | Parse/Model/Generate/Build |
| P0-A判定記録 | `plan/gate/P0A_VERTICAL_LOWERING_EVIDENCE_2026-11.md` | 段階別PASSと未達成の明示 |
| migration Source | `reference/workspace/databases/sales-core/migrations/20260826_001/migration.tei` | 独立Expectedの正本 |
| application Source | `reference/workspace/applications/order-entry/application.tei` | Rule closure |
| semantic Source | `reference/workspace/semantic/sales/order.tei`, `customer/customer.tei` | Subject集合 |
| P1-Bサンプル | `reference/candidates/domain-diagram-to-build/order_entry.tei.md` | 図表・表のドメイン知識 |

**新規作成禁止**: 第二のParser、Source Registry、Acceptance、Semantic SSOT、Test Runner、Host。既存OwnerのFixtureを再利用する。

## 6. エスカレーション（迷ったとき）

| 状況 |  escal先 |
| --- | --- |
| `*.tei.md` の文法・Role・Span が未決 | #2621（Authoring）／#988（Language Plugin） |
| Reader/Loader が読まない | #2823（Loader）／#3030（移行） |
| 図表投影・ナビゲーションの製品要件 | #1124（Diagram）／RQ-036 |
| 独立Expected の設計 | #2514／#1180 |
| 実Provider Read | #2622／#2628／#2586 |
| Owner意味受理 | #267（Shared Acceptance） |
| Host Grant / Effect | #1174／#2516 |
| 同じ失敗を独立に2回試して改善しない | 残論点を限定して該当Ownerへ。無限に再試行しない |

## 7. 各段階の「PASS」の定義（再掲）

| 段階 | PASSの意味 | PASSが保証しないもの |
| --- | --- | --- |
| Parse | Subject・Rule・SQL が独立Expectedと一致 | Generate/Build/Execute |
| Model | GenerationContext に同一内容 | Generate/Build/Execute |
| Generate | ファイル集合・内容・provenance が一致 | Build/Execute |
| Build | 単一バイナリが生成され起動する | Execute の実Provider適合 |
| Execute | 独立Expectedに対する実出力が一致 | 実Host Grant・Owner意味受理 |

## 8. 状態

- P0-A: **成立**（旧 `.tei` 限定）
- P1-B: **サンプル作成済み**。read-only受入は本書 §2.2 で確認可能
- P2-C: **NOT RUN**（#3030主担当）
- P3-D: **NOT RUN**（#1124/#2497/#2516主担当）
- 本書は運用手順の追補であり、新Parser/Runner/Registry/第二Acceptance/Semantic SSOT/Host は作らない。
