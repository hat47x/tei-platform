# Capability Outcome v1 実装前設計

## 目的

この文書は、`ISSUE-000009` / GitHub Issue #77で扱う「Capabilityの業務結果と実行エラーの分離」を、最初の実装へ渡せる粒度まで具体化するものです。

中心となる判断は[ADR-0020](../adr/0020-capability-business-outcome-boundary.md)に置きます。この文書では、その判断を現在のSemantic model、Parser、Plugin Runtime、Driver計画、第一Web Target計画へどのように接続するかを整理します。

v1では一般的な例外体系やsum typeを作りません。既存の仕組みをできるだけ保ったまま、業務上の分岐だけを明示できる最小拡張を行います。

## 現行実装から確認できる前提

### Semantic Capabilityは一つの`Output`を持つ

現行の`semantic.Capability`は次の情報を持ちます。

```text
ID
Symbol
Inputs
Output
```

`Output`は一つのCanonical IDです。

この構造は、通常の成功時に一つの値を返すCapabilityには十分です。一方、Authenticationのように「処理は正常に完了したが業務上の結末が複数ある」Capabilityでは、分岐を型として表せません。

### Semantic Parserは`input`と`output`を直接読む

現行Parserでは、Capability本文に`input`または`output`を記述します。

```text
capability sales.order.submit {
  input order sales.order
  output sales.order
}
```

改行自体を構文上の区切りには利用していません。このため、v1の`outcome`構文ではpayload型を省略可能にせず、必ず型を明示します。

### Plugin Runtimeには実行失敗を表す経路がすでにある

Plugin Runtimeの応答には`ok / error`のstatusがあり、Host境界ではGo `error`も利用されています。

Resource Leaseの取得失敗、process起動失敗、artifact verification失敗、response identity不一致、Pluginからのerror応答などは、すでにInvocationを正常完了できなかった状態として扱えます。

この既存経路を業務上の`denied`や`rejected`に流用しません。

## v1の意味モデル

### 名前付きOutcome

Capabilityに、業務上の分岐を表す名前付きOutcomeを追加します。

概念上のGo modelは次を第一候補とします。

```go
type Outcome struct {
    Name    core.LocalName   `json:"name"`
    Payload core.CanonicalID `json:"payload"`
}

type Capability struct {
    ID       core.CanonicalID `json:"id"`
    Symbol   string           `json:"symbol"`
    Inputs   []Parameter      `json:"inputs,omitempty"`
    Output   core.CanonicalID `json:"output,omitempty"`
    Outcomes []Outcome        `json:"outcomes,omitempty"`
}
```

field名は実装時のGo整合で微調整して構いませんが、次の意味は変えません。

- Outcome名はCapability内のlocal nameである
- Outcomeごとにpayload型を一つ持つ
- `Output`と`Outcomes`は排他的である
- Outcomeのidentityはfile pathではなくCapability ID + local nameで決まる

Outcome自体へ独立したglobal Canonical IDはv1では与えません。

### `output`を残す理由

既存Capabilityを一括移行しません。

`output`は、「正常に完了したとき一つの通常resultを返す」という現在の契約として残します。

名前付きOutcomeは、「業務上の結末を区別すること自体が意味を持つ」Capabilityで使います。

```text
output
  単一の通常result

outcomes
  業務上の複数の結末を名前付きで表す
```

同一Capabilityで両方を定義することは認めません。

また、既存`output`を内部的に`success`や`completed`という名前付きOutcomeへ自動変換しません。作者が定義していない業務語彙をCanonicalへ追加しないためです。

## Authoring DSL

### 基本構文

v1では次の構文を第一候補とします。

```text
capability security.authenticate {
  input loginId std.identifier
  input credential security.credential

  outcome authenticated security.authenticated_subject
  outcome denied security.authentication_denial
}
```

構文は次の三要素から成ります。

```text
outcome <local-name> <payload-type>
```

`payload-type`はv1では必須です。

### payloadなしのOutcomeをまだ導入しない

次のような省略形はv1では扱いません。

```text
outcome accepted
```

現在のParserは改行を意味境界として使わないため、省略可能な第三要素は構文を複雑にします。

また、現行`std.*`には`unit`型がありません。

値を持たないOutcomeが実際のReferenceで必要になった場合は、次のいずれが自然かを別途判断します。

- `std.unit`を標準型として導入する
- domain側の明示的なresult dataを使う
- Outcome payloadを省略可能にするParser拡張を行う

