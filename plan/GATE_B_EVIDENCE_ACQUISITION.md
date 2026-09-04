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

2026-09-04時点のGate B残件は、主に次の四系統です。

1. external-observed `master_data`のactual Oracle物理 / read-only Evidence
2. header-detailの実PostgreSQL engine Evidence
3. Reference World / Plugin Lock / Durable Runの正規再生成Evidence
4. 最終UI-less ScenarioとEvidence受入境界の通常Go runner実行Evidence

GitHub Actionsの再開・継続実行は重要ですが、上記Evidenceそのものとは分けます。Actions停止をSemantic design gapとは扱わず、未実行をgreenとも扱いません。

## Evidenceの分類

Gate BではEvidenceを少なくとも三種類に分けます。

| 種類 | 代表例 | 証明するもの | 代替しないもの |
| --- | --- | --- | --- |
| deterministic repository Evidence | Workspace Validation、Go test、Reference golden再生成 | Repository内のauthoring・runtime境界・生成再現性 | 実外部資産の物理事実 |
| real-provider fixture Evidence | header-detail PostgreSQL tool | 実provider / driver / native SQLを通るruntime | production既存DBの事実、Oracle Evidence |
| actual external asset Evidence | `master_data` Oracle | TEI非所有資産の実物理・provenance・read-only境界 | synthetic / ephemeral fixture |

一種類の成功を別種類のEvidenceへ格上げしません。

## Evidence A — external-observed `master_data`

関連: `ISSUE-000022` / GitHub Issue #214

### 性質

これは**actual external asset Evidence**です。

PR #288でEvidence intake boundaryは成立し、PR #304で受入前にCanonical Workspaceの通常Validation Errorを拒否するfail-closed境界も成立しました。

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
- credential / secret /不要な実データを通常Evidenceへ保存していない

未観測の物理名をTEI側から補った場合は完了Evidenceとして採用しません。

## Evidence B — header-detail実PostgreSQL

関連: `ISSUE-000024` / GitHub Issue #244

### 性質

これは**real PostgreSQL engine + ephemeral fixture Evidence**です。

実PostgreSQL provider / driver / native SQL executionを通しますが、tool自身が作る一時fixtureを利用します。production既存Databaseの物理事実を観測するEvidenceではなく、Evidence AのOracle実外部資産を代替しません。

### 安全境界

PR #294以降、既存`public.order_lines`が存在する接続先ではfail closedで停止し、既存tableをfixtureへ置換するoverrideを持ちません。

PR #304以降はさらに、**Databaseへ接続する前に**`reference/workspace`を読み込み、通常`validation.Validate`のErrorがあれば停止します。

したがって順序上の契約は次です。

```text
Workspace Load
  ↓
Canonical Validation
  ↓ Errorなら停止
PostgreSQL接続
  ↓
既存fixture guard
  ↓
tool-owned fixture
  ↓
runtime Evidence
```

### 証明対象

- `order_id=1001`で複数件を正常collectionとして取得できる
- `order_id=9999`で0件を正常な空collectionとして扱える
- physical table欠落をExecution Errorとして区別できる
- physical parameter identityを保持できる
- parameter valueを通常Evidenceへ保存しない
- raw resultを明示mappingからSemantic OrderLineへ変換する
- Relation Storageの`order_id` support columnをSemantic attributeへ逆流させない
- Environment = `development`
- Database instance = `sales_core.dev`
- executor Capability = `database.query.postgres`

### 完了候補

- Evidence専用の空PostgreSQL Databaseでtoolが正常終了する
- JSON Evidenceを再確認できる
- 2件 / 0件 / Execution Errorを一runで区別できる
- credential / DSN / runtime parameter valueをEvidenceへ含めない
- `evidenceNature = real-postgres-ephemeral-fixture`として他Evidence種別と区別できる

## Evidence C — Reference artifact再生成

関連: `ISSUE-000025` / GitHub Issue #255

### 性質

これは**deterministic repository Evidence**です。

Canonical Workspaceから派生する生成物をTEI自身の正規経路で再現できることを確認します。

対象は少なくとも次です。

- `reference/world.json`
- `reference/workspace/.tei/plugin-lock.json`
- `reference/order_rollout.durable-run.m0.21.start.json`

`reference/observed/*`やlegacy-auth fixtureは、Canonical authoring変更だけを理由に自動更新しません。

### 証明対象

- Worldを最新Canonical Workspaceから再生成できる
- Plugin Lockを同じWorkspaceから再生成できる
- 保存Durable RunのPlugin Lock依存を正規`deployment-start`から再現できる
- 差分が現在のauthoring / Plugin Lock / fixed observed fixtureから説明できる
- fingerprintや生成JSONを人手で合わせていない

### 完了候補

- 正規生成後に`reference-golden`のcheckが一致する
- artifact差分を意味上説明できる
- external-observed `master_data`の未観測物理情報が生成物へ紛れ込んでいない
- observed fixtureを生成結果へ合わせて書き換えていない

具体的な安全な再生成順序、Durable Runの一時file利用、差分確認は`GATE_B_EVIDENCE_RUNBOOK.md`を正とします。

## Evidence D — UI-less Scenario / Evidence境界の通常Go runner

### 性質

これは**deterministic repository Evidence**です。

最終UI-less Scenarioはsynthetic adapterを含みます。通常Go runnerで成功しても実Oracle / 実PostgreSQL / 実外部Command Evidenceへ格上げしません。

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

加えて、external Evidence intakeのunit/tool testsを通常runnerで実行し、PR #288 / #304の受入境界がコードとして再現することを確認します。

### 完了候補

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

GitHub Actionsが再開された場合には、通常Go tests、Reference golden再現性、PostgreSQL integration等を継続的に再実行するregression経路として利用します。

CI停止だけを理由に意味設計を未成立とは扱わず、逆に未実行をgreenとも記録しません。

## Evidence取得後

Evidenceを取得するたびに、少なくとも次を更新します。

1. 対応するRepository Issue / GitHub Issueの現在地
2. [`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)の条件表
3. Evidence種別（repository / provider fixture / actual external asset）
4. unresolved事項

四系統のEvidenceが揃った時点でGate B監査を再実行します。

Gate BがReadyになった場合は[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md)に従ってHandover Baselineを固定し、その後に第一Web Targetへ進みます。

## 非目標

- この文書を第二の実行runbookにすること
- Gate B前にWeb実装を開始すること
- CIをこの文書から再有効化すること
- external物理情報を推測すること
- synthetic / fixture Evidenceをactual external asset Evidenceとして扱うこと
- 生成物やfingerprintを手編集して一致させること
- Gate Bのために汎用Workflow / Event Sourcing / Presentation IRを先行設計すること

## 関連

- [`GATE_B_EVIDENCE_RUNBOOK.md`](GATE_B_EVIDENCE_RUNBOOK.md): 実行順・具体コマンド・停止条件の正本
- [`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md): Gate B判定の正本
- [`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md): Gate B通過後のbaseline引き渡し
- `issue/000022-external-master-data-physical-query-evidence.md`
- `issue/000024-keyless-relation-physical-storage.md`
- `issue/000025-reference-generated-artifacts-refresh.md`
- GitHub #214 / #244 / #255 / #273
