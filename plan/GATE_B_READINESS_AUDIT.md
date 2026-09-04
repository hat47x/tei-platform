# Gate B Readiness Audit

- 判定日: 2026-09-04
- 対象: `order_entry` Application Driver / Gate B
- 判定: **Not Ready**

## 目的

Gate B「UIなしで主要Scenarioを再現できる」を、個別IssueやPRのclose状態ではなく、現在のReference / Conformance / 実行Evidenceから監査します。

Gate Bの目的は、第一Web Targetへ進む前に、Application上の不足をWeb固有コードで補わずに済む状態を確認することです。

この監査では、実装済み境界、synthetic Evidence、未取得の外部Evidence、生成物再現性、通常runner検証を分けて扱います。

## 結論

Application Driverの**意味境界と主要runtime接続は大きく成立しており、browse -> record -> reference -> header-detail -> Validation -> Authorization -> Command -> Observationを同じReference World上で接続するtest-localなUI-less複合Scenarioも追加しています**。

ただし、この複合Scenarioはsynthetic adapterを含む構造実証であり、実外部Database Evidenceや通常runner成功を代替しません。2026-09-04時点では次が残るため、Gate BをReadyとは判定しません。

1. external-observed `master_data`の実物理Evidenceが未確定で、`Order -> Customer`を実readまで通せない
2. `Order -> OrderLine[]`の実PostgreSQL Evidence toolはあるが、実接続成功をまだ取得していない
3. Reference World / Plugin Lock / 依存Durable Run等の生成物を最新Canonical Workspaceから正規再生成・確認していない
4. 新しいUI-less複合Scenarioを通常Go runnerで実行・成功確認できていない
5. GitHub Actionsは明示停止中であり、通常runnerでの再検証は保留されている

これらは「Application Driverの設計が未成立」という一種類の不足ではありません。**外部事実、実DB Evidence、生成物再現性、Scenario実行Evidence、通常runner検証**という異なる残件です。

## 現在までに成立している領域

### Application Pattern / Canonical resolution

Readyです。

`browse / record / reference / header-detail / command`の5 Patternを同じ`order_entry`から解決できます。

5分類を永久的な標準語彙へ固定したとはみなしません。

### Capability Binding / Authentication / Authorization

Readyです。

Application + Capability identityからBindingを明示解決し、Authentication / Authorization / Commandを同じInvocation境界から扱えます。

legacy schemaやreturn codeをDriver Coreへ逆流させません。

### browse / Query result mapping

Readyです。

`order_entry.recent_orders`について、logical Datasource、native Query、provider-side parameter binding、raw result、明示的なSemantic result mappingまで接続しています。

raw列名や列順からSemantic attributeを推測しません。

### `Order -> Customer` referenceの意味・runtime境界

**構造はReady、実物理EvidenceはBlocked**です。

次は成立しています。

- Semantic Relation key
- Reference Contract Query invocation
- typed Contract Query signature
- Contract Query SQL realization model
- Environment / Database instance / executor Capability composition
- instance-aware runtime Registry
- Oracle向け最小`database/sql` Runner
- 0件とExecution Errorを分離する契約

UI-less複合Scenarioでも、browseでSemantic mapping済みのOrder rowからCustomer Relation keyを取得し、`development / master_data.dev / database.query.oracle`という明示Bindingへ渡す構造を再利用します。

ただし、external-observed `master_data`の実際のtable / view / parameter / raw column / native SQL / provenanceはRepository上にありません。

未観測のOracle物理情報をTEI側から作らないため、#214をGate Bの外部Evidence blockerとして維持します。

### `Order -> OrderLine[]` header-detail

**意味・保存・runtimeはReady、実DB Evidenceは保留**です。

次は成立しています。

- keylessなSemantic Relationを維持したtraversal
- Pattern外の`HeaderDetailContractQueryBinding`
- Database-ownedなRelation Storage
- `sales.order.order_id -> order_id` support column realization
- Desired Physical Schema / migration経路
- `find_by_order` PostgreSQL SQL realization
- PostgreSQL Environment composition
- Contract Query runtime実行境界
- 0件 / 複数件 / Execution Errorの区別
- 実PostgreSQL Evidence tool

UI-less複合Scenarioでも、browseの同じSemantic mapped rowから`sales.order.order_id`を読み、明示Bindingを通して`sales_core.dev`のOrderLine Contract Queryへ渡します。

実装と実DB成功Evidenceを混同しません。

`tools/reference-postgres-header-detail-runtime`を実PostgreSQLへ接続し、複数件、0件、Execution Errorを実際に確認する作業が残っています。

