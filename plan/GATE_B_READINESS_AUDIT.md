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

現在のGate B残件は「Driverの中心設計が未成立」という種類ではなく、**actual external facts、実provider、generated artifact再現、通常Go runnerによる実Evidence取得**へ収束しています。

Gate Bを直接止めているのは次の4 Evidence laneです。

| Lane | Repository / GitHub Issue | Evidence nature | 現在地 |
| --- | --- | --- | --- |
| A | `ISSUE-000022` / #214 | actual external asset | **Blocked** — actual Oracle facts / actual read待ち |
| B | `ISSUE-000031` / #308 | real-provider ephemeral fixture | **Ready to run** — Evidence専用空PostgreSQL DB待ち |
| C | `ISSUE-000025` / #255 | generated artifact reproducibility | **Ready to check** — clean normal Go environment待ち |
| D | `ISSUE-000032` / #310 | deterministic repository | **Ready to run** — clean checkout / normal Go runner待ち |

4 laneは相互代替しません。

`GATE_B_EVIDENCE_ACQUISITION.md`も同じ4 Evidence nature、source-fixed境界、直近の#214 / #308 / #255安全条件へ同期しています。

GitHub Actionsによる継続的な再検証は重要ですが、この4 laneのProduct Evidenceと同一ではありません。Actions停止そのものを意味設計Gapへ読み替えず、逆に未実行をgreenとも扱いません。

## Gate B candidate Ready revision

各Evidence laneが個別run内で`repositoryRevision`を固定していても、異なるrevisionのEvidenceを寄せ集めた状態ではGate B Readyと判定しません。

最終監査では、一つのclean Repository revisionを**candidate Ready revision**として明示します。

candidateを`R`とする場合、少なくとも次をRへ揃えます。

| Lane | Rへ固定するもの |
| --- | --- |
| #310 | deterministic Repository Evidence JSONの`repositoryRevision` |
| #255 | generated artifact Evidence JSONの`repositoryRevision` |
| #308 | real PostgreSQL Evidence JSONの`repositoryRevision` |
| #214 | external Evidence acceptance receiptの`repositoryRevision` |
| #214 actual runtime | Rのauthoring / runtime境界を利用したactual Oracle read / Semantic E2Eであることを追跡可能にする |

Repository revisionを持つEvidenceでは、`repositoryRevision = R`かつ`reproducibleRevision = true`を要求します。

次のような混在はReady Evidenceとして採用しません。

```text
#310 -> R1
#255 -> R2
#308 -> R3
#214 acceptance -> R4
```

### external factsの扱い

Oracleのphysical object、raw columns、provenance、read-only権限等の外部観測事実そのものはGit revisionに属しません。

同じexternal Evidence artifact / provenanceが引き続き有効であると確認できる場合、観測事実をrevisionごとに捨てる必要はありません。

ただし、そのEvidenceをTEI sourceへ結び付ける次はcandidate Rへ固定します。

- acceptance receipt
- SQL realization / mappingとの照合
- actual Oracle runtime / Semantic E2E

external Evidence artifactを再利用しても、旧revisionのacceptance receiptやruntime結果をRへ黙ってcarry-forwardしません。

### candidateが変わる場合

candidate RのEvidence取得中にsource修正が必要になった場合、Rを暗黙に新revisionへ読み替えません。

1. 必要なsource / generated artifactを修正する
2. commitする
3. 新candidate R'を選ぶ
4. #310 / #255 / #308 / #214 acceptance receiptをR'で再取得する
5. #214 actual runtimeもR'のauthoring / runtimeから追跡できるよう再確認する
6. R'でGate B Readyを再監査する

#255のrepair commitが生じた場合も同じです。repair前revisionで取得したsource-dependent Evidenceをrepair後candidateへ混ぜません。

### mainの進行とcandidateを分離する

candidateを選んだ後に並行作業で`main`が進んでも、candidateを自動的に最新mainへ動かしません。

