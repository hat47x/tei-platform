# ISSUE-000038: 業務の状態遷移と実行条件をCanonicalな意味として定義・検証する

- 状態: Proposed
- 提案日: 2026-09-04
- 要件詳細化: 2026-09-05（評価待ち。採用が決まるまで実装やReference変更の根拠にしません）
- 提案順位: 1（根幹機能の新規提案内）
- 着手条件: 最小Scenarioと所有境界の設計は現行主線と並走可能。本格実装はGate C後に基線を固定して行い、Gate B / Cの既存完了条件を増やさない
- 関連RQ: RQ-001 / RQ-002 / RQ-003 / RQ-008 / RQ-012 / RQ-014 / RQ-018 / RQ-019 / RQ-020 / RQ-021
- 設計判断: [ADR-0024](../adr/0024-business-lifecycle-ownership.md) / [ADR-0025](../adr/0025-transition-evaluation-and-conflict-result.md)（いずれも提案中）

## 提供する根幹機能

「どの業務状態から、どの条件で、どの操作によって、どの状態へ進めるか」を明示的に定義し、DriverとTargetが同じ規則を利用できるようにします。業務ルールをGo実装や画面イベントへ閉じ込めず、変更箇所、適用条件、実行結果、観測結果へ追跡できることが目的です。

受注確定、承認、取消等の状態管理は、業務システムの中心機能です。最初は受注確定の一つの遷移に絞り、他の業務へ一般化できる意味と固有の実現を分けます。

## 現在あるものと不足

- [Order定義](../../reference/workspace/semantic/sales/order.tei)には`OrderStatus from std.code`と`sales.order.status`がありますが、許容状態と遷移規則のauthoringはありません。`std.code`は[標準型](../../internal/stdlib/semantic.go)の表現であり、値の集合を持ちません。
- [Semantic model](../../internal/semantic/model.go)にはData / Relation / Capability / Validationがあります。既存Validationは属性の正数判定やRelation内の一意性を扱い、参照整合性の検査は[Workspace Validation](../../internal/validation/semantic_business_validation.go)、実データの評価は[Evaluator](../../internal/semantic/validationeval/evaluator.go)が担当します。Evaluatorの入力境界は`Number(attribute)`と`RelationKeys(relation, attribute)`だけで、状態値の比較は扱えません。
- `sales.order.submit`は`output sales.order`の単一output契約です。[capabilityinvoke](../../internal/semantic/capabilityinvoke)は`output`契約と名前付き`outcome`契約を排他として検査します。
- [submit Scenario](../../internal/driver/submit_order_observation_scenario_test.go)はsynthetic adapter内で状態を`submitted`へ変更します。遷移元の条件と期待する遷移を業務定義から解決する機能は、このGo処理だけでは成立しません。
- [Observation境界](../../internal/driver/observation.go)と[ADR-0023](../adr/0023-command-observation-explicit-read-boundary.md)により、CommandのBusiness Resultと別操作のObservationは既に分離されています。本Issueではこの境界を利用します。
- [Binding identity](../../internal/realization/binding.go)は`application/capability`です。「どのApplicationがどの実装を選んだか」を表す既存の軸がここにあります。
- [Persistence Mapping](../../reference/workspace/databases/sales-core/persistence/order.tei)は`sales.order.status`を`status storage varchar`へ対応付けます。列と型は分かりますが、`"pending"`という値がどの業務状態を指すかは分かりません。
- [ISSUE-000002](000002-v1-application-vertical-slice.md)には最小state transitionの要望があります。本Issueはその業務意味と実行への接続を具体化します。[Master PlanのWorkstream B](../MASTER_PLAN.md)も状態遷移等から再利用可能な意味知識を実証する方針です。

追加するのは、認証、Validation評価器、Command呼出し、Observation readの作り直しではなく、それらを**業務上の遷移規則へ明示的に結び付ける機能**です。Deployment / Plugin RuntimeのLifecycleを、そのまま業務状態の意味へ転用しません。

