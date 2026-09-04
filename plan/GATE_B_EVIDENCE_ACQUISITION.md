# Gate B Evidence Acquisition

## 目的

この文書は、Gate Bを通過するために**何のEvidenceが必要か、各Evidenceが何を証明し、何を証明しないか**を定めます。

具体的なコマンド、実行順、保存前の差分確認、fail-closedの停止条件は[`GATE_B_EVIDENCE_RUNBOOK.md`](GATE_B_EVIDENCE_RUNBOOK.md)を唯一の実行手順として扱います。本書では同じコマンド列を重複して管理しません。

Gate Bの正式なReady / Not Ready判定は[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)を正とします。

したがって三文書の役割は次です。

```text
GATE_B_EVIDENCE_ACQUISITION.md
  └─ 何をEvidenceとして取得するか / 証明範囲 / 完了条件

GATE_B_EVIDENCE_RUNBOOK.md
  └─ どの順序・コマンド・停止条件で取得するか

GATE_B_READINESS_AUDIT.md
  └─ 取得済みEvidenceからGate Bをどう判定するか
```

## 現在地

Application Driverの主要意味境界、最終UI-less Scenario構造、external-observed Query Evidenceの受入境界、header-detail PostgreSQL fixtureの安全境界は成立しています。

PR #379を起点とした`ISSUE-000035` / #381の再監査により、actual Oracle固有EvidenceはGate Bの必須完了条件から外し、継続actual external asset Evidenceとして扱います。

2026-09-04時点で、Gate B Readyに直接必要なsource-dependent実行Evidenceは次の三つです。

1. `ISSUE-000032` / #310 — 最終UI-less ScenarioとEvidence受入境界のdeterministic Repository / normal Go runner Evidence
2. `ISSUE-000025` / #255 — Reference World / Plugin Lock / Durable Runのgenerated artifact reproducibility Evidence
3. `ISSUE-000031` / #308 — header-detailのreal PostgreSQL engine + ephemeral fixture Evidence

これとは別に、actual external Oracleの継続Evidence trackを保持します。

- `ISSUE-000022` / #214 — external-observed `master_data`のactual Oracle物理 / provenance / actual read
  - `ISSUE-000033` / #371 — actual Oracle driver / Host assembly
  - `ISSUE-000034` / #376 — actual Oracle read-only Evidence

#214 / #371 / #376は廃止しません。actual Oracle compatibility / actual external asset integrationを主張するためのEvidenceとして継続します。ただし、外部環境情報が得られないことだけで第一Web TargetへのProduct学習を止めません。

GitHub Actionsの再開・継続実行は重要ですが、上記Evidenceそのものとは分けます。Actions停止をSemantic design gapとは扱わず、未実行をgreenとも扱いません。

## Evidenceの分類

TEIではEvidenceを少なくとも四種類に分けます。ただし、四種類すべてをGate Bの必須完了条件にするわけではありません。

| 種類 | 代表例 | Gate Bでの位置づけ | 証明するもの | 代替しないもの |
| --- | --- | --- | --- | --- |
| deterministic repository Evidence | Workspace Validation、UI-less Scenario、external Evidence intake tests | **必須** | Repository内のauthoring・Driver / Evidence境界が固定revisionで再現すること | generated artifact再現性、実provider、実外部資産 |
| generated artifact reproducibility Evidence | World / Plugin Lock / Durable Run golden | **必須** | 正規入力から保存generated artifactをclean revisionで再現できること | Driver Scenario、実provider、実外部資産 |
| real-provider ephemeral fixture Evidence | header-detail PostgreSQL tool | **必須** | 実provider / driver / native SQLを通るruntimeとfixture前後の安全状態 | production既存DBの事実、Oracle Evidence |
| actual external asset Evidence | `master_data` Oracle | **継続Evidence / Gate Bではunresolved可** | TEI非所有資産の実物理・provenance・read-only・actual read | synthetic / generated / ephemeral fixture |

一種類の成功を別種類のEvidenceへ格上げしません。

特に、#310 synthetic Scenarioや#308 PostgreSQL Provider Evidenceをactual Oracle Evidenceへ読み替えません。

## 最終Ready判定では3必須Evidenceを同じrevisionへ揃える

Gate Bの最終Ready判定では、一つのcleanなcandidate repository revision `R`を定め、**完了判定に用いる3必須EvidenceをすべてRへ揃えます**。

少なくとも次を要求します。

