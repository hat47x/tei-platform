# Lane B — Business Procedure Validation Reference Necessity Audit — 2026-09

Status: **Lane B / B-F25 source-level resolved in Lane C / actual execution Evidence pending / no production schema adoption**

## 1. 問い

Primary `order_entry` Business Procedureでは当初、BP-03とBP-05が同じValidationをexplicit参照していました。

```text
BP-03
  review the items and quantities represented by the order lines
  -> order_entry.order_lines
  -> quantity_positive
  -> line_number_unique

BP-05
  establish whether accepted deterministic order constraints are satisfied
  -> quantity_positive
  -> line_number_unique
```

この重複が本当にBusiness cross-view meaningとして必要かを監査しました。

## 2. B-F25 — related meaning != required explicit ref

`order_entry.order_lines`は`sales.order.lines` Relationを使うheader-detail Patternです。

一方、二つのValidationは独立Semantic authorityです。

```text
sales.order_line.quantity_positive
sales.order.lines.line_number_unique
```

BP-03の「明細をレビューする」というBusiness Actionが、これらのValidationを確立することまで意味するEvidenceはありません。

それを明示しているのはBP-05です。

したがって:

```text
semantically related
  != Business Step must explicitly reference

explicit refs are evidence-driven, not proximity-driven
```

を採用しました。

## 3. Lane C response

Lane C PR #482はこのpressureをsource/test levelで取り込みました。

Primary fixtureからBP-03のValidation refsを除去し:

```text
BP-03
  applicationPatterns = [order_entry.order_lines]
  validations = []

BP-05
  validations = [
    sales.order_line.quantity_positive,
    sales.order.lines.line_number_unique
  ]
```

へ変更しています。

`explicit_ref_authority_pressure_test.go`では:

- BP-03にValidation authorityを重複保持しない
- BP-05にdeterministic Validation refsを保持する
- BP-04 / BP-08 / BP-09へ未証明explicit refを付けない
- current Referenceではdirect Data / Relation refsをD3 minimumと過大主張しない
- BP-07 Pattern-owned `sales.order.submit` CapabilityをBusiness sourceへコピーしない

というnegative pressureも追加されました。

`canonical_meaning_pressure_test.go`もBP-03とBP-05の役割分離へ追随しています。

## 4. 判定

```text
B-F25 BP-03 Validation ref necessity
  source-level resolved: not required

BP-03 Validation refs
  removed in Lane C Reference branch

BP-05 Validation refs
  retained as evidence-backed direct refs

generic supports/checks role taxonomy
  still not justified

actual execution Evidence
  pending
```

したがってB-F25はArchitecture上の未解決ではなく、actual runner確認待ちへ縮退します。

## 5. D3への含意

この結果はD3の`explicit semantic refs`を増やす方向ではなく、より小さくします。

```text
explicit ref
  = accepted Business Action自身が直接依存し
    existing authorityからlosslessly deriveできない場合だけ保持
```

current Referenceでminimumとして実証されつつあるcategoryは:

```text
Application Pattern
Validation
Capability
```

です。

ただしcategoryが存在することと、すべてのStepがそのcategoryを持つことは別です。

Direct Data / Relation refsはcurrent Referenceではminimum必要性未証明です。

## 6. B-F23/B-F24との関係

B-F25が解けたことで、BP-03/BP-05を理由にgeneric role-qualified ref taxonomyへ進む必要性はさらに弱まりました。

残る強いref-role pressureはBP-06です。

```text
supporting semantic authority = security.authorize
protected Application target  = order_entry.submit_order
Application Function realization = none
```

これはValidation重複とは別問題であり、`target ref != realization ref`として引き続きO1 test-only pressureを待ちます。

## 7. PAS-07 mapping

新E7は追加しません。

```text
E7-02
  evidence-backed explicit refsだけをexact traceし、meaningを水増ししない

E7-04
  Business-only meaningを保持するが、単なる関連性をdependencyへ昇格しない

E7-09
  generic ref-role taxonomy / Core relationを一例から固定しない

S7-01
  manual business judgment != deterministic Validation
```

## 8. 次

1. actual Repository runnerでLane CのB-F25関連testsがgreenになることを確認する。
2. B-F25を理由にA54 / 新E7を追加しない。
3. BP-06 protected-target O1 pressureを別問題として継続する。
4. 複数realistic Referenceで異なるref-roleが反復するまでgeneric role taxonomyをreopenしない。
5. matrix completenessを目的にexplicit refsを増やさない。

## 9. 境界

- Product Requirement追加なし
- PAS / E7追加なし
- production ref-role enum追加なし
- Validation execution ownership変更なし
- Gate B / Gate C Required Evidence変更なし
