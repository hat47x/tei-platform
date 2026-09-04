# Gate B Evidence Runbook

## 目的

Gate Bで残っている作業を、設計議論へ戻さず**再現可能なEvidence取得**として実行するための正規手順です。

Evidenceの必要性・証明範囲・完了条件は[`GATE_B_EVIDENCE_ACQUISITION.md`](GATE_B_EVIDENCE_ACQUISITION.md)、取得済みEvidenceからのReady / Not Ready判定は[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)を正とします。

**具体的なコマンド、実行順、差分確認、fail-closed停止条件については本書を唯一の正本とします。** 他文書に同じコマンドが残っている場合、本書と矛盾する手順を採用しません。

2026-09-04時点では、Application Driverの主要意味境界、最終UI-less Scenario構造、external-observed Query Evidence受入境界、PostgreSQL fixtureの安全境界まで成立しています。

PR #379を起点とした`ISSUE-000035` / #381の再監査により、Gate Bを直接止める必須Evidenceは次の3本へ収束しました。

1. `ISSUE-000032` / #310: deterministic Repository / normal Go runner Evidence
2. `ISSUE-000025` / #255: generated artifact reproducibility Evidence
3. `ISSUE-000031` / #308: real PostgreSQL engine + ephemeral fixture Evidence

actual external Oracleは継続Evidence trackとして残します。

- `ISSUE-000022` / #214: actual Oracle `master_data`物理・provenance・actual read
- `ISSUE-000033` / #371: actual Oracle driver / Host assembly
- `ISSUE-000034` / #376: actual Oracle read-only Evidence

#214 / #371 / #376は廃止しません。actual Oracle compatibility / actual external asset integrationを主張するために必要ですが、外部環境情報が得られないことだけでGate Bを止めません。

GitHub Actionsによる継続的なregression確認は重要ですが、このProduct Evidenceとは分けます。現行RepositoryではActionsを意図的に使用しておらず、本Runbookから再有効化しません。

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

production既存Databaseの物理事実やactual Oracle Evidenceではありません。

### D. actual external asset Evidence — #214（Gate Bでは継続track）

TEIが所有していない既存Oracle資産の観測事実とactual readです。

- physical object / parameter / raw columns
- raw column physical metadata / NULL上の扱い
- native realization source
- provenance
- #376 effective read-only Evidence
- #371 Oracle driver / Host assembly
- Workspace → actual read → Semantic result

Gate Bの必須完了条件ではありません。未完了ならHandoverへ`unresolved / not verified`として残します。

## Gate B Evidence取得は二段階で行う

最終Ready判定では、一つのcandidate repository revision `R`を固定し、**#310 / #255 / #308の3必須EvidenceをRへ揃えます。**

```text
#310 完了Evidence ─┐
#255 完了Evidence ─┼─ repositoryRevision = R
#308 完了Evidence ─┘
                    ↓
             Gate B Ready for R
                    ↓
             Handover Baseline R
```

#214 actual Oracleはこの必須収束対象ではありません。

### Phase 1 — 事前確認 / sourceを確定する

1. #310をcleanな通常Go runnerで事前実行する
2. #255をcheck-onlyし、staleなら正規toolで修復する
3. #308をEvidence専用の空PostgreSQL Databaseで事前実行する
4. Gate B必須Evidenceの不足からsource / generated artifact変更が必要ならcommitする
5. working treeをcleanにする

#214 / #371 / #376はactual Oracle環境情報が得られた時点で並行継続します。Gate Bを通すためだけに未観測Oracle情報を仮定しません。

Phase 1で成功JSONを得ても、その後repository sourceが変わった場合、そのJSONを最終完了Evidenceとして流用しません。

### Phase 2 — 最終収束 / candidate revision Rを固定する

1. sourceとgenerated artifactを確定したclean revisionをRとして選ぶ
2. Rで#310を実行する
3. Rで#255をcheck-only実行する
4. Rで#308を実行する
5. #310 / #255 / #308の完了EvidenceがRへ揃っていることを確認する
6. #214 actual Oracleの状態を`verified`または`unresolved / not verified`として明示する
7. negative caseを再監査する
8. Rを対象にGate BをReady / Not Readyで判定する
9. Readyなら#291でRをHandover Baselineとして固定する

Phase 2中に必須Evidenceがsource変更を要求した場合は、その変更をcommitした新revisionを新しいRとして選び直し、3 Evidenceをそのrevisionへもう一度揃えます。

