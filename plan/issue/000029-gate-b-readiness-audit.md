# ISSUE-000029: Gate B readinessをReference / Evidenceから再監査する

- 状態: In Progress
- GitHub Issue: #273
- 対象: `ISSUE-000006` / GitHub Issue #69
- 次段: `ISSUE-000007` / GitHub Issue #70
- Handover: `ISSUE-000030` / GitHub Issue #291

## 背景

Application Driverの主要な意味境界と最終UI-less Scenario構造は成立しています。

Gate B readinessは「どの実装がまだbranchに残っているか」ではなく、**必要な種類のEvidenceが現在のrevisionに対して揃っているか**で監査します。

判定内容の正本は[`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md)に置きます。

Evidence取得の責務は次へ分けます。

- [`GATE_B_EVIDENCE_ACQUISITION.md`](../GATE_B_EVIDENCE_ACQUISITION.md): 何を取得するか / 何を証明するか
- [`GATE_B_EVIDENCE_RUNBOOK.md`](../GATE_B_EVIDENCE_RUNBOOK.md): 具体コマンド / 実行順 / 差分確認 / fail-closed停止条件
- 本Issue: 4 Evidence laneの進行を監査し、Gate B再判定へ戻す

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
- external Evidence受入れをclean source revision / artifact fingerprintへ固定する境界
- real PostgreSQL Evidence toolの既存資産非破壊境界
- PostgreSQL fixture作成前 / cleanup後の安全状態をEvidence化する境界
- deterministic Repository Evidenceを一括実行するrunner
- World / Plugin Lock / Durable Runを正規生成経路から照合するtool
- generated artifact完了条件を一つのcheck-only Evidence runnerへ束ねる境界

GitHub #179 / #185とRepository `ISSUE-000018 / 000019`は実装責務としてDoneです。actual Oracle Evidenceやnormal runner Evidenceの未取得を理由に、成立済み実装Issueを再びIn Progressへ戻しません。

## Gate Bの4 Evidence lane

Gate Bでは性質の異なるEvidenceを相互代替しません。

| Lane | Repository / GitHub Issue | Evidence nature | 現在地 |
| --- | --- | --- | --- |
| A | `ISSUE-000022` / #214 | actual external asset | Blocked — actual Oracle facts待ち。source-fixed acceptance receipt境界はReady |
| B | `ISSUE-000031` / #308 | real-provider ephemeral fixture | Ready to run — source固定・fixture前後安全境界はReady、Evidence専用空PostgreSQL DB待ち |
| C | `ISSUE-000025` / #255 | generated artifact reproducibility | Ready to check — check-only Evidence runnerはReady、clean normal Go environment待ち |
| D | `ISSUE-000032` / #310 | deterministic repository | Ready to run — clean checkout / normal Go runner待ち |

取得結果は本Issueと`GATE_B_READINESS_AUDIT.md`で総合判定します。

## Lane A — actual external Oracle / ISSUE-000022 / #214

#214は次の三層へ分けます。

- Evidence intake boundary: **Ready**
- source-fixed acceptance receipt boundary: **Ready**
- actual Oracle physical facts / read-only / driver integration / actual read: **Blocked**

非Canonical Evidence intakeでは少なくとも次を明示SQL realizationと照合できます。

- target / provider / `external-observed` ownership
- Contract input → physical parameter
- raw result → Semantic attribute
- native source fingerprint
- provenance
- effective read-only mechanism / Evidence参照
- Canonical Workspace Validation

SQL realizationがなければEvidenceから生成・補完せず停止します。

PR #340以降、`tools/reference-external-query-evidence`はEvidence本文を読む前にclean Repository revisionを要求し、照合後にもHEAD / working treeを再確認します。

成功時は、受入れたEvidence JSON exact bytesのSHA-256を含む小さいacceptance receiptを出します。

- `evidenceNature = external-query-evidence-acceptance`
- `repositoryRevision`
- `reproducibleRevision = true`
- post-run Repository state
- `evidenceArtifactFingerprint = sha256:<hex>`
- target / provider
- `accepted = true`

receiptへphysical object、parameter / raw columns、provenance本文、read-only mechanism本文、Evidence file path、credential / secretを複製しません。

`evidenceArtifactFingerprint`は受入れたEvidence artifactのexact bytes、`nativeSource.fingerprint`は現在authoring済みSQL本文との一致をそれぞれ追跡します。相互代替しません。

**acceptance receiptはactual Oracle read Evidenceではありません。**

残るactual external Evidence:

- 実object / parameter / raw result identity
- native realization source
- Semantic Customerへのexplicit mapping
- provenance
- clean revisionでの実acceptance receipt取得
- 実効的read-only強制
- Oracle driver integration
- 1件 / 0件 / Execution Error
- Workspace → actual Oracle read → Semantic result E2E

未観測のOracle object / column / SQLをTEI側から推測しません。

## Lane B — real PostgreSQL engine / ISSUE-000031 / #308

keyless Relation Storageの設計・実装は`ISSUE-000024` / #244でCompletedです。

実PostgreSQL engine Evidenceだけを本laneへ分離しています。

PR #294 / #300 / #306 / #324 / #335 / #336 / #341により、`tools/reference-postgres-header-detail-runtime`は次の順でfail closedに確認します。

1. Repository revision / working tree cleanを確認する
2. dirtyならDB接続前に停止する
3. WorkspaceをLoad / Validateする
4. Validation ErrorならDB接続前に停止する
5. PostgreSQLへ接続する
6. `postgres` / `template0` / `template1`を拒否する
7. system / temporary領域を除くuser relation件数が0であることを確認する
8. `public.order_lines`が存在しないことを再確認する
9. すべて成立した場合だけtool-owned fixtureをCREATEする
10. 2件 / 0件のProvider runtime Evidenceを確認する
11. tool-owned fixtureをcleanupする
12. fixture欠落をExecution Errorとして確認する
13. system / temporary領域を除くuser relationが0件へ戻ったことを再確認する
14. `public.order_lines`が存在しないことを再確認する
15. Repository HEAD / working treeを再確認する
16. Databaseまたはsource状態が期待状態へ戻っていなければ成功JSONを出さず停止する

既存relationを削除してEvidence用DBへ転用する処理やoverrideはありません。

このguardだけで「Evidence専用DBという運用意図」を証明したとは扱わず、実行先はEvidence専用に作成した空PostgreSQL Databaseへ限定します。

fixture前安全状態は`fixtureDatabaseSafety`、cleanup後の空状態は`fixtureCleanupSafety`として、Database名 / DSN / credentialを含まない非機密metadataでEvidence化します。

残る実Evidence:

- `repositoryRevision` / `reproducibleRevision = true`
- 実行前後でHEAD同一 / working tree clean
- `fixtureDatabaseSafety.guardPassed = true`
- fixture前user relation = 0
- `order_id=1001` → 2件
- `order_id=9999` → 正常な0件
- tool-owned fixture欠落 → Execution Error
- `fixtureCleanupSafety.verified = true`
- cleanup後user relation = 0
- cleanup後`public.order_lines`不在
- Environment / Database instance / executor Capability / Contract Query target identity
- explicit Semantic mapping
- support column非露出
- runtime parameter値 / credential / DSN / Database名非保存
- `evidenceNature = real-postgres-ephemeral-fixture`

## Lane C — generated artifact reproducibility / ISSUE-000025 / #255

#255はWorld / Plugin Lock / Durable Runの**generated artifact再現性だけ**を扱います。

正規tool:

- `tools/reference-golden`: World / Plugin Lock check/write + Durable RunのPlugin Lock依存チェック
- `tools/reference-durable-run-golden`: Durable Run全体の正規生成・完全一致check / safe write
- `tools/reference-generated-artifact-evidence`: clean revisionから両golden checkとsource revision不変性を束ねる完了Evidence runner

Durable Run toolは、Workspace Validation、exact Solution Release、fixed observed deployment state Validation、`releaseplan.CompileWithDeployment`、`pluginruntime.BuildPlan`、Plugin Lock / Runtime Policy fingerprint、`deploymentdurable.StartWithRuntimePolicy`を通る正規internal API経路を利用します。

PR #326では`-solution`のCanonical IDを`ParseCanonicalID`で検証し、不正入力をpanicではなく通常errorでfail closedするようにしました。

### 修復とEvidenceを分離する

`-write`はstaleなgenerated artifactを正規状態へ戻す**修復操作**です。`-write`直後のdirty treeでcheckが成功しても、Gate B完了Evidenceにはしません。

PR #331以降、最終Evidenceは`reference-generated-artifact-evidence`へ束ねます。このrunnerは`-write`を呼びません。

staleだった場合は、

1. check-only runnerでstaleを検出する
2. 個別golden toolで必要なartifactだけ修復する
3. `git diff`で意味上の差分を確認する
4. 意図したgenerated artifactだけをcommitする
5. 新しいclean revisionでEvidence runnerを再実行する
6. `reproducibleRevision = true`を確認する

という順にします。

最終Evidence候補には次をすべて要求します。

- final check開始時にworking tree clean
- `reference-golden`一致
- `reference-durable-run-golden`一致
- final check実行前後でHEAD同一
- final check終了後もworking tree clean
- `reproducibleRevision = true`

2026-09-04の静的監査では保存済みproduction Plugin Lockと保存Durable RunのPlugin Lock fingerprintは相互整合していました。ただし、それだけで現在のCanonical Workspaceからの再現性を証明したとは扱いません。

## Lane D — deterministic Repository / ISSUE-000032 / #310

正規runnerは`tools/reference-gate-b-repository-evidence`です。

runnerは次を一つのrunで扱います。

- 実行前repository revision
- 実行前working tree clean
- Go toolchain identity
- Canonical Workspace Validation
- 最終UI-less Scenario
- external Evidence intake test
- 明示指定時だけRepository全体のbroad test
- 実行後HEAD / working tree再確認
- 小さい非Canonical JSON Evidence

PR #321 / #336以降、Repository source状態の観測は`tools/internal/repositoryevidence`へ収束しています。

`reproducibleRevision=true`には次をすべて要求します。

- 実行前clean
- 実行後確認済み
- HEAD同一
- 実行後clean

Test自体が成功してもsource revisionが実行中に変わったrun、実行後にtracked / untracked差分が残ったrun、診断用`-allow-dirty` runはGate B完了Evidenceへ採用しません。

## Observationとの関係

Command / ObservationはADR-0023に従いv1境界が成立し、GitHub #266は完了しています。

production Driverの`ExecuteObservationRead`は、Commandとは別の明示read境界としてread identity / Evidence nature / resultを保持します。

Observationの汎用化はGate B blockerではありません。

## GitHub Actionsとの関係

GitHub Actions workflowは現行Repositoryでは意図的に使用していません。

過去のrunner未割当事象を追跡した#192は現行運用では対象外として`not_planned`でclosedです。

CIサービスのgreenをLane A〜DのProduct Evidenceへ代替しません。逆に未実行をgreenとも扱いません。

将来Actionsを再導入する場合は、その時点のworkflow / runner / Repository要件で新しいCI検証Issueを作り、workflow failureがpushそのものを不必要にblockしない運用を維持します。

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
- [x] external Evidence受入れをclean revision / Evidence artifact fingerprintへ固定するreceipt境界を実装した（PR #340）
- [x] keyless Relation Storage実装を#244でCompletedへ収束した
- [x] real PostgreSQL Evidenceを`ISSUE-000031` / #308へ分離した
- [x] generated artifact Evidenceを`ISSUE-000025` / #255へ分離した
- [x] deterministic Repository Evidenceを`ISSUE-000032` / #310へ分離した
- [x] #310 runnerで実行前後のsource不変性をfail closedにした（PR #315 / #321 / #336）
- [x] #255用のWorld / Plugin Lock / Durable Run専用toolを整備した（PR #318 / #323）
- [x] #255完了条件をcheck-only Evidence runnerへ集約した（PR #331）
- [x] #308で既定DB / 既存user relationをfixture先として拒否した（PR #324）
- [x] #308 fixture前安全判定を非機密Evidenceへ残す境界を実装した（PR #335）
- [x] #308 Provider Evidenceをclean Repository revisionへ固定した（PR #336）
- [x] #308 cleanup後にuser relation 0件 / fixture不在を再確認する境界を実装した（PR #341）
- [x] Durable Run goldenの不正Canonical IDをpanicではなくfail-closed errorにした（PR #326）
- [x] Evidence Acquisition / Runbook / Readiness Auditの権威関係を分離した
- [x] Gate B→C Handover契約を定義した（#291）
- [ ] #310をcleanな通常Go runnerで実行する
- [ ] #255をclean revisionのEvidence runnerで実行し、staleなら修復 → commit → clean revision再checkする
- [ ] #308をEvidence専用の空PostgreSQL Databaseで実行し、source固定 / fixture前後安全 / runtime結果を実証する
- [ ] #214 actual Oracle物理Evidenceを得て、clean revisionでacceptance receiptを実取得する
- [ ] #214 read-only / Oracle driver / actual read / Semantic E2E Evidenceを得る
- [ ] Lane A〜D取得後にnegative caseを再監査する
- [ ] Gate BをReady / Not Readyで最終再判定する
- [ ] Gate B Ready時の第一Handover Baselineを発行する

## 非目標

- Gate B前に第一Web Targetを本格実装すること
- CIをこのIssueから再有効化すること
- external `master_data`の物理情報を推測すること
- external EvidenceをCanonical Worldへ昇格すること
- acceptance receiptをactual Oracle read Evidenceとして扱うこと
- synthetic / generated / provider fixture Evidenceをactual external Evidenceとして扱うこと
- 成立済み実装IssueをEvidence待ちで再びIn Progressへ戻すこと
- generated artifactとdeterministic Repository Evidenceを同じlaneへ畳むこと
- dirty working treeや実行中にsourceが変わったrunを完了Evidenceとして採用すること
- cleanup後にuser relationが残るPostgreSQL runを成功Evidenceとして採用すること
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
- GitHub #69 / #70 / #192 / #214 / #244 / #255 / #270 / #273 / #291 / #308 / #310
- PR #281 / #285 / #288 / #294 / #300 / #306 / #311 / #315 / #318 / #321 / #323 / #324 / #326 / #331 / #335 / #336 / #340 / #341