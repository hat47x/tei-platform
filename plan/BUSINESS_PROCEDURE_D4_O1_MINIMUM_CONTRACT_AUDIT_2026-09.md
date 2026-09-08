# Business Procedure D4 + O1 Minimum Contract Audit — 2026-09

Status: **Lane C / minimum semantic candidate refinement / no production Canonical adoption**

## 1. 問い

Current candidate stackではBusiness Procedure Step bodyをD3からD4へ縮小し、protected Application targetをO1 test-only overlayとして別pressureにしてきた。

本監査では次を判定する。

```text
Is D4 alone lossless for current accepted Business meaning?

If not,
what is the narrowest addition that preserves the missing meaning without
re-expanding the Step into D3 or introducing a generic ref-role model?
```

## 2. Current accepted Business meaning

Primary ReferenceのBP-06は:

```text
businessAction:
  determine whether the authenticated subject may execute the submit application action

realization:
  security-boundary

capabilities:
  security.authorize
```

をauthorしている。

したがってBusiness meaningとして少なくとも:

```text
there is a protected Application action
```

が存在する。

Cross-view pressureではそのexact targetを:

```text
order_entry.submit_order
```

として解決してきた。

同じBP-06はそのFunctionでrealizeされるわけではない。

```text
supporting Semantic authority
  security.authorize

protected Application target
  order_entry.submit_order

direct Application Function realization
  none
```

この三者分離はcurrent Referenceで意味を持つ。

## 3. D4 alone

D4 Step bodyは:

```text
businessAction
applicationPatterns[]  // direct realization
validations[]
capabilities[]
```

だけを持つ。

BP-06では:

```text
businessAction = authorization Business wording
capabilities   = security.authorize
applicationPatterns = []
```

を保持できる。

しかしprotected target exact identityは保持できない。

Target-only change:

```text
order_entry.submit_order
  -> order_entry.order_record
```

が起きてもD4 bodyは不変である。

したがって、protected target exact traceをaccepted Business meaningとして保持するなら:

> **D4 alone is lossy.**

これはD4全体をrejectする意味ではない。D4 bodyの縮小判断は維持し、missing role relationだけを別axisで補う。

## 4. D3へ戻っても解決しない

D3 / earlier thin StepにはD4より広い:

```text
realization
Data[]
Relations[]
```

等がある。

しかしD3の`applicationPatterns[]`もdirect Application Function realizationとして扱っている。

したがってBP-06へ:

```text
applicationPatterns:
  order_entry.submit_order
```

を入れると:

```text
protected target
  -> direct realization
```

へ意味を誤変換する。

つまりO1 pressureはD4縮小によって生じた欠損ではない。

```text
D3
  broader than D4
  AND
  still lacks target-vs-realization role separation
```

である。

このためO1解決のためにD3へ戻る理由はない。

## 5. Narrow minimum candidate

Current Evidenceからの最小比較候補を:

```text
D4 body
  businessAction
  applicationPatterns[]
  validations[]
  capabilities[]

+

optional narrow protected-target relation
  protectedApplicationPatternTargets[]
```

とする。

名称は最終production field名ではなくcomparison vocabularyである。

重要なのはshapeより意味境界:

```text
Application Pattern realization ref
  !=
protected Application Pattern target ref
```

である。

## 6. Why narrow field before generic role model

Current Evidenceが必要としているrole差は具体的に:

```text
realizes
vs
is protected target of authorization Business Action
```

である。

まだ次のtaxonomyは証明されていない。

```text
supports
concerns
checks
protects
observes
precedes
owns
...
```

したがって:

```text
GenericSemanticRef {
  Kind
  Role
  Target
}
```

を導入しない。

O1はApplication Patternという既存authority kindに対する一つのnarrow Business relationとしてPressureする。

## 7. Machine pressure

`d4_o1_minimum_losslessness_pressure_test.go`で次を追加した。

### M1 — D4 alone loses target-only meaning

同じD4 bodyに対し:

```text
protected target A = order_entry.submit_order
protected target B = order_entry.order_record
```

を比較するとD4 aloneは差を表現できない。

D4 + O1 narrow candidateは差を保持する。