## 最小の縦切り

既存fixtureに合わせ、`order_entry`の受注確定による`pending → submitted`を候補にします。現在の状態文字列を直ちに全業務共通の語彙へ固定せず、許容状態と業務上の意味を設計時に明示します。取消、承認階層、長時間Workflowは初回に含めません。

```text
sales.order.status
   pending ──[ sales.order.submit ]──▶ submitted
```

1. 状態と遷移のidentity、対象Data / status attribute、遷移元・先、実行Capability、適用する既存Validationをsource上で明示する。
2. Authorizationと業務上の遷移可能性を分ける。認可規則をLifecycleへ複製せず既存境界へ接続する。
3. Workspace loadと共通Validationで参照欠落、異なるDataへの誤結合、未定義状態、競合する遷移候補を検出し、Sourceへ戻れるDiagnosticを返す。
4. Driverが現状態と必要な入力から遷移条件を評価する。条件不成立と、入力を取得できず判定できない状態を区別する。
5. 明示Binding先のCapabilityを利用し、Business Resultと、別readで得た状態・期待状態との照合を別々に返す。readの選択と状態コードの物理MappingはRealization側へ置く。
6. 同じ規則を第一Web Targetから利用し、画面側への業務Rule複製を必要としないことを確認する。

正本の所有者は、Semantic側のLifecycle / ConstraintとApplication側の利用意図を比較して決め、広い設計判断はADRへ残します。`StateMachine`専用Core kindや汎用式言語を初めから決めず、既存の小さい意味モデルと参照で実現できる範囲を先に確かめます。

以降は、2026-09-05時点でこの6項目を実装可能な粒度へ詳細化した内容です。所有境界はADR-0024、実行境界と結果契約はADR-0025として提案しており、いずれも評価待ちです。

## 意味モデル

Lifecycleは、Semantic namespaceが所有する要素として`semantic.Namespace`へ追加します。Application Pattern kindを増やす案を採らない理由はADR-0024に記録しています。

```go
// internal/semantic/model.go への追加案
type LifecycleState struct {
    ID     core.CanonicalID `json:"id"`
    Symbol string           `json:"symbol"`
}

type LifecycleTransition struct {
    ID         core.CanonicalID   `json:"id"`
    Symbol     string             `json:"symbol"`
    From       []core.CanonicalID `json:"from"`
    To         core.CanonicalID   `json:"to"`
    Capability core.CanonicalID   `json:"capability"`
    Requires   []core.CanonicalID `json:"requires,omitempty"`
}

type Lifecycle struct {
    ID          core.CanonicalID      `json:"id"`
    Symbol      string                `json:"symbol"`
    Data        core.CanonicalID      `json:"data"`
    Attribute   core.CanonicalID      `json:"attribute"`
    States      []LifecycleState      `json:"states"`
    Transitions []LifecycleTransition `json:"transitions"`
}
```

`Namespace`へ`Lifecycles []Lifecycle`を追加します。[`internal/ir`](../../internal/ir)の`World`にはfieldを追加しません。Lifecycleは既存の`Semantics`の内側に入るため、World構造の階層を増やさずに済みます。

`From`を配列にするのは、`submitted`と`accepted`の双方から`cancelled`へ進むような形が業務では普通に現れるためです。v1のReferenceでは1件だけを使いますが、Canonical modelを1対1へ固定して後から作り直すことは避けます。

### 状態のidentityと物理コードを分ける

状態は`"pending"`という文字列ではなくCanonical IDで識別します。

```text
sales.order.status_lifecycle.pending
sales.order.status_lifecycle.submitted
```

物理DBの`"pending"`、legacy systemの`"P"`、外部APIの`"PENDING"`は、いずれもこのidentityに対する実現側の表現です。Canonicalへ物理コードを混ぜないことで、異なる状態コードを持つAdapterでも同じ意味を保てます（RQ-001 / RQ-007）。

