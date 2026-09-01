# ISSUE-000011: `order_entry.recent_orders`を実Reference Patternとして固定する

- 状態: 進行中
- GitHub Issue: #100
- 上位Issue: [`ISSUE-000005`](000005-application-pattern-reference.md)

## 背景

Application Patternの`browse`基盤は、Canonical model、Parser、Loader、Normalization、Source Index、参照整合性Validation、Diagnostic projectionまで段階的に実証してきました。

PR #101では、既存のv1 Reference Applicationである`order_entry`へ最初の実Reference Patternを追加し、Golden Worldまでmainへ固定しました。

このIssueで残っている主なEvidenceは、PR #89で小さいfixtureを使って確認したfile pathとPattern identityの分離を、実Referenceでも再現することです。

## PR #101で実装済み

### 実Reference Pattern

`reference/workspace/applications/order-entry/patterns/recent-orders.tei`に次が入りました。

```text
pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}
```

新しい業務SemanticやQueryを作らず、既存のSemantic Data `sales.order`とnative SQL Query `order_entry.recent_orders`をidentityで参照しています。

### Datasourceを二重管理しない

Query `order_entry.recent_orders`はすでにlogical datasource `sales`を保持しています。

Reference PatternではDatasourceを空のままにし、Query側の`sales`を利用します。Conformanceでも、Pattern側に同じDatasourceを重複記述していないことを確認しています。

### Reference World

Pattern追加後のnormalized Worldを`reference/world.json`へ反映しました。

Golden差分には次のCanonicalな意味だけが追加されています。

- ID: `order_entry.recent_orders`
- Kind: `browse`
- Data: `sales.order`
- Query: `order_entry.recent_orders`

file pathやsource range等のauthoring都合はCanonical Patternへ入りません。

### 実ReferenceのSource Index

実Reference Patternの`application-pattern:order_entry.recent_orders` Subjectから、`applications/order-entry/patterns/recent-orders.tei`へ戻れることも確認しています。

## 残っている作業

### 実Referenceでfile rename invarianceを確認する

実Reference Patternのファイル名だけを変更した一時Workspace等を使い、rename前後を比較します。

次を一続きに確認します。

- Pattern Canonical IDが変わらない
- normalized Canonical Worldが変わらない
- Source IndexのSubject identityが変わらない
- Source Artifact pathだけが新しいpathへ追随する
- 古いpathが再構築後のSource Indexへ残らない

必要に応じてfingerprintとsource rangeも新しいauthoring sourceと一致することを確認します。

Reference Repository上の実ファイル名を恒久的に変更する必要はありません。Conformanceの中でrename前後の意味を比較できれば十分です。

## 共通Validationとの関係

Pattern固有Validationの共通`validation.Validate`への統合は[`ISSUE-000010`](000010-application-pattern-validation-integration.md) / GitHub Issue #98で扱います。

PR #101の実Referenceは、Queryが保持するDatasourceを再利用できることまで確認しています。`ISSUE-000010`が完了した後は、この実Referenceを通常のWorkspace validationからも追加呼び出しなしで検査できる状態になります。

## 設計上の境界

- Pattern identityをfile pathから導出しない
- native SQLをPatternへコピーしない
- Queryが持つDatasourceを理由なく二重管理しない
- `browse`をWeb画面テンプレートとして定義しない
- PatternからPersistence MappingやDatabase FKを暗黙生成しない
- このReference一件を理由に5分類を永久的な標準語彙へ昇格させない

## 非目標

- 共通`validation.Validate`へのPattern Validation統合
- `record` / `reference` / `command` / `header-detail`の追加
- `OrderLine`の追加
- Web TargetやDriverの実装
- Application Pattern Catalogの標準化

## 完了条件

PR #101までに、次の1〜4は確認済みです。

1. `order_entry.recent_orders`が実Reference Patternとして読み込まれる
2. `sales.order`と既存native SQL Queryをidentityで参照し、内容を複製しない
3. Datasourceの不要な二重管理を避けられる
4. Patternを含む`reference/world.json`をRepositoryだけから再生成できる

このIssueを閉じるには、さらに次を確認します。

5. 実Referenceのファイル名を変えてもCanonical WorldとPattern Subject identityが変わらない
6. Source IndexのArtifact pathだけが新しいauthoring sourceへ追随する

このEvidenceが揃えば、`browse`の実Reference側は閉じたとみなし、`record` / `reference` / `command`へ進めます。

## 関連

- GitHub Issue #100
- GitHub Issue #67
- GitHub Issue #98
- PR #95
- PR #99
- PR #101
- ADR-0017
- ADR-0018
- `plan/APPLICATION_PATTERN_V1_DESIGN.md`

関連RQ: RQ-001, RQ-002, RQ-003, RQ-014, RQ-017, RQ-021
