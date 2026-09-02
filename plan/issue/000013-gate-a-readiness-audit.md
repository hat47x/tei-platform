# ISSUE-000013: Gate Aの到達性を横断監査する

- 状態: In Progress
- GitHub Issue: #124
- 前段: `ISSUE-000005` / GitHub Issue #67（完了）
- 後段: `ISSUE-000006` / GitHub Issue #69（Application Driver）
- 現在の残条件: #68 / #77 / #125
- 完了した関連Issue: #126（Capability Binding）
- 監査結果: [`GATE_A_READINESS_AUDIT.md`](../GATE_A_READINESS_AUDIT.md)

## 目的

Application Pattern候補5種類がすべて実Referenceまで成立したため、Patternを増やす段階から、Driverへ渡すApplication上の意味を横断的に確認する段階へ移っています。

Gate Aでは、Web画面やDriver専用設定を見なければ分からない情報を残さないことが重要です。

Driver #69が、Repository上のCanonicalな意味、Contract、Bindingから少なくとも次を解決できるかを監査します。

- browse対象
- record対象
- reference対象RelationとDatasource
- header-detail対象Relation
- command対象Capability
- 業務Validation
- Authentication / Authorization
- Datasource read / write境界
- Capability Binding
- Business Result / Execution Error

## 現在の判定

Application Pattern側はReadyです。

Capability BindingもIssue #126 / PR #135でReadyになりました。

また、Capability OutcomeはPR #132で名前付きOutcomeのCanonical基盤まで実装され、設計だけの状態から前進しています。ただしGate Aに必要なReference / Invocation境界はまだ未完了です。

したがって、Gate A全体は引き続きBlockedです。現在の独立した残条件は次の3つです。

- #68: 既存認証Compatibility
- #77: named Outcomeを実際のCapability呼び出しでBusiness Resultとして扱い、Execution Errorと分離するEvidence
- #125: 業務Validationの正本と最小Reference

詳細は`GATE_A_READINESS_AUDIT.md`を正とします。

## 完了した監査項目

### Application Pattern

5種類のPatternは実Reference、Source Index、共通Validation、Golden再現性まで成立しています。

### Datasource境界

`sales` read-write / `master` readのlogical datasource境界をApplication定義から追跡できます。

### Capability Binding

Issue #126 / PR #135で、`sales.order.submit`をApplication + Capabilityから明示的に解決できるBinding Referenceを実証しました。

Driverがfile path、Go hook、列挙順からimplementationを推測する必要はありません。

Binding未定義または複数定義は、業務結果として暗黙処理せずerrorにできます。

## 判定原則

Gate Aを「関連Issueがすべてcloseしたか」だけでは判定しません。

Driverへ渡すために必要な意味をRepository上で再現できるかを基準にします。

そのため、#77の第一Web TargetへのProjectionなど、Gate A以後でよい作業はblockerにはしません。

逆に、Issueがcloseしていても、Driverが暗黙のGoコードやWeb設定から意味を補う必要がある場合はReadyとは判定しません。

評価には次の区分を使います。

- **Ready**: Gate Aに必要な範囲をReference / Conformanceから再現できる
- **Partial**: 境界の一部は成立しているが、Driverへ渡す情報が不足している
- **Blocked**: Driverへ進む前に独立した実証が必要である
- **Deferred**: 必要な作業だがGate A以後でよい

## 残る作業

1. #68でAuthentication / Authorization Referenceを成立させる
2. #77でnamed OutcomeのReferenceとInvocation上のBusiness Result / Execution Error分離を確認する
3. #125で業務Validationの正本とidentityを実証する
4. Gate Aの各項目を再評価する
5. 必須項目がすべてReadyになった時点でDriver #69の開始可否を明示する
6. 判定を`V1_REFERENCE_APPLICATION_PLAN.md`と`plan/README.md`へ同期する

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

- GitHub Issue #124: Gate A到達性監査
- GitHub Issue #67: Application Pattern（完了）
- GitHub Issue #68: 既存認証Compatibility
- GitHub Issue #69: Application Driver
- GitHub Issue #77: Capability Outcome / Execution Error
- GitHub Issue #125: 業務Validation
- GitHub Issue #126: Capability Binding（完了）
- GitHub PR #132 / #135
- ADR-0019 / ADR-0020
- RQ-005, RQ-008, RQ-009, RQ-012, RQ-014, RQ-017, RQ-018, RQ-019, RQ-021
