# Gate B Evidence Runbook

## 目的

Gate Bで残っている作業を、設計議論へ戻さず**再現可能なEvidence取得**として実行するための手順です。

何を取得するかは[`GATE_B_EVIDENCE_ACQUISITION.md`](GATE_B_EVIDENCE_ACQUISITION.md)、Gate Bの最終判定は[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md) / GitHub Issue #273を正とします。

2026-09-04時点では、Application Driverの主要意味境界、UI-less最終Scenario構造、external-observed Evidence intake、PostgreSQL fixture安全境界、Invalid Workspaceでのfail-closedまで成立しています。

Gate Bを直接止めているEvidenceは四系統です。

| Lane | Issue | Evidence |
| --- | --- | --- |
| A | #214 | actual external Oracle `master_data` Evidence |
| B | #308 | real PostgreSQL engine + ephemeral fixture Evidence |
| C | #255 | generated Reference artifact再現Evidence |
| D | #310 | deterministic Repository / normal Go runner Evidence |

GitHub Actionsは別の継続的regression経路です。本runbookから再有効化しません。

## Evidenceの種類

### A. actual external asset Evidence

TEIが所有していない既存資産から観測した事実です。

第一Referenceでは#214のOracle `master_data`が該当します。

synthetic dataやPostgreSQL fixtureで代替しません。

### B. real-provider ephemeral fixture Evidence

実provider engineを利用しますが、TEIのtool自身が作る一時fixtureを対象とします。

第一Referenceでは#308 / `tools/reference-postgres-header-detail-runtime`が該当します。

production既存Databaseの物理事実を観測したEvidenceにはしません。

### C. generated artifact Evidence

Canonical authoringから正規生成できるartifactの再現性です。

- `reference/world.json`
- `reference/workspace/.tei/plugin-lock.json`
- 必要なDurable Run

### D. deterministic Repository Evidence

Repository checkoutとGo toolchainだけで再現できるEvidenceです。

- Workspace Validation
- 最終UI-less synthetic Scenario
- external Evidence intake package / tool test
- 可能なら`go test ./...`

## 共通原則

- 未観測の外部事実を名前や慣習から補完しない
- Canonical WorkspaceがInvalidなら後続Evidence取得へ進まない
- 生成物を手編集しない
- fingerprintを人手で置換しない
- runtime credential / DSN / token / secretを通常Evidenceへ保存しない
- 保存不要なruntime parameter値を通常Evidenceへ保存しない
- synthetic Evidenceをreal-provider / actual external Evidenceへ格上げしない
- provider fixture Evidenceをproduction asset Evidenceへ格上げしない
- 実行失敗を0件、NotSatisfied、denied、Business Resultへ変換しない
- Evidence artifactをCanonical meaningの第二の正本にしない
- GitHub Actionsをこの手順から再有効化しない

## 0. 実行revisionと作業状態を固定する

Repository rootで実行します。

```bash
git rev-parse HEAD
git status --short
go version
```

Evidenceには対象revisionを記録します。

生成物差分を監査するため、意図しない未commit差分がある場合は先に識別します。既存差分を機械的に破棄しません。

## 1. Canonical WorkspaceをValidationする

```bash
go run ./cmd/tei validate -workspace reference/workspace
```

期待値:

```text
ok
```

Validation Errorがある場合は、Lane B / C / Dの後続処理やLane AのEvidence intakeへ進みません。生成物やEvidence側を手編集して回避しません。

PR #306により、次のEvidence tool自身もこの前提をfail closedで強制します。

- `tools/reference-external-query-evidence`
- `tools/reference-postgres-header-detail-runtime`

external Evidence toolはEvidence受入前、PostgreSQL toolはDB接続前に通常Validationを実行します。

## 2. Lane D — deterministic Repository Evidenceを取得する

関連: GitHub Issue #310

### 2.1 最終UI-less Scenario

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

同じ`order_entry` / 同じSemantic Order rowから、`customer_id` / `order_id`を物理名やFKから再推測せず後続境界へ引き継げることを確認します。

このtestはsynthetic adapterを含みます。成功してもLane A / BのEvidenceにはしません。

