# Gate B Readiness Audit

- 判定日: 2026-09-04
- 対象: `order_entry` Application Driver / Gate B
- 判定: **Not Ready**
- Evidence取得計画: [`GATE_B_EVIDENCE_ACQUISITION.md`](GATE_B_EVIDENCE_ACQUISITION.md)
- 再現手順: [`GATE_B_EVIDENCE_RUNBOOK.md`](GATE_B_EVIDENCE_RUNBOOK.md)

## 目的

Gate B「UIなしで主要Scenarioを再現できる」を、IssueやPRのclose数ではなく、現在のReference / Conformance / 実行Evidenceから判定します。

第一Web Targetへ進む前に、Application上の不足をWeb固有コードで補わなくてもよい状態かを確認することが目的です。

この監査では、次を混同しません。

- Semantic / architecture readiness
- syntheticな構造Evidence
- 実provider engine上のfixture Evidence
- external-observed assetの実物理Evidence
- Reference artifactの再現性
- 通常Go runnerによる実行Evidence
- CIによる継続的なregression検知

## 結論

Application Driverの主要な意味境界とScenario構造は成立しています。

PR #285で、Contract Query、header-detail、Relation Storage、Observation、最終UI-less Scenario、Reference authoringを`main`へ収束しました。さらにPR #300で、PR #288 / #294 / #296から次を最新`main`へ追加収束しました。

- external-observed Contract Query Evidenceの受入境界
- PostgreSQL header-detail Evidence toolのfail-closed安全境界
- Gate B Evidence Runbook
- #214のRepository Issue文書

したがって、現在のGate B残件は「Driverの中心設計が未成立」という種類ではなく、**外部事実、実provider、生成物再現、通常runnerのEvidence取得**へ収束しています。

Gate Bを直接止めているEvidenceは、次の四系統です。

1. external-observed `master_data`のactual Oracle物理 / read-only Evidence
2. `Order -> OrderLine[]`の実PostgreSQL engine Evidence
3. Reference World / Plugin Lock / Durable Runの正規再生成Evidence
4. 最終UI-less Scenarioと関連Evidence受入境界の通常Go runner実行Evidence

GitHub Actionsによる継続的な再検証は重要ですが、この四系統のProduct Evidenceと同一ではありません。Actions停止そのものを意味設計Gapへ読み替えず、逆に未実行をgreenとも扱いません。

## Evidenceの性質

Gate Bでは、Evidenceの出所を少なくとも次の三種類に分けます。

### deterministic repository Evidence

Repository checkoutとGo toolchainだけで再現できるEvidenceです。

- Canonical Workspace Validation
- 最終UI-less synthetic Scenario
- external Evidence intakeのunit / tool test
- World / Plugin Lock / Durable Runの再生成

### real-provider ephemeral fixture Evidence

実provider engineを利用しますが、TEIのEvidence tool自身が作る一時fixtureを対象とします。

第一Referenceでは`tools/reference-postgres-header-detail-runtime`が該当します。

これはproduction既存資産の物理事実を観測したEvidenceではありません。

### actual external asset Evidence

TEIが所有していない既存資産から観測した事実です。

第一Referenceでは#214のOracle `master_data`が該当します。synthetic dataやprovider fixtureで代替しません。

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

**意味・runtime・Evidence intakeはReady、actual external readはBlocked**です。

成立済み:

- Semantic Relation key
- Reference Contract Query invocation
- typed Contract Query signature
- Contract Query SQL realization model
- explicit input / result mapping
- Environment / Database instance / executor Capability composition
- instance-aware runtime Registry
- Oracle向け最小`database/sql` Runner
- 0件とExecution Errorの分離

#### external Evidence intake

PR #288で導入し、PR #300で`main`へ収束した`internal/database/externalqueryevidence`と`tools/reference-external-query-evidence`により、非Canonicalな外部Evidenceを明示authoring済みSQL realizationと照合できます。

少なくとも次を検査します。

- Contract Query target identity
- provider
- `external-observed` ownership
- Contract input → physical parameter
- raw result column → Semantic attribute
- native source fingerprint
- physical facts provenance
- effective read-only mechanism / Evidence参照

SQL realizationがWorkspaceに存在しない場合は、Evidenceから生成・補完せず停止します。external Evidenceを`ir.World`へ取り込みません。

このため、#214は次の二層に分かれます。

- **Evidence intake boundary: Ready**
- **actual Oracle physical facts / read-only Evidence: Blocked**

第一Referenceで本当に利用するOracle object、parameter、raw columns、native realization source、provenance、read-only強制Evidenceはまだ未観測です。未観測の`CUSTOMER`や`CUSTOMER_ID`等をTEI側から作りません。

### `Order -> OrderLine[]` header-detail

**意味・保存・runtime・fixture safetyはReady、実provider EvidenceはPending**です。

成立済み:

- keylessなSemantic Relationを維持したtraversal
- Pattern外の`HeaderDetailContractQueryBinding`
- Database-owned Relation Storage
- `sales.order.order_id -> order_id` support column realization
- Desired Physical Schema / migration経路
- `find_by_order` PostgreSQL SQL realization
- PostgreSQL Environment composition
- Contract Query runtime境界
- 0件 / 複数件 / Execution Errorの区別
- 実PostgreSQL engine用Evidence tool

#### PostgreSQL fixture safety

PR #294で安全境界を固定し、PR #300で`main`へ収束しました。

`tools/reference-postgres-header-detail-runtime`は起動時に`public.order_lines`の既存有無を確認し、存在する場合はfail closedで停止します。