### M2 — target does not become realization

BP-06で:

```text
ApplicationPatterns = []
Capabilities = [security.authorize]
ProtectedApplicationPatternTargets = [order_entry.submit_order]
```

を同時に保持する。

### M3 — no generic role taxonomy

Candidate Stepは:

```text
Body
ProtectedApplicationPatternTargets
```

だけで比較し、Role / Kind / Policy / Principal / Permission等を持たない。

### M4 — fail closed

Unknown Step / duplicate protected targetを黙ってnormalizeしない。

### M5 — no symmetry pressure

BP-06以外のStepへprotected target fieldを埋めることを要求しない。

Optional relationとして扱う。

## 8. Source placement comparison

### S0 — wording only

```text
BusinessAction text says "submit application action"
```

Pros:
- source最小

Cons:
- exact identity traceがない
- rename / impact / stale ref detectionができない
- wording解析へauthorityを移す危険

Current: **insufficient if exact cross-view trace remains accepted**。

### S1 — separate production sidecar

Current test-only overlayはmeaningの独立性をPressureする装置として有用。

しかし別sourceにするとProcedure-only source fingerprintはtarget-only changeを検出できない。

Production採用には:

```text
Procedure source + O1 source
  -> composite Shared Acceptance target
```

が必要になる。

これはone narrow relationのためにauthoring / acceptance modelを複雑化する。

Current: **useful pressure apparatus / weak production candidate**。

### S2 — same Business source, narrow optional field

Conceptual source:

```text
BP-06
  businessAction ...
  capability security.authorize
  protectedApplicationPattern order_entry.submit_order
```

Pros:
- protected target changeがProcedure source fingerprintへ自然に含まれる
- current Shared Acceptance stale ruleを再利用しやすい
- targetとBusiness Actionが同じreview unitにある
- sidecar locator / composite acceptanceが不要

Risks:
- current one strong caseだけでfieldをproduction固定する危険
- naming / cardinalityはまだ未確定

Current: **preferred minimum source candidate if O1 survives actual pressure**。

### S3 — generic role-qualified refs

Current: **defer / over-generalized**。

## 9. Candidate ranking update

Before:

```text
D3 broader candidate
D4 smaller candidate
O1 separate target overlay pressure
```

After this audit:

```text
D3
  not preferred as minimum
  broader fields do not solve O1 role loss

D4 alone
  minimal body
  but lossy for accepted protected-target exact trace

D4 + narrow O1
  current minimum semantic candidate

O1 sidecar
  pressure apparatus, not preferred production source shape

generic ref-role model
  deferred
```

D3はhistorical comparison Evidenceとして残すが、current minimum candidate競争の先頭から外す。

## 10. What is now proven vs not proven

### Proven as design pressure

```text
BP-06 has protected Application action meaning
protected target != direct realization
D4 alone cannot preserve exact target-only change
D3 breadth does not solve role distinction
narrow field can preserve current difference without generic role taxonomy
separate sidecar requires broader Acceptance machinery
```

### Not proven

```text
production field name
single vs multiple target cardinality
production Canonical BusinessProcedure kind
production World placement
second independent protected-target example
generic role taxonomy
actual runner green
```

## 11. Current decision

Minimum semantic candidate for current Reference becomes:

```text
D4 + narrow O1
```

rather than:

```text
D4 alone
```

This is a **candidate refinement**, not production adoption.

Production source change remains deferred until:

1. actual Repository tests are green;
2. Pre-ADR Gate is otherwise ready enough to justify source shape freezing;
3. narrow target field still proves smaller than keeping an external source-set Acceptance model;
4. exact target meaning remains necessary after another Business-design review / realistic pressure.

A second protected-target case is required before generalizing a role taxonomy, but not used to erase the already-observed BP-06 exact-trace loss.

## 12. Consequence for Pre-ADR

Pre-ADR comparison should now ask:

```text
Can D4 + narrow O1 + S4-A + H1/R3 preserve all current accepted Business meaning?
```

not:

```text
D3 vs D4 alone
```

D3 remains useful as subtraction history; D4 alone remains useful as proof of why O1 is a separate axis.

Neither remains the strongest standalone production candidate.
