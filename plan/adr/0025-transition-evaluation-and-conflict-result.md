# ADR-0025: 遷移の評価順序を固定し、成立しなかった遷移を業務結果として返す

- 状態: 提案中
- 日付: 2026-09-05
- 関連Issue: [ISSUE-000038](../issue/000038-business-lifecycle-semantics.md)（作業範囲と実装順）
- 関連設計: [業務Lifecycle v1 設計](../design/BUSINESS_LIFECYCLE_V1_DESIGN.md)（評価待ち）

この判断は評価待ちです。採用が決まるまで、実装やCapability契約の変更の根拠にしません。

## 背景

[ADR-0024](0024-business-lifecycle-ownership.md)で、業務Lifecycleの正本をSemanticへ置き、遷移がCapabilityを明示参照する形を決めました。

残るのは実行時の設計です。遷移を実行するとき、少なくとも次を区別する必要があります。

```text
実行してよい
業務上の理由で実行しない        遷移元が違う / Validation不成立 / Authorization拒否
判定できない                    現状態をreadできない / 状態コードを解決できない
実行したが遷移は成立しなかった  条件付き更新が競合した
```

現在のsubmit Scenarioは、[`submit_order_gate_scenario_test.go`](../../internal/driver/submit_order_gate_scenario_test.go)でAuthentication → 業務Validation → Authorization → Commandの順序を実証しています。ここへ「現在の状態を確認する」段が加わります。

同時に、`sales.order.submit`は`output sales.order`の単一output契約です。この契約では「確定しなかった」という結末を表せません。

## 判断

### 評価順序を固定する

```text
Authentication
    ↓
Canonical業務Validation（transitionのrequires）
    ↓
Authorization
    ↓
現状態の明示read
    ↓
from判定
    ↓
条件付きCommand
    ↓
別操作としてのObservation read
```

現状態のreadはAuthorizationの後に置きます。理由は次の三つです。

1. 未認可の主体に対して対象データのreadを実行しません。readそのものが情報の露出になり得ます。
2. 既存Scenarioが実証しているAuthentication → Validation → Authorization → Commandの順序を壊しません。Gate B Evidenceの前提を変えずに済みます。
3. `requires`のValidationは入力内容の妥当性であり、現在の状態を知らなくても評価できます。

### 拒否理由は最初の不成立で確定させる

実行しない理由をすべて集めて返しません。Authorization拒否の後に業務内容の詳細を返す経路を作らないためです。ただし、その時点までに評価済みのValidation結果は保持し、どこまで進んだかを追跡できるようにします。

### 判定できない状態はExecution Errorとして分ける

readで得た状態コードがどのstateへも解決できない場合、それを「遷移できない」という業務結果へ変換しません。authoringやMappingの不足を業務上の意味へ書き換えることになります。

read自体の失敗、未知の状態コード、状態コードMappingの欠落は、いずれも判定を完了できない状態として区別できる形で返します。RQ-006がDatabase側で求める「情報が不十分ならUnresolvedを保持する」考え方を、Application側でも同じ形で守ります。

### 事前readを遷移の保証にしない

from判定を通過したことは、Commandを実行する瞬間にも同じ状態である保証にはなりません。

原子的な条件付き更新（現在の状態コードが期待値である場合にだけ更新する）はAdapter / Provider側の責務とします。Driverは「事前readで許可されたから成立した」と扱いません。条件を守れない実装を、意味上同じ遷移の実装として暗黙に受け入れません。

### 成立しなかった遷移は業務結果として返す

条件付き更新が成立しなかった状態は、実行を完了できなかった状態ではありません。処理は正常に終わり、業務としては「確定しなかった」という結末です。

そのため`sales.order.submit`を名前付きOutcome契約へ変更します。

```text
outcome submitted      sales.order
outcome not_submitted  sales.order_submission_conflict
```

`not_submitted`のpayloadへ現在の状態を入れません。条件付き更新が観測しているのは「期待した遷移元ではなかった」ことだけであり、いま何の状態かは別のreadなしには分かりません。`already_submitted`という名前も採りません。観測していない事実を結果の名前で主張しないためです。

### 自動で再実行しない

Command結果が不明な場合に、同じ遷移を自動でreplayしません。Command結果不明、Observation失敗、期待不一致は別々に保持します。観測できないことを業務status上の`UNKNOWN`へ書き換えることもしません。

## 採用しない案

### 競合をExecution Errorとして返す

Adapterは処理を最後まで実行しています。errorへ畳むと、外部障害による不明状態と、正常に判定された業務上の結末が同じ経路に混ざります。RQ-020が求める「不明を安全に扱う」ことができなくなります。

### 競合をCommand payloadの値で表す

`output sales.order`のまま、payloadの`status`から呼び出し側が結末を推測する案です。ADR-0023が禁じているpayloadからの意味推測そのものであり、採りません。

### `sales.order.submit`をそのままにして、Driver側だけで競合を保持する

Driverが持つ情報は、Web TargetやEditorが同じ契約から受け取れません。業務上の結末をCapability契約の外へ置くと、Targetごとに解釈が分かれます。

### 現状態のreadをAuthorizationより前に置く

readを先に行えば、認可前に「そもそも遷移できない」と早く返せます。しかし未認可の主体に対象データのreadを実行することになり、既存Scenarioの順序も変わります。応答の速さのために信頼境界を緩めません。

### 事前readの結果をもって遷移成立とみなす

事前チェックだけでは、read後Command前に別の主体が状態を変える競合を防げません。「チェックしたから安全」という前提を実装へ埋め込みません。

## 影響

### 良くなる点

- 業務上の拒否、判定不能、実行時競合が別々の結果として残る
- 二重確定と並行確定を、成功へ丸めずに検証できる
- DriverとTargetが同じ結果契約を共有できる
- 観測していない状態を結果へ書かない

### 増える負担

`sales.order.submit`の契約変更は、既存のDriver Scenarioとその期待値に影響します。Gate Bの必須Evidenceは同じcandidate revisionへ揃える必要があるため、**この契約変更はGate B Ready判定と[ISSUE-000030](../issue/000030-gate-b-to-c-handover-baseline.md)のHandover Baseline固定より前に入れません**。

また、payloadを持たないOutcomeが自然に見える最初の実例でもあります。v1では`std.unit`もpayload省略構文も導入せず、domain側の明示的なresult dataを使います。ADR-0020が残した論点への入力として記録します。

## 検証

ISSUE-000038で、第一Referenceに対して次を確認します。

1. 正常遷移でBusiness Resultが`submitted`になる
2. `submitted`への再確定が`state-not-allowed`で実行されない
3. Validation不成立でAuthorizationにもCommandにも進まない
4. Authorization拒否がExecution Errorにならない
5. 未知の状態コードとMapping欠落が業務拒否へ変換されない
6. 同時2要求で遷移成立が高々1回になり、未成立側が成功や通信障害へ丸められない
7. acknowledgementだけで完了とせず、別readで状態を確認する

決定論的なDriver fixtureと、隔離した実Provider上の条件付き更新は別のEvidenceとして扱います。synthetic成功を実DBや本番運用の保証へ昇格させません。

## 関連

- ADR-0019: Authentication / Authorizationの境界
- ADR-0020: 業務Outcomeと実行エラーの分離
- ADR-0023: Command acknowledgementとObservationの明示read境界
- ADR-0024: 業務Lifecycleの所有境界
- [`requirements/TEI_PRODUCT_REQUIREMENTS.md`](../../requirements/TEI_PRODUCT_REQUIREMENTS.md) RQ-006 / RQ-018 / RQ-019 / RQ-020
