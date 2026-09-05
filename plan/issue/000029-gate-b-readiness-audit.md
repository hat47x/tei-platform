# ISSUE-000029: Gate B readinessをReference / Evidenceから再監査する

- 状態: In Progress — Required Evidence / final convergence契約済み、actual final Evidence待ち
- GitHub Issue: #273
- 対象: `ISSUE-000006` / GitHub Issue #69
- 次段: `ISSUE-000007` / GitHub Issue #70
- Handover: `ISSUE-000030` / GitHub Issue #291
- Gate B scope再監査: `ISSUE-000035` / GitHub Issue #380
- final convergence: `ISSUE-000037` / GitHub Issue #387
- Product Acceptance Scenario: PAS-02 primary / PAS-03 secondary

## 目的

Application Driver #69のGate B readinessを、Issue close数ではなく、**現在のReference / Conformance / 実行Evidenceが必要な種類ごとに揃い、同じsourceへ収束しているか**で監査します。

判定内容の正本は[`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md)です。

文書の責務を次のように分けます。

- [`GATE_B_EVIDENCE_ACQUISITION.md`](../GATE_B_EVIDENCE_ACQUISITION.md): 何を取得するか / 何を証明するか
- [`GATE_B_EVIDENCE_RUNBOOK.md`](../GATE_B_EVIDENCE_RUNBOOK.md): 具体コマンド / 実行順 / fail-closed停止条件
- [`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md): 取得済みEvidenceからのReady / Not Ready判定
- [`GATE_B_TO_C_HANDOVER_CONTRACT.md`](../GATE_B_TO_C_HANDOVER_CONTRACT.md): Ready revisionをGate Cへ固定する方法
- 本Issue: actual Evidence取得から再判定・Handoverまでの進行を追跡する

Runbook以外に具体コマンドを重複正本として持ちません。

## 現行Gate B scope

Gate B Ready判定に必要なsource-dependent実行Evidenceは次の三つです。

1. `ISSUE-000032` / #310 — deterministic Repository Evidence
2. `ISSUE-000025` / #255 — generated artifact reproducibility Evidence
3. `ISSUE-000031` / #308 — real PostgreSQL Provider + ephemeral fixture Evidence

candidate revisionを`R`とする場合、この三つを同じRへ揃えます。

```text
#310 ─┐
#255 ─┼─ R
#308 ─┘
       ↓
#387 final convergence
       ↓
本Issue / Readiness Audit
       ↓ Ready
#291 Handover Baseline R
```

`ISSUE-000022` / #214、`ISSUE-000033` / #371、`ISSUE-000034` / #376のactual Oracle Evidenceは第四Required laneではありません。

未完了なら未完了のまま保持し、Gate B Ready時には未検証範囲をHandover `unresolved`へ明示します。

## 現在の判定

**Gate B: Not Ready**です。

現在の不足は主要意味境界の未設計ではなく、actual Evidence取得です。

| Work | 現在地 |
| --- | --- |
| #310 deterministic Repository | runner / source固定境界 Ready、actual clean checkout run待ち |
| #255 generated artifact reproducibility | check / repair境界 Ready、actual clean check-only待ち |
| #308 real PostgreSQL | runner / empty-DB safety / ownership cleanup境界 Ready、actual empty PostgreSQL run待ち |
| #387 final convergence | runner / Go環境 / source / secret境界 Ready、actual same-R final run待ち |
| 本Issue / #273 | actual Evidenceを受け取ってRを再監査する段階 |

isolated test、runner実装、文書同期だけでReadyにはしません。

## 成立済みの主要意味境界

次はGate Bの中心的な未設計blockerではありません。

- browse / record / reference / header-detail / commandのApplication境界
- Contract Query signature / explicit realization / mapping
- Environment / Database instance / executor Capability
- keyless Relation Storage
- Validation → Authorization → Command
- Business ResultとExecution Errorの分離
- Command Business Result → separate Observation read
- final UI-less Scenario
- external-observed Evidenceを非Canonicalのまま照合する境界
- generated artifact check-only / repair分離
- real PostgreSQL fixture前後安全境界
- Repository source固定境界

成立済み設計Issueを、actual Evidence未取得だけを理由に再び設計In Progressへ戻しません。

## Required Evidenceの監査

### #310

Ready判定時に少なくとも次を確認します。

- `repositoryRevision = R`
- `reproducibleRevision = true`
- Canonical Workspace Validation成功
- final UI-less Scenario成功
- external Evidence intake Conformance成功
- Repository source固定Conformance成功
- post-run同一clean source

synthetic Evidenceをactual Oracleへ格上げしません。

### #255

少なくとも次を確認します。

- `repositoryRevision = R`
- `reproducibleRevision = true`
- World / Plugin Lock golden一致
- Durable Run golden一致
- check-only run
- post-run同一clean source

stale修復が必要なら修復差分をcommitし、新revisionを新candidateとして三つのRequired Evidenceを取り直します。

### #308

少なくとも次を確認します。

