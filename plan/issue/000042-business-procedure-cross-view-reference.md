# 000042 Business Procedure cross-view Reference

Status: In Progress

## 目的

`order_entry`を起点にData / Function-Interaction / Businessの三視点cross-checkをReferenceとしてPressureし、Business Procedureの最小production Canonical contractをEvidenceから決める。

先にWorkflow Engine、full Procedure schema、global Step identity、full Business Flow schema、generic semantic ref bag / role taxonomyを固定しない。

## 正式基線

- `requirements/TEI_PRODUCT_REQUIREMENTS.md` — RQ-025 / RQ-026
- `requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`
- `docs/architecture.md`
- `docs/business-system-design.md`
- `plan/LANE_B_RQ002_BUSINESS_PROCEDURE_CANONICAL_TRACE_AUDIT_2026-09.md`
- `plan/LANE_B_RQ025_RQ026_TRACE_AUDIT_2026-09.md`
- `plan/RQ025_RQ026_ACCEPTANCE_TRACE.md`
- `plan/BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md`
- `plan/BUSINESS_PROCEDURE_ADR_VS_PRODUCTION_ADOPTION_READINESS_SPLIT_2026-09.md`
- `plan/BUSINESS_AUTHORING_PRODUCTION_2026-09.md`（Part 2、旧`PRODUCTION_ADOPTION_EVIDENCE_CONTRACT`）
- `plan/BUSINESS_PROCEDURE_MAIN_BASELINE_REAUDIT_2026-09.md`
- `plan/BUSINESS_PROCEDURE_EXPLICIT_REF_AUTHORITY_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_D4_REF_KIND_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_APPLICATION_PATTERN_TARGET_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_D4_APPLICATION_PATTERN_TARGET_ROLE_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_O1_AUTHORITY_DIRECTION_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_MANUFACTURING_INVENTORY_CORPUS_PRESSURE_OUTLINE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_SECOND_CORPUS_CONTRACT_PRESSURE_AUDIT_2026-09.md`
- `plan/BUSINESS_PROCEDURE_PHYSICAL_OBSERVATION_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_SECOND_CORPUS_DATA_AUTHORITY_DEFERRAL_2026-09.md`
- `plan/BUSINESS_PROCEDURE_SECOND_CORPUS_HANDOFF_INFERENCE_PRESSURE_2026-09.md`
- `plan/BUSINESS_ACTION_DATA_PROJECTION_PRESSURE_2026-09.md`
- `plan/BUSINESS_ACTION_CRUD_PROJECTION_PRESSURE_2026-09.md`
- `plan/BUSINESS_ACTION_CRUD_OPERATION_GAP_AUDIT_2026-09.md`
- `plan/BUSINESS_AUTHORING_AUTHORITY_CUTOVER_PRESSURE_2026-09.md`
- `plan/BUSINESS_AUTHORING_PRODUCTION_2026-09.md`（Part 1、旧`PRODUCTION_LOADER_VERSIONING_OPTION_AUDIT`）
- `plan/BUSINESS_AUTHORING_WORKSPACE_DISCOVERY_BOUNDARY_PRESSURE_2026-09.md`
- `plan/BUSINESS_AUTHORING_LOGICAL_SOURCE_IDENTITY_PRESSURE_2026-09.md`
- `plan/BUSINESS_AUTHORING_EXISTING_ACCEPTANCE_IMPLEMENTATION_REUSE_AUDIT_2026-09.md`
- `plan/BUSINESS_MODEL_PRODUCTION_ACCEPTANCE_BOUNDARY_AUDIT_2026-09.md`
- `plan/BUSINESS_MODEL_PHYSICAL_BASE_REVISION_OPTION_AUDIT_2026-09.md`
- `plan/BUSINESS_HANDOFF_STANDALONE_AUTHORING_PRESSURE_2026-09.md`
- `plan/BUSINESS_HANDOFF_WATANABE_SECOND_REALISTIC_EVIDENCE_PRESSURE_2026-09.md`
- `plan/BUSINESS_HANDOFF_TEI_ORIGINAL_SECOND_CASE_SEARCH_2026-09.md`

`plan/XTEA_DESIGN_EVIDENCE_REGISTER_2026-09.md`は現在、公開横断の詳細Evidence正本ではなくExternal Design Reference Noteである。X-TEA / XEAD / 渡辺幸三氏に関する詳細corroborationは、Business Flow / Handoff / O1 / CRUD等のquestion-specific Lane C auditへ近接保持する。

## Lane境界

本IssueはLane CのBusiness Procedure Reference / Evidenceを所有する。RQ-025 / RQ-026はProduct Requirementとして採用済み。PAS-07 Acceptance trace正本はLane B側へ残す。

本Issueの**Lane C Pre-ADR Gate C**はHandoff/address auditであり、`GATE_C_EVIDENCE_CONTRACT.md`の**Product Gate C**とは別物である。Product Gate B / Gate C scope、Business Lifecycle 000038、共通change graph 000039を本Issueから拡張しない。

