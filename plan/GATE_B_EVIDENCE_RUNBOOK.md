# Gate B Evidence Runbook

## 目的

Gate Bで残っている作業を、設計議論へ戻さず**再現可能なEvidence取得**として実行するための正規手順です。

Evidenceの必要性・証明範囲・完了条件は[`GATE_B_EVIDENCE_ACQUISITION.md`](GATE_B_EVIDENCE_ACQUISITION.md)、Gate BのReady / Not Ready判定は[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)を正とします。

**具体的なコマンド、実行順、差分確認、fail-closedの停止条件については本書を唯一の正本とします。** 他文書に同じコマンドが残っている場合、本書と矛盾する手順を採用しません。

2026-09-04時点では、Application Driverの主要意味境界、UI-less最終Scenario構造、external-observed Query Evidence受入境界、PostgreSQL fixtureの安全境界まで成立しています。

Gate Bを直接止めているEvidence laneは次の四つです。

1. `ISSUE-000022` / #214: actual Oracle `master_data`物理・provenance・read-only Evidence
2. `ISSUE-000031` / #308: real PostgreSQL engine + ephemeral fixture Evidence
3. `ISSUE-000025` / #255: World / Plugin Lock / Durable Runのgenerated artifact再現Evidence
4. `ISSUE-000032` / #310: deterministic Repository / normal Go runner Evidence

GitHub Actionsが別判断で再開された場合の継続的regression確認は重要ですが、この四laneのProduct Evidenceそのものとは分けます。

## Evidenceの種類

### A. deterministic repository Evidence

Repository checkoutとGo toolchainだけで再現できるApplication Driver / Evidence intakeのEvidenceです。

- Workspace Validation
- UI-less synthetic Scenario
- external Evidence intake tests

通常runner Evidenceの一括入口は`tools/reference-gate-b-repository-evidence`です。

### B. generated artifact reproducibility Evidence

Canonical Workspace等の正規入力から、保存generated artifactを同じclean revisionで再現できることのEvidenceです。

- World / Plugin Lock
- Durable Run
- 実行前後のsource revision不変性

#255の最終完了Evidence入口は`tools/reference-generated-artifact-evidence`です。`tools/reference-golden`と`tools/reference-durable-run-golden`は個別check / repairの正規toolとして残します。

### C. real-provider fixture Evidence

実provider engineを使いますが、toolが作るephemeral fixtureに対するEvidenceです。

- `tools/reference-postgres-header-detail-runtime`

これはproduction既存資産の物理事実を観測したEvidenceではありません。

### D. actual external asset Evidence

TEIが所有していない既存資産の観測事実です。

- #214 `master_data` Oracle physical object / parameter / raw columns / native source / provenance / read-only proof

これはsynthetic / generated artifact / ephemeral fixtureで代替しません。

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
- GitHub Actionsをこの手順から再有効化しない

## 0. 実行前確認

Repository rootで作業します。

```bash
git status --short
git rev-parse HEAD
go version
```

Evidenceを特定revisionへ固定するため、意図しない未commit差分がある場合は先に識別します。既存差分を機械的に破棄しません。

#310の完了Evidenceでは、実行前だけでなく**実行後も同じHEADかつclean working tree**であることを要求します。診断目的のdirty実行と、Gate Bへ採用する再現可能Evidenceを混同しません。

#255も最終的な完了Evidenceはclean revisionへ固定します。stale修復のための`-write`で生成物差分が生じた場合、そのdirty tree上のcheck成功だけでは完了とせず、差分を確認・commitした後のclean revisionで`tools/reference-generated-artifact-evidence`を再実行します。

#308と#214のEvidence toolも、Gate Bへ採用するrunでは開始前にclean Repository revisionを要求し、実行後にHEAD / working treeを再確認します。DBやexternal Evidenceの検査が成功していても、source revisionを固定できないrunは完了Evidenceにしません。

## 1. Canonical WorkspaceをValidationする

```bash
go run ./cmd/tei validate -workspace reference/workspace
```

期待値:

```text
ok
```

Validation Errorがある場合は、その後のgolden再生成やprovider / external Evidence実行へ進みません。生成物側を手編集して回避しません。

external Evidence tool、PostgreSQL Evidence tool、Repository Evidence runner、generated artifact Evidence runner、Durable Run golden tool自身も通常Validation Errorでfail closedします。ただし、最初の明示Validationは前提不整合を早期に分離するため残します。

## 2. #310 deterministic Repository Evidenceを取得する

### 正規入口

