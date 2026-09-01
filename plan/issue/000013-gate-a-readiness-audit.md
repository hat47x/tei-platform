# ISSUE-000013: Gate A readinessを横断監査する

- 状態: In Progress
- GitHub Issue: #124
- 前段: `ISSUE-000005` / GitHub Issue #67（完了）
- 後段: `ISSUE-000006` / GitHub Issue #69（Application Driver）
- 関連Issue: #68 / #77 / #125 / #126
- 監査結果: [`GATE_A_READINESS_AUDIT.md`](../GATE_A_READINESS_AUDIT.md)

## 目的

Application Pattern候補5種類がすべて実Referenceまで成立したため、Patternを増やす作業から、Driverへ渡すApplication上の意味を横断的に確認する段階へ移ります。

Gate Aでは、Web画面やDriver専用設定を見なければ分からない情報を残さないことが重要です。

Driver #69が、Repository上のCanonicalな意味、Contract、Bindingから次を解決できるかを監査します。

- browse対象
- record対象
- reference対象RelationとDatasource
- header-detail対象Relation
- command対象Capability
- 業務Validation
- Authentication / Authorization
- Datasource read / write境界
- Capability Binding identity
- Business Result / Execution Error

## 現在の判定

Application Pattern側はReadyです。

一方、Gate A全体はまだReadyではありません。現在の主な残条件は次です。

- #68: 既存認証Compatibility
- #77: named OutcomeとExecution Errorの実装Evidence
- #125: 業務Validationの正本と最小Reference
- #126: `sales.order.submit` Capability Binding identity

詳細は`GATE_A_READINESS_AUDIT.md`を正とします。

## 判定原則

Gate Aを「関連Issueがすべてcloseしたか」だけでは判定しません。

Driverへ渡すために必要な意味がRepository上で再現できるかを基準にします。

そのため、#77の第一Web TargetへのProjectionなどGate A以後でよい作業は、Gate Aのblockerにはしません。

逆に、IssueがcloseしていてもDriverが暗黙のGoコードやWeb設定から意味を補う必要がある場合はReadyとは判定しません。

評価には次の区分を使います。

- **Ready**: Gate Aに必要な範囲をReference / Conformanceから再現できる
- **Partial**: 境界の一部は成立しているが、Driverへ渡す情報が不足している
- **Blocked**: Driverへ進む前に独立した実証が必要である
- **Deferred**: 必要な作業だがGate A以後でよい

## 作業

1. 5 PatternのEvidenceを一つの表へ統合する
2. Datasource read / write境界を確認する
3. #68からGate Aに必要なAuthentication / Authorization情報を抽出する
4. #77からGate Aに必要なOutcome / Execution Error範囲を抽出する
5. 業務Validationの正本を#125で実証する
6. `sales.order.submit`のBinding identityを#126で実証する
7. Gate A各項目を再評価する
8. 必須項目がすべてReadyになった時点でDriver #69の開始可否を明示する
9. 判定を`V1_REFERENCE_APPLICATION_PLAN.md`と`plan/README.md`へ同期する

## 非目標

- Driver自体の実装
- 第一Web Target
- production-ready IdP
- production transaction coordinator
- 5 Patternの標準Catalog昇格
- Gate A以後のOutcome Projection

## 完了条件

- Gate Aの入力項目ごとにEvidenceと不足を説明できる
- Application Pattern Stage完了とGate A全体完了を混同していない
- 未完了項目が既存Issueまたは独立Issueとして追跡されている
- Driver #69がWebやDriver専用設定から業務意味を補わなくてよい
- Driver #69を開始してよいかを明示的に判定できる

## 関連

- GitHub Issue #124: Gate A readiness監査
- GitHub Issue #67: Application Pattern（完了）
- GitHub Issue #68: 既存認証Compatibility
- GitHub Issue #69: Application Driver
- GitHub Issue #77: Capability Outcome / Execution Error
- GitHub Issue #125: 業務Validation
- GitHub Issue #126: Capability Binding
- ADR-0019 / ADR-0020
- RQ-005, RQ-008, RQ-009, RQ-012, RQ-014, RQ-017, RQ-018, RQ-019, RQ-021
