# Business Procedure Step Address — 2026-09

Status: **Lane C / test-only architecture pressure / no Core adoption**

## この文書について

本書は、Business Procedure Stepのidentity/addressabilityに関する調査を時系列で行った次の3文書を1本へ統合したものである。

1. `BUSINESS_PROCEDURE_STEP_IDENTITY_PRESSURE_2026-09.md`（元・Step durable identityの要否を問う最初のPressure）
2. `BUSINESS_PROCEDURE_STEP_ADDRESS_OPTION_AUDIT_2026-09.md`（元・S1〜S4の選択肢比較、S4を優先候補に決定）
3. `BUSINESS_PROCEDURE_STEP_ADDRESS_LAYER_PRESSURE_2026-09.md`（元・S4-A/S4-Bのlayer比較）

いずれもorphan（他文書から参照されていない）で、単一の連続した調査だったため統合した。個別ファイルの履歴はGit historyから追跡できる。

---

# Part 1 — Step Identity Pressure（出発点の問い）

## 1. 問い

Business Procedure Stepの`BP-04`等のkeyは、production Canonical body meaningそのものに必要なdurable identityか。

current ReferenceではStep keyが二つの役割を持つ。

1. Procedure内部のauthoring / review上のlocal key
2. handoff endpointを外部relationから指すaddress

この二つを同一視しない。

## 2. body meaningからのidentity subtraction

Step IDを除いたbody candidate:

```text
Procedure
  id
  purpose
  trigger
  actor
  ordered steps

Step
  businessAction
  explicit semantic refs
```

ordered sequenceは保持する。

このcandidateではStep local key renameだけではProcedure body meaningを変えない。

## 3. addressability pressure

一方、current handoffは

```text
Procedure/Step -> Procedure/Step
```

をexact endpointとして使う。

handoff元`BP-09`だけをrenameしReference setを更新しなければstale linkとしてfail closedする必要がある。

位置や名称類似から補完しない。

したがって、**body meaningにStep IDが不要であること**と、**cross-referenceのためにstable Step addressが必要であること**は両立する。

## 4. test-only Pressure（Step Identity）

`tools/reference-business-procedure-check/step_identity_pressure_test.go`で次を確認する。

1. Procedure bodyをStep IDなしで投影できる。
2. Step local key renameだけではID-free body meaningを変えない。
3. Step order入れ替えはbody meaning changeとして残る。
4. current handoff endpointのStep key renameはstale linkとしてfail closedする。
5. endpoint identityへruntime execution / instance / state identityを混ぜない。

actual Repository runnerで実行されるまではgreen Evidenceではない。

## 5. Part 1時点の判断

Step durable Canonical identityはまだ採用しない。

current Evidenceが示したのは次までである。

```text
ordered Business Action body meaning
  does not inherently require Step ID

external handoff / review addressability
  does require a stable exact endpoint
```

次に必要なのは、accepted cross-procedure handoffやrevision-aware reviewがSource-local keyだけで安全に維持できるかのEvidenceである。

Source-local keyで十分ならCoreへStep identityを増やさない。十分でなければ、body semantic identityとaddress identityを分離する案も含めて再評価する（→ Part 2）。

---

# Part 2 — Step Address Option Audit（S1〜S4比較）

## 1. 問い

Stepについて必要なのは、Business body meaningそのもののdurable global identityか、それともhandoff / review / Acceptanceから安全に指せるaddressabilityか。

前段Pressure（Part 1）では次を分離した。

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

## 8. test-only Pressure（Option Audit）

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

## 10. Part 2時点の優先候補

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

のように、同じnormalized objectに共存させる案と、Source Indexへaddressだけ残す案を次に比較する（→ Part 3）。

## 11. 次のPressure（Part 3へ）

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

---

# Part 3 — Step Address Layer Pressure（S4-A vs S4-B、現在の結論）

## 1. 問い

Procedure-scoped Step local keyを採用候補にする場合、それをどのlayerへ置くべきか。