### Validation -> Authorization -> Command

Readyです。

PR #265で、第一Referenceについて次の順序を既存境界だけから再現しています。

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

次を区別できます。

- Validation `NotSatisfied`
- Validation評価不能のExecution Error
- Authorization `denied`
- Command Business Result

Validation不成立、評価不能、Authorization deniedの各場合にCommandへ進まないことも固定しています。

Command Business Resultだけを「外部状態変更を確認済み」とは扱いません。

### Command acknowledgement -> Observation

**境界はReadyです。ただしEvidenceはsyntheticです。**

PR #275以降のReference Scenarioでは、CommandとObservation readを別操作として扱います。

```text
sales.order.submit
  ↓
Command Business Result / acknowledgement

--- 別操作 ---

synthetic.order_state.read
  ↓
Observed state
  ↓
Expected stateとの評価
```

次をConformanceで区別します。

- Command実行だけではObservation readを暗黙実行しない
- Command payloadへObserved stateを混入させない
- Observation readは明示されたread boundary identityを持つ
- Observation read成功と、期待状態との一致を別に評価する
- readが成功しても期待状態と不一致であり得る
- Observation Execution ErrorをCommand Business Resultへ逆流させない
- synthetic Evidenceを実外部状態のEvidenceとして扱わない

このEvidenceは、CommandとObservationを分離する設計境界を確認するためのtest-localなstateful adapterによるものです。実外部システムの状態変更精度を証明するものではありません。

v1では新しいObservation Pattern、汎用Observation result schema、Workflow Engine、Event Sourcing基盤を追加していません。既存Query / Contract Query / read adapter等の明示されたread境界をObservation用途でも利用する方針を維持します。

詳細判断はADR-0023と#266を正とします。

## UI-less複合Scenario

**構造統合は実装済み、実行EvidenceはPending**です。

`internal/driver/gate_b_ui_less_scenario_test.go`で、同じReference Worldから次を一つのtest-local Scenarioとして接続します。

```text
browse recent_orders
  ↓ Semantic mapped Order row
record order_record
  ↓
reference customer_reference
  ↓
header-detail order_lines
  ↓
Canonical Validation
  ↓
Authentication / Authorization
  ↓
sales.order.submit
  ↓ Command Business Result
separate explicit Observation read
```

このScenarioで新しいproduction Workflow / Scenario / Evidence modelは導入していません。既存Driver APIとtest helperを順に呼び、最小のtest-local Evidence構造でidentityを確認します。

また、外部系をsyntheticに置き換えた箇所を実外部Evidenceとして昇格させません。

- browse rowはsynthetic
- Customer / OrderLine Contract Query executorもsynthetic runtime adapter
- Authentication / Authorization / Commandも既存Capability Binding境界へ注入したsynthetic adapter
- Observation natureは`synthetic`

このため、このScenarioが通常runnerで成功しても、それだけで#214や実PostgreSQL Evidenceの代替にはなりません。目的は「個別に成立したDriver境界が、同じApplication文脈で無理なく接続可能か」を検証することです。

本チャット環境からGitHub RepositoryをcheckoutしてGo testを実行する経路は利用できず、GitHub Actionsも停止中です。したがって、現時点ではコード追加・静的契約確認までを成立事実とし、実行成功とは記録しません。

## Reference生成物

Gate Bでは、authoritative sourceから同じCanonical meaningを再構成できる必要があります。

Reference data access / Relation Storage / Plugin Capability等のauthoring更新後、少なくとも`reference/world.json`が現在のCanonical Workspaceより古いことが確認されています。

また、OrderLineのRelation Storage / SQL realizationについてbranch履歴上の正本脱落を検出し、PR #274でPR #251由来の正本authoringを復元しています。この復元をgolden再生成完了とは扱いません。

#255ではReference artifactの依存監査まで進み、次を区別しています。

- `reference/world.json`: Canonical Workspaceからの生成golden
- `reference/workspace/.tei/plugin-lock.json`: Canonical Workspaceからの生成golden
- `reference/order_rollout.durable-run.m0.21.start.json`: Workspace + Plugin Lock + fixed observed fixtureから`tei deployment-start`で生成するgolden
- `reference/observed/*`: external stateを模した固定fixtureであり、Canonical変更だけでは自動更新しない

PR #280で`reference-golden`にPlugin Lockと保存済みDurable Runのfingerprint依存stale検知も追加しました。

ただし、正規GoコマンドによるWorld / Plugin Lock / Durable Runの実再生成はまだ実行していません。

