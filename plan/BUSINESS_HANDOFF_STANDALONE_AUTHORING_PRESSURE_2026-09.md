# Business Handoff Standalone Authoring Pressure — 2026-09

Status: **Lane C / test-only source placement pressure / no production adoption**

## 1. 問い

Current Reference Setは次を同じfileに持つ。

```text
Procedure membership
Procedure artifact locator
Handoff relation
```

Reference fixtureとしては便利だが、production Business authoring候補としては:

```text
Source locator metadata
  +
accepted Business Handoff meaning
```

が混在する。

本PressureではH1/R3を、Procedure artifact locatorを持たないstandalone Business relation sourceとしてlosslessに表現できるかを確認する。

## 2. Test-only source shape

```text
BusinessHandoffSource
  Handoffs[]
    from { procedure, step }
    to   { procedure, step }
    condition
```

持たないもの:

- Procedure file path / artifact locator
- Runtime state
- retry / timer / queue / scheduler
- Business Flow / DFD node
- generic Edge type
- Data / payload flow
- global Handoff ID

## 3. Mechanical Pressure

`handoff_standalone_authoring_pressure_test.go`で次を確認する。

### Same normalized meaning

Current Reference SetのHandoffとstandalone sourceから、同じ`canonicalBusinessHandoffProbe`を生成できる。

```text
Reference Set Handoff
  == normalized meaning ==
standalone Handoff source
```

### Locator independence

standalone sourceはProcedure artifact locatorを持たない。

したがってProcedure file moveはHandoff source rewriteを要求しない。

### Exact endpoint validation

Procedure / Step endpointはcurrent Business Procedure identitiesへexact resolutionする。

Stale Step addressはfail closedする。

### Handoff-only boundary

Strict decodeにより`retry`等のWorkflow fieldを黙って受理しない。

`business_authoring_duplicate_handoff_source_pressure_test.go`では、複数のmanaged standalone sourceを束ねる際のauthority境界を追加Pressureする。

### Duplicate source authority boundary

同一のH1/R3 tuple:

```text
from { procedure, step }
to   { procedure, step }
condition
```

が異なるphysical sourceから複数回authorされた場合はduplicate authorityとしてfail closedする。

```text
same H1/R3 meaning
+ different paths
  -> reject
```

入力順、lexical path、first-writer / last-writerでauthorityを解決しない。

一方、current H1/R3にはglobal Handoff IDがないため、endpoint pairだけを暗黙identityとして発明しない。

```text
same from/to
+ different condition
  -> distinct H1/R3 meaning candidates
  -> endpoint-only identityを発明してrejectしない
```

これは同じendpoint間の複数relationをproductionで必ず許可するという契約採用ではない。Current representationのidentityをpathやendpoint pairへ勝手に縮めないためのtest-only Pressureである。

## 4. Relation to S4-A

Handoff endpointは:

```text
Procedure identity
+ Procedure-scoped Step localKey
```

を参照する。

Global Canonical Step IDは要求しない。

Standalone sourceでもS4-A addressabilityで十分なPressureになっている。

Handoff source自身についてもphysical pathをlogical identityへ昇格しない。

## 5. Relation to Reference Set

Current `order-entry-procedure-reference-set.json`はReference fixtureとして維持してよい。

ただしproduction source candidateとして:

```text
Reference Set manifest shape
```

をそのまま採用する根拠にはしない。

Reference Setはfixture assembly / artifact discoveryの役割を持つため、Handoff semantic sourceと同一contractにする必要はない。

## 6. Current source ranking

```text
Procedure-local inline Handoff
  -> cross-Procedure relationなので弱い

Reference Set manifest内Handoff
  -> fixtureには便利だがlocator metadataとmeaningが混在

standalone Handoff relation source
  -> current strongest production authoring candidate
```

ただしproduction syntax / package / Shared Acceptance APIはまだ採用しない。

## 7. Acceptance implication

Standalone Handoff sourceを採る場合も、semantic revisionはraw file fingerprintだけではなくnormalized Business meaning単位で考える必要がある。

Current test-only normalized Business fingerprintでは:

```text
Handoff condition change -> Business semantic revision changes
Procedure artifact move  -> Business semantic meaning unchanged
```

を保持できる。

Physical source discovery側ではさらに:

```text
source path
  -> locator / discovery metadata

exact H1/R3 tuple
  -> current relation meaning key for duplicate detection

source authority
  -> accepted managed source-set boundary
```

を分離する。

したがってHandoff source file自身の物理配置をBusiness identityへしない。

Duplicate exact relationを複数sourceから受理すると、同じaccepted meaningを複数sourceがauthorするため、Procedure duplicate logical sourceと同様にauthority ambiguityになる。これはfreshness envelopeのsource-set membership / logical uniqueness Pressureとも整合する。

ただしfinal production source key、revision token、Handoff durable identityはここでは決めない。

## 8. Current decision

```text
H1/R3 normalized meaning
  -> current minimum candidateに必要

Handoff authority
  -> explicit Business Handoff source only

production authoring candidate
  -> standalone set-level relation sourceが現在最有力

physical path
  -> Handoff identityではない

exact duplicate H1/R3 meaning across managed sources
  -> fail closed

endpoint pair only
  -> implicit Handoff identityにしない

global Handoff ID
  -> not introduced

Reference Set manifest shape
  -> production schemaとしては未採用

BusinessHandoff Canonical top-level kind
  -> still not proven
```

次の未解決は:

1. standalone sourceをproduction Business authoring syntaxとして採るか。
2. Business semantic revision / Shared Acceptanceへどう接続するか。
3. production source discoveryでHandoff source-set uniquenessをどのphysical token / indexで検証するか。
4. second realistic Handoffで`from / to / condition`だけで十分か。

Actual Repository runner greenまではproduction contractへ進めない。
