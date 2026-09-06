# Business Procedure Step Address Layer Pressure — 2026-09

Status: **Lane C / test-only architecture pressure / no Core adoption**

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

## 6. test-only Pressure

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

## 8. 現時点の優先順位

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
