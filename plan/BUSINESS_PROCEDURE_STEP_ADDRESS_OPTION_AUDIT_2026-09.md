# Business Procedure Step Address Option Audit — 2026-09

Status: **Lane C / option audit / no Core adoption**

## 1. 問い

Stepについて必要なのは、Business body meaningそのもののdurable global identityか、それともhandoff / review / Acceptanceから安全に指せるaddressabilityか。

前段Pressureでは次を分離した。

```text
ordered Business Action body
  Step IDなしでも意味候補を表現できる

external relation / edit target
  exact stable Step addressが必要
```

この監査では、global Canonical Step IDを増やす前に、**Procedure-scoped local Step key**で十分かを比較する。

## 2. current Referenceがすでに持つ形

### Handoff

```text
from
  Procedure = order-entry.order-handling
  Step      = BP-09
```

### Acceptance candidate

```text
TargetProcedure       = order-entry.order-handling
Step                  = BP-04
BaseSourceFingerprint = ...
```

### Source registry

```text
Procedure key
  -> artifact locator
```

つまりStepは現在、file pathやglobal runtime identityではなく、Procedure内local keyとして利用されている。

## 3. Option S4 — Procedure-scoped Step address

候補:

```text
StepAddress
  procedure
  stepLocalKey
```

性質:

- `stepLocalKey`はProcedure内でunique
- file pathを含めない
- global Canonical namespaceを要求しない
- runtime execution / instance / state identityではない
- Business Action bodyのsemantic equalityとは分離できる
- rename時はreferential impactとして扱う

## 4. Business meaningとaddress metadataの分離

Step local key renameを次の二つへ分ける。

```text
Business body meaning
  unchanged when action / refs / order are unchanged

address graph
  changed because handoff / review / Acceptance endpoints must migrate
```

これは「renameを無視する」ことではない。

Business content diffではなく、referential / structural impactとして明示的に扱う。

## 5. Source snapshotとの組み合わせ

AcceptanceではStep addressだけでなくartifact fingerprintを使う。

```text
Procedure-scoped Step address
        +
base source fingerprint
```

これにより、同じStep local keyでもsource内容が先に変更されたstale candidateを拒否できる。

fingerprintはStep addressそのものではなく、Acceptance snapshot guardとして残す。

## 6. file move

Procedure -> artifact locator registryを更新すれば、pure file moveでStep addressは変わらない。

```text
before
  order-entry.order-handling / BP-09
  -> procedures/order-handling.json

after
  order-entry.order-handling / BP-09
  -> procedures/sales/order-handling.json
```

Step addressとBusiness body meaningは不変でよい。

## 7. logical Step rename

```text
BP-04
  -> BP-04-renamed
```

はBusiness Action wordingが同じでもaddress graph changeである。

旧handoff / candidate / review linkを名称類似や位置から補修しない。

明示的にendpointを更新する。

## 8. test-only Pressure

`tools/reference-business-procedure-check/step_address_option_pressure_test.go`で次を確認する。

1. address shapeを`Procedure + Step`に限定する。
2. handoffとAcceptanceが同じcomposite address shapeを利用できる。
3. artifact locator moveでaddress不変。
4. Step local key rename後の旧Acceptance targetはfail closed。
5. 新local keyへ明示更新すればtargetを解決できる。
6. global/runtime/source identityをStep addressへ混ぜない。

actual Repository runnerで実行されるまではgreen Evidenceではない。

## 9. Option比較

### S1 — global Canonical Step ID

長所:
- 参照が直接的
- Canonical graphだけでaddress可能

短所:
- identity概念を増やす
- local authoring renameをglobal semantic identity changeへ昇格しやすい
- Runtime / workflow identityとの混同余地

### S2 — ID-free body +独立address object

長所:
- body meaningとaddressを明確に分離

短所:
- mapping objectが増える
- Source / Canonical二重Authorityになり得る

### S3 — Reference-local keyのみ

長所:
- Core変更不要

短所:
- accepted Canonical handoff / change graphがSourceへ依存し過ぎる可能性

### S4 — Procedure-scoped local address metadata

長所:
- global Step namespace不要
- handoff / Acceptance / reviewのshapeを共通化できる
- body semantic equalityとreferential impactを分けられる
- file move independent

未解決:
- normalized Canonical modelへaddress metadataを保持するか
- Source Indexだけに保持するか
- 000039 semantic diffでaddress renameをどのcategoryへ出すか

## 10. 現時点の優先候補

**S4を次の比較対象として優先する。**

これはCanonical Step ID採用ではない。

またStep keyをBusiness Action bodyそのもののsemantic identityともみなさない。

```text
BusinessProcedureStep
  addressMetadata
    localKey
  bodyMeaning
    businessAction
    explicit refs
```

のように、同じnormalized objectに共存させる案と、Source Indexへaddressだけ残す案を次に比較する。

## 11. 次のPressure

次は二案を比較する。

### S4-A normalized address metadata

accepted normalized ProcedureにStep local keyをaddress metadataとして保持する。

- handoff exact resolution
- cross-view impact
- reload consistency
- rename structural diff

をCanonical/normalized layerで扱える。

### S4-B Source Index address only

normalized Business bodyにはStep keyを入れず、Source IndexがProcedure + Step keyを管理する。

Coreはさらに薄いが、handoff / change graphがSource infrastructureに依存しすぎないかを確認する。

この比較が終わるまでglobal Canonical Step IDを採用しない。
