# Business Procedure O1 Cardinality Pressure — 2026-09

Status: **Lane C / minimum-source cardinality pressure / no production schema adoption**

## 1. 問い

`D4 + narrow O1`をcurrent minimum semantic candidateとした後、protected Application targetを最初からcollectionとして固定すべきかを確認する。

Current strong caseはBP-06だけであり、accepted targetは一つである。

```text
BP-06
  supporting capability = security.authorize
  protected Application Pattern = order_entry.submit_order
  direct Application Function = none
```

## 2. Current evidence

Current Referenceには、一つのBusiness Actionが複数のApplication Patternを同時にprotected targetとして持つaccepted caseはない。

またO1の目的はgeneric relation modelを作ることではなく、BP-06で失われるexact target traceを最小限に保存することである。

したがってcollection cardinalityを先回りして固定する根拠はない。

## 3. Minimum candidate

Current comparison vocabularyは:

```text
D4 Step
  Body
  protectedApplicationPattern?  // zero or one
```

とする。

ここでzeroは:

```text
no protected target authored for this Business Action
```

であり、

```text
there can never be a protected target
```

を意味しない。

oneはcurrent BP-06を表す。

## 4. Multiple target future case

将来realistic Business sourceで:

```text
one Business Action
  -> protects Application Pattern A
  -> protects Application Pattern B
```

がacceptedされた場合、current minimum projectionは黙って:

- 最初の一件だけ残す
- 重複除去してcollectionへ暗黙拡張する
- Business wordingへ押し戻す

ことをしない。

Fail closedしてO1 cardinalityをreopenする。

比較候補はその時点で:

```text
zero-or-many explicit targets
one Business Action split into narrower Business Actions
a named protected-target relation
another domain-specific boundary
```

をEvidenceから選ぶ。

## 5. Why singular before plural

### Singular

Pros:
- current Evidenceと一致
- source / reviewが最小
- generic relation modelへ近づかない
- one Business Action / one targetというcurrent意味が明瞭

Risk:
- future multi-target caseでschema reopenが必要

これは意図的なfail-closed costとして受け入れる。

### Plural from the start

Pros:
- future multi-targetを受けられる

Risks:
- current未証明cardinalityを固定
- collection ordering / duplicate semantics / empty-vs-absent等を先回り
- generic target relationへの滑り込みを招く

Current: **not preferred**。

## 6. Test pressure

`d4_o1_minimum_losslessness_pressure_test.go`のminimum candidateをscalarへ縮小した。

Pressure:

1. BP-06のsingle exact targetを保持する。
2. target-only changeをmeaning差として保持する。
3. targetをdirect realizationへ変換しない。
4. unrelated Stepにsymmetry目的のfieldを要求しない。
5. unknown Stepはfail closedする。
6. multiple target inputはcurrent Evidence未証明としてfail closedする。

## 7. Test-only sidecarとの違い

`order-entry-protected-target-overlay.json`はO1 meaning / reload / AcceptanceをPressureするための実験artifactであり、現在は`applicationPatternTargets[]`というcollection shapeを使う。

これはproduction cardinality decisionではない。

```text
pressure apparatus shape
  != production minimum source shape
```

とする。

Sidecarは複数source Acceptanceのcostを露出させるため残し、minimum production comparisonはscalar narrow fieldで行う。

## 8. Current decision

```text
O1 semantic necessity
  current BP-06でconfirmed as pressure

minimum cardinality
  zero-or-one

minimum comparison field
  protectedApplicationPattern?

multiple target
  fail closed / reopen

production field adoption
  deferred

generic role relation
  deferred
```

この判断は`BUSINESS_PROCEDURE_D4_O1_MINIMUM_CONTRACT_AUDIT_2026-09.md`をさらに縮小するcardinality refinementである。
