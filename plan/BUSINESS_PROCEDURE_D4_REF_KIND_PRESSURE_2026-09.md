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

そこでD3より一段小さい**Step body比較候補**D4を置く。

```text
D4 = D3 body
     - unproven direct Data ref slot
     - unproven direct Relation ref slot
     + fail-closed reopen rule
```

D4はproduction採用案ではなく、current Evidenceより強いsurfaceを先取りしていないか確認するtest-only Pressureである。

Step addressabilityはD4自身へ入れず、S4-Aとして独立に合成する。

## 2. current Evidenceで実証済みのref kind

### Application Pattern

```text
BP-00 -> order_entry.recent_orders
BP-01 -> order_entry.order_record
BP-02 -> order_entry.customer_reference
BP-03 -> order_entry.order_lines
BP-07 -> order_entry.submit_order
```

Pattern-owned Data / Relation / Query / Capability / Datasource / owner Applicationはnormalized Worldからderiveする。

### Validation

```text
BP-05
  -> sales.order_line.quantity_positive
  -> sales.order.lines.line_number_unique
```

BP-03 line reviewへ同じValidation authorityを重複しない。

### Capability

```text
BP-06 -> security.authorize
```

BP-07は`order_entry.submit_order` Patternから`sales.order.submit`をderiveできるためCapabilityを重複保持しない。

## 3. D4 test-only body shape

```text
Procedure body
  id
  purpose
  trigger
  actor
  ordered step bodies[]
    businessAction
    applicationPatterns[]
    validations[]
    capabilities[]
```

D4 bodyはStep local keyを所有しない。

```text
D4 body meaning
  !=
S4-A Step address metadata
```

current testではsource-local Step keyを使ってD4 bodyを検査対象へ対応付けるが、これはD4 bodyのfieldではない。

## 4. authority kindを失わない

D4ではrefをgeneric `[]CanonicalID`やkind-less `semanticRefs[]`へ畳まない。

```text
Application Pattern ref
Validation ref
Capability ref
```

は異なるauthority kindである。literal ID文字列が偶然同じでも同一refとして扱わない。

RQ-026のcross-view correctionでは、どのauthorityへ戻るべきかを失わないことが重要である。

## 5. Data / Relation direct refsの扱い

current Referenceからはminimum必要性が未証明である。

したがってD4 bodyはcurrent ReferenceではData / Relation slotを持たない。

ただし将来sourceにdirect Data / Relation refが現れた場合、D4はそれを黙って捨てない。

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
D3 body
  businessAction
  ApplicationPatterns
  Data
  Relations
  Validations
  Capabilities

D4 body
  businessAction
  ApplicationPatterns
  Validations
  Capabilities

future direct Data / Relation appears
  -> fail closed / reopen
```

D4がcurrent Referenceに対して意味保存できるなら、production ADRでD3のData / Relation slotを先行採用する根拠は弱くなる。

別のrealistic Referenceでdirect Data / Relationが必要ならD4は棄却または拡張する。

## 8. `realization`との関係

D4でも`realization`をproduction meaningへ戻さない。

ref kindはauthoring labelから推測しない。

- `validation-support`からValidationをguessしない
- `security-boundary`からCapabilityをguessしない
- `application-use`からPatternをguessしない

explicit refsはsourceに明示され、normalized authorityへexact resolutionされる。

## 9. S4-A / H1 / R3との合成

D4はStep bodyのminimum比較だけを担当する。

別pressureで次を合成する。

```text
D4 body
  + S4-A Procedure-scoped Step local address
  + H1/R3 handoff-only normalized relation
```

`d4_normalized_composition_pressure_test.go`でsource-level compositionを実装済み。

合成後も:

- D4 bodyへStep local addressを混ぜない
- stale handoff endpointをfail closed
- address + endpoint協調renameを許容
- Handoff condition changeをbodyから独立したBusiness meaning差として保持
- future direct Data / Relation refをaddress/handoff layerで隠さずD4 derivationでfail closed
- normalize後のmodelへduplicate Handoff relationが混入してもnormalized validator自身がfail closed

をPressureする。

最後のduplicate検査はsource-side `deriveCanonicalBusinessHandoffProbes()`の事前検査だけに依存しない。normalized modelだけでH1/R3 integrityを閉じるという契約を維持するためである。

D4からgeneric Business Flow / Edge / Runtime Workflowを導かない。

## 10. Repository Pressure

`tools/reference-business-procedure-check/d4_ref_kind_pressure_test.go`でD4 body単体を、`d4_normalized_composition_pressure_test.go`でS4-A/H1/R3との合成をPressureする。

D4 body単体:

1. D4 bodyがBusiness Action / Pattern / Validation / Capabilityだけを持つ
2. D4 bodyはData / Relation / Realization / Role / Kindを持たない
3. current ReferenceのBP-03/BP-05 authority分離を保持
4. BP-06 `security.authorize`を保持
5. BP-07へPattern-owned Capabilityをコピーしない
6. BP-04/BP-08/BP-09へfake refを付けない
7. direct Data refが現れたらfail closed
8. direct Relation refが現れたらfail closed
9. literal identity textが同じでもauthority kindを区別

Composition:

10. D4 bodyとS4-A address layerを分離
11. stale / missing Handoff endpointをfail closed
12. address + Handoff endpoint協調renameを許容
13. condition changeをHandoff meaning差として保持
14. duplicate Handoff relationをnormalized layerでfail closed
15. new ref kindをcomposition layerで隠さない

actual Repository runnerでgreenになるまではexecution Evidenceとは数えない。

## 11. pre-ADR判断

```text
D3
  broader Step body comparison candidate

D4
  smaller evidence-backed Step body candidate
  current Referenceで比較する価値あり
  not adopted

S4-A
  addressability concern
  D4とは独立
```

production contractでref kindをどこまで持つかは、current `order_entry`だけでなく次のrealistic Referenceから決める。

current Evidenceだけを理由にgeneric `semanticRefs[]`やdirect Data / Relation slotを先取りしない。

## 12. 次

1. D4 / D4+S4-A+H1/R3 Pressureをactual testsで確認
2. second realistic Referenceが必要な理由をApplication数ではなくdirect authority / Business meaning不足から判断
3. direct Data / Relation requirementが現れた場合はD4をfail closedさせ設計をreopen
4. normalized duplicate Handoff invariantをactual testsで確認
5. actual greenまでproduction Canonical ADRを作らない
