# Gate B Evidence Runbook

## 目的

Gate Bで残っている作業を、設計議論へ戻さず**再現可能なEvidence取得**として実行するための正規手順です。

Evidenceの必要性・証明範囲・完了条件は[`GATE_B_EVIDENCE_ACQUISITION.md`](GATE_B_EVIDENCE_ACQUISITION.md)、取得済みEvidenceからのReady / Not Ready判定は[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)を正とします。

**具体的なGate Bのコマンド、実行順、差分確認、fail-closed停止条件については本書を唯一の正本とします。** 個別Issueやtool READMEは各境界の理由・実装詳細を保持しますが、本書と矛盾する起動例を最終Evidenceには採用しません。

この文書は現在の実行契約だけを保持します。旧4-lane Gate B判断の履歴はGit history、`ISSUE-000035` / #380以前のIssue、ADR等から追跡し、本Runbookの現行手順へ混在させません。

## 現行Gate B scope

`ISSUE-000035` / #380以降、Gate B Ready判定へ必須とするsource-dependent実行Evidenceは次の三つです。

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL Provider + ephemeral fixture Evidence

三つを**同じclean candidate repository revision `R`**で成立させます。

```text
#310 deterministic Repository ─┐
#255 generated artifacts      ─┼─ repositoryRevision = R
#308 real PostgreSQL          ─┘
                               ↓
                    final 3-lane convergence
                               ↓
                       Gate B audit for R
                               ↓
                     Handover Baseline R
```

`ISSUE-000022` / #214、`ISSUE-000033` / #371、`ISSUE-000034` / #376のactual Oracle EvidenceはGate B Readyの第四Required laneではありません。

- 取得済みでRに対するverified claimとして十分なら、Handoverで追加のverified external asset Evidenceとして参照する
- 未取得、actual read未完了、またはRへの結び付きが不足する場合は`unresolved`として引き渡す

#310 synthetic Evidenceや#308 PostgreSQL Evidenceをactual Oracle Evidenceへ格上げしません。

## Evidence取得の基本方針

Gate B Evidenceは二段階で扱います。

### 1. 事前確認・修復

candidateを固定する前に、個別runnerで不足を露出させます。

- #310を事前実行してdeterministicな問題を確認する
- #255をcheck-only実行し、staleなら正規toolだけで修復する
- #308をEvidence専用空PostgreSQL Databaseで事前確認する
- sourceまたはgenerated artifactを変更した場合は意味を確認してcommitする

この段階の成功JSONは、その後sourceが変わったら最終Evidenceへ流用しません。

### 2. 最終収束

source変更が出尽くしたclean revisionを`R`として選び、`tools/reference-gate-b-final-evidence`から三つの正規runnerを同じRで実行します。

最終runnerは各child Evidenceの`evidenceNature`、`repositoryRevision`、`reproducibleRevision`を照合し、lane間と最終source確認までRに揃った場合だけ成功します。

## 共通原則

- 未観測の外部事実を名前・慣習・別Providerから補完しない
- Evidence前にCanonical WorkspaceのValidityを確認する
- generated artifactを手編集しない
- fingerprintを人手で置換しない
- `-write`を完了Evidenceそのものにしない
- Evidence runは開始時だけでなく、各runnerが要求する終了時まで同じHEAD / clean working treeへ固定する
- Evidence JSONをRepository内の新規fileへ直接redirectしない
- DSN、credential、secret、runtime parameter値、実Database名を通常Evidenceへ保存しない
- synthetic / generated artifact / real-provider fixture / actual external Evidenceを相互代替しない
- Business Result / Execution Error / Observation / Diagnosticを相互に読み替えない
- sourceを変更したら、以前のrevisionの成功Evidenceを新revisionへ読み替えない
- GitHub ActionsのgreenをProduct Evidenceの代替にしない
- 本手順からGitHub Actionsを再有効化しない

## 0. 実行環境を用意する

### Repository

専用のTEI checkoutを使い、Repository rootで開始します。

```bash
git status --short
git rev-parse HEAD
go version
```

意図しない差分がある場合は停止します。既存差分を機械的に破棄しません。

### PostgreSQL

