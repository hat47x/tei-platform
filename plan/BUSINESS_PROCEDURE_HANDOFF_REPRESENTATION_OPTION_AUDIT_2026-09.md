# Business Procedure Handoff Representation Option Audit — 2026-09

Status: **Lane C / pre-ADR option audit / no production Core adoption**

## 1. 前提

`BUSINESS_PROCEDURE_HANDOFF_CANONICAL_NECESSITY_AUDIT_2026-09.md`では、current Referenceでhandoffをaccepted Business meaningとして扱う限り、authoring-only H0ではsemantic diff / impactからrelation changeを失うため、薄いnormalized handoff meaning H1を主候補とした。

次に問うのはH1をどの既存 / 新規boundaryへ置くかである。

## 2. Option R1 — existing `semantic.Relation`を再利用

現行`internal/semantic.Relation`はData配下にあり、次の意味を持つ。

```text
Data
  Relations[]
    id
    symbol
    target Data
    cardinality
    semantic attribute keys
```

これはData instance間の関係である。

Business handoffは:

```text
Procedure/Step
  -> Procedure/Step
  + Business condition
```

であり、Data cardinality / RelationKeyとは意味が異なる。

### 判定

**Reject for current pressure.**

専用kind追加を避けるためだけにBusiness handoffをData Relationへ押し込むと、三視点の境界を崩す。

## 3. Option R2 — generic Core `Edge` / `Relation` primitiveを新設

現行`internal/core`はIdentity / Version等の小さいprimitiveに留まり、汎用graph edge primitiveを持たない。

Handoff一件を表現するために:

```text
CoreEdge
  from any subject
  to any subject
  kind
  metadata
```

のような万能edgeを導入すると、

- Data Relation
- Application dependency
- Business handoff
- Source trace
- release dependency

等を一つの抽象へ早期統合する圧力が生まれる。

### 判定

**Reject / defer.**

複数domain relationから同じprimitive必要性が独立に出るまでMicro Kernelへ追加しない。

## 4. Option R3 — Business semantic boundaryのthin relation

Procedure bodyと同じBusiness normalized boundaryへ、bodyとは別collection / relationとして保持する。

概念形:

```text
BusinessModel
  Procedures[]
  Handoffs[]

Handoff
  from Procedure/StepAddress
  to Procedure/StepAddress
  condition
```

ここで`BusinessModel` / `Handoff`はarchitecture vocabularyであり正式type名ではない。

### 利点

- Data Relationと混ざらない
- Procedure bodyへruntime control flowを埋め込まない
- Handoff changeをnormalized semantic diffで観測できる
- Source path / fingerprintを含めない
- S4-AのProcedure-scoped Step local addressと整合する
- 将来Business Flowと分離したまま接続可能

### 判定

**Primary pre-ADR candidate.**

## 5. Option R4 — future Business Flow graphへ先送り

DFD型Business Flowは、Procedure handoffより広い。

```text
Event / circumstance
Business Activity
information / document / data source
external party
flow
```

current handoffをすべて将来Business Flowへ先送りすると、current accepted Procedure relationをnormalized layerから失う期間が生じる。

逆にhandoffのためだけにBusiness Flow全体を今実装すると過剰である。

### 判定

**Do not block H1 on future Business Flow.**

thin handoff relationをBusiness Flow全体と同一objectへ畳まない。

## 6. `ir.World`への意味

現行`ir.World`はSemantics / Applications / Databases等の領域集合を明示的に持ち、汎用edge bagを持たない。

したがってproduction採用時の自然な圧力は:

```text
World
  ...
  Business ?
```

または同等のBusiness normalized unitを検討することである。

ただし本監査から`ir.World.Business`追加を決定しない。

Procedure production adoption自体が未決だからである。

## 7. Step addressとの接続

R3 + H1を採用する場合:

```text
normalized Procedure
  ordered Step
    Procedure-scoped local address
    body meaning

normalized Business relation
  Handoff
    from local address
    to local address
    condition
```

となるS4-Aが有力になる。

local addressは:

- path independent
- Procedure scoped
- runtime identityではない
- source fingerprintではない

を維持する。

## 8. conditionの扱い

current Referenceの自然言語conditionをaccepted Business meaningとして保持する。

まだ次へ変換しない。

- Boolean expression
- executable predicate
- branch DSL
- runtime event condition
- state machine guard

condition wording changeをsemantic diffとして観測できることを先に満たす。

## 9. 現時点の順位

```text
R3 Business-boundary thin relation   1st
R4 future Business Flow              later integration concern
R2 generic Core Edge                 defer / reject for now
R1 semantic.Data.Relation reuse      reject
```

## 10. ADR前に必要なEvidence

1. actual testsでH0 semantic-loss pressureがgreen
2. H1 thin relationでcondition / endpoint changeを観測
3. S4-Aだけでhandoff referential integrityを検査
4. source moveでrelation meaning不変
5. global Step IDなしで成立
6. runtime Workflow fieldなしで成立
7. realistic multi-Procedure Referenceでhandoffが実際に設計価値を持つ

## 11. 非対象

- production Go package / field名
- `ir.World.Business`採用
- Business Flow schema
- generic graph engine
- runtime Workflow Engine
- 000039 change graph実装
