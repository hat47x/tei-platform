# ISSUE-000029: Gate B readinessをReference / Evidenceから再監査する

- 状態: In Progress
- GitHub Issue: #273
- 対象: `ISSUE-000006` / GitHub Issue #69
- 次段: `ISSUE-000007` / GitHub Issue #70
- Handover: `ISSUE-000030` / GitHub Issue #291

## 背景

Application Driverの主要な意味境界とScenario構造は成立し、PR #285で`main`へ収束しました。

さらにPR #300で、PR #288 / #294 / #296から次を最新`main`へ収束しました。

- external-observed Contract Query Evidence intake
- PostgreSQL header-detail Evidence toolのfail-closed安全境界
- Gate B Evidence Runbook
- #214のRepository Issue文書

したがって、Gate B readinessは「どの実装がまだbranchに残っているか」ではなく、必要なEvidenceが揃っているかで監査します。

監査結果の正本は[`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md)に置きます。

Evidence取得については役割を分けます。

- [`GATE_B_EVIDENCE_ACQUISITION.md`](../GATE_B_EVIDENCE_ACQUISITION.md): 何を取得するか
- [`GATE_B_EVIDENCE_RUNBOOK.md`](../GATE_B_EVIDENCE_RUNBOOK.md): どの順序・安全条件・停止条件で再現するか

## 現在の判定

2026-09-04時点では **Gate B: Not Ready** です。

一方、次は未設計blockerではありません。

- browse / record / reference / header-detail / commandのApplication境界
- Contract Query signature / SQL realization / explicit mapping
- Environment / Database instance / executor Capabilityを保持するruntime境界
- Relation Storage
- Validation -> Authorization -> Command
- Command acknowledgement -> separate Observation read
- 最終UI-less Scenarioの構造統合
- external-observed Evidenceを非Canonicalのまま受け入れる境界
- PostgreSQL Evidence fixtureの既存資産非破壊境界

## Evidenceの区別

Gate Bでは次を相互代替しません。

1. **deterministic repository Evidence**
   - Workspace Validation
   - UI-less synthetic Scenario
   - external Evidence intake test
   - Reference artifact再生成
2. **real-provider ephemeral fixture Evidence**
   - 実PostgreSQL engine上でtool自身が作る一時fixture
3. **actual external asset Evidence**
   - #214のOracle `master_data`から観測した実物理事実

synthetic ScenarioやPostgreSQL fixtureが成功しても、actual Oracle Evidenceを取得したことにはしません。

## Gate Bを直接止めているEvidence

### A. external-observed `master_data`

GitHub Issue #214は現在、次の二層へ分かれます。

- **Evidence intake boundary: Ready**
- **actual Oracle physical facts / read-only Evidence: Blocked**

PR #288 / #300で、`internal/database/externalqueryevidence`と`tools/reference-external-query-evidence`を`main`へ収束しました。

受入境界では少なくとも次を、明示authoring済みSQL realizationと照合します。

- target / provider / `external-observed` ownership
- Contract input → physical parameter
- raw result → Semantic attribute
- native source fingerprint
- physical facts provenance
- effective read-only mechanism / Evidence参照

SQL realizationがなければEvidenceから生成・補完せず停止します。

残るactual external Evidenceは次です。

- 実object / parameter / raw result identity
- native realization source
- Semantic Customerへのexplicit mapping
- provenance
- 実効的なread-only境界
- Oracle driverでの実接続
- 1件 / 0件 / Execution Error

未観測のOracle object名やcolumn名をTEI側から推測しません。

### B. header-detail実PostgreSQL

`tools/reference-postgres-header-detail-runtime`のfixture safety boundaryは**Ready**です。

PR #294 / #300で次を固定しました。

- `public.order_lines`が既に存在すればfail closedで停止する
- 既存tableをDROPしてfixtureへ置換しない
- overrideを設けない
- tool自身がCREATEしたfixtureだけをcleanupする
- `evidenceNature = real-postgres-ephemeral-fixture`を明示する

残るのは、専用の空PostgreSQL Databaseでtoolを実行し、2件 / 0件 / Execution Error、Semantic mapping、support column非露出を実provider Evidenceとして確認することです。

### C. Reference golden

GitHub Issue #255でWorld / Plugin Lock / 必要なDurable Runを正規経路から再生成します。

`reference-golden`や`tei deployment-start`を使用し、fingerprintや生成物を手編集して合わせません。

Durable Runは保存pathへ直接redirectせず、一時ファイルへの正規生成が成功した後に置換します。

### D. 通常Go runner

少なくとも次を通常Go runnerで実行します。

```bash
go test ./internal/driver \
  -run '^TestOrderEntryGateBScenarioConnectsMajorBoundariesWithoutUI$' \
  -count=1

go test ./internal/database/externalqueryevidence \
  ./tools/reference-external-query-evidence
```

repository revisionとEvidenceの性質を記録します。

可能な環境では`go test ./...`も実行します。

## Observationとの関係

Command / ObservationはADR-0023に従いv1境界が成立し、GitHub Issue #266は完了として閉じました。

production Driverの`ExecuteObservationRead`は、Commandとは別の明示read境界としてread identity / Evidence nature / resultを保持します。

Observation設計未決はGate B blockerではありません。

## GitHub Actionsとの関係

GitHub Actionsによる継続的な再検証は、A〜DのProduct Evidenceと分けます。

Actions停止そのものを意味設計Gapへ読み替えず、逆に未実行をgreenとも扱いません。再開後のActionsはregression検知の継続経路として扱います。

## Gate B通過後

A〜Dを満たしてGate BをReadyと判定した後、すぐに「最新Driver」を暗黙のWeb baselineにはしません。

[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](../GATE_B_TO_C_HANDOVER_CONTRACT.md) / `ISSUE-000030` / #291に従い、その時点のrepository revision、Canonical identity、Driver Scenario、Gate B EvidenceをHandover Baselineとして固定してから第一Web Targetへ渡します。

## 完了条件

- [x] `GATE_B_READINESS_AUDIT.md`を主要Driver境界へ同期した
- [x] Driver / Reference / Conformanceを`main`へ収束した（PR #285）
- [x] 各Gate B条件をReady / Partial / Blocked / Pendingへ分類した
- [x] Observationのv1境界をADR-0023へ固定し、#266を完了した
- [x] 最終UI-less Scenarioの構造を同じReference World上で接続した
- [x] external Evidence intake boundaryを実装した（PR #288 / #300）
- [x] PostgreSQL fixture safetyをfail closedへした（PR #294 / #300）
- [x] Gate B Evidence Runbookを`main`へ置いた（PR #296 / #300）
- [x] Gate B→C Handover契約を定義した（#291）
- [ ] #214 actual Oracle物理 / read-only Evidenceを得る
- [ ] 専用空DBでheader-detail実PostgreSQL Evidenceを得る
- [ ] #255 Reference goldenを正規再生成する
- [ ] 最終UI-less Scenarioとexternal Evidence intakeを通常Go runnerで実行する
- [ ] A〜D取得後にnegative caseを再監査する
- [ ] Gate BをReady / Not Readyで最終再判定する
- [ ] Gate B Ready時の第一Handover Baselineを発行する

## 非目標

- Gate B前に第一Web Targetを本格実装すること
- CIをこのIssueから再有効化すること
- external `master_data`の物理情報を推測すること
- external EvidenceをCanonical Worldへ昇格すること
- synthetic / provider fixture Evidenceをactual external Evidenceとして扱うこと
- Observationを汎用Workflow / Event Sourcing基盤へ一般化すること
- Gate BのためにEditor / AI / 第二Targetを先行させること

## 関連

- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000022-external-master-data-physical-query-evidence.md`
- `plan/issue/000024-keyless-relation-physical-storage.md`
- `plan/issue/000025-reference-generated-artifacts-refresh.md`
- `plan/issue/000030-gate-b-to-c-handover-baseline.md`
- GitHub #69 / #70 / #214 / #244 / #255 / #270 / #273 / #291
- PR #281 / #285 / #288 / #294 / #296 / #300
