# Actor Context R3Q — Purpose-bound Projection Selective Freshness

- Status: ISSUE-000053 / Phase 3 selective-freshness pressure / Reference-only
- Date: 2026-09-18
- Parent: `plan/ACTOR_CONTEXT_R3P_PURPOSE_BOUND_STOP_PROJECTION_OMISSION_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3_SELECTIVE_FRESHNESS_2026-09.md`
  - `plan/ACTOR_CONTEXT_PROJECTIONS_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3Pでpurpose-bound projectionはselected / omitted categoryを明示できるようになった。

次はfreshnessをその境界へ合わせる。

```text
purpose-bound selection
  -> selected semantic dependencies
  -> bounded projection freshness
```

重要なのは:

```text
omitted detail changed
  != bounded projection automatically stale
```

である。

## 2. file-level stale-allを避けるPressure

一つのsource artifactにselected meaningとomitted detailが同居する場合、file hashだけをprojection freshnessへ使うと過剰invalidatonが起こり得る。

R3Qではtest-onlyにcategory semantic fingerprintを作る。

```text
category identity
  + selected semantic value
  -> category fingerprint
```

bounded viewはselected categoryだけをRequiredCategoryFingerprintsへ持つ。

production fingerprint protocolを追加する意図ではない。

## 3. Human orientationとAgent resolutionの差

R3P Human orientationは:

```text
selected:
  stop-identity
  machine-reasons
  resolution-needs
  runtime-outcome
  source-refs
```

なので、たとえばknown-authority-factsだけが変化してもHuman compact projectionのsemantic payloadは変わらない。

一方Agent resolution-inspectionは全categoryをselectedするため、同じknown fact変更でstaleになる。

```text
same source change
  -> Human bounded view: still current
  -> Agent full view: stale
```

という差をPressureする。

## 4. SourceRefs / drill-down integrityはselected dependency

Human compact viewはSourceRefsをinline selectionとして保持する。

したがって:

```text
source ref identity changed
  -> Human projection stale
```

である。

omitted detailのcontent changeと、selected drill-down identityのchangeを同一視しない。

## 5. selected meaningが変わればstale

次はHuman / Agent双方でstaleにする。

- stop identity
- machine reason
- resolution need
- runtime outcome
- source refs

たとえ変更原因がomitted detail側にあっても、最終的にselected semantic categoryが変化したならfreshness dependencyである。

```text
origin of change
  != freshness criterion

selected semantic result changed
  -> stale
```

## 6. profile definitionもfreshness dependency

selected / omitted partition自体が変われば、同じcoreでも別projectionである。

R3Qではtest-onlyにProfileFingerprintを持つ。

```text
audience
purpose
selected categories
omitted categories
  -> profile fingerprint
```

selected-but-empty categoryをomittedへ変更した場合もprofile fingerprintが変わる。

## 7. test-only source

- `tools/reference-business-procedure-check/actor_context_r3q_projection_selective_freshness_pressure_test.go`

Pressureするcase:

1. known-authority-factsだけの変更はHuman orientationをstaleにしない
2. 同じ変更はAgent resolution viewをstaleにする
3. machine-reasons変更はHuman / Agent双方をstaleにする
4. SourceRefs変更はHumanをstaleにする
5. omitted authority-limits変更はHumanをstaleにしない
6. selected / omitted profile変更はprofile fingerprintを変える
7. omitted category fingerprintをRequiredCategoryFingerprintsへ混ぜない
8. unknown / invalid category partitionをfreshness snapshotへ進ませない

## 8. current conclusion

R3Qで固定するのは:

```text
projection freshness
  = profile identity
  + selected semantic dependencies

  != all upstream artifact bytes
```

というPressureである。

ただし:

```text
semantic category fingerprint
  != production cache key
  != Canonical version
```

である。

production invalidation engine、cache、event bus、watcherは追加しない。

## 9. 次

次段ではR3Qのsemantic dependencyをactual Source Index / subject identityへどこまで逆追跡できるかを検証する。

特に:

- selected categoryを構成したauthoritative subjects
- runtime Observation dependency
- source ref locator change
- static source freshnessとruntime decision freshness

を一つのhashへ潰さずに接続する。
