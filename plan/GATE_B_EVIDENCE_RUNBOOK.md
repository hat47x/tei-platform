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

Repository checkoutとGo toolchainだけで再現できるApplication Driver / Evidence intakeのEvidenceです。

- Workspace Validation
- UI-less synthetic Scenario
- external Evidence intake tests
- Repository source固定境界のpure Conformance
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
- raw column physical metadata / NULL上の扱い
- native realization source
- provenance
- effective read-only proof
- Oracle driver integration
- Workspace → actual read → Semantic result

synthetic / generated artifact / ephemeral fixtureで代替しません。

## Gate B Evidence取得は二段階で行う

PR #354以降、4 laneが個別にsource-fixedであっても、**別々のrepository revisionで成功したEvidenceを寄せ集めてGate B Readyにはしません。**

最終Ready判定では、一つのcandidate repository revision `R`を固定し、4 laneの完了判定に用いるEvidenceをすべてRへ揃えます。

```text
#214 acceptance + actual read ─┐
#308 Provider Evidence        ─┼─ repositoryRevision = R
#255 generated artifact       ─┤
#310 deterministic Repository ─┘
                               ↓
                        Gate B Ready for R
                               ↓
                        Handover Baseline R
```

そのため、作業を二段階に分けます。

### Phase 1 — 事前確認 / sourceを確定する

目的は、実行可能なlaneを先に動かして不足を露出させ、Gate Bに必要なsource変更とgenerated artifact修復をすべて確定することです。

1. #310をcleanな通常Go runnerで事前実行する
2. #255をcheck-onlyし、staleなら正規toolで修復する
3. #308をEvidence専用の空PostgreSQL Databaseで事前実行する
4. #214のactual Oracle physical facts / provenance / read-only Evidenceを取得する
5. 観測事実を根拠に必要なOracle realizationを明示authoringする
6. #214 acceptance / Oracle driver / actual readを事前確認する
7. 必要なsource / generated artifact変更をすべてcommitする

Phase 1で成功JSONを得ても、その後repository sourceが変わった場合、そのJSONを最終完了Evidenceとして流用しません。

### Phase 2 — 最終収束 / candidate revision Rを固定する

sourceとgenerated artifactを確定したclean revisionをRとして選びます。

1. Rで#310を実行する
2. Rで#255をcheck-only実行する
3. Rで#308を実行する
4. Rで#214 acceptance receiptを再取得し、Rのruntime / realizationでactual Oracle readを行う
5. 4 laneすべての完了EvidenceがRを参照していることを確認する
6. negative caseを再監査する
7. Rを対象にGate BをReady / Not Readyで判定する
8. Readyなら#291でRをHandover Baselineとして固定する

Phase 2中にsource変更が必要になった場合は、その変更をcommitした新revisionを新しいRとして選び直し、4 laneをそのrevisionへもう一度揃えます。

外部観測資料そのものの観測時刻はRより前でも構いません。ただし、その資料を現在authoringへ照合する#214 acceptance receiptとactual readはRへ結び直します。

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
- Phase 1の成功Evidenceを、source変更後のPhase 2完了Evidenceへ読み替えない
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

Phase 2では、実行前に次も記録します。

```bash
export TEI_GATE_B_CANDIDATE_REVISION="$(git rev-parse HEAD)"
printf '%s\n' "$TEI_GATE_B_CANDIDATE_REVISION"
```

この環境変数は便宜上の照合用です。toolへ新しい意味を導入するものではありません。最終判定では各JSONの`repositoryRevision`そのものを確認します。

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

runnerは次を一つのEvidenceへ束ねます。

1. 実行前repository revision
2. 実行前working tree clean
3. Go toolchain identity
4. Canonical Workspace Validation
5. 最終UI-less Scenario
6. external Evidence intake boundary tests
7. Repository source固定境界のpure Conformance
8. 実行後repository revision
9. 実行前後のHEAD同一性
10. 実行後working tree clean
11. `reproducibleRevision`

PR #336以降、Repository状態観測は#255 / #308と同じ`tools/internal/repositoryevidence`へ収束しています。PR #346以降は、その共有helper自体のpure Conformanceもtargeted Evidenceとして独立実行します。

PR #351以降、開始時working treeがcleanであっても、post-run確認が完了するまでは`reproducibleRevision=false`を維持します。Go toolchain取得、Workspace Load / Validation、targeted test等で途中失敗したJSONを、再現可能な成功Evidenceとして扱いません。

`reproducibleRevision = true`になるには、実行前後でHEADが同一で、両時点のworking treeがcleanである必要があります。test自体が成功していても、並行操作やtest実行でsource状態が変わったrunはGate B完了Evidenceとして採用しません。

