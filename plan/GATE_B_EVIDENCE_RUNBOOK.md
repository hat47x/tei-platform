# Gate B Evidence Runbook

## 目的

Gate Bで残っている作業を、設計議論へ戻さず**再現可能なactual Evidence取得**として実行するための正規手順です。

- 何のEvidenceが必要か: [`GATE_B_EVIDENCE_ACQUISITION.md`](GATE_B_EVIDENCE_ACQUISITION.md)
- Ready / Not Ready判定: [`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)
- Gate B → C引渡し: [`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md)

**具体的なGate B command、順序、安全条件、停止条件は本書を唯一の実行正本とします。**

履歴はGit history / ADR / Repository Issueから追跡し、本書へ旧実行契約を混在させません。

## 現行Gate B scope

Gate B Readyへ必須のsource-dependent Evidenceは次の三つです。

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL Provider + ephemeral fixture Evidence

三つを同じclean candidate revision `R`へ揃えます。

```text
#310 ─┐
#255 ─┼─ repositoryRevision = R
#308 ─┘
       ↓
#387 final convergence
       ↓
Gate B Readiness Audit for R
       ↓
Handover Baseline B / revision R
```

actual Oracle #214 / #371 / #376は重要なactual external asset Evidenceとして継続しますが、第四Required laneではありません。取得済みならHandoverへverified external Evidenceとして参照し、未完了なら`unresolved`として渡します。

## Evidence natureを相互代替しない

次の読み替えをしません。

- deterministic Repository成功 → actual external asset成功
- generated artifact一致 → Driver Scenario成功
- PostgreSQL ephemeral fixture成功 → production既存DB成功
- PostgreSQL成功 → actual Oracle成功
- CI green → Product Evidence取得済み

Business Result / Execution Error / Observation / Diagnosticも相互に読み替えません。

## 0. 実行環境

### TEI checkout

専用checkoutのRepository rootから開始します。

```bash
git status --short --untracked-files=all
git rev-parse HEAD
go version
```

意図しない差分があれば停止します。既存差分を機械的に破棄しません。

### Go environment

final convergenceではrunner自身のcompile前から次を明示します。

```text
GOENV=off
GOWORK=off
GOFLAGS=
```

外部`go.work`、user-level Go environment file、`-overlay` / `-modfile` / `-tags`等の暗黙flagsをcandidate sourceへ持ち込みません。

### PostgreSQL

#308には**Evidence専用の空PostgreSQL Database**を用意します。

使わないもの:

- `postgres`
- `template0`
- `template1`
- production / shared Database
- 既存user relationを含むDatabase
- Event Triggerを含むDatabase

### DSN secret file

final convergenceではDSN本文をlaunch environmentへ置きません。

Repository外のexisting regular fileへ一行DSNだけを保存し、`TEI_TEST_POSTGRES_DSN_FILE`で指定します。

PR #466以降、runnerは**DSN secret fileをopen / readする前**に、PostgreSQL secretを除いたenvironmentでRepository sourceを観測します。

```text
HEAD before
  ↓
git status --porcelain --untracked-files=all
  ↓
HEAD after
  ↓
HEAD stable && working tree clean
  ↓ only then
secret file open / read
```

source観測不能、観測中のHEAD drift、dirty working treeではsecret loaderへ進みません。このpreflightは後段のcandidate source確認を置き換えません。secret読込後にsourceが変化した場合も、lane開始前の再確認でfail closedします。

PR #477以降、共有`tools/internal/repositoryevidence`はuntracked fileをambient Git configで隠せないよう、開始時・終了時とも`--untracked-files=all`を明示します。さらに、実行checkoutとは別のrepository / work tree / index等を観測し得るGit identity overrideに加え、`GIT_CONFIG_COUNT` / `GIT_CONFIG_PARAMETERS` / `GIT_CONFIG_KEY_*` / `GIT_CONFIG_VALUE_*`によるcommand-scope config injectionを大小文字違いも含めfail closedで拒否します。