`OrderStatus from std.code`を列挙型へ変更する案は採りません。型は表現、許容値集合はLifecycleの所有物として分けます。同じ`std.code`を使う別属性へ、この受注固有の値集合が波及することを避けるためです。

### 遷移条件は既存Validationの明示参照に限る

`requires`には既存の`semantic.Validation`のCanonical IDだけを書けます。新しい条件式構文は導入しません。これにより、遷移条件の評価は既存の`validationeval.Evaluate`をそのまま使え、成立・不成立と評価不能の分離も既存の形を継承できます。

「遷移元が`pending`であること」は`requires`へ書きません。それはLifecycle構造そのものが表しており、二重定義になります。

「誰がこの遷移を実行してよいか」も書きません。[ADR-0019](../adr/0019-authentication-capability-boundary.md)の`security.authorize`が既に所有しています。Lifecycleへ複製すると認可規則の正本が二つになります（RQ-003）。

## Authoring

```tei
namespace sales

lifecycle sales.order.status_lifecycle {
    data sales.order
    attribute sales.order.status

    state pending
    state submitted

    transition submit {
        from pending
        to submitted
        capability sales.order.submit
        requires sales.order_line.quantity_positive
        requires sales.order.lines.line_number_unique
    }
}
```

既存の`validation`宣言と同じく、`lifecycle`は`namespace`直下の宣言として[Parser](../../internal/syntax/parser.go)へ追加します。Canonical IDはnamespace prefixを必須とし、宣言されたnamespaceに属さないIDを拒否します。

`state`と`transition`のCanonical IDはlocal symbolから導出します。

```text
state       : <lifecycle-id>.<symbol>
transition  : <lifecycle-id>.<symbol>
```

`state`と`transition`は同じ名前空間を共有するため、同一lifecycle内でsymbolが衝突した場合は診断します。

`state`の宣言順は業務上の順序ではありません。順序は`transition`の`from` / `to`だけが表します。NormalizationではCanonical ID順へ整列し、Referenceの並べ替えだけでCanonical meaningが変わらないことをConformanceで確認します。

### Applicationは第二の遷移正本を持たない

`pattern order_entry.submit_order command { capability sales.order.submit }`は変更しません。

Application Patternへ`transition` fieldを追加すると、「どの遷移を起こすか」の正本がSemanticとApplicationの二か所になります。DriverはCapability identityからtransitionを解決します。そのために、Workspace全体で次の制約を置きます。

> 一つのCapabilityを`capability`として参照するtransitionは、高々一つである。

この制約により、Capabilityからtransitionへの解決が決定論的になります。同じCapabilityで複数の遷移を起こす業務が現れた場合は、その時点でApplication側に選択を持たせるかどうかを判断します。先回りしてfieldを増やしません。

### まだ導入しない構文

- 状態ごとの説明文、表示名、色などTarget表現に属する情報
- `guard`、`when`、条件式
- `on-entry` / `on-exit`のような副作用hook
- 初期状態と終了状態の宣言
- 状態の階層化・並行状態

初期状態は「新規作成時にどの状態を書くか」というrecord側の関心であり、遷移の意味とは別です。v1では扱いません。

## Workspace Validationと診断

load時に次を検査します。診断codeは既存のSemantic Validation検査の命名に揃えます。Subject Kindは`semantic-lifecycle`です。

