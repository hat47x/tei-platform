# Gate B Evidence Acquisition

## 目的

この文書は、Gate Bの残件を「設計をさらに増やす作業」ではなく、**成立済み境界へ不足しているEvidenceを取得する作業**として分解します。

正式な合否判定は[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md) / `ISSUE-000029` / GitHub Issue #273を正とします。

具体的なコマンド、安全条件、停止条件は[`GATE_B_EVIDENCE_RUNBOOK.md`](GATE_B_EVIDENCE_RUNBOOK.md)を正とします。

この文書は「何を取得すれば各laneが完了するか」を定義します。

## 現在地

PR #285でApplication Driver / Reference data access / header-detail / Observation / 最終UI-less Scenarioの主要境界を`main`へ収束しました。

その後も次の安全境界が成立しています。

- PR #288 / #300: external-observed Evidence intake
- PR #294 / #300: PostgreSQL fixtureの既存資産非破壊
- PR #306: Invalid WorkspaceではEvidence受入・DB接続前にfail closed
- PR #309: Relation Storage実装完了とreal PostgreSQL Evidenceの責務分離

したがってGate Bを直接止めているEvidenceは、次の四系統へ固定します。

| Lane | GitHub Issue | Evidenceの性質 |
| --- | --- | --- |
| A | #214 | actual external Oracle asset Evidence |
| B | #308 | real PostgreSQL engine + tool-owned ephemeral fixture Evidence |
| C | #255 | generated Reference artifact再現Evidence |
| D | #310 | deterministic Repository / normal Go runner Evidence |

Gate B総合判定は#273、Driver全体の作業入口は#69です。

GitHub Actionsはこの4 laneとは別の継続的regression経路です。Actions停止そのものをProduct Evidence不足へ読み替えず、逆に未実行をgreenとも扱いません。

## Evidence共通原則

すべてのlaneで次を守ります。

- 未観測のexternal物理情報を名前や慣習から推測しない
- Canonical WorkspaceにValidation Errorがあれば後続Evidence取得へ進まない
- generated artifactを手編集して合わせない
- fingerprintを人手で置換しない
- credential / DSN / token / secretを通常Evidenceへ保存しない
- 保存不要なruntime parameter値を通常Evidenceへ残さない
- synthetic Evidenceを実provider / actual external Evidenceへ格上げしない
- provider fixture Evidenceをproduction既存asset Evidenceへ格上げしない
- 実行失敗を0件、NotSatisfied、denied、Business Resultへ変換しない
- Evidence artifactをCanonical meaningの第二の正本にしない

## Lane A — actual external Oracle `master_data`

関連: `ISSUE-000022` / GitHub Issue #214

### 現在地

external Evidenceを受け取る非Canonical intake boundaryはReadyです。

`internal/database/externalqueryevidence` / `tools/reference-external-query-evidence`は、Evidenceを明示authoring済み`ContractQuerySQLRealization`と照合します。

PR #306により、Canonical WorkspaceがInvalidな場合はEvidenceを受け入れる前に停止します。

一方、actual Oracleの物理事実そのものは未取得です。

### 必要な観測事実

少なくとも次を実対象から確認します。

- physical object kind / identity
- Contract input `id`に対応するphysical parameter
- parameter type / NULL上の扱い
- raw result columns
- raw column type / NULL上の扱い
- raw result → `customer.customer` Semantic attribute mapping
- native Oracle SQLまたは他realization source
- source / provenance
- 匿名化・縮約等を行った場合の加工内容
- 実効的なread-only mechanism
- 採用Oracle driverでのnamed parameter / result / NULL / error伝播

### read-only Evidence

`ownership external-observed`やSQLが`SELECT`に見えることだけではread-onlyを証明したことにしません。

実際に有効な境界をEvidenceとして残します。

例:

- read-only credential / role
- write privilegeを持たないprincipal
- session / transaction read-only設定
- read-only view / API
- 外部側の権限制御

### 完了候補

