# ISSUE-000006: Application Driverの最小vertical sliceを実装する

- 状態: In Progress — 主要境界は成立、Gate B Evidence取得中
- GitHub Issue: #69
- 前段: `ISSUE-000013` / GitHub Issue #124（Gate A Ready）
- 次のGate: Gate B「UIなしで主要Scenarioを再現できる」
- Gate B監査: `../GATE_B_READINESS_AUDIT.md` / `ISSUE-000029` / GitHub Issue #273
- Evidence取得計画: `../GATE_B_EVIDENCE_ACQUISITION.md`
- Evidence実行手順: `../GATE_B_EVIDENCE_RUNBOOK.md`
- Gate B通過後: `ISSUE-000030` / GitHub Issue #291（Gate B → C Handover Baseline）

## 目的

Application Driverはproduction Runtimeの簡易版ではありません。

設計途中のApplicationをUIなしで動かし、Application上の意味、Database / Provider realization、Binding、安全境界の不足や混同を、第一Web Targetへ進む前に露出させる設計検証環境です。

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

Driverは、不足する意味や外部事実を独自規則で補いません。Gapが見つかった場合は、その意味や事実を所有するApplication / Semantic / Database Contract / Realization / Binding / external Evidenceへ戻します。

## 現在地

2026-09-04時点で、Driverの主要な意味境界とruntime境界、最終UI-less Scenario構造は成立しています。

PR #285で、Contract Query、header-detail、Relation Storage、Observation、最終Scenario、Reference authoringを`main`へ収束しました。

その後、Evidence取得の安全境界も段階的に成立しています。

- PR #288 / #300: external-observed Evidence intake
- PR #294 / #300: PostgreSQL fixtureの既存資産非破壊
- PR #306: Invalid Workspaceではexternal Evidence受入・PostgreSQL接続前にfail closed
- PR #309: keyless Relation Storage実装完了とreal PostgreSQL Evidenceの責務分離
- PR #311: Evidence Acquisition / Runbook / Readiness Auditの権威関係を分離

一方、**Gate B全体はNot Ready**です。

残件は新しいDriver抽象の追加ではなく、成立済み境界へのEvidence取得へ収束しています。

## 成立しているvertical slice

### Application Pattern / Canonical resolution

`internal/driver`を正式なDriver入口として、同じ`order_entry` Applicationから次をCanonical identityで解決できます。

- `browse`
- `record`
- `reference`
- `header-detail`
- `command`

この5種類は第一Referenceで成立した候補であり、TEI全体の永久的な標準一覧とはみなしません。

recordは現行契約どおりSemantic Data解決までです。未定義のedit runtimeやPersistence動作をGate B都合で補いません。

### Capability Binding / Authentication / Authorization

Application + Capability identityからBindingを明示解決します。

```text
order_entry / security.authenticate
order_entry / security.authorize
order_entry / sales.order.submit
```

Authentication / Authorization / CommandごとにDriver専用の別実行体系を作りません。

legacy organization / role / return code等はAdapter側へ残し、Driver Coreへ逆流させません。

### browse Query

`order_entry.recent_orders`について、logical Datasourceからnative Queryを実行し、raw resultを明示mappingからSemantic resultへ変換する境界が成立しています。

次からSemantic mappingを推測しません。

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
Oracle database/sql Runner
        ↓
raw result -> explicit Semantic mapping
```

成立済み:

- Semantic Relation key
- typed Database Contract Query
- Reference Contract Query invocation
- Contract Query SQL realization model
- Environment / Database instance / executor Capability composition
- instance-aware runtime Registry
- Oracle向け最小`database/sql` Runner
- 0件とExecution Errorの分離
- realization欠如時にphysical Runnerへ進まない境界

#### external Evidence intake

PR #288 / #300で、actual external assetから得るEvidenceをCanonical Worldへ取り込まず、明示authoring済みSQL realizationと照合する境界が成立しました。

少なくとも次を照合できます。

- target / provider / `external-observed` ownership
- Contract input → physical parameter
- raw result → Semantic attribute
- native source fingerprint
- provenance
- effective read-only mechanism / Evidence参照

PR #306により、Canonical WorkspaceにValidation Errorがある場合はEvidence受入前に停止します。

SQL realizationが存在しない場合も、Evidenceから生成・補完せず停止します。

したがって`ISSUE-000022` / #214は現在、

- Evidence intake boundary: **Ready**
- actual Oracle physical facts / read-only Evidence: **Blocked**

です。

未観測のOracle object / column / SQLをTEI側から推測しません。

### keyless `Order -> OrderLine[]` header-detail

`sales.order.lines`をkeylessなSemantic Relationのまま維持し、Pattern外のApplication固有realizationとDatabase-owned Relation Storageへ分離しています。

```text
sales.order.order_id
        ↓ explicit binding
