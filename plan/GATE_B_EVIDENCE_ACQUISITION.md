# Gate B Evidence Acquisition

## 目的

この文書は、Gate Bの残件を「設計をさらに増やす作業」ではなく、既に成立した境界へ不足しているEvidenceを取得する作業として実行できる形へ整理します。

Gate Bの正式な判定は[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)を正とします。この文書はその判定条件を変更せず、残件ごとの入力、実行経路、保存してよいEvidence、完了判定を具体化するrunbookです。

## 現在地

PR #285でApplication Driver / Reference data access / header-detail / Observation / 最終UI-less Scenarioの実装・Reference authoring・Conformanceを`main`へ収束しました。

したがって、現在のGate B残件は主に次の四系統です。

1. external-observed `master_data`の実物理 / read-only Evidence
2. header-detailの実PostgreSQL Evidence
3. Reference World / Plugin Lock / Durable Runの正規再生成Evidence
4. 最終UI-less Scenarioを通常Go runnerで実行したEvidence

GitHub Actionsによる再検証は重要ですが、上記Evidenceと同一ではありません。Actionsが停止していること自体をGate Bの意味設計Gapへ読み替えず、再開後の継続的な再検証経路として別に追跡します。

## Evidence A — external-observed `master_data`

関連: `ISSUE-000022` / GitHub Issue #214

### 取得前に必要な実物理情報

TEI側で名前を推測せず、利用対象から少なくとも次を取得します。

- 実際の物理object identity
  - table / view / API等
- `customer/find_by_id`へ渡す物理parameter identity
- raw resultとして返る列identity
- `customer.customer.customer_id` / `customer.customer.name`への明示mapping
- native SQLまたはAPI realization source
- その情報をどこから確認したかというprovenance
- read-onlyを実効的に保証する境界

### read-onlyのEvidence

`ownership external-observed`という宣言だけでread-onlyを証明したことにはしません。

利用する環境に応じて、少なくとも次のうち実際に効いている境界を記録します。

- read-only credential / database role
- session / transaction read-only設定
- read-only view / API
- write権限を持たない接続先
- 外部側の権限制御

SQL文字列が`SELECT`に見えることだけをread-only Evidenceにはしません。

### 保存しないもの

通常Evidenceへ次を保存しません。

- password
- DSNに含まれるcredential
- token / secret
- 実行時parameter valueで保存不要なもの
- 業務上不要な実データ

### 完了判定

少なくとも次を実接続で区別できたら、このEvidence laneを完了候補とします。

- 1件取得
- 0件を正常なnot-foundとして保持
- 実行失敗をExecution Errorとして保持
- 明示mappingからSemantic Customerへ変換
- read-only境界が実効的であることを説明可能
- Environment / Database instance / executor Capabilityのidentityを保持

未観測の物理名をTEI側から補った場合はEvidenceとして採用しません。

## Evidence B — header-detail実PostgreSQL

関連: `ISSUE-000024` / GitHub Issue #244

実行入口は`tools/reference-postgres-header-detail-runtime`です。

### 実行環境

このtoolはfixtureとして`public.order_lines`を**作成し、Evidence取得途中で削除します**。既存の業務Databaseや共有Schemaでは実行せず、破棄可能な専用PostgreSQL databaseを使用します。

接続情報はRepositoryへ保存せず、環境変数から渡します。

```bash
export TEI_TEST_POSTGRES_DSN='postgres://...'
go run ./tools/reference-postgres-header-detail-runtime
```

### toolが確認すること

- `order_id=1001`で2件取得
- `order_id=9999`で0件を正常結果として取得
- table削除後のreadをExecution Errorとして区別
- physical parameter名は`order_id`
- parameter valueをEvidenceへ保存しない
- raw resultを明示mappingからSemantic OrderLineへ変換
- Relation Storageの`order_id` support columnをSemantic attributeへ逆流させない
- Environment = `development`
- Database instance = `sales_core.dev`
- executor Capability = `database.query.postgres`

正常終了時にJSON Evidenceを標準出力へ返します。

### 完了判定

- 実PostgreSQLへ接続してtoolが正常終了する
- JSON Evidenceを再確認できる
- credential / DSNをEvidenceへ含めない
- 2件 / 0件 / Execution Errorの三状態を区別できる
- physical support columnをSemantic attributeへ格上げしていない

## Evidence C — Reference golden再生成

関連: `ISSUE-000025` / GitHub Issue #255

### 1. 現在差分の検査

