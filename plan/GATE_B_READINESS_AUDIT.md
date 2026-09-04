# Gate B Readiness Audit

- 判定日: 2026-09-04
- 対象: `order_entry` Application Driver / Gate B
- 判定: **Not Ready**
- Evidence取得計画: [`GATE_B_EVIDENCE_ACQUISITION.md`](GATE_B_EVIDENCE_ACQUISITION.md)
- 再現手順: [`GATE_B_EVIDENCE_RUNBOOK.md`](GATE_B_EVIDENCE_RUNBOOK.md)
- Repository Issue: [`issue/000029-gate-b-readiness-audit.md`](issue/000029-gate-b-readiness-audit.md) / GitHub #273

## 目的

Gate B「UIなしで主要Scenarioを再現できる」を、IssueやPRのclose数ではなく、現在のReference / Conformance / 実行Evidenceから判定します。

第一Web Targetへ進む前に、Application上の意味不足をWeb固有コードで補わなくてもよい状態かを確認することが目的です。

この監査では、次を混同しません。

- Semantic / architecture readiness
- syntheticな構造Evidence
- deterministic Repository / normal runner Evidence
- generated artifact reproducibility Evidence
- real-provider ephemeral fixture Evidence
- actual external asset Evidence
- CIによる継続的なregression検知

## 結論

Application Driverの主要な意味境界と最終UI-less Scenario構造は成立しています。

現在のGate B残件は「Driverの中心設計が未成立」という種類ではなく、**actual external facts、実provider、generated artifact再現、通常Go runnerによるEvidence取得**へ収束しています。

Gate Bを直接止めているのは、次の4 Evidence laneです。

| Lane | Repository / GitHub Issue | Evidence nature | 現在地 |
| --- | --- | --- | --- |
| A | `ISSUE-000022` / #214 | actual external asset | **Blocked** — actual Oracle facts待ち |
| B | `ISSUE-000031` / #308 | real-provider ephemeral fixture | **Ready to run** — Evidence専用空PostgreSQL DB待ち |
| C | `ISSUE-000025` / #255 | generated artifact reproducibility | **Ready to check** — clean normal Go environment待ち |
| D | `ISSUE-000032` / #310 | deterministic repository | **Ready to run** — clean checkout / normal Go runner待ち |

4 laneは相互代替しません。

GitHub Actionsによる継続的な再検証は重要ですが、この4 laneのProduct Evidenceと同一ではありません。Actions停止そのものを意味設計Gapへ読み替えず、逆に未実行をgreenとも扱いません。

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

GitHub #179 / #185とRepository `ISSUE-000018 / 000019`は実装責務として完了しています。actual Oracle Evidence待ちを理由に、成立済み実装Issueを再びOpen / In Progressへ戻しません。

#### external Evidence intake

非Canonicalな外部Evidenceを、明示authoring済みSQL realizationと照合する境界はReadyです。

少なくとも次を検査します。

- Contract Query target identity
- provider
- `external-observed` ownership
- Contract input → physical parameter
- raw result column → Semantic attribute
- native source fingerprint
- physical facts provenance
- effective read-only mechanism / Evidence参照
- Canonical Workspace Validation

SQL realizationがWorkspaceに存在しない場合は、Evidenceから生成・補完せず停止します。external Evidenceを`ir.World`へ取り込みません。

したがって`ISSUE-000022` / #214は、

- Evidence intake boundary: **Ready**
- actual Oracle physical facts / read-only / driver integration / actual read: **Blocked**

です。

未観測の`CUSTOMER`、`CUSTOMER_ID`、Oracle SQL等をTEI側から作りません。

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
- real PostgreSQL engine用Evidence tool

`ISSUE-000024` / #244は設計・実装IssueとしてCompletedです。実provider Evidenceだけを`ISSUE-000031` / #308へ分離しています。

#### PostgreSQL Evidence safety

PR #294 / #300 / #306 / #324でfail-closed境界を段階的に強化しました。

`tools/reference-postgres-header-detail-runtime`はfixture作成前に次を確認します。

1. Canonical WorkspaceをLoad / Validateする
2. Validation ErrorならDB接続前に停止する
3. PostgreSQLへ接続する
4. `current_database()`を確認する
5. `postgres` / `template0` / `template1`をfixture先として拒否する
6. system / temporary領域を除くuser relation件数を確認する
7. user relationが1件でもあれば停止する
8. `public.order_lines`が既に存在しないことを再確認する
9. すべて成立した場合だけtool-owned fixtureを作成する
10. tool自身がCREATEしたfixtureだけをcleanupする

既存relationを削除してEvidence用DBへ転用する処理や、既定Databaseで続行するoverrideはありません。

このguardは「Evidence専用DBである」という運用意図そのものを証明するものではありません。実行先は引き続き、Evidence専用に作成した空PostgreSQL Databaseへ限定します。

