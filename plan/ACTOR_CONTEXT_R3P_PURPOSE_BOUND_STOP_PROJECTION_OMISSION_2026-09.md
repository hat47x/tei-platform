# Actor Context R3P — Purpose-bound Stop Projection / Explicit Omission Boundary

- Status: ISSUE-000053 / Phase 3 bounded-projection pressure / Reference-only
- Date: 2026-09-18
- Parent: `plan/ACTOR_CONTEXT_R3O_STOP_PROJECTION_CORE_2026-09.md`
- Related: `plan/ACTOR_CONTEXT_PROJECTIONS_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3OではHuman / Agent surfaceの下にaudience-neutral semantic coreを置いた。

次は、そのcoreを各purposeへ必要十分に絞る。

```text
shared semantic core
  -> purpose-bound selection
  -> bounded Human / Agent projection
```

ここで最も重要なのは:

```text
omitted
  != absent
```

である。

bounded projectionに含まれないcategoryを、sourceにも存在しない意味へ変換しない。

## 2. category vocabulary

R3Pではtest-onlyに次のcategoryを使う。

```text
stop-identity
machine-reasons
known-authority-facts
resolution-needs
runtime-outcome
downstream-not-attempted
source-refs
resolution-request
handoff-recipient-status
authority-limits
```

production field / enumではない。

profileはこの集合をselected / omittedへ完全分割し、重複やunknown categoryを許さない。

## 3. Human orientation view

Human向けcompact orientationでは、まず停止理由と追跡先が分かればよい。

test-only selection:

```text
selected:
  stop-identity
  machine-reasons
  resolution-needs
  runtime-outcome
  source-refs

omitted:
  known-authority-facts
  downstream-not-attempted
  resolution-request
  handoff-recipient-status
  authority-limits
```

省略された詳細はpayloadから物理的に落とす。

ただし:

- `OmittedCategories`
- `DrillDownRefs`
- unresolved category

を残し、存在しないと誤認させない。

## 4. Agent resolution-inspection view

Agent向けresolution inspectionではR3O coreを厚く保持できる。

test-onlyでは全categoryをselectedとする。

ただしHuman表示文は持たせない。

```text
machine detail selected
  != Agent next action authorized
```

である。

NextAction / SuggestedAction / HandoffOwner / CanExecute等は導入しない。

## 5. selected-but-empty と omitted を分ける

BP-06 action-authority stopではruntime outcomeはまだない。

Human compact viewでも `runtime-outcome` category自体はselectedである。

```text
runtime-outcome selected
value = empty

  !=

runtime-outcome omitted
```

selected emptyは「このcategoryを評価した結果、current coreに値がない」。

omittedは「このbounded viewでは表示していない」。

この差をmetadataで保持する。

## 6. unresolved は omission ではない

BP-06では:

```text
resolution need = authorization-action-authority
```

がある。

Human viewがknown facts等を省略しても、このunresolvedを消さない。

```text
detail omitted
  != unresolved resolved
  != unresolved absent
```

denied caseではresolution needが本当に空なので、selected `resolution-needs` が空になる。

## 7. drill-down boundary

何かを省略するHuman viewではSourceRefsをselectedのまま残し、DrillDownRefsとしても提示する。

```text
bounded display
  + explicit omission
  + drill-down refs
  -> recoverable detail

drill-down available
  != detail disclosed inline
```

SourceRefsまで省略してopaque summaryにしない。

## 8. test-only source

- `tools/reference-business-procedure-check/actor_context_r3p_stop_projection_omission_pressure_test.go`

Pressureするcase:

1. BP-06 Human compact viewで詳細を物理的に省略する
2. omitted categoryを明示する
3. source drill-downを保持する
4. unresolved action authorityを省略で消さない
5. selected-but-empty runtime outcomeとomittedを分ける
6. Agent resolution viewはfull machine coreを保持しHuman display textを持たない
7. deniedではresolution-needs selected + emptyを保持する
8. category partitionの重複 / unknown / 欠落をfail closedする

## 9. current conclusion

R3Pで固定するのは:

```text
purpose-bound projection
  != universal packet

omitted
  != absent

selected empty
  != omitted

unresolved
  != omitted

bounded projection
  != second source of truth
```

である。

production audience profile、disclosure policy、wire schema、UI layoutは追加しない。

## 10. 次

次段ではaudienceごとのactual dependency / freshnessをPressureする。

Human compact viewで物理的にomitされたdetailが変更されたとき、そのviewを本当にstaleにすべきかを検討する。

```text
omitted source detail changed
  != bounded projection automatically stale
```

とは限らない。

ただしdrill-down/ref integrityやselected meaningへ影響する変更はfreshnessへ反映する必要がある。
