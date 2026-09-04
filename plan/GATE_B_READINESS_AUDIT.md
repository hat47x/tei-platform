# Gate B Readiness Audit

- 判定日: 2026-09-04
- 対象: `order_entry` Application Driver / Gate B
- 判定: **Not Ready**

## 目的

Gate B「UIなしで主要Scenarioを再現できる」を、IssueやPRのclose数ではなく、現在のReference / Conformance / 実行Evidenceから監査します。

第一Web Targetへ進む前に、Application上の不足をWeb固有コードで補わなくてもよい状態かを確認することが目的です。

この監査では、次を混同しません。

- Semantic / architecture readiness
- syntheticな構造Evidence
- 実Database / 外部システムEvidence
- Reference artifactの再現性
- 通常Go runner / CIによる実行Evidence

## 結論

Application Driverの意味境界と主要runtime接続は大きく成立しました。

PR #281では、同じ`order_entry` Applicationと同じSemantic Order rowを使い、主要境界を一つのtest-local UI-less Scenarioへ接続しています。

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

またPR #288では、external-observed Contract Queryの物理EvidenceをCanonical Worldから分離して受け入れ、明示authoringされたSQL realizationと照合する境界を追加しました。

これにより、Gate Bの主要な未完了は「Driver内部の意味設計不足」から、**実外部事実・実DB Evidence・生成物再現・runner Evidence**へ収束しています。

ただし2026-09-04時点では、次が残るためGate BをReadyとは判定しません。

1. #214 external-observed `master_data`の実Oracle物理Evidence / read-only実証
2. `Order -> OrderLine[]`の実PostgreSQL接続成功Evidence
3. #255 Reference World / Plugin Lock / Durable Runの正規再生成
4. PR #281 UI-less Scenarioの通常Go runner実行Evidence
5. PR #288 external Evidence受入境界の通常Go runner実行Evidence
6. GitHub Actionsが別判断で再開された後の通常runner再検証

## 成立している領域

### Application Pattern / Canonical resolution

**Ready**です。

同じ`order_entry`から`browse / record / reference / header-detail / command`をCanonical identityで解決できます。

この5分類をTEI全体の永久的な標準語彙へ固定したとはみなしません。

### Capability Binding / Authentication / Authorization

**Ready**です。

Application + Capability identityからBindingを明示解決し、Authentication / Authorization / Commandを同じInvocation境界から扱えます。

legacy schema、organization / role、return code等をDriver Coreへ逆流させません。

### browse / Query result mapping

**Ready**です。

`order_entry.recent_orders`について、logical Datasource、native Query、provider-side parameter binding、raw result、明示Semantic mappingまで接続しています。

raw列名、列順、Persistence MappingからSemantic attributeを推測しません。

### `Order -> Customer` reference

**意味・runtime境界はReady、実外部readはBlocked**です。

成立済み:

- Semantic Relation key
- Reference Contract Query invocation
- typed Contract Query signature
- Contract Query SQL realization model
- Environment / Database instance / executor Capability composition
- instance-aware runtime Registry
- Oracle向け最小`database/sql` Runner
- 0件とExecution Errorの分離
- SQL realization欠如時に物理Runnerを呼ばず停止する境界

PR #281でもbrowseの`sales.order.customer_id`を、別名や物理FKから再推測せずCustomer referenceへ渡します。

#### external Evidence intake

PR #288で**Evidence intake boundaryはReady**になりました。

`internal/database/externalqueryevidence`と`tools/reference-external-query-evidence`では次を照合します。

- Database / Contract / Query target
- provider
- `external-observed` ownership
- Contract input → physical parameter
- raw result → Semantic attribute
- native SQL source fingerprint
- physical facts provenance
- read-only mechanism / Evidence参照

対応するSQL realizationがWorkspaceに存在しない場合、Evidenceから生成・補完せず停止します。

一方、第一Referenceで本当に利用するOracle object / parameter / raw columns / native SQL / provenance / read-only強制Evidenceはまだありません。

