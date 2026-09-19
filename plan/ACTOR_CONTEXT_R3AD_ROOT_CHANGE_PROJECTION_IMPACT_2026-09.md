# Actor Context R3AD — Provenance Root Delta / Existing Projection Dependency Binding

- Status: ISSUE-000053 / Phase 3 root-delta handoff pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AC_FRESHNESS_DRILLDOWN_ROOT_CHANGE_CLASSIFICATION_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3S_PROJECTION_DEPENDENCY_REASON_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3T_PROJECTION_REEVALUATION_REASON_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3U_MULTI_REASON_AGGREGATION_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3ACではprovenance root deltaをrole-awareに分類した。

R3ADでは、そのdeltaをR3Tのbounded category re-evaluation reasonへ接続する。

ただし:

```text
provenance root changed
  != projection dependency changed
```

である。

R3ADで許すのは:

```text
root delta
  + explicit root-to-existing-dependency binding
  + selected R3S dependency reason edge
  -> R3T bounded re-evaluation reason
```

だけである。

## 2. parallel dependency keyを作らない

R3R / R3Sは既にprojection dependency keyを持つ。

Reference inputなら:

```text
reference:<input-class>:<locator>
```

である。

R3ADは新しい `provenance-root:<root-address>` をparallel dependency graphとして作らない。

R3AB root identityとR3R dependency identityは別なので、explicit bindingで接続する。

```text
root identity
  != dependency identity
```

である。

## 3. current freshness stopでbindingできるroot

current BP-06 freshness stopで、既存R3R Reference dependencyへbindingできるrootは:

```text
Business Procedure Reference root
  -> reference:business-procedure-reference:<locator>

protected-target overlay root
  -> reference:protected-target-overlay:<locator>
```

である。

bindingはroot roleやlocator類似から自動推測せず、test-only fixtureで明示する。

## 4. R3ACからR3Tへはlosslessなchange classだけを写す

explicit dependency bindingがあるrootについてのみ:

```text
reference-evidence-content-changed
  -> reference-input-changed

locator-refresh-required
  -> locator-refresh-required

runtime-observation-content-changed
  -> runtime-observation-changed
```

を許す。

ただしruntime mappingはexact runtime dependency bindingが存在する場合だけである。

current R3Tに同値classがない:

```text
test-only-context-changed
corroboration-changed
provenance-root-set-changed
```

は近いclassへ強制変換しない。

## 5. residualを二種類に分ける

selected impactへ変換できないroot deltaを消さない。

### A. dependency binding unresolved

current projection dependencyへのexplicit bindingがない場合:

```text
projection-dependency-binding-unresolved
```

として保持する。

### B. R3T change class unmapped

bindingは存在してもexact R3T classがない場合:

```text
r3t-change-class-unmapped
```

として保持する。

```text
residual
  != globally irrelevant
```

である。

## 6. R3S reason edgeがselected impactを決める

root-to-dependency bindingだけではcategory impactを作らない。

既存R3S reason edgeを通し、current bounded viewでselectedされたcategoryだけをR3T reasonへする。

Business Procedure Reference content deltaなら:

```text
Human
  stop-identity
  machine-reasons

Agent
  stop-identity
  machine-reasons
  known-authority-facts
```

となり得る。

Human / Agent共通categoryのreason identityは同じdependency key / reason codeへ戻る。

## 7. locator-only change

Reference rootのlocator-only deltaはsame dependency bindingを使い:

```text
source-refs
  -> locator-refresh-required
```

だけへ接続する。

```text
locator moved
  != machine-reasons semantic change
```

を維持する。

content + locatorが同時に変われば両reasonを保持する。

## 8. prior Observationをcurrent runtime dependencyへ昇格させない

R3AB runtime rootはprior Authorization Observationである。

R3AA freshness authority unresolved中:

```text
core.RuntimeOutcome = empty
```

であるため、current freshness-stop binding setにはruntime-outcome dependencyを置かない。

prior Observationが `allowed -> denied` へ変化しても:

```text
runtime root delta
  -> residual: projection-dependency-binding-unresolved

  != current runtime-outcome changed
```

である。

actual current Authorization Observationを扱うR3R/R3S denied caseとは別である。

## 9. test-only context / corroborationも自動bindingしない

R3ABの:

- test-only Authorization context
- Legacy policy corroboration

はcurrent freshness-stop R3R dependency envelopeへ存在しない。

したがってroleだけでbounded categoryへbindingしない。

将来consumer dependencyがexplicitに成立した場合にのみ別途bindingを検討する。

## 10. bindingあり / selected reasonなし

explicit dependency bindingが存在しても、current R3S reason envelopeにselected edgeがない場合はR3Tの `UnselectedChanges` へ残す。

```text
bound dependency
  != selected category
```

である。

## 11. test-only source

- `tools/reference-business-procedure-check/actor_context_r3ad_root_change_projection_impact_pressure_test.go`

Pressureするcase:

1. Business Procedure content delta -> existing Reference dependency -> Human / Agent共通reason
2. Agentだけknown-authority-facts reasonを追加できる
3. overlay locator-only delta -> source-refsだけへ接続
4. overlay content + locator delta -> semantic / locator reasonをlosslessに保持
5. prior Observation delta -> current runtime-outcomeへbindingせずresidual保持
6. test-only context / corroboration delta -> binding unresolved residual
7. bound Reference root-set change -> exact R3T classがないためunmapped residual
8. bindingなしReference delta -> role推測せずresidual保持
9. bindingあり / selected R3S edgeなし -> R3T UnselectedChangesへ保持
10. Stale / RebuildAll / AutoRefresh / ShouldReevaluate / Authorized等を追加しない

## 12. current conclusion

R3ADで固定するのは:

```text
root delta
  != projection impact

root delta
  + explicit existing-dependency binding
  + selected dependency reason
  -> bounded re-evaluation reason

no binding / no exact class
  -> residual preserved
```

である。

production dependency registry、impact graph、watcher、scheduler、cache invalidation、auto reauthorizationは追加しない。

## 13. 次

次段ではR3AD outputをR3Uへ渡し:

```text
selected reasons
+ R3T unselected changes
+ R3AD residual root deltas
```

を複数root deltaのaggregationでもlosslessに保持できるかをPressureする。
