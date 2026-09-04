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
| real-provider ephemeral fixture Evidence | header-detail PostgreSQL tool | 実provider / driver / native SQLを通るruntime | production既存DBの事実、Oracle Evidence |
| actual external asset Evidence | `master_data` Oracle | TEI非所有資産の実物理・provenance・read-only境界 | synthetic / generated / ephemeral fixture |

一種類の成功を別種類のEvidenceへ格上げしません。

同じRepository checkoutとGo toolchainだけで再現できるという理由だけで、generated artifact reproducibilityをdeterministic Repository / Driver Evidenceへ畳みません。両者は「何を証明するか」が異なるため、Gate Bでも別laneとして判定します。

## Evidence A — external-observed `master_data`

関連: `ISSUE-000022` / GitHub Issue #214

### 性質

これは**actual external asset Evidence**です。

PR #288でEvidence intake boundaryは成立し、PR #304で収束線へ、PR #306で`main`へ、受入前にCanonical Workspaceの通常Validation Errorを拒否するfail-closed境界も成立しました。

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

`nativeSource.fingerprint`はraw fileのGit SHAではなく、Workspace Loaderが保持する実行SQL本文とのstale検知に利用します。独立したprovenanceは`nativeSource.evidenceRef` / `provenance.sourceRef`で保持します。

### 完了候補

少なくとも次を説明・再現できることを求めます。

- 実Oracle物理情報が観測sourceへ追跡できる
- 1件取得をSemantic Customerまで明示mappingできる
- 0件を正常なnot-foundとして保持できる
- 実行失敗をExecution Errorとして保持できる
- read-only強制をSQL文字列以外の実効境界から説明できる
- Environment / Database instance / executor Capability identityを保持できる
- credential / secret / 不要な実データを通常Evidenceへ保存していない

未観測の物理名をTEI側から補った場合は完了Evidenceとして採用しません。

## Evidence B — header-detail real PostgreSQL engine

実Provider Evidence: `ISSUE-000031` / GitHub Issue #308

成立済み設計: `ISSUE-000024` / GitHub Issue #244

### 責務分離

#244では、keyless Relation Storage、`find_by_order` SQL realization、Driver runtime、Evidence toolの安全境界まで成立しており、設計IssueとしてCompletedです。

**実PostgreSQL engine上でEvidenceを取得する作業だけを#308で追跡します。** 実Provider実行が未完了であることを理由に、Relation Storage設計を再びIn Progressへ戻しません。

### 性質

これは**real PostgreSQL engine + ephemeral fixture Evidence**です。

実PostgreSQL provider / driver / native SQL executionを通しますが、tool自身が作る一時fixtureを利用します。production既存Databaseの物理事実を観測するEvidenceではなく、Evidence AのOracle実外部資産を代替しません。

### 安全境界

PR #294 / #300 / #306 / #324で、安全境界を段階的に強化しています。

Evidence toolはfixture作成前に次を確認します。

```text
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
public.order_lines既存確認
  ↓ 既存なら停止
tool-owned fixture
  ↓
runtime Evidence
```

既存relationを削除してEvidence用Databaseへ転用する処理や、既定Databaseで続行するoverrideは持ちません。

このguardを通過したことだけでEvidence専用Databaseという運用意図を推測せず、実行先はEvidence専用に作成した空PostgreSQL Databaseへ限定します。

### 証明対象

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

### 完了候補

#308の完了条件として、少なくとも次を確認します。

- Validation成功revisionを記録できる
- Evidence専用の空PostgreSQL Databaseでtoolが正常終了する
- 既定Databaseや既存user relationを含む接続先ではfixture作成前に停止する
- 2件 / 0件 / Execution Errorを同一runで区別できる
- Semantic mapping / support column非露出を確認できる
- credential / DSN / runtime parameter valueをEvidenceへ含めない
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

### repairとの関係

`reference-golden -write` / `reference-durable-run-golden -write`はstale解消の**修復操作**です。

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
- 結果を#273 Gate B監査へ対応付けられる

具体的なcheck / repair順序、差分確認は`GATE_B_EVIDENCE_RUNBOOK.md`を正とします。

## Evidence D — UI-less Scenario / Evidence境界の通常Go runner

関連: `ISSUE-000032` / GitHub Issue #310

### 性質

これは**deterministic repository Evidence**です。

最終UI-less Scenarioはsynthetic adapterを含みます。通常Go runnerで成功しても実Oracle / 実PostgreSQL / 実外部Command Evidenceへ格上げしません。

### 正規Evidence

PR #315 / #321で`tools/reference-gate-b-repository-evidence`を整備しました。

runnerはWorkspace Validation、最終UI-less Scenario、external Evidence intake testsを一つのrunで扱い、実行前後のrepository状態も確認します。

Gate Bへ採用するEvidenceには少なくとも次を要求します。

- 実行開始時にworking tree clean
- 実行前repository revisionを記録する
- Workspace Validationが成功する
- targeted Scenario / Evidence intake checksが成功する
- 実行後repository revisionを再確認する
- 実行前後でHEADが同一
- 実行後もworking tree clean
- `reproducibleRevision = true`

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

加えて、external Evidence intakeのunit/tool testsを通常runnerで実行し、PR #288 / #304 / #306で成立した受入・fail-closed境界がコードとして再現することを確認します。

### 完了候補

- `reference-gate-b-repository-evidence`が正常終了する
- `reproducibleRevision = true`である
- 最終Scenario testが通常Go runnerで成功する
- external Evidence intakeの関連testsが成功する
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

四laneのEvidenceが揃った時点でGate B監査を再実行します。

Gate BがReadyになった場合は[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md)に従ってHandover Baselineを固定し、その後に第一Web Targetへ進みます。

## 非目標

- この文書を第二の実行runbookにすること
- #244を実Provider Evidence待ちのために再び設計Issueへ戻すこと
- #255へPostgreSQL / Oracle / Driver runtime Evidenceを抱え込むこと
- generated artifact reproducibilityをdeterministic Repository / Driver Evidenceへ畳むこと
- Gate B前にWeb実装を開始すること
- CIをこの文書から再有効化すること
- external物理情報を推測すること
- synthetic / generated / fixture Evidenceをactual external asset Evidenceとして扱うこと
- 生成物やfingerprintを手編集して一致させること
- `-write`直後のdirty treeをgenerated artifact完了Evidenceとして採用すること
- source revisionが実行中に変化したrunを完了Evidenceとして採用すること
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
- GitHub #214 / #244 / #255 / #273 / #308 / #310
