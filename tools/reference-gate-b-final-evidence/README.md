# Gate B 3-lane Final Evidence Runner

Gate Bの三つのRequired Evidenceを、**同じclean candidate repository revision `R`**へ最終収束するtoolです。

現行Required Evidence:

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL Provider + ephemeral fixture Evidence

具体的な実行順、安全条件、停止条件の正本は`plan/GATE_B_EVIDENCE_RUNBOOK.md`です。本READMEはtool固有境界だけを説明します。

## 正規起動

Evidence専用の空PostgreSQL Databaseを用意し、DSN本文を**Repository外の一行secret file**へ保存します。final runnerのlaunch environmentへDSN本文そのものを設定しません。

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
TEI_TEST_POSTGRES_DSN_FILE='/secure/path/to/dsn' \
go run ./tools/reference-gate-b-final-evidence \
  > /tmp/tei-gate-b-final-evidence.json
```

#310でRepository全体の`go test ./...`も要求する場合だけ`-broad`を付けます。

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
TEI_TEST_POSTGRES_DSN_FILE='/secure/path/to/dsn' \
go run ./tools/reference-gate-b-final-evidence -broad \
  > /tmp/tei-gate-b-final-evidence.json
```

Repository内へstdoutをredirectしません。shellがprocess開始前にfileを作るため、開始時clean条件を壊します。

## same-run child Evidenceを保持する

final runnerは、各child runnerがstdoutへ出した**exact JSON bytes**を、final convergenceと同じrunのartifactとして保持します。

保存するfilenameは固定です。

```text
deterministic-repository.json
generated-artifact-reproducibility.json
real-postgres-ephemeral-fixture.json
```

### `-artifact-dir`未指定

owner-onlyのtemporary directoryをRepository外へ新規作成し、保存先をstderrへ表示します。

final JSONへlocal pathは保存しません。

### `-artifact-dir`指定

長期監査用に保存先を明示する場合は、**まだ存在しないRepository外directory**を指定します。parent directoryは既存である必要があります。

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

runner自身が`children` directoryを新規作成します。既存directoryを再利用せず、既存child artifactを上書きしません。

各child fileは0600で新規作成し、exact stdout bytesを保存します。final JSONには各laneについて次を残します。

- `artifactFile`
- `evidenceArtifactFingerprint`

`evidenceArtifactFingerprint`は保存したexact bytesに対するSHA-256です。

成功final JSONではさらに:

```text
childArtifactsRetained = true
```

を要求します。

final success直前には、三つの固定filenameをartifact storeから再openし、regular fileであること、読込中にpath identityが置き換わっていないこと、再読込したexact bytesのSHA-256が各laneの`evidenceArtifactFingerprint`と一致することを再確認します。missing / symlink replacement / fingerprint mismatch等を検出した場合は`childArtifactsRetained=true`へ進まずfail closedします。

これにより、Readiness Auditは「同じRで後から再実行した別artifact」ではなく、**final convergenceが実際に読んだ同一run child Evidence**を再確認できます。Readiness Auditでも保存artifactを独立に再照合し、runner内final確認だけを監査の代替にはしません。

## Repository rootを固定する

正規final runはRepository rootから実行します。

runner起動後、DSN本文やsecret file pathを除いたenvironmentで`git rev-parse --show-toplevel`を確認し、current working directoryがそのrootと一致しなければlaneを開始しません。

これはrelativeな正規tool pathとcandidate source境界を一意にするためです。

## Go実行環境をcandidate revision外へ広げない

final runner自身をcompileする段階から、外部Go environment / workspace / flagsを暗黙入力へしません。

正規起動では次を明示します。

```text
GOENV=off
GOWORK=off
GOFLAGS=
```

runner起動後もlane開始前に確認します。

- process environmentの`GOENV`がexact `off`
- `go env GOWORK`が空または`off`
- `go env GOFLAGS`が空

ambient確認後だけ、parent final runnerのenvironmentを次へ固定します。

```text
GOENV=off
GOWORK=off
GOFLAGS=
```

#310 / #255 / #308とnested Go commandもこの境界を継承します。

外部workspace path、GOFLAGS本文、GOENV path /拒否値はfinal Evidenceへ保存しません。

## PostgreSQL secret境界

`TEI_TEST_POSTGRES_DSN`がfinal runner launch environmentに存在するrunは、値が空でも拒否します。大小文字違いのdirect DSN keyも拒否します。

