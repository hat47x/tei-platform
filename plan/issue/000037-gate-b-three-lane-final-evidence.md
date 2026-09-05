# ISSUE-000037: Gate Bの三つの必須Evidenceを同一revisionへ最終収束する

- 状態: In Progress — runner / fail-closed / same-run child artifact保持まで実装、actual final run待ち
- GitHub Issue: #387
- Gate B scope: `ISSUE-000035` / #380
- Gate B監査: `ISSUE-000029` / #273
- Handover: `ISSUE-000030` / #291
- Product Acceptance Scenario: PAS-02 primary / PAS-03 secondary

## 目的

Gate B Ready判定に必要な三つのRequired Evidenceが、**同じclean candidate repository revision `R`**から得られ、さらにfinal convergenceが実際に読んだ詳細child Evidenceを同一runで再確認できる状態にします。

Required Evidence:

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL Provider + ephemeral fixture Evidence

```text
#310 ─┐
#255 ─┼─ repositoryRevision = R
#308 ─┘
       ↓
#387 final convergence
       ↓
same-run child Evidence artifacts
       ↓
Gate B Readiness Audit for R
       ↓
Handover Baseline B / R
```

actual Oracle #214 / #371 / #376は第四Required laneではありません。

## 権威関係

- command / 順序 /停止条件: [`GATE_B_EVIDENCE_RUNBOOK.md`](../GATE_B_EVIDENCE_RUNBOOK.md)
- Ready判定: [`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md)
- tool固有境界: [`tools/reference-gate-b-final-evidence/README.md`](../../tools/reference-gate-b-final-evidence/README.md)

本Issueへ第二のRunbookを複製しません。

## 正規tool

```text
tools/reference-gate-b-final-evidence
```

概念上のlaunch条件:

```text
Repository root
clean candidate R
GOENV = exact off
GOWORK = off / external workspaceなし
GOFLAGS = empty
TEI_TEST_POSTGRES_DSN_FILE = Repository外existing regular file
TEI_TEST_POSTGRES_DSN = absent
child Evidence artifact store = Repository外
```

実commandはRunbookを正とします。

## lane順序

```text
#310 deterministic Repository
        ↓ source re-check
#255 generated artifact reproducibility
        ↓ source re-check
#308 real PostgreSQL Provider
        ↓ source re-check
final source check
```

#310または#255失敗時は#308を開始しません。

## final envelope照合

各child Evidenceから少なくとも次を照合します。

- `evidenceNature`
- `repositoryRevision`
- `reproducibleRevision`

成功には:

- 全child runner正常終了
- expected Evidence nature一致
- 全lane `repositoryRevision = R`
- 全lane `reproducibleRevision = true`
- 各lane後source確認成功
- final HEAD = R
- final working tree clean
- `allLanesSameRevision = true`
- final `reproducibleRevision = true`

を要求します。

## same-run child Evidence artifact

旧runnerはchild stdout exact bytesのSHA-256だけをfinal summaryへ残し、child本文そのものはmemoryから破棄していました。

その状態では、後から同じRで再実行した別artifactを「final convergenceが実際に読んだartifact」と同一だと推測しない限り、Readiness Auditで詳細Evidenceを再確認できません。

現行runnerでは、各child runnerのvalid Evidence JSON exact bytesを同一runでRepository外へ保存します。

filename:

```text
deterministic-repository.json
generated-artifact-reproducibility.json
real-postgres-ephemeral-fixture.json
```

各lane summaryへ:

- `artifactFile`
- `evidenceArtifactFingerprint`

を残し、成功final summaryでは:

```text
childArtifactsRetained = true
```

を要求します。

fingerprintは保存exact bytesのSHA-256です。Evidence本文やlane固有合否の代替にはしません。

既存artifactを上書きしません。

## artifact directory

`-artifact-dir`指定時は、まだ存在しないRepository外directoryをrunnerが新規作成します。

未指定時はRepository外のowner-only temporary directoryを新規作成し、そのlocal pathをstderrへ表示します。path自体はfinal Evidence JSONへ保存しません。

Repository内artifact directoryは拒否します。

## Repository root / source境界

final runnerはDSN本文・secret pathを除いたenvironmentでGit Repository rootを確認し、current working directoryがrootでなければlaneを開始しません。

開始時、各lane後、final時にRepository sourceを再観測します。

- HEAD drift
- dirty working tree
- source観測不能
- post-run未確認

をfail closedで拒否します。

## Go environment境界

final runner自身のcompile前から:

```text
GOENV=off
GOWORK=off
GOFLAGS=
```

を正規条件にします。

runtime guardでも:

```text
goEnvironment.externalWorkspaceActive = false
goEnvironment.goFlagsEmpty = true
goEnvironment.goEnvDisabled = true
```

を確認します。

ambient確認前に値を書き換えて不安全状態を隠しません。

## PostgreSQL secret境界

DSN本文をouter `go run` / compiler / linkerへ渡しません。

`TEI_TEST_POSTGRES_DSN_FILE`は:

- Repository外
- symlink解決後もRepository外
- existing regular file
- 一行DSN

を要求します。

runner起動後:

```text
Go probe / Git probe / #310 / #255
  -> DSN本文なし
  -> secret pathなし

#308
  -> in-memory DSN本文のみ
  -> secret pathなし
```

secret file path、DSN本文、credential、実Database名を通常Evidenceへ保存しません。

## PostgreSQL fixture安全境界

#308はEvidence専用空PostgreSQL Databaseだけを対象にします。

fixture前:

- default Databaseではない
- user relation 0件
- Event Trigger 0件
- fixture object不在

fixture作成時はDB-side ownership markerを付け、cleanupではlock後にmarker exact一致したtool-owned objectだけをDROPします。

cleanup後:

- user relation 0件
- Event Trigger 0件
- fixture object不在

を再確認します。

## generated artifact

final runnerは#255修復を自動実行しません。

staleなら:

```text
check-only
  ↓
必要な正規generator -write
  ↓
git diffで意味確認
  ↓
commit
  ↓
新clean revision R'
  ↓
三laneを最初から再実行
```

`-write`直後のdirty treeを完了Evidenceにしません。

## actual Oracle

#214 / #371 / #376を本IssueのRequired laneへ戻しません。

- final runner成功 → actual Oracle成功、とはしない
- #310 synthetic → actual Oracle、とはしない
- #308 PostgreSQL → actual Oracle、とはしない
- 未観測physical factsを補完しない

未完了ならHandover `unresolved`へ明示します。

## 現在成立している実装境界

- 3-lane順序 / same-R convergence
- lane failure後source re-check
- lane間source drift拒否
- final source check
- external GOWORK / GOFLAGS拒否
- exact `GOENV=off` launch guard
- child / nested Go environment固定
- Go / Git probeからPostgreSQL secret除去
- direct DSN final launch拒否
- Repository外DSN secret file
- symlink解決後Repository内secret拒否
- #308 childだけへのin-memory DSN注入
- PostgreSQL DB-side fixture ownership marker
- child Evidence exact-byte fingerprint
- same-run child Evidence exact artifact保持
- Repository外artifact directory
- existing artifact上書き拒否

## 現在の未完了条件

残っているのはactual executionです。

- [ ] 実TEI checkoutでfinal runner / repositoryevidence package test
- [ ] Evidence専用空PostgreSQL Database用意
- [ ] Repository外DSN secret file用意
- [ ] source変更が出尽くしたclean R固定
- [ ] Runbook正規commandでactual final run
- [ ] #310 / #255 / #308 same-R成功
- [ ] `childArtifactsRetained = true`
- [ ] 三つのsame-run child exact artifact保存
- [ ] child SHA-256とfinal fingerprint一致
- [ ] child詳細からlane固有安全条件再監査
- [ ] #273 Gate B Readiness Auditへ対応付け
- [ ] Readyなら同じRを#291 Handover Baselineへ固定

actual final runはrunner実装やisolated testで代替しません。

## 非目標

- child runnerを置換する
- #255修復を自動化する
- #308 DB safety guardを迂回する
- actual OracleをRequired laneへ戻す
- fingerprintだけでEvidence本文を証明する
- same-R別runをsame-run artifactへ読み替える
- final exit codeだけでReady宣言する
- GitHub Actions greenをProduct Evidenceへする