| 診断code | 検出する状態 |
| --- | --- |
| `DUPLICATE_SEMANTIC_ELEMENT` | 同じCanonical IDが他のSemantic要素としても宣言されている |
| `DUPLICATE_SEMANTIC_LIFECYCLE` | 同じLifecycle IDが複数のnamespaceで宣言されている |
| `SEMANTIC_LIFECYCLE_NAMESPACE_MISMATCH` | Lifecycle IDが宣言namespaceに属していない |
| `UNKNOWN_LIFECYCLE_DATA` | `data`が未知のSemantic Dataを指している |
| `UNKNOWN_LIFECYCLE_ATTRIBUTE` | `attribute`が未知のAttributeを指している |
| `LIFECYCLE_ATTRIBUTE_OUTSIDE_DATA` | `attribute`が`data`の所有属性ではない |
| `LIFECYCLE_ATTRIBUTE_NOT_CODE` | 状態属性の型がcode系へ解決できない |
| `DUPLICATE_LIFECYCLE_FOR_ATTRIBUTE` | 同じAttributeに複数のLifecycleが宣言されている |
| `DUPLICATE_LIFECYCLE_STATE` | 同一Lifecycle内でstate symbolが重複している |
| `LIFECYCLE_REQUIRES_STATES` | stateが宣言されていない |
| `DUPLICATE_LIFECYCLE_TRANSITION` | 同一Lifecycle内でtransition symbolが重複している |
| `UNKNOWN_LIFECYCLE_TRANSITION_FROM` | `from`が未定義のstateを指している |
| `UNKNOWN_LIFECYCLE_TRANSITION_TO` | `to`が未定義のstateを指している |
| `LIFECYCLE_TRANSITION_FROM_EQUALS_TO` | `from`と`to`が同じstateである |
| `DUPLICATE_LIFECYCLE_TRANSITION_FROM` | 同一transitionの`from`に同じstateが重複している |
| `UNKNOWN_LIFECYCLE_TRANSITION_CAPABILITY` | `capability`が未知のCapabilityを指している |
| `AMBIGUOUS_LIFECYCLE_TRANSITION_CAPABILITY` | 同じCapabilityを複数のtransitionが参照している |
| `UNKNOWN_LIFECYCLE_TRANSITION_VALIDATION` | `requires`が未知のValidationを指している |
| `LIFECYCLE_TRANSITION_VALIDATION_OUTSIDE_DATA` | `requires`が対象Dataとその関係先以外のValidationを指している |

Application側では、[既存のValidation参照検査](../../internal/validation/business_validation_reference.go)と同じ考え方で次を追加します。

| 診断code | 検出する状態 |
| --- | --- |
| `APPLICATION_LIFECYCLE_SEMANTIC_NOT_USED` | command Patternが参照するCapabilityのtransitionが属するnamespaceを`uses semantic`していない |

`LIFECYCLE_ATTRIBUTE_NOT_CODE`は、既存の`positive`が数値型をalias chain経由で判定しているのと同じ形で、状態属性の型が`std.code`へ解決できるかを検査します。`std.text`や`std.identifier`を状態属性として許すかは、実際に必要になるまで広げません。

`requires`の妥当範囲は、対象Dataの属性を扱う`positive`と、対象DataのRelationを扱う`relation-unique`に限ります。無関係なDataのValidationを遷移条件にできてしまうと、実行時に評価入力を用意できない設計を静的に見逃します。

## Normalization / Source Index / 生成物

- Lifecycleのidentityにfile pathを含めません。`sales/order.tei`と`sales/lifecycle.tei`のどちらへ書いても同じCanonical meaningになります（RQ-002）。
- `Lifecycles`は`omitempty`とし、Lifecycleを持たないWorkspaceの`world.json`をbyte単位で変えません。既存のgenerated artifact Evidenceへ不要な差分を出さないためです。
- [Semantic anchor](../../internal/syntax/semantic_anchor.go)へ`semantic-lifecycle` / `semantic-lifecycle-state` / `semantic-lifecycle-transition`を追加し、上記の診断がすべてauthoring sourceの位置へ戻れるようにします。
- Reference WorkspaceへLifecycleを追加する変更は`world.json`のgoldenを更新します。[ISSUE-000025](000025-reference-generated-artifacts-refresh.md)の手順に従い、`-write`直後のdirty treeを完了Evidenceとせず、修復差分をcommitしたclean revisionでcheck-onlyを再実行します。

