# Actor Context R3W — Re-evaluation Reason / Category Value Change Boundary

- Status: ISSUE-000053 / Phase 3 category re-evaluation pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3V_CROSS_AUDIENCE_REASON_IDENTITY_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3Q_PROJECTION_SELECTIVE_FRESHNESS_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3T_PROJECTION_REEVALUATION_REASON_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3U_MULTI_REASON_AGGREGATION_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3T / R3U / R3Vではchangeからselected categoryのre-evaluation reasonを説明し、そのreason identityをbounded audience間でも共有できるようになった。

しかし:

```text
re-evaluation reason exists
  != category value changed
```

である。

R3Wではreason集合を使ってcategoryを再評価した後、R3Qのcategory fingerprintをbefore / afterで比較し、actual selected meaning changeを別段で確定する。

```text
observed change
  -> R3T / R3U reason set
  -> category recomputation
  -> R3Q category fingerprint comparison
  -> changed / unchanged after re-evaluation
```

をPressureする。

## 2. reasonあり / 値不変を正常系として許す

source inputが変わっても、selected categoryとして導出される意味が同じ場合がある。

その場合:

```text
reason = input changed
category status = unchanged-after-reevaluation
```

でよい。

```text
dependency changed
  != semantic category changed
```

を維持する。

## 3. multiple reasons / one category

R3Uで一つのcategoryへ複数reasonが成立する場合でも、category fingerprint比較は一回でよい。

```text
machine-reasons
  <- requirement source
  <- protected resource source
```

のように複数reasonを一つのcategory resultへ保持する。

```text
multiple reasons
  != multiple category copies
```

である。

## 4. changed-after-reevaluation

再構成後にR3Q fingerprintが変われば:

```text
status = changed-after-reevaluation
```

とする。

これはselected meaning changeを示すが:

```text
changed category
  != automatic rebuild
  != runtime Authorization reevaluation
  != next action
```

である。

## 5. reasonなしselected category changeはfail closed

before / afterでselected category fingerprintが変わったにもかかわらず、tracked reasonが一件もない場合:

```text
selected category changed
  + no tracked reason
  -> unexplained category change
```

としてfail closedする。

候補原因は:

- missing dependency edge
- incomplete change observation
- derivation logic change
- incomplete Source / Runtime provenance

である。

R3Sの `derivation-rule-freshness-unresolved` をここで露出させる。

## 6. omitted category changeはbounded resultへ昇格させない

Human orientationでomittedされた `known-authority-facts` 等が変わっても、selected categoryに変化がなければR3W resultへ入れない。

```text
omitted source detail changed
  != selected category changed
```

を維持する。

## 7. locator / runtimeもactual値で確認する

locator refresh reasonがあっても `source-refs` の再構成値が同じならunchangedである。

runtime Observation reasonがあっても、bounded projectionへ現れる `runtime-outcome` が同じならunchangedである。

reason classだけでvalue changeを決めない。

## 8. profile changeはsame-profile比較へ混ぜない

R3Tの `profile-changed` はselected / omitted partition自体の変更である。

R3Wでは:

```text
profile-changed
  -> bounded selectionを再構成
  -> その後category比較
```

とする。

旧profileのままcategory比較しない。

## 9. test-only source

- `tools/reference-business-procedure-check/actor_context_r3w_category_value_change_pressure_test.go`

Pressureするcase:

1. reasonあり + category値不変 -> unchanged-after-reevaluation
2. reasonあり + category値変化 -> changed-after-reevaluation
3. one category / multiple reasonsを一つのresultへ保持
4. locator reason + source-refs不変/変化を分離
5. runtime reason + runtime-outcome変化を検知
6. selected category値変化 + reasonなし -> fail closed
7. Human omitted category変化 + reasonなし -> selected resultなし
8. profile-change reason -> same-profile evaluator拒否
9. Stale / RefreshAll / AutoRebuild / RuntimeReevaluationRequired等を追加しない

## 10. current conclusion

R3Wで固定するのは:

```text
reason set
  -> what to re-evaluate

R3Q fingerprint comparison
  -> whether selected meaning actually changed
```

という分離である。

```text
reason exists
  != value changed

value changed
  != automatic action
```

を維持する。

production rebuild engine / invalidation scheduler / runtime reevaluation policyは追加しない。

## 11. 次

次段では `changed-after-reevaluation` のうち、Authorization runtime decisionのcontext identityへ実際に関係するcategoryを分ける。

特にrequirement / action / resource / principal / runtime Observation tupleと、presentation / locator-only changeを分離し、R3Iで保留したevaluation invalidation / reuse authorityへ接続する。
