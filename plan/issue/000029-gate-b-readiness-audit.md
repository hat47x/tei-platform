# ISSUE-000029: Gate B readinessをReference / Evidenceから再監査する

- 状態: In Progress
- GitHub Issue: #273
- 対象: `ISSUE-000006` / GitHub Issue #69
- 次段: `ISSUE-000007` / GitHub Issue #70
- Handover: `ISSUE-000030` / GitHub Issue #291
- Gate B blocker再監査: `ISSUE-000035` / GitHub Issue #381

## 目的

Application Driver #69のGate B readinessを、Issueのclose数ではなく、**現在のReference / Conformance / 実行Evidenceが必要な種類ごとに揃っているか**で監査します。

判定内容の正本は[`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md)です。

Evidence関連文書の責務は次のように分けます。

- [`GATE_B_EVIDENCE_ACQUISITION.md`](../GATE_B_EVIDENCE_ACQUISITION.md): 何を取得するか / 何を証明するか
- [`GATE_B_EVIDENCE_RUNBOOK.md`](../GATE_B_EVIDENCE_RUNBOOK.md): 具体コマンド / 実行順 / 差分確認 / fail-closed停止条件
- [`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md): 取得済みEvidenceからのReady / Not Ready判定
- [`GATE_B_TO_C_HANDOVER_CONTRACT.md`](../GATE_B_TO_C_HANDOVER_CONTRACT.md): Gate B Ready後にGate Cへ渡すbaselineの固定方法
- 本Issue: 必須Evidenceと継続Evidenceの進行を監査し、再判定へ戻す

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

GitHub #179 / #185とRepository `ISSUE-000018 / 000019`は実装責務としてDoneです。actual Evidenceやnormal runner Evidenceの未取得を理由に、成立済み実装Issueを再びIn Progressへ戻しません。

## Gate B Evidence契約の変遷

PR #342 / #350以降、Gate Bは性質の異なる4 Evidence natureを分離してきました。

```text
deterministic repository
generated artifact reproducibility
real-provider ephemeral fixture
actual external asset
```

PR #354では、4種類すべてをcandidate revision Rへ揃える契約を一度採用しました。

その後PR #379のProduct Gap Auditで、「actual external OracleをGate B必須blockerとする妥当性」をGate B本来の目的から再監査する必要が明確になりました。

`ISSUE-000035` / #381で、Milestone 3、Product Requirements、V1 Reference Plan、Handoverのunresolved契約を再監査した結果、**actual external Oracle EvidenceはGate Bの必須完了条件から外し、継続Evidenceとして残す**ことに変更しました。

これはactual external asset EvidenceというEvidence natureを削除する判断ではありません。Gate Bの通過条件と、Oracle compatibilityを主張する条件を分離する判断です。

## Gate Bの3必須Evidence

| 必須Evidence | Repository / GitHub Issue | Evidence nature | 現在地 |
| --- | --- | --- | --- |
| #310 | `ISSUE-000032` / #310 | deterministic repository | **Ready to run** — clean checkout / normal Go runner待ち |
| #255 | `ISSUE-000025` / #255 | generated artifact reproducibility | **Ready to check** — clean normal Go environment待ち |
| #308 | `ISSUE-000031` / #308 | real-provider ephemeral fixture | **Ready to run** — Evidence専用空PostgreSQL DB待ち |

3 Evidenceは相互代替しません。

## 継続Evidence — actual external Oracle

| Track | Repository / GitHub Issue | 現在地 |
| --- | --- | --- |
| actual external asset | `ISSUE-000022` / #214 | **Blocked / unresolved可** |
| actual Oracle driver / Host assembly | `ISSUE-000033` / #371 | **Blocked / unresolved可** |
| actual Oracle read-only Evidence | `ISSUE-000034` / #376 | **Blocked / unresolved可** |

#214 / #371 / #376は廃止しません。actual Oracle compatibility / actual `master_data` integrationを実証済みと主張するために必要です。

Gate B判定時点で未完了なら、Handoverへ`unresolved / not verified`として明示します。

## #310 deterministic Repository

正規runnerは`tools/reference-gate-b-repository-evidence`です。

PR #346以降、targeted runnerは次の三つを独立checkとして扱います。