## Realization: 状態コードのMapping

Canonicalな状態identityと、実装が実際に読み書きする文字列を対応付ける明示Bindingを置きます。

```tei
lifecycle-state-codes {
    application order_entry
    lifecycle sales.order.status_lifecycle
    state sales.order.status_lifecycle.pending -> "pending"
    state sales.order.status_lifecycle.submitted -> "submitted"
}
```

identityは既存の`BindingIdentity`と同じ考え方で`application/lifecycle`とします。同じSemanticを利用する別Applicationが異なる物理コードを使う可能性を塞がないためです。

- Bindingが存在するのにLifecycleのstateを網羅していない場合は診断する（`LIFECYCLE_STATE_CODE_BINDING_INCOMPLETE`）
- 同じcode文字列を複数のstateへ対応付けている場合は診断する（`DUPLICATE_LIFECYCLE_STATE_CODE`）
- Bindingが存在しないApplicationは診断しない。遷移を実行しないApplicationへMappingを強制しない

この所有位置は暫定です。同じMappingがDatabase Contract側でも必要になった場合、Persistence側が正本になる可能性を残します。未決であることはADR-0024へ明記しています。

## 実行と整合性の境界

事前read後に別の主体が状態を変える競合を、事前チェックだけで防げるとは扱いません。最小Referenceでは同一の状態・版に対する原子的な条件付き遷移をAdapter / Provider側で成立させます。条件を守れない実装を、意味上同じ遷移実装として暗黙に採用しません。

二重・並行確定を検証しますが、任意の外部副作用に対するexactly-onceは保証しません。Command結果不明、Observation失敗、期待不一致は別々に保持し、自動再実行で埋め合わせません。観測不能を業務statusの`UNKNOWN`へ書き換えることもしません。

### 評価順序

```text
Authentication
    ↓
Canonical業務Validation（requires）
    ↓
Authorization
    ↓
現状態の明示read
    ↓
from判定
    ↓
条件付きCommand（Capability invocation）
    ↓
別操作としてのObservation read
```

現状態のreadをAuthorizationの後に置くことが、もっとも意識的な選択です。理由は次の三つです。

1. 未認可の主体に対して対象データのreadを実行しません。read自体が情報の露出になり得ます。
2. [submit Gate Scenario](../../internal/driver/submit_order_gate_scenario_test.go)が実証しているAuthentication → Validation → Authorization → Commandの順序を壊しません。Gate B Evidenceの前提を変えずに済みます。
3. `requires`のValidationは入力内容の妥当性であり、現状態を知らなくても評価できます。

判断の背景はADR-0025に残しています。

### 判定結果のGo API案

```go
// internal/driver への追加案
type TransitionDecision string

const (
    TransitionAllowed  TransitionDecision = "allowed"
    TransitionRejected TransitionDecision = "rejected"
)

type TransitionRejectionReason string

const (
    RejectionStateNotAllowed        TransitionRejectionReason = "state-not-allowed"
    RejectionValidationNotSatisfied TransitionRejectionReason = "validation-not-satisfied"
    RejectionAuthorizationDenied    TransitionRejectionReason = "authorization-denied"
)

type TransitionEvaluation struct {
    Transition    core.CanonicalID
    Capability    core.CanonicalID
    ObservedState core.CanonicalID // 解決できた場合だけ設定する
    Decision      TransitionDecision
    Reason        TransitionRejectionReason
    Validation    map[core.CanonicalID]validationeval.Result
}
```

Capabilityからtransitionを解決する入口は、既存の`ResolveCapabilityBinding`と同じ姿勢にします。

```go
func (r Resolver) ResolveTransitionForCapability(
    applicationID, capabilityID core.CanonicalID,
) (ResolvedTransition, error)
```

