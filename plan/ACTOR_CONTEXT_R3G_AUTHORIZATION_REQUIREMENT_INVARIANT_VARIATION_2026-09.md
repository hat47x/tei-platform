# Actor Context R3G — Authorization Requirement Invariant / Variation Precedence Pressure

- Status: ISSUE-000053 / Phase 3 invariant-variation pressure / Reference-only
- Date: 2026-09-18
- Parent: `plan/ACTOR_CONTEXT_R3E_AUTHORIZATION_REQUIREMENT_SECOND_USE_CASE_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3Eでは、同じApplication Patternが複数Business contextで再利用されても、

```text
same Pattern reused
  != same Authorization requirement already accepted

multiple independent explicit requirements
  -> duplication pressure
  != Pattern authority proven
```

であることを確認した。

次に分離すべきなのは、Pattern側へminimum invariantを置く候補と、Business Step側のvariationをどう共存させるかである。

R3Gではproduction relation / precedence ruleを決めず、少なくとも次の四つを別authorityとしてPressureする。

```text
Pattern minimum invariant
Business Step additional requirement
explicit exception / override intent
precedence decision
```

特に、

```text
more specific
  != automatically higher precedence

exception authored
  != exception already effective
```

を固定する。

## 2. Pattern minimum invariant候補

Pattern側にAuthorization requirementを置くなら、それは単なる共通初期値ではなく、Patternの意味そのものに属するminimum invariantとして扱う候補になる。

test-only candidate:

```text
pattern order_entry.customer_reference
minimum Authorization requirement = required
supporting capability = security.authorize
```

この候補がaccepted authorityになった場合、そのPatternを利用するBusiness contextは、Business Step側に同じ記述がなくてもminimum applicabilityを受ける。

ただしR3Gは、このcandidateがcurrent Referenceですでに成立したとは扱わない。

Pattern minimum invariantの成立には少なくとも、

- 複数Business contextで同じstatusが偶然反復されること以上のEvidence
- requirementがPatternの意味そのものに属するというauthority
- variation / exceptionを許すかどうか
- exceptionを許す場合のexplicit semantics
- provenanceを失わないこと

が必要である。

```text
repeated Business requirements
  != Pattern minimum invariant authority
```

を維持する。

## 3. Business Step additional requirement

Pattern minimum invariant候補がある場合でも、Business Stepは別のBusiness concernを追加し得る。

R3Gでは、Business Step側の記述をPattern authorityのコピーとして扱わない。

```text
Pattern minimum
  + Business Step additional requirement
  -> two authorities with distinct provenance
```

同じ `required` が両方に現れる場合でも、

- Pattern由来
- Business context由来

を一つのsourceへ潰さない。

また、Business Step側が未記述であることを、Pattern minimumの否定として扱わない。

## 4. explicit exception / override intent

Pattern minimumが `required` で、特定Business Stepが `not-required` を要求する場合、そのStep記述を単純な「より具体的なので勝つ」ruleへしない。

少なくとも:

```text
Pattern minimum = required
Business Step = not-required
```

はauthority conflictである。

さらにStep側が:

```text
exception intent = explicit
```

を持っていても、それだけではprecedenceは成立しない。

```text
explicit exception intent
  != accepted exception precedence
```

である。

例外を有効にするには、別authorityとして少なくとも:

- exceptionを許すこと
- exception scope
- precedence
- provenance
- rationale / Evidence

が必要になる。

R3Gではそのproduction表現を追加しない。

## 5. precedenceをapplicabilityから分離する

Authorization requirementのapplicability sourceと、複数sourceが衝突したときのprecedenceは別問題である。

```text
requirement authored
  != precedence resolved

scope more specific
  != precedence winner

later source
  != precedence winner

closer file
  != precedence winner
```

R3G test-only evaluatorでは、precedenceを明示的な入力として扱い、未解決ならfail closedする。

候補state:

```text
precedence = unresolved
precedence = pattern-minimum
precedence = step-exception
```

ただし後二つはtest-only比較用であり、current production ruleではない。

## 6. Pressure matrix

### Case A — Pattern minimum candidateのみ

```text
Pattern minimum = required
Business Step = unresolved / absent
```

Pattern minimum authorityがacceptedという仮定下では、effective applicabilityはrequiredになり得る。

ただしStep側のabsenceを独立した `not-required` authorityへ変換しない。

### Case B — Pattern minimum + Step additional required

```text
Pattern minimum = required
Business Step = required
```

effective statusは衝突しないが、二つのprovenanceを保持する。

同値だからといってBusiness Step sourceを削除したり、Pattern authority成立のEvidenceへ循環利用しない。

### Case C — Pattern minimum + Step not-required

```text
Pattern minimum = required
Business Step = not-required
precedence = unresolved
```

結果はconflict unresolvedであり、どちらも自動採用しない。

### Case D — explicit exception intent

```text
Pattern minimum = required
Business Step = not-required
exception intent = explicit
precedence = unresolved
```

exception intentが明示されてもconflictは残る。

test-onlyにprecedence decisionを与えた場合だけ、そのdecisionに従う結果を比較できる。

この比較はprecedence候補の意味差をPressureするためのものであり、そのruleを採用したことを意味しない。

## 7. variationがPattern invariant候補へ返すEvidence

同じPatternについてBusiness context間で:

```text
required
not-required
```

がともにexplicit authorityとして現れた場合、少なくとも無条件のPattern-wide invariant候補には反証圧力がかかる。

そのとき候補は:

1. Pattern minimum invariantを棄却する
2. explicit exception semantics付きのminimum invariantへ進む
3. Business Step scopeを維持する
4. 別の明示scopeを導入する

などへ分岐し得る。

R3Gはwinnerを選ばない。

重要なのはvariationを隠してPattern共通化を先行させないことである。

## 8. test-only source

- `tools/reference-business-procedure-check/actor_context_r3f_authorization_invariant_variation_pressure_test.go`

Pressureする不変条件:

- repeated statusだけでPattern invariantを成立させない
- Pattern minimumとBusiness Step authorityのprovenanceを統合しない
- Business Stepのabsenceをnot-requiredへしない
- scope specificityからprecedenceを生成しない
- explicit exception intentだけでoverrideを成立させない
- precedence unresolvedではconflictをfail closedする
- test-only precedence candidateをproduction ruleとして扱わない

## 9. current conclusion

R3Gで確定するのは次までである。

```text
Pattern minimum invariant
  != duplicated Business requirements

Business Step additional requirement
  != Pattern authority copy

explicit exception intent
  != accepted override

more specific scope
  != automatic precedence

applicability authority
  != precedence authority
```

したがって、R3Gでもproduction Business Procedure schema、Application Pattern grammar、override relation、precedence ruleは追加しない。

## 10. 次

次段では、Authorization requirementが `required` と確定した後の

```text
requirement applicability
  -> protected target/action authority
  -> evaluation cadence
```

を再接続する。

特に、

- Pattern minimumが成立した場合にtarget/actionまでPatternから継承してよいか
- Business Step variationがtarget/action variationも意味するか
- 一度のAuthorization evaluationをどのscopeで再利用できるか

を分離してPressureする。

requirement scopeをtarget authorityやevaluation cache scopeへ自動伝播させない。