#308用に、**Evidence専用の空PostgreSQL Database**を用意します。

次のDatabaseは使いません。

- `postgres`
- `template0`
- `template1`

実際のtoolはさらに、fixture作成前にuser relation 0件、Event Trigger 0件、`public.order_lines`不在等をfail closedで確認します。

### DSN secret file

最終runnerへDSN本文をlaunch environmentから直接渡しません。

Repository外のsecret fileに一行DSNを保存し、そのpathだけを指定します。

```text
/secure/path/to/dsn
```

secret fileは次だけを許容します。

- 改行なしの一行DSN
- 末尾に単一LFがある一行DSN
- 末尾に単一CRLFがある一行DSN

empty、複数行、残存CR/LF、NULはfinal runnerが拒否します。

secret file自体、path、DSN本文をRepositoryへ保存しません。

## 1. Canonical WorkspaceをValidationする

事前確認ではまず次を実行します。

```bash
go run ./cmd/tei validate -workspace reference/workspace
```

期待値:

```text
ok
```

Validation Errorがあれば#255修復やProvider Evidenceへ進みません。

最終runner内の各正規runnerも必要なValidationをfail closedで行いますが、事前確認では不足を早く分離するため明示Validationを残します。

## 2. #310 deterministic Repository Evidenceを事前確認する

clean revisionで実行します。

```bash
go run ./tools/reference-gate-b-repository-evidence \
  > /tmp/tei-gate-b-repository-evidence.json
```

必要な場合だけ広いRepository testも同じrunnerへ要求します。

```bash
go run ./tools/reference-gate-b-repository-evidence -broad \
  > /tmp/tei-gate-b-repository-evidence.json
```

完了Evidence候補には少なくとも次を要求します。

- `evidenceNature = deterministic-repository`
- Canonical Workspace Validation成功
- `final-ui-less-scenario`成功
- `external-evidence-intake`成功
- `repository-evidence-source-boundary`成功
- `repositoryRevision`が空でない
- `reproducibleRevision = true`
- post-run source確認済み
- 実行前後でHEADが同一
- 実行後もworking treeがclean

UI-less Scenarioには次の意味境界を含みます。

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

synthetic adapterを含むためactual external state Evidenceへ格上げしません。

## 3. #255 generated artifact Evidenceを事前確認する

clean revisionでcheck-only Evidence runnerを実行します。

```bash
go run ./tools/reference-generated-artifact-evidence \
  > /tmp/tei-gate-b-generated-artifact-evidence.json
```

このrunnerは`-write`を行いません。

成功候補には少なくとも次を要求します。

- `reference-golden`一致
- `reference-durable-run-golden`一致
- `repositoryRevision`が対象revision
- `reproducibleRevision = true`
- post-run source確認済み
- 実行前後で同じHEAD / clean working tree

### staleの場合だけ修復する

World / Plugin Lock:

```bash
go run ./tools/reference-golden
```

staleで、現在のauthoringから差分を説明できる場合だけ:

```bash
go run ./tools/reference-golden -write

git diff -- \
  reference/world.json \
  reference/workspace/.tei/plugin-lock.json
```

Durable Run:

```bash
go run ./tools/reference-durable-run-golden
```

staleの場合だけ:

```bash
go run ./tools/reference-durable-run-golden -write

git diff -- reference/order_rollout.durable-run.m0.21.start.json
```

修復後は、意味上説明可能なgenerated artifactだけをcommitします。

- fixed observed fixtureを生成結果へ合わせて変更しない
- external physical情報を推測して埋めない
- fingerprintやJSON断片を手編集しない
- DSN、secret、一時Evidence fileをcommitしない

新しいclean revisionでcheck-only Evidence runnerを再実行します。

`-write`直後のdirty treeでcheckが成功しても#255完了Evidenceにはしません。

## 4. #308 real PostgreSQL Evidenceを事前確認する

standalone toolの入力契約は`TEI_TEST_POSTGRES_DSN`です。事前確認で直接実行する場合だけ、Evidence専用空DatabaseのDSNをprocessへ与えます。