- 既存tableをDROPしてfixtureへ置換しない
- 既存tableを削除して続行するoverrideを提供しない
- tool自身がCREATEしたfixtureだけをcleanupする
- Evidenceへ`evidenceNature = real-postgres-ephemeral-fixture`を残す
- DSN / credential / runtime parameter値を通常Evidenceへ保存しない

したがってfixture safety boundaryは**Ready**です。残るのは専用の空PostgreSQL Databaseで実行し、2件、0件、Execution Errorを同じrunで確認することです。

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
Command Business Result / acknowledgement
        ↓ separate operation
explicit read boundary
        ↓
Observation read result / Execution Error
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

通常runnerでは少なくとも次を再現します。

```bash
go test ./internal/driver \
  -run '^TestOrderEntryGateBScenarioConnectsMajorBoundariesWithoutUI$' \
  -count=1

go test ./internal/database/externalqueryevidence \
  ./tools/reference-external-query-evidence
```

これらが成功しても、実Oracle / 実PostgreSQL / 実外部状態のEvidenceへ格上げしません。

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

Durable Runがstaleになった場合はfingerprintだけを手編集せず、`tei deployment-start`から出力全体を再生成します。保存pathへ直接redirectせず、一時ファイルへの生成が成功した後に置換します。

詳細なコマンドと停止条件は`GATE_B_EVIDENCE_RUNBOOK.md`を正とします。

## Gate B条件別監査

| 条件 | 状態 | 根拠 / 残件 |
| --- | --- | --- |
| 同じApplication定義から主要Scenarioを構成できる | Ready (synthetic structure) | 最終UI-less Scenario |
| 同じSemantic Order identityを後続境界へ引き継げる | Ready (synthetic structure) | customer_id / order_idを明示継承 |
| Pattern / Query / Relation / Datasource / Capability / Bindingを追跡できる | Ready | Driver / Source Index / Conformance |
| browseをSemantic resultまで実行できる | Ready | explicit result mapping |
| external Evidence intake境界が成立 | Ready | PR #288 / #300 |
| referenceをactual Oracle readまで実行できる | Blocked | #214 actual external Evidence待ち |
| read-only `master_data`を実効的に保証 | Blocked | #214 actual read-only Evidence待ち |
| header-detailの意味・storage・runtimeが成立 | Ready | Relation Storage / Contract Query runtime |
| PostgreSQL fixture安全境界 | Ready | PR #294 / #300 |
| header-detail実PostgreSQL engine Evidence | Pending | 専用空DBでtool実行待ち |
| Validation不成立時にCommandへ進まない | Ready | Driver Conformance |
| Authorization denied時にCommandへ進まない | Ready | Driver Conformance |
| Business ResultとExecution Errorを区別 | Ready | ADR-0020 / Driver |
| Command acknowledgementとObservationを分離 | Ready (synthetic) | ADR-0023 / `ExecuteObservationRead` |
| secret / runtime parameterを通常Evidenceへ保存しない | Ready | runtime / Evidence境界 |
| 最新authoringからReference生成物を再現 | Pending | #255 / 正規再生成待ち |
| 最終Scenarioを通常Go runnerで実証 | Pending | runner実行待ち |
| external Evidence intakeを通常Go runnerで実証 | Pending | runner実行待ち |
| 第一Web Targetへ同じ意味を渡せる | Partial | 四系統のEvidence完成待ち |

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

具体的なコマンド、安全条件、停止条件は[`GATE_B_EVIDENCE_RUNBOOK.md`](GATE_B_EVIDENCE_RUNBOOK.md)を正とします。

1. Canonical WorkspaceをValidationする
2. 最終UI-less Scenarioとexternal Evidence intakeを通常Go runnerで実行する
3. #255でReference World / Plugin Lock / Durable Runを正規再生成する
4. 専用の空PostgreSQL Databaseでheader-detail Evidence toolを実行する
5. #214のactual Oracle physical facts / provenance / read-only Evidenceを取得する
6. 観測事実を根拠に必要なOracle realizationを明示authoringし、external Evidence intakeで照合する
7. external / provider / golden / runner Evidenceを本監査へ対応付ける
8. secret / read-only / 0件 / Execution Error / Observation期待不一致等のnegative caseを再監査する
9. Gate BをReady / Not Readyで再判定する

#214に必要な実外部事実が得られない場合は、その領域をBlockedとして保持し、未観測情報を埋めて先へ進めません。

GitHub Actionsが別判断で再開された場合は、同じConformanceを継続的なregression検知として再実行します。これは上記手順を置き換えません。

## Gate B Readyの定義

Gate Bは、次をEvidenceから説明できるときReadyとします。

- UIなしで主要Scenarioを一続きに再現できる
- 同じ業務identityを重要境界間で明示的に継承できる
- すべての重要な意味・実現選択へSource / Bindingを辿れる
- 外部資産の未観測事実をTEIが推測していない
- external EvidenceをCanonical meaningや第二の正本へ昇格していない
- Business Result / Execution Error / Observationを区別できる
- synthetic / real-provider fixture / actual external asset Evidenceを区別できる
- read-only / secret等の安全境界を実Evidenceから説明できる
- 最新authoringからCanonical / Reference artifactを再現できる
- 第一Web Targetへ、Web側で意味を補わず同じbaselineを渡せる

Gate B Ready後は[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md) / #291に従い、その時点のrevisionをHandover Baselineとして固定してから第一Web Targetへ進みます。

この条件が揃うまでは、第一Web Targetの本格実装着手条件を満たしたとは判定しません。