## CI / 通常runner

GitHub Actionsは2026-09-03にRepositoryから明示的に停止されています。

CI停止そのものをApplication意味の設計Gapとは扱いません。一方、通常runnerで再検証されていない状態をgreenとも扱いません。

したがってGate B監査では、次を分けます。

- Semantic / architecture readiness
- synthetic / local / manual / deterministic Evidence readiness
- external-system Evidence readiness
- normal CI verification readiness

CI再開は別の明示判断に従い、この監査から独断で変更しません。

## Gate B条件別監査

| 条件 | 状態 | 根拠 / 残件 |
| --- | --- | --- |
| 同じApplication定義からScenarioを構成できる | Ready (synthetic structure) | UI-less複合Scenarioを追加。実行確認はPending |
| Pattern / Query / Relation / Datasource / Capability / Bindingを追跡できる | Ready | Driver / Source Index / 各Conformance |
| browseをSemantic resultまで実行できる | Ready | 実PostgreSQL raw result + explicit mappingの既存Evidence |
| referenceを実外部readまで実行できる | Blocked | #214 external `master_data`物理Evidence待ち |
| header-detailの意味・storage・runtimeが成立 | Ready | PR #235〜#262および正本復元PR #274 |
| header-detail実PostgreSQL Evidence | Pending | tool実行待ち |
| Validation不成立時にCommandへ進まない | Ready | PR #265 |
| Authorization denied時にCommandへ進まない | Ready | PR #265 |
| Business ResultとExecution Errorを区別 | Ready | ADR-0020 / Driver Conformance |
| Command acknowledgementとObservationを分離 | Ready (synthetic) | ADR-0023 / #266 / PR #275以降のConformance |
| Observation read結果と期待状態評価を分離 | Ready (synthetic) | test-local Observation Evidence |
| read-only `master_data`を実効的に保証 | Blocked | #214 |
| secret / runtime parameterを通常Evidenceへ保存しない | Ready | 各runtime境界 |
| 最新authoringからReference生成物を再現 | Pending | #255。依存監査/stale検知済み、実再生成待ち |
| 最終Scenario Evidenceを一続きに再現 | Partial | synthetic構造統合済み、Go実行・外部Evidence統合待ち |
| 通常runnerで再検証済み | Pending | CI再開判断待ち |
| 第一Web Targetへ同じ意味を渡せる | Partial | Gate C契約は定義済み。Gate B external/runtime Evidence完成待ち |

## Gate Bを止めないもの

次はGate Bの前提にしません。

- Web frameworkの選定
- 第一Web Targetの実装
- 完全なTarget-neutral Presentation IR
- Wails採否
- Editor実装
- AI Proposal schema / AI provider
- 第二Target
- 汎用Workflow Engine
- Event Sourcing基盤
- physical Foreign Keyの自動生成
- production HA / Connection Pool一般化
- production向け汎用Observation schema

## 次の実行順

Gate Bへ最短で進む順序を、実装量ではなくblocker解消効果から次のように整理します。

1. #214でexternal `master_data`の物理Evidenceを得る
2. `tools/reference-postgres-header-detail-runtime`を実PostgreSQLで実行する
3. #255でReference World / Plugin Lock / Durable Runを正規再生成する
4. UI-less複合ScenarioをGo runnerで実行し、synthetic構造Evidenceを確定する
5. external/read/golden Evidenceを最終Gate B Scenarioへ対応付ける
6. secret / read-only / 0件 / Execution Error / Observation期待不一致等のnegative caseを再監査する
7. CIが別判断で再開された後、通常runnerで関連Conformanceを再実行する
8. 本文書を更新し、Gate BをReady / Not Readyで再判定する

Observation境界のためだけに新しいproduction schemaを追加する作業は、この実行順には含めません。

## Gate B Readyの定義

Gate BはIssue close数ではなく、次をEvidenceから説明できるときReadyとします。

- UIなしで主要Scenarioを一続きに再現できる
- すべての重要な意味・実現選択へSource / Bindingを辿れる
- 外部資産の未観測事実をTEIが推測していない
- Business Result / Execution Error / Observationを区別できる
- Observation read結果と期待状態評価を混同しない
- synthetic Evidenceと実外部Evidenceを区別できる
- read-only / secret等の安全境界を保持できる
- 最新authoringからCanonical / Reference artifactを再現できる
- 第一Web Targetへ、Web側で意味を補わず同じbaselineを渡せる

この条件が揃うまでは、第一Web Targetの実装着手条件を満たしたとは判定しません。
