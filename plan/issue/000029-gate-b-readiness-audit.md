# ISSUE-000029: Gate B readinessをReference / Evidenceから再監査する

- 状態: Done — Ready判定済み（R = `e5231839a5dcc62cce1181a2731fcb7c70d16736`、2026-09-19）
- GitHub Issue: #273
- 対象: `ISSUE-000006` / GitHub Issue #69
- 次段: `ISSUE-000007` / GitHub Issue #70
- Handover: `ISSUE-000030` / GitHub Issue #291
- final convergence: `ISSUE-000037` / GitHub Issue #387
- Product Acceptance Scenario: PAS-02 primary / PAS-03 secondary

## 目的

Application Driver #69のGate B readinessを、Issue close数やrunner実装済みという主張ではなく、**actual source-fixed Evidence**から判定します。

判定内容の正本は[`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md)です。

- 何を取得するか: [`GATE_B_EVIDENCE_ACQUISITION.md`](../GATE_B_EVIDENCE_ACQUISITION.md)
- 実行command / 順序 / fail-closed条件: [`GATE_B_EVIDENCE_RUNBOOK.md`](../GATE_B_EVIDENCE_RUNBOOK.md)
- Ready / Not Ready判定: [`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md)
- Gate Cへの固定: [`GATE_B_TO_C_HANDOVER_CONTRACT.md`](../GATE_B_TO_C_HANDOVER_CONTRACT.md)

本Issueはactual Evidence取得から再判定・Handoverまでの進行を追跡します。Runbook以外へ第二の実行正本を作りません。

## 現行Gate B scope

Required Evidenceは次の三つです。

1. `ISSUE-000032` / #310 — deterministic Repository Evidence
2. `ISSUE-000025` / #255 — generated artifact reproducibility Evidence
3. `ISSUE-000031` / #308 — real PostgreSQL Provider + ephemeral fixture Evidence

candidate revisionを`R`とすると:

```text
#310 ─┐
#255 ─┼─ repositoryRevision = R
#308 ─┘
       ↓
#387 final convergence
       ↓
same-run child exact Evidence artifacts
       ↓
本Issue / #273 Readiness Audit for R
       ↓ Ready only
#291 Handover Baseline B / R
```

actual Oracle #214 / #371 / #376は第四Required laneではありません。未完了ならHandoverへ`unresolved`として明示します。

## 現在の判定

**Gate B: Ready**です（candidate revision `R = e5231839a5dcc62cce1181a2731fcb7c70d16736`、判定日2026-09-19）。

