# Gate B Readiness Audit

- 判定日: 2026-09-04
- 対象: `order_entry` Application Driver / Gate B
- 判定: **Not Ready**

## 目的

Gate B「UIなしで主要Scenarioを再現できる」を、個別IssueやPRのclose状態ではなく、現在のReference / Conformance / 実行Evidenceから監査します。

Gate Bの目的は、第一Web Targetへ進む前に、Application上の不足をWeb固有コードで補わずに済む状態を確認することです。

この監査では、実装済み境界、未取得Evidence、外部依存、検証保留を分けて扱います。

## 結論

Application Driverの**意味境界と主要なruntime接続は大きく前進しており、Gate Bの中心設計はほぼ見えています**。

一方、2026-09-04時点では次が残るため、Gate BをReadyとは判定しません。

1. external-observed `master_data`の実物理Evidenceが未確定で、`Order -> Customer`を実readまで通せない
2. `Order -> OrderLine[]`の実PostgreSQL Evidence toolはあるが、実接続成功をまだ取得していない
3. Command acknowledgementと外部状態のObservation境界が未実証
4. Reference World / Plugin Lock等の生成物が最新Canonical Workspaceから再生成・確認されていない
5. 上記を含む最終UI-less Scenarioを一続きのEvidenceとしてまだ固定していない
6. GitHub Actionsは明示停止中であり、通常runnerでの再検証は保留されている

これらは「Application Driverの設計が未成立」という一種類の不足ではありません。**外部事実、実DB Evidence、Observation、生成物再現性、通常runner検証**という異なる残件です。

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

ただし現行ReferenceのCommand adapterは、外部状態変更を確認する実装ではありません。そのためCommand Business Resultだけを「状態変更確認済み」とは扱いません。

## Observationの扱い

Gate BではCommand acknowledgementとObserved stateを分離します。

v1では新しいObservation Patternや汎用Workflow / Event Sourcing基盤を先に作りません。

最初の方針は次です。

```text
Command
  ↓
Business Result / acknowledgement
  ↓ 別操作
明示されたread境界
  ↓
read result
  ↓
Observation Evidence
```

Observation対象をCommand payload、HTTP status、名前の類似等から推測しません。

第一Referenceでは、既存Query / Contract Query / read adapter等の**明示されたread境界をObservation用途として再利用する**ことを優先します。Observationという役割のためだけに同じread意味を別のCanonical正本へ複製しません。

実外部観測先がない場合、syntheticなstateful adapterでCommandとreadを別境界として実証できます。ただし、そのEvidenceは`synthetic`であることを明示し、実外部状態を確認したEvidenceとは扱いません。

詳細判断はADR-0023と#266で扱います。

## Reference生成物

Gate Bでは、authoritative sourceから同じCanonical meaningを再構成できる必要があります。

現在、Reference data access / Relation Storage / Plugin Capability等のauthoring更新後、少なくとも`reference/world.json`が現在のCanonical Workspaceより古いことが確認されています。

したがって#255で次を正規生成経路から再確認します。

- `reference/world.json`
- `reference/workspace/.tei/plugin-lock.json`
- Plugin Lock fingerprintへ依存するDurable Run等

生成物を手編集して合わせません。

## CI / 通常runner

GitHub Actionsは2026-09-03にRepositoryから明示的に停止されています。

CI停止そのものをApplication意味の設計Gapとは扱いません。一方、通常runnerで再検証されていない状態をgreenとも扱いません。

したがってGate B監査では、次を分けます。

- Semantic / architecture readiness
- local / manual / deterministic Evidence readiness
- normal CI verification readiness

CI再開は別の明示判断に従い、この監査から独断で変更しません。

## Gate B条件別監査

| 条件 | 状態 | 根拠 / 残件 |
| --- | --- | --- |
| 同じApplication定義からScenarioを構成できる | Partial | 個別経路は成立。全Scenario統合は未完 |
| Pattern / Query / Relation / Datasource / Capability / Bindingを追跡できる | Ready | Driver / Source Index /各Conformance |
| browseをSemantic resultまで実行できる | Ready | 実PostgreSQL raw result + explicit mapping |
| referenceを実外部readまで実行できる | Blocked | #214 external `master_data`物理Evidence待ち |
| header-detailの意味・storage・runtimeが成立 | Ready | PR #235〜#262 |
| header-detail実PostgreSQL Evidence | Pending | tool実行待ち |
| Validation不成立時にCommandへ進まない | Ready | PR #265 |
| Authorization denied時にCommandへ進まない | Ready | PR #265 |
| Business ResultとExecution Errorを区別 | Ready | ADR-0020 / Driver Conformance |
| Command acknowledgementとObservationを分離 | Blocked | #266 |
| read-only `master_data`を実効的に保証 | Blocked | #214 |
| secret / runtime parameterを通常Evidenceへ保存しない | Ready | 各runtime境界 |
| 最新authoringからReference生成物を再現 | Pending | #255 |
| 最終Scenario Evidenceを一続きに再現 | Blocked | 上記blocker解消後 |
| 通常runnerで再検証済み | Pending | CI再開判断待ち |
| 第一Web Targetへ同じ意味を渡せる | Partial | Gate C契約は定義済み。Gate B baseline完成待ち |

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

## 次の実行順

Gate Bへ最短で進む順序を、実装量ではなくblocker解消効果から次のように整理します。

1. #266でCommandとObservationの最小境界を実証する
2. #214でexternal `master_data`の物理Evidenceを得る
3. `tools/reference-postgres-header-detail-runtime`を実PostgreSQLで実行する
4. #255でReference World / Plugin Lock等を正規再生成する
5. browse -> record -> reference -> header-detail -> Validation -> Authorization -> Command -> Observationを一つのUI-less Scenario Evidenceへ統合する
6. secret / read-only / 0件 / Execution Error等のnegative caseを再監査する
7. CIが別判断で再開された後、通常runnerで関連Conformanceを再実行する
8. 本文書を更新し、Gate BをReady / Not Readyで再判定する

## Gate B Readyの定義

Gate BはIssue close数ではなく、次をEvidenceから説明できるときReadyとします。

- UIなしで主要Scenarioを一続きに再現できる
- すべての重要な意味・実現選択へSource / Bindingを辿れる
- 外部資産の未観測事実をTEIが推測していない
- Business Result / Execution Error / Observationを区別できる
- read-only / secret等の安全境界を保持できる
- 最新authoringからCanonical / Reference artifactを再現できる
- 第一Web Targetへ、Web側で意味を補わず同じbaselineを渡せる

この条件が揃うまでは、第一Web Targetの実装着手条件を満たしたとは判定しません。
