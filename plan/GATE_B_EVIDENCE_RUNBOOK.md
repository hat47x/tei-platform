# Gate B Evidence Runbook

## 目的

Gate Bで残っている作業を、設計議論へ戻さず**再現可能なEvidence取得**として実行するための正規手順です。

Evidenceの必要性・証明範囲・完了条件は[`GATE_B_EVIDENCE_ACQUISITION.md`](GATE_B_EVIDENCE_ACQUISITION.md)、取得済みEvidenceからのReady / Not Ready判定は[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)を正とします。

**具体的なコマンド、実行順、差分確認、fail-closed停止条件については本書を唯一の正本とします。** 他文書に同じコマンドが残っている場合、本書と矛盾する手順を採用しません。

2026-09-04時点では、Application Driverの主要意味境界、最終UI-less Scenario構造、external-observed Query Evidence受入境界、PostgreSQL fixtureの安全境界まで成立しています。

Gate Bを直接止めているEvidence laneは次の四つです。

1. `ISSUE-000022` / #214: actual Oracle `master_data`物理・provenance・read-only / actual read Evidence
2. `ISSUE-000031` / #308: real PostgreSQL engine + ephemeral fixture Evidence
3. `ISSUE-000025` / #255: World / Plugin Lock / Durable Runのgenerated artifact reproducibility Evidence
4. `ISSUE-000032` / #310: deterministic Repository / normal Go runner Evidence

GitHub Actionsによる継続的なregression確認は重要ですが、この四laneのProduct Evidenceそのものとは分けます。現行RepositoryではActionsを意図的に使用しておらず、本Runbookから再有効化しません。

## Evidenceの種類

### A. deterministic repository Evidence — #310

Repository checkoutとGo toolchainだけで再現できるApplication Driver / Evidence intake / Repository source固定境界のEvidenceです。

- Workspace Validation
- UI-less synthetic Scenario
- external Evidence intake tests
- Repository source境界のpure Conformance
- 実行前後のsource revision不変性

正規入口は`tools/reference-gate-b-repository-evidence`です。

### B. generated artifact reproducibility Evidence — #255

Canonical Workspace等の正規入力から、保存generated artifactを同じclean revisionで再現できることのEvidenceです。

- World
- Plugin Lock
- Durable Run
- 実行前後のsource revision不変性

最終完了Evidence入口は`tools/reference-generated-artifact-evidence`です。`tools/reference-golden`と`tools/reference-durable-run-golden`は個別check / repairの正規toolとして残します。

### C. real-provider ephemeral fixture Evidence — #308

実PostgreSQL provider engineを使いますが、toolが作るephemeral fixtureに対するEvidenceです。

- `tools/reference-postgres-header-detail-runtime`

production既存Databaseの物理事実や実データEvidenceではありません。

### D. actual external asset Evidence — #214

TEIが所有していない既存Oracle資産の観測事実とactual readです。

- physical object / parameter / raw columns
- native realization source
- provenance
- effective read-only proof
- Oracle driver integration
- Workspace → actual read → Semantic result

synthetic / generated artifact / ephemeral fixtureで代替しません。

## 共通原則

- 未観測の外部事実を名前や慣習から補完しない
- Evidence実行前にCanonical WorkspaceのValidityを確認する
- 生成物を手編集しない
- fingerprintを人手で置換しない
- runtime credential / DSN / parameter値を通常Evidenceへ保存しない
- synthetic / generated artifact / provider fixture / actual external Evidenceを相互代替しない
- 失敗を0件・NotSatisfied・denied等の正常業務結果へ変換しない
- 一致済み生成物を形式的に書き換えない
- generated artifactの`-write`を完了Evidenceそのものとして扱わない
- Gate Bへ採用するEvidenceは、後から同じsourceへ戻れるclean revisionへ固定する
- Evidence run実行前後でsource revisionが変化した場合は完了Evidenceとして採用しない
- stdoutでEvidenceを保存するとき、Repository内の新規fileをredirect先にして開始前clean条件を壊さない
- GitHub Actionsを本手順から再有効化しない

## 0. 実行前確認

Repository rootで作業します。

```bash
git status --short
git rev-parse HEAD
go version
```

