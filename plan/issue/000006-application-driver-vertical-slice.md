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

さらに、`internal/driver/gate_b_ui_less_scenario_test.go`で、browse -> record -> reference -> header-detail -> Validation -> Authorization -> Command -> separate Observation readを同じReference World上で接続するtest-localな複合Scenarioを追加しました。

ただし、**Gate BはNot Ready**です。

残っているのは、初期段階のようなPattern解決やQuery実行APIの不足ではなく、主に次のEvidenceです。

1. external-observed `master_data`の実物理Evidence
2. header-detailの実PostgreSQL成功Evidence
3. 最新Canonical WorkspaceからのReference World / Plugin Lock / Durable Run再現
4. 新しいUI-less複合ScenarioのGo runner実行Evidence
5. CI再開後の通常runner再検証

Command acknowledgement / Observationの設計境界は、ADR-0023、production `ExecuteObservationRead`、synthetic Scenarioで成立済みです。最終UI-less Scenarioについても構造統合は実装済みですが、synthetic adapterを含むため外部Evidenceの代替にはしません。

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

### 1. Application Pattern / Canonical resolution

`internal/driver`を正式なDriver入口として、Application IDとPattern IDから次を解決できます。

- `browse`: Semantic Data / Application Query / logical Datasource
- `record`: Semantic Data
- `reference`: Relation / source・target Data / logical Datasource / Contract Query invocation
- `header-detail`: `many` Relation / source・target Data / traversal realization
- `command`: Semantic Capability

5種類は第一Referenceで成立した候補であり、TEI全体の永久的な標準一覧とはみなしません。

### 2. Capability Binding / Authentication / Authorization

Application + Capability identityからBindingを一意に解決し、既存Invocation境界から実行できます。

```text
order_entry / security.authenticate
order_entry / security.authorize
order_entry / sales.order.submit
```

Authentication / Authorization / Commandで別々のDriver専用実行体系を作りません。

legacy organization / role / return codeはAdapter側へ残し、Driver Coreへ逆流させません。

Business ResultとExecution Errorも分離します。

### 3. browse Query / PostgreSQL

`order_entry.recent_orders`では、logical Datasource `sales`からEnvironment上のDatabaseへ接続し、native QueryをProvider側Executorへ渡せます。

named parameterのPostgreSQL placeholder変換もProvider側責務です。

raw resultとSemantic resultは分け、raw column labelからSemantic attributeへの対応はQuery authoringで明示します。

```text
total_amount -> sales.order.total
```

次から推測しません。

- raw列名の類似
- SELECT列順
- Persistence Mapping
- provider固有規則

### 4. `Order -> Customer` reference

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
instance-aware runtime Registry
        ↓
Oracle SQL Runner
        ↓
raw result -> explicit Semantic mapping
```

Contract Query input / result / cardinality、SQL realization model、Environment composition、Oracle向け最小`database/sql` Runnerまで実装済みです。

UI-less複合Scenarioでは、browseでSemantic mapping済みのOrder rowをそのまま入力にし、Customer Relation keyから`master_data.dev` / `database.query.oracle`の明示Bindingへ接続します。物理列名やFKから補完しません。

ただし`master_data`は`external-observed`です。実際のtable / view / parameter / raw column / native SQL / provenanceをRepositoryからは観測できていません。

未観測の`CUSTOMER`表や`CUSTOMER_ID`列等を推測して埋めません。

この実外部Evidenceは`ISSUE-000022` / #214で扱います。

### 5. keyless `Order -> OrderLine[]` header-detail

`sales.order.lines`をkeylessなSemantic Relationのまま維持しています。

Application PatternもRelation-onlyです。

```tei
pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}
```

取得方法はADR-0021に従い、Pattern外のApplication固有realizationへ分離します。

```text
sales.order.order_id
        ↓ explicit binding
sales_core.order_line / find_by_order(order_id)
```

物理保存はDatabase-ownedなRelation Storageとして表します。

```text
sales.order.lines
  source sales.order.order_id
        ↓
public.order_lines.order_id
```

`order_id`を`sales.order_line`のSemantic attributeへ追加しません。

次まで成立しています。

- HeaderDetailContractQueryBinding
- Relation Storage model / Validation / authoring / Source Index
- Desired Physical Schema / migration
- `find_by_order` PostgreSQL SQL realization
- PostgreSQL Environment composition
- instance-aware runtime execution
- 0件を正常な空collectionとして扱う
- 複数件を正常なcollectionとして扱う
- Execution Errorを0件へ変換しない

UI-less複合Scenarioでも、browseの同じSemantic mapped Order rowから`sales.order.order_id`を明示BindingでOrderLine Contract Queryへ渡します。

`tools/reference-postgres-header-detail-runtime`もありますが、実PostgreSQLへの成功Evidenceはまだ取得していません。

### 6. Validation -> Authorization -> Command

PR #265で、第一Referenceについて次を一つのScenarioとして固定しました。

```text
Authentication
  ↓ authenticated subject
Canonical業務Validation
  ↓ satisfied
Authorization
  ↓ allowed
sales.order.submit
  ↓
Command Business Result
```

次を区別します。

- Validation `Satisfied`
- Validation `NotSatisfied`
- Validation評価不能のExecution Error
- Authorization `allowed`
- Authorization `denied`
- Command Business Result
- Command Execution Error

Validation不成立・評価不能・Authorization deniedの場合はCommandへ進みません。

## Observation

Command Business Resultは外部状態の確認結果ではありません。

ADR-0023に従い、v1ではObservationをCommand結果から派生させず、**明示されたread境界を別操作として実行した結果**として扱います。

```text
Command invocation
        ↓
Business Result / acknowledgement
        ↓ separate operation
explicit read boundary
  Query / Contract Query / read adapter
        ↓