PR #465以降、secret fileはsymlink解決後もRepository外のexisting regular fileであることを確認したうえで、**検証済みのresolved pathを一度だけopen**します。

- 事前に確認したfile identityとopen済みdescriptorを`os.SameFile`で照合する
- 元の指定pathを読込前に再解決し、open済みfileと同じexternal file identityであることを確認する
- DSN exact bytesはそのopen済みdescriptorから読む
- 読込後にも元pathを再解決し、同じexternal file identityであることを確認する
- path replacement / symlink target replacement / Repository内target / non-regular fileはfail closedにする

Repository外regular fileへ向くRepository外symlinkは、読込前後とも同一external file identityを指す場合だけ受理します。

受理する内容:

- 一行の非empty DSN
- optionalな末尾単一LF / CRLF

拒否する内容:

- empty
- multiline
- 残存CR/LF
- NUL
- Repository内file
- symlink解決後にRepository内へ入るfile
- non-regular file
- 読込前後でidentityが変化するfile / path

DSN本文、secret file path、credentialを通常Evidenceへ保存しません。

## 1. Canonical Workspace Validation

事前確認では最初に:

```bash
go run ./cmd/tei validate -workspace reference/workspace
```

期待値:

```text
ok
```

Validation ErrorがあればEvidence取得へ進みません。

## 2. #310 deterministic Repository Evidence事前確認

clean revisionで:

```bash
go run ./tools/reference-gate-b-repository-evidence \
  > /tmp/tei-gate-b-repository-evidence.json
```

必要な場合だけ:

```bash
go run ./tools/reference-gate-b-repository-evidence -broad \
  > /tmp/tei-gate-b-repository-evidence.json
```

少なくとも次を確認します。

- `evidenceNature = deterministic-repository`
- Workspace Validation成功
- final UI-less Scenario成功
- external Evidence intake成功
- repository source-boundary Conformance成功
- `repositoryRevision`非empty
- `reproducibleRevision = true`
- post-run source確認済み
- 実行前後同一HEAD
- 実行後clean

UI-less Scenario:

```text
Authentication
 -> browse
 -> record
 -> reference
 -> header-detail
 -> Canonical Validation
 -> Authorization
 -> Command Business Result
 -> separate Observation read
```

synthetic adapterを含むためactual external Evidenceへ格上げしません。

## 3. #255 generated artifact事前確認

```bash
go run ./tools/reference-generated-artifact-evidence \
  > /tmp/tei-gate-b-generated-artifact-evidence.json
```

このrunnerはcheck-onlyです。

確認点:

- World / Plugin Lock一致
- Durable Run一致
- `repositoryRevision`が対象revision
- `reproducibleRevision = true`
- post-run source確認済み

staleの場合だけ正規generatorを使います。

```bash
go run ./tools/reference-golden

go run ./tools/reference-durable-run-golden
```

修復が必要な場合だけ`-write`を使い、差分を意味上確認してcommitします。

```bash
go run ./tools/reference-golden -write
git diff -- reference/world.json reference/workspace/.tei/plugin-lock.json

go run ./tools/reference-durable-run-golden -write
git diff -- reference/order_rollout.durable-run.m0.21.start.json
```

禁止:

- generated artifactの手編集
- fingerprintの手置換
- fixed observed fixtureをgenerated結果へ合わせること
- secret / temporary Evidenceのcommit

`-write`直後のdirty treeは完了Evidenceではありません。新しいclean revisionでcheck-onlyを再実行します。

## 4. #308 real PostgreSQL事前確認

standalone toolでは従来どおり`TEI_TEST_POSTGRES_DSN`を入力にできます。

```bash
TEI_TEST_POSTGRES_DSN='postgres://...' \
go run ./tools/reference-postgres-header-detail-runtime \
  > /tmp/tei-gate-b-postgres-header-detail.json
```