Evidenceを特定revisionへ固定するため、意図しない未commit差分がある場合は先に識別します。既存差分を機械的に破棄しません。

#310 / #255 / #308 / #214 acceptanceの完了Evidenceは、いずれも開始時cleanだけでなく、toolが要求する範囲で**実行後も同じHEADかつclean working tree**であることを要求します。

## 1. Canonical WorkspaceをValidationする

```bash
go run ./cmd/tei validate -workspace reference/workspace
```

期待値:

```text
ok
```

Validation Errorがある場合は、その後のgolden再生成やprovider / external Evidence実行へ進みません。生成物側を手編集して回避しません。

external Evidence tool、PostgreSQL Evidence tool、Repository Evidence runner、generated artifact Evidence runner、Durable Run golden tool自身も通常Validation Errorでfail closedします。ただし最初の明示Validationは、前提不整合を早期に分離するため残します。

## 2. #310 deterministic Repository Evidenceを取得する

### 正規入口

clean revisionで次を実行します。

```bash
go run ./tools/reference-gate-b-repository-evidence \
  > /tmp/tei-gate-b-repository-evidence.json
```

Repository内の新規fileへ直接redirectしません。shellはcommand開始前にredirect先を作成するため、それ自体がworking treeをdirtyにし、runnerが意図どおりfail closedします。

runnerは一つのEvidenceへ次を束ねます。

1. 実行前repository revision
2. 実行前working tree clean
3. Go toolchain identity
4. Canonical Workspace Validation
5. 最終UI-less Scenario check
6. external Evidence intake check
7. Repository source境界Conformance check
8. `-broad`指定時だけRepository全体test
9. 実行後repository revision
10. 実行前後のHEAD同一性
11. 実行後working tree clean
12. `reproducibleRevision`

PR #336以降、Repository状態観測は#255 / #308と同じ`tools/internal/repositoryevidence`へ収束しています。

PR #346以降、その共有source固定境界のpure Conformanceもtargeted Evidenceの独立checkとして実行します。

`reproducibleRevision = true`になるには、実行前後でHEADが同一で、両時点のworking treeがcleanである必要があります。test自体が成功していても、並行操作やtest実行でsource状態が変わったrunはGate B完了Evidenceとして採用しません。

### targeted check 1: 最終UI-less Scenario

等価コマンド:

```bash
go test ./internal/driver \
  -run '^TestOrderEntryGateBScenarioConnectsMajorBoundariesWithoutUI$' \
  -count=1
```

確認対象:

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

synthetic adapterを含むため、成功してもactual Oracle / real PostgreSQL / 実外部状態のEvidenceにはしません。

### targeted check 2: external Evidence intake

等価コマンド:

```bash
go test ./internal/database/externalqueryevidence \
  ./tools/reference-external-query-evidence \
  -count=1
```

確認対象:

- external EvidenceとCanonical Worldの分離
- target / provider / ownership照合
- input → physical parameter照合
- raw result → Semantic mapping照合
- native source fingerprint stale検知
- provenance必須
- read-only mechanism / Evidence参照必須
- EvidenceからSQL realizationを生成・補完しないこと
- Workspace Validation Errorがある前提でEvidenceを受理しないこと
- source-fixed acceptance receiptのConformance
- Evidence artifact fingerprintのConformance

ここでのtest成功は#214 actual external asset Evidenceではありません。

### targeted check 3: Repository source境界Conformance

PR #346で追加した等価コマンド:

```bash
go test ./tools/internal/repositoryevidence -count=1
```

JSONでは`repository-evidence-source-boundary`として、他のtargeted checkとは別に残します。

確認対象:

- cleanな同一revisionの開始 / 終了だけを再現可能と判定する
- dirty startを拒否する
- post-run未確認を拒否する
- HEAD変化を拒否する
- post-run dirtyを拒否する
- revision文字列と`RevisionStable`の矛盾を受理しない

runner自身のpackage `./tools/reference-gate-b-repository-evidence`はこのtargeted `go test`へ含めません。将来runner testが`run()`を呼ぶようになった場合でも自己再帰の経路を作らず、共有helperのpure契約だけを検査します。

