# Business Procedure Application Pattern Target Pressure — 2026-09

Status: **Lane C / O1 test-only representation pressure / no authoring or production schema adoption / actual execution Evidence pending**

## 1. 問い

Lane B B-F23 / B-F24から、BP-06 Authorization concernには次の三つが同時に存在することが分かった。

```text
supporting semantic authority
  security.authorize

protected Application target
  order_entry.submit_order

direct Application Function realization
  none
```

current Reference checkerでは`applicationPatterns[]`がdirect Application Function realizationを表している。

したがってBP-06へ単純に:

```text
applicationPatterns:
  - order_entry.submit_order
```

を追加すると:

```text
BP-06 concerns/protects submit action
  -> BP-06 is realized by submit action
```

へ意味を誤変換する。

本Pressureでは、Application Patternという同じauthority kindへのrefでも、**target / concernとrealizationを混同しない最小表現が可能か**をtest-onlyに確認する。

## 2. D4との関係

D4はcurrent Evidenceから未証明のdirect Data / Relation slotをminimum candidateからsubtractし、Application Pattern / Validation / Capabilityというauthority kindを保持する比較候補である。

しかし:

```text
ref kind separation
  !=
ref role separation
```

である。

BP-06とBP-07はいずれもApplication Pattern identityへ関係するが、意味は異なる。

```text
BP-07 -> order_entry.submit_order
  direct realization

BP-06 -> order_entry.submit_order
  protected target / concern
  not realization
```

したがってO1はD4を置き換えるものではなく、D4だけでは解けない別pressureである。

## 3. O1 test-only shape

Production fieldを先に固定せず、test-only candidateとして次を比較する。

```text
TargetAwareStepProbe
  body
    businessAction
    applicationPatterns[]  // current D3/D4 body。direct realization only
    validations[]
    capabilities[]
  applicationPatternTargets[]  // test-only protected target / concern
```

ここで`applicationPatternTargets`は**比較用語彙**である。

- current `step` fixture schemaへ追加しない
- `business-procedure-reference-v0`を変更しない
- production Canonical fieldとして採用しない
- generic ref `role` enumを導入しない

まず意味の必要性だけを検証する。

## 4. source-level test-only Pressure

`tools/reference-business-procedure-check/application_pattern_target_pressure_test.go`で次を実装した。

### O1-1 target != realization

BP-06について:

```text
ApplicationPatterns = []
ApplicationPatternTargets = [order_entry.submit_order]
Capabilities = [security.authorize]
NoApplicationFunction = true
```

を同時に保持できること。

### O1-2 exact target resolution

`order_entry.submit_order`をnormalized `ir.World`でexactに1件解決する。

名称類似や文字列prefixから補完しない。

### O1-3 missing / ambiguous fail closed

missing targetは:

```text
BUSINESS_APPLICATION_PATTERN_TARGET_REF_NOT_FOUND
```

ambiguous targetは:

```text
BUSINESS_APPLICATION_PATTERN_TARGET_REF_AMBIGUOUS
```

としてblocking diagnosticへできることをPressureする。

このdiagnostic kind自体もtest-only comparison vocabularyであり、production Conformance code採用ではない。

### O1-4 target-only change

```text
order_entry.submit_order
  -> order_entry.order_record
```

のようにtargetだけを別のaccepted Patternへ変更した場合:

- BP-06 Business bodyは不変
- direct Application Function realizationはnoneのまま
- target-aware cross-view meaning差は観測できる

ことを確認する。

### O1-5 adjacencyからderiveしない

BP-07のdirect realization Patternを変更しても、BP-06 targetを自動変更しない。

```text
ordered adjacency
  != Authorization protected-target relation
```

を固定する。

### O1-6 Legacy policy shapeを逆流させない

Test-only target candidateへ次を入れない。

- organization code
- role code
- allow / deny effect
- Principal
- Permission
- provider policy

Legacy Auth Referenceに`execute + order_entry.submit_order` ruleが存在することはtarget identityのcross-view corroborationには使えるが、Business Procedureのauthorityにはしない。

## 5. なぜauthoring fixtureへまだ追加しないか

O1がsource-level testとして説明できても、ただちにPrimary fixtureへ新fieldを追加すると:

```text
representation pressure found
  -> authoring schema adopted
```

という先行固定になる。

現在必要なのはまず:

```text
target / concern relation carries accepted meaning
  AND
it cannot be represented as realization
```

をactual runnerでPressureすることである。

actual green後に、次を改めて比較する。

1. current Business wordingだけで十分か
2. O1のfield-specific authoring refが必要か
3. D4 bodyへどの形で統合するか
4. 複数realistic Referenceで同じrole差が反復するか

## 6. generic role-qualified refをまだ作らない

次はdeferする。

```text
ExplicitSemanticRef {
  Kind
  Role
  Target
}
```

理由:

- current強いrole pressureはBP-06一例
- BP-03/BP-05 Validation重複はref削減で解けた
- Observationは独立Semantic authority自体が未証明
- `supports / concerns / protects / checks / realizes` taxonomyは未証明

したがって:

```text
O1 field-specific pressure
  before
O2 generic role taxonomy
```

を維持する。

## 7. RQ / PASへの対応

新しいProduct Requirement / PAS / E7 / Acceptance IDは追加しない。

RQ-025は既に:

```text
Business Action != Application Function
どのApplication Functionをどう利用するかを設計できる
```

を要求している。

RQ-026はBusiness ProcedureからApplication Pattern / Action / Capabilityへのtraceabilityを要求している。

O1はこの既存RequirementをReference representationへPressureする。

PAS-07では既存E7-01 / E7-02 / E7-04 / E7-09で扱える。

## 8. 現時点の判定

```text
protected target exact identity need
  Confirmed as representation pressure

applicationPatterns realization field reuse
  Reject

adjacency / wording / Legacy policy derivation
  Reject

O1 field-specific target-aware test shape
  Source-level implemented
  actual execution pending

authoring fixture field adoption
  Not decided

generic role-qualified ref
  Deferred

production schema change
  Not proven
```

## 9. 次

1. actual Repository environmentでO1 testを実行する。
2. O1がgreenでも即authoring schemaへ昇格しない。
3. D4 ref-kind pressureとO1 role pressureを組み合わせたとき、current meaningを最小に保持できるか比較する。
4. 複数realistic Referenceでtarget/concern roleが反復するまでgeneric role taxonomyを作らない。
5. Business ProcedureからLegacy Authorization policyを参照・複製しない。
6. actual greenまでproduction Canonical ADRへ進まない。