これはfinal convergenceのsecret-file経路とは別の事前確認です。

### fixture前安全条件

少なくとも次を確認してからfixtureを作ります。

- Repository開始時clean
- Workspace Validation成功
- PostgreSQL接続成功
- default Databaseではない
- user relation 0件
- Event Trigger 0件
- `public.order_lines`不在

### fixture ownership / cleanup

fixture tableへtool ownership markerを付けます。

cleanupではtableをlockし、DB上markerが作成時markerとexact一致した場合だけDROPします。

同名objectが別主体に置換された疑いがある場合は削除せずfail closedします。

### #308 Evidence

同じrunで少なくとも次を確認します。

- `evidenceNature = real-postgres-ephemeral-fixture`
- `repositoryRevision`が対象revision
- `reproducibleRevision = true`
- fixture前安全条件成立
- `order_id=1001`で2件
- `order_id=9999`で正常0件
- fixture削除後missing-tableをExecution Errorとして扱う
- raw result → Semantic OrderLine mapping明示
- support column `order_id`をSemantic attributeへ漏らさない
- parameter値をEvidenceへ保存しない
- cleanup ownership確認成功
- cleanup後user relation 0件 / Event Trigger 0件 / fixture不在
- post-run Repositoryが同一clean revision

#308はreal engine + tool-owned fixture Evidenceです。production既存Database Evidenceではありません。

## 5. source変更を出し切る

#310 / #255 / #308の事前確認でsource変更が必要なら、修正・意味確認・commitを完了します。

最終収束前に:

```bash
git status --short --untracked-files=all
git rev-parse HEAD
```

`git status --short --untracked-files=all`が空であることを要求します。

このclean revisionをcandidate `R`とします。

## 6. final runner package直前検査

同じcheckoutで:

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
go test \
  ./tools/internal/repositoryevidence \
  ./tools/reference-gate-b-final-evidence \
  -count=1
```

これはProduct Evidenceそのものではありません。actual checkoutでfinal runnerとshared source helperがcompile / testできることの直前検査です。

sourceを変更した、またはworking treeがdirtyになった場合はfinal runへ進みません。

## 7. final convergence

### 推奨: same-run child Evidence保存先を明示

Repository外にrun用parent directoryを作り、child directory自体はrunnerに新規作成させます。

```bash
mkdir -m 700 /tmp/tei-gate-b-R

GOENV=off \
GOWORK=off \
GOFLAGS= \
TEI_TEST_POSTGRES_DSN_FILE='/secure/path/to/dsn' \
go run ./tools/reference-gate-b-final-evidence \
  -artifact-dir /tmp/tei-gate-b-R/children \
  > /tmp/tei-gate-b-R/final.json
```

`children`が既に存在する場合、runnerは上書きせず停止します。

`-artifact-dir`を省略した場合は、runnerがRepository外のowner-only temporary directoryを新規作成し、そのpathをstderrへ表示します。

### broad

#310で`go test ./...`も要求する場合だけ:

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
TEI_TEST_POSTGRES_DSN_FILE='/secure/path/to/dsn' \
go run ./tools/reference-gate-b-final-evidence \
  -artifact-dir /tmp/tei-gate-b-R/children \
  -broad \
  > /tmp/tei-gate-b-R/final.json
```

### final runner launch guard

secret read / lane開始までに少なくとも次を確認します。

- current working directory = Git Repository root
- process `GOENV` = exact `off`
- `go env GOWORK` = emptyまたは`off`
- `go env GOFLAGS` = empty
- direct `TEI_TEST_POSTGRES_DSN`不存在
- DSN secret fileがRepository外regular file
- child Evidence保存先がRepository外

現行実装の順序は概念上次です。

```text
Go environment / Repository root確認
        ↓
Repository外child artifact store準備
        ↓
PostgreSQL secretを除いたenvironmentでclean-source preflight
        ↓
Repository外secret fileをstable single-openで読込
        ↓
Repository外owner-only temporary handoff file作成
        ↓
candidate sourceを再確認
        ↓
#310 → #255 → #308
```