## 固定する境界

```text
Business Action != Application Function
Business Flow != Procedure != Runtime Workflow Orchestration
Business Handoff != full Business Flow / DFD
Business Actor / Job != Authentication Principal != Authorization Role / Policy
Business Procedure != single Application child
Runtime Observation operation != Business / Semantic identity
Command accepted != physical-world fact observed
physical-world fact observed != accepted responsibility Handoff
Step order / adjacency != accepted responsibility Handoff
Application Function sequence != accepted responsibility Handoff
Data trace projection != complete Data dependency authority
no current Data trace != no Data dependency
semantic Data trace != persistence CRUD operation trace
no current CRUD trace != no persistence operation
Database write authority != Business Action CRUD attribution
Business Lifecycle transition != persistence CRUD U
ref kind separation != ref role separation
Data / Function-Interaction / Business = 同じaccepted meaningを相互牽制する視点
Business source freshness != Business semantic revision != cross-view Conformance base
file exists != authoritative source reachable
physical source path != Business logical identity != source authority
exact duplicate H1/R3 meaning across managed sources = duplicate authority
Handoff endpoint pair != implicit Handoff identity
global Handoff ID necessity != proven
schemaVersion != source authority priority
review / impact projection != authoritative source
Deployment Run.Revision != Business semantic revision
existing primitive != Shared Acceptance authority
Lane C Pre-ADR Gate C != Product Gate C
Contract ADR readiness != Production adoption readiness
qualifying C/U/D case absence != Business Procedure Contract ADR blocker
no ir.World.Business necessity Evidence != placement blocker
```

Business ProcedureにはApplication / Semantic Worldから機械的に導出できないresidual meaningがある。

```text
purpose
trigger / event + epistemic status
actor / job context + epistemic status
manual / external / unresolved Business Action
judgment / continuation
handoff
```

Pattern-owned Data / Relation / Query / Capability / Datasource / Application ownership / datasource policyはnormalized `ir.World`からderiveし、Business sourceへコピーしない。

## Primary Reference

`reference/business-system-design/order-entry-business-procedure.json`

- BP-00〜BP-09
- BP-04 / BP-09はPatternなしのmanual / external Business Action
- BP-05だけがaccepted deterministic Validation authorityをdirect refとして保持
- BP-06は`security.authorize` Capabilityをdirect refとして保持
- BP-07は`order_entry.submit_order`をdirect Function realizationとして利用
- BP-08はObservation intentionをBusiness wordingとして保持しruntime Observation identityをfake Semantic refへ昇格しない

Follow-up / Handoff / intentional defect:

- `order-entry-follow-up-procedure.json`
- `order-entry-procedure-reference-set.json`
- `order-entry-customer-edit-conflict.json`

## Gate 15 Reference / Evidence分類

### 15a — multi-Application structural pressure
Synthetic second Applicationで`Business Procedure != single Application child`を確認する。

### 15b — social-system realism pressure
Current `order_entry`だけでmanaged-here sales、external-observed master、human judgment、Authorization、Command、Observation intention、manual/external continuationを同一Business Procedure文脈に保持する。

### 15c — conditional broader realistic pressure
15a + 15bでminimum contract sufficiencyを説明できない具体的不足が出た場合だけbroader realistic pressureを追加する。Application数を成熟度KPIにしない。

Manufacturing / inventory second-corpus outlineではphysical-world `Command != actual completion`をtest-only Pressureまで進めた。H1/R3についてはStep adjacencyやpositive physical ObservationからHandoffを生成せず、explicit Handoff sourceだけをnormalized relationのauthorityとする。一方、exact TEI Data authority、O1 protected-target roleのsecond-domain反復、new production contract不足はまだ証明されない。Gate 15cは**未起動**を維持する。

Current TEI-original Repositoryにはindependent second realistic Handoffを安全に昇格できる既存caseがない。adjacency / intentional defect / Lifecycle / DeploymentをHandoffへ読み替えず、**Lane C Pre-ADR Gate C**のsecond realistic Handoff blockerは未充足として残す。

## Cross-view checker原則

