# 機能設計文書

機能ごとの設計文書を置きます。対象は、複数のIssueやPRにまたがる一つの機能について、意味モデル、authoring、Validation、実行境界、Conformanceをまとめて示す文書です。

- 何を行うかと完了条件: [`../issue/`](../issue/README.md)
- なぜその方針を採ったか: [`../adr/`](../adr/)
- 何から進め、何をEvidenceとするか: [`../README.md`](../README.md)

設計文書はIssueとADRの代わりではありません。作業の進行はIssue、採用した判断はADRが正本です。ここに置くのは、その二つだけでは分断される機能単位の設計本文です。小さい機能では新しい文書を作らず、Issue本文へ書きます。

## 一覧

- [`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md): Application Patternの設計・実装基線（[ISSUE-000005](../issue/000005-application-pattern-reference.md) / ADR-0017 / ADR-0018）
- [`LEGACY_AUTH_V1_DESIGN.md`](LEGACY_AUTH_V1_DESIGN.md): 既存認証仕様を保持してAuthentication / Authorization Capabilityへ接続する設計（[ISSUE-000003](../issue/000003-legacy-authentication-compatibility-reference.md) / ADR-0019）
- [`CAPABILITY_OUTCOME_V1_DESIGN.md`](CAPABILITY_OUTCOME_V1_DESIGN.md): CapabilityのBusiness Result / named OutcomeとExecution Errorを分離するv1設計（[ISSUE-000009](../issue/000009-capability-outcome-and-execution-error.md) / ADR-0020）

業務Lifecycleの要件詳細化は[ISSUE-000038](../issue/000038-business-lifecycle-semantics.md)本文にあります（ADR-0024 / ADR-0025、いずれも評価待ち）。