- candidate RをそのままGate B Ready監査するなら、RとRのEvidenceを固定して扱う
- 後続commitをGate B baselineへ含めたいなら、新candidateを選び直してsource-dependent Evidenceを揃える

これにより、並行開発のたびに比較対象が暗黙に移動することを防ぎます。

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

## Lane A — `Order -> Customer` / actual external Oracle / #214

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
- actual read時のraw result列名を保持するOracle固有Payload（PR #364）
- driverが提供できたraw列metadataを未提供値の推測なしで保持する境界（PR #366）
- 0件とExecution Errorの分離

GitHub #179 / #185とRepository `ISSUE-000018 / 000019`は実装責務として完了しています。actual Oracle Evidence待ちを理由に成立済み実装Issueを再びOpen / In Progressへ戻しません。

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

### source-fixed acceptance receipt — PR #340

PR #340で、external Evidenceを「何となく受理済み」と扱わず、**どのEvidence artifactを、どのclean Repository revisionで照合したか**を機械可読receiptへ固定しました。

`tools/reference-external-query-evidence`はEvidenceを読む前にRepository revision / working treeを確認し、dirtyなら停止します。

Evidence JSON exact bytesのSHA-256を計算し、照合後にHEAD / working treeを再確認します。実行中にHEADが変化したrunや実行後に差分が残ったrunは成功receiptを出しません。

成功receiptの主な項目:

```text
evidenceNature = external-query-evidence-acceptance
repositoryRevision
workingTreeClean = true
reproducibleRevision = true
postRunRepository.checked = true
postRunRepository.repositoryRevision = repositoryRevision
postRunRepository.revisionStable = true
postRunRepository.workingTreeClean = true
evidenceArtifactFingerprint = sha256:<hex>
target
provider
accepted = true
```

`evidenceArtifactFingerprint`はexternal Evidence JSONのexact bytesを識別します。

`nativeSource.fingerprint`は、観測したnative sourceと現在authoringされている`ContractQuerySQLRealization.Source`が一致するかを確認します。

両者は相互代替しません。

receiptへphysical object / parameter / raw columns / provenance本文 / read-only mechanism本文 / Evidence path / credential等を複製しません。

また、**acceptance receiptはactual Oracle read Evidenceではありません**。

### actual read時のdriver観測 — PR #364 / #366

Oracle Runner固有Payloadはactual read時に次を保持できます。

- named parameterの名前
- 実際に返されたraw result列名
- driverが提供した場合の`databaseTypeName`
- driverが提供した場合の`nullable`
- driverが提供した場合の`length`
- driverが提供した場合の`precision` / `scale`

未提供metadataをfalseや0へ推測しません。Go側の`ScanType`やraw row値を通常Evidenceへ複製しません。

このruntime metadataは、provenance付きexternal Evidenceの`PhysicalType` / `Nullable`を置き換える第二の物理正本ではありません。実Oracle / 採用driverを観測する前に`PhysicalType`と`DatabaseTypeName`の文字列一致規則を先行固定しません。

したがって`ISSUE-000022` / #214は、

- Evidence intake boundary: **Ready**
- source-fixed acceptance receipt boundary: **Ready**
- actual-read観測metadata保持境界: **Ready**
- actual Oracle physical facts / read-only / driver integration / actual read: **Blocked**

です。

未観測の`CUSTOMER`、`CUSTOMER_ID`、Oracle SQL等をTEI側から作りません。

残るactual external Evidence:

- 実physical object / parameter / raw columns
- raw columnsの実型 / NULL扱い
- native realization source
- Semantic Customerへのexplicit mapping
- provenance
- 実効的read-only強制
- 採用Oracle driver integration
- actual readでraw result列名 / driver-reported metadataを観測した結果
- actual Evidence artifactのsource-fixed acceptance receipt
- Workspace → actual Oracle read → Semantic result E2E

## Lane B — `Order -> OrderLine[]` / real PostgreSQL engine / #308

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

成功Evidenceには少なくとも次を要求します。