```bash
TEI_TEST_POSTGRES_DSN='postgres://...' \
go run ./tools/reference-postgres-header-detail-runtime \
  > /tmp/tei-gate-b-postgres-header-detail.json
```

DSN本文をshell history等へ残したくない環境では、環境に適した安全なsecret注入方法を利用します。最終収束では後述のDSN secret file経路だけを正規とします。

### fixture作成前の安全条件

#308は少なくとも次を確認してからfixtureを作ります。

- Repository開始時clean / revision取得済み
- Canonical Workspace Validation成功
- PostgreSQL接続確認成功
- default Databaseではない
- system / temporary領域を除くuser relation 0件
- Event Trigger 0件
- `public.order_lines`不在

通常EvidenceではDatabase名、DSN、credentialを保存しません。

### fixture ownership

fixture作成時はDB側ownership markerを付与します。

```text
BEGIN
  CREATE TABLE public.order_lines (...)
  COMMENT ON TABLE public.order_lines IS '<tool ownership marker>'
COMMIT
```

cleanupではtableをlockし、DB上のmarkerが作成時markerとexact matchした場合だけDROPします。

```text
BEGIN
  LOCK TABLE public.order_lines IN ACCESS EXCLUSIVE MODE
  verify ownership marker
  DROP TABLE public.order_lines
COMMIT
```

同名objectが別主体に置換された疑いがある場合は削除せずfail closedします。cleanup不能でtool fixtureが残る可能性は、他主体のobjectを推測して削除することより優先します。

### Provider Evidence

同じrunで少なくとも次を確認します。

- `evidenceNature = real-postgres-ephemeral-fixture`
- `repositoryRevision`が対象revision
- `reproducibleRevision = true`
- `fixtureDatabaseSafety.guardPassed = true`
- fixture前user relation 0件
- fixture前Event Trigger 0件
- fixture object事前不在
- `order_id=1001`で2件
- `order_id=9999`で0件を正常な空collectionとして扱う
- fixture削除後のmissing-tableをExecution Errorとして扱う
- raw result → Semantic OrderLine mappingが明示される
- Relation Storage support column `order_id`をSemantic attributeへ漏らさない
- parameter valueをEvidenceへ保存しない
- `fixtureCleanupSafety.verified = true`
- cleanup後user relation 0件
- cleanup後Event Trigger 0件
- fixture object不在
- post-run Repository sourceが同一clean revision

#308はproduction既存PostgreSQL assetの観測Evidenceではなく、real engine + tool-owned ephemeral fixture Evidenceです。

## 5. source変更を出し切る

#310 / #255 / #308の事前確認でsourceまたはgenerated artifact変更が必要になった場合、必要な修正を完了し、意味を確認してcommitします。

最終収束を始める前に:

```bash
git status --short
git rev-parse HEAD
```

`git status --short`が空であることを確認します。

このclean revisionをcandidate `R`とします。

最終runnerは実際の`repositoryRevision`をJSONへ記録するため、別のversion IDを新設しません。

## 6. final runner packageを通常TEI checkoutで確認する

actual final runの直前に、同じcheckoutで少なくとも次を確認します。

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
go test \
  ./tools/internal/repositoryevidence \
  ./tools/reference-gate-b-final-evidence \
  -count=1
```

これはGate B Product Evidenceそのものではありません。isolated testだけではなく、actual TEI checkoutでfinal runnerと共有source helperがcompile / testできることを確認する直前検査です。

このtestがsourceを変更した場合やworking treeをdirtyにした場合はfinal runへ進みません。

## 7. 三つのRequired Evidenceを同じRへ最終収束する

### 正規起動

final runner自身をcompileする段階から、Repository外のuser Go environment、Go workspace、GOFLAGS、DSN本文を暗黙入力へしません。

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
TEI_TEST_POSTGRES_DSN_FILE='/secure/path/to/dsn' \
go run ./tools/reference-gate-b-final-evidence \
  > /tmp/tei-gate-b-final-evidence.json
```

#310で`go test ./...`も要求する場合だけ:

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
TEI_TEST_POSTGRES_DSN_FILE='/secure/path/to/dsn' \
go run ./tools/reference-gate-b-final-evidence -broad \
  > /tmp/tei-gate-b-final-evidence.json
