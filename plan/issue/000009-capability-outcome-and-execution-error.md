# ISSUE-000009: Capabilityの業務Outcomeと実行エラーを分離する

- 状態: In Progress
- GitHub Issue: #77
- 設計判断: ADR-0020
- 最初の実装Evidence: PR #132

## 背景

既存認証Compatibilityを具体化する中で、Authenticationに固有ではないCapability実行上の不足が見つかりました。

従来のSemantic Capabilityは一つの`output`型を持つことができました。一方、実際の業務処理では、処理を正常に実行した結果として複数の業務上の結末があり得ます。

Authenticationであれば、次の違いです。

```text
Authenticated(subject)
Denied(reason)
ExecutionError(error)
```

credentialの不一致やaccount lockは、認証処理を正常に実行した結果としての拒否です。Database接続失敗やadapter異常は、認証判断そのものを完了できなかった状態です。

この境界はAuthenticationだけでなく、受注確定、審査、在庫引当、外部申請等のSemantic Capabilityにも現れます。

ADR-0020では、業務上の分岐をSemantic Capabilityの**名前付きOutcome**として表し、Execution ErrorはInvocation境界へ残す方針を採用しました。

詳細な設計は[`CAPABILITY_OUTCOME_V1_DESIGN.md`](../CAPABILITY_OUTCOME_V1_DESIGN.md)に置きます。

## 現在地

PR #132で、名前付きOutcomeのCanonical基盤がmainへ入りました。

これにより、当初の実装順のうち次が成立しています。

- Semantic modelへOutcomeを追加する
- `outcome <name> <payload-type>`をParserから扱う
- `output`と名前付きOutcomeの併用を拒否する
- Outcome名の重複を拒否する
- Outcome payload型の存在を検査する
- Outcomeの宣言順をCanonicalな意味にしない
- 既存`output` Capabilityをそのまま維持する

一方、Issue #77全体はまだ完了していません。

Gate Aへ必要な次のEvidenceが残っています。

- `security.authenticate`の実Reference
- `authenticated` / `denied`をBusiness Resultとして再現するfixture
- Database / adapter等の障害をExecution Errorとして別経路で再現するfixture
- Invocation側で既存`output`とnamed Outcomeを同じBusiness Result境界から扱う契約
- 未定義Outcomeやpayload不一致を業務拒否ではなく契約違反として扱う境界
- Driverが結果契約を利用できる入口

したがって、Gate A上の評価は`Blocked`ではなく**Partial**です。

## 関連するProduct Requirement

- RQ-001 業務上の意味を実装技術から独立して保持する
- RQ-008 Capability単位で実装を置き換える
- RQ-009 既存認証・認可仕様を保持する
- RQ-012 Driverで設計を実行・検証する
- RQ-018 CommandとObservationを分離する
- RQ-021 Reference / Conformanceで再現可能性を示す

## 採用した境界

### 業務上の分岐は名前付きOutcomeにする

v1では一般的なsum typeをSemantic型体系へ追加せず、Capabilityに直接ぶら下がる名前付きOutcomeを利用します。

```tei
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

現在のParserは改行を構文上の意味境界として利用しておらず、現行`std.*`には`unit`型もありません。

そのため、v1では次のようなpayload省略構文を導入しません。

```text
outcome accepted
```

各Outcomeに明示的なpayload型を要求します。payloadなしOutcomeや`std.unit`等は、実例から必要性を確認してから別途検討します。

### 既存`output`を維持する

一つの通常resultだけを返す既存Capabilityは、従来どおり`output`を利用できます。

```tei
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
- Binding解決失敗
- adapter異常
- response identity不一致
- 未定義Outcomeの返却
- Outcome payloadの契約不一致

Go境界では通常の`error`を利用できます。Plugin Runtimeでは`ResponseError`またはGo `error`をExecution Errorとして扱います。

業務上の`denied`や`rejected`を`ResponseError`へ変換しません。

## PR #132で成立したCanonical基盤

### Semantic model

`semantic.Capability`へ`Outcomes`が追加され、Outcomeは次を持ちます。

