# ISSUE-000013: Gate Aの到達性を横断監査する

- 状態: In Progress
- GitHub Issue: #124
- 前段: `ISSUE-000005` / GitHub Issue #67（完了）
- 後段: `ISSUE-000006` / GitHub Issue #69（Application Driver）
- 現在の残条件: #68 / #77
- Gate AでReadyになった関連Issue: #125（業務Validation） / #126（Capability Binding）
- 監査結果: [`GATE_A_READINESS_AUDIT.md`](../GATE_A_READINESS_AUDIT.md)

## 目的

Application Pattern候補5種類がすべて実Referenceまで成立したため、Patternを増やす段階から、Driverへ渡すApplication上の意味を横断的に確認する段階へ移っています。

Gate Aでは、Web画面やDriver専用設定を見なければ分からない情報を残さないことが重要です。

Driver #69が、Repository上のCanonicalな意味、Contract、Bindingから少なくとも次を解決できるかを監査します。

- browse / record / reference / header-detail / commandの対象
- Datasource read / write境界
- 業務Validation
- Authentication / Authorization
- Capability Binding
- Business Result / Execution Error
- Command / Observationの境界

## 現在の判定

Gate A全体は引き続きBlockedです。ただし、残条件は2つまで減りました。

- #68: 既存認証Compatibility
- #77: named Outcomeを実際のCapability呼び出しでBusiness Resultとして扱い、Execution Errorと分離するEvidence

業務Validation #125はPR #136、#139、#140でGate Aに必要な範囲を完了しました。

- Semantic ValidationのCanonical契約
- `sales` Namespaceの実Reference
- Validation identityからauthoring sourceへの追跡
- `positive` / `relation-unique`の最小Evaluator
- `Satisfied / NotSatisfied`と評価errorの分離

Capability Binding #126もPR #135でReadyです。

詳細な項目別判定は`GATE_A_READINESS_AUDIT.md`を正とします。

## Readyになった主な項目

### Application Pattern

`order_entry`では、`browse`、`record`、`reference`、`command`、`header-detail`の5種類が実Reference、Source Index、共通Validation、Golden再現性まで成立しています。

### Datasource境界

`sales` read-write / `master` readのlogical datasource境界をApplication定義から追跡できます。

### Capability Binding

Issue #126 / PR #135で、`sales.order.submit`をApplication + Capabilityから明示的に解決できるBinding Referenceを実証しました。

Driverがfile path、Go hook、列挙順からimplementationを推測する必要はありません。

### 業務Validation

Issue #125 / PR #136 / #139 / #140で、業務ruleの正本、実Reference、評価境界が成立しました。

Driverは`quantity > 0`や明細番号の一意性を専用Goコードへ書き直さず、Canonical ValidationとEvaluatorを利用できます。

評価不能は`NotSatisfied`へ畳まず、評価errorとして区別できます。

## 残る二つの条件

### #68 既存認証Compatibility

既存schemaや既存ruleをTEI標準User schemaへ先に作り替えず、Authentication / Authorization Capabilityへ接続できるReferenceが必要です。

Gate Aへは少なくとも、authenticated subject、Capability identity、Application上のaction / resource、legacy mapping、secret解決境界を渡せる必要があります。

### #77 Capability Outcome / Execution Error

PR #132で名前付きOutcomeのCanonical基盤までは成立しています。

次は`security.authenticate`等の実Referenceを通じ、`authenticated` / `denied`を正常なBusiness Resultとして扱い、Databaseやadapterの障害をExecution Errorとして分離できることを確認します。

#68と#77はAuthentication Referenceで重なるため、別々の大きな仕組みを先に作らず、一つの実例から両方のEvidenceを増やします。

## 判定原則

Gate Aを「関連Issueがすべてcloseしたか」だけでは判定しません。

Driverへ渡すために必要な意味をRepository上のReference / Conformanceから再現できるかを基準にします。

そのため、#77の第一Web TargetへのProjectionなど、Gate A以後でよい作業はblockerにはしません。

逆にIssueがcloseしていても、Driverが暗黙のGoコードやWeb設定から意味を補う必要がある場合はReadyとは判定しません。

## 残る作業

1. #68 / #77の共通入口として`security.authenticate`の実Referenceを成立させる
2. named Outcomeを正常なBusiness Resultとして扱うInvocation境界を実証する
3. expected denialとExecution Errorを分離する
4. legacy認証fixtureとCapabilityを接続する
5. Authorizationを`order_entry`のaction / resourceへ接続する
6. Gate Aを再評価する
7. 必須項目がすべてReadyになった時点でDriver #69の開始可否を明示する
8. 判定を`V1_REFERENCE_APPLICATION_PLAN.md`等へ同期する

## 非目標

- Driver自体の主実装
- 第一Web Target
- production-ready IdP
- production transaction coordinator
- 5 Patternの標準Catalog昇格
- Gate A以後のOutcome Projection
- 汎用Validation rule engine

## 完了条件

- Gate Aの入力項目ごとにEvidenceと不足を説明できる
- Application Pattern Stage完了とGate A全体完了を混同していない
- 未完了項目が既存Issueとして追跡されている
- Driver #69がWebやDriver専用設定から業務意味を補わなくてよい
- Driver #69を開始してよいかを明示的に判定できる

## 関連

- GitHub Issue #124: Gate A到達性監査
- GitHub Issue #67: Application Pattern（完了）
- GitHub Issue #68: 既存認証Compatibility
- GitHub Issue #69: Application Driver
- GitHub Issue #77: Capability Outcome / Execution Error
- GitHub Issue #125: 業務Validation（完了）
- GitHub Issue #126: Capability Binding（完了）
- GitHub PR #132 / #135 / #136 / #139 / #140
- ADR-0019 / ADR-0020
- RQ-005, RQ-008, RQ-009, RQ-012, RQ-014, RQ-017, RQ-018, RQ-019, RQ-021
