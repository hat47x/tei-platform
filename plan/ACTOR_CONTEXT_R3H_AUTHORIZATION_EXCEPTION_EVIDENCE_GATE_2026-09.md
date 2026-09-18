# Actor Context R3H — Authorization Exception Evidence Gate

- Status: ISSUE-000053 / Phase 3 exception-evidence pressure / Reference-only
- Date: 2026-09-18
- Parent: `plan/ACTOR_CONTEXT_R3G_AUTHORIZATION_REQUIREMENT_COMPOSITION_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3F_REPOSITORY_BACKED_AUTHORIZATION_SCOPE_2026-09.md`
  - `reference/business-system-design/order-entry-business-procedure.json`
  - `reference/business-system-design/order-entry-follow-up-procedure.json`
  - `reference/business-system-design/order-entry-protected-target-overlay.json`
  - `reference/legacy-auth/authorization.json`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3Gでは、Pattern minimum invariant候補とBusiness Step requirementが食い違う場合に、

```text
required vs not-required
  -> explicit-exception-authority-required
```

として停止し、silent overrideを禁止した。

次に確認すべきなのは、例外・override mechanismを設計することではなく、**そのmechanismを必要とするrepository-backed Business caseが実際に存在するか**である。

R3Hでは、例外機構をproductionへ追加する前のEvidence Gateを定義する。

```text
semantic possibility
  != repository-backed need
  != production mechanism justified
```

を固定する。

## 2. current Reference監査

current `order_entry` Referenceでは、少なくとも次が確認できる。

### BP-00 / FU-00

```text
order-entry.order-handling/BP-00
  -> order_entry.recent_orders
  -> Authorization requirement = unresolved

order-entry.follow-up/FU-00
  -> order_entry.recent_orders
  -> Authorization requirement = unresolved
```

同じPatternをcross-Procedureで再利用しているが、どちらもexplicit `required` / `not-required` をauthorしていない。

### BP-06

```text
order-entry.order-handling/BP-06
  realization = security-boundary
  capability = security.authorize
  protected target overlay = order_entry.submit_order
  -> Authorization requirement = required
```

これはBusiness Step側のexplicit requirement Evidenceである。

しかし同じ `order_entry.submit_order` を別Business contextが利用し、そこだけexplicit `not-required` となるrepository-backed caseはcurrent corpusに存在しない。

### Legacy Authorization policy

current Legacy policyには:

```text
browse / order_entry.recent_orders / allow
execute / order_entry.submit_order / allow
```

がある。

ただしR3A〜R3Gの境界どおり:

```text
policy rule
  != Authorization requirement authority
  != Pattern minimum authority
  != exception authority
```

である。

## 3. current corpusにはexplicit not-required caseがない

R3Hで重要なのは、current Referenceに:

```text
explicit Authorization requirement = not-required
```

としてacceptedに扱えるBusiness / Pattern authorityが見当たらないことである。

したがってcurrent corpusだけから:

- `override`
- `exclude`
- `inherit`
- priority number
- nearest-scope-wins
- last-writer-wins

をproduction mechanismとして設計する根拠はない。

これは機能不足の証明ではない。

```text
no qualifying exception case yet
  -> exception mechanism Evidence Gate not satisfied
  != exceptions can never exist
```

である。

## 4. Exception Evidence Gate

production exception / precedence mechanismを検討対象へ上げるには、少なくとも次のEvidenceを分離して満たす必要がある。

### E1 — Pattern minimum authority

Pattern側の `required` が、単なる反復・policy rule・technical realizationではなく、明示的authorityとして存在する。

### E2 — contradictory Business variation

同じPatternを利用する特定Business contextで、明示的 `not-required` がBusiness authorityとして存在する。

### E3 — distinct provenance

Pattern minimumとBusiness variationが別source / authorityとして追跡できる。

```text
same source rewritten twice
  != two-authority conflict
```

### E4 — repository-backed Business need

synthetic pressure recordだけではなく、actual Reference / accepted Business requirementから例外の必要性が発生している。

### E5 — no policy inference

Legacy Authorization allow/deny ruleやruntime decisionから、E1 / E2を逆算しない。

## 5. Gate outcome

R3Hでは三状態を区別する。

```text
insufficient-evidence
synthetic-only-pressure
exception-mechanism-pressure-qualified
```

### insufficient-evidence

E1 / E2のいずれかが欠けるcurrent Repositoryはここに留まる。

### synthetic-only-pressure

test-onlyにrequired / not-required conflictを置くことはsemantic boundary確認には有効だが、production mechanismの採用根拠にはしない。

### exception-mechanism-pressure-qualified

将来、repository-backedにE1〜E5がそろった場合だけ、exception mechanismの設計Pressureを開始してよい。

ただし、この状態でも:

```text
exception mechanism needed
  != precedence winner known
  != override syntax chosen
```

である。

## 6. test-only source

- `tools/reference-business-procedure-check/actor_context_r3h_authorization_exception_evidence_gate_pressure_test.go`

Pressureすること:

- current BP-00 / FU-00はunresolvedのまま
- current BP-06はrequired
- current corpusにexplicit not-requiredを発明しない
- Legacy policyからPattern minimum / exceptionを生成しない
- synthetic conflictはproduction exception needへ昇格しない
- repository-backed explicit conflictだけがmechanism-design pressureを開ける
- Gate通過後もWinner / Precedence / Override syntaxを生成しない

## 7. current conclusion

current Repositoryについては:

```text
Pattern reuse exists
Business Step required case exists
Legacy policy exists

but

repository-backed Pattern minimum required
  + explicit Business Step not-required
  = not observed
```

したがって:

```text
current exception Evidence Gate
  = not satisfied

production override / precedence mechanism
  = do not add
```

とする。

R3Gの `explicit-exception-authority-required` は、現時点では**将来のqualifying caseに対する停止線**として保持し、機構設計へ進めない。

## 8. 次

exception mechanism lineはqualifying Evidenceが出るまで保留し、read-only AI Business Actorの本線へ戻る。

次段では:

```text
Authorization requirement = required
  != protected target/action authority
  != evaluation cadence
```

を再接続する。

特に:

- requirement scopeからtarget/action scopeを自動継承しない
- Pattern requirementからpolicy resource/actionを逆算しない
- Business Step requirementからevaluation cache scopeを決めない
- one successful evaluationを別Step / Procedureへ暗黙再利用しない

ことをPressureする。