- S4-A: normalized / Canonical Business Procedure側のaddress metadata
- S4-B: Authoring / Source Index側だけのaddress

## 2. formal architecture constraint

`docs/architecture.md`では、Canonical Semantic Layerはauthoring形式やfile pathから独立したidentityと意味を保持し、Semantic validationはSource Indexへ依存しない。

Source Indexはlocator / fingerprint / range等を持つderived Authoring-side projectionである。

したがって、accepted Canonical handoff relationがStep endpointをsemantic validation / impact analysisで解決するなら、その解決にSource Indexを必須とする設計はformal architectureと緊張する。

## 3. S4-A — normalized address metadata

```text
BusinessProcedure
  ...
  orderedSteps
    - addressMetadata
        localKey
      bodyMeaning
        businessAction
        explicit refs
```

### 性質

- localKeyはfile pathではない
- Procedure内namespace
- runtime identityではない
- Business body meaningとは別のstructural / referential metadata
- normalized graphだけでhandoff endpointをexact解決できる

Step local key renameは、Business Action body changeではなくaddress graph / referential structure changeとして観測する。

## 4. S4-B — Source Index only

```text
Canonical Procedure body
  ordered Business Action

Source Index
  Procedure + Step local key
    -> source location / body position
```

Coreはさらに薄くなる。

しかしCanonical handoff relationを保持する場合、endpoint解決のためにSource Indexへ戻る必要が出る。

これはSemantic validationがSource Indexへ依存しないという現行architectureと合いにくい。

## 5. 条件分岐

### handoffがCanonical semantic relationになる場合

S4-Aを優先候補とする。

理由:

- Canonical/normalized meaningだけでreferential integrityを検査できる
- Source locator / fingerprintをCoreへ入れずに済む
- Step global IDも不要

### handoffがAuthoring-only relationに留まる場合

S4-Bも成立し得る。

その場合、handoffのdiff / impact / AI read-only exploration / reload ConformanceまでSource-side relationで十分かを別Evidenceで確認する。

## 6. test-only Pressure（Layer Pressure）

`tools/reference-business-procedure-check/step_address_layer_pressure_test.go`で次を確認する。

1. normalized addressed Stepは`LocalKey + Body`だけを持ち、path / fingerprint / revisionを含まない。
2. current Canonical handoff probeをnormalized Procedure address metadataだけでexact解決できる。
3. Step local key renameはBusiness body meaningを変えない。
4. 同renameはaddress structure changeとしては観測される。
5. ID-free bodyだけにはhandoff endpointを解決するaddressabilityが存在しないことを明示する。

actual Repository runnerで実行されるまではgreen Evidenceではない。

## 7. semantic diffの分類候補

Step local key renameを一つの`Business meaning changed`だけで処理すると粗すぎる。

将来000039と接続する場合、少なくとも概念上は次を分ける価値がある。

```text
BodyMeaningChanged
ReferentialAddressChanged
SourceLocatorChanged
```

- Business Action wording変更: BodyMeaningChanged
- Step local key rename: ReferentialAddressChanged
- file move: SourceLocatorChangedのみ、semantic no-opになり得る

このtaxonomyを今すぐproduction enumへ固定しない。

## 8. 現時点（全体）の優先順位

1. global Canonical Step IDは導入しない。
2. Canonical handoffを採用候補として残すならS4-Aを優先してPressureする。
3. Step local keyはsemantic body identityではなくProcedure-scoped address metadataとして扱う。
4. Source path / fingerprint / revisionはSource Index / Acceptance側に残す。
5. handoff自体をCanonical化しない結論になった場合はS4-Bへ戻れるようにする。

## 9. まだ決めないもの

- production struct / DSL syntax
- `localKey`という正式名称
- local key生成方法
- UUID採用
- rename migration UX
- 000039 diff categoryの正式enum
- handoff Canonical adoption

Reference Evidenceで必要性を確認してからADRへ進む。