- #310 deterministic Repository Evidenceの`repositoryRevision = R`かつ`reproducibleRevision = true`
- #255 generated artifact Evidenceの`repositoryRevision = R`かつ`reproducibleRevision = true`
- #308 Provider Evidenceの`repositoryRevision = R`かつ`reproducibleRevision = true`
- `GATE_B_READINESS_AUDIT.md`がRを対象にReady / Not Readyを判定している

Rと異なるrevisionでしか成功していない必須Evidenceは、「意味は変わっていないはずだ」と推測して流用せず、Rで再実行します。

実行順はRunbookの二段階手順を正とします。事前確認の後でsourceやgenerated artifactが変わった場合、それ以前の成功JSONを最終完了Evidenceへ読み替えません。

### #214 actual Oracleの扱い

#214がGate B最終判定時点で完了している場合は、actual external Evidence / acceptance receipt / actual readをHandover Baseline Rへ追跡可能にします。

未完了の場合でもGate Bを自動的にNot Readyへ戻しません。その代わりHandoverへ次をunresolvedとして明示します。

- #214 actual external Oracle Evidence: not verified
- #371 actual Oracle driver / Host assembly: 未確定または未検証
- #376 actual Oracle read-only Evidence: 未取得
- Oracle compatibility / actual `master_data` integrationを実証済みとは主張しない
- Web側でOracle physical object / datatype / read-only方式 / driver都合を補わない

## Evidence A — deterministic Repository / UI-less Scenario

関連: `ISSUE-000032` / GitHub Issue #310

### 性質

これは**deterministic repository Evidence**です。

最終UI-less Scenarioはsynthetic adapterを含みます。通常Go runnerで成功しても実Oracle / 実PostgreSQL / 実外部Command Evidenceへ格上げしません。

### 正規Evidence

PR #315 / #321 / #336 / #346 / #351 / #355で`tools/reference-gate-b-repository-evidence`と共通Repository source観測境界を整備しました。

runnerはWorkspace Validation、最終UI-less Scenario、external Evidence intake tests、Repository source境界のpure Conformanceを一つのrunで独立checkとして扱い、実行前後のrepository状態も確認します。

Gate Bへ採用するEvidenceには少なくとも次を要求します。

- 実行開始時にworking tree clean
- 実行前repository revisionを記録する
- Workspace Validationが成功する
- 最終UI-less Scenario checkが成功する
- external Evidence intake checkが成功する
- `tools/internal/repositoryevidence`のsource-boundary Conformance checkが成功する
- 実行後repository revisionを再確認する
- 実行前後でHEADが同一
- 実行後もworking tree clean
- `reproducibleRevision = true`

PR #346でsource-boundary Conformanceをtargeted Evidenceへ独立checkとして追加しています。runner自身のpackageをtargeted testへ含めず、source観測helperのpure契約だけを検証することで自己再帰を避けます。

PR #351以降、開始時working treeがcleanであっても、post-run確認が終わるまでは`reproducibleRevision=false`を維持します。途中失敗を再現可能な成功Evidenceとして扱いません。

PR #355以降、`postRunRepository.checked = true`になるのは、実行後repository revisionとworking tree状態の両方を観測できた後だけです。観測自体に失敗した状態と、観測した結果cleanではなかった状態を分けます。

### 証明対象

同じ`order_entry` Application / Semantic Order identityから、少なくとも次の境界を一続きに接続できることを確認します。

```text
Authentication
  -> browse
  -> record Pattern resolution
  -> reference
  -> header-detail
  -> Canonical Validation
  -> Authorization
  -> Command Business Result
  -> separate Observation read
```

加えて、external Evidence intakeとRepository source固定境界のConformanceを通常runnerで実行します。

ここで確認するexternal Evidence intakeは、actual Oracleが未取得でも次を守れることのGate B必須Evidenceです。

- external EvidenceをCanonicalへ昇格させない
- provenance / read-only Evidence参照を要求する
- SQL realizationがなければEvidenceから補完しない
- native source fingerprint staleを検出する
- invalid Workspaceを受理しない

### 完了候補

- `reference-gate-b-repository-evidence`が正常終了する
- `reproducibleRevision = true`である
- 最終Scenario testが通常Go runnerで成功する
- external Evidence intakeの関連testsが成功する
- Repository source-boundary Conformanceが成功する
- repository revisionを記録できる
- synthetic Evidenceであることを明示できる
- 実外部Evidenceとの区別を維持する

## Evidence B — generated Reference artifact reproducibility

関連: `ISSUE-000025` / GitHub Issue #255

### 責務

