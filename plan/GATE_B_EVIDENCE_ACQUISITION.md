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

### 2026-09-04 現行Gate B scope（ISSUE-000035 / #380）

`ISSUE-000035` / #380の再監査により、Gate B Ready判定へ必須とするsource-dependent実行Evidenceは次の三つです。

1. `ISSUE-000032` / #310 — deterministic Repository Evidence
2. `ISSUE-000025` / #255 — generated artifact reproducibility Evidence
3. `ISSUE-000031` / #308 — real PostgreSQL Provider Evidence

最終Ready判定では、この三つの完了Evidenceを同じclean candidate revision `R`へ揃えます。各runnerが要求する`reproducibleRevision = true`等のsource固定条件は従来どおり維持します。

`ISSUE-000022` / #214、`ISSUE-000033` / #371、`ISSUE-000034` / #376のactual Oracle EvidenceはGate B必須blockerではありません。ただしactual external asset EvidenceというEvidence種別自体を削除せず、legacy / external asset実証と第二Database Provider圧力試験として継続します。

Gate Bではexternal-observed assetについて、少なくとも次の境界を引き続き必須とします。

- 未観測のphysical object / column / SQLを推測しない
- external EvidenceをCanonicalへ昇格させない
- Evidenceから不足realizationを生成・補完しない
- 明示authoringとEvidenceを照合する
- synthetic / generated / PostgreSQL fixture Evidenceをactual Oracle Evidenceへ格上げしない

これらの回帰は#310の`external-evidence-intake`等で確認します。

#214が未完了のままGate Bを通過する場合、#291 Handover Baselineへ#214 / #371 / #376と未観測事項を`unresolved`として明示します。第一Web Target側でOracle物理情報やread-only方式を補完しません。

**Gate Bはこのscope変更だけではReadyになりません。** #310 / #255 / #308のactual run Evidenceは未取得です。

以下の「四laneをGate B必須とする」記述は`ISSUE-000035`以前のEvidence契約と判断経緯を保持するための履歴です。現行Gate B必須範囲と矛盾する場合は、本節を優先します。一方、各Evidence種別の証明範囲、非代替性、安全条件は引き続き有効です。

Application Driverの主要意味境界、最終UI-less Scenario構造、external-observed Query Evidenceの受入境界、header-detail PostgreSQL fixtureの安全境界は成立しています。

2026-09-04時点のGate B残件は、次の四laneです。

1. `ISSUE-000022` / #214 — external-observed `master_data`のactual Oracle物理 / provenance / read-only Evidence
2. `ISSUE-000031` / #308 — header-detailのreal PostgreSQL engine + ephemeral fixture Evidence
3. `ISSUE-000025` / #255 — Reference World / Plugin Lock / Durable Runのgenerated artifact reproducibility Evidence
4. `ISSUE-000032` / #310 — 最終UI-less ScenarioとEvidence受入境界のdeterministic Repository / normal Go runner Evidence

GitHub Actionsの再開・継続実行は重要ですが、上記Evidenceそのものとは分けます。Actions停止をSemantic design gapとは扱わず、未実行をgreenとも扱いません。

## Evidenceの分類

Gate BではEvidenceを少なくとも四種類に分けます。

| 種類 | 代表例 | 証明するもの | 代替しないもの |
| --- | --- | --- | --- |
| deterministic repository Evidence | Workspace Validation、UI-less Scenario、external Evidence intake tests | Repository内のauthoring・Driver / Evidence境界が固定revisionで再現すること | generated artifact再現性、実provider、実外部資産 |
| generated artifact reproducibility Evidence | World / Plugin Lock / Durable Run golden | 正規入力から保存generated artifactをclean revisionで再現できること | Driver Scenario、実provider、実外部資産 |
| real-provider ephemeral fixture Evidence | header-detail PostgreSQL tool | 実provider / driver / native SQLを通るruntimeとfixture前後の安全状態 | production既存DBの事実、Oracle Evidence |
| actual external asset Evidence | `master_data` Oracle | TEI非所有資産の実物理・provenance・read-only境界 | synthetic / generated / ephemeral fixture |

