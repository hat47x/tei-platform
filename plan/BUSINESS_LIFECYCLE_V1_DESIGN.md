# 業務Lifecycle v1 実装前設計

- 状態: 評価待ち（未採用）
- 日付: 2026-09-05
- 対象Issue: [ISSUE-000038](issue/000038-business-lifecycle-semantics.md)
- 関連ADR: [ADR-0024](adr/0024-business-lifecycle-ownership.md) / [ADR-0025](adr/0025-transition-evaluation-and-conflict-result.md)（いずれも提案中）

この文書は要件の詳細化であり、採用済みの設計基線ではありません。評価が終わるまで、ここに書いた意味モデル、構文、診断、実行順序、結果契約を実装の根拠にしません。

## 目的

この文書は、[`ISSUE-000038`](issue/000038-business-lifecycle-semantics.md)が提案する「業務の状態遷移と実行条件をCanonicalな意味として定義・検証する」機能について、実装を始める前に決めておく必要のある意味境界、authoring、Validation、実行順序、結果契約を確定するための設計です。

対象は`order_entry`の受注確定`pending → submitted`という一つの遷移に限定します。汎用Workflow engineやEvent Sourcingの設計ではありません。

Issueが「何を行うか」を管理するのに対し、この文書は「どの形で実装するか」を実装前に固定します。長く残る判断は[`ADR-0024`](adr/0024-business-lifecycle-ownership.md)と[`ADR-0025`](adr/0025-transition-evaluation-and-conflict-result.md)へ分けています。

## この設計が答える問いと答えない問い

答える問いは次です。

- 状態と遷移の正本をどの層が所有するか
- 状態のidentityと物理的な状態コードをどこで分けるか
- 遷移条件をどこまで宣言的に書き、どこから既存境界へ委ねるか
- Authentication、業務Validation、Authorization、現状態のread、Command、Observationをどの順序で扱うか
- 遷移が成立しなかった場合に、それを業務結果として返すのか実行エラーとして返すのか
- 何をReferenceとConformanceで再現するか

答えない問いは次です。

- 複数の状態属性、承認階層、長時間Workflow、補償処理をどう表すか
- 状態遷移の履歴をどこへ保持するか
- 汎用的な条件式言語をSemanticへ導入するか
- 遷移をAI提案やEditorからどう編集するか

これらは第一Referenceだけでは共通性を確認できません。必要になった時点で、実際のReferenceを根拠に別途判断します。

## 現行実装から確認できる前提

### Semantic modelは意図的に小さい

[`internal/semantic/model.go`](../internal/semantic/model.go)には`Type` / `Data` / `Attribute` / `Relation` / `Capability` / `Validation`しかありません。`Validation`の`Kind`も`positive`と`relation-unique`の二つだけで、汎用式言語はありません。

[`internal/semantic/validationeval/evaluator.go`](../internal/semantic/validationeval/evaluator.go)の評価入口も`Number(attribute)`と`RelationKeys(relation, attribute)`の二つに絞られています。状態値の比較はこの境界では扱えません。

つまり、遷移条件を「既存Validationの拡張」で表そうとすると、評価器の入力境界そのものを広げることになります。本設計はその方向を採りません。

### 状態はすでに存在するが、許容値と遷移は存在しない

[`reference/workspace/semantic/sales/order.tei`](../reference/workspace/semantic/sales/order.tei)には次があります。

```tei
type OrderStatus from std.code

data Order {
    attribute status type sales.order_status
    ...
}
```

`std.code`は表現上の型であり、`pending` / `submitted`という値の集合を持ちません。どの値が正当か、どの順で進むかはRepository内のどこにも書かれていません。

現在その知識は[`submit_order_observation_scenario_test.go`](../internal/driver/submit_order_observation_scenario_test.go)のsynthetic adapterの中にだけあり、Go実装を読まなければ分かりません。

### Capabilityは単一outputか名前付きOutcomeの排他

[`internal/semantic/capabilityinvoke`](../internal/semantic/capabilityinvoke)の`ValidateBusinessResult`は、`output`契約と`outcome`契約を排他として検査します。`sales.order.submit`は現在`output sales.order`の単一output契約です。