```text
Outcome
- Name
- Payload
```

既存`Output`は残っています。

### Parser

Capability本文で次を扱えます。

```text
outcome <local-name> <payload-type>
```

### Validation

少なくとも次を共通Semantic Validationから検査できます。

- `output`と`outcome`の併用
- Outcome名の欠損
- Outcome payloadの欠損
- Outcome名の重複
- 解決できないpayload型

Parserだけの規則にはしていないため、Canonical Worldを直接構築した場合も同じ境界を検査できます。

### Normalization

Outcomeの宣言順に意味を持たせず、local name順に正規化します。

単なる記述順の違いでCanonical World差分が発生しません。

### 既存互換

`sales.order.submit`等の既存`output` Capabilityは変更していません。

名前付きOutcomeの導入を理由に、既存の単一resultを暗黙変換しないことを維持しています。

## 次のReference — `security.authenticate`

Gate Aで次に必要なのは、Outcome構文そのものの追加ではなく、実際のAuthentication Referenceです。

少なくとも次を区別します。

1. 正しいcredential → `authenticated`
2. credential不一致 → `denied`
3. account disabled / locked → `denied`
4. Database / adapter障害 → Execution Error

2と3はDriver Scenarioの実行障害ではありません。

Outcome payloadやScenario Evidenceへpassword平文、token等のsecretを保存しません。

## 既存認証Compatibility #68との接続

#68では、既存システム固有の認証状態をTEI標準User schemaへ作り替えず、Authentication / Authorization Capabilityへ接続します。

#77で成立したOutcome基盤は、そのReferenceが次を表すために利用します。

```text
legacy authentication state
          ↓ Mapping / Binding
security.authenticate
          ↓
authenticated / denied
```

ただし、legacy schemaそのものをOutcome modelへ取り込みません。

認証実装の物理構造はBinding / implementation側へ残します。

## Invocation境界

Canonical Outcomeを宣言できるだけでは、Driverはまだ利用できません。

Invocation側では、概念上次を一つの結果境界から扱える必要があります。

```text
Capability Invocation
  ├─ Business Result
  │   ├─ Output value
  │   └─ Named Outcome(name, payload)
  └─ Execution Error
```

次を満たす最小表現を実Referenceから決めます。

- 定義済みOutcomeを正常に返せる
- 未定義Outcomeを契約違反として扱う
- payload不一致をExecution Errorとして扱う
- Business ResultとExecution Errorを同時に成立させない
- Plugin Runtime `ResponseError`をSemantic denialへ変換しない
- 既存`output` Capabilityも同じInvocation境界から扱える

一般的なExecution Error taxonomyを先に設計しません。

## Authentication以外の圧力試験

Authenticationだけに都合のよい抽象になっていないことを確認するため、後続で`sales.order.submit`等の別Capabilityでも検証します。

ただし、現行`sales.order.submit`の`output sales.order`を最初から書き換えません。

Commandを正常に実行して初めて分かる業務上の不成立が本当に必要かを確認してから、named Outcomeの適用を検討します。

ValidationやAuthorizationで事前に判定できる不成立をCommand Outcomeへ重複させません。

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

Execution ErrorはCapability invocationを完了できなかった状態として扱います。

副作用を持つCapabilityでExecution Errorが返った場合、外部状態を確認せず自動retryしません。必要な場合はRQ-018 / RQ-020のObservation、unknown、takeover等の既存境界を利用します。

## 第一Web Targetへの引き継ぎ

Web TargetはOutcomeをHTTP statusや利用者向けmessageへProjectionできます。

ただし、この作業はGate A以後へDeferredします。

Web固有表現をSemantic Outcomeの定義にはしません。

また、次を一つの「エラー」へ無条件に統合しません。

- Validationの不成立
- Authorization denial
- CapabilityのBusiness Outcome
- Execution Error
- Observationで確認した状態
- developer向けDiagnostic

## 実装順と現在の進捗

### Step 1 — Semantic model

**完了 — PR #132**

- Outcome modelを追加
- Capabilityに`Outcomes`を追加
- 既存`Output`を維持
- Outcomeの安定したNormalization順を確定

