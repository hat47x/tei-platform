# Gate B Readiness Audit

- 判定日: 2026-09-04
- 対象: `order_entry` Application Driver / Gate B
- 判定: **Not Ready**

## 目的

Gate B「UIなしで主要Scenarioを再現できる」を、個別IssueやPRのclose状態ではなく、現在のReference / Conformance / 実行Evidenceから監査します。

Gate Bの目的は、第一Web Targetへ進む前に、Application上の不足をWeb固有コードで補わずに済む状態を確認することです。

この監査では、実装済み境界、synthetic Evidence、未取得の外部Evidence、生成物再現性、通常runner検証を分けて扱います。

## 結論

Application Driverの主要な意味境界とruntime接続は、synthetic Evidence上では一続きに再現できるところまで到達しました。

PR #281では、同じ`order_entry` Application定義と同じbrowse結果のOrder rowを使い、次を一つのUI-less Scenarioとして接続しています。

```text
Authentication
  ↓ authenticated subject
browse
  ↓ 同じSemantic Order row
record Pattern resolution
  ↓
reference Customer
  ↓
header-detail OrderLine[]
  ↓
Canonical Validation
  ↓
Authorization
  ↓
Command Business Result
  ↓ separate operation
explicit Observation read
```

このScenarioでは、browseで得た`order_id`と`customer_id`を別名や型から再推測せず、reference、header-detail、Commandへ明示的に引き継ぎます。recordは現行Driverが所有するSemantic Data解決までとし、未定義のedit runtimeをtest都合で補っていません。

したがって、従来の「主要境界を一つの最終Scenarioへ接続できていない」という不足は、**synthetic Evidenceとしては解消**しました。

一方、2026-09-04時点では次の4系統が残るため、Gate BをReadyとは判定しません。

1. external-observed `master_data`の実物理Evidenceが未確定で、`Order -> Customer`を実readまで通せない
2. `Order -> OrderLine[]`の実PostgreSQL Evidence toolはあるが、実接続成功をまだ取得していない
3. Reference World / Plugin Lock等の生成物が最新Canonical Workspaceから正規再生成・確認されていない
4. GitHub Actionsは明示停止中であり、通常runnerでの再検証が保留されている

これらはApplication Driverの意味設計不足ではなく、**外部事実、実DB Evidence、生成物再現性、通常runner検証**という異なる残件です。

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

### record

現行Driverの境界としてReadyです。

`order_entry.order_record`は`sales.order`を明示的に解決します。現時点ではrecord PatternからPersistenceやedit runtimeを推測しません。

PR #281の最終Scenarioでも、この現在の責務をそのまま利用しています。Gate Bだけのために未定義の編集機構を追加しません。

### `Order -> Customer` reference

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
- synthetic Executorを用いた最終Scenarioへの接続

ただし、external-observed `master_data`の実際のtable / view / parameter / raw column / native SQL / provenanceはRepository上にありません。

未観測のOracle物理情報をTEI側から作らないため、#214をGate Bの外部Evidence blockerとして維持します。

### `Order -> OrderLine[]` header-detail

**意味・保存・runtimeはReady、実DB EvidenceはPending**です。

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
- synthetic Executorを用いた最終Scenarioへの接続

実装と実DB成功Evidenceを混同しません。

`tools/reference-postgres-header-detail-runtime`を実PostgreSQLへ接続し、複数件、0件、Execution Errorを実際に確認する作業が残っています。

### Validation -> Authorization -> Command

Readyです。

第一Referenceでは、Authenticationで得たsubjectを保持し、Canonical業務Validationが成立した場合だけAuthorizationへ進み、`allowed`の場合だけ`sales.order.submit`を実行できます。

次を区別します。

- Validation `NotSatisfied`
- Validation評価不能のExecution Error
- Authorization `denied`
- Command Business Result

PR #281では、header-detailで得たOrderLine値を同じScenarioのValidation入力として利用し、その後にAuthorization / Commandへ進む経路まで接続しています。

### Command acknowledgement -> Observation

**境界はReadyです。ただしEvidenceはsyntheticです。**

CommandとObservation readを別操作として扱います。

```text
sales.order.submit
  ↓
Command Business Result / acknowledgement

--- 別操作 ---

explicit read boundary
  ↓
Observed state
  ↓
Expected stateとの評価
```

次をConformanceで区別します。

- Command実行だけではObservation readを暗黙実行しない
- Command payloadへObserved stateを混入させない
- Observation readは明示されたread boundary identityを持つ
- Observation read成功と期待状態との一致を別に扱える
- Observation Execution ErrorをCommand Business Resultへ逆流させない
- synthetic Evidenceを実外部Evidenceとして扱わない

