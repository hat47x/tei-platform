# Actor Context R3AG — Purpose-bound Outcome Explanation Materialization

- Status: ISSUE-000053 / Phase 3 purpose-bound explanation materialization pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AF_REEVALUATION_OUTCOME_EXPLANATION_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3P_PURPOSE_BOUND_STOP_PROJECTION_OMISSION_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3V_CROSS_AUDIENCE_REASON_IDENTITY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3AB_CROSS_AUDIENCE_FRESHNESS_DRILLDOWN_PROVENANCE_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3AFでは:

```text
category outcome
freshness authority status
unselected changes
residual root deltas
```

を同じunderlying Operational explanationに保持した。

R3AGでは、このunderlying explanationを変えずにHuman / Agentのpurposeへmaterializeするとき:

```text
inline
drill-down
omitted
```

を明示的に分ける。

```text
presentation selection
  != truth selection
```

をPressureする。

## 2. detail class

test-onlyでは少なくとも次を独立classとして扱う。

```text
category-outcome-summary
category-outcome-fingerprints
category-outcome-reasons
freshness-authority-status
resolution-needs
unselected-changes
residual-root-deltas
provenance-drilldown
```

各classはprofileごとにexactly one of:

```text
inline
drill-down
omitted
```

へ属する。

```text
drill-down
  != omitted

selected-but-empty
  != omitted
```

である。

## 3. Human orientation

Human orientationでは、まず現在の状態を短く理解できることを優先する。

test-only profile:

### inline

- category-outcome-summary
- freshness-authority-status
- resolution-needs

### drill-down

- category-outcome-fingerprints
- category-outcome-reasons
- residual-root-deltas
- provenance-drilldown

### omitted

- unselected-changes

Human inlineではfingerprintやdependency reasonを直接並べなくてもよい。

ただしcategory summaryにはunderlying outcome identityへのtest-only referenceを保持し:

```text
inline detail omitted
  != underlying identity lost
```

とする。

ResidualRootDeltaが空でも `residual-root-deltas` はdrill-down selected-but-emptyであり、omittedへ変換しない。

## 4. Agent resolution-inspection

Agent resolution-inspectionでは機械的なtraceabilityを優先する。

test-only profile:

### inline

- category-outcome-summary
- category-outcome-fingerprints
- category-outcome-reasons
- freshness-authority-status
- resolution-needs
- unselected-changes
- residual-root-deltas

### drill-down

- provenance-drilldown

### omitted

- none

Agent inlineが詳細でも、そのdetailが新しいauthorityになるわけではない。

## 5. shared outcome identity

R3AFのexact outcome identityは少なくとも:

```text
category
status
before fingerprint
after fingerprint
exact reason identities
```

からtest-onlyに導出する。

Human / Agentで同じcategory outcomeを共有する場合:

```text
Human summary outcome identity
  = Agent detailed outcome identity
```

でなければならない。

audience / wording / render orderをidentityへ含めない。

## 6. freshness authorityは常に別軸

Human / Agentどちらでも:

```text
category outcome materialization
  != freshness authority resolution
```

である。

current freshness-stopでは:

```text
reevaluation authority = unresolved
reuse authority = unresolved
```

をmaterialization profileにかかわらず保持する。

Human向けに短く表示しても `resolved` / `fresh` / `safe` へ変換しない。

## 7. residual / unselectedの扱い

R3AD ResidualRootDeltaとR3T UnselectedChangesは別概念のまま保持する。

Human orientationでは:

- residual = drill-down
- unselected = omitted

Agent resolution-inspectionでは:

- residual = inline
- unselected = inline

とできる。

ただし:

```text
Human omitted unselected
  != unselected absent from underlying explanation

Human drill-down residual
  != residual resolved

Agent inline residual
  != residual authoritative
```

である。

## 8. provenance drill-down

`provenance-drilldown` はpresentation entry pointでありauthority sourceそのものではない。

drill-down先ではR3AB / R3R / R3Sで既に保持した:

- root identity / role
- existing projection dependency key
- dependency reason
- current locator

へ戻れる。

```text
drill-down locator
  != authority role
```

を維持する。

R3AG用に第二provenance graphを作らない。

## 9. wordingはderived

R3AGはstable Human copyを固定しない。

Human orientationでは概念上:

```text
入力変更を受けて再確認しました。
現在表示している意味は変わっていません。
Authorization freshness authorityは引き続き未解決です。
```

のような短い表示へmaterializeできる。

ただし文言はderived presentationであり、outcome identity / authority statusではない。

## 10. test-only source

- `tools/reference-business-procedure-check/actor_context_r3ag_purpose_bound_outcome_explanation_materialization_pressure_test.go`

Pressureするcase:

1. Human inlineはsummary + freshness authority + resolution needsだけを持つ
2. Human fingerprints / reasons / residual / provenanceはdrill-downへ残る
3. Human unselected changeはomittedでもunderlying explanationから消えない
4. Agentはexact fingerprints / reasons / residual / unselectedをinlineに持てる
5. shared machine-reasons outcome identityがHuman / Agentで一致する
6. selected-but-empty residual detailとomitted detailを区別する
7. freshness authority unresolvedが両profileで同一
8. display wording / audience / orderをoutcome identityへ入れない
9. Fresh / Safe / Reusable / ShouldReevaluate / NextAction等を追加しない

## 11. current conclusion

R3AGで固定するのは:

```text
one underlying explanation
  -> purpose-bound materialization

inline
  != complete truth

drill-down
  != omitted

omitted
  != absent from underlying explanation

shared outcome identity
  != shared presentation
```

である。

production UI schema、API response contract、notification transport、ticket routingは追加しない。

## 12. 次

次段ではmaterialized explanationからdrill-downした際に:

```text
summary outcome identity
  -> exact category outcome
  -> exact reason
  -> existing dependency
  -> provenance root / current locator
```

をcross-audienceで同じchainとして辿れるかをPressureする。
