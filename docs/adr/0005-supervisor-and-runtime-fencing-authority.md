# ADR-0005: Supervisor authorityとruntime fencing authorityを分離する

- 状態: 採用
- 日付: 2026-08-31
- 関連Issue: #9

## 背景

M0.22.1では、Host takeover後の二重実行や古いHostからの操作を抑止するため、Supervisor Ownershipのmonotonic fencing tokenと、外部runtimeへ再接続するときのruntime-side fencingを導入しました。

`DurableInvocationHandle`には`SupervisorOwnerID`と`FencingToken`を保存し、`ReconcileDurableInvocation`は新Supervisorが取得したtokenを使って外部runtimeへ`ReattachInvocation`します。

通常の`running` / `starting`では、runtimeへのreattachが成功した時点で、Supervisor側とruntime側の双方が新しいgenerationへ進みます。

しかし`RuntimeUnknown`では事情が異なります。`ObserveInvocation`は非authority-bearingであり、runtimeの状態を確定できないためreattachを行いません。一方、現行実装はDurable Handleの`SupervisorOwnerID` / `FencingToken`だけを新世代へ進めます。

この状態では、一つの`FencingToken`というfieldが次の二つの事実を同時に表しているように読めてしまいます。

1. Durable control plane上で、どのSupervisor generationが現在の調停権を持つか
2. 外部runtime自身が、どのfencing generationを実際に受理済みか

`RuntimeUnknown`では前者だけが確立し、後者は確立していません。この差を隠すと、Handleだけを見て「runtimeにも新tokenが設置済み」と誤解する危険があります。

## 判断

**Supervisor authorityとruntime fencing authorityは、同じgeneration番号体系を使っても、別の事実としてDurable Handleへ記録します。**

既存の`SupervisorOwnerID`と`FencingToken`は、Durable control plane上のSupervisor authorityを表すfieldとして維持します。

これに加えて`RuntimeFencingToken`を持ちます。

```text
Supervisor Ownership Store
    ↓ acquire / takeover
SupervisorOwnerID + FencingToken
    ↓
Durable Invocation Handle
    ├─ supervisor generation = FencingToken
    └─ runtime-confirmed generation = RuntimeFencingToken
                                    ↓
                         authority-bearing reattach
                                    ↓
                              external runtime
```

`RuntimeFencingToken`は、**外部runtimeがauthority-bearingな操作を通じて受理したことをTEIが確認できた最後のgeneration**です。

したがって、基本不変条件は次です。

```text
RuntimeFencingToken <= FencingToken
```

`RuntimeFencingToken == FencingToken`なら、少なくとも最後に確認した時点ではSupervisor generationとruntime-side authorityが揃っています。

`RuntimeFencingToken < FencingToken`なら、control plane上のtakeoverは進んでいるものの、runtime-side authorityはまだ追いついていません。

`RuntimeFencingToken == 0`は、runtime側で確認済みのgenerationをDurable Handleから確定できない状態を表します。既存Handleにfieldがない場合も、安全側に倒してこの状態として扱い、`FencingToken`から暗黙に推測しません。

## RuntimeUnknownの扱い

`RuntimeUnknown`を観測した場合は、次のように扱います。

- 新Supervisorの`SupervisorOwnerID` / `FencingToken`はDurable Handleへ保存する
- `RuntimeFencingToken`は進めない
- runtimeへ`ReattachInvocation`しない
- outcomeは`waiting-observation`のままにする
- runtime-side authorityが未確立であることをFindingへ明示する
- missing / failed / succeededのいずれにも推測しない
- 自動replayしない

これにより、新SupervisorがDurableな調停権を取得したことと、runtimeへのauthority設置が未完了であることを同時に表現できます。

## unknownからrunningへ戻った場合

後続観測で同じruntime handleが`running` / `starting`として確認できた場合、新Supervisorは現在の`FencingToken`を使って`ReattachInvocation`します。

reattachが成功し、live runtimeへauthorityを確立できた場合にだけ`RuntimeFencingToken`を現在の`FencingToken`へ進めます。