#255は**generated Reference artifactの再現性だけ**を扱います。

real PostgreSQL engine Evidenceは#308、actual Oracle Evidenceは#214、deterministic Repository / Driver Evidenceは#310へ分離します。

### 性質

これは**generated artifact reproducibility Evidence**です。

Canonical Workspace、Plugin Lock、fixed observed deployment state等の正規入力から、保存generated artifactを同じclean repository revisionで再現できることを確認します。

対象は少なくとも次です。

- `reference/world.json`
- `reference/workspace/.tei/plugin-lock.json`
- `reference/order_rollout.durable-run.m0.21.start.json`

`reference/observed/*`やlegacy-auth fixtureは、Canonical authoring変更だけを理由に自動更新しません。

### 正規完了Evidence

PR #331で、#255の完了条件を`tools/reference-generated-artifact-evidence`へ集約しました。

このrunnerは修復操作を行わず、内部で次のcheck-only toolを実行します。

- `tools/reference-golden`
- `tools/reference-durable-run-golden`

そのうえで実行前後のrepository HEAD / working treeも確認します。

最終Evidence候補には少なくとも次を要求します。

- 実行開始時にworking tree clean
- World / Plugin Lockが正規生成結果と一致する
- Durable Run全体が正規生成結果と一致する
- 実行前後でHEADが同一
- 実行後もworking tree clean
- `reproducibleRevision = true`

PR #355以降、post-runの`Checked = true`は、実行後repository revisionとworking tree状態の両方を観測できた後にだけ設定されます。

### repairとの関係

`reference-golden -write` / `reference-durable-run-golden -write`はstale解消の**修復操作**です。

PR #367以降、World / Plugin Lockの`reference-golden -write`も既存goldenを先にtruncateせず、一時fileへの全量write / Sync / Close後にRenameするsafe replaceへ揃っています。これは修復途中の失敗で既存artifactを壊しにくくする安全境界であり、safe replace成功そのものを完了Evidenceへ格上げしません。

修復差分が生じた場合は、差分を意味上確認し、意図したgenerated artifactだけをcommitし、新しいclean revisionで`reference-generated-artifact-evidence`を再実行します。

`-write`直後のdirty working treeで個別checkが通ったことだけを完了Evidenceへ採用しません。

### 証明対象

- Worldを最新Canonical Workspaceから再現できる
- Plugin Lockを同じWorkspaceから再現できる
- Durable Run全体を正規internal API経路から再現できる
- Plugin Lock / Runtime Policy依存が正規生成結果と一致する
- 差分が現在のauthoring / Plugin Lock / fixed observed fixtureから説明できる
- fingerprintや生成JSONを人手で合わせていない
- Evidence runが単一のclean repository revisionへ固定されている

### 完了候補

- `reference-generated-artifact-evidence`が正常終了する
- `reproducibleRevision = true`である
- artifact差分を意味上説明できる、または最初から差分がない
- external-observed `master_data`の未観測物理情報が生成物へ紛れ込んでいない
- observed fixtureを生成結果へ合わせて書き換えていない
- safe replaceや`-write`の成功だけを完了Evidenceへ読み替えていない
- 結果を#273 Gate B監査へ対応付けられる

具体的なcheck / repair順序、差分確認は`GATE_B_EVIDENCE_RUNBOOK.md`を正とします。

## Evidence C — header-detail real PostgreSQL engine

実Provider Evidence: `ISSUE-000031` / GitHub Issue #308

成立済み設計: `ISSUE-000024` / GitHub Issue #244

### 責務分離

#244では、keyless Relation Storage、`find_by_order` SQL realization、Driver runtime、Evidence toolの安全境界まで成立しており、設計IssueとしてCompletedです。

**実PostgreSQL engine上でEvidenceを取得する作業だけを#308で追跡します。** 実Provider実行が未完了であることを理由に、Relation Storage設計を再びIn Progressへ戻しません。

### 性質

これは**real PostgreSQL engine + ephemeral fixture Evidence**です。

実PostgreSQL provider / driver / native SQL executionを通しますが、tool自身が作る一時fixtureを利用します。production既存Databaseの物理事実を観測するEvidenceではなく、#214のactual Oracle実外部資産を代替しません。

actual OracleをGate B必須から外しても、#308を必須に残します。Gate Bをsynthetic Evidenceだけへ縮退させず、少なくとも一つのreal provider engineでDatabase Binding / native SQL / explicit mapping / failure boundaryを確認するためです。

### Repository source固定

