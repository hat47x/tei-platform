# Lane B — Business Procedure Canonical Trace Target Audit — 2026-09

Status: **supporting architecture-pressure audit / no ADR / no production Canonical adoption**

Related:

- `requirements/TEI_PRODUCT_REQUIREMENTS.md` — RQ-001 / RQ-002 / RQ-003 / RQ-025 / RQ-026
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md` — PAS-01 / PAS-07
- `plan/RQ025_RQ026_ACCEPTANCE_TRACE.md`
- `plan/LANE_B_RQ025_RQ026_TRACE_AUDIT_2026-09.md`
- Lane C PR #482 / `plan/issue/000042-business-procedure-cross-view-reference.md`

## 1. 目的

Lane CのBusiness Procedure pressureが、Reference-local identityからproduction Canonical contract候補の比較へ進んだため、Lane BからRQ-002 / Subject / Source Index / PAS-01 / PAS-07 / ISSUE-000039への波及を横断監査する。

本書はPAS-07の第二正本ではない。

Business Procedure fixture / checker / Canonical probeも所有しない。

production `ir.World`、Canonical schema、ADR、Gateを変更しない。

## 2. 現行contractの事実

### RQ-002

RQ-002はauthoring sourceから「どのCanonical subjectへ寄与したか」を追跡することを要求している。

source locator / fingerprint / range / roleはCanonical Worldへ無条件に混ぜず、source-side metadataとして扱う。

### `subject.Subject`

現行`internal/subject.Subject`の契約は明示的に:

```text
path-independent semantic or logical target inside a TEI World
```

である。

したがって`Subject{Kind, ID}`というstruct shapeが小さいことと、World外のaccepted authoring targetをSubjectとして扱ってよいことは同じではない。

### Source Index

`internal/sourceindex`は`subject.Subject`をaliasし、Artifact / Reference / Range / Roleを扱う。

Source Index validationは主として:

- artifact path normalization
- fingerprint整合
- referenced artifact existence
- byte range / line-column freshness

を検査する。

Subjectが実際に`ir.World`へ存在するかをSource Index単独では再検証しない。

`workspace.BuildSourceIndex()`には`SourceContributions`からSubjectを流し込める技術的拡張口もある。

したがって:

```text
Source Indexがstructとして受理できる
  !=
current Subject contract上のCanonical Subjectである
  !=
RQ-002を満たす
```

を固定する。

### `ir.World`

現行`ir.World`にはSemantics / Applications / Databases / Persistence / Releases / Environments / Queries / Bindings等があるが、Business Procedure fieldは存在しない。

したがってBusiness ProcedureをWorld内SubjectへするOptionは、identity-onlyであっても実際にはWorld / loader / normalize / validation / serialization / conformance等へ波及し得る。

## 3. B-F10 — Source Indexの機械的受理とCanonical traceを混同する危険

### 判定

Confirmed semantic-contract distinction.

### 発見

Lane CのSource Index pressureは現在、Business Procedure artifactから既存Canonical `application-pattern` Subjectへのderived referenceやArtifact fingerprint / file move性質を確認している。

これは有効なEvidenceだが、Business Procedure自身をCanonical Subjectとして確立したEvidenceではない。

また将来`SourceContributions`へProcedure keyを入れれば、Source Indexのstruct / validationを機械的に通せる可能性がある。

しかし現行`subject.Subject`契約はWorld内targetである。

### 結論

Option B（Subject contract拡張）は「Source IndexへKindを1個足すだけ」の変更ではない。

少なくとも次を明示的に再判断する必要がある。

- `Subject`はCanonical World targetだけか
- accepted file-native design targetもSubjectか
- RQ-002の`Canonical subject`をどう読むか
- Source Index利用者が`Subjectにある = Canonical`と誤読しない契約

この再定義なしにWorld外Procedure keyをproduction Subjectとして流し込まない。

## 4. B-F11 — identity-onlyではBusiness semantic changeを保持できない

### 判定

Confirmed pressure against Option A as a sufficient contract.

### 発見

Reference-local Procedure keyだけで次の機械的安全性はpressureできている。

```text
Preview / Reject
stale detection
Accept / reload
file move / logical rename separation
review / handoff target continuity
```

しかし同じProcedure IDのまま:

```text
purpose
trigger
actor
manual Business Action
ordered Business Action
```

がaccepted changeした場合、ID-only Canonical shellは不変である。

したがってID-onlyではBusiness-only Semantic Diffを表せない。

### Lane B判定

Option Aはidentity primitiveとして有用であっても、RQ-001 / RQ-002 / RQ-026を現行architectureの`Canonical meaning`基線へ接続する最小十分contractとはまだ言えない。

`Shared Acceptance targetをpathから分離できる`ことと、`accepted Business meaningをCanonical側で保持できる`ことを分ける。

## 5. B-F12 — thin normalized Business residual meaningの候補性

### 判定

Evidence-supported candidate pressure; not adopted.

Lane Cは現在、Option Dとしてfull Workflowではない薄いnormalized Business meaningをtest-onlyで比較している。

現在のcandidate coreは概ね:

```text
BusinessProcedure
  id
  purpose
  trigger
    epistemic status
    meaning
  actor
    epistemic status
    business label
  ordered Business Actions
  explicit semantic refs