1. 最終UI-less Scenario
2. external Evidence intake
3. `tools/internal/repositoryevidence`のsource境界Conformance

source境界Conformanceの等価コマンド:

```bash
go test ./tools/internal/repositoryevidence -count=1
```

runner全体では、実行前repository revision / clean、Go toolchain identity、Canonical Workspace Validation、targeted checks、実行後HEAD / clean、小さい非Canonical JSON Evidenceを扱います。

PR #351以降、開始時working treeがcleanでも、post-run確認が終わるまでは`reproducibleRevision=false`を維持します。この意味はPR #353で`ISSUE-000032`にも同期済みです。

### external Evidence intakeを必須checkに残す

actual OracleをGate B必須から外しても、external Evidence intakeは#310の必須targeted checkに残します。

これにより、actual Oracle環境がなくても次を通常runnerから確認します。

- external EvidenceをCanonical Worldへ昇格しない
- SQL realizationがなければEvidenceから補完しない
- provenance / read-only Evidence参照を要求する
- native source fingerprint staleを検知する
- invalid Workspaceを受理しない

このcheck成功をactual Oracle read成功へ格上げしません。

## #255 generated artifact reproducibility

#255はWorld / Plugin Lock / Durable Runの**generated artifact再現性だけ**を扱います。

正規個別tool:

- `tools/reference-golden`
- `tools/reference-durable-run-golden`

最終完了Evidence runner:

- `tools/reference-generated-artifact-evidence`

PR #367以降、World / Plugin Lockの修復writeも既存goldenを先にtruncateせずsafe replaceします。

`-write`やsafe replaceの成功そのものをGate B完了Evidenceへ格上げしません。修復差分をcommitした新しいclean revisionでcheck-only runnerを再実行します。

## #308 real PostgreSQL engine

keyless Relation Storageの設計・実装は`ISSUE-000024` / #244でCompletedです。

実PostgreSQL engine Evidenceだけを本laneへ分離しています。

### Repository source固定 — PR #336

Databaseへ接続する前にclean working treeとHEADを確認し、Provider runtime / cleanup / Database復元確認後にも再確認します。

### fixture前安全境界 — PR #324 / #335 / #365

- Workspace Validation
- PostgreSQL既定DB拒否
- user relation 0件
- Event Trigger 0件
- `public.order_lines`不在
- すべて成立した場合だけtool-owned fixtureをCREATE

既存relationやEvent Triggerを削除・無効化してEvidence用DBへ転用する処理はありません。

### Provider runtime Evidence

同一runで次を確認します。

- `order_id=1001` → 2件
- `order_id=9999` → 正常な0件
- tool-owned fixture欠落 → Execution Error
- Environment / Database instance / executor Capability / Contract Query target identity
- explicit Semantic mapping
- support column非露出
- runtime parameter値 / credential / DSN / 実Database名非保存
- `evidenceNature = real-postgres-ephemeral-fixture`

### cleanup後の復元確認 — PR #341 / #365

```json
{
  "verified": true,
  "userRelationsAfterCleanup": 0,
  "eventTriggersAfterCleanup": 0,
  "fixtureObjectAbsentAfterCleanup": true
}
```

#308はactual OracleをGate B必須から外しても必須Evidenceに残します。Gate Bをsynthetic Evidenceだけへ縮退させず、少なくとも一つのreal provider runtimeを確認するためです。

## #214 actual external Oracle — 継続track

### external Evidence intake

非Canonical Evidence intake境界とsource-fixed acceptance receipt境界はReadyです。

PR #340で、どのexternal Evidence artifactをどのclean Repository revisionで照合したかをreceiptへ固定できるようにしました。

acceptance receiptはactual Oracle read Evidenceではありません。

### actual read時のdriver観測 — PR #364 / #366

Oracle Runner固有Payloadはactual read時に、raw result列名とdriverが提供できたcolumn metadataを未提供値の推測なしで保持できます。

### actual Oracle driver integration — #371

具体driver / Host assemblyはactual Oracle version / authentication / encryption / datatype / NULL / Host制約を確認してから確定します。

### actual Oracle read-only Evidence — #376

principal / grant Evidenceを主たるread-only根拠とし、必要に応じたread-only transactionとquery-only runtimeを追加防御として扱います。