```text
repositoryRevision
workingTreeClean = true
reproducibleRevision = true
postRunRepository.checked = true
postRunRepository.repositoryRevision = repositoryRevision
postRunRepository.revisionStable = true
postRunRepository.workingTreeClean = true
```

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

Event Triggerは通常のuser relationとは別にDatabase単位で存在でき、fixtureのDDLへ介入し得るため、disabledを含め既存なら拒否します。

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

このguardは「Evidence専用DBである」という運用意図そのものを完全証明するものではありません。実行先はEvidence専用に作成した空PostgreSQL Databaseへ限定します。

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

PR #341で、fixtureをDROPしてmissing-table Execution Errorを確認しただけで成功にせず、PostgreSQL catalogを再観測する境界を追加しました。PR #365以降はEvent Triggerも再観測します。

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

#308の実装存在をreal PostgreSQL成功Evidenceとは扱いません。実runは未取得です。

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

正規入口:

```bash
go run ./tools/reference-gate-b-repository-evidence
```

PR #346以降、targeted runnerは少なくとも次の三つを**独立check**として一つのrunで扱います。

1. 最終UI-less Scenario
2. external Evidence intake
3. Repository source境界Conformance

三つ目の等価コマンドは次です。

```bash
go test ./tools/internal/repositoryevidence -count=1
```

JSONでは`repository-evidence-source-boundary`として別checkに残します。

runner自身のpackageをtargeted `go test`へ含めず、共有helperのpure契約だけを検証するため、将来runner testが`run()`を呼ぶ場合にも自己再帰の経路を作りません。

runner全体ではさらに次を扱います。

- 実行前repository revision取得
- 実行前working tree clean確認
- Go toolchain identity取得
- Canonical Workspace Validation
- 上記targeted checks
- `-broad`指定時だけ`go test ./...`
- 実行後HEAD / working tree再確認
- 小さい非Canonical JSON Evidence出力

PR #321 / #336 / #351以降、`reproducibleRevision=true`は開始時cleanだけでは設定しません。post-run確認後に次のすべてを満たした場合だけtrueにします。

- 開始revision非空
- 実行前clean
- 実行後確認済み
- 実行前後でrevision文字列が同一
- `revisionStable = true`
- 実行後clean

PR #336以降、source状態の観測規則は#255 / #308と同じtool-only packageへ収束しています。

PR #346はその共有source境界のpure Conformanceをtargeted Evidenceへ直接含めますが、PRの存在自体を#310成功Evidenceとは扱いません。

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

### 正規個別tool

`tools/reference-golden`:

- Worldのcheck / write
- Plugin Lockのcheck / write
- 保存Durable RunのPlugin Lock依存チェック
- PR #367以降、World / Plugin Lock writeも既存goldenを先にtruncateしないsafe replace

`tools/reference-durable-run-golden`:

- Durable Run全体を`deployment-start`と同じinternal API経路から正規生成
- 保存goldenとbyte-for-byteで比較
- stale時だけsafe write

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

PR #326で不正Solution Canonical IDもpanicではなく通常errorでfail closedするようにしました。

### 最終完了Evidence runner — PR #331 / #334

#255の最終Evidence入口は次です。

```bash
go run ./tools/reference-generated-artifact-evidence
```

runnerは`-write`を呼ばず、両golden toolをcheck-onlyで実行し、実行前後のRepository状態も確認します。

最終Evidence候補には次をすべて要求します。

- final check開始時にworking tree clean
- `reference-golden`一致
- `reference-durable-run-golden`一致
- final check実行前後でHEAD同一
- final check終了後もworking tree clean
- `reproducibleRevision = true`

PR #336以降、source観測は#310 / #308と共通化されています。

### Evidenceと修復を分離する

`-write`はstaleなgenerated artifactを正規状態へ戻す**修復操作**です。PR #367のsafe replaceも修復途中の破損リスクを下げる境界であり、`-write`やsafe replaceの成功だけをGate B完了Evidenceにはしません。