#214がGate B判定前に完了した場合は、actual external Evidence / acceptance receipt / actual readをRへ追跡可能にしてHandoverへ含めます。未完了なら#214 / #371 / #376をunresolvedとして残します。

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
- Gate Bへ採用する必須Evidenceは、後から同じsourceへ戻れるclean revisionへ固定する
- Evidence run実行前後でsource revisionが変化した場合は完了Evidenceとして採用しない
- stdoutでEvidenceを保存するとき、Repository内の新規fileをredirect先にして開始前clean条件を壊さない
- Phase 1の成功Evidenceを、source変更後のPhase 2完了Evidenceへ読み替えない
- #214未完了をactual Oracle成功へ読み替えない
- #310 synthetic Scenarioや#308 PostgreSQL Evidenceをactual Oracle Evidenceへ格上げしない
- Web側でunresolved Oracle物理情報を補完しない
- read-only確認のためだけに実外部assetへDML / DDL / `SELECT ... FOR UPDATE` / exclusive lock等を試行しない
- GitHub Actionsを本手順から再有効化しない

## 0. 実行前確認

Repository rootで作業します。

```bash
git status --short
git rev-parse HEAD
go version
```

Evidenceを特定revisionへ固定するため、意図しない未commit差分がある場合は先に識別します。既存差分を機械的に破棄しません。

#310 / #255 / #308の完了Evidenceは、開始時cleanだけでなく、toolが要求する範囲で**実行後も同じHEADかつclean working tree**であることを要求します。

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

Validation Errorがある場合は、その後のgolden再生成やprovider Evidence実行へ進みません。生成物側を手編集して回避しません。

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

PR #351以降、開始時working treeがcleanであっても、post-run確認が完了するまでは`reproducibleRevision=false`を維持します。

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

synthetic adapterを含むため、成功してもactual Oracle / real PostgreSQL / 実外部Command Evidenceにはしません。

Customer referenceの意味・identity・result boundaryをRepositoryだけから再現できることを確認しますが、actual Oracle compatibilityは主張しません。

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

このcheckは、actual Oracleが未取得でも**未観測外部事実を推測しないfail-closed境界**をGate Bの必須Evidenceとして確認するものです。

ここでのtest成功は#214 actual external asset Evidenceではありません。

### Repository source固定境界

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

### 広いGo test

横断回帰も同じrunで確認できる環境では`-broad`を指定します。

```bash
go run ./tools/reference-gate-b-repository-evidence -broad \
  > /tmp/tei-gate-b-repository-evidence.json
```

未実行をgreenとは扱いません。

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

成功条件:

- `reference-golden`が一致する
- `reference-durable-run-golden`が一致する
- 実行前後でHEADが同一
- 実行前後でworking treeがclean
- `reproducibleRevision = true`

最初から成功した場合は形式的に`-write`しません。

### 個別診断: World / Plugin Lock

```bash
go run ./tools/reference-golden
```

一致している場合は形式的に`-write`しません。

## 4. World / Plugin Lockを正規再生成する — staleの場合だけ

```bash
go run ./tools/reference-golden -write
```

PR #367以降、World / Plugin Lockの修復writeも既存goldenを最初にtruncateしません。同じdirectoryの一時fileへ全内容を書き込み、Sync / Closeした後にRenameし、一時fileは成功・失敗のどちらでもcleanupします。

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

actual OracleをGate B必須から外しても、この実Provider Evidenceは必須に残します。Gate Bをsynthetic Evidenceだけへ縮退させないためです。

### source revisionのfail-closed境界

PR #336以降、Databaseへ接続する前にRepository sourceを確認します。

```text
git rev-parse HEAD
git status --porcelain
```

working treeがdirtyならWorkspace ValidationやPostgreSQL接続へ進みません。

Provider Evidence、fixture cleanup、Database復元確認の後にもHEAD / working treeを再確認します。

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

`TEI_TEST_POSTGRES_DSN`には**Evidence専用に作成した空PostgreSQL Database**を指定します。

```bash
export TEI_TEST_POSTGRES_DSN='postgres://...'
```

DSN、credential、接続先の実Database名を通常Evidenceへ保存しません。

### fixture前安全判定Evidence

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

### 実行

```bash
go run ./tools/reference-postgres-header-detail-runtime \
  > /tmp/tei-gate-b-postgres-header-detail.json
```

Repository内へ直接redirectしません。

期待するProvider Evidence:

- `evidenceNature = real-postgres-ephemeral-fixture`
- `repositoryRevision`
- `reproducibleRevision = true`
- fixture前安全条件
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
- Relation Storage column = `order_id`
- Semantic mapping applied = true
- support columnをSemantic attributeへ追加していない

### fixture cleanup / missing-table Error / 空状態復元