clean revisionで次を実行します。

```bash
go run ./tools/reference-gate-b-repository-evidence \
  > /tmp/tei-gate-b-repository-evidence.json
```

JSON保存先をRepository内の新規fileへ直接redirectしません。shellはcommand開始前にredirect先を作成するため、それ自体がworking treeをdirtyにし、runnerが意図どおりfail closedします。

このrunnerは次を一つのEvidenceへ束ねます。

1. 実行前repository revision
2. 実行前working tree clean
3. Go toolchain identity
4. Canonical Workspace Validation
5. 最終UI-less Scenario
6. external Evidence intake boundary tests
7. 実行後repository revision
8. 実行前後のHEAD同一性
9. 実行後working tree clean

`reproducibleRevision = true`になるには、実行前後でHEADが同一で、両時点のworking treeがcleanである必要があります。test自体が成功していても、並行操作やtest実行でsource状態が変わったrunはGate B完了Evidenceとして採用しません。

### 最終UI-less Scenario

runnerが実行する等価コマンドは次です。

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
 -> Command
 -> separate Observation read
```

synthetic adapterを含むため、成功してもactual Oracle / real PostgreSQL / 実外部状態のEvidenceにはしません。

### external Evidence intake

runnerが実行する等価コマンドは次です。

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
- native SQL fingerprint stale検知
- provenance必須
- read-only mechanism / Evidence参照必須
- EvidenceからSQL realizationを生成・補完しないこと
- Workspace Validation Errorがある前提でEvidenceを受理しないこと
- acceptance receiptへexternal Evidence本文を複製しないこと
- Evidence artifact fingerprintがexact bytesを識別すること

ここで実行するtestsは受入境界のConformanceです。実際の#214 acceptance receiptやactual Oracle readの代替ではありません。

Evidence formatとfingerprintの意味は`tools/reference-external-query-evidence/README.md`を参照します。

### 広いGo test

横断回帰も同じrunで確認できる環境では`-broad`を指定します。

```bash
go run ./tools/reference-gate-b-repository-evidence -broad \
  > /tmp/tei-gate-b-repository-evidence.json
```

これはtargeted Evidence成功後に`go test ./...`を別checkとして実行します。広いtestの失敗をtargeted Evidenceの成功へ畳み込まず、逆に未実行をgreenとも扱いません。

## 3. #255 generated artifactのcheck-only Evidenceを取得する

clean revisionで、まずRepositoryを書き換えずに最終Evidence runnerを実行します。

```bash
go run ./tools/reference-generated-artifact-evidence \
  > /tmp/tei-gate-b-generated-artifact-evidence.json
