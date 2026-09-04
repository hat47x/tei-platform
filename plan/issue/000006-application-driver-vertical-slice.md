# ISSUE-000006: Application Driverの最小vertical sliceを実装する

- 状態: In Progress
- GitHub Issue: #69
- 前段: `ISSUE-000013` / GitHub Issue #124（Gate A Ready）
- 次のGate: Gate B「UIなしで主要Scenarioを再現できる」
- Gate B監査: `../GATE_B_READINESS_AUDIT.md` / `ISSUE-000029` / GitHub Issue #273

## 目的

Application Driverはproduction Runtimeの簡易版ではありません。

設計途中のApplicationをUIなしで実際に動かし、Application上の意味、Database / Provider実現、Binding、安全境界の不足や混同を、第一Web Targetへ進む前に露出させるための設計検証環境です。

```text
Application定義
       ↓
Pattern / Datasource / Bindingの解決
       ↓
Driver Scenario
       ↓
Authentication / Browse / Record / Reference / Header-detail
       ↓
Validation / Authorization / Command
       ↓
Business Result / Execution Error
       ↓ separate explicit read
Observation / Evidence
```

Driverは不足する意味や外部事実を独自規則で補いません。Gapが見つかった場合は、その意味や事実を所有すべきApplication / Semantic / Database Contract / Realization / Binding / external Evidenceへ戻します。

## 現在地

2026-09-04時点で、Driverの中心設計と主要runtime境界は大きく成立しています。

PR #281で`internal/driver/final_gate_b_scenario_test.go`を追加し、同じ`order_entry` Applicationと同じSemantic Order rowを使って、主要境界を一つのtest-local UI-less Scenarioへ統合しました。

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

browseで得た`sales.order.customer_id` / `sales.order.order_id`を別名や物理FKから再推測せず、reference / header-detail / command / observationへ明示的に引き継ぎます。header-detailで得たOrderLine値はValidation inputへ接続します。

ただし、**Gate BはNot Ready**です。

残っているのは主に次のEvidenceです。

1. external-observed `master_data`の実物理Evidence
2. header-detailの実PostgreSQL成功Evidence
3. 最新Canonical WorkspaceからのReference World / Plugin Lock / Durable Run再現
4. PR #281 UI-less Scenarioの通常Go runner実行Evidence
5. CI再開後の通常runner再検証

PR #281はDatabase read / Command side effect / Observation readにsynthetic adapterを含むため、構造統合Evidenceと実外部Evidenceを混同しません。

