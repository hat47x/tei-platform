# ISSUE-000029: Gate B readinessをReference / Evidenceから再監査する

- 状態: In Progress
- GitHub Issue: #273
- 対象: `ISSUE-000006` / GitHub Issue #69
- 次段: `ISSUE-000007` / GitHub Issue #70
- Handover: `ISSUE-000030` / GitHub Issue #291

## 目的

Application Driver #69のGate B readinessを、Issue close数ではなく、**現在のReference / Conformance / 実行Evidenceが必要な種類ごとに揃っているか**で監査します。

判定内容の正本は[`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md)です。

Evidence関連文書の責務は次へ分けます。

- [`GATE_B_EVIDENCE_ACQUISITION.md`](../GATE_B_EVIDENCE_ACQUISITION.md): 何を取得するか / 何を証明するか
- [`GATE_B_EVIDENCE_RUNBOOK.md`](../GATE_B_EVIDENCE_RUNBOOK.md): 具体コマンド / 実行順 / 差分確認 / fail-closed停止条件
- [`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md): 取得済みEvidenceからのReady / Not Ready判定
- 本Issue: 4 Evidence laneの進行を監査し、再判定へ戻す

Runbook以外に具体コマンドを重複正本として持ちません。

## 現在の判定

2026-09-04時点では **Gate B: Not Ready** です。

ただし、次は未設計blockerではありません。

- browse / record / reference / header-detail / commandのApplication境界
- Contract Query signature / SQL realization / explicit mapping
- reference PatternからContract Queryへの明示invocation
- Environment / Database instance / executor Capabilityを保持するruntime境界
- keyless Relation Storage
- Validation → Authorization → Command
- Command Business Result → separate Observation read
- 最終UI-less Scenarioの構造統合
- external-observed Evidenceを非Canonicalのまま受け入れる境界
- real PostgreSQL Evidence toolの既存資産非破壊・source固定・cleanup復元境界
- deterministic Repository Evidence runner
- generated artifact check-only Evidence runner

GitHub #179 / #185とRepository `ISSUE-000018 / 000019`は実装責務としてDoneです。actual Oracle Evidenceやnormal runner Evidenceの未取得を理由に、成立済み実装Issueを再びIn Progressへ戻しません。

## Gate Bの4 Evidence lane

Gate Bでは性質の異なるEvidenceを相互代替しません。

| Lane | Repository / GitHub Issue | Evidence nature | 現在地 |
| --- | --- | --- | --- |
| A | `ISSUE-000022` / #214 | actual external asset | **Blocked** — actual Oracle facts / read-only / actual read待ち |
| B | `ISSUE-000031` / #308 | real-provider ephemeral fixture | **Ready to run** — Evidence専用空PostgreSQL DB待ち |
| C | `ISSUE-000025` / #255 | generated artifact reproducibility | **Ready to check** — clean normal Go environment待ち |
| D | `ISSUE-000032` / #310 | deterministic repository | **Ready to run** — clean checkout / normal Go runner待ち |

PR #342で`GATE_B_EVIDENCE_ACQUISITION.md`に残っていた旧3分類も、この4 Evidence natureへ同期しました。

取得結果は本Issueと`GATE_B_READINESS_AUDIT.md`で総合判定します。

## Lane A — actual external Oracle / ISSUE-000022 / #214

#214は次の三層へ分けます。

- Evidence intake boundary: **Ready**
- source-fixed acceptance receipt boundary: **Ready**
- actual Oracle physical facts / read-only / driver integration / actual read: **Blocked**

### Evidence intake

非Canonical Evidence intakeでは少なくとも次を明示SQL realizationと照合できます。

- target / provider / `external-observed` ownership
- Contract input → physical parameter
- raw result → Semantic attribute
- native source fingerprint
- provenance
- effective read-only mechanism / Evidence参照
- Canonical Workspace Validation

SQL realizationがなければEvidenceから生成・補完せず停止します。external EvidenceをCanonical Worldへ取り込みません。

### source-fixed acceptance receipt — PR #340

PR #340で、どのexternal Evidence artifactをどのclean Repository revisionで照合したかを機械可読に残す境界を追加しました。

受入toolはEvidenceを読む前にRepository revision / working treeを確認し、dirtyなら停止します。

Evidence JSON exact bytesのSHA-256を`evidenceArtifactFingerprint`として計算し、strict load / Workspace Validation / realizationとの照合後にもHEAD / working treeを再確認します。

成功receiptの主な項目:

```text
evidenceNature = external-query-evidence-acceptance
repositoryRevision
workingTreeClean = true
reproducibleRevision = true
postRunRepository.checked = true
postRunRepository.repositoryRevision = repositoryRevision
postRunRepository.revisionStable = true
postRunRepository.workingTreeClean = true
evidenceArtifactFingerprint = sha256:<hex>
target
provider
accepted = true
```

