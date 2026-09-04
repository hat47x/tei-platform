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

Repository checkoutとGo toolchainだけで再現できるものです。

- Workspace Validation
- UI-less synthetic Scenario
- external Evidence intake tests
- World / Plugin Lock再生成
- Durable Run再生成

通常runner Evidenceの一括入口は`tools/reference-gate-b-repository-evidence`です。generated artifactの再現は#255として独立に判定します。

### B. real-provider fixture Evidence

実provider engineを使いますが、toolが作るephemeral fixtureに対するEvidenceです。

- `tools/reference-postgres-header-detail-runtime`

これはproduction既存資産の物理事実を観測したEvidenceではありません。

### C. actual external asset Evidence

TEIが所有していない既存資産の観測事実です。

- #214 `master_data` Oracle physical object / parameter / raw columns / native source / provenance / read-only proof

これはsynthetic / ephemeral fixtureで代替しません。

## 共通原則

- 未観測の外部事実を名前や慣習から補完しない
- Evidence実行前にCanonical WorkspaceのValidityを確認する
- 生成物を手編集しない
- fingerprintを人手で置換しない
- runtime credential / DSN / parameter値を通常Evidenceへ保存しない
- synthetic Evidenceを実外部Evidenceへ格上げしない
- 失敗を0件・NotSatisfied・denied等の正常業務結果へ変換しない
- 一致済み生成物を形式的に書き換えない
- generated artifactの`-write`を完了Evidenceそのものとして扱わない
- Gate Bへ採用するEvidenceは、後から同じsourceへ戻れるrevisionへ固定する
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

#255も最終的な完了Evidenceはclean revisionへ固定します。stale修復のための`-write`で生成物差分が生じた場合、そのdirty tree上のcheck成功だけでは完了とせず、差分を確認・commitした後のclean revisionでcheck-onlyを再実行します。

## 1. Canonical WorkspaceをValidationする

```bash
go run ./cmd/tei validate -workspace reference/workspace
```

期待値:

```text
ok
```

Validation Errorがある場合は、その後のgolden再生成やprovider / external Evidence実行へ進みません。生成物側を手編集して回避しません。

external Evidence tool、PostgreSQL Evidence tool、Repository Evidence runner、Durable Run golden tool自身も通常Validation Errorでfail closedします。ただし、最初の明示Validationは前提不整合を早期に分離するため残します。

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

Evidence formatとfingerprintの意味は`tools/reference-external-query-evidence/README.md`を参照します。

### 広いGo test

横断回帰も同じrunで確認できる環境では`-broad`を指定します。

```bash
go run ./tools/reference-gate-b-repository-evidence -broad \
  > /tmp/tei-gate-b-repository-evidence.json
```

これはtargeted Evidence成功後に`go test ./...`を別checkとして実行します。広いtestの失敗をtargeted Evidenceの成功へ畳み込まず、逆に未実行をgreenとも扱いません。

## 3. #255 World / Plugin Lockの現在差分を確認する

clean revisionで、まずRepositoryを書き換えずに確認します。

```bash
go run ./tools/reference-golden
```

このtoolはCanonical WorkspaceからWorld / Plugin Lockを生成して保存goldenと比較し、保存Durable RunのPlugin Lock依存も確認します。

一致している場合は形式的に`-write`しません。Durable Run全体のcheckも行い、両toolが一致し、実行前後で同じHEADかつcleanであることを確認できれば、そのrevisionを#255 Evidence候補にできます。

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

### #255最終確認 — clean revisionのcheck-only

```bash
git status --short
git rev-parse HEAD

go run ./tools/reference-golden
go run ./tools/reference-durable-run-golden

git rev-parse HEAD
git status --short
```

次をすべて満たした場合だけ、World / Plugin Lock / Durable Runのgenerated artifact再現性を#255 Evidence候補として扱います。

- final check開始時のworking treeがclean
- `reference-golden`が一致
- `reference-durable-run-golden`が一致
- final check実行前後でHEADが同一
- final check終了後もworking treeがclean

両toolがdirty tree上で一致しただけでは#255を完了にしません。

## 6. #308 実PostgreSQL engineでheader-detailを確認する

### Evidenceの性質