#77を解くためだけに先回りして決めません。

## Validation

CapabilityのValidationには少なくとも次を追加します。

### `output`と`outcome`の排他

次はinvalidです。

```text
capability example.run {
  output example.value
  outcome denied example.denial
}
```

どちらが正本か不明になるため、ParserまたはSemantic validationで明示的に拒否します。

### Outcome名はCapability内で一意

次もinvalidです。

```text
outcome denied example.denial_a
outcome denied example.denial_b
```

Outcomeのlocal nameは一つのCapability内で一意にします。

### payload型を解決できること

Outcomeのpayloadは、現行`output`と同等に、Semantic Worldから解決できるCanonical type / data identityである必要があります。

どの種類のSemantic subjectをpayloadとして許すかは、現行`Output` validationを再利用し、#77だけで新しい型解決規則を作らないことを優先します。

### Outcome数

`outcome`は一件以上定義できるものとします。

「複数でなければならない」という制約は置きません。業務上の結末へ明示的な名前を与えること自体に意味がある場合を不必要に禁止しないためです。

ただし、単一の通常resultしか必要ない場合は既存`output`を優先します。

## NormalizationとCanonical World

### 宣言順に意味を持たせない

Outcomeの宣言順は業務上の優先順位や成功度を意味しません。

Canonical Worldの再現性を保つため、NormalizationではCapability内のOutcomeをlocal name等の安定したkeyで整列することを検討します。

Referenceの並べ替えだけでCanonical meaningが変わらないことをConformanceで確認します。

### file pathをidentityに含めない

Outcomeのidentityは次で表せます。

```text
Capability: security.authenticate
Outcome:    denied
```

`semantic/security/auth.tei`等のpathはidentityにしません。

source fileのrename / moveでCanonical Outcomeの意味が変わらないことを、既存のSource-to-Canonical原則に合わせます。

## Source IndexとDiagnostic

OutcomeはCapabilityのauthoring sourceに含まれるため、Diagnosticから元の宣言へ戻れることが必要です。

v1の最小実装では、少なくともCapability sourceへ戻れる既存経路を維持します。

Outcome個別のrangeまでSource Indexへ持たせるかは実装負荷を見て決めますが、最終的には次のようなSubjectを表せると扱いやすくなります。

```text
semantic-capability-outcome:
  capability = security.authenticate
  outcome    = denied
```

ただしOutcomeにglobal Canonical IDを新設するためだけにSource Index構造を複雑化しません。

## Invocation境界

### Business ResultとExecution Error

DriverやSemantic Capability adapterから見た概念契約は次です。

```text
Invoke Capability
       │
       ├─ 正常に実行できた
       │      ↓
       │   Business Result
       │      ├─ Output value
       │      └─ Named Outcome(name, payload)
       │
       └─ 実行を完了できなかった
              ↓
          Execution Error
```

名前付きOutcomeを持つCapabilityでは、一回の正常なInvocationにつきOutcomeを一つだけ選びます。

### Go API

v1ではGoの一般的な`error`をExecution Errorとして利用する方が、既存コードとの整合がよいと考えます。

概念上は次の形です。

```go
result, err := Invoke(...)
if err != nil {
    // Execution Error
}

// result はBusiness Result
```

`ExecutionError`という大きな独自例外階層を先に作りません。

必要になれば、retryable、unknown、contract violation等の分類を後から追加できます。

### Business Resultの最小表現

Driver等が扱う共通結果は、概念上次の情報を持てれば十分です。

```text
BusinessResult
- capability identity
- mode: output | outcome
- outcome name（outcomeの場合）
- payload
```

具体的なGo structはDriver実装と合わせて確定します。

## Plugin RuntimeとのMapping

Semantic Capabilityの実現方式としてprocess Plugin等を使う場合、次の対応を基本とします。

```text
Plugin Runtime ResponseOK
        ↓
Semantic Business Result

Plugin Runtime ResponseError / Go error
        ↓
Execution Error
```

### `denied`を`ResponseError`へ変換しない

たとえばAuthenticationでcredentialが一致しなかった場合、認証処理自体は正常に完了しています。

したがってprocess境界では概念上、次のように扱います。

```text
Status  = ok
Payload = {
  outcome: "denied",
  value: ...
}
```

Payloadのwire formatはSemantic Capability adapterの実装時に確定します。

### 契約違反はExecution Error

次はBusiness Outcomeではありません。