read-only確認目的で実外部assetへDML / DDL / `SELECT ... FOR UPDATE` / exclusive lockを試行しません。

### Gate Bでの扱い

#214 / #371 / #376が未完了でもGate Bを自動的にNot Readyへ戻しません。

その代わり、Handoverへ次を必須unresolvedとして残します。

- actual Oracle Evidence: not verified
- driver / Host assembly: 未確定または未検証
- read-only Evidence: 未取得
- Oracle compatibility / actual `master_data` integrationを実証済みとは主張しない
- Web側でOracle固有情報を補わない

## candidate revision Rへ3必須Evidenceを揃える

Gate Bの最終Ready判定では、一つのcandidate repository revision `R`を固定します。

```text
#310 deterministic Repository ─┐
#255 generated artifact       ─┼─ repositoryRevision = R
#308 Provider Evidence        ─┘
                               ↓
                        Gate B Ready for R
                               ↓
                        Handover Baseline R
```

Rと異なるrevisionの完了Evidenceしかない場合は、「意味は変わっていないはずだ」と推測して流用せず、Rで再実行します。

#214がGate B判定時点で完了している場合は、actual external EvidenceもRへ追跡可能にしてHandoverへ含めます。未完了ならunresolvedです。

## Observationとの関係

Command / ObservationはADR-0023に従いv1境界が成立し、GitHub #266は完了しています。

production Driverの`ExecuteObservationRead`は、Commandとは別の明示read境界としてread identity / Evidence nature / resultを保持します。

Observationの汎用化はGate B blockerではありません。

## GitHub Actionsとの関係

GitHub Actions workflowは現行Repositoryでは意図的に使用していません。

過去のrunner未割当事象を追跡した#192は現行運用では対象外として`not_planned`でclosedです。

CIサービスのgreenをProduct Evidenceへ代替しません。逆に未実行をgreenとも扱いません。

## 実行順

具体コマンド、安全条件、停止条件は`GATE_B_EVIDENCE_RUNBOOK.md`だけを正とします。

### Phase 1 — 事前確認 / sourceを確定する

1. #310をcleanな通常Go runnerで実行し、deterministicな問題を先に露出させる
2. #255をcheck-onlyし、staleなら正規toolで修復 → 差分確認 → commitする
3. #308をEvidence専用の空PostgreSQL Databaseで事前実行し、Provider / fixture safety / cleanup復元上の問題を露出させる
4. Gate Bに必要なsource / generated artifact変更をすべてcommitする

#214 / #371 / #376はactual Oracle環境情報が得られた時点で並行継続します。Gate Bを通すためだけに未観測情報を埋めません。

### Phase 2 — 最終収束 / candidate revision Rを固定する

1. cleanなcandidate revision Rを固定する
2. Rで#310 `reference-gate-b-repository-evidence`を実行する
3. Rで#255 `reference-generated-artifact-evidence`をcheck-only実行する
4. Rで#308 PostgreSQL Provider Evidenceを実行する
5. #310 / #255 / #308の完了EvidenceがRを参照していることを確認する
6. #214 actual Oracleのverified / unresolved状態を確認する
7. negative caseを再監査する
8. Rを対象にGate BをReady / Not Readyで再判定する
9. Readyなら#291でRをHandover Baselineとして固定し、#214 unresolvedなら明示する
10. その後に第一Web Target #70 / Gate C #270へ進む

Phase 2中にsource変更が必要になった場合は、その変更をcommitした新revisionをRとして選び直し、#310 / #255 / #308をもう一度そのrevisionへ揃えます。

## Gate B通過後

3必須Evidenceを同じrevision Rへ揃え、Gate BをRについてReadyと判定した後も、すぐに「最新Driver」を暗黙のWeb baselineにはしません。

[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](../GATE_B_TO_C_HANDOVER_CONTRACT.md) / `ISSUE-000030` / #291に従い、R、Canonical identity、Driver Scenario、Gate B Evidence、#214 actual Oracleのverified / unresolved状態をHandover Baselineとして固定してから第一Web Target #70へ渡します。

## 完了条件

