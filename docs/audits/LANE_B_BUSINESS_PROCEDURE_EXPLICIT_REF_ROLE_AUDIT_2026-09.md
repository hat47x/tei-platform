# Lane B — Business Procedure Explicit Reference Role Audit — 2026-09

Status: **Lane B / supporting architecture-pressure audit / no production schema adoption**

## 1. 目的

Lane C PR #482のD3 explicit semantic refsを、次の原則で再監査する。

```text
explicit refs are evidence-driven, not symmetry-driven
Business Action != Application Function
reference target != realization
```

既存identityが存在するかだけでなく、Business Actionがそのidentityを**どの意味で参照するのか**を現在のReference表現が失っていないかを確認する。

本書はPAS-07 Trace正本ではない。新しいProduct Requirement、E7、Gate、production kindを追加しない。

## 2. current fixture / checker

Primary Procedure fixtureのStepは概ね次を持つ。

```text
applicationPatterns[]
data[]
relations[]
validations[]
capabilities[]
```

checkerは`applicationPatterns`について:

```text
NoApplicationFunction = len(applicationPatterns) == 0
```

としている。

したがってcurrent experimentでは:

```text
applicationPatterns ref
  = direct Application Function realizationとして扱われる
```

という意味が事実上固定されている。

一方、`validations` / `capabilities`等はexplicit Business semantic refとして解決されるが、refごとのroleは表現しない。

## 3. B-F23 — BP-06 protected-action trace gap

BP-06は現在:

```text
businessAction:
  determine whether the authenticated subject may execute the submit application action

capabilities:
  security.authorize
```

である。

`security.authorize`は既存Semantic authorityであり、次のinputを持つ。

```text
subject
action
resource
```

したがって「認可という意味」へのcross-view traceは成立している。

しかしcurrent BP-06には:

```text
which Application action?
which semantic action/resource?
```

を機械的に特定するexplicit refがない。

`submit application action`というBusiness wordingから人間はBP-07 / `order_entry.submit_order`を推測できるが、exact identity traceとしては未表現である。

### これはfake Function不足ではない

単純にBP-06へ:

```text
applicationPatterns:
  - order_entry.submit_order
```

を追加すると、current checkerはBP-06を:

```text
NoApplicationFunction = false
```

として扱う。

しかしBP-06のBusiness Actionは`order_entry.submit_order`によってrealizeされるわけではない。

```text
BP-06 Authorization concern
  references / protects submit action
  !=
BP-06 is realized by submit-order Application Pattern
```

である。

したがってPattern ref追加だけで解決すると:

```text
reference target
  -> realization
```

へ意味を誤昇格させ、RQ-025の境界を弱める。

## 4. Capabilityを二つ並べるだけでもroleは失われる

別案としてBP-06へ:

```text
capabilities:
  - security.authorize
  - sales.order.submit
```

を置けば両identityはexact解決できる。

しかしcurrent representationでは:

```text
security.authorize = authorization mechanism / semantic authority
sales.order.submit = protected action target
```

という**ref roleの違い**をmachine-readableに表現できない。

単にCapabilityが二件並ぶだけでは、どちらが何を意味するかをBusiness wordingへ依存する。

したがって問題はidentity existenceではなく:

```text
cross-view reference role
```

である。

## 5. current explicit-ref categoriesの意味を分ける

少なくとも次の三つは同一ではない。

```text
realization ref
  Business ActionのApplication realizationを示す

supporting semantic ref
  Validation / Authorization等、Business Actionを支える既存意味を参照する

target / concern ref
  Business Actionが対象とする別のApplication / Semantic actionを参照する
```

current fixtureは最初の二つを部分的に表現できるが、三つ目を明示しにくい。

この不足を理由に直ちにgeneric `SemanticRef { role, target }`をproduction Coreへ導入しない。

## 6. BP-03 / BP-05 Validation重複はsecondary pressure

current fixtureではBP-03とBP-05が同じ二つのValidationをexplicit参照する。

```text
BP-03
  review items and quantities
  -> quantity_positive
  -> line_number_unique

BP-05
  establish whether accepted deterministic constraints are satisfied
  -> quantity_positive
  -> line_number_unique
```

これは直ちにdefectとは判定しない。

同じValidationが複数Business Actionへ関係すること自体は成立し得る。

ただしcurrent representationは:

```text
BP-03  Validation is contextual/supporting
BP-05  Validation is the deterministic check being established
```

のようなrole差を表現していない。

したがってB-F23と同じpressureとして、BP-03側Validation refを一時subtractしたときcross-view diagnosis / impact analysisで何が失われるかをtest-onlyに比較する価値がある。