- explicit referenceはexact identityで解決
- Evidence-backedなBusiness dependencyだけを保持
- 名称類似で補完しない
- Pattern ID文字列からownerを推測しない
- Pattern + owner Applicationをnormalized Worldからexact解決
- Projectionだけでaccepted authorityを拡張しない
- Business Action × Data projectionはevidence pathだけを表示し、complete dependencyを主張しない
- current refsからData traceがない行を`No Data`へ丸めない
- Business Action × CRUDはoperation authorityまで辿れたcellだけを埋める
- Data identityが見えるだけでCRUD letterを付与しない
- `record` / `command` Pattern kindだけからCRUDを推測しない
- Database Contract writeをBusiness Actionへ暗黙帰属しない
- Lifecycle transitionをCRUD Uへ短絡しない
- `realization` labelからref kind / role / CRUD effectを推測しない
- runtime Observation APIをSemantic identityへ昇格しない
- Command acceptanceからphysical completionを推測しない
- physical ObservationからHandoff endpoint / conditionを生成しない
- Step order / adjacencyからHandoffを生成しない
- Application Function sequenceからHandoffを生成しない
- exact TEI Data / Pattern identityなしにsecond-corpus ownershipをsyntheticに作らない
- adjacencyからprotected targetを推測しない
- Authorization policyはprotected targetをcorroborateできるが、Business targetをauthor / backfillしない
- Procedure actorをHandoff endpoint identityへ複製しない
- Procedure triggerとHandoff conditionを暗黙同期しない
- legacy + candidateをpermanent dual authorityとしてmergeしない
- source pathをBusiness semantic identityへしない
- duplicate logical Procedure sourceをpath順 / input順 / first-writer / last-writerで解決しない
- exact duplicate standalone H1/R3 meaningを複数managed sourceから受理しない
- same Handoff endpoint pairだけを暗黙identityとして発明しない
- Git HEADだけを無条件なfreshness proofとしない
- Workbench / impact projectionをauthoritative sourceへ昇格しない
- Deployment revision / fingerprint機構をBusiness semantic revision / source transactionへ読み替えない
- implementation gapをBusiness schema field追加のEvidenceへしない
- Production implementation未完成をContract ADR blockerへ自動昇格しない
- C/U/D authority不足をBusiness Procedure CRUD field必要性へ読み替えない
- `ir.World.Business`必要性EvidenceがないことをNot Ready理由にしない

## Acceptance / Evidence Pressure matrix

このmatrixはLane Cの**Evidence / Pressure inventory**であり、独立したReady / Not Ready判定を所有しない。Contract ADR readinessの唯一の正本は`BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md`とする。