```

を中心にしている。

### この候補を支持するpressure

- Business-only wording changeをmeaning changeとして観測できる
- pure file moveはmeaning no-op
- manual Business Actionをfake Functionなしで保持できる
- Pattern-owned Data / Relation / Query / Capability / DatasourceをBusiness側へコピーしない
- explicit Validation等、Business側から必要なdirect refだけを保持できる
- trigger / actorのcandidate / unresolved等の不確定性を既知へ丸めない
- 一つのProcedureを単一Application ownerへ固定せず、qualified Pattern refで複数Applicationを跨げる

### 重要な非採用

これらは`test-only pressure`である。

actual Repository runnerが実行されておらず、production `ir.World`へは未導入である。

Lane BはOption DをPAS-07から自動採用しない。

## 6. Canonical candidateからさらに引けるもの

Lane Cのsubtraction pressureは、thin candidate自体もさらに縮約できる可能性を示している。

### `realization`

Referenceの`application-use` / `manual-unresolved` / `security-boundary`等はreviewには有用だが、現時点ではCore enumとして必須ではない。

`realization`を外しても:

- manual Business Action wording
- Function refsがないこと
- explicit Pattern / Validation refs

を保持できるpressureがある。

したがってproduction候補へ`realization` enumを先に固定しない。

### Step ID

Procedure bodyのordered Business meaningだけなら、Reference-local Step keyを外してもmeaningを再構成できるpressureがある。

一方、handoff endpointは`Procedure key + Step key`でaddressabilityを必要としている。

したがって:

```text
Step body meaning
  !=
Step addressability / relation endpoint identity
```

を分けて検討する。

current pressureでは、Step bodyからIDを外しつつ、normalized handoff endpointのためにProcedure-scoped local address metadataを保持するS4-Aが有力である。

local keyはglobal Canonical Step identityでもRuntime execution identityでもない。

## 7. Business Handoffを独立relationとして扱うpressure

Lane CはhandoffをProcedure body内のruntime control flowへ埋め込まず、test-onlyで:

```text
Procedure/Step
  -> Procedure/Step
  + natural-language condition
```

という薄いBusiness semantic relationとしてpressureしている。

観測している性質:

- artifact locator変更でhandoff meaningは変わらない
- condition変更はBusiness semantic change
- missing endpointはfail closed
- duplicate relationはreject
- endpointへruntime execution / token / instance / stateを入れない

これはX-TEAでBusiness definitionとBusiness Flowを分けてcross-referenceする設計原点とも整合する。

### H0 / H1 necessity pressure

current Referenceでhandoffをaccepted Business meaningとして扱う場合:

```text
H0  handoff authoring-only
  -> condition / endpointだけの変更をnormalized semantic diffから失う

H1  thin normalized handoff meaning
  -> from / to / conditionをnormalized Business meaningとして保持
```

となる。

したがってH1がcurrent primary candidate pressureである。

ただしproduction `BusinessHandoff`専用kindの採用とは別問題である。

### Representation R1〜R4

Lane Cの比較は次である。

```text
R1  existing semantic.Data.Relation reuse
    -> reject

R2  generic Core Edge / Relation
    -> defer / reject for now

R3  Business semantic boundaryのthin relation
    -> primary pre-ADR candidate

R4  future Business Flowへ統合
    -> later integration concern
```

R1はData RelationとBusiness handoffの意味を混ぜるため不適切である。

R2はhandoff一件のために万能edgeをMicro Kernelへ持ち込むため過剰である。

R4について、DFD型Business FlowはEvent / Business Activity / information source / external party / flowを含みhandoffより広い。Business Flow全体を今実装する必要はない一方、将来接続できる余地を残す。

このためcurrent pressureではR3 + S4-Aが最も説明力を持つ。

### test-only normalized model

```text
NormalizedBusinessModel
  Procedures[]
    id / purpose / trigger / actor
    ordered Steps[]
      localKey
      body
        businessAction
        explicit semantic refs

  Handoffs[]
    from Procedure/localKey
    to   Procedure/localKey
    condition