`security.authenticate` / `security.authorize`はすでに名前付きOutcome契約であり、業務上の拒否をExecution Errorへ落とさない形が実証されています。

### Bindingのidentityは Application + Capability

[`internal/realization/binding.go`](../internal/realization/binding.go)の`BindingIdentity`は`application/capability`です。新しいID体系を増やさずに「どのApplicationがどの実装を選んだか」を表す既存の軸がここにあります。

### CommandとObservationはすでに分離されている

[`ADR-0023`](adr/0023-command-observation-explicit-read-boundary.md)と[`internal/driver/observation.go`](../internal/driver/observation.go)により、Command Business Resultと別操作の明示readは分離済みです。本設計はこの境界を作り直さず利用します。

### 物理的な状態値はPersistence側にある

[`reference/workspace/databases/sales-core/persistence/order.tei`](../reference/workspace/databases/sales-core/persistence/order.tei)では`sales.order.status`が`status storage varchar`へ対応付けられています。列の存在と型は分かりますが、`"pending"`という値がどの業務状態を指すかは分かりません。

## v1で扱う業務例

```text
sales.order.status
   pending ──[ sales.order.submit ]──▶ submitted
```

取消、承認階層、差戻し、複数の状態属性、期限による自動遷移は初回に含めません。`pending`と`submitted`という現在のfixture上の値を、TEI全体の共通語彙として固定することもしません。

## 意味モデル

### Lifecycleの正本はSemantic側に置く

状態と遷移は、画面やDriverの都合ではなく業務そのものの意味です。同じ受注に対して、Driver、第一Web Target、将来の第二Targetが同じ規則を使う必要があります。

したがってLifecycleはSemantic namespaceが所有する要素とし、`semantic.Namespace`へ追加します。Application Pattern kindを増やす案は採りません。理由と却下した代案は[`ADR-0024`](adr/0024-business-lifecycle-ownership.md)に記録します。

### Canonical model

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

`Namespace`へ`Lifecycles []Lifecycle`を追加します。[`internal/ir`](../internal/ir)の`World`にはfieldを追加しません。Lifecycleは既存の`Semantics`の内側に入るため、World構造の階層を増やさずに済みます。

`From`を配列にするのは、`submitted`と`accepted`の双方から`cancelled`へ進むような形が業務では普通に現れるためです。v1のReferenceでは1件だけを使いますが、Canonical modelを1対1へ固定して後から作り直すことは避けます。

### 状態のidentityと物理コードを分ける

状態は`"pending"`という文字列ではなくCanonical IDで識別します。

```text
sales.order.status_lifecycle.pending
sales.order.status_lifecycle.submitted
```

物理DBに入っている`"pending"`、legacy systemの`"P"`、外部APIの`"PENDING"`は、いずれもこのidentityに対する実現側の表現です。Canonicalへ物理コードを混ぜないことで、異なる状態コードを持つAdapterでも同じ意味を保てます（RQ-001 / RQ-007）。

`OrderStatus from std.code`を列挙型へ変更する案は採りません。型は表現、許容値集合はLifecycleの所有物として分けます。同じ`std.code`を使う別属性へ、この受注固有の値集合が波及することを避けるためです。

### 遷移条件は既存Validationの明示参照に限る

`requires`には既存の`semantic.Validation`のCanonical IDだけを書けます。新しい条件式構文は導入しません。

これにより、遷移条件の評価は[`validationeval.Evaluate`](../internal/semantic/validationeval/evaluator.go)をそのまま使えます。評価の成立・不成立と、評価そのものを完了できないerrorの分離も既存の形を継承できます。

「遷移元が`pending`であること」は`requires`へ書きません。それはLifecycle構造そのものが表しており、二重定義になります。

### Authorizationを書かない

「誰がこの遷移を実行してよいか」はLifecycleへ書きません。[`ADR-0019`](adr/0019-authentication-capability-boundary.md)の`security.authorize`がすでに所有しています。Lifecycleへ複製すると認可規則の正本が二つになります（RQ-003）。

## Authoring DSL

### 基本構文

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