| ID | Pressure | 現在の扱い |
| --- | --- | --- |
| A1 | exact semantic / Pattern reference resolution | test-only実装済み |
| A2 | manual / external actionにfake Functionを要求しない | 実装済み |
| A3 | Pattern-owned meaningをBusiness sourceへ複製しない | 実装済み |
| A4 | Patternから導けないexplicit refsだけBusiness側に保持 | 実装済み |
| A5 | intentional datasource access conflict | 実装済み |
| A6 | baseline clean | actual runner待ち |
| A7 | unknown field / schemaVersion等fail closed | 実装済み |
| A8 | purpose / trigger / actor / manual actionはBusiness residual | 実装済み |
| A9 | Business-only changeとPattern meaning changeを分離 | 実装済み |
| A10 | multiple Procedure -> same Pattern | 実装済み |
| A11 | Procedure keyをfile pathから分離 | 実装済み |
| A12 | pure file move = semantic no-op候補 | 実装済み |
| A13 | logical Procedure renameはstale link fail closed | 実装済み |
| A14 | review target continuity | 実装済み |
| A15 | approvalはrevision / fingerprint bound | 実装済み |
| A16 | file-native Procedure source pressure | 実装済み |
| A17 | Shared Acceptance target / stale pressure | test-only実装済み |
| A18 | identity-only shell | 比較用、最小十分とはみなさない |
| A19 | thin Business meaning | test-only実装済み |
| A20 | Canonical candidateでもPattern meaning非重複 | 実装済み |
| A21 | manual actionをthin candidateへ保持 | 実装済み |
| A22 | handoffをthin Business relationとして表現 | test-only実装済み |
| A23 | handoff condition change = semantic change | 実装済み |
| A24 | runtime Workflow detailを侵入させない | 実装済み |
| A25 | one Procedure -> multiple Applications | 実装済み |
| A26 | multi-Application PressureをGate B realizationへ結合しない | 実装済み |
| A27 | trigger / actor epistemic statusをlossless保持 | 実装済み |
| A28 | uncertainty change = Business semantic change | 実装済み |
| A29 | actor uncertainty != Authorization change | 実装済み |
| A30 | uncertainty vocabularyを早期固定しない | 方針固定 |
| A31 | realization subtraction | test-only実装済み |
| A32 | realization-only label changeをCore changeへ自動昇格しない | 実装済み |
| A33 | manual action survives realization subtraction | 実装済み |
| A34 | Step body meaning can be ID-free | test-only実装済み |
| A35 | handoffはexact Step addressabilityを要求 | 実装済み |
| A36 | Step addressability != runtime identity | 方針固定 |
| A37 | multi-App + manual + external coexistence | Gate 15a構造Pressureとして実装済み |
| A38 | Application associationはWorld解決でderive | 実装済み |
| A39 | access policyはPattern owner Applicationを使う | checker修正 + 回帰Pressure実装済み |
| A40 | fixture Application context != Procedure owner | 実装済み |
| A41 | normalized handoffはStep address解決をSource Index依存にしない | S4-A優先Pressure |
| A42 | Step local key renameをBodyMeaningChangedと分離 | test-only実装済み |
| A43 | global Canonical Step IDをdefault導入しない | 方針固定 |
| A44 | primary Application contextをsemantic prerequisiteからsubtract | C1実装済み、actual runner待ち |
| A45 | H0ではhandoff changeをnormalized diffから失う | 反証Pressure実装済み |
| A46 | H1でaccepted handoff relation changeを保持 | primary candidate / test-only |
| A47 | Data Relation流用 / generic Core Edge新設を避ける | R1 reject / R2 defer |
| A48 | Business-side thin relation R3 | primary pre-ADR candidate |
| A49 | R3 + S4-AをSource Index非依存validate | test-only実装済み |
| A50 | Business Flowはthin Handoffより広く、外部corroborationはquestion-specific Business Flow / Handoff auditへ保持する | Evidence責務固定 / no cross-register authority |
| A51 | R3をhandoff-only relationへ限定 | negative Pressure実装済み |
| A52 | handoff PASSをBusiness Flow / DFD Conformanceと呼ばない | 境界固定 |
| A53 | full Business Flowは別pressure時だけreopen | 方針固定 |
| A54 | `realization`変更でcross-view diagnostics不変 | test-only実装済み |
| A55 | BP-06 Authorizationは`security.authorize`へexplicit trace | 実装済み |
| A56 | BP-08 Observationはruntime read identityをSemantic refへしない | negative Pressure実装済み |
| A57 | explicit refs are evidence-driven, not symmetry-driven | 方針固定 |
| A58 | BP-03 line reviewとBP-05 Validation authorityを分離 | Reference修正 + test-only実装済み |
| A59 | Pattern-owned CapabilityをBusiness sourceへ重複しない | negative Pressure実装済み |
| A60 | current Referenceでdirect Data / Relation ref minimum必要性未証明 | Evidence固定 |
| A61 | D4 bodyをPattern / Validation / Capability refsへ縮小 | test-only実装済み |
| A62 | future direct Data / Relation refをD4がfail closed / reopen | test-only実装済み |
| A63 | ref authority kindをkind-less bagへ畳まない | test-only実装済み |
| A64 | D4 bodyとS4-A addressabilityを分離 | test-only構造固定 |
| A65 | D4 + S4-A + H1/R3 composition、normalized duplicate Handoffもfail closed | test-only実装済み |
| A66 | Application Pattern ref kindとref roleを分離 | O1 source-level Pressure実装済み |
| A67 | BP-06 protected target != direct Function realization | test-only実装済み |
| A68 | protected targetをexact resolutionしadjacencyからderiveしない | test-only実装済み |
| A69 | D4 + O1 target-role composition、generic Role taxonomyはdefer | test-only実装済み |
| A70 | Legacy Authorization policyはO1 targetをcorroborateできるがauthor / backfillしない | test-only実装済み |
| A71 | X-TEA TaskAction / TaskFunctionIO分離はAction/Function分離をcorroborateするがgeneric protected-target roleは証明せず、詳細はO1 / role-specific auditへ保持する | external corroboration / no cross-register authority |
| A72 | manufacturing / inventory second-corpus outlineではO1 role反復・D4/H1/R3不足をまだ証明せず、Command/Observation境界だけを強く再現 | outline-level negative Evidence / Gate 15c未起動 |
| A73 | Command acceptanceだけではphysical completionを成立させない | second-corpus test-only Pressure実装済み |
| A74 | Observation read成功とphysical completionを分離し、positive external Observationだけをtest-local fact成立として扱う | test-only Pressure実装済み |
| A75 | exact second-corpus TEI Data / Pattern identityなしにownershipをsynthetic化しない | deferral boundary固定 |
| A76 | physical fact observedだけからH1/R3 Handoffを生成しない | boundary固定 |
| A77 | Procedure Step順序 / adjacencyからresponsibility Handoffを生成しない | test-only negative Pressure実装済み |
| A78 | positive physical Observationからresponsibility Handoffを生成しない | test-only negative Pressure実装済み |
| A79 | normalized H1/R3 relationはexplicit Handoff sourceだけから生成する | test-only authority Pressure実装済み |
| A80 | Business Action × Data/Relation traceをPattern / Validation / Capabilityのexact refsからderiveする | test-only projection実装済み |
| A81 | current sourceにData/Relationを重複authorせず有用なtrace rowsを得る | test-only Evidence |
| A82 | current refsからtraceがないStepを`No Data`へ丸めず`NoDataTraceFromCurrentRefs`として保持する | test-only unresolved preservation |
| A83 | BP-06 protected-target DataをO1 test helper / adjacent BP-07 / wordingから推測せず、security traceだけをnon-exhaustiveに保持する | test-only partial-coverage Evidence |
| A84 | Business Action × CRUDをexact accepted operation authorityからpartial deriveする | test-only projection実装済み |
| A85 | BP-00 / BP-02 / BP-03のRをQuery / Contract Query / Database Contract readからderiveする | test-only Evidence |
| A86 | semantic Data traceとCRUD operation traceを分離し、Relation両端が見えるだけでCRUD letterを付けない | test-only negative Pressure実装済み |
| A87 | `record` / `command` Pattern kindだけからR/U/C/Dを推測しない | test-only negative Pressure実装済み |
| A88 | `sales_core.order write sales.order.status`をaccepted action linkなしにBP-07=Uへ帰属させない | test-only negative Pressure実装済み |
| A89 | Business Lifecycle transitionをCRUD Uへ短絡せずBusiness state meaningとphysical mutationを分離する | gap audit固定 |
| A90 | current C/U/D attribution不足をBusiness Procedure field不足と断定せずCapability / realization / Database effect boundaryのpressureとして保持する | pre-contract gap audit / generic CRUD field defer |
| A91 | inline Procedure複数source + standalone Handoff sourceをone normalized Business semantic revisionへ束ねる | test-only実装済み |
| A92 | Business source freshnessをsemantic revisionから分離し、logical source key + content stateでstaleを検出する | test-only実装済み / production token未採用 |
| A93 | Business sourceがfreshでもcross-view Conformance baseが変わればPreviewをstaleとする | test-only実装済み |
| A94 | Procedure Step rename + Handoff endpoint renameは片側だけならfail closed、coordinated change-setならone new semantic revisionになる | test-only実装済み |
| A95 | migration previewでlegacy sourceを読めるが、legacy + candidate simultaneous authorityをrejectし、one cutover後candidate-only reloadへ進む | test-only実装済み / production cutover未採用 |
| A96 | Procedure actor context変更をHandoff endpoint identityへ複製せずProcedure + Step local addressでsame relationをresolveする | test-only structural Pressure |
| A97 | Procedure triggerとHandoff conditionを別authorityとして扱い、片側変更を他方へ暗黙同期しない | test-only structural Pressure |
| A98 | Watanabe external evidenceはthree-view mutual constraint / event-driven Business Flowをcorroborateするがactor/event/document field追加authorityにはしない | external corroboration / no schema expansion |
| A99 | current TEI-original Repositoryにindependent second realistic Handoffはなく、adjacency / defect / Lifecycle / Deploymentをsecond caseへ昇格しない | negative Evidence / Pre-ADR Gate C未充足 |
| A100 | workspace manifestのdeclared source rootをBusiness physical discovery候補にしてもunknown rootは`ir.World` Business embeddingを自動発生させない | test-only実装 / existing loader corroboration |
| A101 | declared Business root外にlegacy Reference fileが存在してもmanaged authority candidateへ入れず`file exists != authority reachable`を表現する | test-only実装済み |
| A102 | declared Business managed artifactはSource Index fingerprintを持ち、same bytesのphysical moveではlocatorだけが変わり`ir.World`は変わらない | test-only実装 / Business semantic move invarianceは未証明 |
| A103 | explicit `businessAuthoringMode`を追加せずmigration adapter + workspace/source-root discovery + candidate-only loaderでcutoverを表現できる可能性が高い | option Pressure / mode marker defer |
| A104 | candidate Procedure + standalone Handoffをdiskへ書き、legacy inputを読まずstrict decoderだけでsame normalized BusinessModelを再構成しcurrent WorldへConformanceする | test-only disk reload Pressure |
| A105 | candidate-only disk reloadはunknown compatibility fieldをfail closedしsteady-state loaderへのlegacy fallback再侵入を許さない | test-only negative Pressure |
| A106 | one physical `baseModelRevision`はBusiness source base + World / Conformance baseをfalse-negativeなくcoverするcomplete stale envelopeなら許容できる | option audit固定 / token schema未採用 |
| A107 | Git HEAD equalityだけではdirty / untracked authoritative source変更を検出できずunconditional freshness proofにはならない | negative boundary固定 |
| A108 | unrelated repository changeによるfalse-positive staleは許容できるがrelevant Business / World changeを見逃すfalse-negative staleはrejectする | correctness rule固定 |
| A109 | physical freshnessはfile contentだけでなくsource add/remove・authoritative source-root membership changeもcoverする | discovery-set boundary固定 |
| A110 | Business / World source planeが独立versioningされる場合は両stale conditionを独立検出またはcomposite tokenでcoverする | future reopen boundary固定 |
| A111 | Workspace / Source Indexはsource-root discovery・ManagedFiles・content fingerprint・stale validationのprimitiveを提供するがcandidate authority / Accept / atomic applyを所有しない | existing implementation reuse boundary |
| A112 | Workbench / Consumer Impact / Release Graphはdeterministic review / impact projectionの先例でありsecond source of truth / authoritative applyへ昇格しない | existing architecture corroboration |
| A113 | Deployment CLIの`expected-revision`とdurable plan/plugin/runtime fingerprintsはstale-reject先例だがDeployment Run.Revision / state machineをBusiness semantic/source authorityへ流用しない | existing implementation precedent / explicit non-reuse |
| A114 | current audited implementation surfacesではgeneric source-authoring Shared Acceptance / multi-file atomic apply / authority cutover / fresh-process Business reloadは確立されていない | scoped implementation gap / not schema Evidence |
| A115 | Lane C Pre-ADR Gate C（Handoff/address）とProduct Gate C（first Web Target）を別gateとして扱う | readiness naming boundary固定 |
| A116 | Contract ADR readinessとProduction adoption readinessを分離し、production source-authority実装完成をADR前提にしない | readiness boundary固定 |
| A117 | Product Gate C後のISSUE-000027 / 000046 Studio・AI Shared Acceptance end-to-endはBusiness Procedure Contract ADRの前提ではない | cross-roadmap dependency整理 |
| A118 | candidate-only sourceをlegacy inputなしにfresh process / separate executable boundaryで再構成できるfeasibilityはContract ADR Evidenceとして残す | executable feasibility blocker / actual未取得 |
| A119 | qualifying C/U/D case不存在はBusiness Procedure Contract ADR blockerではなく、unknown operation authorityを推測しないguardrail Evidenceとする | blocker縮約 / future mutation reopen trigger |
| A120 | `ir.World.Business`必要性Evidence不存在はblockerではなくseparate Business planeを維持するnegative Evidenceとする | placement blocker縮約 |
| A121 | duplicate `Procedure.id`が異なるcandidate physical pathからauthorされた場合、同一meaningでもconflicting meaningでもfail closedしpath / input orderでauthorityを決めない | source-level / test-only Pressure |
| A122 | exact same H1/R3 `from/to/condition` meaningが複数managed standalone Handoff sourceからauthorされた場合はduplicate authorityとしてfail closedする | source-level / test-only Pressure |
| A123 | same Handoff endpoints + different conditionをendpoint-only identityで潰さず、global Handoff IDも先取りしない | identity-boundary Pressure / no new ID |
| A124 | 公開横断External Design Reference Noteを詳細Evidenceの第二正本にせず、外部corroborationはquestion-specific Lane C auditへ近接保持する | evidence-responsibility boundary |

