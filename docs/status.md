# 現在の実装状況

- 更新日: 2026-09-20
- Product phase: architecture & product validation / research implementation
- Gate A: **Ready**
- Gate B: **Ready**（candidate revision `R = e5231839a5dcc62cce1181a2731fcb7c70d16736`、Handover Baseline `B`固定済み）
- Gate C: **Ready**（First Web Target build `T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe`、Handover Baseline `B / R`。Visual Fidelityは単一のAI評価者による評価 — [`GATE_C_READINESS_AUDIT.md`](../plan/GATE_C_READINESS_AUDIT.md)）
- R7 Replayable Behavior: **Source / Design進行中**（R7A handoff、R7B execution intake、exact replay plan、exact-T operational replay contractまで定義済み。Gate C origin EvidenceはRelease `gate-c-evidence-T-0ba87a4` に保管済み。actual R7A/R7B materialization、replay execution、repeatabilityは未実行）

TEIは完成したlow-code製品ではありません。

業務上の意味、authoring source、実現方法、実行結果、外部観測をどこで分けるべきかを、実行可能なcode、Reference、Conformance、Evidenceから段階的に検証しています。

## 現在地

`order_entry` Applicationでは次が成立しています。

- Application Pattern Stage 1
- Gate A
- Application Driverの主要意味境界
- browse / record / reference / header-detail / commandのUI-less Scenario構造
- Validation
- Authentication / Authorization
- Capability Binding
- Business Result / Execution Error
- Command後のseparate Observation read
- external Evidence intake boundary
- generated artifact Evidence runner
- deterministic Repository Evidence runner
- real PostgreSQL ephemeral fixture Evidence runner / safety boundary
- Gate B final convergenceで保持したsame-run child Evidenceの独立artifact監査tool / Runbook接続
- Gate B actual #310 / #255 / #308 final Evidence（candidate `R = e5231839a5dcc62cce1181a2731fcb7c70d16736`、2026-09-19）
- Gate B → Gate C Handover Baseline `B / R`固定（[`GATE_B_TO_C_HANDOVER_BASELINE_B.md`](../plan/GATE_B_TO_C_HANDOVER_BASELINE_B.md)）
- Gate C actual Evidence（Semantic / Functional、Visual Fidelity、architecture conformance、final readiness audit `gateCReady = true`。`T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe`、2026-09-19、[`GATE_C_READINESS_AUDIT.md`](../plan/GATE_C_READINESS_AUDIT.md)）

2026-09-08には、datasource aliasのdeclaration existenceとruntime / authority上のexact-one resolutionを分離する#511と、Business Procedure / Handoffのcross-view pressureを扱う#482がmainへ統合されています。