## 7. 最小pressure candidate

Lane Bからproduction schemaは指定しない。

Lane Cで比較すべき最小候補は次である。

### R0 — residual wording only

```text
BP-06
  security.authorize explicit ref
  protected submit actionはBusiness wordingのみ
```

長所:
- 新しい表現概念なし

弱点:
- submit action rename / replacementのimpactをexact traceできない
- lexical wordingへ依存

### R1 — test-only role-qualified cross-view ref

概念例:

```text
ref
  target = existing exact identity
  role   = realization | supports | concerns
```

これは**比較用のtest-only shape**でありproduction contract提案ではない。

確認したいのはrole vocabulary自体ではなく:

```text
target ref change can be observed
without converting the Business Action into an Application Function realization
```

である。

### R2 — field-specific authoring split

例:

```text
applicationPatterns      // realization only
applicationPatternRefs   // concern/target only
```

実験には単純だが、Data / Capability / Validationごとにref fieldが増殖する可能性がある。

### R3 — generic production relation

現時点ではdeferする。

一つのBP-06例だけを理由にgeneric Core relation / Edgeを作らない。

## 8. 必要なnegative / positive pressure

Lane C側で次をtest-onlyに確認できるとよい。

### P1 — Pattern targetをrealizationへ誤分類しない

BP-06が`order_entry.submit_order`を「認可対象」として参照しても:

```text
NoApplicationFunction = true
```

を維持できること。

### P2 — target refはexact identity

`order_entry.submit_order` rename / missing / ambiguousでfail closedすること。

名称類似やBusiness wordingから補完しない。

### P3 — target ref changeはBusiness cross-view change

認可対象が別accepted actionへ変わった場合、BP-06のBusiness cross-view meaning差として観測できること。

### P4 — realization unchanged

P3が起きてもBP-06へfake Application realizationを作らないこと。

### P5 — production generic relationを要求しない

このpressureがtest-only representationで説明できる間はgeneric Core relation / Business graphを追加しない。

## 9. PAS-07 mapping

B-F23から新E7を追加しない。

### E7-01

```text
Business Action != Application Function
```

target refが存在することをFunction realizationへ変換しない。

### E7-02

既存Application / Semantic authorityをexact identityでcross-view参照し、名称推測や意味複製をしない。

### E7-04

protected-action targetの変更をBusiness cross-view changeとして必要なら観測し、Application-owned meaningの偽変更へしない。

### E7-09

ref-role pressureだけを理由にgeneric Core Edge、universal relation、new Canonical kindを先行導入しない。

## 10. RQ-025 / RQ-026への意味

RQ-025:

```text
Business ActionはApplication Functionから独立できる
```

だけでなく:

```text
Business ActionがApplication Functionを参照する
  !=
そのFunctionがBusiness Actionをrealizeする
```

という境界をPressureする。

RQ-026:

Data / Function-Interaction / Businessをcross-checkするとき、identityだけでなく「なぜそのidentityがBusiness Actionへ結ばれているか」を失わないことが重要になる。

ただしrole taxonomyをProduct Requirementへ追加する段階ではない。

## 11. 現時点の判定

```text
B-F23 explicit cross-view ref role gap
  Confirmed as experiment representation pressure

production schema change necessity
  Not proven

BP-06 exact authorization authority trace
  Present: security.authorize

BP-06 exact protected-action target trace
  Missing / residual wording only

adding submit Pattern as current applicationPatterns
  Reject as semantic conflation

new generic Core relation
  Defer
```

このfindingはD3を否定しない。

むしろD3の`explicit semantic refs`を「identityの集合」とだけ考えると不足し得ることを示すpressureである。

## 12. 次

1. Lane CでBP-06 target-ref pressureをtest-onlyに追加する。
2. current `applicationPatterns`をrealization専用とみなす境界を維持する。
3. R0 / R1 / R2を比較し、最小のAuthoring / normalized representationでexact target traceが成立するか確認する。
4. BP-03 Validation refをsubtractして診断・impact lossを比較する。
5. actual runner Evidenceが得られるまでproduction contract adoptionへ進まない。
6. 複数realistic Referenceでrole-qualified refsが再発するまでgeneric production relationを作らない。

## 13. 境界

- Product Requirement追加なし
- PAS / E7追加なし
- Gate追加なし
- production generic SemanticRef追加なし
- generic Core Edge追加なし
- Business Flow schema追加なし
- Workflow Engine追加なし
- Authorization Role / policy ownership変更なし
- Gate B / Gate C Required Evidence変更なし
