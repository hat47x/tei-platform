# Gate B Readiness Audit

- 判定日: 2026-09-04
- 対象: `order_entry` Application Driver / Gate B
- 判定: **Not Ready**

## 目的

Gate B「UIなしで主要Scenarioを再現できる」を、IssueやPRのclose数ではなく、現在のReference / Conformance / 実行Evidenceから判定します。

Gate Bの目的は、第一Web Targetへ進む前に、Application上の不足をWeb固有コードで補わずに済む状態を確認することです。

Evidence取得の具体的な手順は[`GATE_B_EVIDENCE_ACQUISITION.md`](GATE_B_EVIDENCE_ACQUISITION.md)を参照します。

## 結論

Application Driverの主要な意味境界とScenario構造は成立しています。

PR #285で、Contract Query、header-detail、Relation Storage、Observation、最終UI-less Scenario、Reference authoringを`main`へ収束しました。したがって、現在のGate B残件は「Driverの中心設計が未成立」という種類ではなく、**外部事実、実DB、生成物再現、通常runnerというEvidence取得**へ収束しています。

現在Gate Bを直接止めているのは次の四系統です。

1. external-observed `master_data`の実物理 / read-only Evidence
2. `Order -> OrderLine[]`の実PostgreSQL Evidence
3. Reference World / Plugin Lock / Durable Runの正規再生成Evidence
4. 最終UI-less Scenarioの通常Go runner実行Evidence

GitHub Actionsによる継続的な再検証は重要ですが、この四系統のProduct Evidenceと同一ではありません。Actions停止そのものを意味設計Gapへ読み替えず、逆に未実行をgreenとも扱いません。

## `main`で成立している基線

### Application Pattern / Canonical resolution

**Ready**です。

`browse / record / reference / header-detail / command`を同じ`order_entry` ApplicationからCanonical identityで解決できます。

この5分類をTEI全体の永久的な標準語彙へ固定したとはみなしません。

### Capability Binding / Authentication / Authorization

**Ready**です。

Application + Capability identityからBindingを明示解決し、Authentication / Authorization / Commandを同じInvocation境界から扱えます。

legacy schemaやreturn codeをDriver Coreへ逆流させません。

### browse / Query result mapping

**Ready**です。

`order_entry.recent_orders`について、logical Datasource、native Query、provider-side parameter binding、raw result、明示的なSemantic result mappingまで接続しています。

raw列名、列順、Persistence MappingからSemantic attributeを推測しません。

### `Order -> Customer` reference

**構造はReady、実物理EvidenceはBlocked**です。

成立済み:

- Semantic Relation key
- Reference Contract Query invocation
- typed Contract Query signature
- Contract Query SQL realization model
- explicit input / result mapping
- Environment / Database instance / executor Capability composition
- instance-aware runtime Registry
- Oracle向け最小`database/sql` Runner
- 0件とExecution Errorを分離する契約

一方、external-observed `master_data`の実際のtable / view / parameter / raw column / native SQL / provenanceは未観測です。

未観測のOracle物理情報をTEI側から作らないため、#214を外部Evidence blockerとして維持します。read-onlyについても宣言だけではなく、credential / role / session / view / API等の実効的な強制境界をEvidenceとして確認します。

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

実行入口は`tools/reference-postgres-header-detail-runtime`です。

このtoolはfixture用`public.order_lines`を作成・削除するため、共有・業務DBではなく破棄可能な専用PostgreSQL databaseで実行します。

### Validation -> Authorization -> Command

**Ready**です。

次を区別できます。

- Validation `NotSatisfied`
- Validation評価不能のExecution Error
- Authorization `denied`
- Command Business Result

Validation不成立、評価不能、Authorization deniedではCommandへ進みません。

最終Scenarioではheader-detailのSemantic OrderLine resultをValidation inputへ渡し、Authenticationで得たsubjectをAuthorizationへ、browseで選択した同じ`order_id`をCommandへ引き継ぎます。

### Command acknowledgement -> Observation

**Ready (synthetic Evidence)**です。

ADR-0023に従い、CommandとObservation readを別操作として扱います。

```text
sales.order.submit
  ↓
Command Business Result / acknowledgement

--- separate operation ---

explicit read boundary
  ↓
Observed state / Execution Error
```

production Driverには`ExecuteObservationRead`の最小runtime境界があります。

- Observation対象をCommand payloadや名前から推測しない
- read identity / Evidence nature / resultを保持する
- runtime parameter値やcredentialを通常Evidenceへ保存しない
- read成功と期待状態評価を分ける
- Observation Execution ErrorをCommand Business Resultへ逆流させない
- synthetic Evidenceを実外部Evidenceとして扱わない

この境界は#266で完了し、PR #285で`main`へ統合済みです。

## 最終UI-less Scenario

**構造統合はReady (synthetic)、通常runner実行EvidenceはPending**です。

正本は`internal/driver/final_gate_b_scenario_test.go`です。

