# Actor Context R3U — Multi-reason Re-evaluation Aggregation Boundary

- Status: ISSUE-000053 / Phase 3 multi-reason aggregation pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3T_PROJECTION_REEVALUATION_REASON_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3S_PROJECTION_DEPENDENCY_REASON_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3R_PROJECTION_DEPENDENCY_PROVENANCE_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3Tではinput changeを単一の `stale=true` へ潰さず、category / change class / dependency / reasonを保持したre-evaluation reasonへ変換した。

次にPressureするのは、複数changeが同時に起きた場合である。

```text
multiple changes
  -> multiple independent reasons
  -> exact duplicate normalization
  != one winning reason
```

R3Uではrepresentation上の安定したaggregationだけを扱い、priorityや実行順序を決めない。

## 2. 一つのcategoryに複数reasonを許す

同じ `machine-reasons` categoryでも、Business Procedure Referenceとprotected-target overlayが同時に変化すれば:

```text
machine-reasons
  <- authorization-requirement-source
  <- protected-resource-source
```

のように複数reasonが成立し得る。

一方をPrimaryへ昇格させない。

## 3. freshness planeを跨いだreasonを保持する

同時に:

- semantic source content change
- locator-only change
- Reference input change
- runtime Observation change
- profile change

が起きても、それぞれのchange classを保持する。

```text
semantic-source-changed
  != locator-refresh-required
  != runtime-observation-changed
```

である。

## 4. exact duplicateだけを統合する

同じreasonが複数経路から重複して得られた場合は、次のidentityが完全一致するときだけ一つへ正規化する。

```text
category
+ change class
+ dependency key
+ dependency reason
```

categoryまたはdependency reasonが異なるものを「似ている」という理由で統合しない。

UnselectedChangeも:

```text
change class
+ dependency key
+ category hint
```

の完全一致だけを重複排除する。

## 5. input orderをprecedenceへしない

changeの入力順、dependency edgeの列挙順、並列処理の完了順からwinnerを作らない。

安定したsortはserialization / comparisonのためだけに使う。

```text
stable order
  != priority
  != precedence
```

である。

## 6. selected reasonとunselected changeを混ぜない

同じbounded projectionに対してselected reasonとして説明できるchangeを、同時にunselected changeとして残さない。

一方、reason edgeを持たないchangeはUnselectedChangesとして保持し、消去しない。

```text
unselected in this projection
  != globally irrelevant
```

を維持する。

## 7. audience間でwinnerを共有しない

Human orientationとAgent resolution-inspectionではselected categoryが異なる。

共有categoryについてreason identityを比較できても:

```text
same reason identity
  != same disclosure set
  != one universal impact packet
```

である。

R3Uはaudience横断packetを作らない。

## 8. test-only source

- `tools/reference-business-procedure-check/actor_context_r3u_multi_reason_aggregation_pressure_test.go`

Pressureするcase:

1. one categoryへ複数independent reasonを保持する
2. semantic / locator / runtime / profile reasonを同時保持する
3. exact duplicateだけをdeduplicateする
4. input orderを変えても同じnormalized representationになる
5. selected reasonとUnselectedChangeの二重計上を除く
6.異なるreasonをPrimary / Winnerへ統合しない
7. Priority / Severity / Winner / FirstCause / AutoRebuildを導入しない

## 9. current conclusion

R3Uで固定するのは:

```text
reason aggregation
  = lossless union
  - exact duplicates

  != ranking
  != precedence
  != rebuild plan
```

である。

production impact aggregation、scheduler、retry policy、event coalescingは追加しない。

## 10. 次

次段ではHuman / Agentの異なるbounded projection間で、共有されたreason identityへ同じprovenanceから戻れるかをPressureする。

表示量は異なっても:

```text
shared reason identity
  -> same dependency provenance
```

を維持し、audience-specific wordingをmachine reason identityへ逆流させない。
