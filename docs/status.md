# 現在の実装状況

- 更新日: 2026-09-10
- Product phase: architecture & product validation / research implementation
- Gate A: **Ready**
- Gate B: **Not Ready**
- Gate C: **Blocked — Gate B Ready + Handover待ち**

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

2026-09-08には、datasource aliasのdeclaration existenceとruntime / authority上のexact-one resolutionを分離する#511と、Business Procedure / Handoffのcross-view pressureを扱う#482がmainへ統合されています。

これらのmergeをactual package testやGate B Product Evidenceの成功へ読み替えません。candidate revision `R`はまだ固定しておらず、post-merge mainをactual full checkoutでsource / index preflightし、Gate B pre-checkで追加source修正が不要と確認した後に初めてRを選びます。

ただし、**Gate Bのactual Product Evidenceはまだ揃っていません。**

現在必要なのは、新しいDriver abstractionを増やすことではなく、post-merge mainをactual full checkoutで検証し、#310 / #255 / #308を同じclean candidate revisionへ実際に収束させ、#387 final convergence後にsame-run child exact artifactsを#273で独立再監査することです。

## Business Procedure / Lane C Pre-ADR

本status冒頭およびv1主線の`Gate A — Ready`は、Product主線のGate Aを指します。Business Procedure / HandoffのContract ADR readinessで使う**Lane C Pre-ADR Gate A — actual execution**とは別の判定です。

```text
Product Gate A
  = Ready

Lane C Pre-ADR Gate A
  = actual Repository execution Evidence unresolved

Business Procedure Contract ADR
  = Not Ready
```

Lane Cの作業正本は`ISSUE-000048`で、現在は**In Progress**です。Material blockerは二つです。

```text
Blocker A
  actual Gate A green
  including source-uniqueness / source-root membership /
  candidate-only fresh-process / second-corpus non-inference Pressure

Blocker B
  independent second realistic TEI-original Handoff
  from / to / condition sufficiency
```

現在のexecution priorityはBlocker Aです。`ISSUE-000050`では製造・在庫連携のsecond-corpus research targetとprovenance boundaryまでは選定済みですが、accepted independent Business seed / Procedure / Handoff positive Evidenceはまだ成立していません。Gate A green前はcorpus / provenance researchまでに留め、second-case検証のためのaccepted Handoff fixtureを作りません。

Lane Cの判定正本は[`BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md`](../plan/BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md)、actual execution手順は[`BUSINESS_PROCEDURE_GATE_A_EXECUTION_RUNBOOK_2026-09.md`](../plan/BUSINESS_PROCEDURE_GATE_A_EXECUTION_RUNBOOK_2026-09.md)です。

## 現在のv1主線

```text
Application Pattern Stage 1 — Done
        ↓
Gate A — Ready
        ↓
Application Driver — major semantics ready
        ↓
Gate B — Not Ready
        ↓
Handover Baseline B / revision R
        ↓
First Web Target build T1
        ↓
Gate C
        ↓
   ┌──────┼──────────────┐
   ↓      ↓              ↓
Studio  knowledge / AI  Second Target
PAS-01  PAS-06 / 01     PAS-04
```

## Gate B — actual Evidence待ち

Ready判定へ必須とするsource-dependent Evidenceは次の三つです。

| Evidence | Issue | 実装状態 | actual Evidence |
| --- | --- | --- | --- |
| deterministic Repository | `ISSUE-000032` / #310 | runner / source guard Ready | 未取得 |
| generated artifact reproducibility | `ISSUE-000025` / #255 | check / repair / source guard Ready | 未取得 |
| real PostgreSQL Provider + ephemeral fixture | `ISSUE-000031` / #308 | runner / DB safety / cleanup境界 Ready | 未取得 |

三つを同じclean candidate revision `R`へ揃え、`ISSUE-000037` / #387のfinal runnerで収束確認します。

#273の独立監査は、#387が実際に読んだsame-run 3 child artifactを`tools/reference-gate-b-readiness-artifact-audit`で再確認する形まで機械化し、正規Runbookへ接続済みです。fixed filename / exact bytes SHA-256 / lane identity / repository revision / post-run clean stateに加え、#310はWorkspaceと実行command、#255はgolden check command、#308はSemantic結果・fixture safety・cleanup等を再照合します。このartifact監査は第四Required laneではなく、`artifactAuditPassed=true`だけでGate B Readyにはしません。

actual Oracle #214 / #371 / #376は未実証部分が残っていますが、Gate Bの第四Required laneではありません。

- actual external asset / legacy compatibility / second-provider Evidenceとして継続
- 未完了ならHandover `unresolved`
- #310 synthetic Evidenceや#308 PostgreSQL Evidenceをactual Oracle成功へ読み替えない

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

ADR-0027 / #533の`DropTable` destructive readiness planning / rendering sourceもcurrent mainへ既に含まれています。旧stacked PR #535 head `17e80473adbc2b0cbe7ea213123c9b4296efe79e`がmainの祖先になったため、#535自体はintegration vehicleとして不要となりclosedされました。partition tree / target child / observation unavailableをfail closedし、complete observationを伴うordinary tableだけplain `DROP TABLE`へ進めるsourceは存在しますが、actual Repository qualificationは未取得です。したがってsourceの存在をactual greenやGate B Evidenceへ読み替えず、#533をqualification trackerとして継続します。

#534 / Draft PR #548では、その別incrementとしてPostgreSQL destructive `DropTable`のapply直前にfresh `SchemaObservation`を取得し、fresh observation-aware renderingとplanned statementsのexact一致を再確認する実装が進んでいます。stale / incomplete observation、新しいpartition child / descendant、realization差分ではexecutorを呼ばずfail closedします。これはgeneric migration revision protocolではなくPostgreSQL apply safetyの最小境界で、actual Repository testは未取得です。したがって#548はDraft / unmergedを維持し、source実装をactual qualificationやGate B Evidenceへ読み替えません。

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

Gate B Ready後、candidate revision `R`をHandover Baseline `B`として固定します。

First Web Targetの実装revisionは`T1`として別に追跡します。

```text
Handover B at R
      ↓ meaning baseline
Web build T1
      ↓
Gate C Evidence
```

Gate Cでは次を別Evidenceとして扱います。

1. Semantic Conformance
2. Interaction / Functional Evidence
3. Visual Fidelity Evidence
4. 必要に応じたPerformance Evidence

First Targetはまだ実装 / Evidence取得前です。

## Studio / Authoring

現時点ではDesign Onlyです。

Gate C後の最初のReferenceは`ISSUE-000046`です。

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

`ISSUE-000027` / ADR-0022で扱います。

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

- Gate B Ready
- Gate B actual #310 / #255 / #308 final Evidence
- actual Oracle E2E
- First Web Target / Gate C
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
- [Gate C Evidence Contract](../plan/GATE_C_EVIDENCE_CONTRACT.md)
- [Architecture](architecture.md)

細かなM0.xxの検証履歴は`docs/conformance/`、ADR、Git historyから追跡します。本statusはcurrent snapshotを優先します。