このため、たとえば次の遷移が成立します。

```text
before takeover
  supervisor token = 1
  runtime token    = 1

observe unknown after takeover
  supervisor token = 2
  runtime token    = 1

later observe running + reattach succeeds
  supervisor token = 2
  runtime token    = 2
```

## stale Hostに対する意味

Supervisor Ownership StoreとResource Brokerは、新Supervisorがtoken 2を取得した時点でtoken 1を拒否できます。

一方、runtimeが`unknown`でauthority-bearingな操作を実行できていない間は、runtime側にtoken 2が設置済みとは主張しません。外部runtimeへ直接到達できる古いHostに対して、runtime-side fencingまで完了したとはみなしません。

この期間は**runtime authority unresolved**です。TEIはこの状態でruntime-mutating operationや自動replayへ進みません。

production adapterが、runtime lifecycleの観測とは独立して新generationを確実に設置できるauthority primitiveを提供する場合は、将来その能力を別契約として追加できます。ただし、現在の`ObserveInvocation`をauthority-bearingへ読み替えたり、成功確認なしに`RuntimeFencingToken`を進めたりはしません。

## terminal / missingとの関係

最初の非authority-bearing observationが`succeeded` / `failed` / `missing`だった場合も、runtimeへ新tokenを設置した事実はありません。そのため`RuntimeFencingToken`は進めません。

`running` / `starting`を観測してreattachを試みたものの、その間にruntimeが`missing` / `unknown`へ変わった場合も、live runtime authorityが確立したとはみなさず`RuntimeFencingToken`を進めません。

live runtimeへのreattachが成功した場合のみ、新しいruntime fencing generationをconfirmedとして保存します。

## Durable Handle format

`RuntimeFencingToken`はadditiveなfieldとして現行formatへ追加します。

fieldが存在しない既存Handleを`FencingToken`と同値だと推測しません。値0を「runtime-confirmed generation不明」として扱うことで、古い記録を新しい意味へ過大解釈しないようにします。

Handle validationでは`RuntimeFencingToken > FencingToken`を不正として拒否します。

## 採用しなかった案

### A. `FencingToken`をruntime authorityまで含むfieldとみなし、unknownでも進める

現行挙動をそのまま正当化できますが、runtimeへtokenを提示していないのに「設置済み」と読めるDurable stateになります。観測・調停権とruntime authorityの境界が失われるため採用しません。

### B. runtime authorityが確立するまで`SupervisorOwnerID` / `FencingToken`も進めない

Handle上の表示は保守的になりますが、Supervisor Ownership Storeでは既に新Hostが新generationを取得している事実とDurable Handleがずれます。control plane authorityとruntime authorityを一つのfieldへ押し込める問題自体は解消しないため採用しません。

### C. `RuntimeUnknown`でも強制的にreattachする

状態不明のruntimeへauthority-bearing operationを行うことが常に安全とは限りません。また、現在のinterfaceは「unknownでも確実にfenceだけを設置できる」契約を持ちません。観測不能を成功扱いすることにもつながるため採用しません。

## Conformance

Issue #9の実装では、少なくとも次をテストで固定します。

- initial live handleではSupervisor tokenとruntime-confirmed tokenを別fieldとして保持できる
- `RuntimeUnknown`ではSupervisor generationだけが進み、runtime-confirmed generationは進まない
- unknownではreattachを呼ばない
- unknown後に同じSupervisor generationで再観測し、runningになればreattachしてruntime-confirmed generationが追いつく
- stale Supervisor tokenはDurable Handleでruntime observation前に拒否される
- `RuntimeFencingToken > FencingToken`はHandle validationで拒否される
- missing / terminalの非authority-bearing observationだけではruntime-confirmed generationを進めない

## 影響

Durable Handleを見る側は、`FencingToken`だけで「runtimeにも同じauthorityが設置済み」と判断してはいけません。

Supervisor ownership、Resource authority、runtime authorityを段階的に確立する設計になります。状態は少し増えますが、分散実行で最も危険な「確認していないauthorityを確認済みとして保存する」曖昧さを避けられます。