このConformanceが実装されていることと、#310をnormal Go runnerで成功させたことは別です。

### 広いGo test

横断回帰も同じrunで確認できる環境では`-broad`を指定します。

```bash
go run ./tools/reference-gate-b-repository-evidence -broad \
  > /tmp/tei-gate-b-repository-evidence.json
```

これは三つのtargeted check成功後に`go test ./...`を別checkとして実行します。広いtestの失敗をtargeted Evidenceの成功へ畳み込まず、逆に未実行をgreenとも扱いません。

## 3. #255 generated artifactのcheck-only Evidenceを取得する

clean revisionで、まずRepositoryを書き換えずに最終Evidence runnerを実行します。

```bash
go run ./tools/reference-generated-artifact-evidence \
  > /tmp/tei-gate-b-generated-artifact-evidence.json
```

Repository内の新規fileへ直接redirectしません。

runnerは**`-write`を呼びません**。一つのrunで次を確認します。

1. 実行前repository revision
2. 実行前working tree clean
3. Go toolchain identity
4. `go run ./tools/reference-golden`
5. `go run ./tools/reference-durable-run-golden`
6. 実行後repository revision
7. 実行前後のHEAD同一性
8. 実行後working tree clean
9. `reproducibleRevision`

PR #336以降、source revision観測は#310 / #308と同じtool-only境界を利用します。

成功条件:

- `reference-golden`が一致する
- `reference-durable-run-golden`が一致する
- 実行前後でHEADが同一
- 実行前後でworking treeがclean
- `reproducibleRevision = true`

最初から成功した場合は形式的に`-write`しません。このJSONを#255 Evidence候補として扱います。

不一致の場合は#255完了ではなく、staleを検出した診断結果です。

### 個別診断: World / Plugin Lock

```bash
go run ./tools/reference-golden
```

このtoolはCanonical WorkspaceからWorld / Plugin Lockを生成して保存goldenと比較し、保存Durable RunのPlugin Lock依存も確認します。

一致している場合は形式的に`-write`しません。

## 4. World / Plugin Lockを正規再生成する — staleの場合だけ

```bash
go run ./tools/reference-golden -write
```

更新対象:

```text
reference/world.json
reference/workspace/.tei/plugin-lock.json
```

更新後:

```bash
git diff -- \
  reference/world.json \
  reference/workspace/.tei/plugin-lock.json
```

確認観点:

- 現在のauthoring変更から説明できる差分だけか
- 追加済みCapability等が期待どおり反映されたか
- 未観測のexternal physical情報が混入していないか
- 並び順やfingerprintだけを人手で合わせていないか

`reference-golden -write`後にDurable Run依存がstaleと表示されても、fingerprintだけを手編集しません。

**この`-write`は修復操作です。** working treeがdirtyになった場合、その状態でcheckが通っても#255完了Evidenceにはしません。

## 5. Durable Run全体を正規生成結果と照合する

対象:

```text
reference/order_rollout.durable-run.m0.21.start.json
```

第一Reference identity:

```text
solution: order_rollout
version: 2026.8.27
run-id: order-rollout-prod-m0.21
environment: production
```

### check

```bash
go run ./tools/reference-durable-run-golden
```

内部経路:

```text
Reference Workspace Validation
  -> Solution Release exact resolution
  -> fixed observed deployment state Validation
  -> releaseplan.CompileWithDeployment
  -> pluginruntime.BuildPlan
  -> Plugin Lock / Runtime Policy fingerprint
  -> deploymentdurable.StartWithRuntimePolicy
  -> saved Durable Runとの完全一致check
```

Plugin Lockが変わっていなくても、Durable Run全体の正規再現性を確認するため、このcheckは実行します。

PR #326以降、不正なSolution Canonical IDもpanicではなく通常errorでfail closedします。

### write — staleの場合だけ

```bash
go run ./tools/reference-durable-run-golden -write
```

保存pathを先にtruncateせず、同じdirectoryの一時fileへ生成結果全体を書き、Sync / Close後にRenameします。

更新後:

```bash
git diff -- reference/order_rollout.durable-run.m0.21.start.json
```

確認観点:

- `pluginLockFingerprint`が現在Plugin Lockと整合する
- `runtimePolicyFingerprint`を含む差分が正規生成入力から説明できる
- fixed observed stateを生成結果へ合わせて書き換えていない
- fingerprintやJSONの一部だけを手編集していない

### stale修復後の確定

World / Plugin Lock / Durable Runのいずれかに差分が生じた場合:

1. `git diff`で意味上の差分を確認する
2. 意図したgenerated artifactだけをcommitする
3. DSN、credential、一時Evidence file、fixed observed fixture等が混入していないことを確認する
4. 新しいclean revisionから`tools/reference-generated-artifact-evidence`を再実行する

`-write`直後のdirty treeを最終Evidenceのsource revisionとして記録しません。

## 6. #308 実PostgreSQL engineでheader-detailを確認する

### Evidenceの性質

これは**real PostgreSQL engine + ephemeral fixture runtime Evidence**です。production既存Databaseの物理事実や実データEvidenceではありません。

### source revisionのfail-closed境界

PR #336以降、Databaseへ接続する前にRepository sourceを確認します。

```text
git rev-parse HEAD
git status --porcelain
```

working treeがdirtyならWorkspace ValidationやPostgreSQL接続へ進みません。

Provider Evidence、fixture cleanup、Database復元確認の後にもHEAD / working treeを再確認し、次の場合は成功JSONを出しません。

- 実行中にHEADが変化した
- 実行後にworking tree差分が残った
- post-run revision文字列が開始revisionと一致しない

成功Evidenceには少なくとも次を要求します。

```text
repositoryRevision
workingTreeClean = true
reproducibleRevision = true
postRunRepository.checked = true
postRunRepository.repositoryRevision = repositoryRevision
postRunRepository.revisionStable = true
postRunRepository.workingTreeClean = true
```

### fixture前のfail-closed安全境界

source確認後、`reference/workspace`をLoadし通常Validationを実行します。Validation Error時にはPostgreSQLへ接続しません。

その後だけ`TEI_TEST_POSTGRES_DSN`へ接続し、接続先がEvidence専用空Databaseとして扱える観測条件を確認します。

```text
clean Repository source
  ↓
Workspace Load / Validation
  ↓ Errorなら停止
PostgreSQL接続
  ↓
current_database()確認
  ↓
postgres / template0 / template1なら停止
  ↓
system / temporary領域を除くuser relation件数確認
  ↓ 1件以上なら停止
public.order_lines既存確認
  ↓ 既存なら停止
Ephemeral fixture CREATE
  ↓
Provider runtime Evidence
```

既存relationを削除してEvidence用Databaseへ転用する処理や、既定Databaseで続行するoverrideはありません。

`TEI_TEST_POSTGRES_DSN`には**Evidence専用に作成した空PostgreSQL Database**を指定します。

```bash
export TEI_TEST_POSTGRES_DSN='postgres://...'
```

DSN、credential、接続先の実Database名を通常Evidenceへ保存しません。

### fixture前安全判定Evidence

PR #335以降、guardを通ったという実装事実だけでなく、そのrunで確認した非機密な結果を`fixtureDatabaseSafety`へ残します。

期待値:

```json
{
  "guardPassed": true,
  "databaseIdentityVerified": true,
  "databaseIdentityPersisted": false,
  "nonDefaultDatabaseVerified": true,
  "userRelationsBeforeFixture": 0,
  "fixtureObjectAbsentBeforeCreate": true
}
```

`guardPassed = true`を「組織運用上、このDatabaseがEvidence専用であることの完全証明」へ格上げしません。

### 実行

```bash
go run ./tools/reference-postgres-header-detail-runtime \
  > /tmp/tei-gate-b-postgres-header-detail.json
```

Repository内へ直接redirectしません。`tee`を使う場合も保存先はRepository外とします。

期待するProvider Evidence:

- `evidenceNature = real-postgres-ephemeral-fixture`
- `repositoryRevision`
- `reproducibleRevision = true`
- `fixtureObject = public.order_lines`
- `preexistingFixtureProtected = true`
- `fixtureOwnedByTool = true`
- `fixtureDatabaseSafety.guardPassed = true`
- `fixtureDatabaseSafety.nonDefaultDatabaseVerified = true`
- `fixtureDatabaseSafety.userRelationsBeforeFixture = 0`
- `fixtureDatabaseSafety.fixtureObjectAbsentBeforeCreate = true`
- `fixtureDatabaseSafety.databaseIdentityPersisted = false`
- Pattern = `order_entry.order_lines`
- Relation = `sales.order.lines`
- Environment = `development`
- Database instance = `sales_core.dev`
- executor Capability = `database.query.postgres`
- parameter names = `order_id`
- parameter values persisted = false
- rows = 2
- zeroRows = 0
- Execution Error observed = true
- `sales.order_line.order_id`がSemantic attributeへ追加されていない
- Relation Storage column = `order_id`
- Semantic mapping applied = true

### fixture cleanup / missing-table Error / 空状態復元

Providerの2件 / 0件を確認した後、tool自身が作成した`public.order_lines`をDROPし、同じContract Queryを再実行してphysical table欠落をExecution Errorとして観測します。

PR #341以降、そこで成功扱いせずPostgreSQL catalogを再観測します。

期待値:

```json
{
  "verified": true,
  "userRelationsAfterCleanup": 0,
  "fixtureObjectAbsentAfterCleanup": true
}
```

つまり成功runでは、少なくとも次を一続きに確認します。

```text
fixture前 empty-DB guard
 -> fixture CREATE
 -> 2件 / 0件
 -> fixture DROP
 -> missing-table Execution Error
 -> user relation 0件 / fixture object不在を再観測
 -> Repository HEAD / clean再確認
 -> JSON Evidence
```

終了時にuser relationが残る場合、それがtool由来か他主体の並行変更かを推測せずfail closedします。

## 7. #214 actual Oracle Evidenceを受け取る

この工程は実外部事実が得られるまで**Blockedのまま保持**します。

最低限必要な観測事実:

- physical object kind / identity
- Contract input `id`に対応するphysical parameter
- parameter type / NULL上の扱い
- raw result columns
- raw column type / NULL上の扱い
- raw → `customer.customer` Semantic mapping
- native Oracle SQLまたは他realization source
- source / provenance
- 匿名化・縮約等を行った場合の加工内容
- read-only principal / DB permission / read-only session等の実効的根拠
- 採用Oracle driverのparameter / result / NULL / error伝播Evidence

未観測の`CUSTOMER`、`CUSTOMER_ID`等を仮定しません。

### Evidence取得後のauthoring

1. 観測事実をexternal Evidence JSONへ記録する
2. 観測事実を根拠に`ContractQuerySQLRealization`を明示authoringする
3. native sourceを通常Workspaceから追跡可能にする

EvidenceからSQL realizationを生成しません。

### source-fixed acceptance receipt — PR #340

clean revisionで次を実行します。

```bash
go run ./tools/reference-external-query-evidence \
  -workspace reference/workspace \
  -evidence <observed-evidence.json> \
  > /tmp/tei-external-query-evidence-acceptance.json
```

**Repository内の新規fileへ直接redirectしません。** shellがcommand開始前にfileを作るとworking treeがdirtyになり、toolのclean revision条件と自己矛盾します。

このtoolはEvidence JSONを読む前にRepository revision / working treeを確認し、dirtyなら受理しません。

external Evidence exact bytesのSHA-256を計算した後、strict load / Workspace Validation / target + provider / ownership / mapping / native source / provenance / read-only Evidenceを照合します。

照合後にもHEAD / working treeを再確認し、同じclean revisionへ固定できた場合だけJSON receiptを出します。

期待する主な項目:

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
provider = oracle
accepted = true
```

`evidenceArtifactFingerprint`はtoolが読み込んだexternal Evidence JSONの**exact bytes**を識別します。

既存の`nativeSource.fingerprint`は、観測したnative sourceと現在authoringされている`ContractQuerySQLRealization.Source`が一致するかを確認します。両者は相互代替しません。

receiptへ次を複製しません。

- physical object / parameter / raw columns
- physical type / NULL情報
- provenance本文
- read-only mechanism本文
- Evidence file path
- credential / secret

### acceptanceとactual readを分離する

acceptance receiptが成功しても#214は完了しません。

次に:

1. read-onlyを実効強制したOracle connectionを用意する
2. 採用Oracle driverでnamed parameter / result / NULL / error伝播を確認する
3. `development / master_data.dev / database.query.oracle`のruntime Executorを構成する
4. Workspace → actual Oracle read → explicit Semantic mapping → Customer resultを一続きに確認する

acceptance receiptをactual Oracle read Evidenceへ格上げしません。

## 8. 最終差分を確認する

```bash
git status --short