```

**final runnerのlaunch environmentへ`TEI_TEST_POSTGRES_DSN`本文を直接設定したrunは正規final Evidenceとして使用しません。** 現行runnerはfail closedで拒否します。

### Go環境境界

runnerはlane開始前に少なくとも次を確認します。

- process environmentの`GOENV`がexact `off`
- `go env GOWORK`が空または`off`
- `go env GOFLAGS`が空

ambient確認後だけrunner processを次へ固定します。

```text
GOWORK=off
GOFLAGS=
GOENV=off
```

これを#310 / #255 / #308 childと通常のnested Go commandへ継承します。

成功JSONでは少なくとも次を確認します。

```text
goEnvironment.externalWorkspaceActive = false
goEnvironment.goFlagsEmpty = true
goEnvironment.goEnvDisabled = true
```

### PostgreSQL secret境界

final runnerはRepository外secret fileを起動後に一度だけ読み、DSN本文をmemoryへ保持します。

- outer `go run` / compiler / linker: DSN本文なし
- Go environment probe: DSN本文 / file pathなし
- Git source probe: DSN本文 / file pathなし
- #310 child: DSN本文 / file pathなし
- #255 child: DSN本文 / file pathなし
- #308 child: in-memory DSN本文だけを`TEI_TEST_POSTGRES_DSN`として渡す
- #308 childにもsecret file pathを渡さない

DSN本文やsecret file pathをEvidenceへ保存しません。

### lane順序

```text
clean candidate revision R
        ↓
#310 deterministic Repository Evidence
        ↓ final runner source再確認
#255 generated artifact reproducibility
        ↓ final runner source再確認
#308 real PostgreSQL Provider Evidence
        ↓ final runner source再確認
final source check
```

#310または#255が失敗した場合は#308へ進みません。

各laneの成功・失敗後にfinal runner自身がsourceを再確認し、確認不能、HEAD drift、dirty working tree等を検出したら次laneへ進みません。

### final convergence成功条件

最終JSONから少なくとも次を確認します。

```text
evidenceNature = gate-b-three-lane-final-convergence
repositoryRevision = R
reproducibleRevision = true
allLanesSameRevision = true
postRunRepository.checked = true
postRunRepository.revisionStable = true
postRunRepository.workingTreeClean = true
```

各laneについて:

```text
#310.evidenceNature = deterministic-repository
#310.repositoryRevision = R
#310.reproducibleRevision = true

#255.repositoryRevision = R
#255.reproducibleRevision = true