A91〜A124はcurrent minimum semantic candidateを拡張するものではない。主にauthoring representation、source discovery、freshness、migration cutover、implementation reuse、**ADRとproduction adoptionのreadiness境界**を狭めるPressureである。

## Candidate stack

```text
D3  broader Step body candidate
D4  smaller evidence-backed Step body candidate
S4-A Procedure-scoped Step local address
H1  thin normalized handoff
R3  handoff-only Business relation
C1  optional Reference Application context
O1  Application Pattern protected-target overlay pressure
```

Current minimum candidateは引き続き:

```text
D4 + singular narrow O1 + S4-A + H1/R3
```

であり、A91〜A124から新しいBusiness field / Core kindを追加しない。

### D4 Step body

```text
businessAction
applicationPatternRefs[]   // direct realization only
validationRefs[]
capabilityRefs[]
```

D4自身はStep local addressを持たない。S4-Aを別layerとして合成する。Current Referenceで未証明のdirect Data / Relation refsはminimumへ入れず、future sourceに現れた場合はfail closedして設計をreopenする。

### Business Action × Data projection

D4 / current authoring sourceへData/Relationを複製せずexisting refsからderiveする。

```text
Application Pattern -> Data / Relation / Capability data payload
Validation -> Attribute owner Data / Relation source+target
explicit Capability -> Data payload
```