したがって#214は、**受入設計不足ではなく実外部事実の取得待ち**としてGate B blockerです。

### `Order -> OrderLine[]` header-detail

**意味・保存・runtimeはReady、実DB EvidenceはPending**です。

成立済み:

- keyless Semantic Relationを維持したtraversal
- Pattern外の`HeaderDetailContractQueryBinding`
- Database-owned Relation Storage
- `sales.order.order_id -> order_id` support column realization
- Desired Physical Schema / migration
- `find_by_order` PostgreSQL SQL realization
- PostgreSQL Environment composition
- Contract Query runtime
- 0件 / 複数件 / Execution Errorの区別
- 実PostgreSQL Evidence tool

PR #281でもbrowseの同じ`sales.order.order_id`をheader-detail Queryへ引き継ぎます。

残るのは`tools/reference-postgres-header-detail-runtime`を実PostgreSQLへ接続し、複数件・0件・Execution Errorを確認することです。

### Validation -> Authorization -> Command

**Ready**です。

PR #265で次を分離しています。

- Validation `Satisfied`
- Validation `NotSatisfied`
- Validation評価不能のExecution Error
- Authorization `allowed`
- Authorization `denied`
- Command Business Result
- Command Execution Error

Validation不成立・評価不能・Authorization deniedの場合はCommandへ進みません。

PR #281ではheader-detail resultをValidation inputへ接続し、Authentication subjectをAuthorizationへ、同じ`order_id`をCommandへ引き継ぎます。

### Command acknowledgement -> Observation

**境界はReady、Evidenceはsynthetic**です。

ADR-0023とPR #275以降で、CommandとObservationを別操作として扱います。

```text
Command invocation
  ↓
Business Result / acknowledgement

--- separate operation ---

explicit read boundary
  ↓
Observation result / Execution Error
```

production Driverの`ExecuteObservationRead`は、read identity / Evidence nature / resultを保持し、runtime parameterやcredentialを通常Evidenceへ残しません。

Command payloadからObserved stateを推測せず、Observation失敗で成功済みCommand Business Resultを書き換えません。

## UI-less最終Scenario

**構造統合はReady (synthetic structure)、通常runner実行はPending**です。

正本は`internal/driver/final_gate_b_scenario_test.go`です。

このScenarioは新しいproduction Workflow / Scenario engineを導入していません。既存Driver APIをtest-localに接続しています。

特に、browseで得た同じSemantic identityを次へ引き継ぎます。

- `customer_id` → reference
- `order_id` → header-detail
- `order_id` → command
- `order_id` → separate Observation read

header-detailのOrderLine値もCanonical Validationへ渡します。

Database read / Command side effect / Observation readにはsynthetic adapterを含むため、このScenarioを実Oracle / 実PostgreSQL / 実外部状態Evidenceへ格上げしません。

## Reference artifact

#255では依存関係の監査とstale検知まで進んでいます。

- `reference/world.json`: Canonical Workspaceからの生成golden
- `reference/workspace/.tei/plugin-lock.json`: Canonical Workspaceからの生成golden
- 保存Durable Run: Workspace + Plugin Lock + fixed observed fixtureに依存
- `reference/observed/*`: external stateを模した固定fixture

PR #280で、保存Durable Runの`pluginLockFingerprint`と現在Plugin Lockの不一致も検知します。

残るのは正規Go commandによるWorld / Plugin Lock / Durable Runの実再生成と差分確認です。

## CI / runner

GitHub Actionsは2026-09-03にRepositoryから明示的に停止されています。

CI停止そのものをSemantic design gapとは扱いません。一方、未実行をgreenとも扱いません。

本チャット環境からも`github.com`を名前解決できずcheckoutできないため、PR #281 / #288のGo test successは未確認です。

CI再開は別の明示判断に従い、この監査から独断で変更しません。

## Gate B条件別監査