既存の`validation`宣言と同じく、`lifecycle`は`namespace`直下の宣言として[`internal/syntax/parser.go`](../internal/syntax/parser.go)へ追加します。Canonical IDはnamespace prefixを必須とし、宣言されたnamespaceに属さないIDを拒否します。

`state`と`transition`のCanonical IDはlocal symbolから導出します。

```text
state       : <lifecycle-id>.<symbol>
transition  : <lifecycle-id>.<symbol>
```

`state`と`transition`は同じ名前空間を共有するため、同一lifecycle内でsymbolが衝突した場合は診断します。

### 宣言順に意味を持たせない

`state`の宣言順は業務上の順序ではありません。順序は`transition`の`from` / `to`だけが表します。NormalizationではCanonical ID順へ整列し、Referenceの並べ替えだけでCanonical meaningが変わらないことをConformanceで確認します。

### Applicationは第二の遷移正本を持たない

`pattern order_entry.submit_order command { capability sales.order.submit }`は変更しません。

Application Patternへ`transition` fieldを追加すると、「どの遷移を起こすか」の正本がSemanticとApplicationの二か所になります。DriverはCapability identityからtransitionを解決します。

そのために、Workspace全体で次の制約を置きます。

> 一つのCapabilityを`capability`として参照するtransitionは、高々一つである。

この制約により、Capabilityからtransitionへの解決が決定論的になります。同じCapabilityで複数の遷移を起こす業務が現れた場合は、その時点でApplication側に選択を持たせるかどうかを判断します。先回りしてfieldを増やしません。

### まだ導入しない構文

- 状態ごとの説明文、表示名、色などTarget表現に属する情報
- `guard`、`when`、条件式
- `on-entry` / `on-exit`のような副作用hook
- 初期状態と終了状態の宣言
- 状態の階層化・並行状態

初期状態は「新規作成時にどの状態を書くか」というrecord側の関心であり、遷移の意味とは別です。v1では扱いません。

## Workspace Validation

load時に次を検査します。診断codeは[`internal/validation/semantic_business_validation.go`](../internal/validation/semantic_business_validation.go)の既存命名に揃えます。Subject Kindは`semantic-lifecycle`です。

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

Application側では、[`business_validation_reference.go`](../internal/validation/business_validation_reference.go)の既存検査と同じ考え方で次を追加します。

| 診断code | 検出する状態 |
| --- | --- |
| `APPLICATION_LIFECYCLE_SEMANTIC_NOT_USED` | command Patternが参照するCapabilityのtransitionが属するnamespaceを`uses semantic`していない |

`LIFECYCLE_ATTRIBUTE_NOT_CODE`は、既存の`positive`が数値型をalias chain経由で判定しているのと同じ形で、状態属性の型が[`std.code`](../internal/stdlib/semantic.go)へ解決できるかを検査します。`std.text`や`std.identifier`を状態属性として許すかは、実際に必要になるまで広げません。

`requires`の妥当範囲は、対象Dataの属性を扱う`positive`と、対象DataのRelationを扱う`relation-unique`に限ります。無関係なDataのValidationを遷移条件にできてしまうと、実行時に評価入力を用意できない設計を静的に見逃します。

## Normalization / Canonical World / 生成物

- Lifecycleのidentityにfile pathを含めません。`sales/order.tei`と`sales/lifecycle.tei`のどちらへ書いても同じCanonical meaningになります（RQ-002）。
- `Lifecycles`は`omitempty`とし、Lifecycleを持たないWorkspaceの`world.json`をbyte単位で変えません。既存のgenerated artifact Evidenceへ不要な差分を出さないためです。
- Reference WorkspaceへLifecycleを追加する変更は`world.json`のgoldenを更新します。[`ISSUE-000025`](issue/000025-reference-generated-artifacts-refresh.md)の手順に従い、`-write`直後のdirty treeを完了Evidenceとせず、修復差分をcommitしたclean revisionでcheck-onlyを再実行します。

## Source IndexとDiagnostic

[`internal/syntax/semantic_anchor.go`](../internal/syntax/semantic_anchor.go)へ次のanchor kindを追加します。