```

normalize後のreferential validationをSource Index / file path / fingerprint / Workspaceへ戻らず、このmodelだけで閉じられるかをpressureする。

production `ir.World.Business`や専用Go type名は未採用である。

## 8. B-F17 — handoff normalization necessityと専用kind / Business Flow adoptionを混同しない

### 判定

Confirmed semantic-preservation pressure; representation still unresolved.

### 発見

H0への反証から得られるのは:

```text
accepted handoff meaningをnormalized semantic layerから失わない必要
```

であって:

```text
専用production BusinessHandoff kindが必要
Business Flow全体を今model化する必要
```

ではない。

### Lane B判定

次を分離する。

```text
semantic necessity
  handoff relation-only changeをnormalized diff / impactから失わない

representation choice
  R3 thin Business relation
  future Business Flow integration
  production package / field / type
```

したがってLane BはH1 / R3 / S4-Aをcurrent strongest pressureとして認識するが、ADR採用とはしない。

## 9. B-F18 — Reference Application contextをsemantic prerequisiteへ戻さない

### 判定

Confirmed subtraction result at implementation/specification level; actual runner unresolved.

Lane Cではfixture-level `Application`をoptional reporting contextへ縮退し、空またはWorldに存在しないlabelでもexplicit Pattern refsが解決できる限りcross-view evaluationをblockしないpressureへ進んでいる。

```text
fixture Application
  -> optional reporting / authoring context
  -> semantic ownerではない
  -> datasource policy authorityではない

Pattern owner Application
  -> normalized Worldからexact解決
  -> realization / datasource policy authority