PR #336で、#308も#310 / #255と同じ`tools/internal/repositoryevidence`へsource観測規則を揃えました。

- Workspace Load / DB接続より前にclean Repository revisionを要求する
- Provider runtime、fixture cleanup、Database復元確認後にHEAD / working treeを再確認する
- 実行前後で同じHEADかつcleanでなければ成功Evidenceを出さない
- `repositoryRevision` / `reproducibleRevision` / post-run Repository stateをEvidenceへ残す

PR #355以降、post-runの`Checked = true`は、実行後repository revisionとworking tree状態の両方を観測できた後にだけ設定されます。

### fixture作成前の安全境界

PR #294 / #300 / #306 / #324 / #335 / #365で、安全境界を段階的に強化しています。

Evidence toolはfixture作成前に次を確認します。

```text
clean Repository revision
  ↓
Workspace Load
  ↓
Canonical Validation
  ↓ Errorなら停止
PostgreSQL接続
  ↓
current_database()確認
  ↓
postgres / template0 / template1なら停止
  ↓
system / temporary領域を除くuser relation件数確認
  ↓ 1件以上なら停止
Event Trigger件数確認
  ↓ 1件以上なら停止
public.order_lines既存確認
  ↓ 既存なら停止
tool-owned fixture
  ↓
runtime Evidence
```

Event Triggerはuser relationとは別にDatabase単位で存在でき、fixtureの`CREATE TABLE` / `DROP TABLE`等のDDLへ介入し得るため、disabledを含め1件でも存在すれば停止します。

既存relationやEvent Triggerを削除・無効化してEvidence用Databaseへ転用する処理や、既定Databaseで続行するoverrideは持ちません。

このguardを通過したことだけでEvidence専用Databaseという運用意図を推測せず、実行先はEvidence専用に作成した空PostgreSQL Databaseへ限定します。

PR #335 / #365以降、fixture作成前に観測した安全条件を`fixtureDatabaseSafety`として非機密metadataだけEvidenceへ残します。

少なくとも次を確認します。

- Database identityを確認できた
- 実Database名はEvidenceへ保存しない
- PostgreSQL既定Databaseではない
- fixture前のuser relationが0件
- fixture前のEvent Triggerが0件
- `public.order_lines`がfixture前に存在しない

### cleanup後の安全境界

PR #341で、Provider runtimeの2件 / 0件確認、tool-owned fixture削除、missing-table Execution Error確認の後にPostgreSQL catalogを再観測し、**接続先を空状態へ戻せたこと**まで成功Evidenceの条件にしました。PR #365以降はEvent Trigger件数も再観測します。

`fixtureCleanupSafety`で少なくとも次を確認します。

- `verified = true`
- system / temporary領域を除くuser relationが0件へ戻った
- Event Triggerが0件である
- `public.order_lines`が存在しない

終了時にuser relationまたはEvent Triggerが1件でも存在する場合、それがtool由来か他主体の並行変更かを推測せずfail closedで停止します。cleanup EvidenceにもDatabase名、DSN、credential、Event Trigger名や定義本文は保存しません。

### 証明対象

- cleanなRepository revisionにProvider runを固定できる
- `order_id=1001`で2件を正常collectionとして取得できる
- `order_id=9999`で0件を正常な空collectionとして扱える
- physical table欠落をExecution Errorとして区別できる
- physical parameter identityを保持できる
- parameter valueを通常Evidenceへ保存しない
- raw resultを明示mappingからSemantic OrderLineへ変換する
- Relation Storageの`order_id` support columnをSemantic attributeへ逆流させない
- Environment = `development`
- Database instance = `sales_core.dev`
- executor Capability = `database.query.postgres`
- `evidenceNature = real-postgres-ephemeral-fixture`
- fixture作成前にuser relation 0件 / Event Trigger 0件 / fixture object不在を確認できる
- cleanup後にuser relation 0件 / Event Trigger 0件 / fixture object不在へ戻ったことを確認できる

### 完了候補

- cleanな対象revisionでValidation成功を記録できる
- Evidence専用の空PostgreSQL Databaseでtoolが正常終了する
- `reproducibleRevision = true`である
- 既定Database、既存user relation、Event Triggerを含む接続先ではfixture作成前に停止する
- `fixtureDatabaseSafety`が期待安全条件を示す
- `fixtureCleanupSafety`がcleanup後に同じ空状態へ戻ったことを示す
- 2件 / 0件 / Execution Errorを同一runで区別できる
- Semantic mapping / support column非露出を確認できる
- credential / DSN / runtime parameter value / Database名をEvidenceへ含めない
- Evidence natureを他種別と区別できる
- 結果を#273 Gate B監査へ対応付けられる