### 最終UI-less Scenario

runnerが実行する等価コマンド:

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

### external Evidence intake tests

runnerが実行する等価コマンド:

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

### Repository source固定境界

PR #346以降、runnerは共有source観測helperのpure Conformanceを3本目のtargeted Evidenceとして実行します。

```bash
go test ./tools/internal/repositoryevidence \
  -count=1
```

確認対象:

- 開始revisionが空でない
- 開始時working treeがclean
- post-run状態が確認済み
- post-run revision文字列が開始revisionと同一
- `revisionStable = true`
- 終了時working treeがclean
- どれか一つでも欠ける状態を再現可能と判定しない

runner自身のpackageはこのtargeted `go test`へ含めません。将来runner側のtestが`run()`を呼ぶようになった場合にも自己再帰の経路を作らず、共有helperのpure契約だけを検証します。

### 広いGo test

横断回帰も同じrunで確認できる環境では`-broad`を指定します。

```bash
go run ./tools/reference-gate-b-repository-evidence -broad \
  > /tmp/tei-gate-b-repository-evidence.json
```

これは3本のtargeted Evidence成功後に`go test ./...`を別checkとして実行します。広いtestの失敗をtargeted Evidenceの成功へ畳み込まず、逆に未実行をgreenとも扱いません。

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

PR #367以降、World / Plugin Lockの修復writeも既存goldenを最初にtruncateしません。同じdirectoryの一時fileへ全内容を書き込み、Sync / Closeした後にRenameし、一時fileは成功・失敗のどちらでもcleanupします。これは修復途中のwrite失敗で既存artifactを空または途中内容へ壊さないための境界であり、複数artifact全体を一つのtransactionとして扱う契約ではありません。

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

**この`-write`は修復操作です。** working treeがdirtyになった場合、その状態でcheckが通っても#255完了Evidenceにはしません。safe replace自体の成功も完了Evidenceへ格上げしません。

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

Phase 1で修復差分をcommitした後は、そのcommitを含むrevisionをcandidate Rとして選ぶか、さらに#214 authoring等を終えてからRを選びます。

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
Event Trigger件数確認
  ↓ 1件以上なら停止
public.order_lines既存確認
  ↓ 既存なら停止
Ephemeral fixture CREATE
  ↓
