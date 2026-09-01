# ISSUE-000009: Capabilityの業務Outcomeと実行エラーを分離する

- 状態: 着手可能
- GitHub Issue: #77
- 設計判断: ADR-0020

## 背景

既存認証Compatibilityを具体化する中で、Authenticationに固有ではないCapability実行上の不足が見つかりました。

現行Semantic Capabilityは一つの`output`型を持ちます。一方、実際の業務処理では、処理を正常に実行した結果として複数の業務上の結末があり得ます。

Authenticationであれば、次の違いです。

```text
Authenticated(subject)
Denied(reason)
ExecutionError(error)
```

credentialの不一致やaccount lockは、認証処理を正常に実行した結果としての拒否です。Database接続失敗やadapter異常は、認証判断そのものを完了できなかった状態です。

この境界はAuthenticationだけでなく、受注確定、審査、在庫引当、外部申請等のSemantic Capabilityにも現れます。

ADR-0020では、業務上の分岐をSemantic Capabilityの**名前付きOutcome**として表し、Execution ErrorはInvocation境界へ残す方針を採用しました。

詳細な実装前設計は[`CAPABILITY_OUTCOME_V1_DESIGN.md`](../CAPABILITY_OUTCOME_V1_DESIGN.md)に置きます。

## 関連するProduct Requirement

- RQ-001 業務上の意味を実装技術から独立して保持する
- RQ-008 Capability単位で実装を置き換える
- RQ-009 既存認証・認可仕様を保持する
- RQ-012 Driverで設計を実行・検証する
- RQ-018 CommandとObservationを分離する
- RQ-021 Reference / Conformanceで再現可能性を示す

## 採用した境界

### 業務上の分岐は名前付きOutcomeにする

v1では一般的なsum typeをSemantic型体系へ追加せず、Capabilityに直接ぶら下がる名前付きOutcomeを導入します。

概念例:

```text
capability security.authenticate {
  input loginId std.identifier
  input credential security.credential

  outcome authenticated security.authenticated_subject
  outcome denied security.authentication_denial
}
```

各OutcomeはCapability内で一意なlocal nameとpayload型を持ちます。

Outcomeのidentityをfile pathや宣言順へ依存させません。

### payload型はv1では必須にする

現在のParserは改行を構文上の意味境界として利用していません。また、現行`std.*`には`unit`型がありません。

そのため、v1では次のようなpayload省略構文を導入しません。

```text
outcome accepted
```

各Outcomeに明示的なpayload型を要求し、必要性が確認できてから`std.unit`やoptionalな構文を別途検討します。

### 既存`output`を維持する

一つの通常resultだけを返す既存Capabilityは、従来どおり`output`を利用できます。

```text
capability sales.order.submit {
  input order sales.order
  output sales.order
}
```

`output`と名前付き`outcome`を同一Capabilityで併用することは認めません。

また、既存`output`へ暗黙の`success`や`completed`というOutcome名を付与しません。

### Execution ErrorはInvocation境界に残す

次はSemantic Outcomeではありません。

- Database接続失敗
- process起動失敗
- Resource Lease取得失敗
- adapter異常
- response identity不一致
- 未定義Outcomeを返す等のCapability契約違反

Go境界では通常の`error`を利用できます。Plugin Runtimeでは`ResponseError`またはGo `error`をExecution Errorとして扱います。

業務上の`denied`や`rejected`を`ResponseError`へ変換しません。

## 現行実装との接続

### Semantic model

`semantic.Capability`へ`Outcomes`を追加し、Outcomeは概念上次を持ちます。

```text
Outcome
- Name
- Payload type
```

既存`Output`は残します。

### Parser

Capability本文へ次を追加します。

```text
outcome <local-name> <payload-type>
```

Parser / Validationでは少なくとも次を拒否します。

- `output`と`outcome`の併用
- Outcome名の重複
- 不正なlocal name
- 解決できないpayload型

### Plugin Runtime

現行Plugin Runtimeには`ok / error`とGo `error`が既にあります。

Semantic Capabilityをprocess Plugin等で実現する場合は、概念上次へMappingします。

```text
ResponseOK + Business Result payload
    → Semantic Output / Outcome

ResponseError / Go error
    → Execution Error
```

credential不一致等の正常な拒否は`ResponseOK`側で`denied` Outcomeとして返します。

## Authentication Reference

最初の具体例として`security.authenticate`を利用します。

少なくとも次をReferenceから区別できるようにします。

1. 正しいcredential → `authenticated`
2. credential不一致 → `denied`
3. account disabled / locked → `denied`
4. Database / adapter障害 → Execution Error

2と3はDriver Scenarioの実行障害ではありません。

Outcome payloadやScenario Evidenceへpassword平文、token等のsecretを保存しません。

## Authentication以外の圧力試験

Authenticationだけに都合のよい抽象になっていないことを確認するため、`sales.order.submit`等の別Capabilityでも検証します。

ただし、現行`sales.order.submit`の`output sales.order`を最初から書き換えません。