read result / Execution Error
        ↓
Observation Evidence
```

Observation対象やread方法をCommand payload、Capability名、HTTP status、field名から推測しません。

既存のread意味が使える場合、Observation用途のためだけに第二のCanonical正本を作りません。

現行`sales.order.submit`は実外部状態を書き換えるReferenceではないため、第一実証ではstateful synthetic command/read adapterを利用しています。そのEvidenceは`synthetic`として明示し、実外部Observed stateとは区別します。

production Driverにも、read identity・Evidence nature・read resultだけを保持する`ExecuteObservationRead`境界を追加済みです。実行時parameter値やcredentialはEvidence構造へ保持しません。

## Reference生成物

Reference data access / Relation Storage / Plugin Capability等のauthoring更新により、保存済み`reference/world.json`等が最新Workspaceより古くなっています。

生成物は手編集せず、#255で正規生成経路から再生成します。

対象には少なくとも次を含みます。

- `reference/world.json`
- `reference/workspace/.tei/plugin-lock.json`
- Plugin Lock fingerprintへ依存する`reference/order_rollout.durable-run.m0.21.start.json`

PR #280では`tools/reference-golden`へDurable RunのPlugin Lock依存stale検知を追加しました。Durable Run自体の生成責務は`tei deployment-start`に残します。

`reference/observed/*`は外部状態を模した固定fixtureであり、Canonical authoring変更だけを理由に自動更新しません。

実再生成は、Go commandを実行できる環境で行う必要があります。

## 最終UI-less Scenario

構造統合は実装済みです。

`internal/driver/gate_b_ui_less_scenario_test.go`で、次を同じReference Worldから一続きに接続します。

1. `browse`で受注一覧をSemantic mapped resultまで取得する
2. `record`がData-onlyで解決され、取得方法を暗黙補完しないことを確認する
3. 同じmapped Order rowから`reference`で顧客を解決する
4. 同じmapped Order rowから`header-detail`でOrderLineを解決する
5. Canonical業務Validationを評価する
6. Authentication / Authorizationを既存Capability Binding境界から通す
7. `command`で`sales.order.submit`を呼ぶ
8. Business Resultを得てもObservationを暗黙実行しない
9. 別の明示read境界からObservationを行う
10. test-local Evidenceから主要identityを確認する

この順序を汎用Workflow Engineへ一般化しません。production向けのScenario / Evidence schemaも追加しません。

この複合Scenario中の外部依存はsynthetic adapterで代替しているため、構造実証と実外部Evidenceを区別します。現在のチャット実行環境からRepositoryをcheckoutしてGo testを実行できず、GitHub Actionsも停止中なので、**実装済みだが実行成功未確認**として扱います。

## Gate Bを止めている残件

### Blocked — external `master_data`

#214で実物理Evidenceが必要です。

- physical object
- physical parameter
- raw result columns
- native SQL / API realization
- Semantic mapping
- provenance
- effective read-only boundary

### Pending — header-detail実PostgreSQL Evidence

`tools/reference-postgres-header-detail-runtime`を実PostgreSQLへ接続し、次を確認します。

- 複数件
- 0件
- Execution Error

### Pending — Reference artifact regeneration

#255で最新authoringからWorld / Plugin Lock / Durable Runを正規再生成します。

依存artifact監査とstale検知実装は済んでいますが、生成コマンド自体の実行は未完です。

### Pending — UI-less複合Scenario実行Evidence

構造統合コードは追加済みです。

通常Go runnerで実行し、synthetic構造Evidenceとして成功を確認した後も、#214 / 実PostgreSQL Evidenceの代替にはしません。

### Pending — normal CI verification

GitHub ActionsはRepositoryから明示的に停止されています。

このIssueから独断で再有効化しません。

CI停止をSemantic design gapとは扱いませんが、通常runnerで未検証の成果をgreenとも扱いません。

## Gate B Ready条件

Gate Bはコード量やIssue close数ではなく、Reference Scenarioから次を説明できるときReadyとします。

- 同じApplication定義から主要ScenarioをUIなしで一続きに再現できる
- Pattern / Query / Relation / Contract Query / Datasource / Capability / Bindingを追跡できる
- native SQLやprovider固有処理をDriver Coreへ複製していない
- raw resultとSemantic resultの対応が明示されている
- Relation traversalを物理FKや名前から推測していない
- external assetの未観測事実をTEIが補完していない
- `master_data`へのwriteを実効的な境界で防げる
- Validation不成立時にCommandへ進まない
- Authorization denied時にCommandへ進まない
- Business ResultとExecution Errorを区別できる
- Command acknowledgementとObservationを区別できる
- Query 0件とExecution Errorを区別できる
- secret / runtime parameter値を通常Evidenceへ残さない
- Driver固有状態をCanonical Worldへ逆流させない
- 最新authoringからReference artifactを再現できる
- 第一Web TargetへWeb固有の意味補完なしに同じbaselineを渡せる

## 次の作業

blocker解消効果の高い順に進めます。

1. #214でexternal `master_data`の物理Evidenceを得る
2. header-detail実PostgreSQL Evidenceを取得する
3. #255でReference World / Plugin Lock / Durable Runを正規再生成する
4. UI-less複合ScenarioをGo runnerで実行する
5. external / runtime / golden Evidenceを最終Gate B監査へ対応付ける
6. negative caseを再監査する
7. CIが別判断で再開された後、通常runnerで関連Conformanceを再検証する
8. Gate BをReady / Not Readyで再判定する

## 非目標

- Gate B前の第一Web Target実装
- Web framework選定
- 汎用Workflow Engine
- Event Sourcing基盤
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
- GitHub #69 / #70 / #214 / #244 / #255 / #266 / #270 / #273
