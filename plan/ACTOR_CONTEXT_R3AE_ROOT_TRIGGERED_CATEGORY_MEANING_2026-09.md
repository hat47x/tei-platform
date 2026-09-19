# Actor Context R3AE — Root-triggered Category Re-evaluation / Meaning Change Boundary

- Status: ISSUE-000053 / Phase 3 root-triggered category meaning pressure / Reference-only
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
provenance root delta
  -> bounded category re-evaluation reason
```

まで接続した。

R3AEではそのreasonをR3Wのsame-profile category fingerprint比較へ渡し:

```text
root changed
  -> category re-evaluated
  -> category meaning unchanged / changed
```

をPressureする。

重要なのは:

```text
re-evaluation reason exists
  != category meaning changed
```

である。

## 2. 三段階を混ぜない

R3AEでは少なくとも次を別段として保持する。

```text
1. root delta observed
2. bounded category selected for re-evaluation
3. category meaning changed / unchanged after recomputation
```

したがって:

```text
root content changed
  != projection meaning changed
```

である。

## 3. freshness stopの正常系

current R3AA freshness stopでは:

```text
machine reason
  = authorization-evaluation-freshness-authority-unresolved
```

である。

Business Procedure Referenceやprior Observationがrevisionされても、再計算の結果:

```text
reevaluation authority = unresolved
reuse authority = unresolved
```

のままなら:

```text
machine-reasons
  = unchanged-after-reevaluation
```

でよい。

change reasonを見ただけでstop meaningを変えない。

## 4. runtime Observation revision

R3ADではprior Observation revisionを:

```text
machine-reasons re-evaluation
```

へ接続する。

ただしfreshness authority unresolved中は:

```text
prior Observation allowed -> denied
  != current runtime-outcome = denied
```

である。

再評価後もfreshness stopが同じならmachine-reasonsはunchangedである。

runtime Evidence revisionをcurrent Authorization outcomeへ昇格させない。

## 5. one root / mixed category result

R3X test-only context root変更は:

- stop-identity
- machine-reasons
- resolution-needs

を再評価対象にできる。

再構成後:

```text
stop-identity = unchanged
machine-reasons = changed
resolution-needs = unchanged
```

のような混在を許す。

```text
one root changed
  != all bound categories changed
```

である。

## 6. locator change

locator-only deltaがsource-refsへbindingされても:

```text
locator reason exists
  != source-refs value changed
```

である。

再構成後のSourceRefs fingerprintが同じならunchanged、
current locatorがprojectionへ反映されて値が変わればchangedとする。

## 7. Unselected / Residualをcategory change authorityにしない

R3ADの:

```text
UnselectedChange
ResidualRootChange
```

はR3W selected-category reasonへ昇格させない。

特にcorroboration changeがResidualRootChangeである場合:

```text
corroboration changed
  + selected machine-reasons changed
  + no selected reason
  -> unexplained category change
  -> fail closed
```

とする。

Residual root deltaは消さずにR3AE resultへ保持するが、category changeを正当化しない。

## 8. Human / Agent

同じbefore / after coreと同じshared reasonなら、shared categoryの:

- before fingerprint
- after fingerprint
- changed / unchanged status

はHuman / Agentで一致する。

Agentだけselectedするknown-authority-factsは追加resultを持てる。

```text
different bounded result set
  != different shared category truth
```

である。

## 9. R3U / R3W再利用

R3ADのselected impactはR3Uでexact reason identityを正規化し、その後R3Wへ渡す。

```text
R3AD
  -> R3U lossless reason aggregation
  -> R3W same-profile category fingerprint comparison
```

とする。

R3AE自身は新しいfingerprint algorithmやimpact graphを持たない。

## 10. test-only source

- `tools/reference-business-procedure-check/actor_context_r3ae_root_triggered_category_meaning_pressure_test.go`

Pressureするcase:

1. requirement Reference change + same freshness core -> machine-reasons unchanged-after-reevaluation
2. Human / Agent shared machine-reasons resultが一致し、Agent-only known-authority-facts resultを許す
3. prior Observation revision + same freshness core -> machine-reasons unchanged / runtime-outcome resultなし
4. test-only context root change -> one rootからchanged / unchanged categoryが混在できる
5. locator reason + actual SourceRefs不変/変化を分離する
6. corroboration ResidualRootChangeだけではselected category changeを説明できずfail closed
7. ResidualRootChangesをR3AE resultへ保持する
8. reason / changed statusからShouldReevaluate / Retry / CacheValid等を生成しない

## 11. current conclusion

R3AEで固定するのは:

```text
root delta
  -> re-evaluation reason
  -> actual category fingerprint comparison

reason exists
  != value changed

one root changed
  != all bound categories changed

residual root change
  != selected category change authority

category changed
  != response action selected
```

である。

production rebuild engine、runtime reevaluation scheduler、cache invalidation、event routingは追加しない。

## 12. 次

次段ではR3AEのcategory resultをfreshness stop explanationへ戻し:

```text
root changed
category re-evaluated
meaning unchanged / changed
freshness authority still unresolved
```

をHuman / Agentへどこまで開示するかをPressureする。

特に `unchanged-after-reevaluation` を「安全」「fresh」「reuse可能」へ読み替えない。
