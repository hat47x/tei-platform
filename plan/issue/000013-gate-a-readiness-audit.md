# ISSUE-000013: Gate Aの到達性を横断監査する

- 状態: Completed
- GitHub Issue: #124
- 前段: `ISSUE-000005` / GitHub Issue #67（完了）
- 後段: `ISSUE-000006` / GitHub Issue #69（Application Driver）
- Gate Aで完了した関連Issue: #68 / #77 / #125 / #126
- 監査結果: [`GATE_A_READINESS_AUDIT.md`](../GATE_A_READINESS_AUDIT.md)

## 目的

Application Pattern候補5種類が実Referenceまで成立した後、Driverへ渡すApplication上の意味を横断的に監査しました。

Gate Aでは、Web画面やDriver専用設定を見なければ分からない情報を残さないことを重視します。

Driver #69がRepository上のCanonicalな意味、Contract、Binding、Referenceから、少なくとも次を解決できるかを確認しました。

- browse / record / reference / header-detail / commandの対象
- Datasource read / write境界
- 業務Validation
- Authentication / Authorization
- Capability Binding
- Business Result / Execution Error
- Command / Observationの境界

## 最終判定

**Gate AはReadyです。**

Application Driver #69の主実装へ進んでよいと判断しました。

この判定は、関連Issueのclose状態だけではなく、Driverに必要な意味をRepository内のReference / Conformanceから再現できることに基づきます。

## Readyになった領域

### Application Pattern

`browse` / `record` / `reference` / `command` / `header-detail`の5種類が、同じ`order_entry`で実Referenceとして成立しています。

Pattern identity、Source Index、Diagnostic projection、Golden再現性も確認済みです。

### Datasource

`order_entry`は次の境界をApplication定義から解決できます。

- `sales`: read-write
- `master`: read

PatternやDriver専用設定へ同じaccess情報を複製しません。

### Capability Binding

Application + Capabilityから明示的にBindingを解決できます。

- `order_entry/sales.order.submit`
- `order_entry/security.authenticate`
- `order_entry/security.authorize`

0件は未定義、複数件はambiguousとしてerrorになります。

### 業務Validation

実Referenceから必要になった`positive`と`relation-unique`のCanonical ruleを同じEvaluator境界へ渡せます。

Validation不成立と、評価そのものを完了できないerrorを区別できます。

### Capabilityの結果境界

ADR-0020 / Issue #77で、正常なBusiness ResultとExecution Errorを分離しました。

- 既存単一`output`
- named Outcome
- Execution Error

を同じInvocation境界から扱えます。

### Authentication / Authorization

Issue #68で既存認証Compatibilityを完了しました。

`security.authenticate`では、既存user recordを読む方式と、既存認証関数を呼ぶ方式の双方が同じCanonical契約を満たします。

PR #151では、`order_entry/security.authenticate`という同じBinding identityを維持したまま実現方式を交換できることを確認しました。

`security.authorize`では、既存Application Pattern identityをaction / resourceへ接続し、legacy organization / role / ruleを実装側で評価できます。

ApplicationやDriverへlegacy schemaやroleを公開して認可ruleを再実装させません。

### Secret

password、token、credential hash等のsecret materialをCanonical World、Source Index、Durable artifact、Git管理するReference fixture、通常Evidenceへ保存しません。

提示credentialは実行時だけ認証実装へ渡します。

## Driver #69へ渡せる情報

Gate Aの結果、DriverはRepositoryから次を解決できます。

- Application Pattern identity
- Query identity
- Relation identity
- logical datasourceとaccess
- Validation identity
- Authentication / Authorization Capability identity
- authenticated subject identity
- Authorization action / resource
- Capability Binding
- Business Result / Execution Error契約
- Command / Observation境界

Driver専用の第二の正本を作る必要はありません。

## Deferred

次は重要ですが、Gate A以後で扱います。

- WebでのOutcome表示
- HTTP status / UI messageへのProjection
- production-ready IdP / Policy Engine
- Target-neutral Presentationの最終抽象
- Patternの共有Catalog化
- `OrderLine` Persistence / FK一般化
- 汎用Validation式言語
- Capability Payloadの汎用Object model
- environment / release単位のimplementation選択

## 完了条件の確認

- Gate Aの入力項目ごとにEvidenceを説明できる — 完了
- Stage 1完了とGate A完了を混同しない — 完了
- Gate Aに必要な独立blockerをすべて解消した — 完了
- DriverがWebや専用設定から業務意味を補わなくてよい — 完了
- Driver #69を開始してよいか明示的に判定できる — **Ready**
- 判定をRepository内の監査正本とv1計画へ同期する — この変更で完了

## 次の作業

`ISSUE-000006` / GitHub Issue #69のApplication Driverへ進みます。

Gate Aで成立したResolver、Evaluator、Capability Invocation境界を組み合わせ、UIなしで`order_entry`の主要シナリオを実行します。

## 関連

- GitHub Issue #124: Gate A到達性監査
- GitHub Issue #68: 既存認証Compatibility（完了）
- GitHub Issue #69: Application Driver
- GitHub Issue #77: Capability Outcome / Execution Error（完了）
- GitHub Issue #125: 業務Validation（完了）
- GitHub Issue #126: Capability Binding（完了）
- GitHub PR #135 / #136 / #139 / #140 / #141 / #145 / #148 / #149 / #151
- ADR-0019 / ADR-0020
- RQ-005, RQ-008, RQ-009, RQ-012, RQ-014, RQ-017, RQ-018, RQ-019, RQ-021