まず小さいfixtureで、Commandを正常に実行して初めて分かる業務上の不成立が本当に存在するか確認します。

候補は、競合状態、既確定、外部受付側の業務拒否等です。

ValidationやAuthorizationで事前に判定できる不成立を、Command Outcomeへ重複させません。

名前付きOutcomeが不自然であれば、Authentication向けに抽象を寄せ過ぎていないか設計を見直します。

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

期待した`denied`や`rejected`が返るScenarioは正常終了できます。

Execution ErrorはScenarioを完了できなかった状態として扱います。

副作用を持つCapabilityでExecution Errorが返った場合、外部状態を確認せず自動retryしません。必要な場合はRQ-018 / RQ-020のObservation、unknown、takeover等の既存境界を利用します。

## 第一Web Targetへの引き継ぎ

Web TargetはOutcomeをHTTP statusや利用者向けmessageへProjectionできます。

ただし、Web固有表現をSemantic Outcomeの定義にはしません。

また、次を一つの「エラー」へ無条件に統合しません。

- Validationの不成立
- Authorization denial
- CapabilityのBusiness Outcome
- Execution Error
- Observationで確認した状態
- developer向けDiagnostic

## 実装順

### Step 1 — Semantic model

- Outcome modelを追加する
- Capabilityに`Outcomes`を追加する
- 既存`Output`を維持する
- Outcomeの安定したNormalization順を定める

### Step 2 — Parser / Validation

- `outcome <name> <payload-type>`を追加する
- `output`と`outcome`の併用を拒否する
- Outcome名重複を拒否する
- payload型を検証する
- parse / validation errorをsourceへ戻せるようにする

### Step 3 — 最小fixtureと互換性確認

- named Outcomeを持つ小さいCapabilityを追加する
- 既存`output` CapabilityのCanonical表現が不用意に変わらないことを確認する
- Outcome declarationの並べ替えでnormalized meaningが変わらないことを確認する
- source file renameでOutcome identityが変わらないことを確認する

### Step 4 — Authentication Reference

- `security.authenticate`へ`authenticated` / `denied`を追加する
- legacy認証fixtureと接続する
- denialとExecution Errorを別経路で再現する
- secret非混入を検証する

### Step 5 — Invocation adapter

- Business Resultの最小表現を定める
- Custom Go / process等からOutcomeへMappingする
- unknown Outcomeやpayload不一致を契約違反として扱う
- Runtime errorをBusiness Outcomeへ変換しない

### Step 6 — Driver

- expected OutcomeをScenarioから確認できるようにする
- denial / rejectionをDriver障害にしない
- Execution Errorを別に記録する
- Command / Observation境界を維持する

### Step 7 — 別Capabilityで圧力試験

- `sales.order.submit`等で同じ境界を検証する
- Authentication固有の概念が混入していないか監査する
- 必要ならOutcome modelを修正する

### Step 8 — Web Projection

- Business Outcome / Execution Error / Observationを別のPresentationへ写す
- Web固有statusをCanonicalへ逆流させない
- 第二Targetへ比較可能なEvidenceを残す

## Conformance

少なくとも次を自動検証します。

### Semantic / Parser

- 既存`output` Capabilityを従来どおりparseできる
- named Outcomeをparseできる
- `output` + `outcome`を拒否する
- 重複Outcome名を拒否する
- 不正payload型を拒否する
- Outcome declaration順だけでnormalized meaningが変わらない
- source file renameだけでOutcome identityが変わらない

### Invocation

- 定義済みOutcomeを正常に返せる
- 未定義Outcomeを契約違反として扱う
- payload不一致をExecution Errorとして扱う
- Business ResultとExecution Errorを同時に成立させない
- Plugin Runtime `ResponseError`をSemantic denialへ変換しない

### Reference / Driver

- valid credential → `authenticated`
- credential mismatch → `denied`
- locked / disabled → `denied`
- adapter / Database failure → Execution Error
- expected denial Scenarioを正常なScenarioとして扱える
- secretがCanonical World / Source Index / Scenario Evidenceへ混入しない
- Business OutcomeとObservationを区別できる

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

次をRepository上のReference / Conformanceから説明できることを完了条件とします。

- 業務上の複数の結末をSemantic Capabilityの名前付きOutcomeとして表せる
- 既存の単一`output` Capabilityを不用意に壊さない
- 業務OutcomeとExecution Errorを区別できる
- Authenticationのdenialを正常な業務結果として扱える
- Authenticationの実行障害をdenialへ変換しない
- DriverからOutcomeをScenario Evidenceとして扱える
- Authentication以外のCapabilityにも同じ境界を無理なく適用できる
- Target固有表現をSemantic Outcomeへ逆流させない
- Command / Observation境界を維持できる

## 関連文書

- `plan/adr/0020-capability-business-outcome-boundary.md`
- `plan/CAPABILITY_OUTCOME_V1_DESIGN.md`
- `plan/LEGACY_AUTH_V1_DESIGN.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000007-first-web-target-projection.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md` RQ-001、RQ-008、RQ-009、RQ-012、RQ-018、RQ-021
