# ISSUE-000029: Gate B readinessをReference / Evidenceから再監査する

- 状態: In Progress
- GitHub Issue: #273
- 対象: `ISSUE-000006` / GitHub Issue #69
- 次段: `ISSUE-000007` / GitHub Issue #70
- Handover: `ISSUE-000030` / GitHub Issue #291

## 目的

Application Driver #69のGate B readinessを、Issueのclose数ではなく、**現在のReference / Conformance / 実行Evidenceが必要な種類ごとに揃っているか**で監査します。

判定内容の正本は[`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md)です。

Evidence関連文書の責務は次のように分けます。

- [`GATE_B_EVIDENCE_ACQUISITION.md`](../GATE_B_EVIDENCE_ACQUISITION.md): 何を取得するか / 何を証明するか
- [`GATE_B_EVIDENCE_RUNBOOK.md`](../GATE_B_EVIDENCE_RUNBOOK.md): 具体コマンド / 実行順 / 差分確認 / fail-closed停止条件
- [`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md): 取得済みEvidenceからのReady / Not Ready判定
- [`GATE_B_TO_C_HANDOVER_CONTRACT.md`](../GATE_B_TO_C_HANDOVER_CONTRACT.md): Gate B Ready後にGate Cへ渡すbaselineの固定方法
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
- actual Oracle readでraw列名 / driver提供metadataを失わないruntime境界
- real PostgreSQL Evidence toolの既存資産非破壊・source固定・Event Triggerを含むfixture前後安全境界
- deterministic Repository Evidence runner
- generated artifact check-only Evidence runner
- generated artifact repairで既存goldenを先にtruncateしないsafe replace境界

GitHub #179 / #185とRepository `ISSUE-000018 / 000019`は実装責務としてDoneです。actual Oracle Evidenceやnormal runner Evidenceの未取得を理由に、成立済み実装Issueを再びIn Progressへ戻しません。

## Gate Bの4 Evidence lane

Gate Bでは性質の異なるEvidenceを相互代替しません。

| Lane | Repository / GitHub Issue | Evidence nature | 現在地 |
| --- | --- | --- | --- |
| A | `ISSUE-000022` / #214（driver integration: `ISSUE-000033` / #371） | actual external asset | **Blocked** — actual Oracle facts / read-only / driver integration / actual read待ち |
| B | `ISSUE-000031` / #308 | real-provider ephemeral fixture | **Ready to run** — Evidence専用空PostgreSQL DB待ち |
| C | `ISSUE-000025` / #255 | generated artifact reproducibility | **Ready to check** — clean normal Go environment待ち |
| D | `ISSUE-000032` / #310 | deterministic repository | **Ready to run** — clean checkout / normal Go runner待ち |

`ISSUE-000033` / #371はLane Aの具体driver / Host assemblyを確定する子Issueであり、第五Evidence laneではありません。

取得結果は本Issueと`GATE_B_READINESS_AUDIT.md`で総合判定します。

## Lane A — actual external Oracle / ISSUE-000022 / #214

#214は次の四層へ分けます。

- Evidence intake boundary: **Ready**
- source-fixed acceptance receipt boundary: **Ready**
- actual-read観測metadata保持境界: **Ready**
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

### actual read時のdriver観測 — PR #364 / #366

Oracle Runner固有Payloadはactual read時に次を保持できます。

- `database/sql`へ渡したnamed parameterの名前
- 実際に返されたraw result列名
- driverが提供した場合の`databaseTypeName`
- driverが提供した場合の`nullable`
- driverが提供した場合の`length`
- driverが提供した場合の`precision` / `scale`

未提供metadataをfalse / 0 / Oracle慣習値へ推測しません。Go側の`ScanType`やraw row値を通常Evidenceへ複製しません。

このruntime metadataは、provenance付きexternal Evidenceの`PhysicalType` / `Nullable`を置き換える第二の物理正本ではありません。実Oracle / 採用driverを観測する前に`PhysicalType`と`DatabaseTypeName`の文字列一致規則を先行固定しません。

### actual Oracle driver integration — ISSUE-000033 / #371

ADR-0007に従い、具体driverの選択・登録、DSN、credential、TLS / TCPS、Wallet、connection設定はHost / Reference assembly側の責務として扱います。Canonical modelやOracle Provider Coreへ具体driverを固定しません。

