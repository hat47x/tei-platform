# Business Procedure Handoff vs Business Flow Pressure — 2026-09

Status: **Lane C / X-TEA fallback audit / test-only pressure / no production Business Flow kind adopted**

## 1. 問い

Lane Cではcurrent `order_entry` Referenceから、accepted Business handoffをnormalized meaningから完全に落とすH0は弱く、thin handoff meaning H1を主候補としている。

現在のH1は概ね:

```text
Procedure/Step
  -> Procedure/Step
  + condition
```

である。

ここで次の短絡を防ぐ。

```text
H1がBusiness relationである
  -> H1をそのままBusiness Flow / DFDと呼ぶ
```

本監査では、渡辺幸三氏の言説、X-TEA / XEADの公開説明、XeadModeler sourceへ立ち返り、HandoffとBusiness Flowの境界を再確認する。

## 2. 設計原点のEvidence

### 2.1 三要素分析法

渡辺氏は業務システムをData / Function / Businessの三要素から捉え、Business Modelを業務マニュアルとして説明している。

業務システムは社会システムであり、Business側にはApplicationの内部実行順だけでは表せない人・外部主体・業務手順が含まれる。

### 2.2 X-TEA Modelerの公開説明

X-TEA HOMEではModelerが少なくとも次を作成できるものとして説明されている。

```text
business flow
ER diagram
business manual
```

つまりhistorical design surface上でも、Business FlowとBusiness Manualは同じ一個の図や同じ一個のobjectとして説明されていない。

### 2.3 XeadModeler source

Public repository:

```text
xead/XeadModeler
commit 68f1535b1c5dd80a327dfced37be401ecd3093c9
```

Relevant files:

```text
src/xeadModeler/DialogDataflowNode.java
src/xeadModeler/DialogDataflowLine.java
src/xeadModeler/DialogMatrixList.java
```

`DialogDataflowNode`では、nodeの`Type`が`Process`の場合に`TaskID`を使ってTaskを参照する。

一方、sourceは複数の非Process node typeも別に扱う。また非Process側ではRole参照を持つケースもある。

概念的には:

```text
Process node
  -> Task

other Dataflow nodes
  -> role / external / storage-like context etc.
```

`DialogDataflowLine`はDataflow node間を結ぶ独立要素で、node pairだけでなくline自身のname / direction / representation informationを持つ。

したがってhistorical Business Flowは、単純なTask A -> Task B control sequenceではなく、Processと他のparticipantsの間をflowが結ぶ独立model pressureを持つ。

## 3. TEI current Handoffとの差

### H1 — current thin Handoff

```text
BusinessHandoff-like relation
  from
    Procedure address
    Step local address
  to
    Procedure address
    Step local address
  condition
```

H1が解く問題:

- accepted handoff condition changeをsemantic diffから失わない
- endpoint rename / stale linkをexactに扱う
- file locator変更をsemantic changeへしない
- Runtime Workflowへ変換しない

### Business Flow / DFD pressure

将来のBusiness Flowが必要とする可能性がある意味:

```text
Business Activity / Process
external party / role / organizational participant
information / document / business-data flow
data source / store-like participant
flow direction / semantics
context boundary
```

これらはcurrent H1の意味ではない。

## 4. 固定する非同一性

```text
Business Procedure
  != Business Flow

Business Handoff
  != Business Flow

Business Handoff
  != Runtime Workflow transition

Business Flow
  != Data Relation
```

一つのEditor graphへProjectionすることはできても、一つのCanonical objectへ畳む根拠にはしない。

## 5. H1 / R3 / S4-Aへの影響

現在の主候補:

```text
H1
  thin normalized handoff meaning

R3
  Business semantic boundary内のthin relation

S4-A
  normalized ProcedureにProcedure-scoped Step local address metadata
```

本監査によりR3のscopeをさらに限定する。

```text
R3 = handoff-only relation boundary
```

R3へ次を追加しない。

```text
arbitrary Business graph edge
Dataflow payload
external party node taxonomy
Data store node taxonomy
Business Flow process graph
runtime event transition
```

## 6. なぜgeneric Business Edgeを作らないか

次のような万能edgeはcurrent Evidenceに対して大きすぎる。

```text
BusinessEdge
  from any Business subject
  to any Business subject
  kind
  payload
  metadata
```

これを導入すると、handoffだけを表すために:

- Business Flow
- organizational relation
- information flow
- event relation
- Procedure relation

を一つのstorage abstractionへ早期統合する圧力が生じる。

X-TEAへ戻った結果は逆であり、異なるdesign surfaceをcross-referenceさせることが重要であって、万能edgeへ畳むことではない。

## 7. TEI-local negative Pressure

`tools/reference-business-procedure-check/business_flow_scope_pressure_test.go`で次を固定する。

1. `normalizedBusinessModelProbe`は`Procedures` / `Handoffs`だけを所有し、Business Flow collectionを持たない。
2. `canonicalBusinessHandoffProbe`は`From / To / Condition`だけを持つ。
3. endpointは`Procedure / Step`だけを持つ。
4. Actor / Role / ExternalParty / Process / Data / Information / Document / Store / Payload / FlowType等をthin handoffへ混ぜない。
5. current normalized handoff validation PASSをBusiness Flow / DFD Conformanceと呼ばない。

actual Repository runnerでgreenになるまではexecution Evidenceとは数えない。

## 8. X-TEA Evidence Registerへの接続

`plan/XTEA_DESIGN_EVIDENCE_REGISTER_2026-09.md`へE-014として追加した。

結論:

```text
current H1/R3
  -> handoff-only

full Business Flow / DFD
  -> separate unresolved pressure
```

## 9. Product / Architectureへの影響

RQ-025 / RQ-026はBusiness Procedure viewpointとcross-view correctionを要求するが、current Requirementからfull Business Flow Canonical kindを直ちに導かない。

`docs/business-system-design.md`はすでに:

```text
Business Flow
Business Procedure
Application Interaction
Runtime Workflow Orchestration
```

を分離している。

本監査はそのArchitecture boundaryをReference / source Evidenceで補強する。

## 10. Reopen条件

Business Flow production contractをreopenするのは、realistic Referenceで少なくとも次が必要になった場合とする。

- Procedure handoffだけでは表せないinformation / document / data movement
- non-Procedure participantとのflow
- business activity間の疎結合なevent / information flow
- flow-level impact / review / traceabilityをpath-independentに保持する必要
- derived Projectionだけではaccepted meaningをlosslessに保てないEvidence

その場合もX-TEA DataflowNode / DataflowLine shapeをそのまま移植せず、TEI最小contractをReferenceから決める。

## 11. 現時点の判断

```text
H1 normalized handoff meaning
  -> current primary candidate

R3 thin Business relation
  -> handoff-onlyに限定

S4-A Step local address
  -> H1 endpoint integrityのためのprimary candidate

Business Flow / DFD production kind
  -> not adopted

generic Business Edge
  -> not adopted
```

この境界を維持したまま、D3 Business Procedure candidateの最小十分性を引き続きPressureする。
