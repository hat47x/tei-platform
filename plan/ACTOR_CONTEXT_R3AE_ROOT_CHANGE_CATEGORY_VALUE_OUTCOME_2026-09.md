# Actor Context R3AE — Root Change / Category Re-evaluation Outcome Boundary

- Status: ISSUE-000053 / Phase 3 root-change category outcome pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AD_ROOT_CHANGE_PROJECTION_IMPACT_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3W_CATEGORY_VALUE_CHANGE_BOUNDARY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3U_MULTI_REASON_AGGREGATION_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3AA_AUTHORIZATION_FRESHNESS_STOP_HANDOFF_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3ADでは:

```text
provenance root change
  -> selected bounded category
  -> re-evaluation reason
```

を説明できるようにした。

R3AEでは、そのreasonをR3Wのactual category fingerprint比較へ接続する。

```text
root changed
  -> category re-evaluation reason
  -> category recomputation
  -> before / after fingerprint comparison
  -> changed / unchanged after re-evaluation
```

とする。

## 2. reasonありでもmeaning不変を正常系にする

root inputが変わっても、再構成後のbounded category meaningが同じ場合がある。

```text
root change observed
  + selected re-evaluation reason
  + same category fingerprint
  -> unchanged-after-reevaluation
```

である。

これはchangeを無視したという意味ではない。

## 3. actual meaning changeを別段で確認する

同じreasonでもafter category fingerprintが変われば:

```text
status = changed-after-reevaluation
```

とする。

```text
root changed
  != category meaning changed

category meaning changed
  != automatic action
```

を維持する。

## 4. prior Observation revision

R3AA freshness stopではprior Authorization Observationはprovenance rootだがcurrent `RuntimeOutcome` ではない。

したがって:

```text
prior Observation allowed -> denied
  -> machine-reasons re-evaluation reason

  != runtime-outcome category changed
```

である。

freshness authorityが未解決のままなら、machine reasonが同じ
`authorization-evaluation-freshness-authority-unresolved`
に留まることを正常系とする。

## 5. locator change

locator-only root changeではR3ADどおり `source-refs` だけを再評価する。

locator reasonが存在してもactual `source-refs` valueを更新しなければ `unchanged-after-reevaluation`。

actual locatorを反映したafter coreなら `changed-after-reevaluation`。

## 6. bounded audience difference

requirement Reference root changeでAgent-only `known-authority-facts` bindingがある場合:

- Human: omitted categoryなのでselected resultへ出ない
- Agent: selected categoryなのでactual fingerprintを比較する

とする。

```text
Human omission
  != Agent reason invalid

Agent-only changed result
  != Human inconsistency
```

である。

## 7. ResidualRootChangeはcategory change authorityではない

R3ADのResidualRootChangeにはbinding / semantic mapping自体がない。

したがって:

```text
ResidualRootChange exists
  != selected category change explained
```

である。

ResidualRootChangeしかないのにselected category fingerprintが変われば、R3Wどおりunexplained category changeとしてfail closedする。

## 8. test-only source

- `tools/reference-business-procedure-check/actor_context_r3ae_root_change_category_value_outcome_pressure_test.go`

Pressureするcase:

1. requirement root reason + unchanged machine-reasons -> unchanged-after-reevaluation
2. same reason + actual machine-reasons change -> changed-after-reevaluation
3. prior Observation allowed -> denied + freshness stop unchanged -> machine-reasons unchanged / runtime-outcome resultなし
4. locator reason + source-refs不変/変化を分離
5. requirement root change + Agent known-authority-facts actual change -> Agent changed / Human omitted
6. corroboration ResidualRootChangeだけではselected category changeを説明しない
7. Stale / ShouldReevaluate / AutoRebuild / Retry / NextAction等を追加しない

## 9. current conclusion

R3AEで固定するのは:

```text
root delta
  -> reason

reason
  -> what to recompute

category fingerprint
  -> whether meaning actually changed
```

という三段分離である。

production rebuild engine、scheduler、cache invalidation、Authorization reevaluation policyは追加しない。

## 10. 次

次段ではR3AEのchanged / unchanged outcomeをHuman / Agent explanationへ戻す際に:

- input changed
- category re-evaluated
- meaning unchanged / changed
- residual unmapped

を一つのgeneric `stale` へ潰さず説明できるかをPressureする。
