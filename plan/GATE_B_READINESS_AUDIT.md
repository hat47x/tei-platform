# Gate B Readiness Audit

- 判定日: 2026-09-04
- 対象: `order_entry` Application Driver / Gate B
- 判定: **Not Ready**

## 目的

Gate B「UIなしで主要Scenarioを再現できる」を、個別IssueやPRのclose状態ではなく、現在のReference / Conformance / 実行Evidenceから監査します。

Gate Bの目的は、第一Web Targetへ進む前に、Application上の不足をWeb固有コードで補わずに済む状態を確認することです。

実装済み境界、synthetic Evidence、未取得の外部Evidence、生成物再現性、通常runner検証を分けて扱います。

## 結論

Application Driverの意味境界と主要runtime接続は大きく成立しました。

PR #281では、同じ`order_entry` Application定義と同じSemantic Order rowを使い、次の主要境界を一つのtest-local UI-less Scenarioへ接続しています。

```text
Authentication
  ↓ authenticated subject
browse recent_orders
  ↓ same Semantic Order row
record order_record
  ↓
reference customer_reference
  ↓ same customer_id
header-detail order_lines
  ↓ same order_id / returned OrderLine values
Canonical Validation
  ↓ satisfied
Authorization
  ↓ allowed
sales.order.submit
  ↓ same order_id / Command Business Result
--- separate operation ---
explicit Observation read
  ↓ same order_id / submitted
```

ここで重要なのは、`customer_id`や`order_id`を別名・物理列・FKから再推測せず、browseでSemantic mapping済みの同じrow identityを後続境界へ引き継いでいることです。またheader-detailで得たOrderLine値をValidation inputへ渡しています。

ただしPR #281はDatabase read、Command side effect、Observation readにsynthetic adapterを含む**構造統合Evidence**です。実Oracle / 実PostgreSQL / 実外部CommandのEvidenceや通常runner成功を代替しません。

2026-09-04時点では次が残るため、Gate BをReadyとは判定しません。

1. external-observed `master_data`の実物理Evidenceが未確定で、`Order -> Customer`を実readまで通せない
2. `Order -> OrderLine[]`の実PostgreSQL Evidence toolはあるが、実接続成功をまだ取得していない
3. Reference World / Plugin Lock / 依存Durable Run等を最新Canonical Workspaceから正規再生成・確認していない
4. PR #281を通常Go runnerで実行・成功確認できていない
5. GitHub Actionsは明示停止中であり、通常runner再検証は保留されている

残件は「Driver設計未成立」ではなく、**外部事実、実DB Evidence、生成物再現性、Scenario実行Evidence、通常runner検証**へ収束しています。

## 現在までに成立している領域

### Application Pattern / Canonical resolution

Readyです。

`browse / record / reference / header-detail / command`を同じ`order_entry`から解決できます。

5分類をTEI全体の永久的な標準語彙へ固定したとはみなしません。

### Capability Binding / Authentication / Authorization

Readyです。

Application + Capability identityからBindingを明示解決し、Authentication / Authorization / Commandを同じInvocation境界から扱えます。

legacy schemaやreturn codeをDriver Coreへ逆流させません。

### browse / Query result mapping

Readyです。

`order_entry.recent_orders`について、logical Datasource、native Query、provider-side parameter binding、raw result、明示的なSemantic result mappingまで接続しています。

raw列名や列順からSemantic attributeを推測しません。

### `Order -> Customer` reference

**構造はReady、実物理EvidenceはBlocked**です。

成立済み:

- Semantic Relation key
- Reference Contract Query invocation
- typed Contract Query signature
- Contract Query SQL realization model
- Environment / Database instance / executor Capability composition
- instance-aware runtime Registry
- Oracle向け最小`database/sql` Runner
- 0件とExecution Errorを分離する契約

PR #281でもbrowseのSemantic mapped rowから`sales.order.customer_id`を取り出し、Relation / invocation / `master_data.dev` / `database.query.oracle`という明示境界を通してCustomer resultへ接続します。

ただしexternal-observed `master_data`の実際のtable / view / parameter / raw column / native SQL / provenanceはRepository上にありません。未観測のOracle物理情報をTEI側から作らないため、#214をGate Bの外部Evidence blockerとして維持します。

