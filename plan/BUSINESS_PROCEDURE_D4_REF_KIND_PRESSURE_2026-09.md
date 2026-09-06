# Business Procedure D4 Evidence-Backed Ref Kind Pressure — 2026-09

Status: **Lane C / test-only architecture pressure / D4 candidate comparison / no production Core adoption**

## 1. 問い

D3 candidateはStep bodyにtyped explicit refsを残している。

```text
ApplicationPatterns
Data
Relations
Validations
Capabilities
```

しかしcurrent `order_entry` Referenceで、Business Action自身がPatternから独立してdirect Data / Relation identityを保持しなければ意味を失う例はまだ存在しない。

一方、Application Pattern / Validation / Capabilityについてはcurrent Referenceで必要性が実証されている。

そこでD3より一段小さい比較候補D4を置く。

```text
D4 = D3
     - unproven direct Data ref slot
     - unproven direct Relation ref slot
     + fail-closed reopen rule
```

D4はproduction採用案ではなく、current Evidenceより強いsurfaceを先取りしていないか確認するtest-only Pressureである。

## 2. current Evidenceで実証済みのref kind

### Application Pattern

Business ActionがApplication interactionを利用する場合に直接保持する。

```text
BP-00 -> order_entry.recent_orders
BP-01 -> order_entry.order_record
BP-02 -> order_entry.customer_reference
BP-03 -> order_entry.order_lines
BP-07 -> order_entry.submit_order
```

Pattern-owned Data / Relation / Query / Capability / Datasource / owner Applicationはnormalized Worldからderiveする。

### Validation

BP-05はdeterministic constraint evaluationそのものを表すため、Validation authorityを直接保持する。

```text
BP-05
  -> sales.order_line.quantity_positive
  -> sales.order.lines.line_number_unique
```

### Capability

BP-06はAuthorization判断そのものを表し、既存Semantic authorityがあるためdirect Capability refを保持する。

```text
BP-06 -> security.authorize
```

BP-07は`order_entry.submit_order` Patternから`sales.order.submit`をderiveできるためCapabilityを重複保持しない。

## 3. D4 test-only shape

```text
BusinessProcedure
  id
  purpose
  trigger
  actor
  ordered steps
    localKey
    body
      businessAction
      applicationPatterns[]
      validations[]
      capabilities[]
```

このshapeは正式production typeではない。

## 4. authority kindを失わない

D4ではrefをgeneric `[]CanonicalID`や`[]string`一袋へ畳まない。

```text
Application Pattern ref
Validation ref
Capability ref
```

は異なるauthority kindである。

literal ID文字列が偶然同じでも同一refとして扱わない。

これはRQ-026のcross-view correctionを成立させるために必要である。

## 5. Data / Relation direct refsの扱い

current Referenceからはminimum必要性が未証明である。

したがってD4 projectionはcurrent ReferenceではData / Relation slotを持たない。

ただし将来sourceにdirect Data / Relation refが現れた場合、D4はそれを黙って捨ててはいけない。

```text
future source contains direct Data / Relation ref
  -> D4 derive fails closed
  -> explicit-ref designをreopen
  -> realistic Referenceで必要性を再評価
```

これはData / Relation direct refsの永久禁止ではない。

## 6. なぜfail closedにするか

D4が単にData / Relationをdropすると、future Referenceが新しいaccepted meaningを要求した時にsemantic lossを見逃す。

P11とRQ-003/RQ-017に従い、未対応の意味を「不要だった」と扱わない。

D4はcurrent minimumを狭くする一方、future extensionを黙殺しない。

## 7. D3との比較

```text
D3
  body
    businessAction
    ApplicationPatterns
    Data
    Relations
    Validations
    Capabilities

D4
  body
    businessAction
    ApplicationPatterns
    Validations
    Capabilities

  direct Data / Relation appears
    -> fail closed / reopen
```

D4がcurrent Referenceに対して意味保存できるなら、production ADRでD3のData / Relation slotを先行採用する根拠は弱くなる。

一方、別のrealistic Referenceでdirect Data / Relationが必要ならD4は棄却または拡張する。

## 8. `realization`との関係

D4でも`realization`をproduction meaningへ戻さない。

ref kindはauthoring labelから推測しない。

- `validation-support`という文字列からValidationをguessしない
- `security-boundary`という文字列からCapabilityをguessしない
- `application-use`という文字列からPatternをguessしない

explicit refsはsourceに明示され、normalized authorityへexact resolutionする。

## 9. H1 / R3 / S4-Aとの関係

D4はProcedure Step bodyの比較候補であり、Handoff pressureを変更しない。

```text
D4 body
  + S4-A Procedure-scoped Step local address
  + H1/R3 handoff-only normalized relation
```

を組み合わせられるかは別Pressureで確認する。

D4からgeneric Business Flow / Edge / Runtime Workflowを導かない。

## 10. Repository Pressure

`tools/reference-business-procedure-check/d4_ref_kind_pressure_test.go`で次を固定する。

1. D4 bodyがBusiness Action / Pattern / Validation / Capabilityだけを持つ
2. current ReferenceのBP-03/BP-05 authority分離を保持
3. BP-06 `security.authorize`を保持
4. BP-07へPattern-owned Capabilityをコピーしない
5. BP-04/BP-08/BP-09へfake refを付けない
6. direct Data refが現れたらfail closed
7. direct Relation refが現れたらfail closed
8. literal identity textが同じでもauthority kindを区別

actual Repository runnerでgreenになるまではexecution Evidenceとは数えない。

## 11. pre-ADR判断

現時点では:

```text
D3
  strongest broader comparison candidate

D4
  smaller evidence-backed ref-kind candidate
  current Referenceで比較する価値あり
  not adopted
```

production contractでref kindをどこまで持つかは、current `order_entry`だけでなく複数realistic Referenceから決める。

current Evidenceだけを理由にgeneric `semanticRefs[]`やdirect Data / Relation slotを先取りしない。

## 12. 次

1. D4 Pressureをactual testsで確認
2. D4 + S4-A + H1/R3の組合せでcurrent Reference meaningが失われないか確認
3. second realistic Referenceが必要な理由を「Application数」ではなくdirect authority / Business meaning不足から判断
4. direct Data / Relation requirementが現れた場合はD4をfail closedさせ、設計をreopen
5. actual greenまでproduction Canonical ADRを作らない
