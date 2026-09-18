# Actor Context R3AC — Freshness Drill-down Root Change Classification

- Status: ISSUE-000053 / Phase 3 provenance-root change classification pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AB_CROSS_AUDIENCE_FRESHNESS_DRILLDOWN_PROVENANCE_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3R_PROJECTION_DEPENDENCY_PROVENANCE_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3T_PROJECTION_REEVALUATION_REASON_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3X_AUTHORIZATION_CONTEXT_IDENTITY_BOUNDARY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3Y_AUTHORIZATION_REEVALUATION_REUSE_AUTHORITY_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3ABではfreshness stopから:

```text
Reference Evidence
runtime Observation
test-only Authorization context
Legacy policy corroboration
```

を別のprovenance root roleとして保持した。

R3ACでは、それらのrootが変化した場合も一つのgeneric `source changed` へ潰さない。

```text
root role
  + identity-preserving delta
  -> role-aware change class
```

をPressureする。

## 2. change class

test-onlyでは少なくとも次を分ける。

```text
reference-evidence-content-changed
runtime-observation-content-changed
test-only-context-changed
corroboration-changed
locator-refresh-required
provenance-root-set-changed
```

### Reference Evidence

Business Procedure Reference / protected-target overlay等のcontent fingerprint変更。

これはstatic/reference pressureであり:

```text
Reference changed
  != reevaluation directive authored
  != reuse permission authored
```

である。

### Runtime Observation

prior Authorization Observation contentの変更。

```text
runtime Observation changed
  != Repository Reference changed
```

であり、R3AA freshness authority unresolved中は新outcomeをcurrent RuntimeOutcomeへ自動昇格させない。

### Test-only Context

R3X same-tuple pressureで使うexplicit test contextの変更。

これはR3X context identityの再確認対象になり得るが:

```text
test-only context changed
  != production action authority changed
```

である。

### Corroboration

Legacy Authorization policy等のcorroborating input変更。

```text
corroboration changed
  != Business/Application authority changed
  != freshness directive changed
```

である。

### Locator

content fingerprintが同じでlocatorだけ変わる場合:

```text
locator-refresh-required
  != content changed
  != authority changed
```

とする。

## 3. identityとdeltaを分ける

R3AB root identityはlocatorを含めない。

したがって同一root identityに対する:

- fingerprint delta
- locator delta

を独立に観測する。

同時に起きた場合は二つのchange reasonを保持し、一方へ丸めない。

## 4. root set change

root identityそのものが追加 / 削除された場合は:

```text
provenance-root-set-changed
```

として保持する。

missing rootを別roleへ推測移送しない。

```text
root disappeared
  != authority moved
  != source superseded
```

である。

## 5. Human / Agentに同じdelta classificationを使う

R3ABではHuman / Agentのshared rootsは同一で、inline disclosureだけが異なる。

したがって同じbefore / after root setなら:

```text
Human change classification
  = Agent change classification
```

を要求する。

inline disclosure差をfreshness semanticsへ逆流させない。

## 6. re-evaluation actionを決めない

R3AC change classは観測されたdeltaの種類であり:

- ShouldReevaluate
- ReusePermitted
- RetryRequired
- CacheValid
- AutoRefresh
- HandoffOwner

を決めない。

```text
root delta classified
  != response action selected
```

である。

## 7. test-only source

- `tools/reference-business-procedure-check/actor_context_r3ac_freshness_drilldown_root_change_pressure_test.go`

Pressureするcase:

1. Reference Evidence content変更を専用classにする
2. runtime Observation変更をReference変更へ潰さない
3. test-only context変更をproduction authority変更へしない
4. corroboration変更をauthority変更へしない
5. locator-only moveをcontent changeへしない
6. fingerprint + locatorの同時変更は両reasonを保持する
7. root追加 / 削除をroot-set changeとして保持する
8. Human / Agentで同一root deltaが同一classificationになる
9. Stale / ShouldReevaluate / ReusePermitted / AutoRefresh等を追加しない

## 8. current conclusion

R3ACで固定するのは:

```text
Reference delta
  != runtime Observation delta
  != test-only context delta
  != corroboration delta
  != locator delta

role-aware change classification
  != authority decision
  != re-evaluation action
```

である。

production watcher、cache invalidation、event bus、runtime Observation storeは追加しない。

## 9. 次

次段ではR3ACのchange classをR3Tのre-evaluation explanationへ接続し、

```text
what root changed
  -> what bounded category is affected
  -> why

  != what action must run
```

を保持できるかをPressureする。