Current主要結果:

```text
BP-00 -> sales.order
BP-01 -> sales.order
BP-02 -> sales.order + customer.customer / sales.order.customer
BP-03 -> sales.order + sales.order_line / sales.order.lines
BP-05 -> sales.order + sales.order_line / sales.order.lines
BP-06 -> security.authenticated_subject + authorization grant/denial
BP-07 -> sales.order
BP-04 / BP-08 / BP-09 -> no current Data trace
```

最後の3件をNo Dataとは扱わない。BP-06のsecurity traceが存在してもData dependency completenessを主張せず、authoritative sourceにないO1 protected-target Dataを隣接BP-07等から補完しない。

### Business Action × CRUD partial projection

Data traceとoperation traceを分離する。

Current exact rows:

```text
BP-00 -> R sales.order
BP-02 -> R customer.customer
BP-03 -> R sales.order_line
```

Current Database Contractの`sales_core.order write sales.order.status`を、accepted action linkなしにBP-07 / `sales.order.submit`へ帰属させない。Business Lifecycle proposalの`pending -> submitted`もphysical mutation / CRUD Uへ自動変換しない。

C/U/D不足はBusiness Procedure schema不足とは限らず、Capability / realization / Database effect boundaryのfuture pressureである。

### O1 target-role overlay

```text
BP-06
  D4 body
    applicationPatterns = []
    capabilities = [security.authorize]

  O1
    protected target = order_entry.submit_order
```

BP-06 target refを`applicationPatterns[]`へ流用するとdirect Function realizationへ意味を誤変換するためrejectする。

Legacy Authorization policyはcross-view corroborationには使えるが、policyからO1 targetを生成・補完しない。Generic ref-role taxonomyはcurrent一例から先行採用しない。

## Handoff / Business Flow boundary

