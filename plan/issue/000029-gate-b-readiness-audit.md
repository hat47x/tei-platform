# ISSUE-000029: Gate B readinessをReference / Evidenceから再監査する

- 状態: In Progress
- GitHub Issue: #273
- 対象: `ISSUE-000006` / GitHub Issue #69
- 次段: `ISSUE-000007` / GitHub Issue #70
- Handover: `ISSUE-000030` / GitHub Issue #291

## 背景

Application Driverの主要な意味境界とScenario構造は成立し、PR #285で`main`へ収束しました。

その後も、external Evidence intake、PostgreSQL fixture安全境界、通常runner用のEvidence入口、各Evidence laneのRepository Issue化を進めています。

Gate B readinessは「どの実装がまだbranchに残っているか」ではなく、**必要な種類のEvidenceが現在のrevisionに対して揃っているか**で監査します。

監査結果の正本は[`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md)に置きます。

Evidence取得については役割を分けます。

- [`GATE_B_EVIDENCE_ACQUISITION.md`](../GATE_B_EVIDENCE_ACQUISITION.md): 何を取得するか / 何を証明するか
- [`GATE_B_EVIDENCE_RUNBOOK.md`](../GATE_B_EVIDENCE_RUNBOOK.md): 具体的なコマンド / 実行順 / 差分確認 / fail-closed停止条件
- 本Issue: 4 Evidence laneの進行とGate B再判定へ戻す責務

## 現在の判定

2026-09-04時点では **Gate B: Not Ready** です。

ただし、次は未設計blockerではありません。

- browse / record / reference / header-detail / commandのApplication境界
- Contract Query signature / SQL realization / explicit mapping
- reference PatternからContract Queryへの明示invocation
- Environment / Database instance / executor Capabilityを保持するruntime境界
- Relation Storage
- Validation -> Authorization -> Command
- Command acknowledgement -> separate Observation read
- 最終UI-less Scenarioの構造統合
- external-observed Evidenceを非Canonicalのまま受け入れる境界
- PostgreSQL Evidence fixtureの既存資産非破壊境界
- deterministic Repository Evidenceを一括実行するrunner

PR #316で、GitHub Issue #179 / #185に対応する`ISSUE-000018 / 000019`も、設計履歴を保持したままDoneへ同期しました。actual Oracle E2Eや通常runner Evidenceの未取得を理由に、成立済み実装Issueを再びIn Progressへ戻しません。

## Gate Bの4 Evidence lane

Gate Bでは、性質の異なるEvidenceを相互代替しません。

### A. actual external Oracle Evidence

- Repository Issue: `ISSUE-000022`
- GitHub Issue: #214
- Evidence nature: actual external asset

### B. real PostgreSQL engine Evidence

- Repository Issue: `ISSUE-000031`
- GitHub Issue: #308
- Evidence nature: real-provider ephemeral fixture

### C. generated Reference artifact再現Evidence

- Repository Issue: `ISSUE-000025`
- GitHub Issue: #255
- Evidence nature: generated artifact reproducibility

### D. deterministic Repository / normal Go runner Evidence

- Repository Issue: `ISSUE-000032`
- GitHub Issue: #310
- Evidence nature: deterministic repository

この4 laneの取得結果を、本Issueと[`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md)で総合判定します。

## A. actual external Oracle Evidence — ISSUE-000022 / #214

#214は現在、次の二層へ分かれます。

- **Evidence intake boundary: Ready**
- **actual Oracle physical facts / read-only Evidence: Blocked**

PR #288 / #300で`internal/database/externalqueryevidence`と`tools/reference-external-query-evidence`を`main`へ収束し、PR #306でWorkspace Validation fail-closedも`main`へ入っています。

受入境界では、少なくとも次を明示authoring済みSQL realizationと照合します。

- target / provider / `external-observed` ownership
- Contract input -> physical parameter
- raw result -> Semantic attribute
- native source fingerprint
- physical facts provenance
- effective read-only mechanism / Evidence参照

