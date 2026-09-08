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

## 7. Watanabe external evidenceからのcorroboration

`BUSINESS_HANDOFF_WATANABE_SECOND_REALISTIC_EVIDENCE_PRESSURE_2026-09.md`では、渡辺幸三氏の公開されている三要素分析法資料をexternal design evidenceとして確認した。

Relevant source meaning:

```text
Data model
Function model
Business model
  -> independent views with mutual constraint

Business model
  -> work / role / procedure / social-system meaning

Business Flow
  -> three viewsを広く俯瞰する別view
  -> extended DFD
  -> process activation eventを含む
```

特に、現場の仕事が単純な業務間logicではなくevent-drivenに配置されるという説明は:

```text
Step adjacency
  -X-> responsibility Handoff

Application Function sequence
  -X-> responsibility Handoff
```

というcurrent authority boundaryを外部からcorroborateする。

同時にBusiness Flowがevent / process / broader flowを持つことは:

```text
R3 handoff-only
  != R4 Business Flow
```

を補強する。

ただしexternal conceptual evidenceだけで:

- Handoff `eventId`
- document / payload field
- participant taxonomy
- full DFD
- SubjectArea Canonical kind

を採用しない。

Watanabe evidenceはcurrent H1/R3 insufficiencyをまだ証明しない。

## 8. Step addressとの接続

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

## 9. conditionの扱い

current Referenceでは自然言語conditionをaccepted Business relation meaningとして保持する。

まだ次へ変換しない。

- Boolean expression
- executable predicate
- branch DSL
- runtime event guard
- state machine guard

Watanabe external evidenceはBusiness workにevent contextがあることを補強するが、Handoff condition自体をruntime event guardへ変換する根拠にはならない。

condition wording changeをsemantic diffとして観測できることを先に満たす。

## 10. Handoff conditionとProcedure trigger

Current Procedure trigger:

```text
trigger { epistemicStatus, meaning }
```

とHandoff conditionは別authorityとして維持する。

```text
Handoff condition
  != target Procedure trigger by definition
```

Handoff以外のeventからProcedureが始まる場合もあり得るため、一方から他方を自動deriveしない。

Actual green後のsecond realistic Handoffでは、両者が同じcase / 異なるcaseをTEI-original sourceでPressureする価値がある。

## 11. 現在の順位

```text
R3  Business-boundary thin handoff relation  primary
R4  full Business Flow / DFD                 separate pressure
R2  generic Core Edge                        reject / defer
R1  semantic.Data.Relation reuse             reject
```

R3は**handoff-only**であり、generic Business relation bagではない。

X-TEAに加えてWatanabe三要素分析法の公開資料からもR3/R4 separationへのcorroborationが得られたが、production adoption条件は変えない。

## 12. test-only Evidence

関連tests:

- `canonical_handoff_pressure_test.go`
- `handoff_canonical_necessity_pressure_test.go`
- `step_address_layer_pressure_test.go`
- `normalized_business_model_pressure_test.go`
- `business_flow_scope_pressure_test.go`
- `handoff_standalone_authoring_pressure_test.go`
- `business_authoring_authority_cutover_pressure_test.go`

Pressure:

1. H0ではrelation changeをbody-only snapshotから失う
2. H1ではcondition / endpoint changeを保持
3. S4-Aだけでendpoint integrityを検査
4. source locator moveでrelation meaning不変
5. global Step IDなしで成立
6. runtime Workflow fieldなしで成立
7. Handoff typeへBusiness Flow participant / payload意味を混ぜない
8. current validator PASSをBusiness Flow / DFD Conformanceと呼ばない
9. legacy Reference Set Handoffとstandalone Handoffをaccepted dual authorityにしない
10. migration previewでlegacy sourceを読んでもcandidate非authoritativeならmultiple SOTとはみなさない

actual Repository runnerでgreenになるまではexecution Evidenceではない。

## 13. ADR前にまだ決めないもの

- production Go package / field名
- `ir.World.Business`採用
- production `BusinessHandoff` kind
- Business Flow / DFD schema
- generic graph engine
- condition expression language
- Step local address syntax / generation rule
- 000039 change graph integration
- permanent old/new Handoff dual-read
- Handoff event/document/payload fields

Production採用はactual green、より現実的なReference、Lane B再監査を経て判断する。

## 14. Current Gate C implication

Watanabe external evidenceにより:

```text
Handoff non-inference boundary
R3 / R4 separation
Business / Function / Data cross-view structure
```

は強くなった。

一方Pre-ADR Gate Cの:

```text
second realistic Handoffで
from / to / condition sufficiencyを確認
```

は未充足である。

External patternをTEI fixtureへコピーせず、actual Repository green後にTEI-original small realistic caseをauthorして判断する。
