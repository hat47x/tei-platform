# ISSUE-000037: Gate Bの三つの必須Evidenceを同一revisionへ最終収束する

- 状態: In Progress — runner / fail-closed / same-run artifact / secret stable-readまで実装、actual final run待ち
- GitHub Issue: #387
- Gate B監査: `ISSUE-000029` / #273
- Handover: `ISSUE-000030` / #291
- Product Acceptance Scenario: PAS-02 primary / PAS-03 secondary

## 目的

Gate B Required Evidenceを一つのclean candidate revision `R`へ揃え、final convergenceが実際に読んだ詳細Evidenceを**同じrunのexact artifact**として再確認できる状態にします。

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
same-run child exact Evidence artifacts
       ↓
#273 Readiness Audit for R
       ↓ Ready only
#291 Handover B / R
```

actual Oracle #214 / #371 / #376は第四Required laneではありません。

## 権威関係

- command / 順序 /停止条件: `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- Ready判定: `plan/GATE_B_READINESS_AUDIT.md`
- tool固有境界: `tools/reference-gate-b-final-evidence/README.md`

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
GOWORK = off
GOFLAGS = empty
TEI_TEST_POSTGRES_DSN_FILE = Repository外existing regular file
TEI_TEST_POSTGRES_DSN = absent
child Evidence artifact store = Repository外
```

実commandはRunbookを正とします。

## lane順序 / source固定

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

開始時、各lane後、final時に同じclean `R`を要求します。

- HEAD drift
- dirty working tree
- source観測不能
- post-run未確認

をfail closedで拒否します。

## child Evidence envelope

各child Evidenceから少なくとも次を照合します。

- expected `evidenceNature`
- `repositoryRevision = R`
- `reproducibleRevision = true`

三laneすべてが成立しない限り:

```text
allLanesSameRevision = true
reproducibleRevision = true
```

へ進みません。

## same-run child Evidence artifact

各child runnerがstdoutへ出したvalid Evidence JSONのexact bytesを、同一runでRepository外へ保存します。

固定filename:

```text
deterministic-repository.json
generated-artifact-reproducibility.json
real-postgres-ephemeral-fixture.json
```

各lane summary:

- `artifactFile`
- `evidenceArtifactFingerprint`

fingerprintはchild stdout exact bytesのSHA-256です。

同じRで後から再実行した別artifactを、final convergenceが実際に読んだartifactへ読み替えません。

### `childArtifactsRetained=true`の意味

PR #461以降、metadataが三つ揃っただけでは`childArtifactsRetained=true`にしません。

final success直前に保存artifactを再openし、少なくとも次を再確認します。

- laneに対応するfixed filename
- regular file
- symlink replacementではない
- open前後 / read後でfile identityが置き換わっていない
- exact bytesを再読込できる
- 再計算SHA-256が`evidenceArtifactFingerprint`と一致

missing / bytes変更 / symlink置換 / unexpected filename / fingerprint mismatch等を検出した場合はfail closedします。

このrunner内final再検証は#273 Readiness Auditの代替ではありません。#273でも保存same-run artifactsを独立に再照合します。

## artifact directory

`-artifact-dir`指定時:

- Repository外
- parentはexisting
- child directoryはまだ存在しない
- runner自身が0700で新規作成
- existing artifactを上書きしない

未指定時はRepository外owner-only temporary directoryを作成し、pathをstderrへ表示します。

final JSONへlocal artifact path自体は保存しません。

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

を要求します。

ambient検査後だけchild / nested Go environmentを同じ安全値へ固定します。

## PostgreSQL secret境界

DSN本文をouter `go run` / compiler / linkerへ渡しません。

`TEI_TEST_POSTGRES_DSN_FILE`は:

- Repository外
- symlink解決後もRepository外
- existing regular file
- 一行DSN

を要求します。

### secret read前のclean-source preflight

PR #466以降、DSN secret fileをopen / readする前に、PostgreSQL secretを除いたenvironmentでRepository sourceを観測します。

```text
HEAD before
  ↓
git status --porcelain
  ↓
HEAD after
  ↓
HEAD stable && working tree clean
  ↓ only then
