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

現在のGate B残件は「Driverの中心設計が未成立」という種類ではなく、**通常Go runner、generated artifact再現、real PostgreSQL Providerの実Evidence取得**へ収束しています。

PR #379を起点とした`ISSUE-000035` / #381の再監査により、actual Oracle固有EvidenceはGate Bの必須完了条件から外しました。

Gate Bを直接止めている必須Evidenceは次の3本です。

| 必須Evidence | Repository / GitHub Issue | Evidence nature | 現在地 |
| --- | --- | --- | --- |
| #310 | `ISSUE-000032` / #310 | deterministic repository | **Ready to run** — clean checkout / normal Go runner待ち |
| #255 | `ISSUE-000025` / #255 | generated artifact reproducibility | **Ready to check** — clean normal Go environment待ち |
| #308 | `ISSUE-000031` / #308 | real-provider ephemeral fixture | **Ready to run** — Evidence専用空PostgreSQL DB待ち |

actual external Oracleは別の継続Evidence trackとして保持します。

| 継続Evidence | Repository / GitHub Issue | 現在地 |
| --- | --- | --- |
| actual Oracle asset | `ISSUE-000022` / #214 | **Blocked / unresolved可** — actual physical facts / actual read待ち |
| driver / Host assembly | `ISSUE-000033` / #371 | **Blocked / unresolved可** — actual Oracle環境条件待ち |
| read-only Evidence | `ISSUE-000034` / #376 | **Blocked / unresolved可** — actual principal / grant等待ち |

#214 / #371 / #376は不要になったわけではありません。actual Oracle compatibility / external `master_data` integrationを実証済みと主張するために必要です。ただし、それらの外部環境情報が得られないことだけで第一Web TargetへのProduct学習を止めません。

GitHub Actionsによる継続的な再検証は重要ですが、このProduct Evidenceと同一ではありません。Actions停止そのものを意味設計Gapへ読み替えず、逆に未実行をgreenとも扱いません。

## Gate B candidate Ready revision

各必須Evidenceが個別run内で`repositoryRevision`を固定していても、異なるrevisionのEvidenceを寄せ集めた状態ではGate B Readyと判定しません。

最終監査では、一つのclean Repository revisionを**candidate Ready revision**として明示します。

candidateを`R`とする場合、少なくとも次をRへ揃えます。

| 必須Evidence | Rへ固定するもの |
| --- | --- |
| #310 | deterministic Repository Evidence JSONの`repositoryRevision` |
| #255 | generated artifact Evidence JSONの`repositoryRevision` |
| #308 | real PostgreSQL Evidence JSONの`repositoryRevision` |

Repository revisionを持つEvidenceでは、`repositoryRevision = R`かつ`reproducibleRevision = true`を要求します。

次のような混在はReady Evidenceとして採用しません。

```text
#310 -> R1
#255 -> R2
#308 -> R3
```

### #214 external factsの扱い

Oracleのphysical object、raw columns、provenance、read-only権限等の外部観測事実そのものはGit revisionに属しません。

#214がGate B判定時点で完了している場合は、Handover Baseline Rへactual external Evidence / acceptance receipt / actual readを追跡可能にします。

未完了の場合は、Handoverへ次をunresolvedとして明示します。

- #214 actual external Oracle Evidence: not verified
- #371 actual Oracle driver / Host assembly: 未確定または未検証
- #376 actual Oracle read-only Evidence: 未取得
- Oracle compatibility / actual `master_data` integrationを実証済みとは主張しない
- Web側でOracle固有物理情報やread-only方式を補わない

#310 synthetic Scenarioや#308 PostgreSQL Evidenceをactual Oracle成功へ読み替えません。

### candidateが変わる場合

candidate RのEvidence取得中にsource修正が必要になった場合、Rを暗黙に新revisionへ読み替えません。

1. 必要なsource / generated artifactを修正する
2. commitする
3. 新candidate R'を選ぶ
4. #310 / #255 / #308をR'で再実行する
5. R'でGate B Readyを再監査する