## 継続Evidence — actual external `master_data` Oracle

関連: `ISSUE-000022` / #214、`ISSUE-000033` / #371、`ISSUE-000034` / #376

### 性質

これは**actual external asset Evidence**です。

Gate Bの必須完了条件ではありませんが、actual Oracle compatibility / external-observed asset integrationを主張するために必要です。

PR #288 / #300 / #306でEvidence intake boundaryとCanonical Workspace Validation fail-closedが成立しました。PR #340で、その受入れをclean Repository revisionへ固定し、どのexternal Evidence artifactを照合したかを機械可読なacceptance receiptへ残す境界も成立しています。

PR #364 / #366以降、Oracle Runnerはactual read時にdriverが実際に返したraw result列名と、driverが提供できた範囲の列metadataをOracle固有Payloadへ保持できます。

PR #372で具体driver / Host assemblyを#371へ、PR #378でread-only Evidenceを#376へ分離しました。

一方、第一Referenceで利用する実Oracle物理事実そのものはまだ未取得です。

### 必要な入力

TEI側で名前を推測せず、利用対象から少なくとも次を観測します。

- physical objectのkind / identity
- Contract input `id`へ対応するphysical parameter identity
- parameterの実型とNULL上の扱い
- raw result column identity
- raw columnの実型とNULL上の扱い
- raw resultから`customer.customer`の必要なSemantic attributeへの明示mapping
- native Oracle SQLまたは実際のrealization source
- 物理情報を確認したsource / provenance
- 匿名化・縮約等を行った場合の加工内容
- read-onlyを実効的に保証する境界とそのEvidence参照

### read-only Evidence — #376

`ownership = external-observed`という宣言、SQLが`SELECT`に見えること、TEI Runnerが`QueryContext`を使うことのいずれか一つだけをread-only Evidenceにはしません。

第一Referenceでは、実Oracle version / grant構成に応じて少なくとも次を分離して確認します。

- 専用のleast-privilege principal / roleと対象objectへのquery privilege
- system / object privilege、role、必要に応じてPUBLIC等を再確認できるgrant Evidence
- 採用する場合のread-only session / transaction境界
- write操作をReference read pathへ露出させないquery-only runtime境界
- actual Queryが上記境界の中で成功したこと

principal / grant Evidenceを主たる根拠とし、transactionやquery-only runtimeを追加防御として扱います。

read-only確認のためだけに、実外部assetへDML / DDL、`SELECT ... FOR UPDATE`、exclusive lock等を試行しません。

### actual Oracle driver / Host assembly — #371

具体driverの選択・登録、DSN / credential、TLS / TCPS、Wallet、connection設定はHost / Reference assembly側の責務として確定します。Canonical modelやOracle Provider Coreへ具体driverを固定しません。

実Oracle version / authentication / encryption / datatype / NULL / Host制約を確認してから採用driverを決定します。

### Evidence intakeとの関係

観測事実は非Canonical Evidenceとして保持し、別途明示authoringした`ContractQuerySQLRealization`と照合します。

`tools/reference-external-query-evidence`は、対応realizationがなければ停止し、EvidenceからSQL realizationを生成・補完しません。

acceptance receiptの成功はactual Oracle read、実効read-only principal、Semantic Customerの実取得を証明しません。

### actual read時のdriver観測

PR #364 / #366以降、actual Oracle read時には次の非secret情報をOracle Runner固有Payloadから追跡できます。

- 実際に`database/sql`へ渡したnamed parameterの名前
- Databaseから実際に返されたraw result列名
- driverが提供できた場合の`databaseTypeName`
- driverが提供できた場合の`nullable`
- driverが提供できた場合の`length`
- driverが提供できた場合の`precision` / `scale`

`database/sql.ColumnType`が提供しないmetadataをfalseや0へ補完しません。Go側の`ScanType`やraw row値も通常Evidenceへ複製しません。

このruntime metadataは、provenance付きexternal Evidenceの`PhysicalType` / `Nullable`を置き換える第二の物理正本ではありません。

### Gate B / Handoverとの関係

Gate B最終判定時に#214が未完了なら、Handoverへ#214 / #371 / #376をunresolvedとして明示します。

Gate B Readyをactual Oracle Readyへ読み替えず、Web側でOracle固有情報を補完しません。

