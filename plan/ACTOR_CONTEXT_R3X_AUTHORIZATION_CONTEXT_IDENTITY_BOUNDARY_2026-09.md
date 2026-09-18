# Actor Context R3X — Authorization Observation Context Identity / Freshness Authority Boundary

- Status: ISSUE-000053 / Phase 3 Authorization evaluation-context pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3W_CATEGORY_VALUE_CHANGE_BOUNDARY_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3I_AUTHORIZATION_TARGET_EVALUATION_CADENCE_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3N_STAGE_AWARE_OPERATIONAL_STOP_TAXONOMY_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3Wではselected projection meaningのactual changeをR3Q fingerprintで確定できるようになった。

次は、そのchangeがprior Authorization Observationのcontext identityへどう関係するかを分離する。

R3Iのcurrent Observation tupleは:

```text
Business address
subject id
action
resource
```

である。

R3Xでは:

```text
projection meaning changed
  != Authorization context tuple changed

Authorization context tuple changed
  -> prior Observation cannot be current evidence

Authorization context tuple unchanged
  != prior Observation reusable
  != prior Observation still fresh
```

をPressureする。

## 2. tuple component changeはdeterministic mismatch

prior Observationとcurrent contextの:

- Business address
- subject id
- action
- resource

のいずれかが変われば、prior Observationはcurrent-context Observationではない。

```text
tuple component changed
  -> observation-context-mismatch
```

である。

ここではreevaluationを自動実行しない。

```text
context mismatch
  != authorize again automatically
```

を維持する。

## 3. requirement applicabilityはtuple外

Authorization requirement statusは、R3I Observation tupleそのものではない。

required / unresolved / explicitly-not-required等のapplicability meaningが変化した場合:

```text
requirement applicability changed
  -> requirement transition authority needed
```

とする。

tupleが同じだからprior decisionを再利用してよいとも、requirementが変わったからprior Observationを破棄すべきとも、R3Xでは決めない。

## 4. projection categoryからtupleを逆推論しない

R3Wで:

- machine-reasons
- resolution-needs
- source-refs
- runtime-outcome

等がchanged-after-reevaluationになっても、それだけではどのAuthorization tuple componentが変わったかは分からない。

```text
machine-reasons changed
  != action changed

resolution-needs changed
  != resource changed

source-refs changed
  != Authorization context changed
```

である。

context component changeはexplicit evidenceからのみ受け取る。

## 5. locator / presentation change

SourceRefsやHuman表示文の変更だけなら:

```text
Authorization tuple identity unchanged
```

である。

ただし:

```text
tuple unchanged
  != decision freshness proven
```

なので、prior Observation freshness/reuse authorityはunresolvedのままにする。

## 6. static semantic change with same tuple

Business Procedure / target authority / Pattern等のstatic meaningが変わったが、current tupleの値自体は同一という場合がある。

そのとき:

```text
tuple-compatible
  + static semantic change
  -> evaluation freshness authority unresolved
```

とする。

R3Xは:

- prior decision automatically invalid
- prior decision automatically reusable
- immediately reauthorize

のいずれも選ばない。

## 7. runtime outcome changeはidentity changeではない

同じBusiness address / subject / action / resourceについて:

```text
allowed -> denied
```

の新Observationが得られてもtuple identityは同じである。

outcomeはObservation valueでありidentity componentではない。

```text
runtime outcome changed
  != context identity changed
```

を維持する。

## 8. test-only source

- `tools/reference-business-procedure-check/actor_context_r3x_authorization_context_identity_pressure_test.go`

Pressureするcase:

1. action change -> prior Observation mismatch
2. resource / subject / Business address change -> prior Observation mismatch
3. requirement status change + same tuple -> requirement-transition-authority-unresolved
4. source-refs change + same tuple -> context identity unchanged / freshness authority unresolved
5. machine-reasons changeだけではtuple componentを推論しない
6. static semantic change + same tuple -> freshness authority unresolved
7. allowed -> denied + same tuple -> outcome change only
8. Reusable / Valid / CacheValid / ShouldReevaluate / AutoReauthorize等を追加しない

## 9. current conclusion

R3Xで固定するのは:

```text
tuple mismatch
  -> prior Observation is not current-context evidence

tuple match
  != freshness proven
  != reuse authorized

requirement transition
  != tuple mismatch

projection category change
  != tuple component change
```

である。

production TTL / revocation watcher / Authorization cache / reevaluation schedulerは追加しない。

## 10. 次

次段では、tuple matchのままstatic semantic meaningが変わったcaseについて:

```text
what authority can require reevaluation?
what authority can permit reuse?
```

を別々にPressureする。

特にrequirement / target authority revisionとpolicy-provider revisionを同一視せず、R3Iのcross-context reuse boundaryへ戻す。