#255のrepair commitが生じた場合も同じです。repair前revisionで取得したsource-dependent Evidenceをrepair後candidateへ混ぜません。

#214 / #371 / #376の作業が後からsource / Driver meaningを変更し、その変更をGate Cへ取り込む場合は、影響するGate B Evidenceを再確認し、新しいHandover Baselineを発行します。

### mainの進行とcandidateを分離する

candidateを選んだ後に並行作業で`main`が進んでも、candidateを自動的に最新mainへ動かしません。

- candidate RをそのままGate B Ready監査するなら、RとRのEvidenceを固定して扱う
- 後続commitをGate B baselineへ含めたいなら、新candidateを選び直してsource-dependent Evidenceを揃える

Gate B Readyと判定したcandidate Rは、そのまま#291 Handover Baselineのrepository revisionになります。

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

## Customer Reference / external Oracle

**意味・runtime・Evidence intakeはReady、actual external Oracleはunresolved / not verified**です。

成立済み:

- Semantic Relation key
- Reference Contract Query invocation
- typed Contract Query signature
- Contract Query SQL realization model
- explicit input / result mapping
- Environment / Database instance / executor Capability composition
- instance-aware runtime Registry
- Oracle向け最小`database/sql` Runner
- actual read時のraw result列名を保持するOracle固有Payload（PR #364）
- driverが提供できたraw列metadataを未提供値の推測なしで保持する境界（PR #366）
- actual Oracle driver / Host assemblyを`ISSUE-000033` / #371へ分離した管理境界（PR #372）
- actual Oracle read-only Evidenceを`ISSUE-000034` / #376へ分離した管理境界（PR #378）
- 0件とExecution Errorの分離

GitHub #179 / #185とRepository `ISSUE-000018 / 000019`は実装責務として完了しています。actual Oracle Evidence待ちを理由に成立済み実装Issueを再びOpen / In Progressへ戻しません。

#371 / #376を作成したことは責務分離の完了であって、driver integrationやread-only Evidenceの完了ではありません。

### external Evidence intake

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

このintake boundaryのConformanceは#310 deterministic Repository Evidenceの必須checkに含めます。

### source-fixed acceptance receipt — PR #340

PR #340で、external Evidenceを「何となく受理済み」と扱わず、**どのEvidence artifactを、どのclean Repository revisionで照合したか**を機械可読receiptへ固定しました。

実際のexternal Evidence artifactが得られた場合は、`evidenceArtifactFingerprint`とsource-fixed receiptを利用します。

acceptance receiptはactual Oracle read Evidenceではありません。

### actual read時のdriver観測 — PR #364 / #366

Oracle Runner固有Payloadはactual read時に次を保持できます。

- named parameterの名前
- 実際に返されたraw result列名
- driverが提供した場合の`databaseTypeName`
- driverが提供した場合の`nullable`
- driverが提供した場合の`length`
- driverが提供した場合の`precision` / `scale`

未提供metadataをfalseや0へ推測しません。Go側の`ScanType`やraw row値を通常Evidenceへ複製しません。

このruntime metadataは、provenance付きexternal Evidenceの`PhysicalType` / `Nullable`を置き換える第二の物理正本ではありません。

### actual Oracle driver / Host assembly — #371

具体driverの選択・登録、DSN / credential、TLS / TCPS、Wallet等のHost設定はProvider CoreやCanonical modelではなくHost / Reference assembly側で確定します。

実Oracle version / authentication / encryption / datatype / NULL / Host制約が未観測のため、採用driverやdependencyは未確定です。

### read-only Evidence — #376

`SELECT`に見えるSQL、`QueryContext`、read-only transaction等の一層だけでDatabase credentialのwrite不可を証明したとは扱いません。

少なくとも、専用least-privilege principal / role、grant Evidence、必要に応じたread-only transaction、TEI Reference read pathのquery-only境界、actual Query成功を組み合わせて確認します。

read-only確認のためだけに実external assetへDML / DDL、`SELECT ... FOR UPDATE`、exclusive lock等を試行しません。