```text
H1/R3
  Procedure/localStep -> Procedure/localStep + condition

Business Flow / DFD
  activity / participant / information-document-data flow等の別pressure
```

Current Handoff validator PASSをBusiness Flow / DFD Conformanceとはみなさない。

Authority direction:

```text
explicit accepted Handoff source
  -> normalized H1/R3 relation

Step order / adjacency
physical Observation
Application Function sequence
  -X-> Handoff
```

Actor / trigger boundary:

```text
Procedure.actor
  != Handoff endpoint identity

Procedure.trigger
  != Handoff.condition
```

双方は関連し得るが、Handoffへactor/event fieldを複製する根拠は現在ない。

## Authoring / discovery / freshness boundary

Current strongest physical direction:

```text
migration adapter
  -> legacy authorityをpreview inputとしてread-onlyに読む

candidate Procedure + standalone Handoff
  -> non-authoritative staging

accepted one-time authority cutover

steady-state discovery / loader
  -> candidate-only
```

Workspaceのdeclared source-rootをphysical discoveryに再利用できる可能性が高く、current Evidenceでは独立`businessAuthoringMode`を要求しない。

```text
file exists != authoritative source reachable
physical source path != Business logical identity != source authority
schemaVersion != source authority priority
```

Freshness:

```text
Business source freshness
  != Business semantic revision
  != cross-view Conformance base
```

Complete stale envelopeは少なくとも次をfalse-negativeなくcoverする。

```text
relevant source bytes
+ source add/remove
+ authoritative source-root membership
+ logical source uniqueness
+ World / Conformance base
```

One opaque tokenでもこのenvelopeをcoverできればよい。Git HEAD単体はdirty / untracked authoritative sourceを見逃し得るため無条件なfreshness proofにはしない。

### Existing implementation reuse boundary

```text
workspace sources / ManagedFiles
  -> physical source-set discovery

Source Index Artifact / fingerprint / Validate
  -> physical source freshness / trace

Workbench / Consumer Impact / Release Graph
  -> deterministic review / impact projection

Deployment expected-revision / bound fingerprints
  -> stale-reject precedent
```

ただしsource discovery primitiveはcandidate authorityではなく、fingerprint primitiveはcomplete Shared Acceptance revisionではない。Deployment Run.RevisionもBusiness semantic revisionではない。

Current audited surfacesでgeneric source-authoring Shared Acceptance / production atomic writer / production Business loaderが未確立でも、それだけではContract ADR blockerにしない。

### ADR readinessとProduction adoption

```text
Contract ADR
  -> semantic / authority / freshness invariantsを決めてよいか

Production adoption
  -> production adapter / stale envelope / atomic apply / authority cutover / reloadを安全に実装できたか

PAS-01 / Studio / AI
  -> Product Gate C後のend-to-end integration
```

Contract ADR前にはcandidate-only fresh-process **feasibility**を要求するが、production Shared Acceptance service完成は要求しない。

## Authorization / Observation boundary

```text
BP-06 Authorization
  supporting authority -> security.authorize
  protected target     -> order_entry.submit_order
  direct Function      -> none

BP-08 Observation
  Business intentionあり
  independent Semantic authority未証明
  -> Business wordingを保持
  -> runtime ReadIdentityをrefにしない
```

Manufacturing / inventory pressureでもCommand acceptedとphysical completionを分離し、positive external Observationからresponsibility Handoffを生成しない。

## 非対象

- production Canonical Procedure / Handoff schema採用
- `ir.World.Business`採用
- full Business Flow / DFD schema採用
- Workflow Engine / retry / timer / queue / saga / scheduler
- generic Core Edge / graph engine
- generic `semanticRefs[]` / kind-less Canonical ID bag
- generic explicit-ref Role enum
- global Canonical Step ID
- global Canonical Handoff ID
- Observation専用Canonical identity
- second-corpus synthetic Data ownership / Application Pattern
- inferred Handoff from adjacency / Observation / Function sequence
- Business Action × Data / CRUD projectionのCanonical化 / completeness claim
- Business Procedure / Pattern / Capabilityへのgeneric CRUD/effect field
- Database Contract write -> CRUD U自動変換
- Lifecycle transition -> CRUD U自動変換
- Business専用transaction / revision subsystem
- Git SHAのplatform-wide `baseModelRevision`固定
- permanent legacy/candidate dual-read
- schemaVersionによるauthority priority
- mandatory `businessAuthoringMode`
- Workbench / impact modelのauthoritative source化
- Deployment durable state machineのBusiness authoring / Workflowへの流用
- Product Gate B / Product Gate C完了条件変更

## Verification status

Repository上にはchecker / fixtures / Go Pressure testsを実装済みだが、actual Repository execution Evidenceは未取得である。Runner availabilityやexecution target revisionは一時状態なので本Issueへ固定せず、`BUSINESS_PROCEDURE_GATE_A_EXECUTION_RUNBOOK_2026-09.md`に従って実行時にexact revision `R`、runner kind、environmentを記録する。

