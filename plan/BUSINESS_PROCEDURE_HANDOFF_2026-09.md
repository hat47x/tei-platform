# Business Procedure Handoff (H1/R3) — 2026-09

Status: **Lane C / pre-ADR option audit / R3 handoff-only primary / no production Core adoption**

## この文書について

本書は、Business Procedure間のhandoffをnormalized/Canonical meaningとしてどこまで・どう保持すべきかを時系列で調査した次の4文書を1本へ統合したものである。

1. `BUSINESS_PROCEDURE_HANDOFF_CANONICAL_PRESSURE_2026-09.md`（元・handoffの最小relation形を定義する最初のPressure）
2. `BUSINESS_PROCEDURE_HANDOFF_CANONICAL_NECESSITY_AUDIT_2026-09.md`（元・H0(authoring-only) vs H1(thin normalized meaning)比較、H1採用）
3. `BUSINESS_PROCEDURE_HANDOFF_REPRESENTATION_OPTION_AUDIT_2026-09.md`（元・H1をどこへ置くかのR1〜R4比較、R3採用）
4. `BUSINESS_PROCEDURE_HANDOFF_VS_BUSINESS_FLOW_PRESSURE_2026-09.md`（元・R3とBusiness Flow/DFDの境界を外部Evidenceで補強）

後続文書は前段の結論を明示的に引用・比較・補強する一続きの調査であるため統合した。個別ファイルの履歴はGit historyから追跡できる。

外部から旧ファイル名を参照していた文書（`plan/BUSINESS_PROCEDURE_IDENTITY_SHELL_PRESSURE_2026-09.md`、`plan/BUSINESS_PROCEDURE_MAIN_BASELINE_REAUDIT_2026-09.md`、`plan/BUSINESS_PROCEDURE_NORMALIZED_MODEL_PRESSURE_2026-09.md`）は本統合と同じPRで本ファイル名へ参照を更新した。

---

# Part 1 — Handoff Canonical Pressure（最小relation形の定義）

## 目的

Business Procedure間handoffを、Procedure本文へ埋め込むruntime control flowではなく、Procedure / Step間のBusiness semantic relationとして正規化できるかを確認する。

前段では`order-entry-procedure-reference-set.json`により、Reference-local Procedure key + Step keyでhandoffをfile pathから分離できた。

次は、そのrelation自体をaccepted Business meaningとして扱う最小形をpressure testする。

## 1. 最小relation

```text
BusinessHandoff
  from
    procedure
    step
  to
    procedure
    step
  condition
```

### 入れないもの

- source file path
- fingerprint
- queue
- retry
- timer
- scheduler
- lease
- saga
- runtime state
- execution ownership
- transport
- Authorization Role

handoffを記述できることからruntime orchestrationを導入しない。

## 2. Procedure bodyと分離する理由

handoffは二つのProcedure / Stepを結ぶrelationである。

一方のProcedure本文へ相手側の業務意味をコピーすると、

- N:M relationを局所objectへ押し込む
- duplicate authorityを作る
- file move / rename impactを広げる
- Business FlowとProcedureを混同する

危険がある。

したがってtest-onlyでは、Procedure body meaningとhandoff relationを別normalized projectionとして扱う。

## 3. locator独立性

Canonical handoff endpointは

```text
Procedure key + Step key
```

である。

Reference setの

```text
Procedure key -> artifact locator
```

mappingはauthoring / Source側に残す。

pure file moveでlocator mappingが変わってもhandoff semantic endpointは変化しない。

## 4. conditionはBusiness meaning

handoff conditionの文言変更はBusiness semantic changeとして扱う。

```text
from/to = same
condition = changed
```

であればProcedure body identityは同じでもhandoff relation meaningは変わる。

identity-only shellではこの差も表現できないため、thin Business meaning pressureを補強する。

## 5. fail-closed endpoint resolution

handoffは存在するProcedure / Stepへexact referenceしなければならない。