git diff -- \
  reference/world.json \
  reference/workspace/.tei/plugin-lock.json \
  reference/order_rollout.durable-run.m0.21.start.json
```

commit対象は、正規生成されたartifactと、意味上説明可能なsource変更だけにします。

DSN、credential、runtime parameter値、一時Evidence fileを誤ってcommitしません。

#255でgenerated artifactを修復した場合は、dirty差分を残したままGate B監査へ進まず、意図したartifactをcommitしてclean revisionへ固定し、`tools/reference-generated-artifact-evidence`を再実行します。

## 9. Gate B監査へ戻す

取得したEvidenceを[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)へ対応付けます。

| Evidence lane | 判定対象 |
| --- | --- |
| #310 `reference-gate-b-repository-evidence` JSON | UI-less Scenario / external Evidence intake / Repository source境界Conformance / normal Go runner / source revision不変性 |
| #255 `reference-generated-artifact-evidence` JSON | World / Plugin Lock / Durable Runのclean revision再現性 |
| #308 PostgreSQL tool JSON | source-fixed real PostgreSQL engine / fixture前安全性 / Provider runtime / cleanup後復元 |
| #214 external Evidence + acceptance receipt + actual Oracle Evidence | external-observed assetの実物理事実 / artifact identity / read-only / Oracle actual read |

#214 acceptance receiptだけをactual external asset Evidence完了とは扱いません。

一つのEvidence成功を別種類のEvidenceの代替にしません。

## 停止条件

次の場合はfail closedで止めます。

- Workspace Validation Error
- #310完了Evidenceをdirty working treeから採用しようとしている
- #310 runner実行前後でHEADが変わった、または実行後working treeがdirtyになった
- #310で三つのtargeted checkのいずれかが失敗する
- #310 source-boundary Conformanceを別laneのEvidence代替として扱おうとしている
- #255の`-write`直後のdirty treeを完了Evidenceとして採用しようとしている
- #255 `reference-generated-artifact-evidence`でgolden不一致がある
- #255 Evidence runner実行前後でHEADが変わった、またはworking treeがdirtyである
- #308開始時のworking treeがdirtyである
- #308実行前後でHEADが変わる、または実行後working treeがdirtyである
- #308で`postgres` / `template0` / `template1`へ接続している
- #308の接続先にsystem / temporary領域以外のuser relationが1件以上存在する
- `public.order_lines`が既に存在するPostgreSQL接続先
- #308 cleanup後にuser relationが残る
- #308 cleanup後に`public.order_lines`が存在する
- #214 acceptance開始時のworking treeがdirtyである
- #214 acceptance実行中にHEADが変わる、または実行後working treeがdirtyである
- #214でprovenanceのない物理情報しかない
- #214でSQL realizationが存在せず、Evidenceから補完しないと進めない
- #214 acceptance receiptだけをactual read完了として扱おうとしている
- read-only保証をSQL文字列だけで主張している
- generated artifactがstaleなのにfingerprintやJSON断片だけを手編集しようとしている
- fixed observed fixtureを生成結果へ合わせて機械的に変更しようとしている
- EvidenceとSQL realizationが一致しない
- 実行失敗を0件・正常Business Resultへ変換しないと先へ進めない

これらは「作業が止まった」のではなく、Gate Bが意図どおり不足を露出させたEvidenceとして扱います。

## Gate B通過後

4 laneのEvidenceが揃い、本監査でGate BをReadyと判定した後は、すぐに最新`main`をWeb baselineへしません。

`GATE_B_TO_C_HANDOVER_CONTRACT.md` / #291に従い、Ready判定対象revisionとEvidenceをHandover Baselineとして固定してから第一Web Target #70 / Gate C #270へ進みます。