transitionが存在しない場合と複数存在する場合は、業務結果ではなく実行準備上の失敗としてerrorを返します。既存の`ResolveCustomGoBinding`と同じ扱いです。

拒否理由は最初の不成立で確定させ、後続を評価しません。「実行しない理由をすべて集める」設計は、Authorization拒否の後に業務内容の詳細を返してしまう経路を作ります。ただし評価済みのValidation結果は保持し、どこまで進んだかを追跡できるようにします。

### 判定できない状態はExecution Errorとして分ける

readで得た状態コードがどのstateにも対応しない場合、それを「遷移不可」という業務結果へ変換しません。authoringやMappingの不足を業務上の意味へ書き換えることになるためです。

```go
type TransitionUnresolvedReason string

const (
    UnresolvedUnknownStateCode TransitionUnresolvedReason = "unknown-state-code"
    UnresolvedStateUnreadable  TransitionUnresolvedReason = "state-unreadable"
    UnresolvedStateCodeMissing TransitionUnresolvedReason = "state-code-binding-missing"
)
```

これらはerrorとして返し、`Decision`を設定しません。readの失敗（外部障害）と、readはできたが意味へ解決できない状態を区別できるようにします。RQ-006がDatabase側で求めている「情報が不十分なときにUnresolvedを保持する」考え方を、Application側でも同じ形で守ります。

### 遷移が成立しなかった場合の結果契約

現在の`sales.order.submit`は単一`output sales.order`です。[ISSUE-000009](000009-capability-outcome-and-execution-error.md) / [ADR-0020](../adr/0020-capability-business-outcome-boundary.md)に従い、事前Validationの不成立をOutcomeへ二重定義せず、理由なく全Capabilityをnamed Outcomeへ変更しません。

一方、条件付き更新が成立しなかった状態は、実行を完了できなかった状態ではありません。処理は正常に終わり、業務としては「確定しなかった」という結末です。事前に判定できない実行時の結末であり、二重定義にはあたりません。そのため名前付きOutcome契約へ変更します。

```tei
data OrderSubmissionConflict {
    attribute orderId type sales.order_id
}

capability sales.order.submit {
    input order sales.order
    outcome submitted sales.order
    outcome not_submitted sales.order_submission_conflict
}
```

`not_submitted`のpayloadへ現在の状態を入れません。条件付き更新が観測しているのは「期待した遷移元ではなかった」ことだけであり、いま何の状態かは別のreadなしには分かりません（ADR-0023）。`already_submitted`という名前を採らないのも同じ理由です。観測していない事実を結果の名前で主張しません。

この変更は、payloadを持たないOutcomeが自然に見える最初の実例でもあります。v1では`std.unit`もpayload省略構文も導入せず、domain側の明示的なresult dataを使います。ADR-0020が残した論点への入力として記録します。既存契約を変えるため、互換性と参照元への影響は実装順の項に記載します。

## Command後のObservation

Command Business Resultを得た後の状態確認は、既存の`ExecuteObservationRead`をそのまま使います。Observation専用のPatternやCapabilityは追加しません。

Command payloadから観測状態を作らないこと、readの失敗と期待不一致を区別することはADR-0023のとおりです。ここへ「観測した状態コードをCanonical stateへ解決する」段を足すだけです。解決できないコードは前述のUnresolvedとして保持します。

## 第一Web Targetとの共有

第一Web Targetは`TransitionEvaluation`とBusiness Resultを受け取ります。画面側へ次を複製しません。

- どの状態からどの状態へ進めるか
- 遷移に必要なValidation
- 状態コードの文字列

Gate Cでは、同じCanonical規則をDriverとWebが利用していること、単一正本の条件変更が双方の結果へ反映されることを、[Gate C Evidence契約](../GATE_C_EVIDENCE_CONTRACT.md)のSemantic / Functional / Visualの区分に沿って確認します。表示できたことをSemantic Conformanceの合格にしません。