v1では新しいObservation Pattern、汎用Observation result schema、Workflow Engine、Event Sourcing基盤を追加していません。

詳細判断はADR-0023と#266を正とします。

### 最終UI-less Scenario

**Ready (synthetic)** です。

PR #281で、個別のConformanceを単に並べるのではなく、同じbrowse結果のOrder rowから得たidentityを後続処理へ引き継いでいます。

```text
order_id=1001
customer_id=11
       ↓
record sales.order
       ↓ customer_id
reference Customer
       ↓ order_id
header-detail OrderLine[]
       ↓ OrderLine values
Validation
       ↓ same authenticated subject
Authorization
       ↓ same order_id
Command
       ↓ separate read
Observation
```

Database read、Command side effect、Observation readはsynthetic adapterを用いているため、このEvidenceは実Oracle / 実PostgreSQL / 実外部状態のEvidenceではありません。

その区別を維持したまま、「Application Driverの主要境界を一つのUI-less Scenarioとして接続できる」というGate B上の構造条件を確認しています。

## Reference生成物

Gate Bでは、authoritative sourceから同じCanonical meaningを再構成できる必要があります。

Reference data access / Relation Storage / Plugin Capability等のauthoring更新後、少なくとも`reference/world.json`が現在のCanonical Workspaceより古いことが確認されています。

OrderLineのRelation Storage / SQL realizationについては、重複authoringを整理し、PR #274でPR #251由来の正本authoringを復元済みです。現在の収束線ではRelation Storage / SQL realizationは一組だけです。

ただし、この復元をgolden再生成完了とは扱いません。

#255で次を正規生成経路から再確認します。

- `reference/world.json`
- `reference/workspace/.tei/plugin-lock.json`
- Plugin Lock fingerprintへ依存するDurable Run等

生成物を手編集して合わせません。

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
| 同じApplication定義から主要Scenarioを構成できる | Ready (synthetic) | PR #281 |
| Pattern / Query / Relation / Datasource / Capability / Bindingを追跡できる | Ready | Driver / Source Index / 各Conformance |
| browseをSemantic resultまで実行できる | Ready | 実PostgreSQL raw result + explicit mapping |
| recordを現在のDriver責務で解決できる | Ready | `order_entry.order_record -> sales.order` |
| referenceを実外部readまで実行できる | Blocked | #214 external `master_data`物理Evidence待ち |
| header-detailの意味・storage・runtimeが成立 | Ready | ADR-0021 / Relation Storage / Contract Query runtime |
| header-detail実PostgreSQL Evidence | Pending | tool実行待ち |
| Validation不成立時にCommandへ進まない | Ready | 既存Conformance |
| Authorization denied時にCommandへ進まない | Ready | 既存Conformance |
| Business ResultとExecution Errorを区別 | Ready | ADR-0020 / Driver Conformance |
| Command acknowledgementとObservationを分離 | Ready (synthetic) | ADR-0023 / #266 |
| Observation read結果と期待状態評価を分離 | Ready (synthetic) | test-local Observation Evidence |
| read-only `master_data`を実効的に保証 | Blocked | #214 |
| secret / runtime parameterを通常Evidenceへ保存しない | Ready | 各runtime境界 |
| 最新authoringからReference生成物を再現 | Pending | #255 |
| 最終UI-less Scenarioを一続きに再現 | Ready (synthetic) | PR #281 |
| 通常runnerで再検証済み | Pending | CI再開判断待ち |
| 第一Web Targetへ同じ意味を渡せる | Partial | Gate C契約は定義済み。外部Evidence / golden / runner確認待ち |

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

1. #214でexternal `master_data`の物理Evidenceとread-only Evidenceを得る
2. `tools/reference-postgres-header-detail-runtime`を実PostgreSQLで実行する
3. #255でReference World / Plugin Lock等を正規再生成する
4. external read / 実DB /生成物を含めたnegative caseを再監査する
5. CIが別判断で再開された後、通常runnerで関連Conformanceを再実行する
6. 本文書を更新し、Gate BをReady / Not Readyで再判定する

最終UI-less Scenarioを構造上接続する作業や、Observation境界のためだけに新しいproduction schemaを追加する作業は、残blockerには含めません。

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

synthetic Scenarioだけではこの定義を満たしません。#214、実PostgreSQL、#255、通常runner再検証が揃うまではGate BをNot Readyとして維持します。