```

Repository内の新規fileへ直接redirectしません。shellがcommand開始前にredirect先を作成し、runnerの開始時clean条件を壊すためです。

このrunnerは**`-write`を呼びません**。一つのrunで次を確認します。

1. 実行前repository revision
2. 実行前working tree clean
3. Go toolchain identity
4. `go run ./tools/reference-golden`
5. `go run ./tools/reference-durable-run-golden`
6. 実行後repository revision
7. 実行前後のHEAD同一性
8. 実行後working tree clean
9. `reproducibleRevision`判定

成功条件:

- `reference-golden`が一致する
- `reference-durable-run-golden`が一致する
- 実行前後でHEADが同一
- 実行前後でworking treeがclean
- `reproducibleRevision = true`

最初から成功した場合は形式的に`-write`しません。このJSONを#255 Evidence候補として扱います。

不一致の場合は#255完了ではなく、staleを検出した診断結果です。runnerがstderrへ流した個別golden toolの診断を確認し、次の修復手順へ進みます。

### 個別診断: World / Plugin Lock

個別に確認する場合は次を実行します。

```bash
go run ./tools/reference-golden
```

このtoolはCanonical WorkspaceからWorld / Plugin Lockを生成して保存goldenと比較し、保存Durable RunのPlugin Lock依存も確認します。

一致している場合は形式的に`-write`しません。

不一致の場合は、表示されたpathを記録して次へ進みます。この時点では#255完了Evidenceではなく、staleを検出した診断結果です。

## 4. World / Plugin Lockを正規再生成する — staleの場合だけ

```bash
go run ./tools/reference-golden -write
```

このtoolが更新するのは次です。

```text
reference/world.json
reference/workspace/.tei/plugin-lock.json
```

更新後は必ず差分を確認します。

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

**この`-write`はstale解消の修復操作です。** この時点でworking treeがdirtyになった場合、その状態でcheckが通っても#255完了Evidenceにはしません。

## 5. Durable Run全体を正規生成結果と照合する

対象:

```text
reference/order_rollout.durable-run.m0.21.start.json
```

第一Referenceのidentity:

```text
solution: order_rollout
version: 2026.8.27
run-id: order-rollout-prod-m0.21
environment: production
```

PR #318以降、Durable Runは`tools/reference-durable-run-golden`を正規入口とします。

### check

Plugin Lock fingerprintだけでなく、Runtime Policyを含む`deployment-start`相当の生成結果全体と比較します。

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

PR #326以降、`-solution`へ不正なCanonical IDを与えた場合もpanicではなく通常errorでfail closedします。

### write — staleの場合だけ

```bash
go run ./tools/reference-durable-run-golden -write
```

このtoolは保存pathを先にtruncateしません。同じdirectoryの一時fileへ生成結果全体を書き、Sync / Close後にRenameして置換します。目的は生成途中の失敗で既存goldenを空にしないことであり、OS横断のRename atomicityそのものは契約にしません。

更新後:

```bash
git diff -- reference/order_rollout.durable-run.m0.21.start.json
```

確認観点:

- `pluginLockFingerprint`が現在Plugin Lockと整合する
- `runtimePolicyFingerprint`を含む差分が正規生成入力から説明できる
- fixed observed stateを生成結果へ合わせて書き換えていない
- fingerprintやJSONの一部だけを手編集していない

この`-write`も修復操作です。差分が生じた場合は、#255のfinal Evidenceへ進む前に生成物差分を確定します。

### staleを修復した場合の確定

World / Plugin Lock / Durable Runのいずれかに差分が生じた場合、次を行います。

1. `git diff`で差分が現在のCanonical authoring、Plugin Lock、fixed observed fixtureから説明できることを確認する
2. 意図したgenerated artifactだけをcommitする
3. DSN、credential、一時Evidence file、fixed observed fixture等が混入していないことを確認する
4. 新しいclean revisionからfinal checkをやり直す

`-write`直後のdirty treeを、最終Evidenceのsource revisionとして記録しません。

### #255最終確認 — clean revisionのcheck-only Evidence runner

修復差分をcommitした新しいclean revisionで、次を再実行します。

```bash
go run ./tools/reference-generated-artifact-evidence \
  > /tmp/tei-gate-b-generated-artifact-evidence.json
```

runnerが内部で`reference-golden`と`reference-durable-run-golden`をcheck-only実行し、実行前後のHEAD / working treeも確認します。

次をすべて満たした場合だけ、World / Plugin Lock / Durable Runのgenerated artifact再現性を#255 Evidence候補として扱います。

- final check開始時のworking treeがclean
- `reference-golden`が一致
- `reference-durable-run-golden`が一致
- final check実行前後でHEADが同一
- final check終了後もworking treeがclean
- `reproducibleRevision = true`

両toolがdirty tree上で一致しただけでは#255を完了にしません。

## 6. #308 実PostgreSQL engineでheader-detailを確認する

### Evidenceの性質

これは**real PostgreSQL engine + ephemeral fixture runtime Evidence**です。

production既存Databaseの実データEvidenceではありません。

### fail-closed前提

Evidence toolはRepository source、Canonical Workspace、Database状態の順にfail closedで確認します。

```text
Repository HEAD / working tree確認
  ↓ dirtyなら停止
Workspace Load
  ↓
Canonical Validation
  ↓ Errorなら停止
PostgreSQL接続
  ↓
current_database()でDatabase identity確認
  ↓
postgres / template0 / template1なら停止
  ↓
system / temporary領域を除くuser relation件数確認
  ↓ 1件以上なら停止
public.order_lines既存確認
  ↓ 既存なら停止
Ephemeral fixture作成
  ↓
2件 / 0件 Provider runtime Evidence
  ↓
fixture DROP
  ↓
missing-table Execution Error
  ↓
user relation 0件 / public.order_lines不在を再観測
  ↓
Repository HEAD / working tree再確認
  ↓