## Reference / Conformance Evidence

追加するReferenceは次です。parserが未対応のうちに置くとWorkspace loadが失敗するため、実装順1が終わるまで追加しません。

```text
reference/workspace/semantic/sales/lifecycle.tei
reference/workspace/applications/order-entry/realizations/lifecycle-state-codes.tei
```

Conformanceは`docs/conformance/business-lifecycle-v1.md`へまとめ、少なくとも次を再現します。

1. 正常遷移: `pending`から`submitted`へ進み、Business Resultが`submitted`になる
2. 未許可の遷移元: `submitted`に対する再確定が`state-not-allowed`で実行されない
3. Validation不成立: `requires`の不成立でAuthorizationにもCommandにも進まない
4. Authorization拒否: 業務結果として拒否され、Execution Errorにならない
5. 未定義状態: 未知の状態コードが`unknown-state-code`のerrorになり、遷移拒否へ変換されない
6. Binding不足: 状態コードMappingの欠落が`state-code-binding-missing`として分離される
7. 競合: 同一受注への同時2要求で`submitted`が高々1回、他方は`not_submitted`になる
8. Observation: acknowledgementだけで完了とせず、別readで状態を確認する。read失敗時は不確実性を保持する
9. identity: file移動でCanonical meaningが変わらない
10. 物理表現の独立: 異なる状態コードのMappingでも同じ意味を保てる

## 実装順とbaselineへの影響

| 順 | 範囲 | baselineへの影響 |
| --- | --- | --- |
| 1 | Semantic model、Parser、anchor（authoringのみ） | なし。既存Worldのbyte表現を変えない |
| 2 | Workspace Validationと診断 | なし |
| 3 | Reference WorkspaceへLifecycle追加、golden再生成 | `world.json`のgoldenが変わる |
| 4 | 状態コードMappingのRealization Binding | Reference追加分のみ |
| 5 | Driverの遷移解決・評価API | 既存Scenarioの順序を変えない |
| 6 | `sales.order.submit`のOutcome契約変更と条件付き更新Adapter | **Capability契約の変更。既存Driver Scenarioとその期待値に影響する** |
| 7 | 第一Web Targetでの共有とGate C Evidenceへの接続 | Gate C側 |

手順6はGate B Evidenceの前提を変えます。[plan/README.md](../README.md)の現行scopeどおり、Gate Bの必須Evidence三系統は同じcandidate revisionへ揃える必要があるため、**手順6はGate B Ready判定と[ISSUE-000030](000030-gate-b-to-c-handover-baseline.md)のHandover Baseline固定より前に入れません**。

手順1から5もGate Bの完了条件を増やすものではありません。着手条件どおり、設計と最小Referenceは並走できますが、Gate判定の対象を広げません。

## 完了条件とEvidence

- [ ] 最小遷移の意味、正本、実行・観測の所有境界と競合時の結果契約を記録し、必要なADRを残した
- [ ] 状態・遷移・参照Ruleをauthoringでき、load → Canonical → Source Index → Diagnosticを追跡できる
- [ ] 正常遷移、未許可の遷移元、Validation不成立、Authorization拒否、未定義状態、Binding不足を区別して再現した
- [ ] 前提データ取得不能時には実行せず、Execution Errorと業務上の拒否を区別する
- [ ] submittedの再確定で更新せず、同一受注への同時2要求で遷移成立は高々1回になり、未成立側を成功や通信障害へ丸めない
- [ ] acknowledgementだけで完了にせず、別readで同一受注の状態を確認する。Observation不一致・read失敗では不確実性を保持する
- [ ] 同じCanonical規則をDriverと第一Web Targetが利用し、単一正本の条件変更が双方の結果へ反映される
- [ ] file移動でidentityと意味が変わらず、異なる物理状態コードのAdapterでも明示Mappingで同じ意味を保てる
- [ ] 固定revision、Scenario、Canonical IDs、before / afterの判定と観測をEvidenceとして残した