### Gate Bでの判定

#214 / #371 / #376が未完了であることだけではGate BをNot Readyにしません。

ただし次を必須とします。

- #310でexternal Evidence intake fail-closed boundaryが通常runnerから再現する
- 未観測Oracle factsをTEI側で補完していない
- actual Oracleを実証済みと主張しない
- Handoverへ#214 / #371 / #376をunresolvedとして明示する
- Web側でOracle固有情報を補わない

## `Order -> OrderLine[]` / real PostgreSQL engine / #308

**意味・保存・runtime・fixture safety・source safetyはReady、実provider EvidenceはPending**です。

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

### source revision境界 — PR #336

PR #336で#310 / #255 / #308のRepository source観測を`tools/internal/repositoryevidence`へ収束しました。

#308ではDB接続前にclean sourceを要求し、Provider runtime / fixture cleanup / Database復元確認後にもHEAD / working treeを再確認します。

### fixture前安全性 — PR #324 / #335 / #365

`tools/reference-postgres-header-detail-runtime`はfixture作成前に次を確認します。

1. Canonical WorkspaceをLoad / Validateする
2. Validation ErrorならDB接続前に停止する
3. PostgreSQLへ接続する
4. `current_database()`を確認する
5. `postgres` / `template0` / `template1`を拒否する
6. system / temporary領域を除くuser relation件数を確認する
7. user relationが1件でもあれば停止する
8. Event Trigger件数を確認する
9. Event Triggerが1件でもあれば停止する
10. `public.order_lines`が存在しないことを再確認する
11. すべて成立した場合だけtool-owned fixtureを作成する

既存relationやEvent Triggerを削除・無効化してEvidence用DBへ転用する処理や、既定Databaseで続行するoverrideはありません。

PR #335 / #365以降、fixture前の非機密な安全判定を`fixtureDatabaseSafety`としてJSONへ残します。

```json
{
  "guardPassed": true,
  "databaseIdentityVerified": true,
  "databaseIdentityPersisted": false,
  "nonDefaultDatabaseVerified": true,
  "userRelationsBeforeFixture": 0,
  "eventTriggersBeforeFixture": 0,
  "fixtureObjectAbsentBeforeCreate": true
}
```

### Provider runtime意味

同一runで確認する対象:

- `order_id=1001` → 2件
- `order_id=9999` → 正常な0件
- tool-owned fixture欠落 → Execution Error
- Environment / Database instance / executor Capability / Contract Query target identity
- explicit Semantic mapping
- support column非露出
- runtime parameter値 / credential / DSN非保存
- `evidenceNature = real-postgres-ephemeral-fixture`

### fixture cleanup後の空状態 — PR #341 / #365

期待値:

```json
{
  "verified": true,
  "userRelationsAfterCleanup": 0,
  "eventTriggersAfterCleanup": 0,
  "fixtureObjectAbsentAfterCleanup": true
}
```

終了時にuser relationまたはEvent Triggerが存在する場合、それがtool由来か他主体の並行変更かを推測せずfail closedします。

#308はactual OracleをGate B必須から外しても**必須Evidenceとして維持**します。少なくとも一つのreal provider engineでDatabase Binding / native SQL / explicit mapping / failure boundaryを確認するためです。

## Validation → Authorization → Command

**Ready**です。

次を別結果として区別します。

- Validation `Satisfied` / `NotSatisfied`
- Validation評価不能のExecution Error
- Authorization `allowed` / `denied`
- Command Business Result / Execution Error

Validation不成立、評価不能、Authorization deniedではCommandへ進みません。

## Command acknowledgement → Observation

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

Customer referenceはsynthetic / deterministicにScenarioへ含まれます。これはactual Oracle Evidenceではありませんが、Gate Bの目的であるApplication / Driver meaningの再現性を確認します。

## #310 deterministic Repository

正規入口:

```bash
go run ./tools/reference-gate-b-repository-evidence
```

PR #346以降、targeted runnerは少なくとも次の三つを独立checkとして一つのrunで扱います。

