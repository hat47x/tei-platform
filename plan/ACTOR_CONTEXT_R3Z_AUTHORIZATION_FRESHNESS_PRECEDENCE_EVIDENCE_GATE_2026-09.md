# Actor Context R3Z — Authorization Freshness Precedence Evidence Gate

- Status: ISSUE-000053 / Phase 3 freshness-precedence Evidence Gate / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3Y_AUTHORIZATION_REEVALUATION_REUSE_AUTHORITY_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3G_AUTHORIZATION_REQUIREMENT_INVARIANT_VARIATION_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3H_AUTHORIZATION_EXCEPTION_EVIDENCE_GATE_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3Yでは:

```text
reevaluation authority
  != prior Observation reuse authority
```

を分離した。

またtest-onlyに:

```text
explicit-reevaluation-required
explicit-reuse-permitted
```

が同時に存在しても、precedenceを決めなかった。

R3Zでは一段前へ戻り、そもそも**production precedence mechanismを必要とするqualifying conflict Evidenceが存在するか**をGateにする。

## 2. generic reuse permissionはreevaluation directiveと直ちに矛盾しない

重要なのは:

```text
prior-observation-reuse-permitted
  != prior-observation-reuse-sufficient-without-reevaluation
```

である。

R3Yのgeneric reuse permissionだけでは:

- reevaluation後にprior Observationを補助Evidenceとして使ってよい
- named conditions下でreuseしてよい
- current decisionをprior Observationだけで満たしてよい

のどれかは決まっていない。

したがって:

```text
reevaluation-required
  + generic reuse-permitted
  != proven contradiction
```

である。

この状態だけでpriority / override / nearest-scope-winsをproduction化しない。

## 3. qualifying conflictに必要なEvidence

R3Zではproduction precedence mechanismの比較を開始するには、少なくとも次を要求する。

### E1 — same current Authorization context

Business address / subject / action / resourceが同じtupleである。

tuple mismatchならR3Xどおりprior Observation reuse comparison自体がcurrent-context問題ではない。

### E2 — explicit reevaluation directive

明示的に:

```text
prior Observationへ依存する前にcurrent evaluationを再実施する
```

という意味を持つauthorityが必要。

単なるstatic source changeやprovider revisionは代用しない。

### E3 — explicit bypass-capable reuse directive

単なる `reuse-permitted` ではなく:

```text
prior Observation alone may satisfy current Authorization evaluation
without a new evaluation
```

に相当する明示意味が必要。

これがなければE2とのsemantic contradictionは成立しない。

### E4 — explicit provenance / provenance relation

両directiveにsource provenanceが必要。

inferred hint / wording / policy similarityは認めない。

さらに:

```text
same source dual directive
  != cross-authority precedence conflict
```

を維持する。

同一sourceがreevaluationとbypass-capable reuseを同時に表現する場合は、source内compatibility semanticsが未解決である可能性があるため `same-source-compatibility-unresolved` とする。

production cross-authority precedence pressureにはdistinct authority provenanceを要求する。

### E5 — same applicability scope

両directiveが同じBusiness / Authorization contextへ適用されることが必要。

scopeが異なるだけならprecedence conflictとは限らない。

### E6 — repository-backed Business need

synthetic test-only contradictionだけではproduction mechanismを開かない。

current / future accepted Reference上の実caseが必要。

## 4. Gate outcome

R3Zでは三状態を区別する。

```text
insufficient-evidence
same-source-compatibility-unresolved
synthetic-conflict-pressure
precedence-mechanism-pressure-qualified
```

### insufficient-evidence

- generic reuse permissionしかない
- explicit provenanceが不足
- scope整合が不明
- same tupleでない

等。

### same-source-compatibility-unresolved

same tuple / same scopeで両directiveが明示されても、同一source由来ならcross-authority precedence conflictとは数えない。

### synthetic-conflict-pressure

E1〜E5を満たし、distinct authority provenanceを持つ明示的なcontradictionをtest-onlyに作れるが、repository-backed Business needがない。

### precedence-mechanism-pressure-qualified

E1〜E6を満たす実caseがある場合のみ。

ただし:

```text
precedence-mechanism-pressure-qualified
  != winner known
  != precedence rule selected
  != syntax selected
```

である。

## 5. current Reference result

current R3Yでaccepted production authorityとして確立したのは:

```text
reevaluation directive = none
reuse directive = none
```

である。

R3Y test-only sourceのexplicit directivesは比較用synthetic authorityであり、repository-backed production authorityではない。

さらにgeneric `prior-observation-reuse-permitted` はE3の「new evaluationを不要にできる明示意味」まで持たない。

したがってcurrent Gateは:

```text
freshness precedence Evidence Gate
  = insufficient-evidence
```

である。

## 6. provider policyをconflict authorityへ昇格させない

Legacy/provider Authorization policyが:

- changed
- same action/resourceを持つ
- prior decisionと異なる結果を返し得る

としても:

```text
provider policy revision
  != explicit reevaluation directive
  != explicit reuse-without-reevaluation directive
```

である。

provider-side cache contract等が将来accepted authorityとして明示される場合は別Evidenceとして評価する。

## 7. production mechanismを追加しない

current Gateが未充足なので、次を追加しない。

- priority number
- override / inherit
- nearest-scope-wins
- provider-wins
- requirement-authority-wins
- target-authority-wins
- last-writer-wins
- TTL precedence
- automatic reauthorization

R3G / R3Hと同じく:

```text
conflict pressure absent
  -> mechanism not justified
```

を採る。

## 8. test-only source

- `tools/reference-business-procedure-check/actor_context_r3z_freshness_precedence_evidence_gate_pressure_test.go`

Pressureするcase:

1. reevaluation-required + generic reuse-permitted -> contradiction未成立
2. explicit reuse-without-reevaluation + reevaluation-required + same source -> same-source-compatibility-unresolved
3. distinct-source explicit conflict + synthetic -> synthetic-conflict-pressure
4. same conflict + repository-backed + same scope + distinct provenance -> qualified
5. tuple mismatch -> insufficient
6. scope mismatch -> insufficient
7. provider policy revisionだけではdirectiveを補完しない
8. Gate resultにWinner / Priority / EffectiveDirectiveを持たせない

## 9. current conclusion

R3Zで固定するのは:

```text
two authority axes present
  != precedence conflict proven

generic reuse permission
  != reevaluation bypass authority

synthetic contradiction
  != production precedence mechanism justified

qualified conflict
  != winner selected
```

である。

## 10. 次

current Gateが未充足ならfreshness precedence branchは停止できる。

次はread-only AI Business Actor本線へ戻り、R3M〜R3Tで整備したOperational Projection / explanationを使って:

```text
unresolved freshness authority
  -> explainable stop / handoff
  != hidden cache decision
```

をPressureする。

production scheduler / cache / notification transportはその後のEvidenceに委ねる。