secret read前preflightとlane開始時source確認は別です。前者の成功後にsourceが変化しても、後者で停止します。

ambient確認後だけchild environmentを:

```text
GOENV=off
GOWORK=off
GOFLAGS=
```

へ固定します。

### secret exposure

```text
outer go run / compiler / linker
  -> DSN本文なし
  -> original secret file pathのみ

final runner runtime
  -> secret-free environmentでclean-source preflight
  -> original external secret fileをstable single-open
  -> 読込前後で同一file identityを再確認
  -> normalized DSN本文をmemoryへ保持
  -> Repository外の0700 temporary directoryへ0600 handoff fileを作成
  -> handoff fileにはnormalized DSN本文だけを保存

Go environment probe / Git probe / #310 / #255
  -> DSN本文なし
  -> original secret file pathなし
  -> handoff file pathなし

#308 go run / compiler / linker
  -> DSN本文なし
  -> original secret file pathなし
  -> temporary handoff file pathのみ

#308 runtime
  -> handoff fileからDSN本文を読込
  -> DSN / file path environmentをsource probe前に消去
  -> 以後DSN本文はmemoryのみ

final runner return
  -> temporary handoff directoryをcleanup
```

handoff fileは通常のfinal runner return時に削除します。processの強制kill・host停止等に対するOS-levelの完全消去までは保証せず、作成中はRepository外のowner-only temporary scopeに限定します。original secret file・handoff fileのpathやDSN本文を通常Evidenceへ保存しません。

## 8. same-run child Evidence retention

final runnerは各child stdoutのvalid Evidence JSON exact bytesを、そのrunのartifactとしてRepository外へ保存します。

filename:

```text
deterministic-repository.json
generated-artifact-reproducibility.json
real-postgres-ephemeral-fixture.json
```

既存fileを上書きしません。

各laneのfinal summaryには:

- `artifactFile`
- `evidenceArtifactFingerprint`

を残します。

fingerprintは保存したexact bytesのSHA-256です。

成功final JSONでは:

```text
childArtifactsRetained = true
```

を要求します。

**後から同じRで再実行した別artifactを、final runnerが実際に読んだartifactと同一だと推測しません。** Readiness Auditではsame-run child artifactを参照します。

## 9. lane順序

```text
clean candidate revision R
        ↓
#310 deterministic Repository
        ↓ final runner source re-check
#255 generated artifact reproducibility
        ↓ final runner source re-check
#308 real PostgreSQL Provider
        ↓ final runner source re-check
final source check
```

#310または#255失敗時は#308を開始しません。

各lane成功・失敗後にfinal runner自身がsourceを再確認します。

## 10. final success条件

final JSON:

```text
evidenceNature = gate-b-three-lane-final-convergence
repositoryRevision = R
workingTreeClean = true
reproducibleRevision = true
allLanesSameRevision = true
childArtifactsRetained = true
postRunRepository.checked = true
postRunRepository.revisionStable = true
postRunRepository.workingTreeClean = true

goEnvironment.externalWorkspaceActive = false
goEnvironment.goFlagsEmpty = true
goEnvironment.goEnvDisabled = true
```

各lane:

```text
passed = true
repositoryRevision = R
reproducibleRevision = true
artifactFile != empty
evidenceArtifactFingerprint != empty
```

Expected Evidence nature:

```text
#310 deterministic-repository
#255 generated-artifact-reproducibility
#308 real-postgres-ephemeral-fixture
```

文字列やfingerprintを人手で書き換えません。

## 11. Gate B Readiness Audit

final convergence成功だけでGate B Readyを自動宣言しません。

`GATE_B_READINESS_AUDIT.md`へ戻り、Rについて少なくとも次を監査します。