これは**real PostgreSQL engine + ephemeral fixture runtime Evidence**です。

production既存Databaseの実データEvidenceではありません。

### fail-closed前提

Evidence toolはまず`reference/workspace`をLoadし、通常Validation Errorがないことを確認します。Validation Error時にはPostgreSQLへ接続しません。

その後だけ`TEI_TEST_POSTGRES_DSN`へ接続し、既存object guardを実行します。

```text
Workspace Load
  ↓
Canonical Validation
  ↓ Errorなら停止
PostgreSQL接続
  ↓
public.order_lines既存確認
  ↓ 既存なら停止
Ephemeral fixture作成
  ↓
Runtime Evidence
```

`TEI_TEST_POSTGRES_DSN`には、Evidence専用の空PostgreSQL Databaseを指定します。

```bash
export TEI_TEST_POSTGRES_DSN='postgres://...'
```

DSNやcredentialをEvidenceファイルへ保存しません。

接続後、toolは次で既存objectを確認します。

```sql
SELECT to_regclass('public.order_lines')::text
```

`public.order_lines`が既に存在する場合はfail closedで停止します。

**既存tableを削除して続行するoverrideはありません。** 別の専用空Databaseを使います。

### 実行

```bash
go run ./tools/reference-postgres-header-detail-runtime \
  | tee /tmp/tei-gate-b-postgres-header-detail.json
```

期待するEvidence:

- `evidenceNature = real-postgres-ephemeral-fixture`
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
4. 次でEvidenceとauthoringを照合する

```bash
go run ./tools/reference-external-query-evidence \
  -workspace reference/workspace \
  -evidence <observed-evidence.json>
```

このtoolもWorkspace Validation ErrorがあればEvidence照合前に停止します。

SQL realizationがない場合はEvidenceから生成せず停止します。`nativeSource.fingerprint`の作り方やprovenanceとの区別は`tools/reference-external-query-evidence/README.md`を正とします。

5. read-onlyを実効強制したOracle connectionでruntime Executorを構成する
6. Workspace → actual read → Semantic resultを一続きに確認する

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

#255でgenerated artifactを修復した場合は、ここでdirty差分を残したままGate B監査へ進まず、意図したartifactをcommitしてclean revisionへ固定し、#255 final check-onlyを再実行します。

## 9. Gate B監査へ戻す

取得したEvidenceを[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)へ対応付けます。

少なくとも次を独立に判定します。

| Evidence lane | 判定対象 |
| --- | --- |
| #310 Repository Evidence runner | UI-less Scenario / external Evidence intake / normal Go runner / source revision不変性 |
| #255 `reference-golden` | World / Plugin LockとCanonical Workspaceの再現性 |
| #255 `reference-durable-run-golden` | Durable Run全体の正規再現性 |
| #308 PostgreSQL tool | real PostgreSQL engine上のfixture runtime |
| #214 Oracle Evidence | external-observed assetの実物理事実とread-only境界 |

#255については、stale修復後のdirty treeではなく、意図した生成物をcommitした**clean revision上のcheck-only一致**を監査Evidenceへ対応付けます。

一つのEvidenceが成功したことを、別種類のEvidenceの代替にしません。

## 停止条件

次の場合はfail closedで止めます。

- Workspace Validation Error
- #310完了Evidenceをdirty working treeから採用しようとしている
- #310 runner実行前後でHEADが変わった、または実行後working treeがdirtyになった
- #255の`-write`直後のdirty treeを完了Evidenceとして採用しようとしている
- #255 final check-only実行前後でHEADが変わった、またはworking treeがdirtyである
- `public.order_lines`が既に存在するPostgreSQL接続先
- #214でprovenanceのない物理情報しかない
- read-only保証をSQL文字列だけで主張している
- generated artifactがstaleなのにfingerprintやJSON断片だけを手編集しようとしている
- fixed observed fixtureを生成結果へ合わせて機械的に変更しようとしている
- EvidenceとSQL realizationが一致しない
- 実行失敗を0件・正常Business Resultへ変換しないと先へ進めない

これらは「作業が止まった」のではなく、Gate Bが意図どおり不足を露出させたEvidenceとして扱います。