| 条件 | 状態 | 根拠 / 残件 |
| --- | --- | --- |
| 同じApplicationから主要Scenarioを構成 | Ready (synthetic structure) | PR #281 |
| 同じSemantic Order identityを後続境界へ継承 | Ready (synthetic structure) | PR #281 |
| Pattern / Query / Relation / Datasource / Capability / Bindingを追跡 | Ready | Driver / Source Index / Conformance |
| browseをSemantic resultまで実行 | Ready | 既存PostgreSQL Evidence |
| external Query Evidenceを非Canonicalに受入 | Ready | PR #288 |
| EvidenceからSQL realizationを推測・補完しない | Ready | PR #288 |
| referenceを実Oracle readまで実行 | Blocked | #214実物理Evidence待ち |
| `master_data` read-onlyを実効保証 | Blocked | #214実Evidence待ち |
| header-detail意味・storage・runtime | Ready | 既存Reference実装 |
| header-detail実PostgreSQL Evidence | Pending | tool実行待ち |
| Validation不成立時にCommandへ進まない | Ready | PR #265 |
| Authorization denied時にCommandへ進まない | Ready | PR #265 |
| Business Result / Execution Errorを分離 | Ready | ADR-0020 |
| Command acknowledgement / Observationを分離 | Ready (synthetic) | ADR-0023 |
| secret / runtime parameterを通常Evidenceへ保存しない | Ready | runtime境界 |
| 最新authoringからReference artifactを再現 | Pending | #255実再生成待ち |
| 最終Scenarioを一続きに構成 | Ready (synthetic structure) | PR #281 |
| PR #281を通常Go runnerで実証 | Pending | 実行環境 / CI待ち |
| PR #288を通常Go runnerで実証 | Pending | 実行環境 / CI待ち |
| 第一Web Targetへ同じ意味を渡せる | Partial | external/runtime Evidence完成待ち |

## Gate Bを止めないもの

次はGate Bの前提にしません。

- Web framework選定
- 第一Web Target実装
- 完全なTarget-neutral Presentation IR
- Wails採否
- Editor / AI実装
- 第二Target
- 汎用Workflow Engine / Event Sourcing基盤
- physical Foreign Key自動生成
- production HA / Connection Pool一般化
- production向け汎用Observation schema
- external Database全体のTEI管理化

## 次の実行順

blocker解消効果の高い順に進めます。

1. #214で実Oracle physical object / parameter / raw columns / native source / provenance / read-only Evidenceを取得する
2. Evidenceを根拠に第一ReferenceのSQL realizationを明示authoringし、PR #288の受入toolで照合する
3. `tools/reference-postgres-header-detail-runtime`を実PostgreSQLで実行する
4. #255でReference World / Plugin Lock / Durable Runを正規再生成する
5. PR #281 / #288をGo runnerで実行する
6. external / DB / golden EvidenceをGate B監査へ対応付ける
7. read-only / secret / 0件 / Execution Error / Observation期待不一致等のnegative caseを再監査する
8. CIが別判断で再開された後、通常runnerで関連Conformanceを再実行する
9. Gate BをReady / Not Readyで再判定する

## Gate B Readyの定義

Gate Bは次をEvidenceから説明できるときReadyとします。

- UIなしで主要Scenarioを一続きに再現できる
- 同じ業務identityを重要境界間で明示的に継承できる
- 重要な意味・実現選択へSource / Bindingを辿れる
- external assetの未観測事実をTEIが推測していない
- external EvidenceをCanonical business meaningと混同していない
- Business Result / Execution Error / Observationを区別できる
- synthetic Evidenceと実外部Evidenceを区別できる
- read-only / secret等の安全境界を実Evidenceから説明できる
- 最新authoringからCanonical / Reference artifactを再現できる
- 第一Web TargetへWeb固有の意味補完なしに同じbaselineを渡せる

この条件が揃うまでは、第一Web Targetの実装着手条件を満たしたとは判定しません。
