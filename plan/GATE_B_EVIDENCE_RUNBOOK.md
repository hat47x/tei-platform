# Gate B Evidence Runbook

## 目的

Gate Bで残っている作業を、設計議論へ戻さず**再現可能なEvidence取得**として実行するための正規手順です。

Evidenceの必要性・証明範囲・完了条件は[`GATE_B_EVIDENCE_ACQUISITION.md`](GATE_B_EVIDENCE_ACQUISITION.md)、Gate BのReady / Not Ready判定は[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)を正とします。

**具体的なコマンド、実行順、差分確認、fail-closedの停止条件については本書を唯一の正本とします。** 他文書に同じコマンドが残っている場合、本書と矛盾する手順を採用しません。

2026-09-04時点では、Application Driverの主要意味境界、UI-less最終Scenario構造、external-observed Query Evidence受入境界、PostgreSQL fixtureの安全境界まで成立しています。

Gate Bを止めているのは主に次です。

1. #214 `master_data`の実Oracle物理Evidence / read-only実証
2. header-detailの実PostgreSQL engine Evidence
3. #255 Reference World / Plugin Lock / Durable Runの正規再生成
4. PR #281 / #288 / #304等の通常Go runner実行Evidence
5. CIが別判断で再開された場合の通常runner再検証

## Evidenceの種類

### A. deterministic repository Evidence

Repository checkoutとGo toolchainだけで再現できるものです。

- Workspace Validation
- PR #281 UI-less synthetic Scenario
- PR #288 / #304 external Evidence intake tests
- Reference World / Plugin Lock再生成
- Durable Run再生成

### B. real-provider fixture Evidence

実provider engineを使いますが、toolが作るephemeral fixtureに対するEvidenceです。

- `tools/reference-postgres-header-detail-runtime`

これはproduction既存資産の物理事実を観測するEvidenceではありません。

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
- GitHub Actionsをこの手順から再有効化しない

## 0. 実行前確認

Repository rootで作業します。

```bash
git status --short
```

生成物差分を監査するため、意図しない未commit差分がある場合は先に識別します。既存差分を機械的に破棄しません。

Go toolchainを確認します。

```bash
go version
```

実行revisionも記録できるよう確認します。

```bash
git rev-parse HEAD
```

## 1. Canonical WorkspaceをValidationする

```bash
go run ./cmd/tei validate -workspace reference/workspace
```

期待値:

```text
ok
```

Validation Errorがある場合は、その後のgolden再生成やprovider / external Evidence実行へ進みません。生成物側を手編集して回避しません。

PR #304以降、external Evidence toolとPostgreSQL Evidence tool自身も通常`validation.Validate`のErrorでfail closedします。ただし、それを理由にこの最初の明示Validationを省略しません。最初に前提不整合を分離するためです。

## 2. Repository内だけで成立するGate B Scenarioを実行する

### PR #281 UI-less最終Scenario

```bash
go test ./internal/driver \
  -run '^TestOrderEntryGateBScenarioConnectsMajorBoundariesWithoutUI$' \
  -count=1
```

ここで確認するのは、同じ`order_entry` / 同じSemantic Order rowから次を一続きに接続できることです。

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

このtestはsynthetic adapterを含みます。成功しても実Oracle / 実PostgreSQL / 実外部状態のEvidenceにはしません。

### PR #288 / #304 external Evidence intake

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

Evidence formatとfingerprintの意味は`tools/reference-external-query-evidence/README.md`も参照します。

## 3. Reference goldenの現在差分を確認する

まずRepositoryを書き換えずに確認します。

```bash
go run ./tools/reference-golden
```

一致していれば、その時点のWorld / Plugin Lock / Durable Run依存は最新です。

不一致の場合は、表示されたpathを記録して次へ進みます。

## 4. World / Plugin Lockを正規再生成する

```bash
go run ./tools/reference-golden -write
```

このtoolが更新するのは、Canonical Workspaceから導出される次のgoldenです。

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
- `database.query.postgres`等、追加済みCapabilityが期待どおり反映されたか
- 不明なexternal physical情報が混入していないか
- 並び順やfingerprintだけを人手で合わせていないか

`reference-golden -write`後、Durable RunのPlugin Lock fingerprintが古い場合はtoolが明示的に停止します。そのfingerprintを手編集しません。

