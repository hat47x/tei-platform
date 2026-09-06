# Business Procedure Handoff Representation Option Audit — 2026-09

Status: **Lane C / pre-ADR option audit / R3 handoff-only primary / no production Core adoption**

## 1. 前提

current Referenceのaccepted handoffをAuthoringだけへ残すH0では、condition / endpoint changeをnormalized semantic diffから失う。

したがってH1として次の薄いnormalized meaningを保持するPressureを置いている。

```text
Procedure/Step
  -> Procedure/Step
  + Business condition
```

本監査は、このH1をどのboundaryへ置くかを比較する。

## 2. R1 — existing `semantic.Relation` reuse

現行`semantic.Relation`はData配下にあり、Data target / cardinality / semantic attribute keysを持つData instance relationである。

Business handoffはData cardinality / RelationKeyとは別意味である。

**判定: reject.**

専用kind追加を避けるためだけにBusiness handoffをData Relationへ押し込まない。

## 3. R2 — generic Core Edge

```text
CoreEdge
  from any subject
  to any subject
  kind
  metadata
```

のような万能edgeをHandoff一件のために導入すると、Data Relation、Application dependency、Business relation、Source trace、release dependency等を一つの抽象へ早期統合する圧力が生まれる。

**判定: reject / defer.**

複数domainから独立に必要性が出るまでMicro Kernelへ追加しない。

## 4. R3 — Business semantic boundaryのthin handoff relation

```text
NormalizedBusinessModel
  Procedures[]
  Handoffs[]

Handoff
  from Procedure/StepLocalAddress
  to   Procedure/StepLocalAddress
  condition
```

`BusinessModel` / `Handoff`は現段階ではarchitecture / test vocabularyであり、正式production type名ではない。

利点:

- Data Relationと混ざらない
- Procedure bodyへruntime control flowを埋め込まない
- accepted handoff changeをnormalized semantic diffで観測できる
- Source path / fingerprintを含めない
- S4-AのProcedure-scoped Step local addressと整合する
- Handoff以外のBusiness Flow意味を所有しない

**判定: primary pre-ADR candidate.**

## 5. R4 — full Business Flow / DFD

R4をR3の「将来格納先」とは扱わない。

X-TEA fallbackで確認したBusiness Flow pressureはHandoffより広い。

```text
Business Activity / Process
external party / role / participant
information / document / business-data flow
data source / store-like participant
flow direction / semantics
```

XeadModeler sourceでも`Process` DataflowNodeはTaskへ参照し、他のDataflow node typeとDataflowLineから独立したflow modelを形成する。

したがって:

```text
R3 thin handoff relation
  !=
R4 full Business Flow / DFD
```

**判定: separate unresolved pressure.**

Business Flowはrealistic Referenceが必要性を示した場合だけ別途reopenする。R3を後からR4へ「統合する」ことを前提にしない。

## 6. X-TEA fallbackからのguardrail

`plan/XTEA_DESIGN_EVIDENCE_REGISTER_2026-09.md` E-014を根拠に次を固定する。

```text
Business Handoff != Business Flow
Business Flow != Procedure
Business Handoff != Runtime Workflow transition
Business Flow != Data Relation
```

一つのEditor graphへProjectionすることは可能でも、一つのCanonical object / generic edgeへ畳む根拠にはしない。

## 7. Step addressとの接続

H1 + R3をnormalized semantic meaningとして扱う場合、handoff endpointはnormalize後にSource Indexへ戻らずexact解決できる必要がある。

現在の主候補:

```text
S4-A
  normalized Procedure
    ordered Step
      Procedure-scoped local address
      body meaning
```

local addressは:

- Procedure scoped
- file path independent
- Source fingerprintではない
- global Step identityではない
- runtime instance / token / state identityではない

Step body meaningとreferential address changeを分ける。

## 8. conditionの扱い

current Referenceでは自然言語conditionをaccepted Business relation meaningとして保持する。

まだ次へ変換しない。

- Boolean expression
- executable predicate
- branch DSL
- runtime event guard
- state machine guard

condition wording changeをsemantic diffとして観測できることを先に満たす。

## 9. 現在の順位

```text
R3  Business-boundary thin handoff relation  primary
R4  full Business Flow / DFD                 separate pressure
R2  generic Core Edge                        reject / defer
R1  semantic.Data.Relation reuse             reject
```

R3は**handoff-only**であり、generic Business relation bagではない。

## 10. test-only Evidence

関連tests:

- `canonical_handoff_pressure_test.go`
- `handoff_canonical_necessity_pressure_test.go`
- `step_address_layer_pressure_test.go`
- `normalized_business_model_pressure_test.go`
- `business_flow_scope_pressure_test.go`

Pressure:

1. H0ではrelation changeをbody-only snapshotから失う
2. H1ではcondition / endpoint changeを保持
3. S4-Aだけでendpoint integrityを検査
4. source locator moveでrelation meaning不変
5. global Step IDなしで成立
6. runtime Workflow fieldなしで成立
7. Handoff typeへBusiness Flow participant / payload意味を混ぜない
8. current validator PASSをBusiness Flow / DFD Conformanceと呼ばない

actual Repository runnerでgreenになるまではexecution Evidenceではない。

## 11. ADR前にまだ決めないもの

- production Go package / field名
- `ir.World.Business`採用
- production `BusinessHandoff` kind
- Business Flow / DFD schema
- generic graph engine
- condition expression language
- Step local address syntax / generation rule
- 000039 change graph integration

Production採用はactual green、より現実的なReference、Lane B再監査を経て判断する。
