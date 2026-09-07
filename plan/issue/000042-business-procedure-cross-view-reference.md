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
- `plan/XTEA_DESIGN_EVIDENCE_REGISTER_2026-09.md`
- `plan/BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md`
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

## Lane境界

本IssueはLane CのBusiness Procedure Reference / Evidenceを所有する。RQ-025 / RQ-026はProduct Requirementとして採用済み。PAS-07 Acceptance trace正本はLane B側へ残す。

Gate B / Gate C scope、Business Lifecycle 000038、共通change graph 000039を本Issueから拡張しない。

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
ref kind separation != ref role separation
Data / Function-Interaction / Business = 同じaccepted meaningを相互牽制する視点
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

Manufacturing / inventory second-corpus outlineではphysical-world `Command != actual completion`をtest-only Pressureまで進めた。さらにH1/R3について、Step adjacencyやpositive physical ObservationからHandoffを生成せず、explicit Handoff sourceだけをnormalized relationのauthorityとするnegative Pressureを追加した。一方、exact TEI Data authority、O1 protected-target roleのsecond-domain反復、new production contract不足はまだ証明されない。したがってGate 15cは**未起動**を維持する。

## Cross-view checker原則

- explicit referenceはexact identityで解決
- Evidence-backedなBusiness dependencyだけを保持
- 名称類似で補完しない
- Pattern ID文字列からownerを推測しない
- Pattern + owner Applicationをnormalized Worldからexact解決
- Projectionだけでaccepted authorityを拡張しない
- Business Action × Data projectionはevidence pathだけを表示し、complete dependencyを主張しない
- current refsからData traceがない行を`No Data`へ丸めない
- `realization` labelからref kind / roleを推測しない
- runtime Observation APIをSemantic identityへ昇格しない
- Command acceptanceからphysical completionを推測しない
- physical ObservationからHandoff endpoint / conditionを生成しない
- Step order / adjacencyからHandoffを生成しない
- Application Function sequenceからHandoffを生成しない
- exact TEI Data / Pattern identityなしにsecond-corpus ownershipをsyntheticに作らない
- adjacencyからprotected targetを推測しない
- Authorization policyはprotected targetをcorroborateできるが、Business targetをauthor / backfillしない

## Acceptance / Evidence Pressure matrix

このmatrixはLane Cの**Evidence / Pressure inventory**であり、独立したReady / Not Ready判定を所有しない。production Canonical contract ADR readinessの判定正本は`BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md`の「ADR作成前Gate」とする。

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
| A50 | X-TEA fallbackでBusiness Flow独立pressureを確認 | E-014追加済み |
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
| A71 | X-TEA TaskAction / TaskFunctionIO分離はAction/Function分離を支持するがgeneric protected-target roleは証明しない | E-015固定 |
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

### D4 Step body

```text
businessAction
applicationPatternRefs[]   // direct realization only
validationRefs[]
capabilityRefs[]
```

D4自身はStep local addressを持たない。S4-Aを別layerとして合成する。Current Referenceで未証明のdirect Data / Relation refsはminimumへ入れず、future sourceに現れた場合はfail closedして設計をreopenする。

### Business Action × Data projection

`business_action_data_projection_pressure_test.go`では、D4 / current authoring sourceへData/Relationを複製せず、existing refsから次をderiveする。

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

最後の3件をNo Dataとは扱わない。さらにBP-06のsecurity traceが存在してもData dependency completenessを主張せず、authoritative sourceにないO1 protected-target Dataを隣接BP-07等から補完しない。

この結果はT-GAP-03へ構造上の回答を与えるがactual runner green待ちである。D4へData/Relation slotを戻す根拠にはならない。

### O1 target-role overlay

```text
BP-06
  D4 body
    applicationPatterns = []
    capabilities = [security.authorize]

  O1 overlay
    applicationPatternTargets = [order_entry.submit_order]
```

BP-06 target refを`applicationPatterns[]`へ流用するとdirect Function realizationへ意味を誤変換するためrejectする。

Target-only changeはD4 bodyを変えずcross-view meaningを変える。BP-07隣接realizationからtargetをderiveしない。

Legacy Authorization policyの`execute / order_entry.submit_order / allow`はO1 targetのcross-view corroborationには使えるが、policyからO1 targetを生成・補完しない。Policy側の変更はBusiness targetを書き換えず、不一致をreview / diagnostic pressureとして露出させる。

Manufacturing / inventory second-corpus outlineにはexact Application Pattern realization / policyがまだないため、O1と同じprotected-target roleをsyntheticに作らない。Generic `ExplicitSemanticRef{Kind,Role,Target}` taxonomyは引き続きcurrent一例から先行採用しない。

Current O1 / D4+O1 testsのtargetはtest helper引数であり、authoritative Business sourceへpersistした意味ではない。O1をproduction accepted meaningとして残す場合だけfile-native source / decode / reload / Shared Acceptance pressureを追加する。残さない場合はproduction source fieldを追加しない。