`evidenceArtifactFingerprint`はEvidence JSON artifactそのもののexact bytesを識別します。

`nativeSource.fingerprint`は観測したnative sourceと現在authoringされている実行sourceの一致を確認します。両者は相互代替しません。

receiptへphysical object / parameter / raw columns / physical type / NULL情報 / provenance本文 / read-only mechanism本文 / Evidence path / credentialを複製しません。

**acceptance receiptはactual Oracle read Evidenceではありません。**

PR #343でRepository `ISSUE-000022`もこの契約へ同期し、PR #344でRunbookの実行順も更新しました。

### 残るactual external Evidence

- 実physical object / parameter / raw result identity
- physical type / NULL扱い
- native realization source
- Semantic Customerへのexplicit mapping
- provenance
- 実効的read-only強制
- 採用Oracle driver integration
- actual Evidence artifactのsource-fixed acceptance receipt
- Workspace → actual Oracle read → Semantic result E2E

未観測のOracle object / column / SQLをTEI側から推測しません。

## Lane B — real PostgreSQL engine / ISSUE-000031 / #308

keyless Relation Storageの設計・実装は`ISSUE-000024` / #244でCompletedです。

実PostgreSQL engine Evidenceだけを本laneへ分離しています。

### Repository source固定 — PR #336

PR #336で#310 / #255 / #308のRepository source観測を`tools/internal/repositoryevidence`へtool-onlyで収束しました。

#308ではDatabaseへ接続する前にclean working treeとHEADを確認し、Provider runtime / cleanup / Database復元確認後にも再確認します。

成功Evidenceには次を要求します。

- 開始時clean
- `repositoryRevision`
- post-run確認済み
- 実行前後HEAD同一
- 終了時clean
- `reproducibleRevision = true`

source観測規則をCanonical / Driver modelへ追加しません。

### fixture前安全境界 — PR #324 / #335

`tools/reference-postgres-header-detail-runtime`はfixture作成前に次をfail closedで確認します。

1. WorkspaceをLoad / Validateする
2. Validation ErrorならDB接続前に停止する
3. PostgreSQLへ接続する
4. `current_database()`を取得する
5. `postgres` / `template0` / `template1`を拒否する
6. system / temporary領域を除くuser relation件数が0であることを確認する
7. `public.order_lines`が存在しないことを再確認する
8. すべて成立した場合だけtool-owned fixtureをCREATEする

既存relationを削除してEvidence用DBへ転用する処理やoverrideはありません。

PR #335以降、そのrunで通過した非機密な安全判定を`fixtureDatabaseSafety`へ残します。

```json
{
  "guardPassed": true,
  "databaseIdentityVerified": true,
  "databaseIdentityPersisted": false,
  "nonDefaultDatabaseVerified": true,
  "userRelationsBeforeFixture": 0,
  "fixtureObjectAbsentBeforeCreate": true
}
```

このguardだけで「Evidence専用DBである」という運用意図を完全証明したとは扱わず、実行先はEvidence専用に作成した空PostgreSQL Databaseへ限定します。

### Provider runtime Evidence

同一runで確認する対象:

- `order_id=1001` → 2件
- `order_id=9999` → 正常な0件
- tool-owned fixture欠落 → Execution Error
- Environment / Database instance / executor Capability / Contract Query target identity
- explicit Semantic mapping
- support column `order_id`非露出
- runtime parameter値 / credential / DSN / 実Database名非保存
- `evidenceNature = real-postgres-ephemeral-fixture`

### cleanup後の復元確認 — PR #341

fixtureをDROPしてmissing-table Execution Errorを確認した後、PostgreSQL catalogを再観測します。

```json
{
  "verified": true,
  "userRelationsAfterCleanup": 0,
  "fixtureObjectAbsentAfterCleanup": true
}
```

終了時にuser relationが残る場合、それがtool由来か他主体の並行変更かを推測せずfail closedします。

PR #338で`ISSUE-000031`をsource固定境界へ同期し、PR #341でcleanup確認まで正本・READMEへ反映しました。PR #344でRunbookも現在手順へ同期済みです。

残るものは**Evidence専用空PostgreSQL Databaseでのactual run**です。実装存在をProvider成功Evidenceとは扱いません。

## Lane C — generated artifact reproducibility / ISSUE-000025 / #255

#255はWorld / Plugin Lock / Durable Runの**generated artifact再現性だけ**を扱います。

### 正規個別tool

- `tools/reference-golden`: World / Plugin Lock check/write + Durable Run Plugin Lock依存check
- `tools/reference-durable-run-golden`: Durable Run全体の正規生成・完全一致check / safe write