まずRepositoryを変更せず、現在のCanonical Workspaceと保存済みgoldenの一致を確認します。

```bash
go run ./tools/reference-golden
```

一致しない場合、表示されたpathをEvidenceとして記録します。

### 2. 意図したauthoring変更だけを反映

Canonical authoringの変更から説明できる差分であることを確認した場合だけ更新します。

```bash
go run ./tools/reference-golden -write

git diff -- \
  reference/world.json \
  reference/workspace/.tei/plugin-lock.json
```

生成物を手編集して合わせません。

### 3. Durable Runがstaleになった場合

Plugin Lock fingerprintが変わった場合は`pluginLockFingerprint`だけを書き換えず、正規の`deployment-start`経路から再生成します。

```bash
go run ./cmd/tei deployment-start \
  -workspace reference/workspace \
  -solution order_rollout \
  -version 2026.8.27 \
  -deployment reference/observed/production.deployment-state.json \
  -run-id order-rollout-prod-m0.21 \
  -plugin-lock reference/workspace/.tei/plugin-lock.json \
  > /tmp/tei-reference-deployment-run.json

diff -u \
  reference/order_rollout.durable-run.m0.21.start.json \
  /tmp/tei-reference-deployment-run.json
```

差分が現在のWorkspace / Plugin Lock / fixed observed fixtureから説明できる場合だけ、出力全体をgoldenとして更新します。

### 4. 最終確認

```bash
go run ./tools/reference-golden
```

World / Plugin Lock / Durable Runの依存がすべて一致することを確認します。

### 完了判定

- 最新authoringからWorldを再現できる
- 最新authoringからPlugin Lockを再現できる
- Durable RunのPlugin Lock依存が一致する
- 更新差分を意味上のauthoring変更から説明できる
- 手編集によるfingerprint合わせをしていない

## Evidence D — 最終UI-less Scenario

正本は`internal/driver/final_gate_b_scenario_test.go`です。

### 最小実行

```bash
go test ./internal/driver \
  -run '^TestOrderEntryGateBScenarioConnectsMajorBoundariesWithoutUI$' \
  -count=1
```

このテストは次を一つのtest-local Scenarioへ接続します。

```text
Authentication
  -> browse
  -> record Pattern resolution
  -> reference Customer
  -> header-detail OrderLine[]
  -> Canonical Validation
  -> Authorization
  -> Command Business Result
  -> separate Observation read
```

synthetic adapterを含むため、成功しても実Oracle / 実PostgreSQL / 実外部Command Evidenceへ格上げしません。

### Repository全体の通常runner確認

実行可能な環境では、個別Scenarioだけでなく通常のGo検証も行います。

```bash
go test ./...
```

GitHub Actionsが停止中でも、独立した通常Go runnerで得た結果はGate BのScenario実行Evidenceとして利用できます。逆に、実行していない状態をgreenとは記録しません。

### 完了判定

- 最終Scenario testを通常Go runnerで実行できる
- testが成功する
- 使用したrepository revisionを記録できる
- synthetic Evidenceであることを明示する
- 実外部Evidenceとの区別を維持する

## Gate BとCIの分離

Gate Bは、特定のCIサービスがgreenかどうかではなく、Product Value上必要な意味境界とEvidenceが揃ったかで判定します。

したがって次を分けます。

### Gate B判定に直接必要

- external-observed assetを推測していない実物理Evidence
- 実PostgreSQL header-detail Evidence
- Reference artifactの再生成性
- UI-less Scenarioの通常runner実行Evidence
- Business Result / Execution Error / Observation等の意味境界
- read-only / secret等の安全境界

### CI再開後に継続確認するもの

- `go test ./...`の継続的再実行
- Reference golden再現性
- PostgreSQL integrationの継続的再実行
- 将来のregression検知

CIをGate Bの代替Evidenceにせず、逆にCI停止だけを理由にProduct Gateを永久に止めないようにします。

## Evidence取得後の更新順

Evidenceを得たら、その都度次を更新します。

1. 対応するGitHub Issue / Repository Issueの現在地
2. `GATE_B_READINESS_AUDIT.md`の条件表
3. synthetic / external-observed / managed-hereのEvidence性質
4. unresolved事項

四系統のEvidenceが揃った時点でGate B監査を再実行します。

Gate Bを通過した場合に初めて、第一Web Targetへ「Web側で意味を補わないbaseline」として同じApplication / Driver境界を引き渡します。