Provider runtime Evidence
```

PR #365以降、Event Triggerはuser relationとは別にDDLへ介入し得るため、relationが0件でもEvent Triggerが1件以上存在するDatabaseはfixture実行先にしません。disabledなEvent Triggerも既存objectとして扱います。

既存relationやEvent Triggerを削除・無効化してEvidence用Databaseへ転用する処理や、既定Databaseで続行するoverrideはありません。

`TEI_TEST_POSTGRES_DSN`には**Evidence専用に作成した空PostgreSQL Database**を指定します。

```bash
export TEI_TEST_POSTGRES_DSN='postgres://...'
```

DSN、credential、接続先の実Database名を通常Evidenceへ保存しません。

### fixture前安全判定Evidence

PR #335以降、guardを通ったという実装事実だけでなく、そのrunで確認した非機密な結果を`fixtureDatabaseSafety`へ残します。PR #365以降はEvent Trigger件数も含めます。

期待値:

```json
{
  "guardPassed": true,
  "databaseIdentityVerified": true,
  "databaseIdentityPersisted": false,
  "nonDefaultDatabaseVerified": true,
  "userRelationsBeforeFixture": 0,
  "eventTriggersBeforeFixture": 0,
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
- `fixtureDatabaseSafety.eventTriggersBeforeFixture = 0`
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

PR #341以降、そこで成功扱いせずPostgreSQL catalogを再観測します。PR #365以降は、user relationとfixture objectに加えてEvent Triggerも再観測します。

期待値:

```json
{
  "verified": true,
  "userRelationsAfterCleanup": 0,
  "eventTriggersAfterCleanup": 0,
  "fixtureObjectAbsentAfterCleanup": true
}
```

つまり成功runでは、少なくとも次を一続きに確認します。

```text
fixture前 empty-DB guard
 -> user relation 0件 / Event Trigger 0件 / fixture object不在
 -> fixture CREATE
 -> 2件 / 0件
 -> fixture DROP
 -> missing-table Execution Error
 -> user relation 0件 / Event Trigger 0件 / fixture object不在を再観測
 -> Repository HEAD / clean再確認
 -> JSON Evidence
```

終了時にuser relationまたはEvent Triggerが残る場合、それがtool由来か他主体の並行変更かを推測せずfail closedします。

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

PR #364 / #366以降、Oracle Runner固有Payloadはactual read時にdriverが観測したraw result列名と、driverが提供できた範囲の列metadataを保持できます。metadataは`name`、`databaseTypeName`、`nullable`、`length`、`precision`、`scale`で、driverが提供できない項目はfalseや0へ補完せず未観測のまま保持します。Go側の`ScanType`やraw row値は通常Evidenceへ追加しません。

このruntime metadataは、provenance付きexternal Evidenceの`PhysicalType` / `Nullable`を置き換える第二の物理正本ではありません。actual read時のdriver観測として照合材料に使います。`PhysicalType`文字列とdriverの`DatabaseTypeName`が同じ表記になるかは実Oracle / 採用driverを観測する前に仮定せず、自動文字列一致規則を先行固定しません。

acceptance receiptをactual Oracle read Evidenceへ格上げしません。

Phase 1でこの工程を通してauthoring変更が必要になった場合は、その変更をcommitしてからPhase 2へ進みます。

Phase 2では、同じexternal Evidence artifactを使う場合でも、candidate revision R上でacceptance receiptを再取得し、Rのruntime / realizationに対してactual readを再確認します。

## 8. Phase 1の最終差分を確認する

```bash
git status --short

git diff -- \
  reference/world.json \
  reference/workspace/.tei/plugin-lock.json \
  reference/order_rollout.durable-run.m0.21.start.json
```

commit対象は、正規生成されたartifactと、意味上説明可能なsource変更だけにします。

DSN、credential、runtime parameter値、一時Evidence fileを誤ってcommitしません。

#255でgenerated artifactを修復した場合は、dirty差分を残したままPhase 2へ進まず、意図したartifactをcommitします。

#214の観測事実からOracle realization等をauthoringした場合も、必要なValidationと差分確認を行い、Phase 2開始前にcommitします。

Phase 2へ入る時点ではworking treeをcleanにします。

## 9. Phase 2でcandidate revision Rへ4 laneを揃える

### 9.1 Rを固定する

```bash
git status --short
export TEI_GATE_B_CANDIDATE_REVISION="$(git rev-parse HEAD)"
printf '%s\n' "$TEI_GATE_B_CANDIDATE_REVISION"
```

`git status --short`が空でない場合はPhase 2を開始しません。

### 9.2 Rで#310を実行する

```bash
go run ./tools/reference-gate-b-repository-evidence \
  > /tmp/tei-gate-b-repository-evidence.json
```

成功後、JSONの`repositoryRevision`がRと一致し、`reproducibleRevision = true`であることを確認します。

### 9.3 Rで#255を実行する

```bash
go run ./tools/reference-generated-artifact-evidence \
  > /tmp/tei-gate-b-generated-artifact-evidence.json
```

この段階では`-write`しません。不一致が見つかった場合はPhase 2を中断し、修復 → commit後の新revisionを新しいRとして最初からやり直します。

### 9.4 Rで#308を実行する

```bash
go run ./tools/reference-postgres-header-detail-runtime \
  > /tmp/tei-gate-b-postgres-header-detail.json
```

Evidence専用の空PostgreSQL Databaseを使用し、JSONの`repositoryRevision`がRと一致することを確認します。fixture前後の`userRelations... = 0`、`eventTriggers... = 0`、fixture object不在も確認します。

### 9.5 Rで#214 acceptance / actual readを再確認する

```bash
go run ./tools/reference-external-query-evidence \
  -workspace reference/workspace \
  -evidence <observed-evidence.json> \
  > /tmp/tei-external-query-evidence-acceptance.json
```

receiptの`repositoryRevision`がRと一致し、`reproducibleRevision = true`であることを確認します。

続いて、Rのruntime / realizationに対してread-only Oracle connectionからactual readを行い、explicit Semantic mappingまで確認します。driverが提供したraw列metadataは、external Evidenceと実行時観測の対応確認に使いますが、未提供項目を推測して埋めません。

### 9.6 4 laneのrevision一致を確認する

少なくとも次を確認します。

```text
#310 repositoryRevision = R
#255 repositoryRevision = R
#308 repositoryRevision = R
#214 acceptance repositoryRevision = R
#214 actual read target source = R
```

一つでもRと異なる場合、そのlaneの古い成功EvidenceをRへ読み替えません。

Phase 2中にsource変更が必要になった場合はRを破棄し、変更をcommitした後、新しいRで9.1からやり直します。

## 10. Gate B監査へ戻す

取得したEvidenceを[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)へ対応付けます。

| Evidence lane | 判定対象 |
| --- | --- |
| #310 `reference-gate-b-repository-evidence` JSON | UI-less Scenario / external Evidence intake Conformance / Repository source固定Conformance / normal Go runner / source revision不変性 |
| #255 `reference-generated-artifact-evidence` JSON | World / Plugin Lock / Durable Runのclean revision再現性 |
| #308 PostgreSQL tool JSON | source-fixed real PostgreSQL engine / fixture前安全性（user relation / Event Trigger / fixture object）/ Provider runtime / cleanup後復元 |
| #214 external Evidence + acceptance receipt + actual Oracle Evidence | external-observed assetの実物理事実 / artifact identity / read-only / Oracle actual read / driver観測raw列metadata |

#214 acceptance receiptだけをactual external asset Evidence完了とは扱いません。

一つのEvidence成功を別種類のEvidenceの代替にしません。

さらに最終Ready判定では、4 laneの完了Evidenceがすべて同じcandidate revision Rへ揃っていることを確認します。

negative caseとして少なくとも次を再監査します。

- secret / credentialがEvidenceへ混入していない
- read-onlyがSQL文字列だけではなく実効的に強制されている
- 0件がExecution Errorへ化けていない
- Execution Errorが正常業務結果へ化けていない
- Observation期待不一致とread Execution Errorを混同していない
- Evidence run実行中にsource revisionが変化していない
- #308 cleanup後にEvidence専用空DBの観測条件（user relation 0件 / Event Trigger 0件 / fixture object不在）へ戻っている
- generated artifactを手編集していない
- external EvidenceをCanonical meaningへ昇格していない
- Oracle driverが提供していない列metadataを推測してEvidenceへ補完していない

## 停止条件

次の場合はfail closedで止めます。

- Workspace Validation Error
- #310完了Evidenceをdirty working treeから採用しようとしている
- #310の`repository-evidence-source-boundary` targeted checkが失敗する
- #310でpost-run確認前の途中失敗JSONを`reproducibleRevision=true`相当として扱おうとしている
- #310 runner実行前後でHEADが変わった、または実行後working treeがdirtyになった
- #255の`-write`直後のdirty treeを完了Evidenceとして採用しようとしている
- #255 `reference-generated-artifact-evidence`でgolden不一致がある
- #255 Evidence runner実行前後でHEADが変わった、またはworking treeがdirtyである
- #308開始時のworking treeがdirtyである
- #308実行前後でHEADが変わる、または実行後working treeがdirtyである
- #308で`postgres` / `template0` / `template1`へ接続している
- #308の接続先にsystem / temporary領域以外のuser relationが1件以上存在する
- #308の接続先にEvent Triggerが1件以上存在する
- `public.order_lines`が既に存在するPostgreSQL接続先
- #308 cleanup後にuser relationが残る
- #308 cleanup後にEvent Triggerが存在する
- #308 cleanup後に`public.order_lines`が存在する
- #214 acceptance開始時のworking treeがdirtyである
- #214 acceptance実行中にHEADが変わる、または実行後working treeがdirtyである
- #214でprovenanceのない物理情報しかない
- #214でSQL realizationが存在せず、Evidenceから補完しないと進めない
- #214 acceptance receiptだけをactual read完了として扱おうとしている
- #214 actual readでdriverが提供していない型 / NULL metadataを推測して補完しないと進めない
- read-only保証をSQL文字列だけで主張している
- generated artifactがstaleなのにfingerprintやJSON断片だけを手編集しようとしている
- fixed observed fixtureを生成結果へ合わせて機械的に変更しようとしている
- EvidenceとSQL realizationが一致しない
- 実行失敗を0件・正常Business Resultへ変換しないと先へ進めない
- Phase 2で4 laneの`repositoryRevision`がcandidate Rへ揃っていない
- Phase 2中にsource変更が生じたのに、古いRの成功Evidenceを流用しようとしている

これらは「作業が止まった」のではなく、Gate Bが意図どおり不足を露出させたEvidenceとして扱います。

## Gate B通過後

4 laneの完了Evidenceが同じrevision Rへ揃い、本監査でRについてGate BをReadyと判定した後も、すぐに最新`main`をWeb baselineへしません。

`GATE_B_TO_C_HANDOVER_CONTRACT.md` / #291に従い、RとGate B EvidenceをHandover Baselineとして固定してから第一Web Target #70 / Gate C #270へ進みます。

Application / Driverの意味が後から変わった場合は既存baselineへ黙って上書きせず、影響するGate B Evidenceを再確認し、4 laneを新しいrevisionへ揃えてから新しいHandover Baselineを発行します。