staleだった場合:

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
reference-generated-artifact-evidenceを再実行
  ↓
同じHEAD / clean working tree / reproducibleRevision=true
  ↓
#255完了Evidence候補
```

最初から一致している場合は形式的な`-write`を行いません。

2026-09-04の静的監査では、保存済みproduction Plugin Lockから計算したEnvironment fingerprintと保存Durable Runの`pluginLockFingerprint`は一致していました。ただし、これは現在のCanonical Workspaceから両artifactを正規再現できることの証明ではありません。

## Gate B条件別監査

| 条件 | 状態 | 根拠 / 残件 |
| --- | --- | --- |
| 同じApplication定義から主要Scenarioを構成できる | Ready (synthetic structure) | 最終UI-less Scenario |
| 同じSemantic Order identityを後続境界へ引き継げる | Ready (synthetic structure) | customer_id / order_idを明示継承 |
| Pattern / Query / Relation / Datasource / Capability / Bindingを追跡できる | Ready | Driver / Source Index / Conformance |
| browseをSemantic resultまで実行できる | Ready | explicit result mapping |
| external Evidence intake境界が成立 | Ready | #214 intake boundary |
| external Evidence acceptanceをclean source / exact artifactへ固定できる | Ready | PR #340 |
| Oracle actual readのraw列名 / driver提供metadataを保持できる | Ready (runtime boundary) | PR #364 / #366。actual read自体はBlocked |
| referenceをactual Oracle readまで実行できる | Blocked | #214 actual external Evidence待ち |
| read-only `master_data`を実効的に保証 | Blocked | #214 actual read-only Evidence待ち |
| header-detailの意味・storage・runtimeが成立 | Ready | #244 Completed |
| PostgreSQL fixture前安全境界 | Ready | PR #294 / #300 / #306 / #324 / #335 / #365 |
| PostgreSQL source revision固定 | Ready | PR #336 |
| PostgreSQL cleanup後空状態再確認 | Ready | PR #341 / #365。user relation 0件 / Event Trigger 0件 / fixture object不在 |
| header-detail実PostgreSQL engine Evidence | Pending | #308専用空DBで実run待ち |
| Validation不成立時にCommandへ進まない | Ready | Driver Conformance |
| Authorization denied時にCommandへ進まない | Ready | Driver Conformance |
| Business ResultとExecution Errorを区別 | Ready | Driver / ADR-0020 |
| Command acknowledgementとObservationを分離 | Ready (synthetic) | ADR-0023 / `ExecuteObservationRead` |
| secret / runtime parameterを通常Evidenceへ保存しない | Ready | runtime / Evidence境界 |
| Repository source境界のtargeted Conformanceがある | Ready | PR #346 |
| generated artifact個別照合toolがある | Ready | `reference-golden` / `reference-durable-run-golden` |
| generated artifact repairが既存artifactを先にtruncateしない | Ready | PR #367 + Durable Run safe write。repair成功≠完了Evidence |
| generated artifact final check-only runnerがある | Ready | PR #331 / #334 |
| 最新authoringからgenerated artifactをclean revisionで再現 | Pending | #255 actual check-only run待ち |
| 最終Scenario / Evidence intake / source境界をnormal runnerで実証 | Pending | #310実run待ち |
| 4 laneのsource-dependent Evidenceを一つのcandidate revisionへ揃える | Pending | 4 lane実Evidence取得後の最終整合監査 |
| 第一Web Targetへ同じ意味を渡せる | Partial | candidate revision整合 + #291 Handover待ち |

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

### 準備段階

candidate Ready revisionを選ぶ前に、source変更を必要とし得る作業を先に解消します。

1. #255をcheckし、staleなら正規toolで修復 → 差分確認 → commitする
2. #214 actual Oracle physical facts / provenance / read-only Evidenceを取得する
3. 観測事実を根拠に必要なOracle realizationを明示authoringし、commitする
4. その他のEvidence準備でsource修正が必要ならcommitする
5. working treeをcleanにし、Gate Bへ含めるsource変更が出揃ったrevisionをcandidate Rとして記録する

### candidate Rでの最終Evidence cycle

6. #255 `reference-generated-artifact-evidence`をRでcheck-only実行する
7. #310 deterministic Repository EvidenceをRで実行する
8. #308 real PostgreSQL EvidenceをRで実行する
9. #214 external EvidenceをRで受理しsource-fixed acceptance receiptを取得する
10. Rのauthoring / runtimeを使ってactual Oracle read → explicit Semantic mappingを確認する
11. #310 / #255 / #308 / #214 acceptance receiptの`repositoryRevision = R`と`reproducibleRevision = true`を確認する
12. actual Oracle runtimeがRへ追跡できることを確認する
13. secret / read-only / 0件 / Execution Error / Observation期待不一致 / source revision不変 / PostgreSQL fixture前後のEvent Trigger不在 / Oracle未提供metadata非推測等のnegative caseを再監査する
14. Gate BをRについてReady / Not Readyで再判定する
15. Readyなら#291で**同じR**をHandover Baselineとして固定する
16. その後に第一Web Target #70 / Gate C Evidence #270へ進む

candidate RでいずれかのEvidenceがsource修正を要求した場合、Rを修正後revisionへ読み替えません。修正をcommitし、新candidate R'を選んでsource-dependent EvidenceをR'へ揃え直します。

#214に必要な実外部事実が得られない場合は、その領域をBlockedとして保持し、未観測情報を埋めて先へ進めません。

## GitHub Actionsとの関係

GitHub Actions workflowは現行Repositoryでは意図的に使用していません。

過去のrunner未割当事象を追跡した#192は、現行運用では対象外として`not_planned`でclosedです。

Actionsを将来再導入する場合は、その時点のworkflow / runner / Repository要件で新しいCI検証Issueを作成します。CIサービスのgreenをGate B Product Evidenceの代替にはしません。また、workflow failureがpushそのものを不必要にblockしない運用を維持します。

## Gate B Readyの定義

Gate Bは、次をEvidenceから説明できるときReadyとします。

- UIなしで主要Scenarioを一続きに再現できる
- 同じ業務identityを重要境界間で明示的に継承できる
- すべての重要な意味・実現選択へSource / Bindingを辿れる
- 外部資産の未観測事実をTEIが推測していない
- external EvidenceをCanonical meaningや第二の正本へ昇格していない
- acceptance receiptをactual external readへ格上げしていない
- actual Oracle readでdriverが提供していないmetadataを推測していない
- Business Result / Execution Error / Observationを区別できる
- deterministic / generated artifact / real-provider fixture / actual external asset Evidenceを区別できる
- read-only / secret等の安全境界を実Evidenceから説明できる
- generated artifactをcleanなsource revisionから再現できる
- `-write`やsafe replaceの成功をgenerated artifact完了Evidenceへ読み替えていない
- Evidence runが参照するsource revisionを実行前後で固定できる
- Repository source固定境界自体のtargeted Conformanceが成立している
- #308 fixtureが終了後に専用空DBの観測条件（user relation 0件 / Event Trigger 0件 / fixture object不在）へ戻ったことを説明できる
- candidate Ready revision Rを一つに明示できる
- #310 / #255 / #308 / #214 acceptance receiptのsource-dependent EvidenceがRへ揃っている
- #214 actual Oracle runtime / Semantic E2EがRのauthoring / runtimeへ追跡できる
- 異なるrevisionのEvidenceを黙ってcarry-forwardしていない
- 第一Web Targetへ、Web側で意味を補わず同じbaselineを渡せる

Gate B Ready後は[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md) / #291に従い、**Readyと判定したcandidate Rそのもの**をHandover Baselineとして固定してから第一Web Targetへ進みます。

この条件が揃うまでは、第一Web Targetの本格実装着手条件を満たしたとは判定しません。