- Capabilityに定義されていないOutcome名を返した
- payload型を満たさない
- Outcomeが二つ返された
- response identityが一致しない
- response自体をdecodeできない

これらは実装またはprotocolの契約違反としてExecution Errorにします。

## Binding / implementation

Outcome名はimplementation固有のreturn codeへ直接固定しません。

既存関数や外部APIが次のような固有codeを返していても、Binding / adapterでSemantic Outcomeへ対応付けます。

```text
legacy code "00" → authenticated
legacy code "01" → denied(reason=credential_mismatch)
legacy code "03" → denied(reason=locked)
transport failure → Execution Error
```

このMappingの来歴を追跡できるようにし、同じSemantic Capability identityのままimplementationを交換できることを優先します。

## Authentication Reference

### Semantic source

v1の最初の実例では、Reference Workspace内の`security`意味空間へ必要なpayload dataを定義します。

概念例:

```text
data AuthenticatedSubject {
  attribute subjectId std.identifier
}

data AuthenticationDenial {
  attribute reason security.authentication_denial_reason
}

capability security.authenticate {
  input loginId std.identifier
  input credential security.credential
  outcome authenticated security.authenticated_subject
  outcome denied security.authentication_denial
}
```

実際のsymbol / file名は既存Referenceの命名規則と合わせて実装時に決めます。

### 確認する結果

少なくとも次を区別します。

1. credentialが一致し、`authenticated`を返す
2. credential不一致で`denied`を返す
3. account disabled / lockedで`denied`を返す
4. Database接続失敗でExecution Errorを返す

2と3はDriver Scenario自体の障害ではありません。

## `sales.order.submit`による圧力試験

Authenticationだけに適した抽象になっていないことを確認するため、次に`sales.order.submit`で同じ境界を検証します。

現行Capabilityは一つの`output sales.order`を持っています。これを直ちに書き換えません。

まず孤立したfixtureまたは小さいbranch上の実験で、たとえば次のような業務結果が本当に必要かを確認します。

```text
outcome submitted sales.order
outcome rejected sales.order_submission_rejection
```

ValidationやAuthorizationで事前に除外できる拒否までCommand Outcomeへ重複させません。

競合状態、既に確定済み、外部受付側の業務拒否など、**Commandを正常に実行して初めて分かる不成立**がReferenceとして必要な場合に名前付きOutcomeを利用します。

この実例が不自然であれば、Outcome modelをAuthenticationへ合わせ過ぎていないか見直します。

## Driverへの引き継ぎ

DriverはOutcomeを「成功 / 失敗」のbooleanへ潰しません。

Scenario Evidenceでは少なくとも次を区別します。

```text
validation result
authorization result
business result
  - output / outcome
execution error
command acknowledgement
observed result
```

### Scenarioの正常終了

`denied`や`rejected`を期待するテストScenarioは、期待したOutcomeが返れば正常終了です。

たとえば「locked userは認証されない」というScenarioで`denied(reason=locked)`が返った場合、それはDriver障害ではありません。

### Execution Error

Database接続失敗等でCapability自体を完了できなかった場合は、Scenarioを実行できなかった状態として記録します。

副作用を持つCapabilityでは、Execution Errorだけを理由に自動retryしません。外部状態がunknownになり得る場合はRQ-018 / RQ-020の既存境界を利用します。

## 第一Web Targetへの引き継ぎ

WebはSemantic OutcomeをTarget固有のPresentationへ写せます。

```text
denied(reason=not_authorized)
        ↓ Web Adapter
HTTP 403 + 利用者向けmessage
```

ただし逆向きには定義しません。

```text
HTTP 403
   ─X→ Semantic Outcomeの定義
```

同じOutcomeをDesktop、API、MCP等では別の表現へ写せる必要があります。

また、次を一つの「エラー表示」へ無条件に統合しません。

- Validationの不成立
- Authorization denial
- CapabilityのBusiness Outcome
- Execution Error
- Observationで確認した状態
- authoring / developer Diagnostic

## Compatibility

### 既存Capabilityを壊さない

`output`だけを持つ既存Capabilityは、そのまま読み込めることを第一条件とします。

#77の導入だけで既存Reference Worldを全面的にOutcome形式へ変換しません。

### 新旧構文の同時使用

Repository全体としては、Capabilityごとに次のいずれかを選べます。

```text
A. outputを使う既存・単一result Capability
B. outcomeを使う名前付き分岐Capability
C. どちらも持たないno-output Capability
```