- missing Procedure
- missing Step
- stale logical rename

を名称類似で補完しない。

Referenceでは既存`validateReferenceSetHandoffs`と新しいnormalized relation probeの両方でこの境界を確認する。

## 6. duplicate relation

同じ

```text
from Procedure/Step
-> to Procedure/Step
+ condition
```

を重複受理しない。

これはduplicate source authorityを防ぐための最小guardであり、将来複数の異なるhandoff conditionを同じendpoint間に持てるかまで固定しない。

## 7. test-only Evidence（Part 1）

`tools/reference-business-procedure-check/canonical_handoff_pressure_test.go`

確認項目:

1. relation fieldを`from / to / condition`へ限定
2. endpoint fieldを`procedure / step`へ限定
3. current Reference setからhandoffを導出
4. Procedure / Step endpointをexact解決
5. artifact locator変更でhandoff meaning不変
6. condition変更でhandoff meaning change
7. Procedure body meaningを不要に変更しない
8. missing endpointをfail closed
9. duplicate relationをfail closed

actual Repository runnerがGo testへ到達するまではgreen Evidenceとは数えない。

## 8. Option Dへの意味

`BUSINESS_PROCEDURE_CANONICAL_MEANING_PRESSURE_2026-09.md`のOption Dを次の二つへ分けられる可能性がある。

```text
Canonical Business Procedure body
  purpose / trigger / actor / ordered step meaning / explicit refs

Canonical Business relation
  handoff between Procedure / Step endpoints
```

これによりBusiness Flow / Procedure / Runtime Workflowを一つの巨大objectへ畳まずに済む。

## 9. まだ固定しないもの（Part 1）

- handoffを`ir.World`のどのfield / packageへ置くか
- handoff relationの専用Canonical kind名
- Business Activity / Eventの別model
- exception flow
- conditional branch syntax
- completion state machine
- runtime queue / retry
- multi-party organization model

current Referenceで必要になった最小semantic relationだけをpressure testする。

## Part 1 次の判断

thin Procedure body + thin handoff relationで、

- Business-only diff
- cross-view impact
- file move / logical rename
- manual work
- multiple Procedure -> same Pattern
- Procedure -> Procedure handoff

を説明でき、actual Repository testがgreenなら、production Canonical contractのADR候補を作る価値が出る。

その場合もWorkflow Engine導入は別Issue / 別Evidenceとする。

---

# Part 2 — Handoff Canonical Necessity Audit（H0 vs H1、H1採用）

## 1. 問い

Business Procedure間handoffをcurrent Referenceでaccepted Business meaningとして扱う場合、そのrelationをAuthoring / Reference setだけへ残し、normalized / Canonical meaningから完全に落としてよいか。

ここで問うのは専用`BusinessHandoff` kindの採用ではない。

```text
H0: accepted handoffはauthoring-only
H1: accepted handoffの薄いnormalized meaningを保持
```

を比較する。

## 2. Requirement pressure

RQ-025解釈ではBusiness Procedureが必要に応じて次を説明する。

```text
purpose
trigger / event
actor / job context
Business Action
manual / external work
Application Function usage
handoff
completion / continuation
```

PAS-07 supporting contractでも、少なくとも次がAcceptance pressureとして置かれている。

```text
P25-4
  handoff != Runtime Workflow

P26-4
  reuse / handoff endpointをlogical keyで追跡
```

したがってcurrent Referenceに明示されたhandoffを単なるgraph decorationとみなす根拠は弱い。

## 3. H0 — authoring-only handoff

```text
Canonical / normalized Procedure bodies
  order-entry.order-handling
  order-entry.follow-up

Authoring-only relation
  order-entry.order-handling/BP-09
    -> order-entry.follow-up/FU-00
    + condition
```

### 問題

handoff conditionだけが変更された場合、Procedure bodyは変わらない。

```text
before body == after body
```

同様に、handoff endpointだけを`BP-09 -> BP-08`へ変更してもProcedure bodyは変わらない。