JSON Evidence
```

`TEI_TEST_POSTGRES_DSN`には、Evidence専用に作成した空PostgreSQL Databaseを指定します。

```bash
export TEI_TEST_POSTGRES_DSN='postgres://...'
```

DSN、credential、接続先の実Database名をEvidenceファイルへ保存しません。

### Repository source固定

PR #336以降、toolはDB接続より前にclean Repository revisionを要求し、Provider runtime / cleanup / Database復元確認後にHEAD / working treeを再確認します。

成功Evidenceには少なくとも次を含みます。

- `repositoryRevision`
- `workingTreeClean = true`
- `reproducibleRevision = true`
- post-run Repository revision / `revisionStable` / working tree clean

実行中にHEADが変わったrun、または実行後にtracked / untracked差分が残るrunは成功Evidenceとして出しません。

### Database側の安全条件

#### 既定Databaseを拒否する

次のPostgreSQL既定Databaseはfixture実行先として利用しません。

```text
postgres
template0
template1
```

`postgres`が偶然空であっても拒否します。

#### 既存user relationを拒否する

`pg_catalog` / `information_schema` / PostgreSQL内部・temporary領域を除き、次のrelationが一つでも存在すれば停止します。

- table / partitioned table
- view / materialized view
- foreign table
- sequence

既存relationの内容や用途を推測せず、削除して空Databaseへ転用する処理も提供しません。

#### fixture objectを再確認する

空Database検査後も`public.order_lines`の既存有無を明示確認します。

```sql
SELECT to_regclass('public.order_lines')::text
```

既存objectがあればfail closedで停止します。

これらの検査を通過した場合だけtool-owned fixtureをCREATEします。**既存relationやtableを削除して続行するoverrideはありません。** 別の専用空Databaseを使用します。

### fixture作成前安全Evidence

PR #335以降、fixture作成前に満たした非機密な安全条件を`fixtureDatabaseSafety`へ残します。

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

これはDatabase名そのものを保存せず、toolが観測可能なfixture前安全条件をそのrunで満たしたことを示します。「組織運用上Evidence専用であること」の完全証明へ格上げしません。

### cleanup後安全Evidence

PR #341以降、missing-table Execution Error確認後にDatabase catalogを再観測し、成功JSONを出す前に空状態へ戻ったことを確認します。

期待値:

```json
{
  "verified": true,
  "userRelationsAfterCleanup": 0,
  "fixtureObjectAbsentAfterCleanup": true
}
```

終了時にuser relationが残る場合、それがtool由来か並行変更かを推測せずfail closedで停止します。

### 実行

```bash
go run ./tools/reference-postgres-header-detail-runtime \
  | tee /tmp/tei-gate-b-postgres-header-detail.json
```

期待するEvidence:

- `evidenceNature = real-postgres-ephemeral-fixture`
- `repositoryRevision`を保持する
- `workingTreeClean = true`
- `reproducibleRevision = true`
- 実行前後でHEAD同一 / working tree clean
- `fixtureObject = public.order_lines`
- `preexistingFixtureProtected = true`
- `fixtureOwnedByTool = true`
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
- `fixtureDatabaseSafety.guardPassed = true`
- `fixtureDatabaseSafety.nonDefaultDatabaseVerified = true`
- `fixtureDatabaseSafety.userRelationsBeforeFixture = 0`
- `fixtureDatabaseSafety.fixtureObjectAbsentBeforeCreate = true`
- `fixtureDatabaseSafety.databaseIdentityPersisted = false`
- `fixtureCleanupSafety.verified = true`
- `fixtureCleanupSafety.userRelationsAfterCleanup = 0`
- `fixtureCleanupSafety.fixtureObjectAbsentAfterCleanup = true`
- Database名 / DSN / credentialを保存していない

出力はまず外部Evidence artifactとして確認します。保存先をRepository標準として固定する判断は、このrunbookから先行しません。

## 7. #214 actual Oracle Evidenceを受け取る

この工程は、実外部事実が得られるまで**Blockedのまま保持**して構いません。

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

未観測の`CUSTOMER`、`CUSTOMER_ID`等を仮定しません。

### Evidence取得後

1. 観測事実をEvidence JSONへ記録する
2. 観測事実を根拠に`ContractQuerySQLRealization`を明示authoringする
3. native sourceを通常Workspaceから追跡可能にする
4. authoring差分があればcommitし、clean revisionへ固定する
5. clean revisionでEvidenceとauthoringを照合し、acceptance receiptを取得する

```bash
go run ./tools/reference-external-query-evidence \
  -workspace reference/workspace \
  -evidence <observed-evidence.json> \
  > /tmp/tei-external-query-evidence-acceptance.json