secret file open / read
```

次の場合はsecret loaderを呼ばずfail closedします。

- Repository source観測不能
- source観測中のHEAD drift
- dirty working tree

このpreflightは後段のlane開始時source確認を置き換えません。secret読込後も改めてsourceを確認し、preflight後のdriftを拒否します。

### stable single-open read

PR #465以降、path検証後に別の`os.ReadFile`で再openしません。

```text
secret path
  ↓ symlink resolve
Repository外existing regular file
  ↓
validated resolved pathをsingle-open
  ↓
事前identity == open済みfile identity
  ↓
元pathを再resolveして同一identityを確認
  ↓
exact bytesをopen済みdescriptorからread
  ↓
元pathを再resolveして同一identityを再確認
```

non-regular input、Repository内target、別fileへのpath replacement、symlink target replacement、読込後のidentity不一致をfail closedで拒否します。

Repository外regular fileへ向くstableなRepository外symlinkは、読込前後とも同じexternal file identityを指す場合に限り許容します。

runner起動後:

```text
Go probe / Git probe / #310 / #255
  -> DSN本文なし
  -> secret pathなし

#308
  -> in-memory DSN本文のみ
  -> secret pathなし
```

DSN本文、secret path、credential、実Database名を通常Evidenceへ保存しません。

## PostgreSQL fixture安全境界

#308はEvidence専用空PostgreSQL Databaseだけを対象にします。

fixture前:

- default Databaseではない
- user relation 0
- Event Trigger 0
- fixture object不在

fixture作成時はDB-side ownership markerを付け、cleanupではlock後にmarker exact一致したtool-owned objectだけをDROPします。

cleanup後:

- user relation 0
- Event Trigger 0
- fixture object不在

を再確認します。

## generated artifact

final runnerは#255修復を自動実行しません。

staleなら:

```text
check-only
  ↓
必要なgenerator -write
  ↓
git diffで意味確認
  ↓
commit
  ↓
new clean R'
  ↓
三laneを最初から再実行
```

`-write`直後のdirty treeを完了Evidenceにしません。

## actual Oracle

#214 / #371 / #376をRequired laneへ戻しません。

- #310 syntheticをactual Oracleへ格上げしない
- #308 PostgreSQLをactual Oracleへ格上げしない
- 未観測physical factsを補完しない

未完了ならHandover `unresolved`へ明示します。

## 現在成立している実装境界

- 3-lane順序 / same-R convergence
- lane失敗後source re-check
- lane間 / final source drift拒否
- external GOWORK / GOFLAGS拒否
- exact `GOENV=off` launch guard
- child / nested Go environment固定
- Go / Git probeからPostgreSQL secret除去
- direct DSN final launch拒否
- secret read前clean-source preflight
- Repository外DSN secret file
- symlink解決後Repository内secret拒否
- secret file stable single-open / read前後identity再確認
- #308 childだけへのin-memory DSN注入
- PostgreSQL DB-side fixture ownership marker
- same-run child exact artifact保存
- child exact-byte fingerprint
- Repository外artifact directory
- existing artifact上書き拒否
- final success直前のartifact再open / identity / SHA-256再照合

## 現在の未完了条件

残っている中心作業はactual executionです。

- [ ] 実TEI checkoutでfinal runner / repositoryevidence package test
- [ ] Evidence専用空PostgreSQL Database用意
- [ ] Repository外DSN secret file用意
- [ ] source変更が出尽くしたclean R固定
- [ ] Runbook正規commandでactual final run
- [ ] #310 / #255 / #308 same-R成功
- [ ] `childArtifactsRetained = true`
- [ ] 三つのsame-run child exact artifacts保存
- [ ] child SHA-256とfinal fingerprint一致
- [ ] child詳細からlane固有安全条件再監査
- [ ] #273 Gate B Readiness Auditへ対応付け
- [ ] Readyなら同じRを#291 Handover Baselineへ固定

actual final runはrunner実装、isolated test、GitHub mergeで代替しません。

## 非目標

- child runner置換
- #255修復自動化
- #308 DB safety guard迂回
- actual OracleをRequired laneへ戻す
- fingerprintだけでEvidence本文を証明する
- same-R別runをsame-run artifactへ読み替える
- final exit codeだけでReady宣言する
- GitHub Actions greenをProduct Evidenceへ格上げする