- [x] `GATE_B_READINESS_AUDIT.md`を主要Driver境界へ同期した
- [x] 各Gate B条件をReady / Partial / Blocked / Pendingへ分類した
- [x] Observationのv1境界をADR-0023へ固定し、#266を完了した
- [x] 最終UI-less Scenarioの構造を同じReference World上で接続した
- [x] external Evidence intake boundaryを実装した
- [x] #214 acceptanceをsource-fixed receipt化した（PR #340）
- [x] #214 Repository正本をreceipt後へ同期した（PR #343）
- [x] #214 actual-read観測metadata保持境界を追加した（PR #364 / #366）
- [x] actual Oracle driver integrationを`ISSUE-000033` / #371へ分離した（PR #372）
- [x] actual Oracle read-only Evidenceを`ISSUE-000034` / #376へ分離した（PR #378）
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
- [x] #310 Repository正本をPR #351後へ同期した（PR #353）
- [x] Evidence Acquisitionを4 Evidence natureへ同期した（PR #342）
- [x] Evidence Acquisitionを直近安全境界へ同期した（PR #375）
- [x] Runbookを#214 receipt / #308 cleanup後へ同期した（PR #344）
- [x] Runbookを#214 metadata / #308 Event Trigger / #255 safe replace後へ同期した（PR #369）
- [x] Gate B→C Handover契約を定義した（#291）
- [x] 4 laneの最終Evidenceを単一revisionへ揃える契約を定義した（PR #354、後に#381で再監査）
- [x] actual OracleをGate B必須blockerとする妥当性を再監査し、継続Evidenceへ分離する判断を記録した（ISSUE-000035 / #381）
- [x] Gate B必須収束を#310 / #255 / #308の3 Evidenceへ更新した
- [ ] #310をcleanな通常Go runnerで事前実行する
- [ ] #255をclean revisionで事前checkし、staleなら修復 → commitする
- [ ] #308をEvidence専用の空PostgreSQL Databaseで事前実行する
- [ ] Gate Bに必要なsource / generated artifact変更をすべてcommitする
- [ ] candidate revision Rで#310 / #255 / #308を再実行する
- [ ] 3必須EvidenceがすべてRを参照していることを確認する
- [ ] #214 actual Oracleのverified / unresolved状態をHandover可能な形で明示する
- [ ] negative caseを再監査する
- [ ] Rを対象にGate BをReady / Not Readyで最終再判定する
- [ ] Gate B Ready時の第一Handover Baseline Rを発行する

## 非目標

- Gate B前に第一Web Targetを本格実装すること
- CIを本Issueから再有効化すること
- external `master_data`の物理情報やOracle datatypeを推測すること
- external EvidenceをCanonical Worldへ昇格すること
- acceptance receiptをactual external readとして扱うこと
- #214 / #371 / #376を削除すること
- #214 unresolvedをactual Oracle成功として扱うこと
- #310 synthetic Evidenceや#308 PostgreSQL Evidenceをactual external Evidenceとして扱うこと
- 成立済み実装IssueをEvidence待ちで再びIn Progressへ戻すこと
- generated artifactとdeterministic Repository Evidenceを同じlaneへ畳むこと
- `-write`やsafe replace成功をgenerated artifact完了Evidenceへ格上げすること
- dirty working treeや実行中にsourceが変わったrunを完了Evidenceとして採用すること
- cleanup後にuser relationまたはEvent Triggerが残る#308 runを成功Evidenceとして採用すること
- 一致済みgenerated artifactを形式的に書き換えること
- Observationを汎用Workflow / Event Sourcing基盤へ一般化すること
- Gate BのためにEditor / AI / 第二Targetを先行させること
- 異なるrevisionで得た#310 / #255 / #308成功Evidenceを一つのReady revisionへ読み替えること

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
- `plan/issue/000034-gate-b-oracle-read-only-evidence.md`
- `plan/issue/000035-gate-b-actual-oracle-blocker-reaudit.md`
- GitHub #69 / #70 / #192 / #214 / #244 / #255 / #266 / #273 / #291 / #308 / #310 / #371 / #376 / #381
- PR #315 / #318 / #321 / #323 / #324 / #326 / #331 / #334 / #335 / #336 / #340 / #341 / #342 / #343 / #344 / #346 / #351 / #353 / #354 / #364 / #365 / #366 / #367 / #369 / #370 / #372 / #373 / #374 / #375 / #378 / #379