```

Repository内の新規fileへ直接redirectしません。shellがcommand開始前にfileを作ってworking treeをdirtyにするためです。

このtoolはEvidenceを読む前にclean Repository revisionを要求し、照合後にもHEAD / working treeを再確認します。Workspace Validation ErrorがあればEvidence照合前に停止し、SQL realizationがなければEvidenceから生成せず停止します。

成功receiptで少なくとも次を確認します。

- `evidenceNature = external-query-evidence-acceptance`
- `repositoryRevision`
- `workingTreeClean = true`
- `reproducibleRevision = true`
- post-run Repository revisionが開始時と同一
- `evidenceArtifactFingerprint = sha256:<hex>`
- target / provider
- `accepted = true`

`evidenceArtifactFingerprint`はEvidence JSONのexact bytesを識別します。`nativeSource.fingerprint`は現在authoring済みSQL本文との一致を確認する別fingerprintであり、相互代替しません。

receiptへphysical object、parameter / raw columns、provenance本文、read-only mechanism本文、Evidence file path、credential / secretを複製しません。

**acceptance receiptはactual Oracle read Evidenceではありません。** 受入照合が成立した後に、actual runtimeを別工程で確認します。

6. read-onlyを実効強制したOracle connectionでruntime Executorを構成する
7. 採用Oracle driverでnamed parameter / raw result / NULL / error伝播を確認する
8. Workspace → actual read → Semantic resultを一続きに確認する
9. 1件 / 0件 / Execution Errorを区別する

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

#255でgenerated artifactを修復した場合は、ここでdirty差分を残したままGate B監査へ進まず、意図したartifactをcommitしてclean revisionへ固定し、`tools/reference-generated-artifact-evidence`を再実行します。

#214でOracle realizationをauthoringした場合も、dirty treeのままacceptance receiptを取得せず、意図したauthoringをcommitしたclean revisionから受入照合します。

## 9. Gate B監査へ戻す

取得したEvidenceを[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)へ対応付けます。

少なくとも次を独立に判定します。

| Evidence lane | 判定対象 |
| --- | --- |
| #310 `reference-gate-b-repository-evidence` JSON | UI-less Scenario / external Evidence intake / normal Go runner / source revision不変性 |
| #255 `reference-generated-artifact-evidence` JSON | World / Plugin Lock / Durable Runのclean revision再現性 |
| #308 PostgreSQL Evidence JSON | real PostgreSQL engine、fixture前安全、runtime、cleanup後空状態、source revision不変性 |
| #214 observed Evidence + acceptance receipt + actual runtime Evidence | external-observed assetの実物理事実、受入artifact identity、read-only境界、actual read |

#255の個別`reference-golden` / `reference-durable-run-golden`はcheck / repairの正規toolですが、Gate Bへ対応付ける最終Evidenceは`reference-generated-artifact-evidence`のclean check-only runとします。

#214のacceptance receiptは観測Evidenceとauthoringの照合証跡であり、actual Oracle runtime Evidenceを代替しません。

一つのEvidenceが成功したことを、別種類のEvidenceの代替にしません。

## 停止条件

次の場合はfail closedで止めます。

- Workspace Validation Error
- #310完了Evidenceをdirty working treeから採用しようとしている
- #310 runner実行前後でHEADが変わった、または実行後working treeがdirtyになった
- #255の`-write`直後のdirty treeを完了Evidenceとして採用しようとしている
- #255 `reference-generated-artifact-evidence`でgolden不一致がある
- #255 Evidence runner実行前後でHEADが変わった、またはworking treeがdirtyである
- #308をdirty working treeから開始しようとしている
- #308で`postgres` / `template0` / `template1`へ接続している
- #308の接続先にsystem / temporary領域以外のuser relationが1件以上存在する
- `public.order_lines`が既に存在するPostgreSQL接続先
- #308 cleanup後にuser relationが0件へ戻らない、または`public.order_lines`が残っている
- #308実行前後でHEADが変わった、または実行後working treeがdirtyである
- #214をdirty working treeから受入れようとしている
- #214でprovenanceのない物理情報しかない
- #214 acceptance receiptの`reproducibleRevision`がtrueでない
- #214 Evidence artifactをfingerprintで特定できない
- acceptance receiptだけをactual Oracle read Evidenceとして扱おうとしている
- read-only保証をSQL文字列だけで主張している
- generated artifactがstaleなのにfingerprintやJSON断片だけを手編集しようとしている
- fixed observed fixtureを生成結果へ合わせて機械的に変更しようとしている
- EvidenceとSQL realizationが一致しない
- 実行失敗を0件・正常Business Resultへ変換しないと先へ進めない

これらは「作業が止まった」のではなく、Gate Bが意図どおり不足を露出させたEvidenceとして扱います。