#308.evidenceNature = real-postgres-ephemeral-fixture
#308.repositoryRevision = R
#308.reproducibleRevision = true
```

#255の正確な`evidenceNature`はrunner出力の期待値とfinal runnerの照合結果を正とします。文字列を人手で推測してEvidenceを書き換えません。

各child Evidence exact bytesの`evidenceArtifactFingerprint`も残ります。これは、final runnerが実際に読んだchild JSONを別途保存したartifactと照合するための追跡情報であり、Evidence本文やlane合否の代替ではありません。

## 8. child Evidenceを必要に応じて別保存する

final JSONはchild Evidence本文を複製しません。

Gate B監査や将来Handoverで詳細参照が必要な場合は、同じR・同じ安全条件で各正規runnerのJSONをRepository外へ別保存します。

ただし、個別再実行結果をfinal runnerが実際に読んだchild artifactと同一とみなす場合は、`evidenceArtifactFingerprint`等でidentityを照合します。

「同じRで後からもう一度実行したから同じartifactのはず」と推測しません。

## 9. Gate B最終監査

final convergence成功だけでGate B Readyを自動宣言しません。

`GATE_B_READINESS_AUDIT.md`へ戻り、Rについて少なくとも次を監査します。

- #310 required Evidence成立
- #255 required Evidence成立
- #308 required Evidence成立
- 三つが同じRへ収束
- Evidence natureを相互代替していない
- external-observed未観測事項を推測していない
- DSN / credential / secret / runtime parameter値を通常Evidenceへ保存していない
- PostgreSQL fixture前後安全性が成立
- Result / Execution Error / Observation境界が保たれている
- unresolvedを成功へ読み替えていない

Gate B監査がRをReadyと判定した場合だけ、次のHandoverへ進みます。

## 10. Gate B → Gate C Handover

Readyとなった同じRを`ISSUE-000030` / #291のHandover Baselineへ固定します。

Handoverでは少なくとも次へ戻れるようにします。

- repository revision R
- Canonical Application / Pattern / Data / Relation / Query / Capability identity
- Driver Scenario identity
- Business Result / Execution Error / Observation境界
- Datasource / authority境界
- #310 / #255 / #308 Required Evidence
- Gate B final audit for R
- actual Oracle等のoptional verified external Evidence
- unresolved

HandoverはCanonicalの第二正本ではありません。

### actual Oracleが未完了の場合

#214 / #371 / #376をGate B blockerへ戻しません。

少なくとも次を`unresolved`として明示します。

- actual Oracle physical object / raw columns / native SQL等が未観測ならその範囲
- actual Oracle readが未実証ならその事実
- actual driver integration / effective read-only保証が未完了ならその範囲

Web側でこれらを補完しません。

### actual Oracle Evidenceを取得済みの場合

Handover Rに対するverified external asset Evidenceとして参照する場合、actual observed artifact、acceptance receipt、actual read、対象runtime / realization / repository revisionのprovenanceを明示します。

external observed artifact自体の`observedAt`がRの時点である必要はありません。ただし、古い成功をRの成功へ黙って読み替えません。

## 停止条件まとめ

次のいずれかが起きたら、そのrunを成功Evidenceへ採用せず停止します。

- Canonical Workspace Validation Error
- 開始時dirty working tree
- source観測不能
- source観測中HEAD drift
- lane後HEAD変化
- lane後working tree dirty
- required lane失敗
- required lane `reproducibleRevision != true`
- laneの`repositoryRevision != R`
- laneの`evidenceNature` mismatch
- final `allLanesSameRevision != true`
- final source確認未完了
- external Go workspace有効
- effective GOFLAGS非空
- exact `GOENV=off`で起動されていない
- final runnerへdirect `TEI_TEST_POSTGRES_DSN`を設定した
- DSN secret fileが空 / multiline /不正形式 /読込不能
- PostgreSQLがdefault Database
- PostgreSQLに既存user relationがある
- PostgreSQLにEvent Triggerがある
- fixture objectが事前存在する
- fixture ownership markerを確認できないcleanup
- cleanup後にuser relation / Event Trigger / fixture objectが残る
- generated artifact staleを修復せず最終収束へ進もうとした
- Evidenceにsecret等を混入させた疑いがある

停止後にsource修正が必要なら、修正をcommitした新しいclean revisionを新Rとして最終cycleを最初からやり直します。

## 非目標

- actual Oracle EvidenceをGate B Readyの第四Required laneへ戻すこと
- Oracle未観測事項をPostgreSQL / synthetic Evidenceから補完すること
- Handover前に第一Web Targetへ着手すること
- 第一TargetだけでTarget-neutral Presentation / Capability taxonomyを完成させること
- generated artifactを手編集すること
- secret manager製品を標準化すること
- Go toolchain / module cacheをRepositoryへvendoringすること
- OS process全体をsandbox化すること
- GitHub ActionsのgreenをGate B Evidenceへ格上げすること

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
- `issue/000035-gate-b-external-provider-scope-audit.md`
- `issue/000037-gate-b-three-lane-final-evidence.md`
- `issue/000038-gate-b-repository-source-observation-stability.md`
- `issue/000039-gate-b-final-go-environment-isolation.md`
- `issue/000040-gate-b-child-go-environment-isolation.md`
- `issue/000041-gate-b-postgres-fixture-ownership-marker.md`
- `issue/000042-gate-b-final-goenv-launch-guard.md`
- `issue/000043-gate-b-probe-postgres-dsn-isolation.md`
- `issue/000044-gate-b-git-probe-postgres-dsn-isolation.md`
- `issue/000045-gate-b-final-postgres-dsn-secret-file.md`
- `tools/reference-gate-b-final-evidence/README.md`