### Step 2 — Parser / Validation

**完了 — PR #132**

- `outcome <name> <payload-type>`を追加
- `output`と`outcome`の併用を拒否
- Outcome名重複を拒否
- payload型を検証
- 必須項目を検証

### Step 3 — Canonical互換性

**基礎完了 — PR #132**

- named Outcomeをparseできる
- 既存`output` Capabilityを維持
- Outcome declaration順をNormalizationで除去

Outcome個別のSource Index rangeや実Reference上のrename invarianceは、`security.authenticate` Referenceと合わせて必要性を確認します。

### Step 4 — Authentication Reference

**未完了**

- `security.authenticate`を実Referenceへ追加
- `authenticated` / `denied`を再現
- legacy認証fixtureと接続
- denialとExecution Errorを別経路で再現
- secret非混入を検証

### Step 5 — Invocation境界

**未完了**

- Business Resultの最小表現を定める
- Custom Go / process等からOutcomeへMappingする
- unknown Outcomeやpayload不一致を契約違反として扱う
- Runtime errorをBusiness Outcomeへ変換しない

### Step 6 — Driver

**Gate A通過後**

- expected OutcomeをScenarioから確認できるようにする
- denial / rejectionをDriver障害にしない
- Execution Errorを別に記録する
- Command / Observation境界を維持する

### Step 7 — 別Capabilityで圧力試験

**Gate A後半またはDriver段階**

- `sales.order.submit`等で同じ境界を検証する
- Authentication固有の概念が混入していないか監査する
- 必要ならOutcome modelを修正する

### Step 8 — Web Projection

**Deferred**

- Business Outcome / Execution Error / Observationを別のPresentationへ写す
- Web固有statusをCanonicalへ逆流させない
- 第二Targetへ比較可能なEvidenceを残す

## Conformance

### 既に成立

- 既存`output` Capabilityを従来どおりparseできる
- named Outcomeをparseできる
- `output` + `outcome`を拒否する
- 重複Outcome名を拒否する
- 不正 / 欠損payloadを拒否する
- Outcome declaration順だけでnormalized meaningが変わらない

### Gate Aへ残るConformance

- `security.authenticate` Referenceを読み込める
- valid credential → `authenticated`
- credential mismatch → `denied`
- locked / disabled → `denied`
- adapter / Database failure → Execution Error
- expected denialを正常なBusiness Resultとして扱える
- 未定義Outcomeを契約違反として扱える
- payload不一致をExecution Errorとして扱える
- Business ResultとExecution Errorを同時に成立させない
- secretがCanonical World / Source Index /通常Evidenceへ混入しない

### Driver以後

- expected denial Scenarioを正常なScenarioとして扱える
- Business OutcomeとObservationを区別できる
- Target固有表現をCanonicalへ逆流させない

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

## Gate Aでの完了条件

Issue #77全体のすべてをGate A前に終える必要はありません。

Gate Aでは少なくとも次をRepository上のReference / Conformanceから説明できることを条件とします。

- 業務上の複数の結末をSemantic Capabilityの名前付きOutcomeとして表せる — **成立済み**
- 既存の単一`output` Capabilityを不用意に壊さない — **成立済み**
- Authenticationのdenialを正常な業務結果として扱える
- Authenticationの実行障害をdenialへ変換しない
- Invocation側でBusiness ResultとExecution Errorを区別できる
- Driverへ渡せる結果契約を説明できる

Web Projection、別Target、一般化されたerror taxonomyはGate A以後へDeferredできます。

## Issue全体の完了条件

次を最終的に説明できることを完了条件とします。

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

- `plan/GATE_A_READINESS_AUDIT.md`
- `plan/adr/0020-capability-business-outcome-boundary.md`
- `plan/CAPABILITY_OUTCOME_V1_DESIGN.md`
- `plan/LEGACY_AUTH_V1_DESIGN.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000007-first-web-target-projection.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md` RQ-001、RQ-008、RQ-009、RQ-012、RQ-018、RQ-021
- GitHub Issue #68 / #69 / #77 / #124
- GitHub PR #132
