# Actor Context R3AF — Category Re-evaluation Outcome / Operational Explanation Boundary

- Status: ISSUE-000053 / Phase 3 re-evaluation outcome explanation pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AE_ROOT_CHANGE_CATEGORY_VALUE_OUTCOME_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3AA_AUTHORIZATION_FRESHNESS_STOP_HANDOFF_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3Y_AUTHORIZATION_REEVALUATION_REUSE_AUTHORITY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3W_CATEGORY_VALUE_CHANGE_BOUNDARY_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3AEでは:

```text
root delta
  -> selected re-evaluation reason
  -> category fingerprint comparison
  -> changed / unchanged-after-reevaluation
```

まで分離した。

R3AFではこの結果をOperational explanationへ戻す。

ただし最も重要なのは:

```text
category unchanged-after-reevaluation
  != Authorization freshness resolved
  != prior Observation reusable
  != safe / fresh / authorized
```

である。

## 2. 二つの状態軸を分離する

R3AF test-only explanationでは少なくとも:

```text
projection category outcome
Authorization freshness authority status
```

を別々に保持する。

current freshness-stop scenarioでは:

```text
reevaluation authority = unresolved
reuse authority = unresolved
```

のまま、category側だけが:

```text
machine-reasons = unchanged-after-reevaluation
```

になり得る。

これは「再計算したprojection meaningが同じだった」という意味だけである。

## 3. changedもnext actionを決めない

categoryが `changed-after-reevaluation` でも:

- retry
- reauthorize
- rebuild
- human approval
- handoff owner

を自動選択しない。

```text
meaning changed
  != response action selected
```

を維持する。

## 4. residual / unselectedも別に説明する

R3AD / R3AEで残る:

- R3T UnselectedChanges
- R3AD ResidualRootDeltas

をcategory outcomeへ吸収しない。

```text
selected category unchanged
  + residual root delta exists
  != all observed changes resolved
```

である。

prior Authorization Observation revisionのようなno-binding residualは、current runtime outcomeやcategory changeへ変換せず残す。

## 5. Human / Agent

Human / Agentのshared category resultは同じidentity / statusを持つ。

Agentだけselectedする `known-authority-facts` はAgent explanationだけに追加できる。

```text
different disclosed result set
  != different shared category truth
```

とする。

ResidualRootDeltaの詳細開示もpurpose-boundにできるが、Humanでinline省略したことを「residualなし」へ変換しない。

## 6. display wordingはderived

R3AFはstable API textを作らない。

Human表示例として概念上:

```text
入力の変更を受けて再確認しましたが、
現在表示している停止理由の意味は変わっていません。
Authorization freshness authorityは引き続き未解決です。
```

と説明できる。

ただし文言はderived presentationでありauthorityではない。

## 7. test-only source

- `tools/reference-business-procedure-check/actor_context_r3af_reevaluation_outcome_explanation_pressure_test.go`

Pressureするcase:

1. requirement Reference change + machine-reasons unchanged -> unchanged outcome + freshness authority unresolved
2. actual machine-reasons change -> changed outcome + freshness authority unresolved
3. unchanged outcomeからFresh / Safe / Reusableを生成しない
4. prior Observation no-binding residual -> category outcomeなし / residual保持
5. Human / Agent shared machine-reasons outcome identityが一致する
6. Agent-only known-authority-facts outcomeを許す
7. unselected / residualをresolvedへ丸めない
8. Retry / Reauthorize / Approval / NextAction / HandoffOwner等を追加しない

## 8. current conclusion

R3AFで固定するのは:

```text
category outcome
  = projection meaning comparison

freshness authority status
  = separate authorization authority question

unchanged category meaning
  != freshness authority resolved

changed category meaning
  != next action selected
```

である。

production UI schema、notification transport、ticket routing、automatic reevaluation schedulerは追加しない。

## 9. 次

次段では、このexplanationをHuman / Agentへpurpose-boundにmaterializeするとき:

- category outcome detail
- residual detail
- unresolved freshness authority
- provenance drill-down

をどこまでinlineに出し、どこをdrill-downへ残すかをPressureする。
