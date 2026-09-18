# Actor Context R3N — Stage-aware Operational Stop Taxonomy

- Status: ISSUE-000053 / Phase 3 stop-taxonomy pressure / Reference-only
- Date: 2026-09-18
- Parent: `plan/ACTOR_CONTEXT_R3M_OPERATIONAL_STOP_EXPLANATION_HANDOFF_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3B_AUTHORIZATION_PREREQUISITE_BOUNDARY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3I_AUTHORIZATION_TARGET_EVALUATION_CADENCE_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3J_AUTHORIZATION_ACTION_AUTHORITY_GATE_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3Mでは、current BP-06の:

```text
Authorization action authority = unresolved
```

を、denied / unauthorized / failedへ丸めず説明する境界を固定した。

次は、異なる停止状態を一つのgeneric `authorization error` へ潰さないことをPressureする。

R3Nでは少なくとも:

```text
requirement authority unresolved
principal unresolved
resource authority unresolved
action authority unresolved
evaluation not attempted
evaluation observation mismatch
evaluation denied
current-context evaluation unresolved
```

を別のstop classとして保持する。

## 2. stop stageとstop classを分離する

同じAuthorization領域でも停止位置と意味は異なる。

例:

```text
stage = authorization-requirement-authority
class = authority-unresolved

stage = authorization-evaluation
class = evaluation-not-attempted

stage = authorization-evaluation
class = evaluated-denied
```

したがって:

```text
same subsystem
  != same stop meaning
```

である。

## 3. earliest blockerを保持する

R3Bの順序をR3Nでも維持する。

```text
requirement unresolved
  -> requirement authority stop

principal unresolved
  -> principal stop

resource/action authority unresolved
  -> target authority stop

target resolved + no evaluation
  -> evaluation-not-attempted
```

上流blockerがある場合、後段の状態をcurrent blockerへ昇格しない。

後段は必要なら:

```text
downstream not attempted
```

として別に示す。

## 4. missing authority classとmissing valueを分離する

authority未解決stateでは:

```text
missing authority class = authorization-action-authority
```

までは説明できる。

しかし:

```text
missing action value = execute
```

を推測しない。

同様にresource authority未解決でも、Pattern identityやLegacy policy resourceをmissing valueとして補わない。

## 5. evaluation not attemptedとdeniedを分離する

R3IではAuthorization Observationが:

```text
subject + action + resource
```

のcurrent tupleに一致し、outcomeが `allowed` / `denied` のときだけcurrent evaluationとして扱われる。

したがって:

```text
no matching Observation
  -> evaluation-not-attempted / observation-mismatch / reuse-unresolved

matching Observation + denied
  -> evaluated-denied
```

である。

`denied` はauthority不足ではなくruntime evaluation outcomeである。

```text
authority unresolved
  != evaluated denied
```

をtaxonomy上でも固定する。

## 6. Observation mismatchをdenialへ丸めない

subject / action / resource / outcome vocabularyがcurrent contextに一致しないObservationは:

```text
evaluation-observation-mismatch
```

として独立に保持する。

これは:

- denied
- not evaluated
- stale reuse accepted

のいずれでもない。

mismatched Observationからauthorization resultを生成しない。

## 7. cross-context reuse unresolvedを独立に保持する

R3Iではprior evaluationが同じsubject/action/resourceでも、別Business contextでreuse authorityがなければ:

```text
authorization-current-context-evaluation-unresolved
```

となる。

R3Nではこれを単なる `not-evaluated` へ潰さない。

理由は:

```text
matching prior Observation exists
  + reuse authority unresolved
```

という追加情報があるためである。

## 8. explanation projection

R3Nのtest-only projectionは最低限:

```text
Business address
stop stage
stop class
machine-readable reasons
known authority facts + provenance
resolution need classes
runtime outcome if observed
downstream not attempted
source refs
authority limits
```

を保持する。

ただし:

- HandoffOwner
- SuggestedAction
- Approved
- Authorized
- CanExecute
- Confidence

等のshortcutは持たない。

## 9. current BP-06

current BP-06はR3Mどおり:

```text
stop stage = authorization-action-authority
stop class = authority-unresolved
reason = authorization-action-authority-unresolved
missing authority class = authorization-action-authority
```

となる。

Legacy policy `execute` やBusiness wordingからmissing valueを補わない。

## 10. test-only source

- `tools/reference-business-procedure-check/actor_context_r3n_operational_stop_taxonomy_pressure_test.go`

Pressureするcase:

1. requirement authority unresolved
2. principal unresolved
3. resource authority unresolved
4. action authority unresolved
5. evaluation not attempted
6. exact current Observation denied
7. Observation tuple mismatch
8. prior-context reuse unresolved

Pressureする不変条件:

- distinct stop statesをgeneric errorへ潰さない
- authority unresolvedとruntime deniedを分離する
- earliest blockerを保持する
- missing authority classからmissing valueを推測しない
- mismatchをdeniedへ変換しない
- prior-context reuse unresolvedをnot-evaluatedへ丸めない
- downstream not-attemptedをcurrent blockerへ昇格しない
- handoff owner / next actionを生成しない

## 11. current conclusion

R3Nで固定するのは:

```text
fail closed
  + stage-aware explanation
  + exact blocker preservation
  + runtime outcome preservation
  + authority provenance preservation
```

である。

```text
authorization problem
```

という一語へ集約しない。

production Agent protocol、UI message schema、ticket taxonomy、telemetry contractは追加しない。

## 12. 次

次段ではR3N taxonomyからhuman / agent向けProjectionへ何を露出すべきかを比較する。

特に:

- machine-readable reason
- short human explanation
- drill-down refs
- known facts
- missing authority class
- runtime outcome

のうち、どれを共通最小shapeにするかをPressureする。

表示文そのものをauthorityやstable API contractへしない。