### 2.2 external Evidence intake boundary

```bash
go test ./internal/database/externalqueryevidence \
  ./tools/reference-external-query-evidence \
  -count=1
```

確認対象:

- external EvidenceとCanonical Worldの分離
- target / provider / `external-observed` ownership照合
- Contract input → physical parameter照合
- raw result → Semantic attribute mapping照合
- native source fingerprint stale検知
- provenance必須
- effective read-only mechanism / Evidence参照必須
- EvidenceからSQL realizationを生成・補完しない
- Invalid Workspaceを受け入れない

### 2.3 広いRepository test

実行可能な環境では次も行います。

```bash
go test ./...
```

失敗した場合は、どのpackage / integration / generated artifactに由来するかを分離します。テストを通すためにCanonical meaningを変更しません。

Lane Dの結果を#310へ記録し、Gate B監査#273へ対応付けます。

## 3. Lane C — Reference goldenの現在差分を確認する

関連: GitHub Issue #255

まずRepositoryを書き換えずに確認します。

```bash
go run ./tools/reference-golden
```

一致していれば、そのrevisionのWorld / Plugin Lock / Durable Run依存は最新です。

不一致の場合は、表示されたpathを記録して次へ進みます。

## 4. Lane C — World / Plugin Lockを正規再生成する

意図したCanonical authoring変更から説明できる場合だけ実行します。

```bash
go run ./tools/reference-golden -write
```

更新対象:

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
- 追加済みCapability / Binding / realizationが期待どおり反映されたか
- external assetの未観測物理情報が混入していないか
- 並び順やfingerprintだけを人手で合わせていないか

`reference-golden -write`後にDurable RunのPlugin Lock fingerprint staleが検出された場合、fingerprintだけを手編集しません。

## 5. Lane C — Durable Runを正規`deployment-start`から再生成する

対象:

```text
reference/order_rollout.durable-run.m0.21.start.json
```

保存runのidentity:

```text
solution: order_rollout
version: 2026.8.27
run-id: order-rollout-prod-m0.21
environment: production
```

既存fileをコマンド失敗で空にしないよう、一時fileへ生成して成功後にだけ置換します。

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

失敗時は既存保存runを変更しません。

再生成後:

```bash
git diff -- reference/order_rollout.durable-run.m0.21.start.json

go run ./tools/reference-golden
```

確認観点:

- `pluginLockFingerprint`が現在Plugin Lockと一致する
- `runtimePolicyFingerprint`等の依存値の差分を正規入力から説明できる
- observed fixtureを生成結果へ合わせて手編集していない
- World / Plugin Lock / Durable Runの依存が最終的に一致する

Lane Cの結果を#255へ記録し、#273へ対応付けます。

## 6. Lane B — real PostgreSQL engineでheader-detailを確認する

関連: GitHub Issue #308

### Evidenceの性質

これは**real PostgreSQL engine + tool-owned ephemeral fixture Evidence**です。

production既存Databaseの実データ / schema観測Evidenceではありません。

### 安全前提

`TEI_TEST_POSTGRES_DSN`にはEvidence専用の**空PostgreSQL Database**を指定します。

```bash
export TEI_TEST_POSTGRES_DSN='postgres://...'
```

DSNやcredentialをEvidenceファイル、Issue、PRへ保存しません。

PR #294 / #300 / #306以降、toolは次の順序を強制します。

```text
Workspace Load
 -> normal Validation
 -> Validation Errorなし
 -> PostgreSQL接続
 -> existing public.order_lines guard
 -> tool-owned fixture create
 -> runtime Evidence
 -> cleanup
```

`public.order_lines`が既に存在する場合はfail closedで停止します。

**既存tableを削除して続行するoverrideはありません。** 別の専用空Databaseを使用します。

### 実行

```bash
go run ./tools/reference-postgres-header-detail-runtime \
  | tee /tmp/tei-gate-b-postgres-header-detail.json
```

期待するEvidence:

- `evidenceNature = real-postgres-ephemeral-fixture`
- fixture object = `public.order_lines`
- preexisting fixtureを保護したこと
- fixtureがtool所有であること
- Pattern = `order_entry.order_lines`
- Relation = `sales.order.lines`
- Environment = `development`
- Database instance = `sales_core.dev`
- executor Capability = `database.query.postgres`
- Contract Query target identity
- parameter names = `order_id`
- parameter values persisted = false
- rows = 2
- zeroRows = 0
- Execution Error observed = true
- `sales.order_line.order_id`がSemantic attributeへ追加されていない
- Relation Storage column = `order_id`
- Semantic mapping applied = true

出力はまずEvidence artifactとして確認します。Repository標準の保存pathをこのrunbookから先行固定しません。

Lane Bの結果を#308へ記録し、#273へ対応付けます。

## 7. Lane A — #214 actual Oracle Evidenceを受け取る

関連: GitHub Issue #214

この工程は実外部事実が得られるまで**Blockedのまま保持して構いません**。

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
- read-only principal / DB permission / session等の実効的根拠
- 採用Oracle driverのnamed parameter / result / NULL / error伝播

未観測の`CUSTOMER`、`CUSTOMER_ID`等を仮定しません。

### Evidence取得後

1. 観測事実をexternal Evidence JSONへ記録する
2. 観測事実を根拠に`ContractQuerySQLRealization`を明示authoringする
3. native sourceを通常Workspaceから追跡可能にする
4. Canonical Workspaceを再Validationする
5. Evidenceとauthoringを照合する

```bash
go run ./tools/reference-external-query-evidence \
  -workspace reference/workspace \
  -evidence <observed-evidence.json>
```

SQL realizationがない場合、このtoolはEvidenceから生成せず停止します。

`nativeSource.fingerprint`はraw SQL file bytesやGit blob SHAではなく、Workspace loaderがtrim後に保持する`ContractQuerySQLRealization.Source`へSource Index形式のSHA-256を適用したものです。provenanceやread-only Evidenceとは別に扱います。

6. read-onlyを実効強制したOracle connectionでruntime Executorを構成する
7. Workspace → actual read → Semantic resultを一続きに確認する
8. 1件 / 0件 / Execution Errorを区別する

Lane Aの結果を#214へ記録し、#273へ対応付けます。

## 8. 最終差分を確認する

```bash
git status --short

git diff -- \
  reference/world.json \
  reference/workspace/.tei/plugin-lock.json \
  reference/order_rollout.durable-run.m0.21.start.json
```

commit対象は、正規生成されたartifactと、観測事実またはCanonical meaningから説明可能なsource変更だけにします。

次を誤ってcommitしません。

- DSN / credential / token / secret
- runtime parameter値
- 一時Evidence file
- 推測したexternal物理情報

## 9. Gate B監査へ戻す

四laneのEvidenceを[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md) / #273へ対応付けます。

| Evidence | 判定対象 |
| --- | --- |
| #310 | UI-less Scenario / external Evidence intakeのdeterministic runner Evidence |
| #255 | authoritative sourceからのgenerated artifact再現性 |
| #308 | real PostgreSQL engine上のephemeral fixture runtime |
| #214 | actual external Oracle物理事実 / driver / read-only境界 |

一つのEvidenceが成功したことを、別laneの代替にしません。

四laneが揃ったらnegative caseを再監査し、Gate BをReady / Not Readyで再判定します。

Readyの場合のみ、[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md) / #291に従って、そのrevisionを第一Handover Baselineとして固定します。

その後に#70第一Web Targetへ進みます。

## 停止条件

次の場合はfail closedで止めます。

- Workspace Validation Error
- `public.order_lines`が既に存在するPostgreSQL接続先
- #214でprovenanceのない物理情報しかない
- read-only保証をSQL文字列だけで主張している
- `reference-golden`がDurable Run staleを検出したのにfingerprintを手編集しようとしている
- EvidenceとSQL realizationが一致しない
- 実行失敗を0件・正常Business Resultへ変換しないと先へ進めない

これらは「作業が失敗したから隠すべき状態」ではなく、Gate Bが不足や危険な前提を正しく露出させたEvidenceとして扱います。
