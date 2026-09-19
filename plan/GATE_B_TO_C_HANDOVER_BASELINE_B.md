# Gate B → Gate C Handover Baseline B

- Handover Baseline identity: `B`
- `baselineRepositoryRevision R`: `e5231839a5dcc62cce1181a2731fcb7c70d16736`
- 固定日: 2026-09-19
- 固定根拠: [`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)（判定日2026-09-19、Ready）
- 規則・原則の正本: [`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md)
- Repository Issue: [`issue/000030-gate-b-to-c-handover-baseline.md`](issue/000030-gate-b-to-c-handover-baseline.md) / GitHub #291

本記録は`GATE_B_TO_C_HANDOVER_CONTRACT.md`が定める規則の下で、実際に固定した最初のHandover Baselineの内容を残します。Contract文書自体を書き換えず、本記録が個別revisionのinstanceです。Handoverは新しいCanonical schemaでも第二の正本でもありません。stable reference / identity / fingerprintから実Evidenceへ戻るための束ねです。

## Baseline identity

- Handover Baseline identity: `B`
- `baselineRepositoryRevision = R = e5231839a5dcc62cce1181a2731fcb7c70d16736`（`hat47x/tei-platform` `main`、PR [#726](https://github.com/hat47x/tei-platform/pull/726) + PR [#727](https://github.com/hat47x/tei-platform/pull/727) squash-merge後のHEAD）
- Reference Workspace identity: `reference/workspace`（`go run ./cmd/tei validate -workspace reference/workspace` = `ok`、既知の3件のinformational warning — `APPLICATION_RELEASE_CONTRACT_NOT_CURRENT` x1 / `DATABASE_RELEASE_CONTRACT_NOT_CURRENT` x2、`sales_core.order_legacy`の historical contract未定義に関するもの、Gate B Ready判定をblockしない）
- Gate B Readiness Audit対象revision: `R`と同一（[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)参照）

## Canonical Application baseline（manifest）

第一Referenceで戻れるidentityの列挙です。意味本文はauthoritative sourceが正本であり、ここでは複製しません。

- Application: `order_entry`
- Application Pattern: browse / record / reference / header-detail / command
- Data / Relation / Query: `reference/workspace/semantic/**`、`reference/workspace/database/**`（`sales_core`ほか）
- Datasource: read / read-write境界を持つlogical Datasource定義
- Validation: Canonical Validation（`internal/validation`系）
- Authentication / Authorization: `order_entry` Driver内のAuthentication → Authorization境界
- Capability / Binding: Contract Query Capability Binding（`internal/realization/contract_query.go`の`ContractQueryTarget.Identity()` = `Database/Contract/Query`三つ組）
- Business Result / named Outcome: `internal/driver`のBusiness Result / Execution Error区別
- Execution Error: Provider raw errorからSemantic Execution Errorへの明示mapping
- Observationに使う明示read境界: Command acknowledgementと分離したObservation read
- Source / Binding traceability: `internal/workspace/source_index.go`（Source Index）

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

再現対象: `internal/driver`の`TestOrderEntryGateBScenarioConnectsMajorBoundariesWithoutUI`（`#310` final-ui-less-scenario checkが同一revisionで`passed = true`）。

Business Result、named Outcome、Execution Error、Observation、DiagnosticをTarget側で一種類へ畳みません。

## Datasource / authority boundary

- read / read-write: `sales_core`データベースに対するContract Query経由read
- ownership性質: `#308` Evidenceの`fixtureOwnedByTool = true` / `preexistingFixtureProtected = true`が示す、tool-ownedなephemeral fixtureとproduction既存Databaseの区別
- Database instance / executor Capability Binding: `databaseInstance = sales_core.dev`、`executorCapability = database.query.postgres`
- secret / credential / runtime parameterを通常Evidenceへ保存しない制約: 全same-run child artifactへ目視・機械監査（`reference-gate-b-readiness-artifact-audit`のsensitive-field拒否check）で確認済み

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

実行環境: Gate B Evidence専用full checkout（`git status --short --untracked-files=all --ignored=matching`空、`git ls-files -v`全`H`）、`GOENV=off` / `GOWORK=off` / `GOFLAGS=`、`TEI_TEST_POSTGRES_DSN_FILE`経由のRepository外secret file。手順は[`GATE_B_EVIDENCE_RUNBOOK.md`](GATE_B_EVIDENCE_RUNBOOK.md)を正規実行手順とし、本記録では複製しません。

### same-run child exact artifacts

`#387`が実際に読み、Repository外owner-only directoryへ保持した3件のchild artifact（固定filename）です。

| lane | evidenceNature | artifactFile | evidenceArtifactFingerprint (SHA-256) |
| --- | --- | --- | --- |
| #310 | `deterministic-repository` | `deterministic-repository.json` | `95074b13665c30bbfa8acb644f96f21e039180de771055f3ce7fa0e2c3ecf09e` |
| #255 | `generated-artifact-reproducibility` | `generated-artifact-reproducibility.json` | `83b780957cdf7edd432e6e5780118eb3048c86a72de6a8d7d0b644d0c35fc847` |
| #308 | `real-postgres-ephemeral-fixture` | `real-postgres-ephemeral-fixture.json` | `fb57ab65519fc17541a1e790c5040361c43a7883ebd68339fbe218a24c8e8a1a` |

各childの`repositoryRevision = e5231839a5dcc62cce1181a2731fcb7c70d16736`、`reproducibleRevision = true`です。

artifact本文はRunbookの規則によりRepository外へのみ保存し、Repositoryへコミットしません。stable referenceはこのfingerprint（SHA-256）と、同じ`R`に対して`GATE_B_EVIDENCE_RUNBOOK.md`の手順を再実行することによる再生成可能性です（`reproducibleRevision = true`がこれを担保します）。

**保管先（locator）**: GitHub Release [`gate-b-evidence-R-e523183`](https://github.com/hat47x/tei-platform/releases/tag/gate-b-evidence-R-e523183)（tagは`R`を指す）のasset `tei-gate-b-evidence-R-e5231839.tar.gz`、SHA-256 `1af53d6ec1d7d61427d98cd0d8fda655fe0c37bb53927f36440d5e17cfec4395`。展開して`MANIFEST.sha256`で各fileを再照合できます（`ISSUE-000057`、2026-09-20追記）。

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
- `issue/000007-first-web-target-projection.md`
- `issue/000028-first-web-target-conformance-evidence.md`
- `issue/000029-gate-b-readiness-audit.md`
- `issue/000030-gate-b-to-c-handover-baseline.md`