#214をGate B後に完了させる場合も、既存Handover Baselineを黙って書き換えません。source / meaning変更がGate C baselineへ影響する場合はGate B Evidenceを再確認し、新しいbaselineを発行します。

## 保存してよいもの / 保存しないもの

通常Evidenceへ保存してよいのは、再現と監査に必要なidentity、結果分類、provenance参照、fingerprint、非機密metadataです。

次は原則として保存しません。

- password
- credentialを含むDSN
- token / secret
- 不要なruntime parameter value
- 業務上不要な実データ
- temporary fixture data
- driverが提供していないexternal物理metadataを推測した値

必要な監査証跡とsecretそのものを混同しません。

## CIとの関係

Gate Bは特定CIサービスのgreenをProduct Evidenceの代替にはしません。

現行RepositoryではGitHub Actionsを意図的に使用していません。将来再導入する場合には、通常Go tests、generated artifact再現性、PostgreSQL integration等を継続的に再実行するregression経路として扱います。

CI停止だけを理由に意味設計を未成立とは扱わず、逆に未実行をgreenとも記録しません。

## Evidence取得後

必須Evidenceを取得するたびに、少なくとも次を更新します。

1. 対応するRepository Issue / GitHub Issueの現在地
2. [`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)の条件表
3. Evidence種別
4. unresolved事項

事前確認でsourceやgenerated artifactの変更が必要になった場合は、必要な変更を確定・commitしてから最終判定用のcandidate revision Rを選びます。

Gate B監査を最終実行するのは、**#310 / #255 / #308の完了Evidenceが同じRへ揃ったとき**です。

Gate BがRについてReadyになった場合は[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md)に従ってRをHandover Baselineとして固定し、#214 actual Oracleのverified / unresolved状態を明示してから第一Web Targetへ進みます。

## 非目標

- この文書を第二の実行runbookにすること
- #244を実Provider Evidence待ちのために再び設計Issueへ戻すこと
- #255へPostgreSQL / Oracle / Driver runtime Evidenceを抱え込むこと
- generated artifact reproducibilityをdeterministic Repository / Driver Evidenceへ畳むこと
- Gate B前にWeb実装を開始すること
- CIをこの文書から再有効化すること
- external物理情報を推測すること
- synthetic / generated / fixture Evidenceをactual external asset Evidenceとして扱うこと
- acceptance receiptをactual Oracle read Evidenceとして扱うこと
- #214 / #371 / #376を削除すること
- #310 / #308の成功をOracle compatibility成功へ読み替えること
- read-only確認のため実外部assetへDML / DDL / `SELECT ... FOR UPDATE` / exclusive lockを試行すること
- driverが提供していないOracle raw列metadataをfalse / 0 / 慣習的型から推測すること
- 生成物やfingerprintを手編集して一致させること
- `-write`やsafe replaceの成功だけをgenerated artifact完了Evidenceとして採用すること
- source revisionが実行中に変化したrunを完了Evidenceとして採用すること
- cleanup後にuser relationまたはEvent Triggerが残るPostgreSQL runを成功Evidenceとして採用すること
- 異なるrevisionで得た#310 / #255 / #308成功Evidenceを、一つのReady revisionへ読み替えること
- 事前確認後にsourceが変わったのに、その前の成功JSONを最終完了Evidenceとして流用すること
- Gate Bのために汎用Workflow / Event Sourcing / Presentation IRを先行設計すること

## 関連

- [`GATE_B_EVIDENCE_RUNBOOK.md`](GATE_B_EVIDENCE_RUNBOOK.md): 実行順・具体コマンド・停止条件の正本
- [`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md): Gate B判定の正本
- [`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md): Gate B通過後のbaseline引き渡し
- `issue/000022-external-master-data-physical-query-evidence.md`
- `issue/000024-keyless-relation-physical-storage.md`
- `issue/000025-reference-generated-artifacts-refresh.md`
- `issue/000031-gate-b-postgres-header-detail-evidence.md`
- `issue/000032-gate-b-deterministic-repository-evidence.md`
- `issue/000033-gate-b-oracle-driver-integration.md`
- `issue/000034-gate-b-oracle-read-only-evidence.md`
- `issue/000035-gate-b-actual-oracle-blocker-reaudit.md`
- GitHub #214 / #244 / #255 / #273 / #291 / #308 / #310 / #371 / #376 / #381
- PR #331 / #335 / #336 / #340 / #341 / #346 / #351 / #354 / #355 / #356 / #364 / #365 / #366 / #367 / #369 / #370 / #372 / #373 / #374 / #378 / #379