1. 最終UI-less Scenario
2. external Evidence intake
3. Repository source境界Conformance

runner全体ではさらに、実行前後repository revision、working tree clean、Go toolchain identity、Canonical Workspace Validation等を扱います。

PR #321 / #336 / #351以降、`reproducibleRevision=true`は開始時cleanだけでは設定しません。post-run確認後に、開始revision非空、実行前clean、post-run確認済み、revision文字列同一、`revisionStable=true`、実行後cleanをすべて要求します。

通常Go runner上の実runはまだ未取得です。

## #255 generated Reference artifact

**生成・照合toolはReady、clean revisionでの実EvidenceはPending**です。

最終Evidence入口:

```bash
go run ./tools/reference-generated-artifact-evidence
```

runnerは`-write`を呼ばず、`reference-golden` / `reference-durable-run-golden`をcheck-onlyで実行し、実行前後のRepository状態も確認します。

PR #367のsafe replaceも修復途中の破損リスクを下げる境界であり、`-write`やsafe replace成功だけをGate B完了Evidenceにはしません。

## Gate B条件別監査

| 条件 | 状態 | 根拠 / 残件 |
| --- | --- | --- |
| 同じApplication定義から主要Scenarioを構成できる | Ready (synthetic structure) | 最終UI-less Scenario |
| 同じSemantic Order identityを後続境界へ引き継げる | Ready (synthetic structure) | customer_id / order_idを明示継承 |
| Pattern / Query / Relation / Datasource / Capability / Bindingを追跡できる | Ready | Driver / Source Index / Conformance |
| browseをSemantic resultまで実行できる | Ready | explicit result mapping |
| external Evidence intake境界が成立 | Ready | #214 intake boundary |
| external Evidence intakeをnormal runnerで再現 | Pending | #310実run待ち |
| Oracle actual readのraw列名 / driver提供metadataを保持できる | Ready (runtime boundary) | PR #364 / #366。actual read自体はunresolved |
| actual Oracle compatibility / `master_data` integration | Unresolved / non-blocking for Gate B | #214 / #371 / #376。実環境情報待ち |
| header-detailの意味・storage・runtimeが成立 | Ready | #244 Completed |
| PostgreSQL fixture前安全境界 | Ready | PR #294 / #300 / #306 / #324 / #335 / #365 |
| PostgreSQL source revision固定 | Ready | PR #336 |
| PostgreSQL cleanup後空状態再確認 | Ready | PR #341 / #365 |
| header-detail実PostgreSQL engine Evidence | Pending | #308専用空DBで実run待ち |
| Validation不成立時にCommandへ進まない | Ready | Driver Conformance |
| Authorization denied時にCommandへ進まない | Ready | Driver Conformance |
| Business ResultとExecution Errorを区別 | Ready | Driver / ADR-0020 |
| Command acknowledgementとObservationを分離 | Ready (synthetic) | ADR-0023 / `ExecuteObservationRead` |
| Repository source境界のtargeted Conformanceがある | Ready | PR #346 |
| generated artifact final check-only runnerがある | Ready | PR #331 / #334 |
| 最新authoringからgenerated artifactをclean revisionで再現 | Pending | #255 actual check-only run待ち |
| 最終Scenario / Evidence intake / source境界をnormal runnerで実証 | Pending | #310実run待ち |
| 3必須Evidenceを一つのcandidate revisionへ揃える | Pending | #310 / #255 / #308実Evidence取得後 |
| 第一Web Targetへ同じ意味を渡せる | Partial | 3 Evidence整合 + #291 Handover待ち |

## Gate Bを止めないもの

次はGate Bの前提にしません。

- actual Oracle実環境が利用可能になること
- #214 / #371 / #376の完了
- Web frameworkの選定
- 第一Web Targetの実装
- 完全なTarget-neutral Presentation IR
- Wails採否
- Editor / AI実装
- 第二Target
- 汎用Workflow Engine / Event Sourcing基盤
- production HA / Connection Pool一般化
- 特定のCIサービスが稼働していること

