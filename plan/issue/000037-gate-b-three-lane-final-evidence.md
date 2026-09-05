# ISSUE-000037: Gate Bの三つの必須Evidenceを同一revisionへ最終収束する

- 状態: In Progress — runner / fail-closed境界 / Runbook同期済み、実TEI checkoutでのactual final run待ち
- GitHub Issue: #387
- Gate B scope判断: `ISSUE-000035` / #380
- Gate B監査: `ISSUE-000029` / #273
- Handover: `ISSUE-000030` / #291
- Product Acceptance Scenario: PAS-02 primary / PAS-03 secondary

## 目的

Gate B Ready判定に必要な三つのsource-dependent Evidenceが、**同じclean candidate repository revision `R`**から得られたことを機械的に確認します。

現行Required Evidenceは次の三つです。

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL Provider + ephemeral fixture Evidence

```text
#310 ─┐
#255 ─┼─ repositoryRevision = R
#308 ─┘
       ↓
gate-b-three-lane-final-convergence
       ↓
Gate B audit for R
       ↓
Handover Baseline R
```

actual Oracle #214 / #371 / #376は第四Required laneではありません。取得済みならHandoverでverified external asset Evidenceとして参照でき、未完了またはRへの結び付きが不足する場合は`unresolved`として保持します。

## 権威関係

具体的なコマンド、実行順、安全条件、停止条件の唯一の正本は[`plan/GATE_B_EVIDENCE_RUNBOOK.md`](../GATE_B_EVIDENCE_RUNBOOK.md)です。

本Issueは「何を完了させるか」を保持し、Runbookのコマンドを第二の実行契約として複製しません。

final runner固有の実装境界は[`tools/reference-gate-b-final-evidence/README.md`](../../tools/reference-gate-b-final-evidence/README.md)で追跡します。

## 正規tool

```text
tools/reference-gate-b-final-evidence
```

現行Runbookでは、final runner自身のcompile前から外部Go環境とPostgreSQL DSN本文を暗黙入力へしない形で起動します。

概念上の起動条件は次です。

```text
GOENV = off
GOWORK = off
GOFLAGS = empty
TEI_TEST_POSTGRES_DSN_FILE = Repository外の一行secret file
```

実際のshell commandはRunbookを正とします。

final runnerのlaunch environmentへ`TEI_TEST_POSTGRES_DSN`本文を直接設定する経路は現行の正規final runではありません。

## 実行順

```text
clean candidate revision R
        ↓
#310 deterministic Repository Evidence
        ↓ final runner source再確認
#255 generated artifact reproducibility Evidence
        ↓ final runner source再確認
#308 real PostgreSQL Provider Evidence
        ↓ final runner source再確認
final source check
```

#310または#255に失敗した場合は#308へ進みません。

各laneの成功・失敗後にfinal runner自身がRepository sourceを再確認し、HEAD drift、source観測不能、dirty working tree等を検出した場合は次laneへ進みません。

## final runnerで照合する情報

各child Evidenceから主に次を照合します。

- `evidenceNature`
- `repositoryRevision`
- `reproducibleRevision`

成功には少なくとも次を要求します。

- 各正規runnerが正常終了する
- `evidenceNature`が期待laneと一致する
- 全laneの`repositoryRevision = R`
- 全laneの`reproducibleRevision = true`
- 各lane後のsource再確認が完了している
- 最終確認時もHEADがR
- 最終確認時もworking treeがclean
- `allLanesSameRevision = true`
- final runner自身の`reproducibleRevision = true`

## child Evidence artifact identity

final runnerが実際に読み込んだ各child Evidence JSONのexact bytesへSHA-256を計算し、`evidenceArtifactFingerprint`として追跡します。

このfingerprintは、別途保存したchild Evidenceとfinal runnerが実際に読んだartifactを突き合わせるためのidentityです。

次の代替にはしません。

- Evidence本文の保存
- `evidenceNature`確認
- revision一致確認
- `reproducibleRevision`確認
- child runnerの成功

fingerprintだけから保存していないEvidence本文を推測・再構成しません。

## Repository source観測境界

`tools/internal/repositoryevidence`は、開始時・post-runとも概念上次の順でsourceを観測します。

```text
HEAD before
  ↓
git status --porcelain
  ↓
HEAD after
```

観測中にHEADが変わった場合は一つのRepository stateとして扱わずfail closedします。

final runnerは開始時、各lane後、最終時にsourceを再確認し、確認済みの同一clean revisionだけを受理します。

## Go実行環境境界

final runner自身のcompile / test入力をcandidate revisionの外へ広げないため、現行経路では次を要求します。

- launch時にexact `GOENV=off`
- external `GOWORK`を有効にしない
- effective `GOFLAGS`を空にする
- ambient確認後、child / nested Go commandへ`GOWORK=off / GOFLAGS= / GOENV=off`を継承する

成功JSONでは少なくとも次を確認します。

```text
goEnvironment.externalWorkspaceActive = false
goEnvironment.goFlagsEmpty = true
goEnvironment.goEnvDisabled = true
```

外部workspace path、GOFLAGS本文、GOENV path /拒否値を通常Evidenceへ保存しません。

## PostgreSQL secret境界