```text
semantic-lifecycle
semantic-lifecycle-state
semantic-lifecycle-transition
```

これにより、上記の診断はすべてauthoring sourceの位置へ戻れます。file moveやrenameでidentityと意味が変わらないことは、既存のtraceability Conformanceと同じ形で確認します。

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

検査は次のとおりです。

- Bindingが存在するのにLifecycleのstateを網羅していない場合は診断する（`LIFECYCLE_STATE_CODE_BINDING_INCOMPLETE`）
- 同じcode文字列を複数のstateへ対応付けている場合は診断する（`DUPLICATE_LIFECYCLE_STATE_CODE`）
- Bindingが存在しないApplicationは診断しない。遷移を実行しないApplicationへMappingを強制しない

この所有位置は暫定です。同じMappingがDatabase Contract側でも必要になった場合、Persistence側が正本になる可能性を残します。未決であることは[`ADR-0024`](adr/0024-business-lifecycle-ownership.md)へ明記します。

## Driver実行境界

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

現状態のreadをAuthorizationの後に置くことが、この設計でもっとも意識的な選択です。理由は次の三つです。

1. 未認可の主体に対して対象データのreadを実行しません。read自体が情報の露出になり得ます。
2. [`submit_order_gate_scenario_test.go`](../internal/driver/submit_order_gate_scenario_test.go)が実証しているAuthentication → Validation → Authorization → Commandの順序を壊しません。Gate B Evidenceの前提を変えずに済みます。
3. `requires`のValidationは入力内容の妥当性であり、現状態を知らなくても評価できます。

判断の背景は[`ADR-0025`](adr/0025-transition-evaluation-and-conflict-result.md)に残します。

### 判定結果のGo API

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

transitionが存在しない場合と複数存在する場合は、業務結果ではなく実行準備上の失敗としてerrorを返します。[`ResolveCustomGoBinding`](../internal/realization/binding.go)と同じ扱いです。

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

### 事前readは遷移の保証ではない

from判定を通過したことは、Commandが実行される瞬間にも同じ状態である保証にはなりません。

原子的な条件付き更新（現在の状態コードが期待値である場合にだけ更新する）はAdapter / Provider側の責務とします。Driverは「事前readで許可されたから成立した」と扱いません。条件を守れない実装を、意味上同じ遷移の実装として暗黙に受け入れません。

### 遷移が成立しなかった場合の結果契約

条件付き更新が成立しなかった状態は、実行を完了できなかった状態ではありません。処理は正常に終わり、業務としては「確定しなかった」という結末です。したがってExecution Errorにはしません。

現在の`sales.order.submit`は単一output契約なので、この結末を表せません。名前付きOutcome契約へ変更します。

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

[`ADR-0020`](adr/0020-capability-business-outcome-boundary.md)は「事前Validationの不成立をOutcomeへ二重定義しない」ことと「理由なく全CapabilityをOutcomeへ変更しない」ことを求めています。ここで扱うのは事前に判定できない実行時の結末であり、二重定義ではありません。変更の理由は本設計と`ADR-0025`に残します。

なお、この変更はpayloadを持たないOutcomeが自然に見える最初の実例でもあります。v1では`std.unit`もpayload省略構文も導入せず、domain側の明示的なresult dataを使います。ADR-0020が残した論点への入力として記録します。

### 再実行しない

Command結果が不明な場合に、同じ遷移を自動で再実行しません（RQ-020）。Command結果不明、Observation失敗、期待不一致は別々に保持します。観測できないことを業務status上の`UNKNOWN`へ書き換えることもしません。

## Command後のObservation

Command Business Resultを得た後の状態確認は、既存の[`ExecuteObservationRead`](../internal/driver/observation.go)をそのまま使います。Observation専用のPatternやCapabilityは追加しません。

Command payloadから観測状態を作らないこと、readの失敗と期待不一致を区別することはADR-0023のとおりです。本設計はそこへ「観測した状態コードをCanonical stateへ解決する」段を足すだけです。解決できないコードは前述のUnresolvedとして保持します。

## 第一Web Targetとの共有

第一Web Targetは`TransitionEvaluation`と`BusinessResult`を受け取ります。画面側へ次を複製しません。