`TEI_TEST_POSTGRES_DSN_FILE`は次を要求します。

- Repository外
- symlink解決後もRepository外
- existing regular file
- 一行DSN
- empty / multiline / NULではない
- 大小文字違いを含めduplicate keyがない

末尾の単一LFまたはCRLFだけは除去します。

original secret file path、temporary handoff path、read errorのunderlying local pathはEvidenceへ保存しません。

runner起動後のsecret exposureは次に限定します。

```text
outer go run / compiler / linker
  -> DSN本文なし
  -> original secret file pathのみ

final runner runtime
  -> secret-free environmentでclean-source preflight
  -> original external secret fileをstable single-open
  -> 読込前後で同一file identityを再確認
  -> normalized DSN本文をmemoryへ保持
  -> Repository外0700 temporary directoryへ0600 handoff fileを作成

Go environment probe
Git source probe
#310
#255
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

parent final runnerのenvironmentへDSN本文を設定し直しません。handoff fileはnormal final-runner return時に削除します。process強制kill / host停止等に対するOS-level完全消去までは保証せず、存在中はRepository外owner-only temporary scopeへ限定します。

## lane順序

```text
clean candidate revision R
        ↓
#310 deterministic Repository Evidence
        ↓ source re-check
#255 generated artifact reproducibility Evidence
        ↓ source re-check
#308 real PostgreSQL Provider Evidence
        ↓ source re-check
final source check
```

#310または#255が失敗した場合、#308へ進みません。

各lane成功・失敗後にfinal runner自身がRepositoryを再観測し、次を満たさなければ後続laneへ進みません。

- `postRunRepository.checked = true`
- HEADがcandidate `R`のまま
- working treeがclean

## lane Evidenceの採用条件

各child JSONについて次を確認します。

- subprocess正常終了
- `evidenceNature`が期待laneと一致
- `repositoryRevision = R`
- `reproducibleRevision = true`
- exact child JSONをsame-run artifact storeへ保持済み

三laneすべてが同じRへ揃うことを要求します。

```text
#310.repositoryRevision == R
#255.repositoryRevision == R
#308.repositoryRevision == R
```

## final success

成功final JSONでは少なくとも次を要求します。

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

各laneには少なくとも次を残します。

- name
- command
- expected / observed Evidence nature
- repository revision
- reproducible revision
- `artifactFile`
- `evidenceArtifactFingerprint`
- pass / fail

final JSONへchild Evidence本文を重複格納しません。詳細Evidenceはsame-run child artifact fileを正とします。

## 保存しないもの

- PostgreSQL DSN本文
- original secret file path
- temporary handoff file path
- credential / secret
- runtime parameter値
- PostgreSQL Database名
- external Go workspace path
- GOFLAGS本文
- GOENV path /拒否値
- child test stdout以外の任意diagnostic dump

child Evidence JSON自体にも各child runnerの通常のsecret非保存契約を適用します。

## fail closed

少なくとも次で成功Evidenceへ採用しません。

- Repository root以外から実行
- 開始時dirty working tree
- source観測不能 / HEAD drift
- lane後dirty / revision drift
- required lane失敗
- Evidence nature / revision mismatch
- `reproducibleRevision != true`
- external Go workspace有効
- GOFLAGS非空
- exact `GOENV=off`でない
- direct `TEI_TEST_POSTGRES_DSN`設定（大小文字違いを含む）
- `TEI_TEST_POSTGRES_DSN_FILE`の大小文字違いを含むduplicate key
- DSN secret fileがRepository内、symlink先がRepository内、non-regular、empty、multiline、NUL、read不能
- temporary handoff fileをRepository外owner-only scopeへ作成できない
- child Evidence artifactをsame-runで保存できない
- final success直前にchild artifactを再読込できない、regular fileではない、置換を検出した、またはfingerprintが一致しない
- artifact directoryがRepository内
- 既存artifactへの上書きが必要になる
- final source check未完了

## 非目標

- actual OracleをGate Bの第四Required laneへ戻す
- PostgreSQL Evidenceをactual Oracleへ読み替える
- final runnerを三つのchild runnerの代替にする
- child EvidenceをCanonicalへ昇格する
- secret manager製品を標準化する
- Go toolchain / module cacheをRepositoryへvendoringする
- OS process全体をsandbox化する
- GitHub Actions greenをGate B Product Evidenceへ格上げする

## 関連

- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `plan/issue/000037-gate-b-three-lane-final-evidence.md`