### `Order -> OrderLine[]` header-detail

**意味・保存・runtimeはReady、実DB EvidenceはPending**です。

成立済み:

- keylessなSemantic Relationを維持したtraversal
- Pattern外の`HeaderDetailContractQueryBinding`
- Database-ownedなRelation Storage
- `sales.order.order_id -> order_id` support column realization
- Desired Physical Schema / migration経路
- `find_by_order` PostgreSQL SQL realization
- PostgreSQL Environment composition
- Contract Query runtime境界
- 0件 / 複数件 / Execution Errorの区別
- 実PostgreSQL Evidence tool

PR #281でもbrowseの同じ`sales.order.order_id`をheader-detail Query inputへ引き継ぎます。

`tools/reference-postgres-header-detail-runtime`を実PostgreSQLへ接続し、複数件、0件、Execution Errorを実際に確認する作業が残っています。

### Validation -> Authorization -> Command

Readyです。

PR #265で、Validation `NotSatisfied`、Validation評価不能のExecution Error、Authorization `denied`、Command Business Resultを分離しています。

PR #281ではさらに、header-detailのSemantic OrderLine resultからValidation inputを構成し、Authenticationで得たsubjectをAuthorizationへ引き継ぎ、browseで選択した同じ`order_id`を`sales.order.submit`へ渡しています。

Command Business Resultだけを「外部状態変更を確認済み」とは扱いません。

### Command acknowledgement -> Observation

**境界はReady、Evidenceはsyntheticです。**

ADR-0023とPR #275以降では、CommandとObservation readを別操作として扱います。

```text
sales.order.submit
  ↓
Command Business Result / acknowledgement

--- separate operation ---

explicit read boundary
  ↓
Observed state / Execution Error
```

production Driverには`ExecuteObservationRead`の最小runtime境界があります。read identity、Evidence nature、read resultを保持し、runtime parameter値やcredentialは通常Evidenceへ持ち込みません。

PR #281でもCommandとObservationを別操作にし、browseから引き継いだ同じ`order_id`のsynthetic stateをreadしています。

synthetic Evidenceを実外部状態のEvidenceとして扱いません。

## UI-less最終Scenario

**構造統合はReady (synthetic)、実行確認はPending**です。

正本は`internal/driver/final_gate_b_scenario_test.go`です。

新しいproduction Workflow / Scenario engine / Evidence modelは導入していません。既存の次の境界をtest-localに接続しています。

- `InvokeCapability`
- `ExecuteBrowseQuery`
- `ResolvePattern`
- `ExecuteReferenceContractQueryInEnvironment`
- `ExecuteHeaderDetailContractQueryInEnvironment`
- Canonical Validation evaluator
- `ExecuteObservationRead`

record Patternは現行契約どおりSemantic Data解決までとし、未定義のedit runtimeやPersistence動作をtest都合で補いません。

このScenarioで確認するのは**境界間の接続とidentity継承**です。次の代替にはしません。

- #214 external `master_data`実物理read / read-only Evidence
- header-detail実PostgreSQL接続成功Evidence
- #255 Reference artifact再生成
- 実外部Command side effect / Observation Evidence
- 通常runner / CI Evidence

本チャットのローカル実行環境からGitHubをcheckoutしようとしましたが、GitHub hostを名前解決できずGo testを実行できませんでした。またGitHub Actionsも停止中です。したがって、PR #281のmergeをテストgreenとは扱いません。

## Reference生成物

Gate Bではauthoritative sourceから同じCanonical meaningを再構成できる必要があります。

#255ではReference artifact依存の監査まで進みました。

- `reference/world.json`: Canonical Workspaceからの生成golden
- `reference/workspace/.tei/plugin-lock.json`: Canonical Workspaceからの生成golden
- `reference/order_rollout.durable-run.m0.21.start.json`: Workspace + Plugin Lock + fixed observed fixtureから`tei deployment-start`で生成するgolden
- `reference/observed/*`: external stateを模した固定fixtureであり、Canonical変更だけでは自動更新しない

PR #280では`reference-golden`へ、現在のPlugin Lockと保存済みDurable Runの`pluginLockFingerprint`依存stale検知を追加しました。Durable Runの生成責務は`tei deployment-start`へ残しています。