- actual physical factsとprovenanceが揃う
- 観測事実を根拠にrealizationを明示authoringできる
- Evidence intakeでauthoringとの一致を確認できる
- read-onlyを実効強制した接続で1件 / 0件 / Execution Errorを区別できる
- raw resultをexplicit mappingからSemantic Customerへ変換できる
- Environment / Database instance / executor Capability identityを保持できる

未観測の物理名をTEI側から補った場合は完了Evidenceにしません。

## Lane B — real PostgreSQL header-detail

関連: GitHub Issue #308  
実装完了境界: `ISSUE-000024` / closed GitHub Issue #244

### Evidenceの性質

これは**real PostgreSQL engine + tool-owned ephemeral fixture Evidence**です。

production既存Databaseの物理事実を観測するEvidenceではありません。

### 安全前提

PR #294 / #300 / #306により、tool自体が次をfail closedで強制します。

1. Workspace Load
2. normal Validation
3. Validation成功後にだけPostgreSQL接続
4. `public.order_lines`の既存有無を確認
5. 既存objectがあれば停止
6. tool自身がCREATEしたfixtureだけをcleanup

したがって実行先は**専用の空PostgreSQL Database**に限定します。

### 完了候補

同じrunで少なくとも次を確認します。

- `order_id=1001` → 2件
- `order_id=9999` → 正常な0件
- fixture table欠落 → Execution Error
- Environment = `development`
- Database instance = `sales_core.dev`
- executor Capability = `database.query.postgres`
- Contract Query target identity
- explicit Semantic result mapping
- Relation Storage support columnをSemantic attributeへ漏らしていない
- parameter名のみをEvidenceへ残し、値を保存していない
- `evidenceNature = real-postgres-ephemeral-fixture`

## Lane C — generated Reference artifact

関連: `ISSUE-000025` / GitHub Issue #255

### 対象

Canonical Workspaceから導出される主要generated artifact:

```text
reference/world.json
reference/workspace/.tei/plugin-lock.json
reference/order_rollout.durable-run.m0.21.start.json
```

observed stateやlegacy-auth fixtureはCanonical変更だけを理由に機械更新しません。

### 完了候補

- Workspace Validationが成功する
- Worldを正規生成経路から再生成できる
- World差分を現在のauthoring変更から説明できる
- Plugin Lockを正規生成経路から再生成できる
- fingerprint差分を説明できる
- 必要ならDurable Runを正規`deployment-start`から再生成できる
- fingerprintを手編集していない
- `go run ./tools/reference-golden`が最終的に一致する

## Lane D — deterministic Repository / normal runner

関連: GitHub Issue #310

### 対象

外部DBや外部assetを使わず、Repository checkoutとGo toolchainだけで再現できるEvidenceです。

最低限:

- Canonical Workspace Validation
- 最終UI-less Scenario
- external Evidence intake package / tool test

可能ならRepository全体の`go test ./...`も実行します。

### 完了候補

- 対象repository revisionを記録する
- Workspace Validationが成功する
- `TestOrderEntryGateBScenarioConnectsMajorBoundariesWithoutUI`が通常Go runnerで成功する
- external Evidence intake package / tool testが通常Go runnerで成功する
- syntheticなEvidence範囲を明示する
- secret / runtime parameter値をEvidenceへ残していない
- 可能なら`go test ./...`の結果を記録する

このlaneの成功だけでLane A / B / Cを完了扱いしません。

## 四lane完了後

四系統のEvidenceを取得したら、#273 / `GATE_B_READINESS_AUDIT.md`でnegative caseを含めて再監査します。

少なくとも次を再確認します。

- 同じ業務identityを境界間で明示的に継承できる
- external assetの未観測事実を推測していない
- read-only / secret境界がEvidenceから説明できる
- 0件とExecution Errorを混同しない
- Validation / Authorization不成立時にCommandへ進まない
- Business Result / Execution Error / Observationを区別する
- synthetic / provider fixture / actual external Evidenceを区別する
- generated artifactをauthoritative sourceから再現できる

Gate BがReadyになった場合に初めて、[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md) / #291に従い、そのrevisionを第一Handover Baselineとして固定します。

その後に#70第一Web Targetへ進みます。