2026-09-19には、actual Evidence取得中に見つかった2件のsource-navigation bug（`semanticDataAnchors`のrelation `key`句未対応、`database-contract-query`のcontract wrapper未対応）をPR [#726](https://github.com/hat47x/tei-platform/pull/726)で、独立artifact auditが検出した`contractQueryTarget`期待値のstaleさをPR [#727](https://github.com/hat47x/tei-platform/pull/727)で修正し、両PRのsquash-merge後の`main` HEAD（`e5231839a5dcc62cce1181a2731fcb7c70d16736`）をclean checkoutして#310 / #255 / #308 / #387 / 独立artifact auditを最初から再実行しました。**Gate Bのactual Product Evidenceは、この`R`についてすべて揃っています。**

判定の詳細は[`GATE_B_READINESS_AUDIT.md`](../plan/GATE_B_READINESS_AUDIT.md)、Handoverは[`GATE_B_TO_C_HANDOVER_BASELINE_B.md`](../plan/GATE_B_TO_C_HANDOVER_BASELINE_B.md)を参照してください。

現在必要なのは、新しいDriver abstractionを増やすことではなく、Gate C通過後の並行lane（Studio / knowledge-first・AI Proposal / 第二Application・第二Target）をどの順で開始するかを、Evidenceに基づいて判断することです。

第一Web Targetでは `tools/reference-first-web-target` にReference-specificなprojectionを置き、Driver Resolverが解決した5種類のApplication PatternをHTML / JSONへProjectionしている。read-only sliceとしてbrowse → record → customer reference → order-lines header-detailをDriver executionへ接続し、さらにAuthentication → Canonical Validation → Authorization → Command Business Result → separate ObservationのScenario sourceまで接続した。Business denial / Validation not-satisfied / Execution Error / Observationを同一errorへ畳まない。Custom UIもJSON APIと同じ`commandScenarioRunner`を共有し、form / layout / HTML renderingだけをTarget固有差として持つ。汎用Presentation IRやWeb frameworkをCanonicalへ追加していない。browseのEnvironment instance selectionはcurrent Driver APIに存在しないためWeb側で推測せずunresolvedとして保持する。actual Target run用のSemantic / Functional Evidence runner / independent audit sourceに加え、同一`T`のScreenshotと評価記録を受け入れるVisual Evidence audit sourceも追加している。 さらにHandover `R`→Target `T`のWorkspace / generated World不変とWeb Target authority boundaryを確認するarchitecture conformance sourceを追加し、same-run `architecture-conformance.json`として取得する設計にした。Custom UIのsuccess / Authorization denied / Validation not-satisfied / Execution Errorを構造化した画面状態として分離し、Semantic / Functional runnerが各stateのHTML artifactも保存・auditするようにした。Runbookには同じ`T`で各Screenshot stateを再現するdeterministic capture recipeも追加した。さらにstandard Projectionと各Custom UI stateのsame-run Target-rendered HTMLをraw artifactとしてfingerprintし、Visual manifestの各Screenshotへ`sourceArtifact / sourceSHA256`を自動で結び付けるようにした。これによりreviewerはcredential付き操作を再入力せず保存済みHTML replayからcaptureできる。さらに`tools/reference-first-web-target-visual-replay`でsame-run digestを再検証し、required HTMLをloopbackからexact bytesで配信できるようにした。 Visual manifest scaffoldを渡せばcapture route / Screenshot filename / evaluator / criterion-capture対応をreview indexで一覧できるが、この画面は閲覧補助でありVisual evaluatorやauditの代替ではない。HTML artifactをVisual Fidelityへ読み替えず、Visual auditがpassしてもそれだけでGate C Readyにしない。2026-09-19に、clean checkout（`T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe`）上でactual server run / Functional artifact / Screenshot / evaluator recordを取得し、final Gate C readiness auditが`gateCReady = true`を返した（[`GATE_C_READINESS_AUDIT.md`](../plan/GATE_C_READINESS_AUDIT.md)）。取得の過程で、Runbookのsource qualificationがuncompilableなtest fileとgofmt差で停止したこと（PR [#815](https://github.com/hat47x/tei-platform/pull/815)）、Visual reviewで標準Projection headerのrevision値が重なる不具合が見つかり修正したこと（PR [#816](https://github.com/hat47x/tei-platform/pull/816)）を、旧`T`のEvidenceをcarry-forwardせず最初から再取得して解消している。Visual Fidelityの評価者は単一のAI評価者（VLM）であり、人による確認ではない。保持したScreenshotは別の評価者が再評価でき、`fail`が出た場合はGate Cを再監査する。

## Business Procedure / Lane C Pre-ADR

本status冒頭およびv1主線の`Gate A — Ready`は、Product主線のGate Aを指します。Business Procedure / HandoffのContract ADR readinessで使う**Lane C Pre-ADR Gate A — actual execution**とは別の判定です。

```text
Product Gate A
  = Ready

Lane C Pre-ADR Gate A
  = actual Repository execution PASS（R = e2856e6、2026-09-20）

Business Procedure Contract ADR
  = Not Ready
```

Lane Cの作業正本は`ISSUE-000048`で、現在は**In Progress**です。Material blockerは当初二つでしたが、Blocker Aは2026-09-20に充足し、Blocker Bは2026-09-21にEvidenceを取得しました。残るのはPre-ADR Auditによる再判定です。

```text
Blocker A — 充足（2026-09-20）
  actual Gate A green
  including source-uniqueness / source-root membership /
  candidate-only fresh-process / second-corpus non-inference Pressure

Blocker B — Evidence取得済み（2026-09-21、R = 82e4301、作成者self-review。再判定は未実施）
  independent second realistic TEI-original Handoff
  from / to / condition sufficiency
```

現在のexecution priorityは、Blocker Bの再判定です。`ISSUE-000050`で`order_entry`から独立した製造・入庫のsecond Referenceをauthorし、責任のHandoffを一件置き、source / test levelと、同一revision `R = 82e4301`のactual execution（Gate A A1〜A8、critical 39件PASS）のEvidenceを取得しました（2026-09-21）。ただし作成者によるself-reviewだけで、独立reviewは未実施です。Handoff一般の妥当性は主張せず、Contract ADRは**Not Ready**のままです。

Lane Cの判定正本は[`BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md`](../plan/BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md)、actual execution手順は[`BUSINESS_PROCEDURE_GATE_A_EXECUTION_RUNBOOK_2026-09.md`](../plan/BUSINESS_PROCEDURE_GATE_A_EXECUTION_RUNBOOK_2026-09.md)です。

## 現在のv1主線

```text
Application Pattern Stage 1 — Done
        ↓
Gate A — Ready
        ↓
Application Driver — major semantics ready
        ↓
Gate B — Ready（R = e5231839a5dcc62cce1181a2731fcb7c70d16736）
        ↓
Handover Baseline B / revision R — 固定済み
        ↓
First Web Target build T — 実装・Evidence取得済み（T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe）
        ↓
Gate C — Ready
        ↓
   ┌──────┼──────────────┐
   ↓      ↓              ↓
Studio  knowledge / AI  Second Target
PAS-01  PAS-06 / 01     PAS-04
```

## Gate B — actual Evidence成立

Ready判定へ必須とするsource-dependent Evidenceは次の三つで、いずれもcandidate revision `R = e5231839a5dcc62cce1181a2731fcb7c70d16736`のsame-runで成立しています（2026-09-19）。

| Evidence | Issue | 実装状態 | actual Evidence |
| --- | --- | --- | --- |
| deterministic Repository | `ISSUE-000032` / #310 | runner / source guard Ready | 成立 |
| generated artifact reproducibility | `ISSUE-000025` / #255 | check / repair / source guard Ready | 成立 |
| real PostgreSQL Provider + ephemeral fixture | `ISSUE-000031` / #308 | runner / DB safety / cleanup境界 Ready | 成立 |

三つは`ISSUE-000037` / #387のfinal runnerでsame-Rの収束を確認済みです。

#273の独立監査は、#387が実際に読んだsame-run 3 child artifactを`tools/reference-gate-b-readiness-artifact-audit`で再確認する形まで機械化し、正規Runbookへ接続済みです。fixed filename / exact bytes SHA-256 / lane identity / repository revision / post-run clean stateに加え、#310はWorkspaceと実行command、#255はgolden check command、#308はSemantic結果・fixture safety・cleanup等を再照合します。このartifact監査は第四Required laneではなく、`artifactAuditPassed=true`だけでGate B Readyにはしません。`R = e5231839a5dcc62cce1181a2731fcb7c70d16736`に対する独立監査は`artifactAuditPassed = true`でした。

actual Oracle #214 / #371 / #376は未実証部分が残っていますが、Gate Bの第四Required laneではありません。

- actual external asset / legacy compatibility / second-provider Evidenceとして継続
- 未完了ならHandover `unresolved`
- #310 synthetic Evidenceや#308 PostgreSQL Evidenceをactual Oracle成功へ読み替えない

### 検証とGitHub Actions

GitHub Actionsは月間quotaの都合で使えず、GitHub-hosted runnerがjob開始前に割り当てられない状態が続いています（`runner_id=0` / `steps=[]`のcheckout前failure）。これをApplication / Driverのcode failureへ読み替えません。

検証の正本はlocalの`tools/repository-health-check.sh`（`gofmt -l` / `go vet` / `go test` / `tei validate`）です。2026-09-19にリポジトリ全体を点検し、gofmt差・コンパイルできないpackage・古くなったtestを解消して、mainがすべてgreenであることを確認しました（[`ISSUE-000055`](../plan/issue/000055-verification-gate-while-ci-suspended.md)）。

Gate BのProduct Evidenceは、利用可能なactual full checkoutでのlocal actual executionだけを根拠にしています。CI greenはProduct Evidenceの代わりにはならず、今後もそうです。過去のGate B専用source qualification workflowは、health checkに包含されるため削除しました。

Gate B判定の正本は[`GATE_B_READINESS_AUDIT.md`](../plan/GATE_B_READINESS_AUDIT.md)です。

## Semantic / Source

実装済みの主な基線:

- Workspace / Canonical ID
- Semantic namespace / type / data / relation / capability
- Source Index
- source fingerprint / range / role
- Diagnostic → Source projection
- native `.sql` authoritative source
- file pathとsemantic identityの分離

file moveだけでlogical identityが変わらない場合はsemantic no-opになり得ます。

logical rename / Canonical identity変更はSemantic changeとして扱います。

## Application / Pattern

`order_entry`では次の5 PatternをApplication-owned intentとして扱えます。

```text
browse
record
reference
header-detail
command
```

成立している主な境界:

- Application `DatabaseRequirement[]`
- logical Datasource
- read / read-write Access
- Pattern Canonical identity
- Pattern Source / Diagnostic
- Pattern Validation
- Capability Binding
- Authentication / Authorization
- Canonical business Validation

5分類をTEI全体の永久的な標準Catalogとはみなしません。

## Driver / Execution

主要なUI-less意味境界は成立しています。

```text
Authentication
 -> browse
 -> record
 -> reference
 -> header-detail
 -> Validation
 -> Authorization
 -> Command
 -> Business Result
 -> separate explicit read
 -> Observation
```

次を混同しません。

- Business Result
- named Outcome
- Execution Error
- Observation
- Developer Diagnostic

Command acknowledgementだけでexternal state成立とはみなしません。

## Database / Provider

実装・Referenceで確認している主な境界:

- Database Contract
- Persistence Mapping
- Desired / Actual Physical Schema
- Diff / Unresolved
- Migration
- Provider-specific normalization / rendering
- PostgreSQL observation
- Contract Query SQL realization
- provider-specific runtime Runner
- Database-owned Relation Storage

PostgreSQL partitionについては、ADR-0026のprovider-owned partition observation incrementがPR #528でmainへ統合済みです。partition childをgeneric Schema Driftの独立convergence subjectとして扱わず、provider observationとして保持する境界までがcurrent baselineです。

ADR-0027 / #533の`DropTable` destructive readiness planning / rendering sourceもcurrent mainへ既に含まれています。旧stacked PR #535 head `17e80473adbc2b0cbe7ea213123c9b4296efe79e`がmainの祖先になったため、#535自体はintegration vehicleとして不要となりclosedされました。partition tree / target child / observation unavailableをfail closedし、complete observationを伴うordinary tableだけplain `DROP TABLE`へ進めるsourceは存在します。#533はsource implementation taskとして2026-09-15にcompleted closeし、actual Repository qualificationはPR #599で拡張したbaseline source qualificationと#273へ集約しました。sourceの存在やIssue closeをactual greenやGate B Evidenceへ読み替えません。

#534 / PR #548の別incrementもcurrent mainへ統合済みです。PostgreSQL destructive `DropTable`のapply直前にfresh `SchemaObservation`を取得し、fresh observation-aware renderingとplanned statementsのexact一致を再確認します。stale / incomplete observation、新しいpartition child / descendant、realization差分ではexecutorを呼ばずfail closedします。これはgeneric migration revision protocolではなくPostgreSQL apply safetyの最小境界です。#534もsource implementation taskとしてcompletedであり、**actual Repository testは未取得のため、merge済みsourceをactual qualificationやGate B Evidenceへ読み替えません。**

PostgreSQLは深いReference Providerとして利用しています。

actual second-provider / existing Oracle assetは別Evidenceとして継続します。

## Plugin / Runtime / Resource

成立している主な境界:

- Plugin / Bundle / Capability negotiation
- Plugin Lock
- in-process / process runtime
- actual child process stdio boundary
- artifact verification
- explicit Permission Grant
- Resource Broker / Lease
- secret / network / filesystem logical binding
- Sandbox Profile
- Host Lease Supervisor
- ownership / heartbeat / fencing
- Durable Invocation Handle
- observe / reattach
- no-auto-replay / unknown保持

Application userのAuthentication / Authorizationと、Plugin / Runtime Permission / Resource authorityは別の責務です。

## Handover / Target

Gate B Readyを受けて、candidate revision `R = e5231839a5dcc62cce1181a2731fcb7c70d16736`をHandover Baseline `B`として固定しました（[`GATE_B_TO_C_HANDOVER_BASELINE_B.md`](../plan/GATE_B_TO_C_HANDOVER_BASELINE_B.md)）。

First Web Targetの実装revisionは`T1`として別に追跡します。

```text
Handover B at R  ← 固定済み
      ↓ meaning baseline
Web build T1  ← source implementation / Evidence tooling成立
      ↓          actual clean T1 runは未取得
Gate C Evidence
```

Gate Cでは次を別Evidenceとして扱います。

1. Semantic Conformance
2. Interaction / Functional Evidence
3. Visual Fidelity Evidence
4. 必要に応じたPerformance Evidence

First Targetは実装済みで、actual clean Target run、Visual capture、final Gate C readiness auditを`T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe`で取得し、Gate C Readyと判定しました（[`GATE_C_READINESS_AUDIT.md`](../plan/GATE_C_READINESS_AUDIT.md)）。Visual Fidelityの評価者は単一のAI評価者で、人による確認は行っていません。

## Studio / Authoring

Studio UIはDesign Onlyです。ただしStudioも使うShared Acceptance Boundaryは、2026-09-21に決定論的なReferenceとして成立しました（[`tools/reference-shared-acceptance`](../tools/reference-shared-acceptance/README.md)）。file / CLI / Studio / AIのどの入口のcandidateも同じ検査を通り、acceptだけがauthoritative sourceを書き換えます。scratch copyでPreview / Reject / Accept / stale / 受理後の再読込を確認済みで、扱うoperationは`add-validation`、既存DataへのRelation追加（`add-relation`。`ISSUE-000046`の最小loop）、Second Application用の型付きoperation（`add-data` / `add-application` / `add-reference-pattern`）です。実際の`reference/workspace`へのSecond Application compositionのPreviewはPASSしていますが、Acceptは実行していません（人のAcceptance decisionが必要）。

Gate C後の最初のStudio Referenceは`ISSUE-000046`です（In Progress。UI・Relation変更は未着手）。

```text
Select
 -> Propose
 -> Preview
 -> Accept / Reject
 -> Shared Acceptance Boundary
 -> authoritative source
 -> reload / Conformance
```

StudioはCanonicalの第二正本にはしません。

Preview / Rejectはnon-mutating、stale `baseModelRevision`は明示処理します。

Studio都合でmandatory authored Stable ID / UUIDを導入しません。

Studioのdeveloper-facing Wails HostをGenerated Wails Desktop Application Targetと混同しません。

## knowledge-first / AI

現時点ではDesign Onlyです。

### knowledge-first exploration

`ISSUE-000047`でGate C後に実証します。

Human / AIが同じauthoritative source / Canonical / Evidenceをread-only探索し、次を分けます。

- applicable
- incompatible
- unknown

search ranking / similarity / embedding / model confidenceをSemantic Evidenceへ格上げしません。

### AI Proposal

`ISSUE-000027` / ADR-0022で扱います。AIによるcandidateの生成は未着手ですが、受け取る側の境界は上記のReferenceとして存在し、AI由来のcandidate（`origin.kind = ai`）も同じ経路を通ります。

AI ProposalはCanonical authorityではありません。

file-native / Studio / CLI / AIをShared Acceptance Boundaryへ収束させます。

AI provider、vector database、Proposal最終schema、chat-first UXは未固定です。

## Second Target

現時点ではDesign Onlyです。

Gate C後、`ISSUE-000026`で第一Targetと十分に異なるApplication Targetを選びます。

比較対象:

- Handover / Canonical meaning
- Target build identity
- Target Capability
- Runtime Binding
- Execution / Trust Profile
- Host control plane
- Semantic / Functional / Visual Evidence
- RQ-024 performance

Studio Wails Hostは第二Application Target Evidenceへ数えません。

## 現時点で完成済みとは言わないもの

- actual Oracle E2E
- Visual Fidelityの人による承認（Gate C Visualは単一のAI評価者による評価）
- complete Target-neutral Presentation IR
- production-ready Web / Desktop / Mobile Target
- Studio actual Authoring Reference
- production-ready GUI Editor
- knowledge-first exploration actual Reference
- AI Proposal / Decision UX actual Reference
- second Application Target
- production-grade Workflow / HA / distributed ownership
- generic OS sandbox
- API compatibility guarantee

設計契約、isolated test、runner実装をactual Product Evidenceへ格上げしません。

## 参照先

- [TEI Master Plan](../plan/MASTER_PLAN.md)
- [v1 Reference Application Plan](../plan/V1_REFERENCE_APPLICATION_PLAN.md)
- [Upstream Acceptance Scenario Traceability](../plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md)
- [Product Requirement Gap Audit](../plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md)
- [Gate B Readiness Audit](../plan/GATE_B_READINESS_AUDIT.md)
- [Gate B Evidence Runbook](../plan/GATE_B_EVIDENCE_RUNBOOK.md)
- [Gate B → C Handover Contract](../plan/GATE_B_TO_C_HANDOVER_CONTRACT.md)
- [Gate B → C Handover Baseline B](../plan/GATE_B_TO_C_HANDOVER_BASELINE_B.md)
- [Gate C Evidence Contract](../plan/GATE_C_EVIDENCE_CONTRACT.md)
- [Gate C First Web Target Evidence Runbook](../plan/GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md)
- [Architecture](architecture.md)

細かなM0.xxの検証履歴は`docs/conformance/`、ADR、Git historyから追跡します。本statusはcurrent snapshotを優先します。