Durable Run toolは、Workspace Validation、exact Solution Release、fixed observed deployment state Validation、`releaseplan.CompileWithDeployment`、`pluginruntime.BuildPlan`、Plugin Lock / Runtime Policy fingerprint、`deploymentdurable.StartWithRuntimePolicy`を通る正規internal API経路を利用します。

PR #326では不正Solution Canonical IDをpanicではなく通常errorでfail closedするようにしました。

### 最終完了Evidence runner — PR #331 / #334

最終完了Evidenceの正規入口は`tools/reference-generated-artifact-evidence`です。

runnerは`-write`を呼ばず、一つのrunで次を確認します。

- 実行前revision / clean
- Go toolchain identity
- `reference-golden`一致
- `reference-durable-run-golden`一致
- 実行後revision / clean
- 実行前後HEAD同一
- `reproducibleRevision = true`

PR #336以降、Repository source観測は#310 / #308と同じtool-only境界を利用します。

### 修復とEvidenceを分離する

`-write`はstaleなgenerated artifactを正規状態へ戻す**修復操作**です。`-write`直後のdirty treeでcheckが成功してもGate B完了Evidenceにはしません。

staleだった場合:

1. 正規toolで修復する
2. `git diff`で意味上の差分を確認する
3. 意図したgenerated artifactだけをcommitする
4. 新しいclean revisionで`reference-generated-artifact-evidence`を再実行する
5. 同じHEAD / clean working tree / `reproducibleRevision = true`を確認する

最初から一致している場合は形式的な`-write`を行いません。

2026-09-04の静的監査では保存済みproduction Plugin Lockと保存Durable RunのPlugin Lock fingerprintは相互整合していました。ただし、それだけで現在のCanonical Workspaceからの再現性を証明したとは扱いません。

残るものは**clean normal Go environmentでのcheck-only実run**です。

## Lane D — deterministic Repository / ISSUE-000032 / #310

正規runnerは`tools/reference-gate-b-repository-evidence`です。

PR #346以降、targeted runnerは次の三つを独立checkとして扱います。

1. 最終UI-less Scenario
2. external Evidence intake
3. `tools/internal/repositoryevidence`のsource境界Conformance

source境界Conformanceの等価コマンド:

```bash
go test ./tools/internal/repositoryevidence -count=1
```

JSONには`repository-evidence-source-boundary`として別checkを残します。

runner自身のpackageはtargeted `go test`へ含めず、共有helperのpure契約だけを検証するため、将来runner testが`run()`を呼ぶ場合にも自己再帰を作りません。

runner全体では次を一つのrunで扱います。

- 実行前repository revision
- 実行前working tree clean
- Go toolchain identity
- Canonical Workspace Validation
- 上記targeted checks
- 明示指定時だけRepository全体のbroad test
- 実行後HEAD / working tree再確認
- 小さい非Canonical JSON Evidence

PR #321以降、`reproducibleRevision=true`には実行前clean / post-run確認 / HEAD同一 / 実行後cleanを要求します。

PR #336以降、source状態観測は#255 / #308と同じtool-only packageを利用します。

PR #346でそのsource固定境界自体のpure Conformanceをtargeted Evidenceへ直接追加しましたが、PRの存在自体を#310成功Evidenceとは扱いません。

Test自体が成功してもsource revisionが実行中に変わったrun、実行後にtracked / untracked差分が残ったrun、診断用`-allow-dirty` runはGate B完了Evidenceへ採用しません。

#310のexternal Evidence intake testsが成功しても、それを#214 actual Oracle Evidenceへ格上げしません。

残るものは**clean checkout / normal Go runnerでのactual run**です。

## Observationとの関係

Command / ObservationはADR-0023に従いv1境界が成立し、GitHub #266は完了しています。

production Driverの`ExecuteObservationRead`は、Commandとは別の明示read境界としてread identity / Evidence nature / resultを保持します。

Observationの汎用化はGate B blockerではありません。

## GitHub Actionsとの関係

GitHub Actions workflowは現行Repositoryでは意図的に使用していません。

過去のrunner未割当事象を追跡した#192は現行運用では対象外として`not_planned`でclosedです。

CIサービスのgreenをLane A〜DのProduct Evidenceへ代替しません。逆に未実行をgreenとも扱いません。

将来Actionsを再導入する場合は、その時点のworkflow / runner / Repository要件で新しいCI検証Issueを作り、workflow failureがpushそのものを不必要にblockしない運用を維持します。

## 実行順

具体コマンド、安全条件、停止条件は`GATE_B_EVIDENCE_RUNBOOK.md`だけを正とします。

現在の論理順序:

1. #310 deterministic Repository Evidenceをclean normal Go runnerで取得する
2. #255 generated artifactをclean revisionでcheck-onlyする
3. staleなら正規toolで修復 → diff確認 → commit → 新clean revisionで#255を再checkする
4. #308をEvidence専用空PostgreSQL Databaseで実行する
5. #214 actual physical facts / provenance / read-only根拠を観測する
6. 観測事実から必要なOracle realizationを明示authoringする
7. #214 external Evidenceをclean revisionで受理しsource-fixed acceptance receiptを取得する
8. read-only Oracle connection + 採用driverでactual read → explicit Semantic mappingを確認する
9. Lane A〜DをGate B監査へ対応付ける
10. negative caseを再監査する
11. Gate BをReady / Not Readyで再判定する
12. Readyなら#291 Handover Baselineを固定する
13. その後に第一Web Target #70 / Gate C #270へ進む

#214に必要な実外部事実が得られない場合はBlockedとして保持し、未観測情報を埋めて先へ進みません。

## Gate B通過後

Lane A〜Dを満たしGate BをReadyと判定した後、すぐに「最新Driver」を暗黙のWeb baselineにはしません。

[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](../GATE_B_TO_C_HANDOVER_CONTRACT.md) / `ISSUE-000030` / #291に従い、そのReady revision、Canonical identity、Driver Scenario、Gate B EvidenceをHandover Baselineとして固定してから第一Web Target #70へ渡します。

Application / Driverの意味が後から変わった場合は既存baselineへ黙って上書きせず、影響するGate B Evidenceを再確認して新しいbaselineを発行します。

## 完了条件

- [x] `GATE_B_READINESS_AUDIT.md`を主要Driver境界へ同期した
- [x] 各Gate B条件をReady / Partial / Blocked / Pendingへ分類した
- [x] Observationのv1境界をADR-0023へ固定し、#266を完了した
- [x] 最終UI-less Scenarioの構造を同じReference World上で接続した
- [x] external Evidence intake boundaryを実装した
- [x] #214 acceptanceをsource-fixed receipt化した（PR #340）
- [x] #214 Repository正本をreceipt後へ同期した（PR #343）
- [x] keyless Relation Storage実装を#244でCompletedへ収束した
- [x] real PostgreSQL Evidenceを`ISSUE-000031` / #308へ分離した
- [x] #308 fixture前安全判定をEvidenceへ残した（PR #335）
- [x] #308 Repository source固定を#310 / #255と共通化した（PR #336）
- [x] #308 cleanup後空状態再確認を追加した（PR #341）
- [x] generated artifact Evidenceを`ISSUE-000025` / #255へ分離した
- [x] #255 final check-only Evidence runnerを追加した（PR #331 / #334）
- [x] deterministic Repository Evidenceを`ISSUE-000032` / #310へ分離した
- [x] #310 runnerで実行前後のsource不変性をfail closedにした（PR #315 / #321）
- [x] #310 / #255 / #308のsource観測をtool-only共有境界へ収束した（PR #336）
- [x] #310 targeted runnerへsource境界Conformanceを追加した（PR #346）
- [x] Evidence Acquisitionを4 Evidence natureへ同期した（PR #342）
- [x] Runbookを#214 receipt / #308 cleanup後へ同期した（PR #344）
- [x] Gate B→C Handover契約を定義した（#291）
- [ ] #310をcleanな通常Go runnerで実行する
- [ ] #255をclean revisionでcheck-onlyし、staleなら修復 → commit → clean revision再checkする
- [ ] #308をEvidence専用の空PostgreSQL Databaseで実行する
- [ ] #214 actual Oracle物理 / read-only / driver integration / actual read Evidenceを得る
- [ ] Lane A〜D取得後にnegative caseを再監査する
- [ ] Gate BをReady / Not Readyで最終再判定する
- [ ] Gate B Ready時の第一Handover Baselineを発行する

## 非目標

- Gate B前に第一Web Targetを本格実装すること
- CIを本Issueから再有効化すること
- external `master_data`の物理情報を推測すること
- external EvidenceをCanonical Worldへ昇格すること
- acceptance receiptをactual external readとして扱うこと
- synthetic / generated / provider fixture Evidenceをactual external Evidenceとして扱うこと
- 成立済み実装IssueをEvidence待ちで再びIn Progressへ戻すこと
- generated artifactとdeterministic Repository Evidenceを同じlaneへ畳むこと
- dirty working treeや実行中にsourceが変わったrunを完了Evidenceとして採用すること
- cleanup後にuser relationが残る#308 runを成功Evidenceとして採用すること
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
- `plan/issue/000030-gate-b-to-c-handover.md`
- `plan/issue/000031-gate-b-postgres-header-detail-evidence.md`
- `plan/issue/000032-gate-b-deterministic-repository-evidence.md`
- GitHub #69 / #70 / #192 / #214 / #244 / #255 / #266 / #273 / #291 / #308 / #310
- PR #315 / #318 / #321 / #323 / #324 / #326 / #331 / #334 / #335 / #336 / #340 / #341 / #342 / #343 / #344 / #346
