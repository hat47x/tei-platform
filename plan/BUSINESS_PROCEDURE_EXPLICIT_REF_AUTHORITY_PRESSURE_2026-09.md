# Business Procedure Explicit Ref Authority Pressure — 2026-09

Status: **Lane C / test-only architecture pressure / no production Core adoption**

## 1. 問い

D3 candidateはBusiness Procedure Step bodyに`explicit semantic refs`を残す。

しかし、参照できるCanonical identityが存在することと、Business sourceがそのidentityを直接保持すべきことは同じではない。

Pattern-owned meaningや別Business ActionのauthorityまでBusiness sourceへ複製すると、三視点cross-checkではなく二重authoringになる。

本Pressureでは、current `order_entry` Referenceのexplicit refsを次の基準で再監査する。

```text
existing authority exists
  AND
Business Action itself explicitly depends on that authority
  AND
reference cannot be losslessly derived from an already-explicit Pattern ref
    -> keep explicit ref

otherwise
    -> do not copy merely for symmetry or convenience
```

## 2. current Referenceで保持するexplicit refs

### Application Pattern

次はBusiness Actionが既存Application interactionを実際に利用するため保持する。

```text
BP-00 -> order_entry.recent_orders
BP-01 -> order_entry.order_record
BP-02 -> order_entry.customer_reference
BP-03 -> order_entry.order_lines
BP-07 -> order_entry.submit_order
```

PatternのData / Relation / Query / Capability / Datasource / owner Applicationはnormalized `ir.World`からderiveする。

Business sourceへ再記述しない。

### Validation

決定的制約を検査するBusiness ActionはBP-05である。

```text
BP-05
  -> sales.order_line.quantity_positive
  -> sales.order.lines.line_number_unique
```

これらはBP-03の「order linesをレビューする」Business Actionから除く。

BP-03が`order_entry.order_lines`を使うことと、その後BP-05でaccepted deterministic constraintsを評価することは別のBusiness Actionである。

同じValidation refsを両方へ付けると、一つのauthorityを二つのStepが所有するように見え、変更時のBusiness semantic diffを不必要に増やす。

### Capability

BP-06はAuthorization判断そのものを行い、既存Semantic authority `security.authorize`が存在するためdirect Capability refを保持する。

```text
BP-06 -> security.authorize
```

一方BP-07の`order_entry.submit_order` Patternが既に`sales.order.submit`を所有しているため、BP-07へ同Capabilityを直接コピーしない。

## 3. explicit refを持たないresidual actions

```text
BP-04 human judgment
BP-08 Observation intention
BP-09 manual / external continuation
```

BP-04 / BP-09にfake Application Functionを作らない。

BP-08には現時点で独立したSemantic Observation authorityが存在しない。Driverの`ExecuteObservationRead` / `ReadIdentity`はruntime/Evidence boundaryであり、Business Procedureの対称性のためにSemantic refへ昇格させない。

## 4. Data / Relation direct refの扱い

current `order_entry` Referenceには、Patternから独立してBusiness Action自身がdirect Data / Relation identityを参照しなければ意味を失う例がない。

したがってcurrent Evidenceから言えることは次までとする。

```text
D3 minimumで実証済みのexplicit ref categories
  Application Pattern
  Validation
  Capability

Direct Data / Relation
  current Referenceではminimum必要性未証明
```

これはData / Relation direct refsの永久禁止ではない。

将来のrealistic Referenceで、Application Pattern利用とは独立したBusiness Actionがaccepted Data / Relation identityを直接参照しなければ意味を失う場合に再openする。

## 5. duplicate authored meaningのnegative Pressure

BP-07は`order_entry.submit_order` Patternだけで`sales.order.submit` Capabilityへtraceできる。

ここへ同CapabilityをBusiness sourceへコピーすると、実際のBusiness decisionを増やしていないのにD3 Business meaningだけが変化する。

```text
before
  BP-07 -> Pattern order_entry.submit_order

redundant copy
  BP-07 -> Pattern order_entry.submit_order
        -> Capability sales.order.submit
```

この差は有益なcross-view差ではなく、二重authoringによるdrift surfaceである。

`explicit_ref_authority_pressure_test.go`でこの性質をnegative Pressureとして固定する。

## 6. realizationとの関係

explicit refの有無は`realization` labelから導出しない。

- `security-boundary`だからCapabilityを推測するのではない
- `observation-boundary`だからObservation Capabilityを捏造するのではない
- `validation-support`だからValidation名を名称規則で推測するのではない

explicit refはBusiness authoring sourceに明示され、normalized Worldでexact resolutionされる。

`realization_diagnostic_independence_pressure_test.go`で、realization-only変更がchecker report/diagnosticsへ影響せず、壊れたexplicit refはrealization wordingに関係なくfail closedするPressureを持つ。

## 7. pre-ADR判断

current D3 candidateの`explicit semantic refs`は、万能な`[]CanonicalID` surfaceを先に固定するための根拠ではない。

Referenceから現在実証できているのは、異なるauthority kindへの薄いcross-view linksである。

production contractでは次を満たす必要がある。

1. authority kindを失わない
2. exact identityで解決する
3. Pattern-owned meaningをコピーしない
4. 別Business Actionのauthorityを重複しない
5. missing authorityを名称類似で補完しない
6. future Referenceで必要になったref kindをEvidenceなしに先取りしない

actual Repository runnerでPressure testsがgreenになるまではproduction Canonical採用へ進めない。