sales_core.order_line / find_by_order(order_id)
        ↓
Database-owned Relation Storage
        ↓
PostgreSQL SQL realization
        ↓
Driver header-detail runtime
```

成立済み:

- `HeaderDetailContractQueryBinding`
- Relation Storage model / Validation / authoring / Source Index
- Desired Physical Schema / migration
- `sales.order.order_id -> public.order_lines.order_id` support column realization
- `find_by_order` typed Contract Query / PostgreSQL SQL realization
- PostgreSQL Environment composition
- instance-aware runtime execution
- 0件 / 複数件 / Execution Errorの分離
- support columnをOrderLine Semantic attributeへ逆流させない境界

`ISSUE-000024` / #244は設計・実装IssueとしてCompletedです。

#### Provider Evidence safety

`tools/reference-postgres-header-detail-runtime`は実PostgreSQL engine + tool-owned ephemeral fixture Evidence用です。

PR #294 / #300 / #306で次をfail closedにしました。

- Invalid WorkspaceならDB接続前に停止
- 既存`public.order_lines`があれば停止
- 既存tableをDROPして続行しない
- tool自身がCREATEしたfixtureだけをcleanupする
- `evidenceNature = real-postgres-ephemeral-fixture`

実PostgreSQL engineでの成功EvidenceはGitHub Issue #308へ分離しています。

### Validation → Authorization → Command

次を意味上の別結果として区別します。

- Validation `Satisfied` / `NotSatisfied`
- Validation評価不能のExecution Error
- Authorization `allowed` / `denied`
- Command Business Result / Execution Error

Validation不成立、評価不能、Authorization deniedではCommandへ進みません。

最終Scenarioではheader-detailのSemantic OrderLine resultをValidation inputへ渡し、Authenticationで得たsubjectをAuthorizationへ、browseで選択した同じ`order_id`をCommandへ引き継ぎます。

### Command → Observation

ADR-0023に従い、Command Business ResultとObservation readを別操作にします。

```text
Command Business Result / acknowledgement
        ↓ separate operation
explicit read boundary
        ↓