一種類の成功を別種類のEvidenceへ格上げしません。

同じRepository checkoutとGo toolchainだけで再現できるという理由だけで、generated artifact reproducibilityをdeterministic Repository / Driver Evidenceへ畳みません。両者は「何を証明するか」が異なるため、Gate Bでも別laneとして判定します。

## 最終Ready判定では4 laneを同じrevisionへ揃える

四種類のEvidenceがそれぞれ一度でも成功すればGate BをReadyにできる、という意味ではありません。

Gate Bの最終Ready判定では、一つのcleanなcandidate repository revision `R`を定め、**完了判定に用いる4 laneのEvidenceをすべてRへ揃えます**。

少なくとも次を要求します。

- #214 acceptance receiptの`repositoryRevision = R`かつ`reproducibleRevision = true`
- #214 actual Oracle readがRのruntime / realizationに対して行われている
- #308 Provider Evidenceの`repositoryRevision = R`かつ`reproducibleRevision = true`
- #255 generated artifact Evidenceの`repositoryRevision = R`かつ`reproducibleRevision = true`
- #310 deterministic Repository Evidenceの`repositoryRevision = R`かつ`reproducibleRevision = true`
- `GATE_B_READINESS_AUDIT.md`がRを対象にReady / Not Readyを判定している

外部観測資料そのものはRより前に取得されていて構いません。ただし、その資料を現在のauthoringへ照合する#214 acceptance receiptとactual readはRへ結び直します。

Rと異なるrevisionでしか成功していないlaneは、「意味は変わっていないはずだ」と推測して流用せず、Rで再実行・再受入れします。

実行順はRunbookの二段階手順を正とします。事前確認の後でsourceやgenerated artifactが変わった場合、それ以前の成功JSONを最終完了Evidenceへ読み替えません。

## Evidence A — external-observed `master_data`

関連: `ISSUE-000022` / GitHub Issue #214

### 性質

これは**actual external asset Evidence**です。

PR #288 / #300 / #306でEvidence intake boundaryとCanonical Workspace Validation fail-closedが成立しました。PR #340で、その受入れをclean Repository revisionへ固定し、**どのexternal Evidence artifactを照合したか**を機械可読なacceptance receiptへ残す境界も成立しています。

PR #364 / #366以降、Oracle Runnerはactual read時にdriverが実際に返したraw result列名と、driverが提供できた範囲の列metadataをOracle固有Payloadへ保持できます。これは外部物理事実を取得済みという意味ではなく、actual readで観測した情報をSemantic mapping後に失わないためのruntime境界です。

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

### read-only Evidence

次のような、実際にwriteを防ぐ境界を確認します。

- read-only principal / credential / database role
- Database permission
- read-only session / transaction
- write操作を公開しないadapter / API
- 外部側で強制される同等の制御

`ownership = external-observed`という宣言や、SQLが`SELECT`に見えることだけをread-only Evidenceにはしません。

### Evidence intakeとの関係

観測事実は非Canonical Evidenceとして保持し、別途明示authoringした`ContractQuerySQLRealization`と照合します。

`tools/reference-external-query-evidence`は、対応realizationがなければ停止し、EvidenceからSQL realizationを生成・補完しません。

Gate Bへ採用する受入れでは、Evidence JSONを読む前にclean Repository revisionを要求し、照合後にもHEAD / working treeを再確認します。実行中にsourceが変化したrunは受理しません。

成功時は次の性質を持つacceptance receiptを出します。

- `evidenceNature = external-query-evidence-acceptance`
- `repositoryRevision`
- `reproducibleRevision = true`
- post-run Repository state
- `evidenceArtifactFingerprint = sha256:<hex>`
- target / provider
- `accepted = true`