ただし#214未完了を「Oracle対応済み」と読み替えないことはGate B / Handoverの必須条件です。

## 次の実行順

具体的なコマンド、安全条件、停止条件は[`GATE_B_EVIDENCE_RUNBOOK.md`](GATE_B_EVIDENCE_RUNBOOK.md)を唯一の正本とします。

### 準備段階

1. #310をcleanな通常Go runnerで事前実行し、不足を露出する
2. #255をcheckし、staleなら正規toolで修復 → 差分確認 → commitする
3. #308をEvidence専用の空PostgreSQL Databaseで事前実行する
4. Gate Bに必要なsource / generated artifact変更をすべてcommitする
5. working treeをcleanにし、candidate Rを記録する

#214 / #371 / #376はactual external Oracle環境情報が得られた時点で並行継続できます。Gate Bを通すためだけに未観測情報を仮定しません。

### candidate Rでの最終Evidence cycle

6. Rで#310 `reference-gate-b-repository-evidence`を実行する
7. Rで#255 `reference-generated-artifact-evidence`をcheck-only実行する
8. Rで#308 real PostgreSQL Evidenceを実行する
9. #310 / #255 / #308の`repositoryRevision = R`と`reproducibleRevision = true`を確認する
10. secret / 0件 / Execution Error / Observation期待不一致 / source revision不変 / PostgreSQL fixture前後安全性 / external Evidence非推測等のnegative caseを再監査する
11. #214 actual Oracleのverified / unresolved状態を明示する
12. Gate BをRについてReady / Not Readyで再判定する
13. Readyなら#291で**同じR**をHandover Baselineとして固定し、#214 unresolvedなら#214 / #371 / #376を明示する
14. その後に第一Web Target #70 / Gate C Evidence #270へ進む

candidate Rで必須Evidenceがsource修正を要求した場合、Rを修正後revisionへ読み替えません。修正をcommitし、新candidate R'を選んで#310 / #255 / #308をR'へ揃え直します。

## GitHub Actionsとの関係

GitHub Actions workflowは現行Repositoryでは意図的に使用していません。

過去のrunner未割当事象を追跡した#192は、現行運用では対象外として`not_planned`でclosedです。

Actionsを将来再導入する場合は、その時点のworkflow / runner / Repository要件で新しいCI検証Issueを作成します。CIサービスのgreenをGate B Product Evidenceの代替にはしません。また、workflow failureがpushそのものを不必要にblockしない運用を維持します。

## Gate B Readyの定義

Gate Bは、次をEvidenceから説明できるときReadyとします。

- UIなしで主要Scenarioを一続きに再現できる
- 同じ業務identityを重要境界間で明示的に継承できる
- すべての重要な意味・実現選択へSource / Bindingを辿れる
- external assetの未観測事実をTEIが推測していない
- external EvidenceをCanonical meaningや第二の正本へ昇格していない
- Business Result / Execution Error / Observationを区別できる
- deterministic / generated artifact / real-provider fixture / actual external asset Evidenceを区別できる
- secret等の安全境界を説明できる
- generated artifactをcleanなsource revisionから再現できる
- Evidence runが参照するsource revisionを実行前後で固定できる
- #308で少なくとも一つのreal provider runtimeを実証できる
- candidate Ready revision Rを一つに明示できる
- #310 / #255 / #308の必須EvidenceがRへ揃っている
- #214 actual Oracleのverified / unresolved状態を明示している
- #214 unresolvedならOracle compatibility / actual `master_data` integrationを実証済みと主張していない
- Web側でOracle固有事実を補わないHandover制約を持つ
- 第一Web Targetへ、Web側で意味を補わず同じbaselineを渡せる

Gate B Ready後は[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md) / #291に従い、**Readyと判定したcandidate Rそのもの**をHandover Baselineとして固定してから第一Web Targetへ進みます。

この条件が揃うまでは、第一Web Targetの本格実装着手条件を満たしたとは判定しません。