一つのCapability内でAとBを混ぜません。

## 実装順

### Step 1 — Semantic model

- `semantic.Outcome`相当を追加する
- `Capability.Outcomes`を追加する
- `Output`を維持する
- Normalizationの安定順序を定める

### Step 2 — Parser / Validation

- `outcome <name> <payload-type>`をParseする
- `output`と`outcome`の併用を拒否する
- Outcome名重複を拒否する
- payload型の解決を検証する
- parse errorを自然なsource diagnosticへつなぐ

### Step 3 — 小さいSemantic fixture

- named Outcomeを持つ最小Capabilityを用意する
- World JSON / goldenの差を確認する
- 既存`output` Capabilityが変わらないことを確認する
- declaration reorder / file renameでidentityが変わらないことを確認する

### Step 4 — Authentication Reference

- `security.authenticate`へ`authenticated` / `denied`を導入する
- legacy認証fixtureとBindingする
- denialとExecution Errorを別経路で再現する
- secretがOutcome payloadやEvidenceへ混入しないことを確認する

### Step 5 — Invocation adapter

- Business Resultの最小表現を定める
- process / Custom Go等からOutcomeへMappingする
- Runtime `error`をBusiness Outcomeへ変換しない
- 未知Outcome等の契約違反をExecution Errorにする

### Step 6 — Driver

- expected OutcomeをScenarioでassertできるようにする
- denial / rejectionをScenario障害にしない
- Execution Errorを別に記録する
- Command / Observation境界を維持する

### Step 7 — 別Capabilityで圧力試験

- `sales.order.submit`等で名前付きOutcomeが自然か確認する
- Authentication固有の概念が混入していないか監査する
- 必要ならOutcome modelを修正する

### Step 8 — Web Projection

- Business Outcome / Execution Error / Observationを別のPresentationへ写す
- HTTP status等をCanonicalへ逆流させない
- Gate Cで第二Targetへ渡す差分を残す

## Conformance

少なくとも次を自動検証します。

### Semantic / Parser

- 既存`output` Capabilityを従来どおりparseできる
- named Outcomeをparseできる
- `output` + `outcome`を拒否する
- 重複Outcome名を拒否する
- 不正なpayload typeを拒否する
- Outcome declarationの並べ替えでnormalized meaningが変わらない
- source file renameでOutcome identityが変わらない

### Invocation

- 定義済みOutcomeを正常に返せる
- unknown Outcomeを契約違反として拒否する
- payload不一致をExecution Errorとして扱う
- Business OutcomeとExecution Errorを同時に返さない
- Plugin Runtime `ResponseError`をSemantic denialへ変換しない

### Authentication

- valid credential → `authenticated`
- credential mismatch → `denied`
- locked / disabled → `denied`
- adapter / Database failure → Execution Error
- secretをCanonical World / Source Index / Scenario Evidenceへ保存しない

### Driver / Target

- expected denial Scenarioを正常なScenarioとして扱える
- Execution Errorを業務result codeへ変換しない
- Business OutcomeとObservationを区別できる
- Web固有表現を外してもSemantic Outcomeが変わらない

## まだ固定しないこと

次はv1の実装Evidenceを得るまで固定しません。

- 一般的なvariant / union type
- optional type
- `std.unit`
- Outcome payloadの省略構文
- Outcomeのglobal Canonical ID
- Outcome hierarchy
- retryable / fatal等の一般的なExecution Error taxonomy
- HTTP / gRPC等への標準status mapping
- 非同期workflow全体のresult model

## 完了条件

#77の最初の実装は、次をRepository上のReference / Conformanceから説明できたときに完了とします。

- 業務上の複数の結末をSemantic Capability契約として表せる
- 既存の単一`output` Capabilityを不用意に壊さない
- 業務OutcomeとExecution Errorを実行境界で区別できる
- Authenticationのdenialを正常な業務結果として扱える
- Authenticationの実行障害をdenialへ変換しない
- DriverがOutcomeをScenario Evidenceとして扱える
- `sales.order.submit`等の別Capabilityでも同じ境界が不自然でない
- Web TargetがOutcomeを表示へProjectionしても上位meaningを変更しない
- Command / Observation境界を維持できる

## 関連文書

- `plan/adr/0020-capability-business-outcome-boundary.md`
- `plan/issue/000009-capability-outcome-and-execution-error.md`
- `plan/design/LEGACY_AUTH_V1_DESIGN.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000007-first-web-target-projection.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
