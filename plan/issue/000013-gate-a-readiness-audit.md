# ISSUE-000013: Gate Aの到達性を横断監査する

- 状態: In Progress
- GitHub Issue: #124
- 前段: `ISSUE-000005` / GitHub Issue #67（完了）
- 後段: `ISSUE-000006` / GitHub Issue #69（Application Driver）
- 現在の残条件: #68
- Gate AでReadyになった関連Issue: #77（Capability Outcome） / #125（業務Validation） / #126（Capability Binding）
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

Gate A全体は引き続きBlockedです。ただし、独立した残条件は**#68 既存認証Compatibilityだけ**になりました。

### Readyになった領域

- Application Pattern 5種類
- Datasource境界
- Capability Binding #126 / PR #135
- 業務Validation #125 / PR #136 / #139 / #140
- Capabilityのnamed Outcome / Execution Error #77 / PR #132 / #141
- 既存単一output Capabilityとの互換性
- Command / Observation分離の基本境界

PR #141では`security.authenticate`の実Referenceを使い、`authenticated` / `denied`を正常なBusiness Resultとして扱い、Database / adapter障害や未定義OutcomeをExecution Errorとして分離できることを確認しました。

したがって#77はGate AでReadyです。第一Web TargetへのProjection等はGate A以後へDeferredします。

## 残る条件 — #68 既存認証Compatibility

`security.authenticate`の契約と結果境界は成立していますが、既存認証仕様をそのまま実装へ接続するEvidenceはまだ不足しています。

Gate Aへは少なくとも次を渡せる必要があります。

- authenticated subject
- Authentication Capability identity
- Authorization Capability identity
- Application上のaction / resource
- legacy field / functionからCapabilityへのMapping
- user ID、disabled、validity、failure count、lock、organization、role、system-specific rule等の既存条件
- secret materialを解決する境界
- named Outcome / Execution Errorの結果契約

既存schemaをTEI標準User schemaへ作り替えてからでなければ利用できない構造にはしません。

## Secretの扱い

`security.authenticate`にはcredentialを表す入力型がありますが、Canonicalに保存されるのはCapability契約と型の意味であり、実際のsecret値ではありません。

password、token、credential hash、接続資格情報等のsecret materialをCanonical World、Source Index、Durable artifact、Scenario log、通常Evidenceへ保存しません。

#68では、実行時に受け取ったcredential valueがEvidenceへ流れないこともConformance対象にします。

## 判定原則

Gate Aを「関連Issueがすべてcloseしたか」だけでは判定しません。

Driverへ渡すために必要な意味をRepository上のReference / Conformanceから再現できるかを基準にします。

Issue全体がopenでも、Gate Aに必要な限定範囲が十分なEvidenceで成立していれば、その項目はReadyと評価できます。

逆にIssueがcloseしていても、Driverが暗黙のGoコードやWeb設定から意味を補う必要がある場合はReadyとは判定しません。

## 残る作業

1. legacy認証fixtureを、既存仕様を崩さない形で用意する
2. `security.authenticate`をlegacy fixtureへ接続するadapter / Bindingを実証する
3. `authenticated` / `denied` / Execution Errorの既存境界を再利用する
4. secret materialがCanonical / Source Index / 通常Evidenceへ流れないことを確認する
5. `security.authorize`の最小契約を実例から作る
6. `order_entry`のbrowse / edit / submitへaction / resourceを接続する
7. Gate Aを再評価する
8. 必須項目がすべてReadyになった時点でDriver #69の開始可否を明示する
9. 判定を`V1_REFERENCE_APPLICATION_PLAN.md`等へ同期する

## 非目標

- Driver自体の主実装
- 第一Web Target
- production-ready IdP
- production transaction coordinator
- 5 Patternの標準Catalog昇格
- Gate A以後のOutcome Projection
- 汎用Validation rule engine
- 認証製品やlegacy schemaの標準化

## 完了条件

- Gate Aの入力項目ごとにEvidenceと不足を説明できる
- Application Pattern Stage完了とGate A全体完了を混同していない
- 未完了項目が既存Issueとして追跡されている
- Driver #69がWebやDriver専用設定から業務意味を補わなくてよい
- Driver #69を開始してよいかを明示的に判定できる

## 関連

- GitHub Issue #124: Gate A到達性監査
- GitHub Issue #68: 既存認証Compatibility
- GitHub Issue #69: Application Driver
- GitHub Issue #77: Capability Outcome / Execution Error（完了）
- GitHub Issue #125: 業務Validation（完了）
- GitHub Issue #126: Capability Binding（完了）
- GitHub PR #132 / #135 / #136 / #139 / #140 / #141 / #143
- ADR-0019 / ADR-0020
- RQ-005, RQ-008, RQ-009, RQ-012, RQ-014, RQ-017, RQ-018, RQ-019, RQ-021