PR #482は2026-09-09にmainへmerge済みで、PR #520でstandalone Handoff source uniqueness Pressureもmainへ入った。PR #529ではGate A runbookをpost-merge exact-revision execution modelへ更新した。ただしReference / Pressure / runbookのmergeはContract ADR adoptionやGate A greenを意味しない。

したがってactual `gofmt / go vet / go test / checker` greenとはまだ主張しない。A91〜A124もsource-level / test-only / architecture-boundary Evidenceであり、actual runner PASSへ読み替えない。Contract ADRは引き続きNot Readyで、material blockerはactual Repository runner greenとindependent second realistic TEI-original Handoffの2件である。

## Operational completion checklist

この一覧はLane Cの実行・整合確認用checklistであり、独立したReady / Not Ready stateを所有しない。Contract ADR readinessはPre-ADR Auditを正とする。

1. actual Repository environmentで`gofmt / go vet / go test`がgreen
2. baseline checker / intentional defectを再現
3. Business-only / uncertainty changeをmeaning差として再現
4. source move / logical rename / Step address rename境界を再現
5. manual / external workをfake Functionなしで保持
6. Pattern-owned meaningを重複しない
7. multi-Application / Application owner boundaryをactual testsで確認
8. `realization` subtraction / diagnostic independenceをactual testsで確認
9. S4-A / H1 / R3をactual testsで確認
10. Gate 15a / 15bをactual Evidenceで確認し15cは具体的不足時だけ起動
11. Authorization / Observation非対称境界をactual testsで確認
12. BP-03 / BP-05 authority separationをactual testsで確認
13. D3 / D4の最小十分性をcurrent Reference + provenance-backed Evidenceで説明
14. D4 future Data / Relation fail-closedをactual testsで確認
15. authority kindをgeneric ref bagへ失わない
16. D4 + S4-A + H1/R3 compositionをactual testsで確認しduplicate Handoffをfail closed
17. O1 protected target != realizationをactual testsで確認
18. D4 + O1 compositionをactual testsで確認
19. generic ref-role taxonomyの必要性を複数realistic pressureから判断
20. Lane B再監査と矛盾しない
21. H1/R3をhandoff-onlyとして保ちBusiness Flow surfaceを混入させない
22. O1をproduction accepted meaningとして残す場合はauthoritative source / decode / reload境界を閉じる
23. Legacy Authorization policyをO1 targetのauthor / backfill sourceにしない
24. second-corpusからexact identity / actor / sequenceを発明しない
25. physical-world Command/Observation Pressureをactual runnerで確認
26. second-corpus Data authorityをsynthetic化しない
27. physical Observationからresponsibility Handoffを生成しない
28. Step adjacencyからresponsibility Handoffを生成しない
29. explicit Handoff sourceだけがH1/R3をauthorすることをactual確認
30. Business Action × Data/Relation projectionをactual確認
31. no-traceをNo Dataへ丸めない
32. Business Action × CRUD partial projectionをactual確認
33. Data traceとCRUD operation traceを混同しない
34. Pattern kind / Database write / Lifecycle transitionからCRUDを補完しない
35. C/U/D不足をBusiness Procedure Contract ADR blockerにせずfuture mutation reopen triggerとして保持
36. multi-source Procedure + standalone Handoffをone Business semantic revisionへactual load
37. Business source freshnessとConformance-base freshnessをactualに独立fail closed
38. migration previewのlegacy-readとaccepted authorityを分離しdual authorityをreject
39. candidate-only sourceをlegacy inputなしのfresh process / separate executable boundaryでreloadしsame meaning + Conformanceを再現する **[Contract ADR feasibility]**
40. complete stale-envelope requirementをactual/prototypeで実証し、production token shapeは固定しない **[Contract ADR]**
41. second realistic TEI-original Handoffで`from/to/condition` sufficiencyを確認
42. production Business adapterではworkspace/Source Index primitiveを再利用する **[Production adoption]**
43. Workbench / impact projectionをauthoritative write authorityへしない
44. Deployment stale-reject precedentをBusiness semantic revision / Workflowへ流用しない
45. production Shared Acceptance substrateが確立するまでLane C専用generic transaction engineを作らない **[Production adoption]**
46. Lane C Pre-ADR Gate CとProduct Gate Cを混同しない
47. Product Gate C後のStudio / AI Shared Acceptance end-to-endをContract ADR blockerにしない
48. `ir.World.Business`必要性Evidenceがないことをplacement blockerにしない
49. production atomic source apply / authority-discovery cutover / partial-failure safetyをContract ADR後のProduction adoption Evidenceとして実証する
50. duplicate logical Procedureが複数managed physical pathから現れた場合、same/conflicting meaningともfail closedしpath/input orderでauthorityを決めない
51. exact duplicate H1/R3が複数managed standalone Handoff sourceから現れた場合はfail closedし、endpoint-only Handoff identityやglobal Handoff IDを発明しない

このchecklistの充足だけで独立にReadyを宣言しない。Contract ADR候補作成の最終判定はPre-ADR Auditに従う。