残る#308 Evidenceは、同じrunで次を確認することです。

- `order_id=1001` → 2件
- `order_id=9999` → 正常な0件
- tool-owned fixture欠落 → Execution Error
- Environment / Database instance / executor Capability / Contract Query target identity
- explicit Semantic mapping
- support column非露出
- runtime parameter値 / credential非保存
- `evidenceNature = real-postgres-ephemeral-fixture`

### Validation → Authorization → Command

**Ready**です。

次を別結果として区別します。

- Validation `Satisfied` / `NotSatisfied`
- Validation評価不能のExecution Error
- Authorization `allowed` / `denied`
- Command Business Result / Execution Error

Validation不成立、評価不能、Authorization deniedではCommandへ進みません。

### Command acknowledgement → Observation

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
- synthetic Evidenceをactual external Evidenceとして扱わない

この境界は#266で完了しています。

## 最終UI-less Scenario

**構造統合はReady (synthetic)、normal runner EvidenceはPending**です。

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

この構造があることと、現在のclean revisionで通常Go runnerから再現したことは別です。後者はLane D / #310で取得します。

## Lane D — deterministic Repository / #310

正規入口は次です。

```bash
go run ./tools/reference-gate-b-repository-evidence
```

runnerは少なくとも次を一つのrunで扱います。

1. 実行前repository revision取得
2. 実行前working tree clean確認
3. Go toolchain identity取得
4. Canonical Workspace Validation
5. 最終UI-less Scenario test
6. external Evidence intake test
7. `-broad`指定時だけ`go test ./...`
8. 実行後HEAD / working tree再確認
9. 小さい非Canonical JSON Evidence出力

PR #321以降、`reproducibleRevision=true`には次のすべてを要求します。

- 実行前clean
- 実行後確認済み
- 実行前後でHEAD同一
- 実行後clean

Test自体が成功しても、実行中にsource revisionが変化したrunや実行後にtracked / untracked差分が残ったrunはGate B完了Evidenceへ採用しません。

診断用`-allow-dirty`もGate B完了Evidenceには使用しません。

## Lane C — generated Reference artifact / #255

**生成・照合toolはReady、clean revisionでの実EvidenceはPending**です。

主要な依存:

```text
reference/workspace/
  ├─> reference/world.json
  └─> reference/workspace/.tei/plugin-lock.json
          │
          ├─> Plugin Lock fingerprint
          └─> Runtime Policy input

reference/workspace/ + Plugin Lock + fixed observed deployment state
  └─> reference/order_rollout.durable-run.m0.21.start.json
```

正規toolは次へ分離しています。

### `tools/reference-golden`

- Worldのcheck / write
- Plugin Lockのcheck / write
- 保存Durable RunのPlugin Lock依存チェック

### `tools/reference-durable-run-golden`

Durable Run全体を、`deployment-start`と同じinternal API経路から正規生成して保存goldenとbyte-for-byteで比較します。

内部では少なくとも次を通ります。

```text
Workspace Validation
 -> exact Solution Release resolution
 -> fixed observed deployment state Validation
 -> releaseplan.CompileWithDeployment
 -> pluginruntime.BuildPlan
 -> Plugin Lock / Runtime Policy fingerprint
 -> deploymentdurable.StartWithRuntimePolicy
 -> saved Durable Runとの完全一致check
```

write時は既存保存fileを先にtruncateせず、同じdirectoryの一時fileへ全JSONを書き、Sync / Close後にRenameします。

PR #326で`-solution`のCanonical IDも`ParseCanonicalID`で検証し、不正入力をpanicではなく通常errorとしてfail closedするようにしました。

### Evidenceと修復を分離する

`-write`はstaleなgenerated artifactを正規状態へ戻す**修復操作**です。`-write`直後のdirty treeでcheckが成功しても、Gate B完了Evidenceにはしません。

staleだった場合は次の順で扱います。

```text
stale検出
  ↓
正規toolの-writeで修復
  ↓
git diffで意味上の差分を確認
  ↓
意図したgenerated artifactだけをcommit
  ↓
新しいclean revision
  ↓
check-onlyを再実行
  ↓
実行前後で同じHEAD / clean working treeを確認
  ↓
#255完了Evidence候補
```

最初から両goldenが一致している場合は、形式的な`-write`を行いません。

最終Evidence候補には次をすべて要求します。

- final check開始時にworking tree clean
- `reference-golden`一致
- `reference-durable-run-golden`一致
- final check実行前後でHEAD同一
- final check終了後もworking tree clean

2026-09-04の静的監査では、保存済みproduction Plugin Lockから計算したEnvironment fingerprintと保存Durable Runの`pluginLockFingerprint`は一致していました。ただし、これは現在のCanonical Workspaceから両artifactを正規再現できることの証明ではありません。