- `repositoryRevision = R`
- `reproducibleRevision = true`
- Evidence専用空PostgreSQL Database
- fixture前user relation 0件 / Event Trigger 0件 / fixture不在
- 2件 / 0件 / missing-table Execution Error
- explicit Semantic mapping
- support column非露出
- DB-side ownership marker一致objectだけをcleanup
- cleanup後user relation 0件 / Event Trigger 0件 / fixture不在
- DSN / credential /実Database名非保存
- post-run同一clean source

## #387 final convergenceの監査

final runnerは#310 → #255 → #308の順で実行し、各lane後と最後にsourceを再確認します。

Ready判定へ渡すfinal JSONには少なくとも次を要求します。

```text
evidenceNature = gate-b-three-lane-final-convergence
repositoryRevision = R
reproducibleRevision = true
allLanesSameRevision = true
postRunRepository.checked = true
postRunRepository.revisionStable = true
postRunRepository.workingTreeClean = true
```

各child Evidenceも期待するnature / R / reproducibleを満たす必要があります。

child `evidenceArtifactFingerprint`はfinal runnerが実際に読んだartifactとの照合用identityであり、Evidence本文やlane合否の代替ではありません。

## actual final runの環境境界

正規条件はRunbookを正とします。

監査では少なくとも次も確認します。

### Go

```text
goEnvironment.externalWorkspaceActive = false
goEnvironment.goFlagsEmpty = true
goEnvironment.goEnvDisabled = true
```

runner compile前からexternal Go environment / workspace / flagsを暗黙入力へしません。

### Repository source

source観測中のHEAD driftをfail closedで拒否し、各lane後も同一clean Rを確認します。

### PostgreSQL secret

final runner launch時にDSN本文を直接渡しません。

Repository外secret fileから起動後に一度読み、#308 childだけへin-memory DSNを渡します。

Go / Git probe、#310、#255へDSN本文またはsecret file pathを渡しません。

## actual Oracle / external-observedの扱い

actual Oracle EvidenceはGate B Required laneではありませんが、未観測事実を自由に補完してよいという意味ではありません。

引き続き次を守ります。

- external EvidenceをCanonicalへ昇格させない
- EvidenceからSQL realizationを生成・補完しない
- physical object / column / SQL / datatype / read-only方式を未観測のまま推測しない
- #310 synthetic Evidenceをactual Oracle成功へしない
- #308 PostgreSQL Evidenceをactual Oracle成功へしない

Gate B通過時:

- verified actual Oracle EvidenceがあればHandoverから追加参照できる
- 未完了またはcandidate Rへの結び付き不足なら`unresolved`へ明示する

external observed artifact自体の観測時刻をRへ無理に合わせません。ただしRに対するverified claimとするならacceptance / actual read provenanceを明示します。

## candidate revisionの扱い

異なるrevisionの成功を寄せ集めません。

```text
#310 -> R1
#255 -> R2
#308 -> R3
```

これはNot Readyです。

candidate Rでsource変更が必要になった場合:

1. final cycleを停止する
2. 必要な変更をcommitする
3. 新candidate R'を選ぶ
4. #310 / #255 / #308をR'で再実行する
5. #387をR'で再実行する
6. 本Issue / Readiness AuditでR'を再判定する

mainがRの後へ進んでも、candidateを暗黙に最新mainへ移動しません。

## Readyへの完了条件

- [x] 現行Gate B Required Evidenceを#310 / #255 / #308へ確定した（#380）
- [x] 三laneを同一revisionへ収束するfinal runnerを用意した（#387）
- [x] source観測 / Go environment / PostgreSQL secret / fixture ownershipのfail-closed境界を実装した
- [x] Runbookを現行3 Required Evidence / secret-file final runへ正規化した（PR #448）
- [x] #387を現行actual final run条件へ正規化した（PR #449）
- [ ] 実TEI checkoutでfinal runner関連package testを成功させる
- [ ] Evidence専用空PostgreSQL Databaseを用意する
- [ ] source変更が出尽くしたclean candidate Rを固定する
- [ ] #387 actual final runを成功させる
- [ ] #310 / #255 / #308 child EvidenceをRで確認する
- [ ] final `allLanesSameRevision = true`を確認する
- [ ] actual runのGo環境 / source / PostgreSQL safety / secret非保存を監査する
- [ ] `GATE_B_READINESS_AUDIT.md`をRについてReady / Not Ready再判定する
- [ ] Readyなら同じRを#291 Handover Baselineへ固定する
- [ ] Oracle未完了範囲を必要に応じてHandover `unresolved`へ記録する

## 非目標

- actual OracleをGate B第四Required laneへ戻すこと
- Runbook以外へ実行コマンドの第二正本を作ること
- isolated test成功だけでGate B Readyとすること
- final runner exit codeだけでReadyを自動宣言すること
- CI greenをProduct Evidenceへ読み替えること
- 第一Web TargetでGate B不足を補完すること

## 関連

- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `ISSUE-000025` / #255
- `ISSUE-000030` / #291
- `ISSUE-000031` / #308
- `ISSUE-000032` / #310
- `ISSUE-000035` / #380
- `ISSUE-000037` / #387
- `ISSUE-000038`～`ISSUE-000045`
