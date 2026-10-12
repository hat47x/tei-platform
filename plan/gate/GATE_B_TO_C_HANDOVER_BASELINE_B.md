# Gate B → Gate C Handover Baseline B

- 引継ぎ基準版の識別情報：`B`
- `baselineRepositoryRevision R`: `e5231839a5dcc62cce1181a2731fcb7c70d16736`
- 固定日：2026年9月19日
- 固定した根拠： [`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)（判定日2026-09-19、Ready）
- 規則と原則の正本： [`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md)
- リポジトリの課題：ISSUE-000030-gate-b-to-c-handover-baseline.md／GitHub #291

この文書には、`GATE_B_TO_C_HANDOVER_CONTRACT.md`の規則に従って実際に固定した、最初の引継ぎ基準版を記録します。契約文書そのものは変更せず、この記録を特定の版に対応する実例とします。引継ぎ記録は、新たな正本のスキーマでも第二の正本でもありません。安定した参照先、識別情報、照合値から実際の検証根拠を追跡するために、それらをまとめたものです。

## Baseline identity

- 引継ぎ基準版の識別情報：`B`
- `baselineRepositoryRevision = R = e5231839a5dcc62cce1181a2731fcb7c70d16736`（`hat47x/tei-platform`の`main`で、PR [#726](https://github.com/hat47x/tei-platform/pull/726)とPR [#727](https://github.com/hat47x/tei-platform/pull/727)をsquash mergeした後のHEAD）
- 参照用作業領域の識別情報：`reference/workspace`（`go run ./cmd/tei validate -workspace reference/workspace` = `ok`。既知の情報通知3件：`APPLICATION_RELEASE_CONTRACT_NOT_CURRENT` x1／`DATABASE_RELEASE_CONTRACT_NOT_CURRENT` x2。`sales_core.order_legacy`の過去の契約が未定義であることに関するもので、判定段階BのReady判定を妨げない）
- 判定段階Bの合格監査の対象版：`R`と同一（[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)参照）

## Canonical Application baseline（manifest）

最初の参照事例から追跡できる識別情報を列挙します。意味の本文は正本となる入力元に置き、この記録には複製しません。

- Application: `order_entry`
- Application Pattern: browse / record / reference / header-detail / command
- Data / Relation / Query: `reference/workspace/semantic/**`、`reference/workspace/database/**`（`sales_core`ほか）
- データ接続先：読取り専用と読書き可能の境界を持つ、論理的なデータ接続先の定義
- 検証：正本上の検証規則（`internal/validation`系）
- 認証・認可：`order_entry`のドライバー内で、認証から認可へ至る境界
- 機能と対応付け：契約に基づく問い合わせの機能の対応付け（`internal/realization/contract_query.go`の`ContractQueryTarget.Identity()` = `Database/Contract/Query`の三つ組）
- 業務上の結果と名前付きの結果：`internal/driver`での業務上の結果と実行時エラーの区別
- 実行時エラー：提供実装の元のエラーから、意味上の実行時エラーへの明示的な対応付け
- 観測で使用する明示的な読取りの境界：命令の受理応答とは分離した観測のための読取り
- 入力元と対応付けの追跡：`internal/workspace/source_index.go`（入力元索引、Source Index）

## Driver Scenario baseline

```text
Authentication
 -> browse
 -> record
 -> reference Customer
 -> header-detail OrderLine[]
 -> Canonical Validation
 -> Authorization
 -> Command Business Result
 -> separate Observation read
```

再現対象：`internal/driver`の`TestOrderEntryGateBScenarioConnectsMajorBoundariesWithoutUI`（同じ版で、`#310`の画面を使わない最終シナリオの確認結果が`passed = true`）。

業務上の結果、名前付きの結果、実行時エラー、観測、診断結果を、実行先で一つの種類にまとめません。

## Datasource / authority boundary

- 読取り専用・読書き可能の境界：`sales_core`データベースに対する、契約に基づく問い合わせ経由の読取り
- 所有・管理上の性質：`#308`の検証根拠にある`fixtureOwnedByTool = true`／`preexistingFixtureProtected = true`が示す、ツール所有の一時的な試験用構成要素と本番の既存データベースの区別
- データベースの実体と実行器の機能の対応付け：`databaseInstance = sales_core.dev`、`executorCapability = database.query.postgres`
- 秘密情報・認証情報・実行時の引数を通常の検証根拠に保存しない制約：同じ実行で得た子処理の成果物すべてについて、目視確認と機械監査（`reference-gate-b-readiness-artifact-audit`の機密項目を拒否する確認）を実施済み

## Gate B Evidence provenance

### #387 final convergence

```text
evidenceNature            = gate-b-three-lane-final-convergence
repositoryRevision        = e5231839a5dcc62cce1181a2731fcb7c70d16736
workingTreeClean          = true
reproducibleRevision      = true
allLanesSameRevision      = true
childArtifactsRetained    = true
postRunRepository.checked         = true
postRunRepository.revisionStable  = true
postRunRepository.workingTreeClean = true
goEnvironment.externalWorkspaceActive = false
goEnvironment.goFlagsEmpty            = true
goEnvironment.goEnvDisabled           = true
```

実行環境：判定段階Bの検証専用にリポジトリ全体を取得（`git status --short --untracked-files=all --ignored=matching`が空、`git ls-files -v`の全項目が`H`）。`GOENV=off`／`GOWORK=off`／`GOFLAGS=`を指定し、`TEI_TEST_POSTGRES_DSN_FILE`でリポジトリ外の秘密情報ファイルを参照しました。実行手順は[`GATE_B_EVIDENCE_RUNBOOK.md`](GATE_B_EVIDENCE_RUNBOOK.md)を正本とし、この記録には複製しません。

### same-run child exact artifacts

以下は`#387`が実際に読み取り、リポジトリ外の所有者だけがアクセスできるディレクトリに保存した、子処理の成果物3件です。ファイル名は固定しています。

| lane | evidenceNature | artifactFile | evidenceArtifactFingerprint (SHA-256) |
| --- | --- | --- | --- |
| #310 | `deterministic-repository` | `deterministic-repository.json` | `95074b13665c30bbfa8acb644f96f21e039180de771055f3ce7fa0e2c3ecf09e` |
| #255 | `generated-artifact-reproducibility` | `generated-artifact-reproducibility.json` | `83b780957cdf7edd432e6e5780118eb3048c86a72de6a8d7d0b644d0c35fc847` |
| #308 | `real-postgres-ephemeral-fixture` | `real-postgres-ephemeral-fixture.json` | `fb57ab65519fc17541a1e790c5040361c43a7883ebd68339fbe218a24c8e8a1a` |

各子処理の記録には、`repositoryRevision = e5231839a5dcc62cce1181a2731fcb7c70d16736`と`reproducibleRevision = true`が設定されています。

成果物の本文は実行手順書の規則に従ってリポジトリの外だけに保存し、リポジトリにはコミットしません。安定した参照には、この照合値（SHA-256）と、同じ`R`について`GATE_B_EVIDENCE_RUNBOOK.md`の手順を再実行することで再生成できる性質を用います（`reproducibleRevision = true`がその根拠です）。

**保管先（locator）**：GitHub Release [`gate-b-evidence-R-e523183`](https://github.com/hat47x/tei-platform/releases/tag/gate-b-evidence-R-e523183)（タグは`R`を指す）の成果物`tei-gate-b-evidence-R-e5231839.tar.gz`。SHA-256は`1af53d6ec1d7d61427d98cd0d8fda655fe0c37bb53927f36440d5e17cfec4395`です。展開後、`MANIFEST.sha256`で各ファイルを再照合できます（`ISSUE-000057`、2026年9月20日追記）。

#### #310 deterministic Repository（same-run）

- Workspace Validation成功
- `final-ui-less-scenario`成功（`TestOrderEntryGateBScenarioConnectsMajorBoundariesWithoutUI`）
- `external-evidence-intake`成功
- `repository-evidence-source-boundary`成功

#### #255 generated artifact reproducibility（same-run）

- World / Plugin Lock golden一致（`tools/reference-golden`）
- Durable Run golden一致（`tools/reference-durable-run-golden`）
- check-only run（`-write`修復操作は本runに含まない）

#### #308 real PostgreSQL Provider（same-run）

- fixture前: `userRelationsBeforeFixture = 0` / `eventTriggersBeforeFixture = 0` / `fixtureObjectAbsentBeforeCreate = true` / `nonDefaultDatabaseVerified = true`
- runtime: 2件取得 / 正常0件 / missing-table `executionErrorObserved = true`
- explicit raw → Semantic mapping: `contractQueryTarget = sales_core/sales_core.order_line/find_by_order`、`semanticAttributes`に support column非露出
- cleanup後: `userRelationsAfterCleanup = 0` / `eventTriggersAfterCleanup = 0` / `fixtureObjectAbsentAfterCleanup = true` / `fixtureCleanupSafety.verified = true`
- DSN、credential、parameter値、実Database名は本Evidenceに含まない

### #273 Gate B Readiness Audit

- 判定: Ready（[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)、判定日2026-09-19）
- 独立artifact audit: `tools/reference-gate-b-readiness-artifact-audit`
  - `artifactAuditPassed = true`
  - `finalFingerprint = sha256:3bfcdc459f9e3dd77019e1b9ae368a64bad760f65917c44e1d7d533b6a1f5f97`
  - 3 lane個別fingerprintは上表と一致（独立tool側の再計算値であり、final runner側の`evidenceArtifactFingerprint`と一致することを確認済み）

final runner exit codeや`childArtifactsRetained = true`だけでReadyを宣言したものではなく、上記独立auditの再照合結果を判定根拠とします。

## actual Oracleの扱い（unresolved）

`#214` / `#371` / `#376`のactual Oracle Evidence（physical object / column / SQL / datatype / driver behavior / real read-only方式のactual read）は、本Baselineの時点で**未取得（unresolved）**です。

- Required Evidence集合（#310 / #255 / #308）には含まれません
- `#310`のsynthetic Evidenceおよび`#308`のPostgreSQL ephemeral fixture Evidenceを、actual Oracle Evidenceへ格上げしていません
- 第一Web Target側でphysical object / column / SQL / datatype / driver behavior / read-only方式を推測・補完しません
- 取得が必要になった場合は、別途actual Oracle Evidenceとして取得し、本Baselineまたはsuccessor Baselineへ追加参照します

## Gate C開始条件との対応

[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md)の「Gate C開始条件」を本Baselineに対して確認します。

- [x] Gate B Ready
- [x] Handover Baseline `B / R`を固定（本記録）
- [x] #310 / #255 / #308 same-run Required EvidenceがRへ揃っている
- [x] #387 final convergenceとsame-run child artifactsへ追跡できる
- [x] #273が同じRをReadyと判定
- [x] Gate C対象Canonical identityを列挙できる（上記「Canonical Application baseline」manifest）
- [x] Driver Scenario baselineを再現できる（上記「Driver Scenario baseline」）
- [x] actual Oracle等のexternal Evidenceをverified / unresolvedとして明示できる（上記「actual Oracleの扱い」= unresolved）
- [x] unresolvedを明示できる（同上）

第一Web Target（`ISSUE-000007` / GitHub #70）は本Baseline `B / R`を比較対象とし、Target build identity `T1`を`R`と分離して追跡します。

## baseline変更時

Target実装中にCanonical / Driver meaningの変更が必要になった場合、本`B / R`へ黙って継続せず、`GATE_B_TO_C_HANDOVER_CONTRACT.md`の「meaning変更時はsuccessor Handover」手順に従い、新しいrevision `R2`でGate B Evidenceを再取得し、`#273`で再監査したうえでsuccessor Handover `B2 / R2`を発行します。

## 関連

- `GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `GATE_B_READINESS_AUDIT.md`
- `GATE_B_EVIDENCE_ACQUISITION.md`
- `GATE_B_EVIDENCE_RUNBOOK.md`
- `GATE_C_EVIDENCE_CONTRACT.md`
- ISSUE-000007-first-web-target-projection.md
- ISSUE-000028-first-web-target-conformance-evidence.md
- ISSUE-000029-gate-b-readiness-audit.md
- ISSUE-000030-gate-b-to-c-handover-baseline.md
