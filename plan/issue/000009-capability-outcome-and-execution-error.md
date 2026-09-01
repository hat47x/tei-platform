# ISSUE-000009: Capabilityの業務結果と実行エラーを分離する

- 状態: 着手可能
- GitHub Issue: #77

## 背景

既存認証Compatibilityを具体化する中で、Authenticationに固有ではないCapability実行上の不足が見つかりました。

現在のSemantic Capabilityは一つのoutput型を持ちます。一方、実際の業務処理では、少なくとも次の三つを区別したい場合があります。

```text
業務上の成功
業務上の不成立・拒否
実行そのものの失敗
```

Authenticationであれば、次の違いです。

```text
Authenticated(subject)
Denied(reason)
ExecutionError(error)
```

credential不一致やaccount lockは、認証処理を正常に実行した結果としての拒否です。Database接続失敗やadapter異常とは意味が異なります。

この区別は、受注確定、審査、在庫引当、外部申請等の業務Capabilityにも現れ得ます。Authenticationだけのために例外的な仕組みを作らず、Capability実行全体の境界として検討します。

## 関連するProduct Requirement

- RQ-001 業務上の意味
- RQ-008 Capability単位の置換
- RQ-012 Driver
- RQ-018 CommandとObservationの分離
- RQ-021 Reference / Conformance

## 問題

### 業務上の不成立と実行エラーを同じerrorへ畳まない

業務上のdenialをGo errorやRuntime errorだけで表すと、DriverやTargetから見た意味が失われます。

逆に、実行エラーまで業務結果codeへ押し込むと、retry、障害解析、Observation等の扱いが曖昧になります。

### 欠損をsentinel値だけで表さない

たとえばAuthentication失敗時に、`subject_id = ""`だけで「subjectが存在しない」と表す設計は避けます。

空文字が正規値なのか欠損なのかを型から判断できず、同じ規約が他Capabilityへ広がるためです。

### 型体系を必要以上に広げない

一方、Authentication一件のために一般的なsum type、例外体系、workflow result model等を一度に導入することも避けます。

既存CapabilityとDriverで本当に必要な最小境界を確認します。

## 比較する案

### 案A — 業務上のoutcomeをSemantic outputとして表す

業務上の成功・不成立をSemantic outputとして明示し、実行エラーはCapability invocation側で別に扱います。

業務上の意味がCanonicalに見える利点がありますが、現行Semantic型にはoptional / variant表現がないため、型拡張の影響を見極める必要があります。

### 案B — Capability invocation envelopeで分離する

概念上、実行結果を次のようなenvelopeとして扱います。

```text
InvocationResult
├─ Outcome
└─ ExecutionError
```

Outcomeの内部に業務上のsuccess / denialを持たせます。

複数Capabilityへ再利用しやすい反面、業務上のresultがSemantic Layerから見えにくくならないよう注意が必要です。

### 案C — Semantic型へoptional / variantを最小追加する

Semantic型体系に最小限の欠損・分岐表現を追加します。

表現力は高まりますが、影響範囲が広いため、最初から採用する前提にはしません。

## 検証方法

最初はAuthenticationを具体例として使います。

少なくとも次を同じ呼び出し境界から区別します。

1. 認証成功
2. credential不一致
3. account disabled / locked等の業務上の拒否
4. 実装側の接続失敗

次に、Authentication以外のCapabilityでも同じ境界が自然に使えるかを確認します。候補として`sales.order.submit`を利用できます。

## Driverへの引き継ぎ

Driverでは、業務上のdenialと実行障害を異なる意味として扱える必要があります。

- denial: Scenarioの正常な結果として記録できる
- execution error: Scenarioを実行できなかった障害として扱う

副作用を持つCapabilityでは、execution errorが返ったという理由だけで自動retryしてはいけません。retry可否は、実行の冪等性や外部状態の観測結果と合わせて判断します。

## Observationとの関係

Capability invocationが成功しても、外部状態の最終確定を意味しない処理があります。

このIssueはRQ-018のCommand / Observation分離を置き換えません。扱うのは、**一回のCapability invocationが何を返したか**の意味境界です。

外部世界で実際に何が成立したかを確かめるObservationは、必要な処理では別に残します。

## 実装順

1. Authenticationのsuccess / denial / execution errorをReferenceで具体化する
2. 案A〜Cを最小実装で比較する
3. Driverから結果categoryを区別できることを確認する
4. `sales.order.submit`等の別Capabilityへ適用する
5. 不自然なAuthentication専用抽象がないか監査する
6. Conformanceで境界を固定する

## 非目標

- 一般的な例外処理言語の設計
- workflow engineの結果model
- あらゆる非同期処理の統一
- HTTP status codeとの一対一対応
- Authentication専用の場当たり的error規約
- RQ-018のObservationをCapability resultへ統合すること

## 完了条件

次を説明できる小さな契約とReferenceが成立することを完了条件とします。

- 業務上の成功と不成立を区別できる
- 業務上の不成立と実行エラーを区別できる
- 欠損を空文字等のsentinelだけに依存させない
- Driverから各結果を意味の違いとして扱える
- RQ-018のObservation境界と混同しない
- Authentication以外のCapabilityにも無理なく適用できる

## 関連文書

- `plan/LEGACY_AUTH_V1_DESIGN.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md` RQ-012、RQ-018、RQ-021
