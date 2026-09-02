# ISSUE-000009: Capabilityの業務Outcomeと実行エラーを分離する

- 状態: In Progress
- GitHub Issue: #77
- 設計判断: ADR-0020

## 背景

既存認証Compatibilityを具体化する中で、Authenticationに固有ではないCapability実行上の不足が見つかりました。

Authenticationでは、少なくとも次を区別する必要があります。

```text
Authenticated(subject)
Denied(reason)
ExecutionError(error)
```

credentialの不一致やaccount lockは、認証処理を正常に実行した結果としての拒否です。一方、Database接続失敗やadapter異常は、認証判断そのものを完了できなかった状態です。

この違いをAuthentication固有の規約にはせず、Semantic Capability共通の境界として扱います。

ADR-0020では、業務上の複数の結末をSemantic Capabilityの**名前付きOutcome**として表し、Execution ErrorはInvocation境界へ残す方針を採用しました。

詳細な設計は[`CAPABILITY_OUTCOME_V1_DESIGN.md`](../CAPABILITY_OUTCOME_V1_DESIGN.md)に置きます。

## 関連するProduct Requirement

- RQ-001 業務上の意味を実装技術から独立して保持する
- RQ-008 Capability単位で実装を置き換える
- RQ-009 既存認証・認可仕様を保持する
- RQ-012 Driverで設計を実行・検証する
- RQ-018 CommandとObservationを分離する
- RQ-021 Reference / Conformanceで再現可能性を示す

## 現在地

PR #132で、名前付きOutcomeをSemantic Capabilityへ保持するCanonical基盤までは実装済みです。

現在は「名前付きOutcomeを表現できるか」を検討する段階ではありません。Gate Aへ進むために、実際のCapability呼び出しでBusiness ResultとExecution Errorを分離できるかを実証する段階です。

### 実装済み

- `semantic.Capability`が名前付きOutcomeを保持できる
- OutcomeはCapability内のlocal nameとpayload型を持つ
- `outcome <name> <payload-type>`をParserから読み込める
- 既存`output`と名前付きOutcomeの併用を拒否する
- Outcome名の重複を拒否する
- payload型を既存Semantic value typeとして検証する
- payloadを必須にする
- Outcomeの宣言順をCanonicalな意味にしない
- 既存の単一`output` Capabilityを変更しない

`sales.order.submit`の`output sales.order`は従来どおり有効です。名前付きOutcomeを導入したことを理由に、暗黙の`success` Outcomeへ書き換えません。

### Gate Aでまだ必要なEvidence

- `security.authenticate`等の実Reference
- `authenticated` / `denied`を正常なBusiness Resultとして扱うInvocation境界
- Binding / adapter / Database障害をExecution Errorとして扱う境界
- 未定義Outcomeやpayload不一致をCapability契約違反として扱う境界
- DriverへBusiness Result / Execution Errorの区別を渡せる最小契約
- secret materialをCanonical / Source Index / Scenario Evidenceへ混入させないこと

第一Web TargetへのProjectionや、Authentication以外のCapabilityでの広い圧力試験は、Gate A通過後へDeferredできます。

## 採用した境界

### 業務上の分岐は名前付きOutcomeにする

v1では一般的なsum typeをSemantic型体系へ追加せず、Capabilityに直接ぶら下がる名前付きOutcomeを利用します。

概念例:

```tei
capability security.authenticate {
    input login std.identifier
    outcome authenticated security.authenticated_subject
    outcome denied security.authentication_denial
}
```

Outcomeのidentityをfile pathや宣言順には依存させません。

### payload型はv1では必須にする

v1では次の構文を使います。

```text
outcome <name> <payload-type>
```

payloadなしのOutcome、optional type、`std.unit`等は、具体的な必要性が得られてから別途検討します。

### 既存`output`を維持する

一つの通常resultを返す既存Capabilityは、従来どおり`output`を利用できます。

```tei
capability sales.order.submit {
    input order sales.order
    output sales.order
}
```

`output`と名前付き`outcome`は同一Capabilityで併用しません。

### Execution ErrorはInvocation境界に残す

次はSemantic Outcomeではありません。

- Database接続失敗
- process起動失敗
- Resource Lease取得失敗
- Binding解決失敗
- adapter異常
- response identity不一致
- 未定義Outcomeの返却
- Outcome payloadの契約不一致

業務上の`denied`や`rejected`をRuntime errorへ変換せず、逆に実行障害を業務Outcomeへ押し込みません。

## Gate Aで使うAuthentication Reference

最初の実Referenceでは、`security.authenticate`を使って少なくとも次を区別します。

1. 正しいcredential → `authenticated`
2. credential不一致 → `denied`
3. account disabled / locked → `denied`
4. Database / adapter障害 → Execution Error

2と3はScenarioの障害ではありません。Capabilityを正常に実行した結果としてのBusiness Resultです。

Outcome payloadやScenario Evidenceへpassword平文、token等のsecret materialを保存しません。

## Invocation境界で確認すること

Capability invocationは、概念上次の二経路を持ちます。

```text
Capability invocation
  ├─ Business Result
  │   ├─ Output value
  │   └─ Named Outcome(name, payload)
  └─ Execution Error
```

実装方式がCustom Goでもprocess Pluginでも、Semantic上のBusiness Resultを実装固有のerror規約へ吸収しません。

また、Invocation側がSemantic契約にないOutcomeを返した場合は、業務上の新しい結末として自動受理せず契約違反として扱います。