`evidenceArtifactFingerprint`は、toolが実際に読み込んだEvidence JSONの**exact bytes**を識別します。

`nativeSource.fingerprint`は、Workspace Loaderが保持する実行SQL本文とのstale検知に利用します。両fingerprintは役割が異なり、相互代替しません。独立したprovenanceは`nativeSource.evidenceRef` / `provenance.sourceRef`で保持します。

acceptance receiptへphysical object、parameter、raw columns、provenance本文、read-only mechanism本文、Evidence file path、credential / secretを複製しません。

receiptの成功は**external Evidence artifactと明示authoringの受入照合**だけを証明します。actual Oracle read、実効read-only principal、Semantic Customerの実取得を証明しません。

### actual read時のdriver観測

PR #364 / #366以降、actual Oracle read時には次の非secret情報をOracle Runner固有Payloadから追跡できます。

- 実際に`database/sql`へ渡したnamed parameterの名前
- Databaseから実際に返されたraw result列名
- driverが提供できた場合の`databaseTypeName`
- driverが提供できた場合の`nullable`
- driverが提供できた場合の`length`
- driverが提供できた場合の`precision` / `scale`

`database/sql.ColumnType`が提供しないmetadataをfalseや0へ補完しません。Go側の`ScanType`はexternal Oracleのphysical typeそのものではないため通常Evidenceへ保存せず、raw row値もOracle固有Payloadへ複製しません。

このruntime metadataは、provenance付きexternal Evidenceの`PhysicalType` / `Nullable`を置き換える第二の物理正本ではありません。外部観測事実とactual read時のdriver挙動を対応付ける材料です。

実Oracle / 採用driverを観測する前に、external Evidenceの`PhysicalType`とdriverの`DatabaseTypeName`が同じ文字列表現になると仮定せず、自動文字列一致規則を先行固定しません。

### 完了候補

少なくとも次を説明・再現できることを求めます。

- 実Oracle物理情報が観測sourceへ追跡できる
- 受入れたEvidence artifactをexact-bytes fingerprintで特定できる
- acceptance receiptを同じclean Repository revisionへ固定できる
- actual readで返ったraw result列名と、driverが提供できたmetadataを観測できる
- driverが提供していない型 / NULL metadataを推測していない
- 1件取得をSemantic Customerまで明示mappingできる
- 0件を正常なnot-foundとして保持できる
- 実行失敗をExecution Errorとして保持できる
- read-only強制をSQL文字列以外の実効境界から説明できる
- Environment / Database instance / executor Capability identityを保持できる
- credential / secret / 不要な実データを通常Evidenceへ保存していない

未観測の物理名をTEI側から補った場合は完了Evidenceとして採用しません。acceptance receiptだけでactual readを完了扱いにもしません。

## Evidence B — header-detail real PostgreSQL engine

実Provider Evidence: `ISSUE-000031` / GitHub Issue #308

成立済み設計: `ISSUE-000024` / GitHub Issue #244

### 責務分離

#244では、keyless Relation Storage、`find_by_order` SQL realization、Driver runtime、Evidence toolの安全境界まで成立しており、設計IssueとしてCompletedです。

**実PostgreSQL engine上でEvidenceを取得する作業だけを#308で追跡します。** 実Provider実行が未完了であることを理由に、Relation Storage設計を再びIn Progressへ戻しません。

### 性質

これは**real PostgreSQL engine + ephemeral fixture Evidence**です。

実PostgreSQL provider / driver / native SQL executionを通しますが、tool自身が作る一時fixtureを利用します。production既存Databaseの物理事実を観測するEvidenceではなく、Evidence AのOracle実外部資産を代替しません。

### Repository source固定

PR #336で、#308も#310 / #255と同じ`tools/internal/repositoryevidence`へsource観測規則を揃えました。