SQL realizationがなければEvidenceから生成・補完せず停止します。Validation ErrorのあるWorkspaceもEvidence受入前に拒否します。

残るactual external Evidenceは次です。

- 実object / parameter / raw result identity
- native realization source
- Semantic Customerへのexplicit mapping
- provenance
- 実効的なread-only境界
- Oracle driverでの実接続
- 1件 / 0件 / Execution Error
- Workspace -> actual Oracle read -> Semantic resultのE2E

未観測のOracle object名やcolumn名をTEI側から推測しません。

## B. real PostgreSQL engine Evidence — ISSUE-000031 / #308

keyless Relation Storageの設計・実装は`ISSUE-000024` / #244でCompletedです。

実PostgreSQL engine上のEvidenceだけを`ISSUE-000031` / #308へ分離しています。

`tools/reference-postgres-header-detail-runtime`の安全境界は、PR #294 / #300 / #306で成立しています。

- Workspace Validation ErrorならDB接続前に停止する
- `public.order_lines`が既に存在すればfail closedで停止する
- 既存tableをDROPしてfixtureへ置換しない
- overrideを設けない
- tool自身がCREATEしたfixtureだけをcleanupする
- `evidenceNature = real-postgres-ephemeral-fixture`を明示する

残るEvidenceは、専用の空PostgreSQL Databaseで同じrunから次を確認することです。

- `order_id=1001` -> 2件
- `order_id=9999` -> 正常な0件
- tool-owned fixture欠落 -> Execution Error
- Environment / Database instance / executor Capability / Contract Query target identity
- explicit Semantic mapping
- support column非露出
- runtime parameter値 / credential非保存

このEvidenceはproduction既存Databaseや#214 actual Oracle asset Evidenceの代替ではありません。

2026-09-04現在、このセッションには`TEI_TEST_POSTGRES_DSN`が設定されていないため、実行済みとは扱いません。

## C. generated Reference artifact再現Evidence — ISSUE-000025 / #255

#255はWorld / Plugin Lock / Durable Runの**generated artifact再現性だけ**を扱います。

PR #319で、生成物を無条件に再生成するのではなく、まず`tools/reference-golden`で現在のCanonical Workspaceとの一致を確認し、staleなartifactだけを正規経路で更新する責務へ整理しました。

### 静的に確認済みの依存

現在保存されているproduction Plugin Lockから`EnvironmentLockFingerprint`と同じ規則で再計算した値は次です。

```text
74d225e985ebf68be9d0289f533d6599b7eb17e6757bb4425c1d0b08cfede220
```

保存済みDurable Runの`pluginLockFingerprint`も同じ値です。

したがって、**保存済みPlugin Lockと保存済みDurable Runの相互依存は現時点で整合しています**。

ただし、保存済みWorld / Plugin Lockそのものが現在のCanonical Workspaceから正規生成した結果と一致することは、通常Go commandで未確認です。

残る手順は次です。

1. Workspace Validation
2. `go run ./tools/reference-golden`で一致確認
3. staleな場合だけWorld / Plugin Lockを`-write`で正規生成
4. Plugin Lock依存がstaleになった場合だけDurable Runを正規`deployment-start`で再生成
5. 最後に`reference-golden`で一致確認

一致済み生成物を形式的に書き換えません。fingerprintを人手で置換しません。

## D. deterministic Repository / normal Go runner Evidence — ISSUE-000032 / #310

PR #315で一括runnerを`main`へ追加しました。

```bash
go run ./tools/reference-gate-b-repository-evidence
```

このrunnerは既定でclean working treeを要求し、一つのrunから次を記録します。

- repository revision
- working tree clean / reproducible revision
- Go version
- Canonical Workspace Validation
- 最終UI-less Scenario test
- external Evidence intake test
- 小さい非Canonical JSON Evidence

広いRepository testは明示指定時だけ追加します。

```bash
go run ./tools/reference-gate-b-repository-evidence -broad
```

診断用`-allow-dirty`は`reproducibleRevision=false`として非0終了するため、Gate B完了Evidenceには使用しません。