```

残る論点はfieldをreporting用途で残すか完全除去するかというrepresentation compatibilityであり、Business semantic prerequisiteかどうかではない。

## 10. B-F13 — ISSUE-000039再利用をCanonical採用理由へ先取りしない

### 判定

Confirmed scope distinction.

ISSUE-000039の初期supported kindsは:

1. Canonical Validation
2. Application + Capability identityで識別されるBinding

である。

依存はCanonical参照から導出する計画だが、Business Procedureを初期change-graph nodeとして必須にはしていない。

したがって:

```text
Option D / R3なら将来000039へ接続しやすい
```

は利点であって、

```text
000039があるから今Canonical Business Procedure / Handoffが必須
```

という採用理由にはしない。

将来、Business-only changeからData / Function / Scenarioへimpactを伝播するProduct Valueを実装するとき、file-native key /別normalized modelでは安全に扱えないEvidenceが出た場合にreopenする。

## 11. Option A / B / C / DのLane B比較

| Option | 現行contractとの整合 | 主な強み | 主な未解決 / risk | Lane B位置づけ |
| --- | --- | --- | --- | --- |
| A: identity shell | `Subject inside World`と整合しやすい | path-independent target / duplicate reject | Business-only meaning changeを表せない | identity primitive候補、単独採用保留 |
| B: Subject拡張 | current Subject contractを変更 | World field追加を避ける | RQ-002再解釈、Business meaning正規化は別問題 | 概念blast radius大、先行しない |
| C: Authoring registry | Worldを保てる | review / handoff locator管理 | Canonical link未解決、三層化 | 反復Evidenceなしに増やさない |
| D: thin Canonical Business meaning | current Canonical architectureと最も直接整合 | Business-only diff / cross-view / Source trace | Core surface追加、field creep / Workflow化の危険 | test-only第一比較候補、未採用 |

重要:

Option Dが現時点で相対的に強く見えることと、production採用決定は同じではない。

Dの内部でも現在は、thin Procedure body + Procedure-scoped local address + thin handoff relationというD3 / H1 / R3 / S4-Aの組合せを主pressureとしている。

## 12. D候補へ要求するhard guard

production ADR候補へ進む前に、少なくとも次を守る。

### Authority

- file-native Business sourceをauthoritative inputとして保つ
- normalized Canonical meaningは同じsourceから決定論的に生成する
- derived Matrix / diagnosticsから逆書きしない
- Pattern-owned meaningをProcedureへ複製しない

### Meaning

- purpose / trigger / actor / Business Actionのaccepted changeをlosslessに扱う
- explicit uncertaintyを既知へ丸めない
- ordered Business Actionの順序をset sortしない
- set-like semantic refsだけをdeterministically normalizeする
- accepted handoff endpoint / condition changeをnormalized layerから失わない

### Boundary

- Business ActorからAuthorization Roleを生成しない
- Business Procedure / Handoffからruntime Workflow / retry / timer / queue / sagaを生成しない
- single Application ownerを必須にしない
- optional reporting Application contextをsemantic prerequisiteにしない
- source path / fingerprint / revisionをCanonical Business meaningへ混ぜない
- `realization` review vocabularyをCore enumへ先取りしない
- Step addressabilityをglobal / runtime identityへ昇格しない
- Data `semantic.Relation`へBusiness handoffを押し込まない
- generic Core Edgeをhandoffのためだけに導入しない
- Business Flow全体をhandoffのためだけに先取り実装しない

### Evidence

- actual Repository runner green
- pure file move invariance
- logical Procedure rename fail-closed
- Business-only semantic diff
- multiple Application pressure
- optional / unknown reporting Application context pressure
- manual / external pressure
- H0 semantic-loss / H1 preservation pressure
- normalized-only handoff referential validation
- handoff endpoint / condition pressure
- realistic Referenceがcurrent `order_entry`より一段広いこと

## 13. PAS-01 / PAS-07との関係

### PAS-07

PAS-07の利用者価値は、Business meaningをFunctionへ還元せず三視点から矛盾を見つけ、authoritative sourceへ戻して修正することである。

Option D / H1 / R3 / S4-AはこのValueを実装する候補だが、PAS-07の定義そのものではない。

したがってPAS-07主Traceはtechnology-neutralのまま維持する。

### PAS-01

Business-only changeをShared Acceptanceへ通す場合、candidate target identityだけでなく、before / afterのaccepted Business meaning差を扱える必要がある。

Lane Cのfile-native helperは機械的Acceptance pressureを示すが、production PAS-01 Shared Acceptanceは未完了である。

Option D等が採用される場合でも、PAS-01 actual Shared Acceptance Evidenceを自動代替しない。

## 14. Gate / Lane境界

- Gate B Required Evidence 3 lane: 変更しない
- Gate C First Web Target Evidence: 変更しない
- PAS-07 Reference Evidence: parallel work
- Lane C #482: Reference / checker / Canonical pressureの正本
- Lane B: Requirement / PAS / Evidence claim / cross-lane contract監査
- Architecture / ADR: production Canonical contract採否

Lane BからCanonical schemaや新Gateを追加しない。

## 15. Reopen / decision conditions

Architecture decisionへ進めるのは、少なくとも次が揃ったときとする。

1. Lane C test-only pressureのactual green
2. thin candidateでBusiness-only diffが成立
3. Pattern-owned meaning非重複が維持
4. realization subtraction後も必要Business meaningが維持
5. Step body / local addressabilityの責務が説明可能
6. H0 semantic-loss / H1 normalized preservationをactual Evidenceで確認
7. R3 + S4-A normalized modelだけでhandoff integrityを検証可能
8. handoffをruntime Workflow化せず扱える
9. multi-Application / manual / external pressureが成立
10. optional / unknown reporting Application contextでcross-view evaluationが成立
11. source / Canonical / Projection authority境界が明確
12. current Referenceだけへの過適合でないと説明できる
13. realistic cross-subject-area / multi-subsystem Referenceで設計価値を確認

その時点でLane Bは、ADR候補がRQ-001 / RQ-002 / RQ-003 / RQ-025 / RQ-026とPAS-01 / PAS-07を同時に満たすか再監査する。

## 16. 現時点の結論

```text
Reference-local Procedure key
  useful and necessary for current authoring/review/handoff pressure

ID-only Canonical Procedure shell
  not sufficient for Business-only semantic meaning

World外Subjectの機械的利用
  not sufficient for RQ-002 under current Subject contract

thin normalized Business residual meaning
  strongest current production candidate pressure
  but not adopted

accepted handoff normalized meaning
  necessary pressure is strengthening
  H1 / R3 / S4-A is current strongest representation pressure
  dedicated production kind not adopted

Business Flow
  broader than Procedure handoff
  do not implement whole flow merely to preserve current handoff

fixture Application context
  optional reporting context candidate
  not a semantic prerequisite / owner / policy authority

production Canonical Procedure / Handoff / BusinessModel
  unresolved
```

actual execution Evidenceが得られるまでは、PR #482 Draft / ISSUE-000042 In ProgressというLane C判定を尊重する。
