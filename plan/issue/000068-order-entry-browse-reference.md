# ISSUE-000068: `order_entry.recent_orders`を実Reference Patternとして固定する

- 状態: Done — 実装済み（`order_entry.recent_orders`はReference Workspaceにauthoring済み。Application Pattern Stage 1 / Gate A Ready）
- GitHub Issue: #100
- 上位Issue: [`ISSUE-000005`](000005-application-pattern-reference.md)

## 背景

Application Patternの`browse`基盤は、Canonical model、Parser、Loader、Normalization、Source Index、参照整合性Validation、Diagnostic projectionまで段階的に実証してきました。

ここまでの検証は、基盤の責務を小さいfixtureで確認するものが中心です。次は既存のv1 Reference Applicationである`order_entry`へ実際のPatternを置き、M0.xxで積み上げた既存資産とApplication Patternを接続します。

`order_entry`には、すでに次があります。

- Semantic Data `sales.order`
- Query `order_entry.recent_orders`
- native SQL `queries/recent_orders.sql`
- logical datasource `sales`
- `sales_core`へのread-write接続

新しい業務Semanticを追加しなくても、これらをPatternから参照することで`browse`の実Referenceを成立させられます。

## 目的

`order_entry.recent_orders`をApplication Patternの実ReferenceとしてRepositoryへ固定し、次を一続きに確認します。

- 既存Data / Queryを内容の複製なしに参照できる
- Queryがすでに持つDatasourceを理由なく二重管理しない
- Pattern追加後もReference Worldを再現できる
- file pathとPattern identityを実Referenceでも分離できる

## Authoring source

Patternの正本はADR-0018に従い、Application配下の`patterns/*.tei`へ置きます。

最初のReferenceは概ね次の形とします。

```text
pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}
```

Query `order_entry.recent_orders`はすでにDatasource `sales`を保持しています。そのため、Pattern側ではDatasourceを重複して書かない形を第一候補とします。

Pattern側にもDatasourceを明示しなければ意味が不足するEvidenceが得られた場合は、その理由を記録して見直します。単に同じ値を二箇所へ手入力するためのfieldにはしません。

## 実装方針

### 実Reference Patternを追加する

`reference/workspace/applications/order-entry/patterns/`へPattern sourceを追加します。

次を既存identityで参照します。

- Data: `sales.order`
- Query: `order_entry.recent_orders`
- Datasource: Query側の`sales`

native SQL本文やQuery定義をPatternへ複製しません。

### Reference Worldを更新する

`go run ./cmd/tei world -workspace reference/workspace`の結果と`reference/world.json`を一致させます。

Golden差分にはPattern追加によるCanonicalな意味だけが現れることを確認します。file path、file name、source range等のauthoring都合をCanonical Worldへ持ち込みません。

### ReferenceのValidationを確認する

Pattern固有Validationを適用し、Data / Query / Datasourceの参照が成立していることを確認します。

共通`validation.Validate`への統合は[`ISSUE-000010`](000010-application-pattern-validation-integration.md)で扱います。両Issueの実装順は固定しませんが、最終的には通常のWorkspace validationからも同じReferenceを検査できる状態へ合流させます。

### 実Referenceでrename invarianceを確認する

PR #89では小さいfixtureで、Pattern Subject identityとfile pathを分離できることを確認しました。

このIssueでは実Reference Patternのファイル名だけを変更してWorkspaceを再構築し、次を確認します。

- Pattern Canonical IDが変わらない
- normalized Canonical Worldが変わらない
- Source IndexのSubject identityが変わらない
- Source Artifact pathだけが新しいpathへ追随する
- 古いpathが再構築後のSource Indexへ残らない

必要に応じてfingerprintとsource rangeも新しいauthoring sourceと一致することを確認します。

## 設計上の境界

- Pattern identityをfile pathから導出しない
- native SQLをPatternへコピーしない
- Queryが持つDatasourceを理由なく二重管理しない
- `browse`をWeb画面テンプレートとして定義しない
- PatternからPersistence MappingやDatabase FKを暗黙生成しない
- このReference一件を理由に5分類を永久的な標準語彙へ昇格させない

## 非目標

- `record` / `reference` / `command` / `header-detail`の追加
- `OrderLine`の追加
- Web Targetの設計
- Driverの実装
- Application Pattern Catalogの標準化
- 共通`validation.Validate`へのPattern Validation統合

## 完了条件

次をRepository内のReference / Conformanceから再現できることを完了条件とします。

1. `order_entry.recent_orders`が実Reference Patternとして読み込まれる
2. `sales.order`と既存native SQL Queryをidentityで参照し、内容を複製しない
3. Datasourceの不要な二重管理を避けられる
4. Patternを含む`reference/world.json`をRepositoryだけから再生成できる
5. 実Referenceのファイル名を変えてもCanonical WorldとPattern Subject identityが変わらない
6. Source IndexのArtifact pathだけが新しいauthoring sourceへ追随する

このEvidenceが揃った後、`record` / `reference` / `command`を順次追加します。

## 関連

- GitHub Issue #100
- GitHub Issue #67
- GitHub Issue #98
- PR #95
- PR #99
- ADR-0017
- ADR-0018
- `plan/design/APPLICATION_PATTERN_V1_DESIGN.md`

関連RQ: RQ-001, RQ-002, RQ-003, RQ-014, RQ-017, RQ-021