## 5. Durable Runを正規`deployment-start`で再生成する

対象:

```text
reference/order_rollout.durable-run.m0.21.start.json
```

既存保存runのidentityは次です。

```text
solution: order_rollout
version: 2026.8.27
run-id: order-rollout-prod-m0.21
environment: production
```

一時ファイルへ生成し、コマンド成功後にだけ置換します。

```bash
tmp="$(mktemp)"

go run ./cmd/tei deployment-start \
  -workspace reference/workspace \
  -solution order_rollout \
  -version 2026.8.27 \
  -deployment reference/observed/production.deployment-state.json \
  -run-id order-rollout-prod-m0.21 \
  -plugin-lock reference/workspace/.tei/plugin-lock.json \
  > "$tmp" && \
  mv "$tmp" reference/order_rollout.durable-run.m0.21.start.json
```

失敗時に既存保存runを空ファイルで上書きしないため、直接保存pathへredirectしません。

再生成後:

```bash
git diff -- reference/order_rollout.durable-run.m0.21.start.json
```

確認観点:

- `pluginLockFingerprint`が現在Plugin Lockと一致する
- `runtimePolicyFingerprint`等、正規生成された依存値の差分を説明できる
- observed stateを手編集して生成結果へ合わせていない

最後に再確認します。

```bash
go run ./tools/reference-golden
```

## 6. 実PostgreSQL engineでheader-detailを確認する

### Evidenceの性質

これは**real PostgreSQL engine + ephemeral fixture runtime Evidence**です。

production既存Databaseの実データEvidenceではありません。

### fail-closed前提

PR #304以降、toolはまず`reference/workspace`をLoadし、通常Validation Errorがないことを確認します。**Validation Error時にはPostgreSQLへ接続しません。**

その後だけ`TEI_TEST_POSTGRES_DSN`へ接続し、PR #294の既存object guardを実行します。

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

PR #304以降、このtoolもWorkspace Validation ErrorがあればEvidence照合前に停止します。

SQL realizationがない場合はEvidenceから生成せず停止します。`nativeSource.fingerprint`の作り方やprovenanceとの区別は`tools/reference-external-query-evidence/README.md`を正とします。

5. read-onlyを実効強制したOracle connectionでruntime Executorを構成する
6. Workspace → actual read → Semantic resultを一続きに確認する

## 8. 広いGo testを実行する

個別Evidenceが成立した後、可能な環境ではRepository全体を確認します。

```bash
go test ./...
```

ここで失敗した場合も、生成物・Evidence・test期待値のどこが古いのかを分離して調べます。テストを通すためにCanonical意味を変更しません。

## 9. 最終差分を確認する

```bash
git status --short

git diff -- \
  reference/world.json \
  reference/workspace/.tei/plugin-lock.json \
  reference/order_rollout.durable-run.m0.21.start.json
```

commit対象は、正規生成されたartifactと、意味上説明可能なsource変更だけにします。

DSN、credential、runtime parameter値、一時Evidence fileを誤ってcommitしません。

## 10. Gate B監査へ戻す

取得したEvidenceを[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)へ対応付けます。

少なくとも次を独立に判定します。

| Evidence | 判定対象 |
| --- | --- |
| PR #281 Go test | UI-less主要Scenarioのsynthetic構造 |
| PR #288 / #304 Go test | external Evidence intake / fail-closed Validation boundary |
| World / Plugin Lock / Durable Run | authoritative sourceからのartifact再現性 |
| PostgreSQL tool | 実PostgreSQL engine上のfixture runtime |
| #214 Oracle Evidence | external-observed assetの実物理事実とread-only境界 |
| `go test ./...` | 通常Go runnerでの横断回帰 |

一つのEvidenceが成功したことを、別種類のEvidenceの代替にしません。

## 停止条件

次の場合はfail closedで止めます。

- Workspace Validation Error
- `public.order_lines`が既に存在するPostgreSQL接続先
- #214でprovenanceのない物理情報しかない
- read-only保証をSQL文字列だけで主張している
- `reference-golden`がDurable Run staleを検出したのにfingerprintを手編集しようとしている
- EvidenceとSQL realizationが一致しない
- 実行失敗を0件・正常Business Resultへ変換しないと先へ進めない

これらは「作業が止まった」のではなく、Gate Bが意図どおり不足を露出させたEvidenceとして扱います。