final runner自身のouter `go run`、compiler、linker等へcredential候補のDSN本文を継承しません。

正規final runではRepository外のsecret file pathだけをlaunch時に渡します。

runner起動後:

```text
final runner
  ├─ Go environment probe -> DSN本文なし / file pathなし
  ├─ Git source probe     -> DSN本文なし / file pathなし
  ├─ #310                 -> DSN本文なし / file pathなし
  ├─ #255                 -> DSN本文なし / file pathなし
  └─ #308                 -> in-memory DSN本文だけ / file pathなし
```

secret file、file path、DSN本文、credential、実Database名を通常Evidenceへ保存しません。

## PostgreSQL fixture安全境界

#308はEvidence専用空PostgreSQL Databaseだけを対象にします。

少なくとも次をfixture作成前に確認します。

- default Databaseではない
- user relation 0件
- Event Trigger 0件
- `public.order_lines`不在

fixture作成時はDB側ownership markerを付与し、cleanupではtable lock後に作成時markerとexact matchした同じobjectだけをDROPします。

同名objectが他主体に置換された疑いがある場合、削除を推測せずfail closedします。

cleanup後もuser relation 0件 / Event Trigger 0件 / fixture object不在を再確認します。

## generated artifactとの関係

final runnerは#255の修復を自動実行しません。

事前checkでstaleが見つかった場合:

```text
check-only
  ↓
必要なgolden toolだけ -write
  ↓
git diffで意味確認
  ↓
意図したgenerated artifactをcommit
  ↓
新しいclean revision
  ↓
check-only再実行
```

`-write`直後のdirty working treeを#255完了Evidenceやfinal convergenceへ使いません。

sourceが変わった場合は新revisionを新しいcandidate Rとして、三lane最終収束を最初からやり直します。

## #214 actual Oracleとの関係

#214 / #371 / #376は本IssueのRequired laneではありません。

- final runner成功をactual Oracle成功へ読み替えない
- #310 synthetic Evidenceをactual Oracleへ格上げしない
- #308 PostgreSQL Evidenceをactual Oracleへ格上げしない
- 未観測Oracle physical factsを補完しない

actual Oracle Evidenceが未完了のままGate B Readyになった場合、その未検証範囲をHandover Baselineの`unresolved`へ明示します。

## これまでに成立した実装境界

final runnerと関連helperでは、少なくとも次のfail-closed境界がRepositoryへ実装済みです。

- 三lane順序とrevision convergence
- lane failure後のsource再確認
- lane間source drift拒否
- `postRunRepository.checked=true`要求
- child Evidence exact bytes fingerprint
- Repository source観測中HEAD drift拒否
- external GOWORK / non-empty GOFLAGS拒否
- exact `GOENV=off` launch guard
- child / nested Go environment固定
- Go utility subprocessからPostgreSQL secret除去
- Git source probeからPostgreSQL secret除去
- direct DSN final launch拒否
- Repository外DSN secret file入力
- #308 childだけへのin-memory DSN注入
- PostgreSQL fixtureのDB-side ownership marker

個々の設計経緯は`ISSUE-000038`～`ISSUE-000045`を参照します。

## 現在の未完了条件

設計・runner追加ではなく、actual Evidence取得が残っています。

- [ ] 実TEI Repository checkoutの通常Go環境で、Runbook指定のfinal runner / shared source helper package testを実行する
- [ ] Evidence専用空PostgreSQL Databaseを用意する
- [ ] Repository外DSN secret fileを用意する
- [ ] source変更が出尽くしたclean candidate revision Rを固定する
- [ ] Runbookの正規final commandでactual final runを実行する
- [ ] #310 / #255 / #308が同じRへ揃ったfinal JSONを取得する
- [ ] `goEnvironment`、source固定、PostgreSQL fixture安全条件をactual JSONで確認する
- [ ] 必要なchild Evidenceを保存し、final runner fingerprintと照合する
- [ ] `ISSUE-000029` / #273のGate B最終監査へ結果を対応付ける
- [ ] Ready判定なら同じRを`ISSUE-000030` / #291のHandover Baselineへ固定する

actual final runは、runner実装やisolated testの成功だけでは代替しません。

## 非目標

- 各laneの正規runnerを置き換えること
- Runbookを本Issueへ複製すること
- #255修復をfinal runnerから自動実行すること
- #308のDatabase安全guardを迂回すること
- #214 actual Oracle EvidenceをRequired laneへ戻すこと
- fingerprintだけでchild Evidence本文を証明・再構成すること
- final runnerのexit codeだけでGate B Readyを自動宣言すること
- GitHub Actionsを再有効化すること

## 関連

- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `ISSUE-000029` / #273
- `ISSUE-000030` / #291
- `ISSUE-000035` / #380
- `ISSUE-000038` / #418
- `ISSUE-000039` / #421
- `ISSUE-000040` / #424
- `ISSUE-000041` / #427
- `ISSUE-000042` / #430
- `ISSUE-000043` / #433
- `ISSUE-000044` / #436
- `ISSUE-000045` / #441
- #255 / #308 / #310 / #387
- PR #419 / #422 / #425 / #428 / #431 / #434 / #437 / #442 / #448
