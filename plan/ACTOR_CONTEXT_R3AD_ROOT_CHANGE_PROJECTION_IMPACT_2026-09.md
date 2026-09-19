# Actor Context R3AD — Provenance Root Change / Bounded Re-evaluation Explanation

- Status: ISSUE-000053 / Phase 3 root-to-projection impact explanation pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AC_FRESHNESS_DRILLDOWN_ROOT_CHANGE_CLASSIFICATION_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3T_PROJECTION_REEVALUATION_REASON_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3U_MULTI_REASON_AGGREGATION_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3V_CROSS_AUDIENCE_REASON_IDENTITY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3AA_AUTHORIZATION_FRESHNESS_STOP_HANDOFF_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3ACではfreshness drill-downのprovenance root変更を:

```text
reference-evidence-content-changed
runtime-observation-content-changed
test-only-context-changed
corroboration-changed
locator-refresh-required
provenance-root-set-changed
```

へ分離した。

R3ADではそのchangeをbounded projectionへ接続し:

```text
what root changed
  -> what selected category is affected
  -> why
```

をR3T型のre-evaluation explanationとして保持する。

ただし:

```text
affected category
  != action to execute
  != Authorization reevaluation directive
```

である。

## 2. R3AC change classをgeneric classへ潰さない

R3TにはReference / runtime / locator等のchange classがあるが、R3ACはさらに:

- test-only context
- corroboration
- root-set

を区別している。

R3ADではR3Tのreason tuple:

```text
category
change class
dependency key
dependency reason
```

を再利用する一方、`change class` にはR3ACのrole-aware classを保持してよい。

```text
test-only-context-changed
  != semantic-source-changed

corroboration-changed
  != reference-input-changed
```

を維持する。

## 3. explicit root-to-category binding

root roleだけから全categoryを推論しない。

test-only bindingはroot identity / claim meaningに基づいて明示する。

### Authorization requirement Reference root

content / root-set change:

- `machine-reasons`
- Agentでselectedなら `known-authority-facts`

locator change:

- `source-refs`

### protected resource Reference root

content / root-set change:

- `machine-reasons`
- `resolution-needs`
- Agentでselectedなら `known-authority-facts`

locator change:

- `source-refs`

### prior Authorization Observation root

content / root-set change:

- `machine-reasons`

locator change:

- `source-refs`

重要:

```text
prior Observation changed
  != current runtime-outcome changed
  != known-authority-facts changed
```

である。

R3AA freshness stopではcurrent `RuntimeOutcome` は空のため、prior Observation revisionから `runtime-outcome` categoryを自動impact対象へしない。

### R3X test-only context root

content / root-set change:

- `stop-identity`
- `machine-reasons`
- `resolution-needs`

これはtuple identity / freshness-stop成立条件を再確認するためのbindingである。

```text
test-only context changed
  != production action authority changed
```

を維持する。

### Legacy policy corroboration root

current freshness stop coreではcorroborationはselected authority categoryを所有しない。

したがってcontent / locator changeは:

```text
observed root change
  + no selected category binding
  -> UnselectedChange
```

として保持する。

```text
corroboration changed
  != machine reason changed automatically
```

である。

## 4. Human / Agent差

R3ABとR3Vに従い、Human / Agentで共有されるselected categoryのreason identityは同じにする。

Agentだけが `known-authority-facts` をselectedする場合:

```text
shared machine-reasons impact
  = same provenance identity

Agent-only known-authority-facts impact
  != Human inconsistency
```

である。

audienceをdependency reason identityへ含めない。

## 5. locator-only change

R3ACの `locator-refresh-required` は、explicit locator bindingがあるrootについて `source-refs` だけへ接続する。

```text
locator moved
  -> source-refs affected

  != machine-reasons affected
  != authority meaning changed
```

を維持する。

## 6. root-set change

known rootの削除 / 追加がexplicit binding対象なら、同じmeaning categoryの再評価理由を作れる。

ただしbindingのない新rootをglobal dependencyへ推測しない。

```text
unknown root added
  != all categories affected
```

である。

binding不明のchangeはUnselectedChangeへ残す。

## 7. dependency key

R3ADのtest-only dependency keyは:

```text
provenance-root:<root address>
```

とする。

root addressはR3ACどおり:

```text
Identity + Role
```

であり、locator / ClaimKindをidentityへ混ぜない。

これはproduction dependency graph keyの採用ではない。

## 8. R3T / R3Uとの関係

R3ADはR3Tのreason tuple構造:

- category
- change class
- dependency key
- dependency reason

を再利用するが、R3Tの既存change-class vocabularyをclosed enumとは扱わない。

R3ACでより精密になった:

- test-only-context-changed
- corroboration-changed
- provenance-root-set-changed

をnearest generic classへ強制変換しない。

outputは次を分ける。

- selected re-evaluation reasons
- explicit bindingはあるがbounded viewで選ばれない UnselectedChanges
- binding / semantic mapping自体がない ResidualRootChanges

```text
unselected
  != unmapped
  != ignored
```

である。

同一reasonが複数経路から出る場合はR3Uどおりexact identityだけをdeduplicateする。

```text
root delta explanation
  != rebuild plan
  != scheduler instruction
```

である。

## 9. test-only source

- `tools/reference-business-procedure-check/actor_context_r3ad_root_change_projection_impact_pressure_test.go`

Pressureするcase:

1. requirement Reference changeがHuman / Agent共通のmachine-reasonsへ同じidentityで接続する
2. Agentだけknown-authority-facts detailを追加できる
3. prior Observation changeがmachine-reasonsへ接続してもruntime-outcomeを捏造しない
4. test-only context changeがstop-identity / machine-reasons / resolution-needsへ接続する
5. corroboration changeはbindingなしのResidualRootChangeとして保持する
6. locator-only changeはsource-refsだけへ接続する
7. explicit bindingのないroot-set changeをResidualRootChangeとして保持し、global impactへしない
8. UnselectedChangeとResidualRootChangeを混同しない
9. audience / input orderでshared reason identityを変えない
10. ShouldReevaluate / AutoReauthorize / Retry / CacheValid / Winner等を追加しない

## 10. current conclusion

R3ADで固定するのは:

```text
role-aware provenance delta
  + explicit bounded-category binding
  -> explainable re-evaluation reason

root changed
  != all categories changed

prior Observation changed
  != current runtime outcome changed

corroboration changed
  != authority changed

unselected
  != unmapped

affected category
  != response action selected
```

である。

production impact graph、watcher、scheduler、cache invalidation、runtime Observation storeは追加しない。

## 11. 次

次段ではR3ADのre-evaluation reasonとR3Wのcategory fingerprint比較を接続し:

```text
root changed
  -> category re-evaluated
  -> category meaning unchanged / changed
```

を確認する。

特にfreshness stop reasonが再評価対象になっても、同じ `evaluation-freshness-authority-unresolved` に留まるcaseを正常系として扱う。