```text
Authentication
  ↓ authenticated subject
browse recent_orders
  ↓ same Semantic Order row
record order_record
  ↓
reference Customer
  ↓ same customer_id
header-detail OrderLine[]
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

`customer_id`や`order_id`を別名・物理列・FKから再推測せず、browseでSemantic mapping済みの同じrow identityを後続境界へ引き継ぎます。

record Patternは現行契約どおりSemantic Data解決までとし、未定義のedit runtimeやPersistence動作をtest都合で補いません。

通常runnerでは次を実行します。

```bash
go test ./internal/driver \
  -run '^TestOrderEntryGateBScenarioConnectsMajorBoundariesWithoutUI$' \
  -count=1
```

このScenarioはsynthetic adapterを含むため、成功しても実Oracle / 実PostgreSQL / 実外部Command Evidenceへ格上げしません。

## Reference生成物

**生成責務はReady、実再生成EvidenceはPending**です。

主要な依存は次です。

```text
reference/workspace/
  ├─> reference/world.json
  └─> reference/workspace/.tei/plugin-lock.json
          │
          └─ Plugin Lock fingerprint

reference/workspace/ + Plugin Lock + fixed observed fixture
  └─> reference/order_rollout.durable-run.m0.21.start.json
```

`tools/reference-golden`はWorld / Plugin Lockの一致と、保存Durable RunのPlugin Lock fingerprint staleを検査できます。

更新は正規生成経路だけを利用します。

```bash
go run ./tools/reference-golden
```

意図したauthoring変更に伴う更新時のみ:

```bash
go run ./tools/reference-golden -write
```

Durable Runがstaleになった場合はfingerprintだけを手編集せず、`tei deployment-start`から出力全体を再生成します。

## Gate B条件別監査

| 条件 | 状態 | 根拠 / 残件 |
| --- | --- | --- |
| 同じApplication定義から主要Scenarioを構成できる | Ready (synthetic structure) | 最終UI-less Scenario |
| 同じSemantic Order identityを後続境界へ引き継げる | Ready (synthetic structure) | customer_id / order_idを明示継承 |
| Pattern / Query / Relation / Datasource / Capability / Bindingを追跡できる | Ready | Driver / Source Index / Conformance |
| browseをSemantic resultまで実行できる | Ready | explicit result mapping |
| referenceを実外部readまで実行できる | Blocked | #214 external `master_data` Evidence待ち |
| read-only `master_data`を実効的に保証 | Blocked | #214 |
| header-detailの意味・storage・runtimeが成立 | Ready | Relation Storage / Contract Query runtime |
| header-detail実PostgreSQL Evidence | Pending | 専用DBでtool実行待ち |
| Validation不成立時にCommandへ進まない | Ready | Driver Conformance |
| Authorization denied時にCommandへ進まない | Ready | Driver Conformance |
| Business ResultとExecution Errorを区別 | Ready | ADR-0020 / Driver |
| Command acknowledgementとObservationを分離 | Ready (synthetic) | ADR-0023 / `ExecuteObservationRead` |
| secret / runtime parameterを通常Evidenceへ保存しない | Ready | runtime境界 |
| 最新authoringからReference生成物を再現 | Pending | #255 / `reference-golden`実行待ち |
| 最終Scenarioを通常Go runnerで実証 | Pending | runner実行待ち |
| 第一Web Targetへ同じ意味を渡せる | Partial | A〜DのEvidence完成待ち |

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
- 特定のCIサービスが稼働していること

## 次の実行順

Evidence取得は相互に独立して進められる部分を並走させます。

1. #214でexternal `master_data`の実物理 / read-only Evidenceを得る
2. `tools/reference-postgres-header-detail-runtime`を専用PostgreSQLで実行する
3. #255でReference World / Plugin Lock / Durable Runを正規再生成する
4. 最終UI-less Scenarioを通常Go runnerで実行する
5. A〜DのEvidenceを本監査へ対応付ける
6. secret / read-only / 0件 / Execution Error / Observation期待不一致等のnegative caseを再監査する
7. Gate BをReady / Not Readyで再判定する

GitHub Actionsが別判断で再開された場合は、同じConformanceを継続的なregression検知として再実行します。これは上記1〜7を置き換えるものではありません。

## Gate B Readyの定義

Gate Bは、次をEvidenceから説明できるときReadyとします。

- UIなしで主要Scenarioを一続きに再現できる
- 同じ業務identityを重要境界間で明示的に継承できる
- すべての重要な意味・実現選択へSource / Bindingを辿れる
- 外部資産の未観測事実をTEIが推測していない
- Business Result / Execution Error / Observationを区別できる
- synthetic Evidenceと実外部Evidenceを区別できる
- read-only / secret等の安全境界を保持できる
- 最新authoringからCanonical / Reference artifactを再現できる
- 第一Web Targetへ、Web側で意味を補わず同じbaselineを渡せる

この条件が揃うまでは、第一Web Targetの本格実装着手条件を満たしたとは判定しません。