Providerの2件 / 0件を確認した後、tool自身が作成した`public.order_lines`をDROPし、同じContract Queryを再実行してphysical table欠落をExecution Errorとして観測します。

その後PostgreSQL catalogを再観測します。

期待値:

```json
{
  "verified": true,
  "userRelationsAfterCleanup": 0,
  "eventTriggersAfterCleanup": 0,
  "fixtureObjectAbsentAfterCleanup": true
}
```

終了時にuser relationまたはEvent Triggerが残る場合、それがtool由来か他主体の並行変更かを推測せずfail closedします。

## 7. #214 actual Oracle Evidence — 継続track

この工程は実外部事実が得られるまで**Blockedのまま保持して構いません**。Gate B必須Evidenceではありません。

#214 / #371 / #376を「Gate Bから外れたので不要」と扱いません。actual Oracle compatibility / actual `master_data` integrationを主張する前に必ず継続します。

最低限必要な観測事実:

- physical object kind / identity
- Contract input `id`に対応するphysical parameter
- parameter type / NULL上の扱い
- raw result columns
- raw column type / NULL上の扱い
- raw → `customer.customer` Semantic mapping
- native Oracle SQLまたは他realization source
- source / provenance
- #376 principal / role / grant等のread-only Evidence
- #371 actual Oracle driver / Host assemblyのparameter / result / NULL / error伝播Evidence

未観測の`CUSTOMER`、`CUSTOMER_ID`等を仮定しません。

### #376 read-only Evidenceを安全に確定する

read-only保証は、**実外部assetへ試し書きして失敗を確認する方法では取得しません。**

第一Referenceでは、actual Oracle version / grant構成に応じて少なくとも次を確認します。

1. 専用least-privilege principal / roleの構成
2. 対象objectへのquery privilegeと、不要なDML / DDL系privilegeがないことを再確認できるEvidence参照
3. indirect role / PUBLIC等を必要範囲で確認できるgrant Evidence
4. 採用する場合のread-only transaction / session境界
5. TEI Reference read pathがwrite用`Exec`を露出させないquery-only runtime境界
6. actual native Queryが上記境界で成功すること

principal / grant Evidenceを主たるread-only根拠とし、transaction / sessionやTEI側query-only runtimeを追加防御として扱います。

次のprobeはread-only確認目的では実行しません。

- `INSERT`
- `UPDATE`
- `DELETE`
- `MERGE`
- `CREATE` / `ALTER` / `DROP`
- `SELECT ... FOR UPDATE`
- exclusive `LOCK TABLE`

### #371 actual Oracle driver / Host assemblyを確定する

具体driverの選択・登録、DSN、credential、TLS / TCPS、Wallet、connection設定はHost / Reference assembly側で確定します。Canonical modelやOracle Provider Coreへ具体driverを固定しません。

実Oracle version / authentication / encryption / datatype / NULL / Host制約を観測してから採用driverを決定します。

### Evidence取得後のauthoring / acceptance / actual read

actual Oracle情報が得られた場合は従来のfail-closed手順を維持します。

1. 観測事実をexternal Evidence JSONへ記録する
2. 観測事実を根拠に`ContractQuerySQLRealization`を明示authoringする
3. native sourceを通常Workspaceから追跡可能にする
4. #371のdriver / Host assemblyにsource変更がある場合は明示authoringする
5. clean revisionで`tools/reference-external-query-evidence`を実行する
6. source-fixed acceptance receiptを取得する
7. #376のread-only境界からactual Oracle connectionを構成する
8. #371のdriverでactual readを行う
9. raw result → Semantic Customerを明示mappingする

acceptance receiptをactual Oracle read Evidenceへ格上げしません。

### Gate B / Handoverとの関係

Gate B判定時点で#214が完了している場合は、candidate Rのauthoring / runtimeへacceptance receiptとactual readを結び付けてHandoverへ含めます。

未完了の場合は、Handoverへ次をunresolvedとして残します。

- #214 actual external Oracle Evidence: not verified
- #371 actual Oracle driver / Host assembly: 未確定または未検証
- #376 actual Oracle read-only Evidence: 未取得
- actual Oracle compatibilityを未検証とするclaim boundary

Web側でこのunresolvedをOracle物理情報の仮置きにより解消しません。

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

#214の継続作業でsource変更が生じても、Gate Cへその変更を取り込むかはHandover契約に従って判断します。Gate Bを通すためだけに未観測Oracle情報をauthoringしません。

Phase 2へ入る時点ではworking treeをcleanにします。

## 9. Phase 2でcandidate revision Rへ3必須Evidenceを揃える

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

