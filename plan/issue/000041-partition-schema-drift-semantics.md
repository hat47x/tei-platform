# ISSUE-000041: Partitionの観測・所有境界とSchema Driftの扱いを定める

- 状態: Proposed
- 登録日: 2026-09-04
- 統合元: 既存GitHub Issue #43、Master Plan Workstream Dのpartition semantics
- 関連RQ: RQ-006 / RQ-007 / RQ-021
- 着手条件: 設計と小さい再現例の整理から開始可能。現行Gate Bの追加条件にはしない

## 目的

partitioned parentとchild partitionを独立した無関係なtableとして扱うことで、正当なchildを不要な`drop-table`候補にする誤判定を防ぎます。同時に、childを観測から無条件に除外して、管理すべき物理構造や未知の状態を失うことも避けます。

本Issueは新しい要求の追加ではなく、計画Issueへ未登録だった既存のpartition設計課題を、所有境界の判断と最小Conformanceへまとめたものです。

## 現在の根拠

[PostgreSQL catalog reader](../../internal/database/providers/postgres/sql_catalog.go)は`relkind`が`r` / `p` / `v` / `m`のrelationを観測します。[Physical Schema](../../internal/database/physical/model.go)にはpartition hierarchyの専用表現がなく、[Schema Diff](../../internal/database/schemadiff/diff.go)はActualにだけ存在するtableを`DropTable`へ分類します。

Desiredがparentだけを表し、Actualにchildが現れた場合の意味を明示する必要があります。実際の危険なDDL適用を試すのではなく、まず最小snapshotと差分結果から誤判定を再現します。

## この単位で決めること

次の候補を、実際の小さいparent / child構成で比較します。

1. Physical Schemaへparent-child構造を明示する。
2. parentのProvider固有realization metadataに保持する。
3. 当面は明示された管理範囲からchildを外し、別の観測情報と比較未確定を保持する。

比較時にはparent / childのidentity、ownership、partition key / bound、Desiredへのauthoringの要否、childのcreate / drop判断、情報不足時のUnresolvedを扱います。PostgreSQL catalogのfieldをそのままprovider-neutralな意味へ昇格させません。

## 成果物と完了条件

- [ ] parentのみDesiredに存在する例、parent / childを明示管理する例、通常の独立table、partition情報不足の4ケースを定義した
- [ ] 上記ケースから、childを無条件にdrop候補へしないことと、通常のtable差分を隠さないことを比較できる
- [ ] 比較対象・所有境界・未知の構造を区別し、無条件除外や推測による一致判定をしない
- [ ] 選択肢、採用理由、非採用理由、Provider/Coreの責務をADRに記録した
- [ ] PostgreSQL normalizationと共通Schema Diffに必要な変更範囲、期待するDiagnostic / Unresolved、Conformance項目を明示した
- [ ] 判断に使用したfixture / snapshot / revisionと、実Provider観測を行ったかを残した

完了は所有境界と後続実装の契約を確定した時点です。実装まで完了したとは扱わず、後続は採用案の責務が判明してから必要最小限のIssueへ分解します。

## 他Issueとの境界

[ISSUE-000039](000039-business-change-rehearsal.md)は業務Meaningの変更とApplicationへの影響を扱います。本IssueはDatabase Physical Schemaの観測・比較です。両者のDiffを一つの巨大な解析器へ統合しません。

[ISSUE-000031](000031-gate-b-postgres-header-detail-evidence.md)の実PostgreSQL Evidenceはheader-detail用であり、partitionの所有・差分判断を証明するものではありません。

## 非目標

- partition用DDL生成、repartitioning、online migrationの実行
- Oracle等の全Providerへ一度に実装すること
- 未知のchildを無条件に削除可能または無視可能と判断すること
- 実外部DBでの破壊的な検証

## 関連

- [Master Plan](../MASTER_PLAN.md) Workstream D
- [Schema Diffの比較未確定Conformance](../../docs/conformance/schema-diff-unresolved-m0.31.md)
- 既存GitHub Issue #43 / #38（履歴参照。進行の管理先は本Issue）