- どの状態からどの状態へ進めるか
- 遷移に必要なValidation
- 状態コードの文字列

Gate Cでは、同じCanonical規則をDriverとWebが利用していること、単一正本の条件変更が双方の結果へ反映されることを、[`GATE_C_EVIDENCE_CONTRACT.md`](GATE_C_EVIDENCE_CONTRACT.md)のSemantic / Functional / Visualの区分に沿って確認します。表示できたことをSemantic Conformanceの合格にしません。

## Reference / Conformance Evidence

追加するReferenceは次です。

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

決定論的なDriver fixtureと、隔離した実Provider上の条件付き更新は別のEvidenceとして扱います。実Provider検証が未実施なら該当条件は未完了のまま残し、synthetic成功を実DBや本番運用の保証へ昇格させません。

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

手順6はGate B Evidenceの前提を変えます。[`plan/README.md`](README.md)の現行scopeどおり、Gate Bの必須Evidence三系統は同じcandidate revisionへ揃える必要があるため、**手順6はGate B Ready判定と[`ISSUE-000030`](issue/000030-gate-b-to-c-handover-baseline.md)のHandover Baseline固定より前に入れません**。

手順1から5もGate Bの完了条件を増やすものではありません。ISSUE-000038の着手条件どおり、設計と最小Referenceは並走できますが、Gate判定の対象を広げません。

## ISSUE-000038の完了条件との対応

| 完了条件 | 対応する設計 |
| --- | --- |
| 意味・正本・所有境界・競合時の結果契約を記録し、ADRを残す | 本文書、ADR-0024、ADR-0025 |
| 状態・遷移・参照Ruleのauthoringとload → Canonical → Source Index → Diagnostic | Authoring DSL、Workspace Validation、Source Index |
| 正常遷移・未許可の遷移元・Validation不成立・Authorization拒否・未定義状態・Binding不足の区別 | Driver実行境界、Conformance 1〜6 |
| 前提データ取得不能時に実行せず、Execution Errorと業務拒否を区別する | 判定できない状態はExecution Error |
| 再確定で更新せず、同時2要求で成立が高々1回 | 事前readは保証ではない、Conformance 2 / 7 |
| acknowledgementだけで完了にせず、別readで確認する | Command後のObservation、Conformance 8 |
| DriverとWebが同じCanonical規則を使う | 第一Web Targetとの共有 |
| file移動で意味が変わらず、異なる状態コードでも同じ意味を保つ | 状態identityと物理コードの分離、Conformance 9 / 10 |
| 固定revision、Scenario、Canonical ID、before / afterのEvidence | Reference / Conformance Evidence、実装順 |

## Product Requirementとの対応

| 要求 | この設計での扱い |
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

- 汎用Workflow engine、BPMN、承認製品、Event Sourcing基盤
- 分散transaction、自動補償、汎用idempotency台帳、exactly-once
- 任意のSQL / Goから状態機械を推測すること
- AI、Editor、Web専用の別正本や別権限経路
- 既存Gate Bの再定義、既存認証・Command / Observation実装の再起票

## 関連文書

- [`ISSUE-000038`](issue/000038-business-lifecycle-semantics.md)
- [`ADR-0024`](adr/0024-business-lifecycle-ownership.md) / [`ADR-0025`](adr/0025-transition-evaluation-and-conflict-result.md)
- [`ADR-0017`](adr/0017-application-pattern-ownership.md) / [`ADR-0019`](adr/0019-authentication-capability-boundary.md) / [`ADR-0020`](adr/0020-capability-business-outcome-boundary.md) / [`ADR-0023`](adr/0023-command-observation-explicit-read-boundary.md)
- [`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md) / [`CAPABILITY_OUTCOME_V1_DESIGN.md`](CAPABILITY_OUTCOME_V1_DESIGN.md) / [`LEGACY_AUTH_V1_DESIGN.md`](LEGACY_AUTH_V1_DESIGN.md)
- [`requirements/TEI_PRODUCT_REQUIREMENTS.md`](../requirements/TEI_PRODUCT_REQUIREMENTS.md)