Canonical / normalized側にbodyしかなければ、accepted handoff relationの意味変更をsemantic diff / impact / Conformanceから観測できない。

これはidentity-only Procedure shellがBusiness-only wording changeを失った問題と同型である。

## 4. H1 — thin normalized handoff meaning

最小Pressure形:

```text
Business handoff meaning
  from
    Procedure address
    Step local address
  to
    Procedure address
    Step local address
  condition
```

### 保持できる差

- condition change
- from endpoint change
- to endpoint change
- stale endpoint
- duplicate relation

### 入れないもの

- source path
- fingerprint
- repository revision
- runtime instance
- queue
- retry
- timer
- scheduler
- saga
- execution ownership
- transport
- Authorization identity

H1はWorkflow Engineではない。

## 5. test-only反証

`tools/reference-business-procedure-check/handoff_canonical_necessity_pressure_test.go`で次をPressureする。

1. handoff conditionだけを変更してもProcedure bodiesは不変
2. body-only snapshotではrelation changeを観測できない
3. normalized handoff projectionを含めるとcondition changeを観測
4. endpoint変更もProcedure bodyを変えずreferential relation差として観測

既存`canonical_handoff_pressure_test.go`と合わせ、locator moveではhandoff meaningを変えず、missing endpointはfail closedする。

actual Repository runnerでgreenになるまでは実行Evidenceとは数えない。

## 6. Source Indexとの境界

H1を採る場合、Canonical semantic validationがSource Indexへ依存しない現行architectureを守る必要がある。

したがってhandoff endpointがStepを指すなら、Step addressabilityをSource Indexだけに置くS4-Bは不利になる。

```text
H1 + Canonical validation
  -> S4-A pressure
     normalized Procedure-scoped Step local address metadata
```

ただしこれはglobal Step IDを要求しない。

## 7. 専用kindはまだ不要

H1から直ちに次を導かない。

```text
ir.World.BusinessHandoffs []BusinessHandoff
```

必要なのは「accepted handoff meaningをnormalized semantic layerが失わないこと」である。

実装候補は別に比較できる。

- thin dedicated relation
- Procedure-related generic Relation
- Business semantic edge collection
- 将来のBusiness Flow modelの一部

既存Micro / Semantic Kernelへ専用kindを増やす負担はADR前に比較する。

## Part 2 現時点の判断

current `order_entry` ReferenceではhandoffをBusiness authoring meaningとして明示し、RQ-025 / PAS-07もhandoffを業務手順のAcceptance pressureとして扱っている。

そのためpre-ADR判断は次とする。

```text
H0 authoring-only handoff
  -> accepted semantic changeをnormalized layerから失うため不十分な可能性が高い

H1 thin normalized handoff meaning
  -> primary candidate
```

ただしproduction dedicated kind採用は未決定。

## 9. 次のPressure（Part 2）

H1を前提候補として、次を確認する。

1. S4-AのProcedure-scoped local Step addressだけでreferential integrityを閉じられるか
2. handoff relationをgeneric semantic relationで表せるか、専用Business relationが必要か
3. handoff conditionを自然言語meaningのまま保持して十分か
4. Business Flow / DFD型flowとProcedure handoffを同一objectへ畳まないか
5. 000039 semantic diff / impactがH1を扱える粒度を持てるか

## 10. 非対象（Part 2）

- production Canonical kind採用
- Workflow Engine
- executable branch / predicate DSL
- runtime event bus
- retry / queue / scheduler
- Business Flow全体のmodel確定
- 000039実装

---

# Part 3 — Handoff Representation Option Audit（R1〜R4比較、R3採用）

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

## 12. test-only Evidence（Part 3）

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

## 13. ADR前にまだ決めないもの（Part 3）

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

---

# Part 4 — Handoff vs Business Flow Pressure（R3とBusiness Flowの境界を外部Evidenceで補強）

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

## Part 4（全体の現時点）判断

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
