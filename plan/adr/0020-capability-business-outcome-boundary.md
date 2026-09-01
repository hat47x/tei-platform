# ADR-0020: Capabilityの業務Outcomeと実行エラーを分離する

- 状態: 採用
- 日付: 2026-09-01

## 背景

TEIのSemantic Capabilityは、Applicationから見た業務上の要求を、実装方式から分離して表すための境界です。

現在のCapabilityは入力と一つの`output`型を持ちます。この形は、通常の成功時に一つの値を返すCapabilityには十分です。一方、既存認証Compatibilityを具体化すると、処理を正常に実行した結果として複数の業務上の結末があり得ることが明確になりました。

Authenticationを例にすると、少なくとも次を区別する必要があります。

```text
Authenticated(subject)
Denied(reason)
ExecutionError(error)
```

credentialの不一致やaccount lockは、認証処理を最後まで実行した結果として「認証しない」と判断した状態です。Database接続失敗やadapter異常は、判断そのものを完了できなかった状態です。

この違いを一つのerrorへ畳むと、DriverやTargetから業務上の意味が失われます。逆に、実行障害まで業務上のresult codeへ押し込むと、retry、障害解析、Observationとの関係が曖昧になります。

現行Plugin Runtimeには、`ResponseStatus = ok / error`とGoの`error`がすでにあり、Hostや実行境界の失敗を表す経路があります。この既存境界を壊して業務上の拒否を流し込む必要はありません。

## 判断

Capabilityの結果を、**業務Outcome**と**実行エラー**に分離します。

### 業務OutcomeはSemantic Capabilityの契約に含める

処理を正常に実行した結果として選ばれる業務上の分岐は、Semantic Capability側で表します。

v1では一般的なsum typeをSemantic型体系へ追加せず、Capabilityに直接ぶら下がる**名前付きOutcome**を導入する方針とします。

概念上は次の形です。

```text
Capability
├─ Inputs
├─ Output        // 現行の単一result。既存互換として残す
└─ Outcomes      // 名前付きの業務上の分岐
    ├─ Name
    └─ PayloadType
```

名前付きOutcomeのidentityは、CapabilityのCanonical IDとOutcomeのlocal nameの組み合わせで決まります。Outcomeをfile pathや表示順へ依存させません。

### v1のOutcomeはpayload型を明示する

v1では、各Outcomeにpayload型を明示します。

```text
outcome authenticated security.authenticated_subject
outcome denied security.authentication_denial
```

「payloadなし」を表す省略構文や一般的な`unit`型は、このADRでは導入しません。現在のParserは改行を意味境界として利用しておらず、optionalなpayload構文を追加すると文法上の曖昧さを増やすためです。

payloadを必要としないOutcomeが実例で必要になった場合は、Referenceから必要性を確認したうえで、`std.unit`等の標準型を別途検討します。

### `output`と`outcome`は同一Capabilityで併用しない

現行の`output`は既存Capabilityとの互換性のため残します。

ただし、一つのCapabilityに`output`と名前付き`outcome`を同時に定義することは認めません。

これにより、同じ業務結果に二つの正本を作ることを避けます。

既存の`output`へ暗黙の`success`や`completed`というOutcome名を与えることもしません。名称を自動で発明すると、作者が定義していない業務上の意味をTEI側が追加することになるためです。

### 実行エラーはInvocation境界に残す

Capabilityの実装を呼び出したものの、処理そのものを完了できなかった状態はSemantic Outcomeにしません。

たとえば次は実行エラーです。

- Databaseへ接続できない
- processを起動できない
- adapterが異常終了する
- Resource Leaseを取得できない
- Plugin Runtimeの応答identityが不正である
- 実装がCapability契約にないOutcomeを返す

Goの実装境界では、v1では通常の`error`を実行エラーとして利用できます。Plugin Runtimeを経由する場合は、`ResponseError`またはGo `error`を実行エラーとして扱います。

業務上の`denied`や`rejected`は`ResponseError`へ変換せず、正常に実行された応答のpayloadとして返します。

### 成功したInvocationでは業務結果を一つだけ選ぶ

名前付きOutcomeを持つCapabilityでは、正常に完了した一回のInvocationにつき、契約に定義されたOutcomeを一つだけ返します。

```text
Invocation
├─ Business Result
│   ├─ Output value
│   └─ Named Outcome(name, payload)
└─ Execution Error
```

`Business Result`と`Execution Error`を同時に成立させません。

実装が未知のOutcome名、型の合わないpayload、複数Outcomeを返した場合は、業務上の拒否ではなく契約違反として扱います。