Observation read result / Execution Error
```

production Driverには`ExecuteObservationRead`の最小runtime境界があります。

- Observation対象をCommand payloadや名前から推測しない
- read identity / Evidence nature / resultを保持する
- read成功と期待状態評価を分ける
- Observation Execution ErrorをCommand Business Resultへ畳まない
- runtime parameter値やcredentialを通常Evidenceへ保存しない
- synthetic Evidenceをactual external Evidenceとして扱わない

GitHub Issue #266はこの境界設計として完了しています。

## 最終UI-less Scenario

正本は`internal/driver/final_gate_b_scenario_test.go`です。

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

browseでSemantic mapping済みの同じOrder rowから`customer_id` / `order_id`を後続境界へ引き継ぎます。別名、physical column、Foreign Keyから再推測しません。

このScenarioはsynthetic adapterを含むため、actual Oracle / real PostgreSQL / 実外部Command状態のEvidenceにはしません。

構造は成立していますが、通常Go runnerでの成功EvidenceはGitHub Issue #310で取得します。

## Gate Bの4 Evidence lane

Gate Bを直接止めている実行作業は次の四系統へ分離します。

| Lane | 追跡Issue | Evidence |
| --- | --- | --- |
| A | #214 | actual external Oracle `master_data` physical / read-only Evidence |
| B | #308 | real PostgreSQL engine + tool-owned ephemeral fixture Evidence |
| C | #255 | generated Reference World / Plugin Lock / Durable Run再現Evidence |
| D | #310 | deterministic Repository / normal Go runner Evidence |

### Lane A — #214

actual Oracle object / parameter / raw columns / native source / provenance / effective read-only強制を観測し、actual read → Semantic Customerまで確認します。

Evidence intake implementationがReadyであっても、actual external facts未取得を代替しません。

### Lane B — #308

専用の空PostgreSQL Databaseでheader-detail toolを実行し、同じrunで2件 / 0件 / Execution Errorを確認します。

これはproduction既存Database Evidenceではありません。

### Lane C — #255

Canonical Workspaceから次を正規経路で再生成し、差分を説明します。

- `reference/world.json`
- `reference/workspace/.tei/plugin-lock.json`
- 必要なDurable Run

fingerprintやgenerated JSONを手編集しません。

### Lane D — #310

外部DBを使わず、通常Go runnerで少なくとも次を再現します。

- Canonical Workspace Validation
- `TestOrderEntryGateBScenarioConnectsMajorBoundariesWithoutUI`
- external Evidence intake package / tool tests
- 可能なら`go test ./...`

synthetic Repository Evidenceである範囲を明示し、他laneへ格上げしません。

## Evidence文書の権威関係

PR #311で次へ一本化しました。

- `GATE_B_EVIDENCE_ACQUISITION.md`: 何をEvidenceとして取得するか / 証明範囲 / 完了条件
- `GATE_B_EVIDENCE_RUNBOOK.md`: 具体コマンド / 実行順 / 差分確認 / fail-closed停止条件
- `GATE_B_READINESS_AUDIT.md`: 取得EvidenceからのGate B判定

本Issueへコマンド列を重複正本として持ちません。

## GitHub Actionsの扱い

GitHub Actions workflowは現行Repositoryでは意図的に使用していません。

過去のrunner未割当事象を追跡していた#192は、現行運用では対象外として`not_planned`でcloseしました。

Gate Bでは特定CIサービスのgreenをProduct Evidenceの代替にしません。一方、未実行をgreenとも扱いません。

将来Actionsを再導入する場合は、その時点のworkflow / runner / Repository要件で別のCI検証Issueを作成します。

## Gate B Ready条件

Gate Bはコード量やIssue close数ではなく、Evidenceから少なくとも次を説明できるときReadyとします。

- 同じApplication定義から主要ScenarioをUIなしで一続きに再現できる
- 同じ業務identityを重要境界へ明示的に継承できる
- Pattern / Query / Relation / Contract Query / Datasource / Capability / Bindingを追跡できる
- raw resultとSemantic resultの対応が明示されている
- Relation traversalをphysical FKや名前から推測していない
- external assetの未観測事実をTEIが補完していない
- `master_data`へのwriteをactual Evidenceから実効的に防げる
- Validation / Authorization不成立時にCommandへ進まない
- Business Result / Execution Error / Observationを区別できる
- Query 0件とExecution Errorを区別できる
- synthetic / provider fixture / actual external asset Evidenceを区別できる
- secret / runtime parameter値を通常Evidenceへ残さない
- Driver固有状態やEvidence artifactをCanonical Worldへ逆流させない
- 最新authoringからReference artifactを再現できる
- 第一Web TargetへWeb固有の意味補完なしに同じbaselineを渡せる

## 次の作業

具体的な実行手順は`GATE_B_EVIDENCE_RUNBOOK.md`だけを正とします。

1. #310 deterministic Repository Evidenceを通常Go runnerで再現する
2. #255 generated Reference artifactを正規再生成する
3. #308 real PostgreSQL engine Evidenceを専用空DBで取得する
4. #214 actual Oracle Evidenceを取得する
5. 四laneの結果を`ISSUE-000029` / #273へ対応付ける
6. negative caseを再監査する
7. Gate BをReady / Not Readyで再判定する
8. Readyなら`ISSUE-000030` / #291でそのrevisionをHandover Baselineとして固定する
9. その後に`ISSUE-000007` / #70第一Web Targetへ進む

#214の実外部事実が得られない場合はBlockedのまま保持し、架空情報で穴埋めしません。

## 非目標

- Gate B前の第一Web Target実装
- Web framework選定
- 汎用Workflow Engine / Event Sourcing基盤
- Observation専用Application Patternの先行導入
- Editor / AI実装
- 第二Target
- production HA / Connection Pool一般化
- physical FKの自動生成
- GitHub Actionsの再有効化

## 関連

- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `ISSUE-000022` / GitHub #214
- `ISSUE-000024` / closed GitHub #244
- `ISSUE-000025` / GitHub #255
- `ISSUE-000029` / GitHub #273
- `ISSUE-000030` / GitHub #291
- GitHub #308 / #310
- ADR-0020 / ADR-0021 / ADR-0023
- PR #281 / #285 / #288 / #294 / #300 / #306 / #309 / #311