最新判定は[`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md)を正とします。

## 関連する実装・設計Issue

- native Query runtime: `ISSUE-000016` / #167
- Relation traversal key: `ISSUE-000017` / #178
- Database Contract Query: `ISSUE-000018` / #179
- Reference invocation: `ISSUE-000019` / #185
- Contract Query runtime instance identity: `ISSUE-000020` / #222
- Oracle Contract Query Runner: `ISSUE-000021` / #224
- external `master_data`物理Evidence: `ISSUE-000022` / #214
- header-detail traversal: `ISSUE-000023` / #233 / ADR-0021
- keyless Relation physical storage: `ISSUE-000024` / #244
- Reference生成物再生成: `ISSUE-000025` / #255
- Command / Observation: GitHub Issue #266 / ADR-0023
- Gate B readiness audit: `ISSUE-000029` / #273

## 成立しているvertical slice

### Application Pattern / Canonical resolution

`internal/driver`を正式なDriver入口として、Application IDとPattern IDから次を解決できます。

- `browse`: Semantic Data / Application Query / logical Datasource
- `record`: Semantic Data
- `reference`: Relation / source・target Data / logical Datasource / Contract Query invocation
- `header-detail`: `many` Relation / source・target Data / traversal realization
- `command`: Semantic Capability

5種類は第一Referenceで成立した候補であり、TEI全体の永久的な標準一覧とはみなしません。

recordは現行契約どおりSemantic Data解決までです。PR #281でも、未定義のedit runtimeやPersistence動作をtest都合で補いません。

### Capability Binding / Authentication / Authorization

Application + Capability identityからBindingを一意に解決し、既存Invocation境界から実行できます。

```text
order_entry / security.authenticate
order_entry / security.authorize
order_entry / sales.order.submit
```

Authentication / Authorization / Commandで別々のDriver専用実行体系を作りません。

legacy organization / role / return codeはAdapter側へ残し、Driver Coreへ逆流させません。

### browse Query / PostgreSQL

`order_entry.recent_orders`ではlogical Datasource `sales`からEnvironment上のDatabaseへ接続し、native QueryをProvider側Executorへ渡せます。

raw resultとSemantic resultは分け、raw column labelからSemantic attributeへの対応はQuery authoringで明示します。

次から推測しません。

- raw列名の類似
- SELECT列順
- Persistence Mapping
- provider固有規則

### `Order -> Customer` reference

provider非依存の意味・runtime境界は成立しています。

```text
sales.order.customer
  source key sales.order.customer_id
  target key customer.customer.customer_id
        ↓
Reference Contract Query invocation
        ↓
master_data.customer / find_by_id
        ↓
Environment / Database instance / executor Capability
        ↓
Oracle SQL Runner
        ↓
raw result -> explicit Semantic mapping
```

PR #281でもbrowseでSemantic mapping済みの同じOrder rowからCustomer Relation keyを取り出し、`master_data.dev` / `database.query.oracle`の明示境界へ渡します。

ただし`master_data`は`external-observed`です。実際のtable / view / parameter / raw column / native SQL / provenanceをRepositoryからは観測できていません。

未観測のOracle物理情報を推測して埋めません。この実外部Evidenceは`ISSUE-000022` / #214で扱います。

### keyless `Order -> OrderLine[]` header-detail

`sales.order.lines`をkeylessなSemantic Relationのまま維持しています。

取得方法はADR-0021に従い、Pattern外のApplication固有realizationへ分離します。

```text
sales.order.order_id
        ↓ explicit binding
sales_core.order_line / find_by_order(order_id)
```

物理保存はDatabase-ownedなRelation Storageとして表します。`order_id`を`sales.order_line`のSemantic attributeへ追加しません。

次まで成立しています。

- HeaderDetailContractQueryBinding
- Relation Storage model / Validation / authoring / Source Index
- Desired Physical Schema / migration
- `find_by_order` PostgreSQL SQL realization
- PostgreSQL Environment composition
- instance-aware runtime execution
- 0件 / 複数件 / Execution Errorの区別

PR #281でもbrowseの同じ`sales.order.order_id`をheader-detail Query inputへ渡します。

`tools/reference-postgres-header-detail-runtime`はありますが、実PostgreSQLへの成功Evidenceはまだ取得していません。

### Validation -> Authorization -> Command

PR #265で次を区別しています。

- Validation `Satisfied` / `NotSatisfied`
- Validation評価不能のExecution Error
- Authorization `allowed` / `denied`
- Command Business Result / Execution Error

PR #281では、header-detail resultからOrderLine quantity / line numberをValidation sourceへ渡し、Authenticationで得たsubjectをAuthorizationへ、browseで選択した同じ`order_id`をCommandへ引き継ぎます。

Command Business Resultだけを「外部状態変更を確認済み」とは扱いません。

## Observation

ADR-0023に従い、ObservationはCommand結果から派生させず、**明示されたread境界を別操作として実行した結果**として扱います。

```text
Command invocation
        ↓
Business Result / acknowledgement

--- separate operation ---

explicit read boundary
        ↓
read result / Execution Error
        ↓
Observation Evidence
```

production Driverには`ExecuteObservationRead`の最小runtime境界を追加済みです。read identity / Evidence nature / resultだけを保持し、runtime parameter値やcredentialを通常Evidenceへ保存しません。

PR #281でもCommandとは別のsynthetic readとして、同じ`order_id`の状態をObservationします。

synthetic Evidenceは実外部Observed stateとは区別します。

## Reference生成物

Reference data access / Relation Storage / Plugin Capability等のauthoring更新により、保存済みReference artifactの再生成確認が必要です。

#255では依存監査まで進み、次を区別しています。

- `reference/world.json`: Canonical Workspaceからの生成golden
- `reference/workspace/.tei/plugin-lock.json`: Canonical Workspaceからの生成golden
- `reference/order_rollout.durable-run.m0.21.start.json`: Workspace + Plugin Lock + observed fixtureから`tei deployment-start`で生成するgolden
- `reference/observed/*`: fixed external-state fixture

PR #280では`reference-golden`へDurable RunのPlugin Lock依存stale検知を追加しました。Durable Run自体の生成責務は`tei deployment-start`へ残します。

実再生成はGo commandを実行できる環境で行う必要があります。

## 最終UI-less Scenario

構造統合はPR #281で実装済みです。

正本は`internal/driver/final_gate_b_scenario_test.go`です。

このScenarioは新しいproduction Workflow / Scenario engine / Evidence schemaを追加せず、既存Driver APIをtest-localに接続します。

Database read / Command side effect / Observation readはsyntheticなので、次の代替にはしません。

- #214 external `master_data`実read / read-only Evidence
- header-detail実PostgreSQL Evidence
- #255 Reference artifact再生成
- 実外部Command / Observation Evidence
- CI / normal runner Evidence

本チャット環境からRepositoryをcheckoutしてGo testを実行しようとしましたが、GitHub hostを名前解決できませんでした。GitHub Actionsも停止中なので、PR #281のmergeをtest greenとは扱いません。

## Gate Bを止めている残件

### Blocked — external `master_data`

#214でphysical object / parameter / raw result columns / native realization / Semantic mapping / provenance / effective read-only Evidenceが必要です。

### Pending — header-detail実PostgreSQL Evidence

`tools/reference-postgres-header-detail-runtime`を実PostgreSQLへ接続し、複数件 / 0件 / Execution Errorを確認します。

### Pending — Reference artifact regeneration

#255でWorld / Plugin Lock / Durable Runを正規再生成します。依存監査とstale検知は済んでいます。

### Pending — PR #281 runner Evidence

UI-less Scenario構造は統合済みですが、通常Go runnerでの成功確認は未実施です。

### Pending — normal CI verification

GitHub ActionsはRepositoryから明示的に停止されています。このIssueから独断で再有効化しません。

## Gate B Ready条件

Gate Bはコード量やIssue close数ではなく、Reference Scenarioから次を説明できるときReadyとします。

- 同じApplication定義から主要ScenarioをUIなしで一続きに再現できる
- 同じ業務identityを重要境界へ明示的に継承できる
- Pattern / Query / Relation / Contract Query / Datasource / Capability / Bindingを追跡できる
- native SQLやprovider固有処理をDriver Coreへ複製していない
- raw resultとSemantic resultの対応が明示されている
- Relation traversalを物理FKや名前から推測していない
- external assetの未観測事実をTEIが補完していない
- `master_data`へのwriteを実効的な境界で防げる
- Validation不成立時にCommandへ進まない
- Authorization denied時にCommandへ進まない
- Business Result / Execution Error / Observationを区別できる
- Query 0件とExecution Errorを区別できる
- secret / runtime parameter値を通常Evidenceへ残さない
- Driver固有状態をCanonical Worldへ逆流させない
- 最新authoringからReference artifactを再現できる
- 第一Web TargetへWeb固有の意味補完なしに同じbaselineを渡せる

## 次の作業

1. #214でexternal `master_data`の物理Evidenceを得る
2. header-detail実PostgreSQL Evidenceを取得する
3. #255でReference World / Plugin Lock / Durable Runを正規再生成する
4. PR #281のUI-less ScenarioをGo runnerで実行する
5. external / runtime / golden EvidenceをGate B監査へ対応付ける
6. negative caseを再監査する
7. CIが別判断で再開された後、通常runnerで関連Conformanceを再検証する
8. Gate BをReady / Not Readyで再判定する

## 非目標

- Gate B前の第一Web Target実装
- Web framework選定
- 汎用Workflow Engine / Event Sourcing基盤
- Observation専用Application Patternの先行導入
- Editor / AI実装
- 第二Target
- production HA / Connection Pool一般化
- physical FKの自動生成

## 関連

- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- ADR-0020 Capability Business Result / Execution Error
- ADR-0021 header-detail traversal realization
- ADR-0023 Command / Observation explicit read boundary
- GitHub #69 / #70 / #214 / #244 / #255 / #266 / #270 / #273 / PR #281