- Workspace Load / DB接続より前にclean Repository revisionを要求する
- Provider runtime、fixture cleanup、Database復元確認後にHEAD / working treeを再確認する
- 実行前後で同じHEADかつcleanでなければ成功Evidenceを出さない
- `repositoryRevision` / `reproducibleRevision` / post-run Repository stateをEvidenceへ残す

PR #355以降、post-runの`Checked = true`は、実行後repository revisionとworking tree状態の両方を観測できた後にだけ設定されます。途中で観測に失敗した状態と、「観測した結果cleanではなかった」状態を区別します。

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

#308の完了条件として、少なくとも次を確認します。

- cleanな対象revisionでValidation成功を記録できる
- Evidence専用の空PostgreSQL Databaseでtoolが正常終了する
- `reproducibleRevision = true`である
- 既定Database、既存user relation、Event Triggerを含む接続先ではfixture作成前に停止する
- `fixtureDatabaseSafety`がuser relation 0件 / Event Trigger 0件 / fixture object不在を含む期待安全条件を示す
- `fixtureCleanupSafety`がcleanup後に同じ空状態へ戻ったことを示す
- 2件 / 0件 / Execution Errorを同一runで区別できる
- Semantic mapping / support column非露出を確認できる
- credential / DSN / runtime parameter value / Database名をEvidenceへ含めない
- Evidence natureを他種別と区別できる
- 結果を#273 Gate B監査へ対応付けられる

## Evidence C — generated Reference artifact reproducibility

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

## Evidence D — UI-less Scenario / Evidence境界の通常Go runner

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

Test自体が成功しても、実行中にsource revisionが変化したrunや実行後に差分が残るrunは完了Evidenceへ採用しません。

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

加えて、external Evidence intakeとRepository source固定境界のConformanceを通常runnerで実行し、成立済みの受入・fail-closed境界がコードとして再現することを確認します。

### 完了候補

- `reference-gate-b-repository-evidence`が正常終了する
- `reproducibleRevision = true`である
- 最終Scenario testが通常Go runnerで成功する
- external Evidence intakeの関連testsが成功する
- Repository source-boundary Conformanceが成功する
- repository revisionを記録できる
- synthetic Evidenceであることを明示できる
- 実外部Evidenceとの区別を維持する

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

Evidenceを取得するたびに、少なくとも次を更新します。

1. 対応するRepository Issue / GitHub Issueの現在地
2. [`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)の条件表
3. Evidence種別（deterministic repository / generated artifact reproducibility / provider fixture / actual external asset）
4. unresolved事項

事前確認でsourceやgenerated artifactの変更が必要になった場合は、必要な変更を確定・commitしてから最終判定用のcandidate revision Rを選びます。

Gate B監査を最終実行するのは、単に四laneの成功履歴が存在するときではなく、**四laneの完了判定に用いるEvidenceがすべて同じRへ揃ったとき**です。

Gate BがRについてReadyになった場合は[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md)に従ってRをHandover Baselineとして固定し、その後に第一Web Targetへ進みます。

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
- driverが提供していないOracle raw列metadataをfalse / 0 / 慣習的型から推測すること
- 実Oracle観測前にexternal Evidenceの`PhysicalType`とdriverの`DatabaseTypeName`の文字列一致規則を固定すること
- 生成物やfingerprintを手編集して一致させること
- `-write`やsafe replaceの成功だけをgenerated artifact完了Evidenceとして採用すること
- source revisionが実行中に変化したrunを完了Evidenceとして採用すること
- cleanup後にuser relationまたはEvent Triggerが残るPostgreSQL runを成功Evidenceとして採用すること
- 異なるrevisionで得た4 laneの成功Evidenceを、一つのReady revisionへ読み替えること
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
- GitHub #214 / #244 / #255 / #273 / #291 / #308 / #310
- PR #331 / #335 / #336 / #340 / #341 / #346 / #351 / #354 / #355 / #356 / #364 / #365 / #366 / #367 / #369 / #370 / #373