2026-09-04時点では`go-ora/v2`が現行Go 1.23.2と整合する第一**検証候補**ですが、採用決定ではありません。PR #374のupstream静的監査も、driverに必要な`database/sql`境界が存在することの確認であり、actual Oracle成功Evidenceではありません。

実Oracleのversion / authentication / TCPS・Wallet / datatype / NULL / read-only方式 / Host制約を観測してからdriverを確定します。

### 残るactual external Evidence

- 実physical object / parameter / raw result identity
- physical type / NULL扱い
- native realization source
- Semantic Customerへのexplicit mapping
- provenance
- 実効的read-only強制
- `ISSUE-000033` / #371での採用Oracle driver integration
- actual readでraw列名 / driver提供metadataを観測した結果
- actual Evidence artifactのsource-fixed acceptance receipt
- Workspace → actual Oracle read → Semantic result E2E

未観測のOracle object / column / SQL / datatypeをTEI側から推測しません。

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

### fixture前安全境界 — PR #324 / #335 / #365

`tools/reference-postgres-header-detail-runtime`はfixture作成前に次をfail closedで確認します。

1. WorkspaceをLoad / Validateする
2. Validation ErrorならDB接続前に停止する
3. PostgreSQLへ接続する
4. `current_database()`を取得する
5. `postgres` / `template0` / `template1`を拒否する
6. system / temporary領域を除くuser relation件数が0であることを確認する
7. Event Trigger件数が0であることを確認する
8. `public.order_lines`が存在しないことを再確認する
9. すべて成立した場合だけtool-owned fixtureをCREATEする

Event Triggerはuser relationとは別にDatabase単位で存在でき、fixtureのDDLへ介入し得るため、disabledを含め既存なら拒否します。

既存relationやEvent Triggerを削除・無効化してEvidence用DBへ転用する処理やoverrideはありません。

PR #335 / #365以降、そのrunで通過した非機密な安全判定を`fixtureDatabaseSafety`へ残します。

```json
{
  "guardPassed": true,
  "databaseIdentityVerified": true,
  "databaseIdentityPersisted": false,
  "nonDefaultDatabaseVerified": true,
  "userRelationsBeforeFixture": 0,
  "eventTriggersBeforeFixture": 0,
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

### cleanup後の復元確認 — PR #341 / #365

fixtureをDROPしてmissing-table Execution Errorを確認した後、PostgreSQL catalogを再観測します。

```json
{
  "verified": true,
  "userRelationsAfterCleanup": 0,
  "eventTriggersAfterCleanup": 0,
  "fixtureObjectAbsentAfterCleanup": true
}
```

終了時にuser relationまたはEvent Triggerが存在する場合、それがtool由来か他主体の並行変更かを推測せずfail closedします。

残るものは**Evidence専用空PostgreSQL Databaseでのactual run**です。実装存在をProvider成功Evidenceとは扱いません。

## Lane C — generated artifact reproducibility / ISSUE-000025 / #255

#255はWorld / Plugin Lock / Durable Runの**generated artifact再現性だけ**を扱います。

### 正規個別tool

- `tools/reference-golden`: World / Plugin Lock check/write + Durable Run Plugin Lock依存check
- `tools/reference-durable-run-golden`: Durable Run全体の正規生成・完全一致check / safe write

PR #367以降、World / Plugin Lockの修復writeも既存goldenを先にtruncateせず、同一directoryの一時fileへ全量write / Sync / Close後にRenameするsafe replaceへ揃っています。

safe replaceの目的は修復途中の既存artifact破損リスクを下げることです。safe replaceや`-write`の成功そのものをGate B完了Evidenceへ格上げしません。

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

PR #351以降、開始時working treeがcleanでも、post-run確認が終わるまでは`reproducibleRevision=false`を維持します。Go toolchain取得、Workspace Load / Validation、targeted test等の途中で失敗したJSONを、再現可能な成功Evidenceへ誤読しません。

Test自体が成功してもsource revisionが実行中に変わったrun、実行後にtracked / untracked差分が残ったrun、診断用`-allow-dirty` runはGate B完了Evidenceへ採用しません。

#310のexternal Evidence intake testsが成功しても、それを#214 actual Oracle Evidenceへ格上げしません。

残るものは**clean checkout / normal Go runnerでのactual run**です。

## 4 laneの最終revisionを揃える — PR #354

4 laneがそれぞれsource-fixedであっても、別々のrevisionで成功したEvidenceを寄せ集めてGate B Readyにはしません。

Gate Bの最終Ready判定では、一つのcandidate repository revision `R`を固定し、4 laneの完了判定に用いるEvidenceをRへ揃えます。

```text
#214 acceptance + actual read ─┐
#308 Provider Evidence        ─┼─ repositoryRevision = R
#255 generated artifact       ─┤
#310 deterministic Repository ─┘
                               ↓
                        Gate B Ready for R
                               ↓
                        Handover Baseline R
