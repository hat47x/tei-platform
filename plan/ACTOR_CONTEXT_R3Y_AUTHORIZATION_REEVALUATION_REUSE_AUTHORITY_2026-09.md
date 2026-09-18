# Actor Context R3Y — Authorization Re-evaluation / Prior Observation Reuse Authority Boundary

- Status: ISSUE-000053 / Phase 3 Authorization freshness-authority pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3X_AUTHORIZATION_CONTEXT_IDENTITY_BOUNDARY_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3I_AUTHORIZATION_TARGET_EVALUATION_CADENCE_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3W_CATEGORY_VALUE_CHANGE_BOUNDARY_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3Xでは、prior Authorization Observationとcurrent contextについて:

```text
tuple mismatch
  -> prior Observation is not current-context evidence

tuple match
  != freshness proven
  != reuse authorized
```

を固定した。

R3Yではtuple match後に残る二つの問いを分ける。

```text
A. current contextでAuthorizationを再評価しなければならないか
B. prior Observationをcurrent decision evidenceとして再利用してよいか
```

この二つを一つの `fresh=true/false` や `cache-valid` に畳まない。

## 2. 二つのauthority axis

test-onlyでは次を独立に保持する。

### Re-evaluation authority

```text
reevaluation authority status
  = unresolved
  | explicit-reevaluation-required
```

### Reuse authority

```text
prior Observation reuse authority status
  = unresolved
  | explicit-reuse-permitted
```

ここではnegative/default ruleを作らない。

```text
no reevaluation directive
  != reevaluation not required

no reuse directive
  != reuse prohibited
```

である。

## 3. static semantic changeはpressureであってauthorityではない

same tupleのまま:

- Authorization requirement source
- protected target authority
- Application / Pattern meaning

等が変化した場合、prior Observationの意味を再確認するpressureは生じる。

しかし:

```text
static semantic change observed
  != explicit reevaluation authority
  != explicit reuse permission
```

である。

R3Yはchangeを理由に自動reauthorizeもしなければ、自動reuseもしない。

## 4. requirement transitionをfreshness ruleで上書きしない

R3Xで:

```text
requirement applicability changed
  -> requirement-transition-authority-unresolved
```

となるcaseは、freshness / reuse authority以前の問題である。

したがって:

```text
reuse permission
  -X-> resolve requirement applicability

reevaluation directive
  -X-> author requirement applicability
```

とする。

Authorizationがそもそもrequiredかどうかのauthorityを、evaluation freshness ruleで補わない。

## 5. protected target authority revision

same tuple valueが維持されても、protected targetのauthority source自体が変更された場合がある。

このとき:

```text
resource value unchanged
  != target authority unchanged
```

である。

R3Yではtarget authority revisionをreevaluation pressureとして保持できるが、それだけで:

- reevaluation-required
- reuse-permitted
- reuse-prohibited

のどれもauthorしない。

## 6. policy provider revisionは別authority

Legacy / provider Authorization policyが変更された場合も同じである。

```text
provider policy changed
  != Business/Application requirement authority changed
  != protected target authority changed
  != reevaluation directive authored
  != prior Observation reuse permission authored
```

providerがactual Authorization evaluation semanticsを所有する場合でも、そのrevisionをどう扱うかというfreshness contractは別authorityとして必要である。

R3Yではprovider revisionをfreshness decisionそのものへ昇格させない。

## 7. explicit directiveだけをtest-only authorityとして認める

R3Yではproduction syntaxを追加せず、test-only authority evidenceとして次の二種類だけを比較する。

```text
evaluation-freshness-directive:
  reevaluation-required

evaluation-freshness-directive:
  prior-observation-reuse-permitted
```

これらにはsource provenanceを要求する。

ただし:

```text
explicit directive
  != production field adopted
  != provider cache contract adopted
```

である。

## 8. 一方のaxisから他方を推論しない

explicit `reevaluation-required` が存在しても:

```text
reuse authority status = unresolved
```

のまま保持できる。

explicit `prior-observation-reuse-permitted` が存在しても:

```text
reevaluation authority status = unresolved
```

を自動的に `not-required` へしない。

二つが同時に存在する場合も、R3Yではprecedence / compatibility ruleを発明しない。

```text
both explicit
  -> coexistence / precedence unresolved
  != last-writer-wins
  != more-specific-wins
```

とする。

## 9. tuple mismatchではR3Y reuse比較へ進まない

R3Xでtuple mismatchならprior Observationはcurrent-context evidenceではない。

したがって:

```text
tuple mismatch
  -> R3Y prior-observation reuse comparison not applicable
```

である。

これはreuse prohibitedという意味ではなく、比較対象となる「同一contextのprior Observation」という前提が成立していない。

## 10. locator / presentation-only change

source locatorやHuman表示文だけが変わり、Authorization tuple / authority meaningが変わっていない場合:

```text
locator/presentation refresh
  != reevaluation pressure
  != reuse authority change
```

とする。

R3Tのlocator-refresh reasonをAuthorization freshnessへ逆流させない。

## 11. current Reference conclusion

current R3X pathではsame-tuple static changeに対して:

```text
freshness authority status = unresolved
```

を保持している。

R3Yでもcurrent Referenceからaccepted production:

- reevaluation directive
- prior Observation reuse permission
- TTL / expiry rule
- revocation freshness contract

を新たに推論しない。

したがってcurrent defaultは:

```text
reevaluation authority = unresolved
reuse authority = unresolved
```

である。

## 12. test-only source

- `tools/reference-business-procedure-check/actor_context_r3y_authorization_reevaluation_reuse_authority_pressure_test.go`

Pressureするcase:

1. same tuple + static semantic change + no explicit directive -> both axes unresolved
2. explicit reevaluation directive -> reevaluation required / reuse unresolved
3. explicit reuse permission -> reuse permitted / reevaluation unresolved
4. both explicit -> both preserved / precedence unresolved
5. requirement transition -> freshness authority does not resolve applicability
6. provider policy revision only -> both axes unresolved
7. locator-only change -> no Authorization freshness pressure
8. tuple mismatch -> reuse comparison not applicable
9. Reusable / CacheValid / ShouldReevaluate / TTL / Winner等のshortcutを作らない

## 13. current conclusion

R3Yで固定するのは:

```text
context identity
  != evaluation freshness authority

reevaluation authority
  != reuse authority

change pressure
  != authority

explicit reevaluation directive
  != implicit reuse prohibition

explicit reuse permission
  != implicit reevaluation-not-required
```

である。

production Authorization cache、TTL、revocation watcher、automatic reevaluation schedulerは追加しない。

## 14. 次

次段では二つのexplicit directiveが同時に存在するcaseについて、production precedence ruleを作る前に:

- 同一provenanceから出た整合directiveか
- 異なるauthority source間の衝突か
- scopeが同一か
- current Repositoryにqualifying conflict caseが存在するか

をEvidence GateとしてPressureする。

R3G / R3Hで採った「exception / precedence mechanismをEvidenceなしに作らない」原則へ戻す。