actual Evidence取得の過程で3件のsource-navigation bug（PR [#726](https://github.com/hat47x/tei-platform/pull/726)、PR [#727](https://github.com/hat47x/tei-platform/pull/727)）を検出・修正し、修正後のPR統合済み`main` HEADをclean checkoutして#310 / #255 / #308 / #387 / 独立artifact auditを最初から再実行しました。判定の詳細と各Evidence field（fingerprint等）は[`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md)が正本です。

Handover Baseline `B / R`は[`GATE_B_TO_C_HANDOVER_BASELINE_B.md`](../GATE_B_TO_C_HANDOVER_BASELINE_B.md)として固定済みです（`ISSUE-000030` / #291）。

## Ready判定へ使うEvidence provenance

Ready判定では、同じRで成功した任意の#310 / #255 / #308 JSONを寄せ集めません。

#387 final runnerが**同一runで実際に読んだexact child Evidence**を使います。

少なくとも次へ戻れる必要があります。

- #387 final convergence artifact
- `repositoryRevision = R`
- `reproducibleRevision = true`
- `allLanesSameRevision = true`
- `childArtifactsRetained = true`
- #310 same-run exact artifact
- #255 same-run exact artifact
- #308 same-run exact artifact
- 各child `evidenceArtifactFingerprint`

同じRで後から再実行した別artifactをsame-run artifactへ読み替えません。

## `childArtifactsRetained=true`を独立再監査する

PR #458以降、#387はchild stdout exact bytesをRepository外へ保持します。

PR #461以降、runner自身もfinal success直前に三つのartifactを再openし、少なくとも次を確認してから`childArtifactsRetained=true`とします。

- fixed filename / lane identity
- regular file
- symlink replacementではない
- open前後 / read後でfile identityが置き換わっていない
- exact bytesを再読込できる
- 再計算SHA-256がlaneの`evidenceArtifactFingerprint`と一致

ただしrunner内確認だけではReadyにしません。本Issue / `GATE_B_READINESS_AUDIT.md`でもsame-run artifactを独立に再照合します。

PR #489 / #493以降、read-only補助tool `tools/reference-gate-b-readiness-artifact-audit` がこの独立再監査を機械化しています。final summary / fixed 3 lane / same-R / exact child SHA-256 / post-run sourceに加え、#310 / #255のcheck集合をexact照合し、#308のCanonical target identity / parameter / deterministic Semantic attribute集合 / fixture前後安全条件も再照合します。通常Evidenceへ保存してはいけないsecret / physical identity fieldもfail closedで拒否します。

PR #498以降は、check名だけではなく、#310のWorkspace identityと#310 / #255各checkのproducer commandを正規contractへexact bindingしています。正規command本文はRunbookを唯一の実行正本とし、本Issueでは「name / command / passed とWorkspace identityを独立artifact auditで再照合する」という判定契約だけを保持します。labelだけ一致しcommandが異なるartifactはfail closedします。

このtoolは第四Required laneではなく、`artifactAuditPassed=true`だけでReadyを宣言しません。実行command / 順序はPR #495以降の`GATE_B_EVIDENCE_RUNBOOK.md`を正とします。

Ready判定では少なくとも:

1. final summaryの`artifactFile`と固定child filenameの対応
2. 保存file exact bytesのSHA-256とfinal fingerprintの一致
3. child JSON本文とfinal lane envelopeの一致
4. #310 Workspace identity / #310・#255 producer commandを含むchild固有contractの一致
5. child本文から#310 / #255 / #308固有安全条件が確認できること

を要求します。

## Required Evidence固有の監査

### #310 deterministic Repository

- `repositoryRevision = R`
- `reproducibleRevision = true`
- Canonical Workspace Validation成功
- final UI-less Scenario成功
- external Evidence intake成功
- Repository source-boundary Conformance成功
- post-run同一clean source/index state

synthetic Evidenceをactual Oracleへ格上げしません。

### #255 generated artifact reproducibility

- `repositoryRevision = R`
- `reproducibleRevision = true`
- World / Plugin Lock golden一致
- Durable Run golden一致
- check-only run
- post-run同一clean source/index state

stale修復が必要なら正規generatorで修復し、diffを意味上確認してcommitした新revisionをcandidateにし、三laneを最初から取り直します。

### #308 real PostgreSQL

- `repositoryRevision = R`
- `reproducibleRevision = true`
- Evidence専用空PostgreSQL Database
- fixture前: default DB拒否 / user relation 0 / Event Trigger 0 / fixture不在
- 2件 / 正常0件 / missing-table Execution Error
- explicit raw result → Semantic mapping
- support column非露出
- DB-side ownership marker exact一致objectだけをcleanup
- cleanup後: user relation 0 / Event Trigger 0 / fixture不在
- DSN / credential / parameter値 /実Database名を通常Evidenceへ保存しない
- post-run同一clean source/index state

## actual final run環境の監査

正規条件はRunbookを正とします。

### Repository source

- Repository rootから起動
- Gate B Evidence専用full checkout
- tracked / untracked / ignored local stateなし
- tracked Git index entryが通常`H`のみ
- `assume-unchanged` / `skip-worktree`なし
- 開始時clean R
- source観測中HEAD driftをfail closed
- 各lane後同一clean R
- final時同一clean R

### Go

```text
goEnvironment.externalWorkspaceActive = false
goEnvironment.goFlagsEmpty = true
goEnvironment.goEnvDisabled = true
```

runner compile前から`GOENV=off` / external GOWORK無効 / GOFLAGS emptyを要求します。

### PostgreSQL secret

- final launch時direct `TEI_TEST_POSTGRES_DSN`不存在
- `TEI_TEST_POSTGRES_DSN_FILE`はRepository外existing regular file
- symlink解決後もRepository外
- final runnerがoriginal secret fileをstable single-openし、normalized DSNをRepository外owner-only temporary handoff fileへ複製する
- Go / Git probe、#310、#255へDSN本文・original secret path・handoff pathを渡さない
- #308 `go run` / compiler / linkerへDSN本文を渡さずtemporary handoff pathだけを渡す
- #308 runtimeがhandoff fileからDSN本文を読み、DSN / file path environmentをsource probe前に消去する
- secret / credential / original secret path / handoff pathを通常Evidenceへ保存しない

## candidate revision

異なるrevisionの成功を寄せ集めません。

candidate `R`はfull checkoutから選び、tracked / untracked / ignored local stateなし、tracked Git index entryが通常`H`のみを要求します。

source変更が必要ならfinal cycleを停止し、変更をcommitした新candidate `R'`で#310 / #255 / #308 / #387を最初から再実行します。

mainが先へ進んでも、candidateを暗黙に最新mainへ移しません。

## actual Oracle / external-observed

actual Oracle EvidenceはRequired laneではありませんが、未観測事実を補完してよいという意味ではありません。

- external EvidenceをCanonicalへ昇格させない
- #310 synthetic / #308 PostgreSQLをactual Oracleへ読み替えない
- physical object / column / SQL / datatype / driver behavior / read-only方式を未観測のまま推測しない

Gate B通過時にverified claimとして十分なら追加Evidenceとして参照し、不足があれば`unresolved`へ残します。

## Readyへの完了条件

- [x] Required Evidenceを#310 / #255 / #308へ確定
- [x] #387 same-R final convergence runnerを実装
- [x] same-run child exact artifact保持を実装（PR #458）
- [x] final success直前のartifact再読込 / fingerprint再照合を実装（PR #461）
- [x] Handoverをsame-run Evidence provenanceへ固定（PR #462）
- [x] same-run child独立再監査toolを実装・strict化（PR #489 / #493）
- [x] Readiness artifact auditを正規Runbookへ接続（PR #495）
- [x] #310 Workspace / #310・#255 producer commandを独立artifact auditへ束縛（PR #498）
- [x] 実TEI full checkoutでfinal runner / repositoryevidence / real PostgreSQL Evidence runner / Readiness artifact audit package test成功
- [x] Evidence専用空PostgreSQL Database用意
- [x] clean candidate R固定（`e5231839a5dcc62cce1181a2731fcb7c70d16736`）
- [x] #387 actual final run成功
- [x] #310 / #255 / #308がsame-R成功
- [x] `childArtifactsRetained = true`
- [x] 三つのsame-run child exact artifact保存
- [x] child SHA-256とfinal fingerprint一致を本監査でも再確認
- [x] child本文からlane固有安全条件を再監査
- [x] actual runのGo / source / PostgreSQL / secret境界成立
- [x] `GATE_B_READINESS_AUDIT.md`をRについてReady / Not Ready再判定（Ready）
- [x] Readyなら#291 Handover Baseline `B / R`を作成
- [x] actual Oracle未完了範囲を必要に応じて`unresolved`へ記録

すべて満たしたためReadyです。

## 非目標

- actual Oracleを第四Required laneへ戻す
- Runbook以外へ実行commandの第二正本を作る
- isolated testだけでGate B Readyとする
- final runner exit codeや`childArtifactsRetained=true`だけで自動Ready宣言する
- same-R別runをsame-run artifactへ読み替える
- CI greenをProduct Evidenceへ格上げする
- 第一Web TargetでGate B不足meaningを補完する

## 関連

- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `tools/reference-gate-b-readiness-artifact-audit/README.md`
- `ISSUE-000025` / #255
- `ISSUE-000030` / #291
- `ISSUE-000031` / #308
- `ISSUE-000032` / #310
- `ISSUE-000037` / #387
