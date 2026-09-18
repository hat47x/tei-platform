# ISSUE-000006: Application Driverの最小vertical sliceを実装する

- 状態: In Progress — 主要境界成立、Gate B Evidence成立（R = `e5231839a5dcc62cce1181a2731fcb7c70d16736`）、第一Web Target着手待ち
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

### 2026-09-19 Gate B Ready（ISSUE-000029 / #273）

Gate Bの必須3-lane Evidence（#310 / #255 / #308）が、同じclean candidate revision `R = e5231839a5dcc62cce1181a2731fcb7c70d16736`のsame-runで成立し、`#387` final convergenceと`#273`独立artifact auditも通過しました。Gate Bは**Ready**です。判定の詳細は[`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md)、Handover Baselineは[`GATE_B_TO_C_HANDOVER_BASELINE_B.md`](../GATE_B_TO_C_HANDOVER_BASELINE_B.md)を参照してください。

actual Evidence取得の過程で、以下2件のsource-navigation bug（Driverではなくsyntax anchor / source index scannerの不具合）と、独立artifact auditの期待値staleさ1件を検出・修正しました（PR [#726](https://github.com/hat47x/tei-platform/pull/726) / [#727](https://github.com/hat47x/tei-platform/pull/727)）。Driver自体の意味境界に変更はありません。

以下の2026-09-04時点の記述は、Ready到達までの設計・計画履歴として保持します。

### 2026-09-04 現行Gate B scope（ISSUE-000035 / #380）

Driverの主要な意味境界、runtime境界、最終UI-less Scenario構造は成立しています。Gate B全体は **Not Ready** ですが、現在の直接Evidence残件は次の三つです。

| Repository / GitHub Issue | Evidence nature | 現在地 |
| --- | --- | --- |
| `ISSUE-000032` / #310 | deterministic repository | Ready to run — clean checkout / normal Go runner待ち |
| `ISSUE-000025` / #255 | generated artifact reproducibility | Ready to check — clean normal Go environment待ち |
| `ISSUE-000031` / #308 | real-provider ephemeral fixture | Ready to run — Evidence専用空PostgreSQL DB待ち |

最終Ready判定では、この三つの完了Evidenceを同じclean candidate revision `R`へ揃えます。

`ISSUE-000022` / #214、`ISSUE-000033` / #371、`ISSUE-000034` / #376のactual Oracle EvidenceはGate B必須blockerではありません。actual Oracle physical facts / provenance / effective read-only / driver integration / actual readは引き続きBlockedであり、完了扱いにはしません。actual external asset / legacy compatibility / 第二Database Provider圧力試験として継続します。

Gate Bではexternal Evidence intakeと未観測事実非推測の境界を#310で検証します。#214未完了時は#291 Handover Baselineの`unresolved`へ明示し、Web側でOracle物理情報やrealizationを補完しません。

**scope再整理だけではGate B Readyではありません。** #310 / #255 / #308のactual run Evidenceはまだ未取得です。

以下の4-lane表、Lane A〜D、旧Phase 1 / Phase 2、旧Ready条件は`ISSUE-000035`以前の設計・計画履歴として保持します。現行Gate B必須範囲と矛盾する場合は本節を優先し、各Evidence種別の非代替性、安全境界、actual Oracle未実証という事実は維持します。

2026-09-04時点で、Driverの主要な意味境界、runtime境界、最終UI-less Scenario構造は成立しています。

Gate B全体は **Not Ready** ですが、残件は新しいDriver抽象の追加ではありません。現在は性質の異なる次の4種類の実行Evidence取得へ収束しています。

| Lane | Repository / GitHub Issue | Evidence nature | 現在地 |
| --- | --- | --- | --- |
| A | `ISSUE-000022` / #214 | actual external asset | Blocked — actual Oracle facts / read-only / actual read待ち |
| B | `ISSUE-000031` / #308 | real-provider ephemeral fixture | Ready to run — Evidence専用空PostgreSQL DB待ち |
| C | `ISSUE-000025` / #255 | generated artifact reproducibility | Ready to check — clean normal Go environment待ち |
| D | `ISSUE-000032` / #310 | deterministic repository | Ready to run — clean checkout / normal Go runner待ち |

この4 laneは相互代替しません。

PR #354以降、最終Ready判定では、一つのcleanなcandidate repository revision `R`を固定し、**完了判定に用いる4 laneのEvidenceをすべてRへ揃えます**。異なるrevisionで得た成功Evidenceを寄せ集めて、一つのReady revisionへ読み替えません。

具体的なコマンド、安全条件、停止条件は`GATE_B_EVIDENCE_RUNBOOK.md`だけを正とします。本IssueはApplication Driverの成立内容と、Gate Bへ残る責務の所在を保持します。

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

Database Contract Queryのsignature / realization / executor / result mappingと、reference Patternからの明示invocationは実装責務として完了しています。GitHub #179 / #185、Repository `ISSUE-000018 / 000019`はDoneであり、actual Oracle Evidence待ちを理由に再びIn Progressへ戻しません。

#### external Evidence intake

actual external assetから得るEvidenceをCanonical Worldへ取り込まず、明示authoring済みSQL realizationと照合する境界が成立しています。

少なくとも次を照合できます。

- target / provider / `external-observed` ownership
- Contract input → physical parameter
- raw result → Semantic attribute
- native source fingerprint
- provenance
- effective read-only mechanism / Evidence参照
- Canonical Workspace Validation

Canonical WorkspaceにValidation Errorがある場合はEvidence受入前に停止します。SQL realizationが存在しない場合も、Evidenceから生成・補完せず停止します。

PR #340以降、受入れ成功は単なる人向けメッセージではなく、**どのexternal Evidence artifactをどのclean Repository revisionで照合したか**を示すsource-fixed acceptance receiptとして残せます。

receiptでは、external Evidence JSON exact bytesを`evidenceArtifactFingerprint`で識別し、実行前後のRepository source不変性を確認します。

ここで、

- `evidenceArtifactFingerprint`: 実際に検査したEvidence JSON artifactのexact bytes
- `nativeSource.fingerprint`: 観測したnative sourceと現在authoringされている実行sourceの一致

を別責務として扱います。

acceptance receiptへphysical object / parameter / raw columns / provenance本文 / read-only mechanism本文 / Evidence path / credentialを複製しません。

したがって`ISSUE-000022` / #214は現在、

- Evidence intake boundary: **Ready**
- source-fixed acceptance receipt boundary: **Ready**
- actual Oracle physical facts / read-only / driver integration / actual read: **Blocked**

です。

**acceptance receiptはactual Oracle read Evidenceではありません。** 未観測のOracle object / column / SQLをTEI側から推測しません。

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

`tools/reference-postgres-header-detail-runtime`はreal PostgreSQL engine + tool-owned ephemeral fixture Evidence用です。

現在は、fixture前・Provider runtime・fixture後・Repository sourceの四面を一つのrunで確認できる構造です。

##### Repository source

PR #336で#310 / #255 / #308のsource状態観測を`tools/internal/repositoryevidence`へ収束しました。

#308ではDatabaseへ接続する前にclean working treeとHEADを確認し、Provider runtime / cleanup / Database復元確認後にも再確認します。

同じclean revisionへ固定できないrunは成功Evidenceとして採用しません。

PR #355以降、`postRunRepository.checked=true`になるのは、実行後repository revisionとworking tree状態の両方を観測できた後だけです。観測自体に失敗した状態と、観測した結果cleanではなかった状態を区別します。このsource観測規則は#214 / #255 / #308 / #310で共有します。

##### fixture前安全性

PR #294 / #300 / #306 / #324 / #335で、fixture作成前のfail-closed境界を段階的に強化しました。

1. WorkspaceをLoad / Validateする
2. Validation ErrorならDB接続前に停止する
3. PostgreSQLへ接続する
4. `current_database()`を確認する
5. `postgres` / `template0` / `template1`をfixture先として拒否する
6. system / temporary領域を除くuser relationが0件であることを確認する
7. `public.order_lines`が存在しないことを再確認する
8. すべて成立した場合だけtool-owned fixtureをCREATEする

既存relationを削除してEvidence用DBへ転用する処理やoverrideはありません。

PR #335以降、そのrunで通過した非機密な安全判定を`fixtureDatabaseSafety`へ残します。

```json
{
  "guardPassed": true,
  "databaseIdentityVerified": true,
  "databaseIdentityPersisted": false,
  "nonDefaultDatabaseVerified": true,
  "userRelationsBeforeFixture": 0,
  "fixtureObjectAbsentBeforeCreate": true
}
```

このguardだけで「Evidence専用DBという運用意図」を完全証明したとは扱わず、実行先はEvidence専用に作成した空PostgreSQL Databaseへ限定します。

##### fixture後安全性

PR #341以降、tool-owned fixtureをDROPしてmissing-table Execution Errorを確認した後、PostgreSQL catalogを再観測します。

```json
{
  "verified": true,
  "userRelationsAfterCleanup": 0,
  "fixtureObjectAbsentAfterCleanup": true
}
```

cleanup後にuser relationが残る場合、それがtool由来か他主体の並行変更かを推測せずfail closedします。

real PostgreSQL engineでの成功Evidenceは`ISSUE-000031` / #308で取得します。実装済み安全境界の存在をactual run成功とは扱いません。

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

GitHub #266はこの境界設計として完了しています。

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

構造は成立していますが、現在のclean revisionでnormal Go runnerから成功したEvidenceは`ISSUE-000032` / #310で取得します。

## Lane A — actual Oracle / #214

actual Oracle object / parameter / raw columns / native source / provenance / effective read-only強制を観測し、actual read → Semantic Customerまで確認します。

現在は、

```text
actual physical facts
 -> external Evidence JSON
 -> 観測事実を根拠に明示authoring
 -> source-fixed acceptance receipt
 -> read-only Oracle connection
 -> Oracle driver integration
 -> actual read
 -> explicit Semantic mapping
```

の順に進めます。

Evidence intake / acceptance receiptがReadyであっても、actual external facts / actual read未取得を代替しません。

実外部事実が得られない場合はBlockedのまま保持し、架空情報で穴埋めしません。

## Lane B — real PostgreSQL / #308

Evidence専用の空PostgreSQL Databaseでheader-detail toolを実行し、同じrunで少なくとも次を確認します。

- cleanな固定Repository revision
- `fixtureDatabaseSafety`の期待値
- 2件取得
- 正常な0件
- tool-owned fixture欠落のExecution Error
- Environment / Database instance / executor Capability / Contract Query target identity
- explicit Semantic mapping
- support column非露出
- runtime parameter値 / credential / DSN / 実Database名非保存
- `fixtureCleanupSafety`の期待値
- `evidenceNature = real-postgres-ephemeral-fixture`
- `reproducibleRevision = true`

これはproduction既存Database Evidenceではありません。

## Lane C — generated artifact reproducibility / #255

Canonical Workspaceから導出されるgenerated artifactの最終完了Evidence入口は`tools/reference-generated-artifact-evidence`です。

このrunnerはcheck-onlyで、次を一つのrunに束ねます。

- 実行前repository revision / clean状態
- Go toolchain identity
- `tools/reference-golden`の一致
- `tools/reference-durable-run-golden`の一致
- 実行後repository revision / clean状態
- 実行前後HEAD同一
- `reproducibleRevision = true`

個別toolの役割:

- `tools/reference-golden`: World / Plugin Lock check/write + Durable RunのPlugin Lock依存check
- `tools/reference-durable-run-golden`: Durable Run全体の正規生成・完全一致check / safe write

Durable Run toolは、Workspace Validation、exact Solution Release、fixed observed deployment state Validation、`releaseplan.CompileWithDeployment`、`pluginruntime.BuildPlan`、Plugin Lock / Runtime Policy fingerprint、`deploymentdurable.StartWithRuntimePolicy`を通る正規internal API経路を使います。

PR #326で`-solution`のCanonical IDも`ParseCanonicalID`へ変更し、不正入力をpanicではなく通常errorとしてfail closedするようにしました。

### `-write`は修復であり完了Evidenceではない

staleなgenerated artifactに対する`-write`は修復操作です。

修復差分が出た場合は、

1. 正規toolで修復する
2. `git diff`で意味上の差分を確認する
3. 意図したgenerated artifactだけをcommitする
4. 新しいclean revisionで`tools/reference-generated-artifact-evidence`を再実行する

という順にします。

`-write`直後のdirty treeをGate B完了Evidenceにはしません。一致済み生成物を形式的に書き換えず、fingerprintやgenerated JSONを手編集しません。

## Lane D — deterministic Repository / #310

正規runnerは`tools/reference-gate-b-repository-evidence`です。

PR #346以降、targeted Evidenceは次の3 checkを独立して扱います。

1. 最終UI-less Scenario
2. external Evidence intake
3. Repository source-boundary pure Conformance

三つ目は`tools/internal/repositoryevidence`のpure契約を検査します。runner自身のpackageをtargeted `go test`へ含めず、将来runner testが`run()`を呼んでも自己再帰する経路を作りません。

一つのrunではさらに次を扱います。

- 実行前repository revision / working tree clean
- Go toolchain identity
- Canonical Workspace Validation
- 上記3 targeted checks
- 明示指定時だけbroad `go test ./...`
- 実行後HEAD / working tree再確認
- 小さい非Canonical JSON Evidence

PR #321以降、`reproducibleRevision=true`には次をすべて要求します。

- 実行前clean
- 実行後確認済み
- HEAD同一
- 実行後clean

PR #336以降、source観測規則は#255 / #308と同じtool-only packageへ収束しています。

PR #351以降、開始時working treeがcleanであっても、post-run確認が終わるまでは`reproducibleRevision=false`を維持します。途中失敗JSONを再現可能な成功Evidenceへ誤読しません。

PR #355以降、`postRunRepository.checked=true`は、実行後repository revisionとworking tree状態の両方を観測できた場合だけです。

TestやConformanceが実装されていても、通常Go runnerで実runを成功させるまでは#310完了Evidenceとは扱いません。診断用`-allow-dirty`も完了Evidenceには使用しません。

## Evidence文書の権威関係

次へ一本化します。

- `GATE_B_EVIDENCE_ACQUISITION.md`: 何をEvidenceとして取得するか / 証明範囲 / 完了条件
- `GATE_B_EVIDENCE_RUNBOOK.md`: 具体コマンド / 実行順 / 差分確認 / fail-closed停止条件
- `GATE_B_READINESS_AUDIT.md`: 取得EvidenceからのGate B判定
- `ISSUE-000029`: 4 laneの進行を総合監査へ戻す責務
- `GATE_B_TO_C_HANDOVER_CONTRACT.md`: Ready revisionをGate Cへ渡す方法

本Issueへコマンド列を重複正本として持ちません。

## GitHub Actionsの扱い

GitHub Actions workflowは現行Repositoryでは意図的に使用していません。

過去のrunner未割当事象を追跡した#192は、現行運用では対象外として`not_planned`でclose済みです。

Gate Bでは特定CIサービスのgreenをProduct Evidenceの代替にしません。一方、未実行をgreenとも扱いません。

将来Actionsを再導入する場合は、その時点のworkflow / runner / Repository要件で別のCI検証Issueを作り、workflow failureがpushそのものを不必要にblockしない運用を維持します。

## Gate B Ready条件

Gate Bはコード量やIssue close数ではなく、Evidenceから少なくとも次を説明できるときReadyとします。

- 同じApplication定義から主要ScenarioをUIなしで一続きに再現できる
- 同じ業務identityを重要境界へ明示的に継承できる
- Pattern / Query / Relation / Contract Query / Datasource / Capability / Bindingを追跡できる
- raw resultとSemantic resultの対応が明示されている
- Relation traversalをphysical FKや名前から推測していない
- external assetの未観測事実をTEIが補完していない
- external Evidence acceptanceをclean source / exact artifactへ固定できる
- acceptance receiptをactual external readへ格上げしていない
- `master_data`へのwriteをactual Evidenceから実効的に防げる
- Validation / Authorization不成立時にCommandへ進まない
- Business Result / Execution Error / Observationを区別できる
- Query 0件とExecution Errorを区別できる
- deterministic / generated artifact / provider fixture / actual external asset Evidenceを区別できる
- secret / runtime parameter値を通常Evidenceへ残さない
- Driver固有状態やEvidence artifactをCanonical Worldへ逆流させない
- generated artifactをcleanなsource revisionから再現できる
- Evidence runが参照するsource revisionを実行前後で固定できる
- Repository source固定境界自体のtargeted Conformanceが成立している
- #308 fixtureが終了後に専用空DBの観測条件へ戻ったことを説明できる
- 4 laneの完了判定に用いるEvidenceが同じcandidate revision Rを参照している
- 第一Web TargetへWeb固有の意味補完なしに同じbaselineを渡せる

## 次の作業

具体的な実行手順は`GATE_B_EVIDENCE_RUNBOOK.md`だけを正とします。

Evidence取得は二段階で進めます。

### Phase 1 — 事前確認 / sourceを確定する

1. #310をcleanな通常Go runnerで事前実行する
2. #255をcheck-onlyし、staleなら正規toolで修復 → 差分確認 → commitする
3. #308をEvidence専用の空PostgreSQL Databaseで事前実行する
4. #214 actual Oracle physical facts / provenance / read-only根拠を取得する
5. 観測事実を根拠に必要なOracle realizationを明示authoringする
6. #214 acceptance / Oracle driver / actual readを事前確認する
7. Gate Bに必要なsource / generated artifact変更をすべてcommitする

この段階で得た成功JSONも、その後sourceが変わった場合は最終完了Evidenceとして流用しません。

### Phase 2 — 最終収束 / candidate revision Rを固定する

1. cleanなcandidate revision Rを固定する
2. Rで#310を実行する
3. Rで#255をcheck-only実行する
4. Rで#308を実行する
5. Rで#214 acceptance receiptを再取得し、Rのruntime / realizationでactual Oracle readを実行する
6. 4 laneすべての完了EvidenceがRを参照していることを確認する
7. negative caseを再監査する
8. Rを対象にGate BをReady / Not Readyで再判定する
9. Readyなら`ISSUE-000030` / #291でRをHandover Baselineとして固定する
10. その後に`ISSUE-000007` / #70第一Web Targetへ進む

Phase 2中にsource変更が必要になった場合は、その変更をcommitした新revisionをRとして選び直し、4 laneをもう一度そのrevisionへ揃えます。

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
- 成立済み実装IssueをEvidence待ちで再Openすること
- acceptance receiptをactual Oracle read Evidenceとして扱うこと
- source-boundary Conformanceを#214 / #255 / #308の代替にすること
- dirty / source-changing runをGate B完了Evidenceへ採用すること
- cleanup後にuser relationが残る#308 runを成功Evidenceとして採用すること
- 異なるrevisionで得た4 laneの成功Evidenceを一つのReady revisionへ読み替えること

## 関連

- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `ISSUE-000022` / GitHub #214
- `ISSUE-000024` / closed GitHub #244
- `ISSUE-000025` / GitHub #255
- `ISSUE-000029` / GitHub #273
- `ISSUE-000030` / GitHub #291
- `ISSUE-000031` / GitHub #308
- `ISSUE-000032` / GitHub #310
- ADR-0020 / ADR-0021 / ADR-0023
- PR #281 / #285 / #288 / #294 / #300 / #306 / #309 / #311 / #315 / #318 / #321 / #323 / #324 / #326 / #328 / #329 / #331 / #334 / #335 / #336 / #340 / #341 / #342 / #343 / #344 / #346 / #347 / #349 / #350 / #351 / #352 / #354 / #355 / #356 / #357 / #358