## Handoff / Business Flow boundary

```text
H1/R3
  Procedure/localStep -> Procedure/localStep + condition

Business Flow / DFD
  activity / participant / information-document-data flow等の別pressure
```

Current Handoff validator PASSをBusiness Flow / DFD Conformanceとはみなさない。

Second corpusではphysical factの観測がresponsibility continuationの前提になり得るが、exact Procedure/Step endpointとaccepted conditionがないためH1/R3を生成しない。

`second_corpus_handoff_inference_pressure_test.go`では、explicit Handoff sourceを除去した状態でStep order / adjacencyを残してもnormalized Handoffs=0、positive physical Observationがあっても0、explicit relationを与えた場合だけ1件になることをPressureする。

Authority direction:

```text
explicit accepted Handoff source
  -> normalized H1/R3 relation

Step order / adjacency
  -X-> Handoff

physical Observation
  -X-> Handoff

Application Function sequence
  -X-> Handoff
```

## Authorization / Observation boundary

```text
BP-06 Authorization
  supporting authority -> security.authorize
  protected target     -> order_entry.submit_order (O1 pressure)
  direct Function      -> none

Legacy Authorization policy
  security-side authority
  may corroborate protected target
  must not author / backfill protected target

BP-08 Observation
  Business intentionあり
  independent Semantic authority未証明
  -> Business wordingを保持
  -> runtime ReadIdentityをrefにしない

manufacturing / inventory pressure
  command accepted != material received / manufacturing completed / stock present / shipment completed
  -> no observation: BUSINESS_PHYSICAL_COMPLETION_REQUIRES_OBSERVATION
  -> observed=false: BUSINESS_PHYSICAL_COMPLETION_NOT_OBSERVED
  -> positive external observation only: test-local physical fact may be observed
  -> no Canonical Observation identity
```

## 非対象

- production Canonical Procedure / Handoff schema採用
- `ir.World.Business`採用
- full Business Flow / DFD schema採用
- Workflow Engine / retry / timer / queue / saga / scheduler
- generic Core Edge / graph engine
- generic `semanticRefs[]` / kind-less Canonical ID bag
- generic explicit-ref Role enum
- production `applicationPatternTargets` field採用
- global Canonical Step ID
- `realization` production enum
- Observation専用Canonical identity
- second-corpus synthetic Data ownership / Application Pattern
- inferred Handoff from adjacency / Observation / Function sequence
- Business Action × Data projectionのCanonical化 / completeness claim
- Gate B / Gate C完了条件変更

## Verification status

Repository上にはchecker / fixtures / Go pressure testsを実装済み。GitHub Actionsは過去にrunner assignment前で終了し、一時workflowはmerge対象から除去した。Remote Desktopも2026-09-07時点で接続端末がなく実行できない。

したがってactual `gofmt / go vet / go test` greenとはまだ主張しない。PR #482はDraftを維持する。

## Operational completion checklist

この一覧はLane Cの実行・整合確認用checklistであり、独立したReady / Not Ready stateを所有しない。production Canonical contract ADR readinessはPre-ADR Gateを正とする。

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
13. D3 / D4の最小十分性をcurrent Reference + provenance-backed second-corpus Evidenceで説明し、fixture成熟度とは分離する
14. D4 future Data / Relation fail-closedをactual testsで確認
15. authority kindをgeneric ref bagへ失わない
16. D4 + S4-A + H1/R3 compositionをactual testsで確認し、normalized duplicate Handoffをfail closed
17. O1 protected target != realizationをactual testsで確認
18. D4 + O1 compositionをactual testsで確認
19. generic ref-role taxonomyの必要性を複数realistic pressureから判断
20. Lane B再監査と矛盾しない
21. H1/R3をhandoff-onlyとして保ちBusiness Flow surfaceを混入させない
22. O1をproduction accepted meaningとして残す場合はauthoritative source / decode / reload / Shared Acceptance境界を閉じる。残さない場合はproduction source fieldを追加しない
23. Legacy Authorization policyをO1 protected targetのauthor / backfill sourceにしない
24. second-corpus outlineからexact identity / actor / sequenceを発明せず、fixture化はactual green後に必要性がある場合だけ行う
25. physical-world Command/Observation test-only Pressureをactual runnerで確認する
26. second-corpus Data authorityはexact TEI-original Data / Pattern / Database authorityが揃うまでsynthetic化しない
27. physical Observationからresponsibility Handoffを自動生成しない
28. Step order / adjacencyからresponsibility Handoffを自動生成しない
29. explicit Handoff sourceだけがnormalized H1/R3 relationをauthorすることをactual runnerで確認する
30. Business Action × Data/Relation trace projectionをactual runnerで確認する
31. no-trace rowsをNo Dataへ丸めず、projection completenessを主張しない

このchecklistの充足だけで独立にReadyを宣言しない。ADR候補作成の最終判定はPre-ADR Gateに従う。