ただしWorld / Plugin Lock / Durable Runの正規Go commandによる実再生成はまだ実行していません。

## CI / 通常runner

GitHub Actionsは2026-09-03にRepositoryから明示的に停止されています。

CI停止そのものをApplication意味の設計Gapとは扱いません。一方、通常runnerで再検証されていない状態をgreenとも扱いません。

CI再開は別の明示判断に従い、この監査から独断で変更しません。

## Gate B条件別監査

| 条件 | 状態 | 根拠 / 残件 |
| --- | --- | --- |
| 同じApplication定義から主要Scenarioを構成できる | Ready (synthetic structure) | PR #281。通常runner実行確認はPending |
| 同じSemantic Order identityを後続境界へ引き継げる | Ready (synthetic structure) | PR #281でcustomer_id / order_idをreference / header-detail / command / observationへ接続 |
| Pattern / Query / Relation / Datasource / Capability / Bindingを追跡できる | Ready | Driver / Source Index / 各Conformance |
| browseをSemantic resultまで実行できる | Ready | 既存PostgreSQL raw result + explicit mapping Evidence |
| referenceを実外部readまで実行できる | Blocked | #214 external `master_data`物理Evidence待ち |
| header-detailの意味・storage・runtimeが成立 | Ready | PR #235〜#262、正本復元PR #274 |
| header-detail実PostgreSQL Evidence | Pending | tool実行待ち |
| Validation不成立時にCommandへ進まない | Ready | PR #265 |
| Authorization denied時にCommandへ進まない | Ready | PR #265 |
| Business ResultとExecution Errorを区別 | Ready | ADR-0020 / Driver Conformance |
| Command acknowledgementとObservationを分離 | Ready (synthetic) | ADR-0023 / #266 / PR #275以降 |
| read-only `master_data`を実効的に保証 | Blocked | #214 |
| secret / runtime parameterを通常Evidenceへ保存しない | Ready | 各runtime境界 |
| 最新authoringからReference生成物を再現 | Pending | #255。依存監査/stale検知済み、実再生成待ち |
| 最終Scenarioを一続きに構成 | Ready (synthetic structure) | PR #281 |
| 最終Scenarioを通常Go runnerで実証 | Pending | 実行環境/CI待ち |
| 通常runnerで関連Conformanceを再検証済み | Pending | CI再開判断待ち |
| 第一Web Targetへ同じ意味を渡せる | Partial | Gate B external/runtime Evidence完成待ち |

## Gate Bを止めないもの

次はGate Bの前提にしません。

- Web frameworkの選定
- 第一Web Targetの実装
- 完全なTarget-neutral Presentation IR
- Wails採否
- Editor / AI実装
- 第二Target
- 汎用Workflow Engine / Event Sourcing基盤
- physical Foreign Keyの自動生成
- production HA / Connection Pool一般化
- production向け汎用Observation schema

## 次の実行順

1. #214でexternal `master_data`の物理Evidenceを得る
2. `tools/reference-postgres-header-detail-runtime`を実PostgreSQLで実行する
3. #255でReference World / Plugin Lock / Durable Runを正規再生成する
4. PR #281のUI-less ScenarioをGo runnerで実行する
5. external / runtime / golden Evidenceを最終Gate B監査へ対応付ける
6. secret / read-only / 0件 / Execution Error / Observation期待不一致等のnegative caseを再監査する
7. CIが別判断で再開された後、通常runnerで関連Conformanceを再実行する
8. 本文書を更新し、Gate BをReady / Not Readyで再判定する

## Gate B Readyの定義

Gate BはIssue close数ではなく、次をEvidenceから説明できるときReadyとします。

- UIなしで主要Scenarioを一続きに再現できる
- 同じ業務identityを重要境界間で明示的に継承できる
- すべての重要な意味・実現選択へSource / Bindingを辿れる
- 外部資産の未観測事実をTEIが推測していない
- Business Result / Execution Error / Observationを区別できる
- synthetic Evidenceと実外部Evidenceを区別できる
- read-only / secret等の安全境界を保持できる
- 最新authoringからCanonical / Reference artifactを再現できる
- 第一Web Targetへ、Web側で意味を補わず同じbaselineを渡せる

この条件が揃うまでは、第一Web Targetの実装着手条件を満たしたとは判定しません。