決定論的なDriver fixtureと、隔離した実Provider上の条件付き更新を分けて検証し、実Provider検証が未実施なら該当条件を未完了として残します。synthetic成功を実DB・既存外部資産・本番運用の保証へ昇格させません。Gate Bの必須3系統のEvidenceも、継続課題であるactual Oracle Evidenceも代替しません。

利用者価値は、同じ状態条件の変更について手動編集が必要な正本の数と、Driver / Target間のルール不一致を記録して確認します。コードが存在することだけで完了とせず、実行と変更伝播を再現します。

## Product Requirementとの対応

| 要求 | この作業での扱い |
| --- | --- |
| RQ-001 | 状態と遷移をCanonical IDで保持し、物理コードから独立させる |
| RQ-002 | lifecycle / state / transitionのanchorでauthoring sourceへ戻れる |
| RQ-003 | 遷移の正本をSemanticへ一本化し、Application / Target / 画面へ複製しない |
| RQ-008 | 遷移の実現はCapability Binding経由で置き換えられる |
| RQ-012 | Driverで遷移条件と結果を検証できる |
| RQ-014 | command Patternを変えずにLifecycleを接続する |
| RQ-018 | Command Business Resultと別readのObservationを分離する |
| RQ-019 | Authorizationを複製せず既存境界を使う |
| RQ-020 | 競合と不明を区別し、自動replayしない |
| RQ-021 | Reference / Conformanceから再現する |

## まだ固定しないこと

- 複数の状態属性を持つData、状態の階層・並行
- 遷移履歴の保持方法
- 初期状態・終了状態の宣言
- 状態に対する条件式言語
- 遷移から導出するUIやCapabilityの自動生成
- `std.*`へのLifecycle標準語彙の予約
- 状態コードMappingの最終的な所有層

## 非目標

- 汎用Workflow engine、BPMN、承認製品、Event Sourcing基盤の完成
- 分散transaction、自動補償、汎用idempotency台帳、exactly-once保証
- 任意SQL / Goから業務状態機械を推測すること
- AI、Editor、Web専用の別正本・権限経路を作ること
- 既存Gate Bの再定義、既存認証・Command / Observation実装の再起票

## 関連

- [ADR-0024](../adr/0024-business-lifecycle-ownership.md): Lifecycleの所有境界（提案中）
- [ADR-0025](../adr/0025-transition-evaluation-and-conflict-result.md): 遷移の評価順序と成立しなかった遷移の結果契約（提案中）
- [Product Value](../../product/value/TEI_PRODUCT_VALUE.md)、[Product Principles](../../product/principles/TEI_PRODUCT_PRINCIPLES.md)、[Product Requirements](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)
- [Application Patternの所有境界](../adr/0017-application-pattern-ownership.md)、[Business Result](../adr/0020-capability-business-outcome-boundary.md)、[Command / Observation](../adr/0023-command-observation-explicit-read-boundary.md)
- [業務Validation Reference](000014-business-validation-reference.md)
- [ISSUE-000039](000039-business-change-rehearsal.md): 遷移・Rule変更の影響を将来追跡する拡張先。初回実装の必須依存にはしない
- [ISSUE-000040](000040-semantic-component-composition.md): 実証後に再利用可能な意味を探索・組み立てへ接続する。初回実装の必須依存にはしない
- 関連Issue: [ISSUE-000006](000006-application-driver-vertical-slice.md) / [ISSUE-000007](000007-first-web-target-projection.md) / [ISSUE-000009](000009-capability-outcome-and-execution-error.md) / [ISSUE-000014](000014-business-validation-reference.md) / [ISSUE-000015](000015-submit-capability-binding-reference.md) / [ISSUE-000028](000028-first-web-target-conformance-evidence.md) / [ISSUE-000029](000029-gate-b-readiness-audit.md)