## Gate B条件別監査

| 条件 | 状態 | 根拠 / 残件 |
| --- | --- | --- |
| 同じApplication定義から主要Scenarioを構成できる | Ready (synthetic structure) | 最終UI-less Scenario |
| 同じSemantic Order identityを後続境界へ引き継げる | Ready (synthetic structure) | customer_id / order_idを明示継承 |
| Pattern / Query / Relation / Datasource / Capability / Bindingを追跡できる | Ready | Driver / Source Index / Conformance |
| browseをSemantic resultまで実行できる | Ready | explicit result mapping |
| external Evidence intake境界が成立 | Ready | #214 intake boundary |
| referenceをactual Oracle readまで実行できる | Blocked | #214 actual external Evidence待ち |
| read-only `master_data`を実効的に保証 | Blocked | #214 actual read-only Evidence待ち |
| header-detailの意味・storage・runtimeが成立 | Ready | #244 Completed |
| PostgreSQL fixture安全境界 | Ready | PR #294 / #300 / #306 / #324 |
| header-detail実PostgreSQL engine Evidence | Pending | #308専用空DBで実run待ち |
| Validation不成立時にCommandへ進まない | Ready | Driver Conformance |
| Authorization denied時にCommandへ進まない | Ready | Driver Conformance |
| Business ResultとExecution Errorを区別 | Ready | Driver / ADR-0020 |
| Command acknowledgementとObservationを分離 | Ready (synthetic) | ADR-0023 / `ExecuteObservationRead` |
| secret / runtime parameterを通常Evidenceへ保存しない | Ready | runtime / Evidence境界 |
| generated artifact照合toolがある | Ready | `reference-golden` / `reference-durable-run-golden` |
| 最新authoringからgenerated artifactをclean revisionで再現 | Pending | #255 final check-only待ち |
| 最終Scenario / Evidence intakeをnormal runnerで実証 | Pending | #310実run待ち |
| 第一Web Targetへ同じ意味を渡せる | Partial | 4 lane完成 + #291 Handover待ち |

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

具体的なコマンド、安全条件、停止条件は[`GATE_B_EVIDENCE_RUNBOOK.md`](GATE_B_EVIDENCE_RUNBOOK.md)を唯一の正本とします。

1. #310をcleanな通常Go runnerで実行する
2. #255をclean revisionでcheck-onlyし、staleなら正規toolで修復 → 差分確認 → commit → 新しいclean revisionでcheck-onlyを再実行する
3. #308をEvidence専用の空PostgreSQL Databaseで実行する
4. #214のactual Oracle physical facts / provenance / read-only Evidenceを取得する
5. 観測事実を根拠に必要なOracle realizationを明示authoringし、external Evidence intakeで照合する
6. 4 laneのEvidenceを本監査へ対応付ける
7. secret / read-only / 0件 / Execution Error / Observation期待不一致 / source revision不変等のnegative caseを再監査する
8. Gate BをReady / Not Readyで再判定する
9. Readyなら#291でそのrevisionをHandover Baselineとして固定する
10. その後に第一Web Target #70 / Gate C Evidence #270へ進む

#214に必要な実外部事実が得られない場合は、その領域をBlockedとして保持し、未観測情報を埋めて先へ進めません。

## GitHub Actionsとの関係

GitHub Actions workflowは現行Repositoryでは意図的に使用していません。

過去のrunner未割当事象を追跡した#192は、現行運用では対象外として`not planned`でclosedです。

Actionsを将来再導入する場合は、その時点のworkflow / runner / Repository要件で新しいCI検証Issueを作成します。CIサービスのgreenをGate B Product Evidenceの代替にはしません。また、workflow failureがpushそのものを不必要にblockしない運用を維持します。

## Gate B Readyの定義

Gate Bは、次をEvidenceから説明できるときReadyとします。

- UIなしで主要Scenarioを一続きに再現できる
- 同じ業務identityを重要境界間で明示的に継承できる
- すべての重要な意味・実現選択へSource / Bindingを辿れる
- 外部資産の未観測事実をTEIが推測していない
- external EvidenceをCanonical meaningや第二の正本へ昇格していない
- Business Result / Execution Error / Observationを区別できる
- deterministic / generated artifact / real-provider fixture / actual external asset Evidenceを区別できる
- read-only / secret等の安全境界を実Evidenceから説明できる
- generated artifactをcleanなsource revisionから再現できる
- Evidence runが参照するsource revisionを実行前後で固定できる
- 第一Web Targetへ、Web側で意味を補わず同じbaselineを渡せる

Gate B Ready後は[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md) / #291に従い、その時点のrevisionをHandover Baselineとして固定してから第一Web Targetへ進みます。

この条件が揃うまでは、第一Web Targetの本格実装着手条件を満たしたとは判定しません。