## Authentication以外の圧力試験

Authenticationだけに都合のよい抽象になっていないことは、Gate A以後に`sales.order.submit`等の別Capabilityでも確認します。

ただし、現行`sales.order.submit`の`output sales.order`を名前付きOutcomeへ置き換えること自体を目的にはしません。

ValidationやAuthorizationで事前に判定できる不成立を、Command Outcomeへ重複させないことも守ります。

## Driverへの引き継ぎ

Driverでは少なくとも次を別のEvidenceとして扱います。

```text
Validation result
Authorization result
Business Result
  - output / named outcome
Execution Error
Command acknowledgement
Observed result
```

期待した`denied`や`rejected`が返るScenarioは正常に完了できます。

Execution Errorは、Capability invocationを完了できなかった状態として扱います。

副作用を持つCapabilityでExecution Errorが返った場合は、外部状態を確認せず自動retryしません。必要な場合はRQ-018 / RQ-020のObservation、unknown、takeover等の既存境界を利用します。

## 第一Web Targetへの引き継ぎ

Web TargetはBusiness ResultやExecution ErrorをHTTP status、message、画面状態等へProjectionできます。

ただし、Web固有表現をSemantic Outcomeの定義にはしません。

また、次を一つの「エラー」へ無条件に統合しません。

- Validationの不成立
- Authorization denial
- CapabilityのBusiness Outcome
- Execution Error
- Observationで確認した状態
- developer向けDiagnostic

## 今後の実装順

### Gate A-1 — Authentication用Semantic Reference

- `security.authenticate`をReferenceへ追加する
- `authenticated` / `denied` Outcomeを実Referenceから読み込めるようにする
- payloadに必要なSemantic typeを最小限追加する
- secret materialをCanonicalへ保存しない

### Gate A-2 — Legacy認証Compatibilityとの接続

- #68のlegacy認証fixtureへ接続する
- credential mismatch / disabled / lockedを`denied`として再現する
- Database / adapter障害を別のExecution Error経路で再現する

### Gate A-3 — Invocation境界

- Output value / Named OutcomeをBusiness Resultとして受け渡せる最小表現を定める
- 未定義Outcomeやpayload不一致を契約違反として扱う
- Runtime errorをBusiness Outcomeへ変換しない
- Business ResultとExecution Errorを同時成立させない

### Gate A-4 — Gate A再監査

- Authentication Referenceから結果境界を再現できることを確認する
- Driver #69が独自のOutcome体系を必要としないことを確認する
- Gate Aに必要な範囲がReadyなら、#77全体の後続作業をDriver / Target段階へ分離する

### Gate A以後

- Driver Scenarioでexpected Outcomeを扱う
- Authentication以外のCapabilityで圧力試験する
- 第一Web TargetへProjectionする

## Conformance

### 既に成立しているSemantic / Parser Evidence

- 既存`output` Capabilityを従来どおりparseできる
- named Outcomeをparseできる
- `output` + `outcome`を拒否する
- 重複Outcome名を拒否する
- 不正payload型を拒否する
- Outcome宣言順だけでnormalized meaningが変わらない

### Gate Aで追加するInvocation Evidence

- 定義済みOutcomeを正常なBusiness Resultとして返せる
- 未定義Outcomeを契約違反として扱う
- payload不一致をExecution Errorとして扱う
- Business ResultとExecution Errorを同時に成立させない
- Runtime errorをSemantic denialへ変換しない

### Gate Aで追加するReference Evidence

- valid credential → `authenticated`
- credential mismatch → `denied`
- locked / disabled → `denied`
- adapter / Database failure → Execution Error
- expected denialを正常なScenarioとして扱える
- secretがCanonical World / Source Index / Scenario Evidenceへ混入しない

## 非目標

- 一般的なvariant / union type
- optional type
- `std.unit`
- Outcome payloadの省略構文
- Outcomeのglobal Canonical ID
- 一般的なExecution Error taxonomy
- workflow engine全体のresult model
- HTTP / gRPC statusとの標準対応
- RQ-018のObservationをBusiness Resultへ統合すること

## 完了条件

Issue全体としては、次をRepository上のReference / Conformanceから説明できることを完了条件とします。

- 業務上の複数の結末をSemantic Capabilityの名前付きOutcomeとして表せる
- 既存の単一`output` Capabilityを不用意に壊さない
- 業務OutcomeとExecution Errorを区別できる
- Authenticationのdenialを正常な業務結果として扱える
- Authenticationの実行障害をdenialへ変換しない
- DriverからOutcomeをScenario Evidenceとして扱える
- Authentication以外のCapabilityにも同じ境界を無理なく適用できる
- Target固有表現をSemantic Outcomeへ逆流させない
- Command / Observation境界を維持できる

Gate Aでは、この完了条件のうちDriver開始に必要な範囲だけを先に評価します。Issue #77全体を第一Web Targetまで完了させることはGate Aの条件にしません。

## 関連文書

- `plan/adr/0020-capability-business-outcome-boundary.md`
- `plan/CAPABILITY_OUTCOME_V1_DESIGN.md`
- `plan/LEGACY_AUTH_V1_DESIGN.md`
- `plan/GATE_A_READINESS_AUDIT.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000007-first-web-target-projection.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md` RQ-001、RQ-008、RQ-009、RQ-012、RQ-018、RQ-021
- GitHub Issue #68 / #69 / #77 / #124
- PR #132