### CommandとObservationの境界は維持する

このADRはRQ-018のCommand / Observation分離を置き換えません。

Capabilityが`submitted`等の業務Outcomeを返しても、それが外部世界の最終状態を保証するとは限りません。必要な処理では、実際に何が成立したかをObservation / Evidenceで別に確認します。

```text
Command Invocation
      ↓
Business Outcome
      ↓
必要ならObservation
      ↓
Observed Actual State
```

### Target固有の表現をOutcomeの意味にしない

Web Targetが`denied`をHTTP 403へ写すなど、Target側でOutcomeを利用者向け表現へ変換することはできます。

ただし、HTTP status、dialog、toast等のTarget固有表現からSemantic Outcomeを逆定義しません。

第二Targetでも同じOutcomeを異なる形で表現できることを前提とします。

## v1のReference

最初のReferenceは`security.authenticate`とします。

概念上、次の二つのOutcomeを持たせます。

```text
capability security.authenticate {
  ...
  outcome authenticated security.authenticated_subject
  outcome denied security.authentication_denial
}
```

`authenticated`はsubjectを返し、`denied`はcredential不一致、disabled、lock等の理由を表すpayloadを返します。

Database接続失敗等はOutcomeへ追加せず、Execution Errorとして外側へ残します。

Authenticationだけに都合のよい抽象になっていないことを確認するため、次に`sales.order.submit`等の別Capabilityで、業務上の成立・不成立を同じ境界から扱えるか検証します。

## 影響

### 良くなる点

- 業務上の拒否をRuntime障害と混同しなくなる
- Driverが正常な業務Scenarioの一部としてdenial / rejectionを扱える
- Targetが業務結果と障害を別のPresentationへ写せる
- 実装方式を交換してもCapabilityの業務Outcomeを維持できる
- Authenticationだけの特殊なerror規約を避けられる
- sentinel値で欠損を表す必要が減る
- Command / Observationとの責務分担を保てる

### 増える負担

- Semantic Capability modelとParserにOutcomeを追加する必要がある
- Outcome名とpayload型のValidationが必要になる
- Binding / adapterで実装結果をOutcomeへ対応付ける必要がある
- DriverやTargetがBusiness ResultとExecution Errorを区別して扱う必要がある

これらは、業務上の意味を失わずに実行境界を保つために受け入れます。

## 採用しない案

### 業務上の拒否をすべてGo error / Runtime errorで返す

実行障害との区別が失われ、DriverやTargetから業務上の意味を追えなくなるため採用しません。

### 実行障害を業務result codeへ含める

retryや障害解析、Resource / Runtimeの責務が業務modelへ逆流するため採用しません。

### 一般的なvariant / sum typeを先にSemantic型体系へ導入する

将来的に必要になる可能性はありますが、#77を解くために型体系全体を先に拡張する必要はありません。まずCapability境界に限定した名前付きOutcomeで実証します。

### `output`を直ちに廃止して全CapabilityをOutcomeへ移行する

既存ReferenceとCanonical Worldへ不要に大きな変更を与えるため採用しません。

### `output`を暗黙の`success` Outcomeへ正規化する

作者が定義していないOutcome名をTEIが発明することになるため採用しません。

## 検証

`ISSUE-000009`では、少なくとも次を確認します。

- 名前付きOutcomeをSemantic sourceからCanonicalへ読み込める
- `output`だけを持つ既存CapabilityのCanonical表現を不用意に変えない
- 一つのCapabilityで`output`と`outcome`の併用を拒否する
- Outcome名の重複を拒否する
- Outcome payload型を検証できる
- `security.authenticate`で`authenticated`と`denied`を区別できる
- credential不一致やlockをExecution Errorへ変換しない
- Database接続失敗等をBusiness Outcomeへ変換しない
- Plugin Runtime経由でもBusiness Outcomeを`ResponseOK`側で保持できる
- DriverがBusiness OutcomeとExecution Errorを別の結果として記録できる
- Web TargetがTarget固有表現へProjectionしてもSemantic Outcomeを変更しない
- `sales.order.submit`等、Authentication以外のCapabilityでも同じ境界が不自然でない
- 必要な処理ではBusiness OutcomeとObservationを別に保持できる

## 関連文書

- `plan/CAPABILITY_OUTCOME_V1_DESIGN.md`
- `plan/LEGACY_AUTH_V1_DESIGN.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000007-first-web-target-projection.md`
- `plan/issue/000009-capability-outcome-and-execution-error.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md` RQ-001、RQ-008、RQ-012、RQ-018、RQ-021