2026-09-04現在、このセッションにはGo toolchain自体はありますが、`github.com`を名前解決できずRepository checkoutを取得できないため、#310は未実行のままです。

## Observationとの関係

Command / ObservationはADR-0023に従いv1境界が成立し、GitHub Issue #266は完了しています。

production Driverの`ExecuteObservationRead`は、Commandとは別の明示read境界としてread identity / Evidence nature / resultを保持します。

Observation設計未決はGate B blockerではありません。

## GitHub Actionsとの関係

GitHub Actionsによる継続的な再検証は、A〜DのProduct Evidenceと分けます。

Actions停止そのものを意味設計Gapへ読み替えず、逆に未実行をgreenとも扱いません。将来再開する場合はregression検知の継続経路として扱います。

特定のCIサービスが動作していることをGate B Readyの定義へ混ぜません。

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
- [x] PostgreSQL fixture safetyをfail closedへした（PR #294 / #300 / #306）
- [x] Gate B Evidence Runbookを`main`へ置いた（PR #296 / #300）
- [x] Evidence Acquisition / Runbook / Readiness Auditの権威関係を分離した（PR #311）
- [x] real PostgreSQL Evidenceを`ISSUE-000031` / #308へ分離した
- [x] deterministic Repository Evidenceを`ISSUE-000032` / #310へ分離した
- [x] #310用一括runnerを`main`へ追加した（PR #315）
- [x] `ISSUE-000018 / 000019`を設計履歴保持のままDoneへ同期した（PR #316）
- [x] #255を一致確認優先・stale時のみ再生成へ整理した（PR #319）
- [x] 保存済みPlugin LockとDurable Runのproduction fingerprint相互整合を静的に確認した
- [x] Gate B→C Handover契約を定義した（#291）
- [ ] `ISSUE-000032` / #310をcleanな通常Go runnerで実行する
- [ ] `ISSUE-000025` / #255でCanonical WorkspaceとReference artifactを照合し、必要なartifactだけ正規更新する
- [ ] `ISSUE-000031` / #308でreal PostgreSQL engine Evidenceを取得する
- [ ] `ISSUE-000022` / #214でactual Oracle物理 / read-only / E2E Evidenceを得る
- [ ] A〜D取得後にnegative caseを再監査する
- [ ] Gate BをReady / Not Readyで最終再判定する
- [ ] Gate B Ready時の第一Handover Baselineを発行する

## 非目標

- Gate B前に第一Web Targetを本格実装すること
- CIをこのIssueから再有効化すること
- external `master_data`の物理情報を推測すること
- external EvidenceをCanonical Worldへ昇格すること
- synthetic / provider fixture Evidenceをactual external Evidenceとして扱うこと
- #244を実Provider Evidence待ちで再びIn Progressへ戻すこと
- #179 / #185をactual Oracle Evidence待ちで再びOpenへ戻すこと
- #255へprovider / external runtime Evidenceを抱え込むこと
- generated artifactとdeterministic Repository Evidenceを同じlaneへ畳むこと
- 一致済みgenerated artifactを形式的に書き換えること
- Observationを汎用Workflow / Event Sourcing基盤へ一般化すること
- Gate BのためにEditor / AI / 第二Targetを先行させること

## 関連

- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000018-database-contract-query-execution.md`
- `plan/issue/000019-reference-contract-query-invocation.md`
- `plan/issue/000022-external-master-data-physical-query-evidence.md`
- `plan/issue/000024-keyless-relation-physical-storage.md`
- `plan/issue/000025-reference-generated-artifacts-refresh.md`
- `plan/issue/000030-gate-b-to-c-handover-baseline.md`
- `plan/issue/000031-gate-b-postgres-header-detail-evidence.md`
- `plan/issue/000032-gate-b-deterministic-repository-evidence.md`
- GitHub #69 / #70 / #179 / #185 / #214 / #244 / #255 / #270 / #273 / #291 / #308 / #310
- PR #281 / #285 / #288 / #294 / #296 / #300 / #306 / #311 / #315 / #316 / #319
