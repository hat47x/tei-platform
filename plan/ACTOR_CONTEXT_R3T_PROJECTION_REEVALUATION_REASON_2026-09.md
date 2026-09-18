# Actor Context R3T — Projection Re-evaluation Reason / Impact Explanation

- Status: ISSUE-000053 / Phase 3 re-evaluation explanation pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3S_PROJECTION_DEPENDENCY_REASON_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3Q_PROJECTION_SELECTIVE_FRESHNESS_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3R_PROJECTION_DEPENDENCY_PROVENANCE_2026-09.md`
  - ISSUE-000039
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3Sでは:

```text
selected category
  -> dependency
  -> reason
```

を保持できるようにした。

次は実際にinput changeが観測されたとき:

```text
what changed
  -> which selected category is affected
  -> why
  -> what kind of re-evaluation is needed
```

を説明できるかをPressureする。

R3Tはproduction impact engineではない。

## 2. re-evaluation reasonをbooleanへ潰さない

R3Tでは少なくとも次を区別する。

```text
semantic-source-changed
reference-input-changed
locator-refresh-required
runtime-observation-changed
profile-changed
```

したがって:

```text
stale = true
```

だけでは表現しない。

同じbounded projectionでも、必要な再評価の意味が異なるためである。

## 3. semantic subject change

Source Index subjectのcontent fingerprintが変わり、そのdependency keyへselected categoryのR3S edgeがある場合:

```text
category
  -> semantic-source-changed
  -> R3S dependency reason
```

として説明する。

locatorが同じでもsemantic staleになり得る。

## 4. Reference input change

Business Procedure Reference / protected-target overlayのcontentが変わった場合、そのReference dependencyへedgeを持つselected categoryだけを影響対象とする。

current BP-06なら例として:

```text
Business Procedure Reference changed
  -> stop-identity
     reason = business-address-source
  -> machine-reasons
     reason = authorization-requirement-source
```

があり得る。

Human viewでomittedされたknown-authority-factsは、このviewのre-evaluation対象へ含めない。

## 5. locator-only change

R3Rで:

```text
semantic meaning current
locator refresh required
```

となる変更は、R3Sの:

```text
source-refs
  <- drill-down-locator
```

edgeへ接続する。

この場合:

```text
source-refs
  -> locator-refresh-required
```

であり、machine-reasonsをsemantic staleへしない。

## 6. runtime Observation change

exact current Authorization Observationが:

```text
allowed -> denied
```

等に変化した場合、runtime-outcome categoryのexact-runtime-observation edgeを通して:

```text
runtime-outcome
  -> runtime-observation-changed
```

と説明する。

static Business / Application source changeへ読み替えない。

## 7. profile change

selected / omitted partition自体が変わった場合:

```text
category selected -> omitted
category omitted -> selected
```

を:

```text
profile-changed
```

として明示する。

これはsource meaning changeではない。

audience / purpose変更をglobal stale-allへ丸めない。

## 8. selected viewに関係しないchange

R3R envelopeにchangeがあっても、current bounded viewのR3S reason edgeがなければ:

```text
change observed
  + no selected dependency edge
  -> no selected-category re-evaluation reason
```

とする。

これは:

```text
change irrelevant globally
```

という意味ではない。

あくまで:

```text
not selected by this bounded projection
```

である。

R3P / R3Qの:

```text
omitted
  != absent
```

を維持する。

## 9. explanation shape

test-only concept:

```text
ReevaluationReason
  category
  changeClass
  dependencyKey
  dependencyReason
```

影響しないchangeは別に:

```text
UnselectedChange
  changeClass
  dependencyKey
```

として保持できる。

次を置かない。

- Stale boolean
- RebuildAll
- AutoRebuild
- ImpactWinner
- NextAction
- Confidence

## 10. ISSUE-000039との境界

R3TはActor Context / bounded projectionのtest-only説明pressureである。

ISSUE-000039のproduction impact analysisへ:

- graph
- invalidation engine
- event routing
- incremental rebuild
- persistence

を前倒ししない。

将来ISSUE-000039がactual impact engineを提供する場合、R3Tのreason分類をEvidenceとして比較できる。

## 11. test-only source

- `tools/reference-business-procedure-check/actor_context_r3t_projection_reevaluation_reason_pressure_test.go`

Pressureするcase:

1. Reference input content changeがedgeを持つselected categoryだけへimpact reasonを作る
2. omitted known-authority-facts changeはHuman viewのimpact reasonへ入れない
3. semantic subject content changeをsemantic-source-changedとして説明する
4. locator-only moveをsource-refs / locator-refresh-requiredだけへ接続する
5. runtime Observation outcome changeをruntime-outcomeへ接続する
6. profile partition changeをprofile-changedとして説明する
7. dependency reason codeを失わない
8. Stale / RebuildAll / Winner / Confidence等のshortcutを作らない

## 12. current conclusion

R3Tで固定するのは:

```text
change
  + bounded selection
  + dependency provenance
  + dependency reason
  -> explainable re-evaluation reasons
```

である。

```text
unrelated change
  != stale-all
```

と:

```text
affected
  != automatically rebuilt
```

を同時に維持する。

## 13. 次

次段では複数changeが同時に起きた場合に:

- duplicate reasonを統合できるか
- semantic / locator / runtimeの独立理由を失わないか
- one categoryへ複数reasonがあることを許せるか
- order-dependentなwinner選択をしないか

をPressureする。