- #310 detailed same-run Evidence成立
- #255 detailed same-run Evidence成立
- #308 detailed same-run Evidence成立
- 三つが同じR
- `childArtifactsRetained = true`
- final summaryのfingerprintと保存child exact bytesが一致
- Evidence nature非代替
- external-observed未観測事項を推測していない
- secret / runtime parameter値を通常Evidenceへ保存していない
- PostgreSQL fixture前後安全性成立
- Business Result / Execution Error / Observation分離
- unresolvedを成功へ読み替えていない

この監査がRをReadyと判定した場合だけHandoverへ進みます。

## 12. Gate B → Gate C Handover

Readyとなった同じRを`ISSUE-000030` / #291のHandover Baselineへ固定します。

Handoverから少なくとも次へ戻れるようにします。

- repository revision R
- Canonical Application / Pattern / Data / Relation / Query / Capability
- Driver Scenario
- Datasource / authority境界
- Business Result / Execution Error / Observation
- #310 / #255 / #308 same-run Evidence
- final convergence artifact
- Gate B final audit for R
- optional verified actual Oracle Evidence
- unresolved

HandoverはCanonicalの第二正本ではありません。

actual Oracle未完了なら#214 / #371 / #376をblockerへ戻さず、未観測範囲を`unresolved`として明示します。Web側で補完しません。

## 停止条件

次のいずれかでは成功Evidenceへ採用しません。

- Workspace Validation Error
- Repository root以外からのfinal run
- 開始時dirty working tree
- secret read前にRepository sourceをcleanかつ安定して観測できない
- Repository source観測environmentにGit identity overrideまたは`GIT_CONFIG_*` command-scope config injectionが存在する
- source観測不能 / HEAD drift
- lane後dirty / revision drift
- required lane失敗
- `reproducibleRevision != true`
- lane `repositoryRevision != R`
- Evidence nature mismatch
- final `allLanesSameRevision != true`
- `childArtifactsRetained != true`
- child artifact保存失敗
- artifact directoryがRepository内
- child artifact既存fileへの上書きが必要
- final source確認未完了
- external Go workspace有効
- effective GOFLAGS非empty
- exact `GOENV=off`で起動されていない
- direct `TEI_TEST_POSTGRES_DSN`設定
- DSN secret fileがRepository内 / symlink先Repository内 / non-regular / empty / multiline / NUL / read不能
- DSN secret fileのpath / symlink target / file identityが読込前後で変化する
- PostgreSQL handoff fileをRepository外owner-only temporary scopeへ作成できない
- PostgreSQLがdefault DB
- PostgreSQLに既存user relation / Event Triggerがある
- fixture objectが事前存在
- fixture ownership marker不一致
- cleanup後fixture / user relation / Event Triggerが残る
- generated artifact stale未修復
- Evidenceへsecret混入の疑い

source修正が必要ならcommitし、新しいclean revisionを新Rとしてcycleを最初からやり直します。

## 非目標

- actual Oracleを第四Required laneへ戻す
- Oracle未観測事項をPostgreSQL / synthetic Evidenceから補完する
- Handover前に第一Web Targetへ着手する
- generated artifactを手編集する
- secret manager製品を標準化する
- Go toolchain / module cacheをRepositoryへvendoringする
- OS process全体をsandbox化する
- GitHub Actions greenをGate B Product Evidenceへ格上げする

## 関連

- `GATE_B_EVIDENCE_ACQUISITION.md`
- `GATE_B_READINESS_AUDIT.md`
- `GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `issue/000025-reference-generated-artifacts-refresh.md`
- `issue/000029-gate-b-readiness-audit.md`
- `issue/000030-gate-b-to-c-handover-baseline.md`
- `issue/000031-gate-b-postgres-header-detail-evidence.md`
- `issue/000032-gate-b-deterministic-repository-evidence.md`
- `issue/000037-gate-b-three-lane-final-evidence.md`
- `tools/reference-gate-b-final-evidence/README.md`
