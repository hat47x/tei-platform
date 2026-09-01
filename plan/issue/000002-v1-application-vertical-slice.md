# ISSUE-000002: v1へ向けたApplication vertical sliceを定義する

- Status: Proposed

## 背景

TEIは、Semantic、Database、Deployment、Plugin / Runtimeの各境界を個別にかなり深く検証してきました。

次の大きな不確実性は、それらを「一つの業務Applicationを設計し、Driverで検証し、第一TargetへProjectionする」という利用者視点の一本の流れへまとめられるかどうかです。

v1を機能一覧から逆算せず、実用的なvertical sliceから定義します。

## 関連するProduct Requirement

- RQ-005 ApplicationとDatabaseのN:M
- RQ-010 Editor
- RQ-012 Driver
- RQ-013 Target-neutral Presentation
- RQ-014 standard Application Pattern
- RQ-018 Command / Observation
- RQ-021 Reference / Conformance

## 検討するReference

業務システムとして十分に現実的でありながら、巨大になりすぎないReference Applicationを選びます。

最低限、次を含む候補を比較します。

- browse / search
- record edit
- reference lookup
- header-detail
- command / state transition
- validation
- authorization
- 複数Databaseまたは複数Datasource
- native SQL
- custom implementationを一箇所以上

## vertical sliceで確かめること

```text
Authoring / Editor Projection
        ↓
Canonical Meaning
        ↓
Validation / Driver
        ↓
Application Pattern / Presentation
        ↓
Database / Capability Binding
        ↓
第一Target
        ↓
Observation / Evidence
```

この流れを、一つのRepository内Referenceから追跡できる状態を目指します。

## 非目標

- 最初から全Targetを完成させること
- あらゆる業務機能を標準化すること
- production HAや大規模負荷を同時に完成させること
- Editorの見た目を先に作り込むこと

## 完了条件

v1へ進むために必要な最小Application vertical sliceが、業務シナリオ、必要Capability、Target、Database、Reference、Conformanceの単位まで具体化され、後続Issueへ分解できること。
