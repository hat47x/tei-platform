# Actor Context R3S — Projection Dependency Reason Boundary

- Status: ISSUE-000053 / Phase 3 dependency-reason pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3R_PROJECTION_DEPENDENCY_PROVENANCE_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3Q_PROJECTION_SELECTIVE_FRESHNESS_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3O_AUDIENCE_NEUTRAL_STOP_PROJECTION_CORE_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3M_OPERATIONAL_STOP_EXPLANATION_HANDOFF_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3Rではprojection dependencyを:

```text
profile
semantic Source Index subject
Reference artifact
runtime Observation
locator integrity
```

へ分離した。

次に必要なのは、単にdependency集合を持つことではなく:

```text
selected projection category
  -> why this dependency is required
  -> exact dependency provenance
```

を説明できることである。

R3Sではこの関係をtest-onlyなdependency reason edgeとしてPressureする。

## 2. reason edgeはauthorityではない

reason edgeは:

- category
- dependency key
- reason code

を結ぶderived relationである。

```text
dependency reason
  != Semantic authority
  != Business authority
  != Authorization authority
  != next action recommendation
```

とする。

reason edgeからmissing value、handoff owner、preferred sourceを生成しない。

## 3. current BP-06での例

current action-authority stopでは:

```text
stop-identity
  <- Business Procedure Reference
     reason = business-address-source

machine-reasons
  <- Business Procedure Reference
     reason = authorization-requirement-source
  <- protected-target overlay
     reason = protected-resource-source

resolution-needs
  <- current stop classification inputs
     reason = unresolved-authority-classification

source-refs
  <- current Reference artifact locators
     reason = drill-down-locator
```

と説明できる。

ここでBusiness Procedure / overlayはcurrent Source Index subjectへ昇格させない。

## 4. known-authority-facts

Agent resolution-inspectionでは `known-authority-facts` がselectedされる。

そのfactごとに:

```text
authorization requirement = required
  <- Business Procedure Reference

protected resource = order_entry.submit_order
  <- protected-target overlay
```

というdependency reasonを保持できる。

Human orientationでは同categoryがomittedなので、そのreason edgeをHuman bounded dependencyへ混ぜない。

## 5. runtime-outcome

runtime outcomeが `allowed` / `denied` 等でnon-emptyなら:

```text
runtime-outcome
  <- exact current Authorization Observation
     reason = exact-runtime-observation
```

を要求する。

static source dependencyだけでruntime outcomeの理由を説明しない。

```text
same Repository source
  != same runtime outcome
```

を維持する。

## 6. source-refsとsemantic subject

Source Index管理下のsubjectをdrill-downとしてselectedする場合は:

```text
source-refs
  <- application-pattern:order_entry.submit_order
     reason = source-index-definition-locator
```

のようにsubject identityへ戻れる。

ただし:

```text
drill-down dependency
  != protected-resource authority
  != Authorization action authority
```

である。

## 7. orphan / omitted edgeをfail closedする

R3Sでは:

- selectedされていないcategoryへのedge
- R3R dependency envelopeに存在しないdependency key
- reason codeが空
- non-empty runtime outcomeなのにruntime Observation reason edgeがない

場合は成立させない。

これにより、説明用dependency graphがsource inventoryから独立して漂流することを防ぐ。

## 8. test-only source

- `tools/reference-business-procedure-check/actor_context_r3s_projection_dependency_reason_pressure_test.go`

Pressureするcase:

1. current BP-06 Human viewでmachine reason / source refのreason edgeを作る
2. omitted known-authority-facts edgeをHuman viewで拒否する
3. Agent viewではknown fact reason edgeを保持できる
4. Source Index subject dependencyをdrill-down reasonへ接続できる
5. denied runtime outcomeをexact runtime Observation reasonへ接続する
6. runtime outcomeのObservation reason欠落をfail closedする
7. orphan dependency keyを拒否する
8. Owner / SuggestedAction / PreferredSource / Confidenceを導入しない

## 9. current conclusion

R3Sで固定するのは:

```text
dependency provenance
  + dependency reason
  -> explainable bounded projection freshness

reason edge
  != authority edge
```

である。

production dependency graph、incremental build engine、impact engine、ticket routingは追加しない。

## 10. 次

次段ではreason edgeを利用し、stale判定を単一booleanではなく:

- semantic-source-changed
- reference-input-changed
- locator-refresh-required
- runtime-observation-changed
- profile-changed

の再評価理由として保持できるかをPressureする。