```

外部観測資料そのものはRより前に取得されていて構いません。ただし、その資料を現在authoringへ照合する#214 acceptance receiptとactual readはRへ結び直します。

Rと異なるrevisionの完了Evidenceしかないlaneは、「意味は変わっていないはずだ」と推測して流用せず、Rで再実行・再受入れします。

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

Evidence取得は二段階に分けます。

### Phase 1 — 事前確認 / sourceを確定する

1. #310をcleanな通常Go runnerで実行し、deterministicな問題を先に露出させる
2. #255をcheck-onlyし、staleなら正規toolで修復 → 差分確認 → commitする
3. #308をEvidence専用の空PostgreSQL Databaseで事前実行し、Provider / fixture safety / cleanup復元上の問題を露出させる
4. #214のactual Oracle physical facts / provenance / read-only Evidenceを取得する
5. `ISSUE-000033` / #371で実Oracle条件に基づくdriver / Host assemblyを確定する
6. 観測事実を根拠に必要なOracle realization / driver assemblyを明示authoringする
7. #214 acceptance / Oracle driver / actual readを事前確認し、不足を修復する
8. Gate Bに必要なsource / generated artifact変更をすべてcommitする

この段階で得た成功JSONも、その後sourceが変わった場合は最終完了Evidenceとして流用しません。

### Phase 2 — 最終収束 / candidate revision Rを固定する

1. cleanなcandidate revision Rを固定する
2. Rで#310 `reference-gate-b-repository-evidence`を実行する
3. Rで#255 `reference-generated-artifact-evidence`をcheck-only実行する
4. Rで#308 PostgreSQL Provider Evidenceを実行する
5. Rで#214 external Evidence acceptance receiptを再取得し、Rのruntime / realization / driver assemblyでactual Oracle readを実行する
6. 4 laneすべての完了EvidenceがRを参照していることを確認する
7. secret / read-only / 0件 / Execution Error / Observation期待不一致 / source revision不変 / Event Trigger残存 / Oracle未提供metadata推測等のnegative caseを再監査する
8. Rを対象にGate BをReady / Not Readyで再判定する
9. Readyなら#291でRをHandover Baselineとして固定する
10. その後に第一Web Target #70 / Gate C #270へ進む

Phase 2中にsource変更が必要になった場合は、その変更をcommitした新revisionをRとして選び直し、4 laneをもう一度そのrevisionへ揃えます。

#214に必要な実外部事実が得られない場合はBlockedとして保持し、未観測情報を埋めて先へ進みません。

## Gate B通過後

4 laneの完了Evidenceを同じrevision Rへ揃え、Gate BをRについてReadyと判定した後も、すぐに「最新Driver」を暗黙のWeb baselineにはしません。

[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](../GATE_B_TO_C_HANDOVER_CONTRACT.md) / `ISSUE-000030` / #291に従い、R、Canonical identity、Driver Scenario、Gate B EvidenceをHandover Baselineとして固定してから第一Web Target #70へ渡します。

Application / Driverの意味が後から変わった場合は既存baselineへ黙って上書きせず、影響するGate B Evidenceを再確認し、4 laneを新しいrevisionへ揃えてから新baselineを発行します。

## 完了条件

- [x] `GATE_B_READINESS_AUDIT.md`を主要Driver境界へ同期した
- [x] 各Gate B条件をReady / Partial / Blocked / Pendingへ分類した
- [x] Observationのv1境界をADR-0023へ固定し、#266を完了した
- [x] 最終UI-less Scenarioの構造を同じReference World上で接続した
- [x] external Evidence intake boundaryを実装した
- [x] #214 acceptanceをsource-fixed receipt化した（PR #340）
- [x] #214 actual-read観測metadata保持境界を追加した（PR #364 / #366）
- [x] actual Oracle driver integrationを`ISSUE-000033` / #371へ分離した（PR #372）
- [x] keyless Relation Storage実装を#244でCompletedへ収束した
- [x] real PostgreSQL Evidenceを`ISSUE-000031` / #308へ分離した
- [x] #308 fixture前安全判定をEvidenceへ残した（PR #335）
- [x] #308 Event Trigger 0件をfixture前後の安全境界へ追加した（PR #365）
- [x] #308 Repository source固定を#310 / #255と共通化した（PR #336）
- [x] #308 cleanup後空状態再確認を追加した（PR #341 / #365）
- [x] generated artifact Evidenceを`ISSUE-000025` / #255へ分離した
- [x] #255 final check-only Evidence runnerを追加した（PR #331 / #334）
- [x] World / Plugin Lock修復writeをsafe replaceへ揃えた（PR #367）
- [x] deterministic Repository Evidenceを`ISSUE-000032` / #310へ分離した
- [x] #310 runnerで実行前後のsource不変性をfail closedにした（PR #315 / #321）
- [x] #310 / #255 / #308のsource観測をtool-only共有境界へ収束した（PR #336）
- [x] #310 targeted runnerへsource境界Conformanceを追加した（PR #346）
- [x] #310の早期失敗で`reproducibleRevision=true`にならない境界を追加した（PR #351）
- [x] Evidence Acquisitionを4 Evidence natureと直近安全境界へ同期した（PR #342 / #375）
- [x] Runbookを#214 metadata / #308 Event Trigger / #255 safe replace後へ同期した（PR #369）
- [x] Gate B→C Handover契約を定義した（#291）
- [x] 4 laneの最終Evidenceを単一revisionへ揃える契約を定義した（PR #354）
- [ ] #310をcleanな通常Go runnerで事前実行する
- [ ] #255をclean revisionで事前checkし、staleなら修復 → commitする
- [ ] #308をEvidence専用の空PostgreSQL Databaseで事前実行する
- [ ] #214 actual Oracle物理 / read-only / driver integration / actual read Evidenceを得る
- [ ] Gate Bに必要なsource / generated artifact変更をすべてcommitする
- [ ] candidate revision Rで#310 / #255 / #308 / #214を再実行・再受入れする
- [ ] 4 laneの完了EvidenceがすべてRを参照していることを確認する
- [ ] negative caseを再監査する
- [ ] Rを対象にGate BをReady / Not Readyで最終再判定する
- [ ] Gate B Ready時の第一Handover Baseline Rを発行する

## 非目標

- Gate B前に第一Web Targetを本格実装すること
- CIを本Issueから再有効化すること
- external `master_data`の物理情報やOracle datatypeを推測すること
- external EvidenceをCanonical Worldへ昇格すること
- acceptance receiptをactual external readとして扱うこと
- driverが提供していないOracle metadataを推測すること
- synthetic / generated / provider fixture Evidenceをactual external Evidenceとして扱うこと
- 成立済み実装IssueをEvidence待ちで再びIn Progressへ戻すこと
- generated artifactとdeterministic Repository Evidenceを同じlaneへ畳むこと
- `-write`やsafe replace成功をgenerated artifact完了Evidenceへ格上げすること
- dirty working treeや実行中にsourceが変わったrunを完了Evidenceとして採用すること
- cleanup後にuser relationまたはEvent Triggerが残る#308 runを成功Evidenceとして採用すること
- 一致済みgenerated artifactを形式的に書き換えること
- Observationを汎用Workflow / Event Sourcing基盤へ一般化すること
- Gate BのためにEditor / AI / 第二Targetを先行させること
- `ISSUE-000033` / #371を第五Evidence laneとして扱うこと
- 異なるrevisionで得た4 laneの成功Evidenceを一つのReady revisionへ読み替えること

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
- `plan/issue/000033-gate-b-oracle-driver-integration.md`
- GitHub #69 / #70 / #192 / #214 / #244 / #255 / #266 / #273 / #291 / #308 / #310 / #371
- PR #315 / #318 / #321 / #323 / #324 / #326 / #331 / #334 / #335 / #336 / #340 / #341 / #342 / #343 / #344 / #346 / #351 / #353 / #354 / #364 / #365 / #366 / #367 / #369 / #370 / #372 / #373 / #374 / #375