### 9.5 3必須Evidenceのrevision一致を確認する

```text
#310 repositoryRevision = R
#255 repositoryRevision = R
#308 repositoryRevision = R
```

一つでもRと異なる場合、その古い成功EvidenceをRへ読み替えません。

Phase 2中にsource変更が必要になった場合はRを破棄し、変更をcommitした後、新しいRで9.1からやり直します。

### 9.6 #214 actual Oracle状態を記録する

Gate B最終監査前に、#214を次のどちらかへ分類します。

```text
verified
unresolved / not verified
```

`verified`の場合はactual external Evidence / acceptance receipt / actual readへ戻れることを確認します。

`unresolved`の場合は#214 / #371 / #376とclaim制約をHandoverへ残します。

## 10. Gate B監査へ戻す

取得したEvidenceを[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)へ対応付けます。

| Evidence | 判定対象 |
| --- | --- |
| #310 `reference-gate-b-repository-evidence` JSON | UI-less Scenario / external Evidence intake Conformance / Repository source固定Conformance / normal Go runner / source revision不変性 |
| #255 `reference-generated-artifact-evidence` JSON | World / Plugin Lock / Durable Runのclean revision再現性 |
| #308 PostgreSQL tool JSON | source-fixed real PostgreSQL engine / fixture前安全性 / Provider runtime / cleanup後復元 |
| #214 status | actual external Oracleがverifiedかunresolvedか、Oracle compatibility claim境界 |

#214がunresolvedでもGate Bを自動的にNot Readyへ戻しません。ただしunresolvedを隠したり、#310 / #308で代替したりしません。

negative caseとして少なくとも次を再監査します。

- secret / credentialがEvidenceへ混入していない
- 0件がExecution Errorへ化けていない
- Execution Errorが正常業務結果へ化けていない
- Observation期待不一致とread Execution Errorを混同していない
- Evidence run実行中にsource revisionが変化していない
- #308 cleanup後にEvidence専用空DBの観測条件へ戻っている
- generated artifactを手編集していない
- external EvidenceをCanonical meaningへ昇格していない
- #214 unresolvedなのにOracle compatibilityを実証済みと表現していない
- Web側でOracle object / column / datatype / read-only方式を補う前提をHandoverへ持ち込んでいない

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
- #308 cleanup後にuser relation / Event Trigger / fixture objectが残る
- generated artifactがstaleなのにfingerprintやJSON断片だけを手編集しようとしている
- fixed observed fixtureを生成結果へ合わせて機械的に変更しようとしている
- 実行失敗を0件・正常Business Resultへ変換しないと先へ進めない
- Phase 2で#310 / #255 / #308の`repositoryRevision`がcandidate Rへ揃っていない
- Phase 2中にsource変更が生じたのに、古いRの成功Evidenceを流用しようとしている
- #214 unresolvedなのにactual Oracle compatibility / `master_data` actual integrationを実証済みとして扱おうとしている
- #310 synthetic Scenarioや#308 PostgreSQL Evidenceをactual Oracle Evidenceへ格上げしようとしている
- Web側でOracle物理情報を仮置きしないと第一Targetへ進めない状態をGate B Readyとみなそうとしている

### #214継続trackの停止条件

#214を実行する場合は、従来のfail-closed安全条件を維持します。

- provenanceのない物理情報しかない
- SQL realizationが存在せず、Evidenceから補完しないと進めない
- acceptance receiptだけをactual read完了として扱おうとしている
- driverが提供していない型 / NULL metadataを推測しないと進めない
- #371でactual Oracle条件を確認せずdriver / Host assemblyを固定しようとしている
- #376でprincipal / grant Evidenceを確認せずread-only全体を主張している
- read-only確認目的で実外部assetへDML / DDL / `SELECT ... FOR UPDATE` / exclusive lock等を試行しようとしている

これらは「作業が止まった」のではなく、不足やunresolvedが意図どおり露出した状態として扱います。

## Gate B通過後

#310 / #255 / #308の完了Evidenceが同じrevision Rへ揃い、本監査でRについてGate BをReadyと判定した後も、すぐに最新`main`をWeb baselineへしません。

`GATE_B_TO_C_HANDOVER_CONTRACT.md` / #291に従い、RとGate B EvidenceをHandover Baselineとして固定し、#214 actual Oracleのverified / unresolved状態を明示してから第一Web Target #70 / Gate C #270へ進みます。

Application / Driverの意味が後から変わった場合は既存baselineへ黙って上書きせず、影響するGate B Evidenceを再確認して新しいHandover Baselineを発行します。
