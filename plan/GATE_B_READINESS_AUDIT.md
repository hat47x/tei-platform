# Gate B Readiness Audit

- 判定日: 2026-09-05（初回 Not Ready）/ 2026-09-19（再監査 Ready）
- 対象: `order_entry` Application Driver / Gate B
- 判定: **Ready**（candidate revision `R = e5231839a5dcc62cce1181a2731fcb7c70d16736`）
- Product Acceptance Scenario: PAS-02 primary / PAS-03 secondary
- Evidence要求: [`GATE_B_EVIDENCE_ACQUISITION.md`](GATE_B_EVIDENCE_ACQUISITION.md)
- 実行手順: [`GATE_B_EVIDENCE_RUNBOOK.md`](GATE_B_EVIDENCE_RUNBOOK.md)
- Repository Issue: [`issue/000029-gate-b-readiness-audit.md`](issue/000029-gate-b-readiness-audit.md) / GitHub #273
- Handover: [`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md) / Baseline記録: [`GATE_B_TO_C_HANDOVER_BASELINE_B.md`](GATE_B_TO_C_HANDOVER_BASELINE_B.md)

## 目的

Gate B「UIなしで主要Scenarioを再現できる」を、Issue close数や設計済みという主張ではなく、**actual source-fixed Evidence**から判定します。

第一Web Targetへ進む前に、Application上の意味をWeb固有コードで補わなくてもよいことと、Required Evidenceを一つのclean revisionへ固定できることを確認します。

## 現行scope

Gate B Readyへ必須のsource-dependent Evidenceは次の三つです。

| Evidence | Repository / GitHub Issue | nature | 現在地 |
| --- | --- | --- | --- |
| #310 | `ISSUE-000032` / #310 | deterministic repository | 成立（R = `e5231839`） |
| #255 | `ISSUE-000025` / #255 | generated artifact reproducibility | 成立（R = `e5231839`） |
| #308 | `ISSUE-000031` / #308 | real-provider ephemeral fixture | 成立（R = `e5231839`） |

三つを同じclean candidate revision `R`へ揃えます。

```text
#310 ─┐
#255 ─┼─ repositoryRevision = R
#308 ─┘
       ↓
#387 final convergence
       ↓
Readiness Audit for R
```

actual Oracle #214 / #371 / #376は重要なactual external asset Evidenceとして継続しますが、第四Required laneではありません。取得状況はGate Bの現在判定に影響していません（下記「actual Oracle」節を参照）。

## 現在の結論

**Gate B: Ready**です（candidate revision `R = e5231839a5dcc62cce1181a2731fcb7c70d16736`）。

### 実行経緯

2026-09-05時点のNot Ready判定後、actual Evidence取得（Gate B Evidence専用full checkout、Evidence専用空PostgreSQL、DSN secret file分離）を実施したところ、`#310`の事前確認である`go run ./cmd/tei validate -workspace reference/workspace`が2件の既存source-navigation bugで失敗しました。

1. `internal/syntax/semantic_anchor.go`の`semanticDataAnchors`が、relation宣言に続く`key <source> -> <target>`句（`reference/workspace/semantic/sales/order.tei`）を未知のmemberとして拒否していた。
2. `internal/workspace/source_index.go`の`database-contract-query` subject lookupが、`contract <id> <version> { ... }`でwrapされたquery宣言に対して top-level専用のanchor helperを誤用していた。

両方とも軽量なanchor/index scannerが、main parserの文法拡張に追従していなかったことが原因で、Canonical meaningやCore schemaの変更ではありません。修正はPR [#726](https://github.com/hat47x/tei-platform/pull/726)（`internal/syntax/semantic_anchor.go`へ`skipSemanticRelationKeys`、`internal/syntax/anchor.go`へ`ContractQueryDeclarationAnchor`を追加、対応するunit testを追加）で行い、Validationが通るようになったことで判明した`reference/world.json` / `reference/workspace/.tei/plugin-lock.json` / durable-run goldenのstaleさを、同PR内で正規generator（`tools/reference-golden -write` / `tools/reference-durable-run-golden -write`）で再生成しました（差分は新規contract-query-executor provider、`sales_core.order_line` contract追加、`order_entry.order_lines`の重複entry除去という実際のCanonical Workspace進展を反映）。

修正後の`#387` final convergence実行で、独立artifact audit（`tools/reference-gate-b-readiness-artifact-audit`）が3件目のバグを検出しました。

3. `tools/reference-gate-b-readiness-artifact-audit/main.go`が`#308`の`contractQueryTarget`期待値をbare query name（`find_by_order`）でhardcodeしていましたが、`internal/realization/contract_query.go`の`ContractQueryTarget.Identity()`は意図的に`Database/Contract/Query`の三つ組（`sales_core/sales_core.order_line/find_by_order`）を返す設計であり、audit tool側の期待値が古い形式のまま残っていました。修正はPR [#727](https://github.com/hat47x/tei-platform/pull/727)（audit toolの期待値と対応するtest fixtureを三つ組形式へ更新）で行いました。

PR #726・#727とも、squash mergeで`main`へ統合しました（`main`のmerge commit: `bba97e21`, `e5231839`）。統合後の`main` HEAD（`e5231839a5dcc62cce1181a2731fcb7c70d16736`）をclean checkoutし、Canonical Workspace Validation・final runner package pre-check・`#387` final convergence・独立artifact auditを最初から再実行し、すべて成功しました。

### 満たされた条件

- #310 / #255 / #308 actual run（same R = `e5231839a5dcc62cce1181a2731fcb7c70d16736`）
- 三つのsame-R final convergence（`#387`）
- final convergenceが実際に読んだsame-run child Evidence artifact保持（`childArtifactsRetained = true`）
- same-run child exact artifactへの独立artifact audit（`artifactAuditPassed = true`）
- Rについて本監査を再実施（本ファイル、判定日2026-09-19）

詳細な条件充足は下記「Ready条件」チェックリストを参照してください。

## 成立済みの意味基線

次は主要な未設計blockerではありません。

### Application / Driver

`browse / record / reference / header-detail / command`を同じ`order_entry` Applicationから解決し、少なくとも次を一続きに接続する構造が成立しています。

```text
Authentication
 -> browse
 -> record
 -> reference
 -> header-detail
 -> Canonical Validation
 -> Authorization
 -> Command Business Result
 -> separate Observation read
```

Command acknowledgementとObserved stateを同一視しません。

### Data / Capability / authority

- logical Datasource / read-read-write境界
- Contract Query
- explicit input / raw result / Semantic mapping
- Relation Storage
- Capability Binding
- Authentication / Authorization
- Business Result / named Outcome / Execution Error
- external Evidence intake

raw physical factsやProvider固有return codeからCanonical meaningを逆定義しません。

## Required Evidence 1 — #310 deterministic Repository

正規runner:

```text
tools/reference-gate-b-repository-evidence
```

Ready判定には少なくとも次を要求します。

- Gate B Evidence専用full checkout
- tracked / untracked / ignored local stateなし
- tracked Git index entryが通常`H`のみ（`assume-unchanged` / `skip-worktree`なし）
- Workspace Validation成功
- final UI-less Scenario成功
- external Evidence intake成功
- repository source-boundary Conformance成功
- `repositoryRevision = R`
- `reproducibleRevision = true`
- post-run source確認済み
- 実行前後同一HEAD
- 終了時clean source/index state

synthetic Evidenceをactual external Evidenceへ読み替えません。

## Required Evidence 2 — #255 generated artifact reproducibility

正規runner:

```text
tools/reference-generated-artifact-evidence
```

Ready判定には少なくとも次を要求します。

- World / Plugin Lock一致
- Durable Run一致
- `repositoryRevision = R`
- `reproducibleRevision = true`
- post-run source確認済み
- 同一clean source/index state

staleの場合だけ正規generatorで修復し、差分を確認してcommitした新しいclean revisionでcheck-onlyを再実行します。

`-write`直後のdirty treeは完了Evidenceではありません。

## Required Evidence 3 — #308 real PostgreSQL

正規runner:

```text
tools/reference-postgres-header-detail-runtime
```

これはproduction既存Database Evidenceではなく、real PostgreSQL engine + tool-owned ephemeral fixture Evidenceです。

Ready判定には少なくとも次を要求します。

### source

- Gate B Evidence専用full checkout
- tracked / untracked / ignored local stateなし
- tracked Git index entryが通常`H`のみ
- `repositoryRevision = R`
- `reproducibleRevision = true`
- post-run source確認済み

### fixture前

- default DBではない
- user relation 0件
- Event Trigger 0件
- fixture object不在
- `fixtureDatabaseSafety.guardPassed = true`

### runtime

- 2件取得
- 正常0件
- missing-table Execution Error
- explicit raw → Semantic mapping
- support column非露出

### cleanup

- DB-side ownership marker exact一致
- cleanup後user relation 0件
- Event Trigger 0件
- fixture object不在
- `fixtureCleanupSafety.verified = true`

DSN、credential、parameter値、実Database名を通常Evidenceへ保存しません。

## #387 final convergence

candidate `R`のsource変更が出尽くした後、`GATE_B_EVIDENCE_RUNBOOK.md`の正規条件でfinal runnerを実行します。

成功summaryには少なくとも次を要求します。

```text
evidenceNature = gate-b-three-lane-final-convergence
repositoryRevision = R
workingTreeClean = true
reproducibleRevision = true
allLanesSameRevision = true
childArtifactsRetained = true
postRunRepository.checked = true
postRunRepository.revisionStable = true
postRunRepository.workingTreeClean = true
```

各lane:

```text
passed = true
repositoryRevision = R
reproducibleRevision = true
artifactFile != empty
evidenceArtifactFingerprint != empty
```

Evidence nature:

```text
#310 deterministic-repository
#255 generated-artifact-reproducibility
#308 real-postgres-ephemeral-fixture
```

## same-run child Evidenceを監査する

final summaryだけでReadyにしません。

final runnerが実際に読んだ三つのchild JSON exact bytesを、同一runのRepository外artifactとして保持していることを要求します。

保存artifact:

```text
deterministic-repository.json
generated-artifact-reproducibility.json
real-postgres-ephemeral-fixture.json
```

Ready判定では、read-only補助tool `tools/reference-gate-b-readiness-artifact-audit` を使い、final summaryとsame-run child exact artifactsを独立に再照合します。実行command / 順序は `GATE_B_EVIDENCE_RUNBOOK.md` を唯一の実行正本とし、本書へcommand本文を複製しません。

このartifact auditは第四Required laneではなく、`artifactAuditPassed = true`だけでGate B Readyにはしません。

Ready判定では:

1. final summaryの`artifactFile`が対応するfixed same-run fileを指すこと
2. file exact bytesのSHA-256が`evidenceArtifactFingerprint`と一致すること
3. child JSON本文とfinal lane envelopeのnature / revision / reproducibleが一致すること
4. #310のWorkspace identityと#310 / #255各checkのproducer commandが正規contractとexact一致すること
5. child JSON本文から#310 / #255 / #308固有の詳細安全条件を確認できること
6. secret / credential / DSN / physical identity等の禁止fieldを通常Evidenceへ保存していないこと

を確認します。check名だけ一致してproducer commandが異なるartifactを同じEvidenceへ読み替えません。

**同じRで後から再実行した別artifactを、final convergenceが読んだartifactと同一だと推測しません。**

## final runner実行環境

### Repository

- Repository rootから起動
- Gate B Evidence専用full checkout
- tracked / untracked / ignored local stateなし
- tracked Git index entryが通常`H`のみ
- `assume-unchanged` / `skip-worktree`なし
- clean candidate R
- 各lane後source再確認
- final source再確認

### Go environment

actual runで:

```text
goEnvironment.externalWorkspaceActive = false
goEnvironment.goFlagsEmpty = true
goEnvironment.goEnvDisabled = true
```

を要求します。

runner自身のcompile前からexact `GOENV=off`、external GOWORK無効、GOFLAGS emptyを要求します。

### PostgreSQL secret

- final launch environmentへdirect `TEI_TEST_POSTGRES_DSN`を置かない
- `TEI_TEST_POSTGRES_DSN_FILE`はRepository外existing regular file
- symlink解決後もRepository外
- final runnerがoriginal secret fileをstable single-openし、normalized DSNをRepository外owner-only temporary handoff fileへ複製する
- Go / Git probe、#310、#255へDSN本文・original secret path・handoff pathを渡さない
- #308 `go run` / compiler / linkerへDSN本文を渡さずtemporary handoff pathだけを渡す
- #308 runtimeがhandoff fileからDSN本文を読み、DSN / file path environmentをsource probe前に消去する
- DSN本文・original secret path・handoff pathをEvidenceへ保存しない

### child Evidence artifact store

- Repository外
- owner-only directory
- child fileは新規作成
- 既存fileを上書きしない
- exact child stdout bytesを保持

## actual Oracle

#214 / #371 / #376は削除・完了扱いにしません。

actual physical facts / provenance / effective read-only / concrete driver integration / actual readが未完了なら、そのまま未完了として保持します。

Gate B Readyの第四Required laneにはしません。

Handover時:

- 十分なactual Oracle Evidenceがあれば追加verified external Evidenceとして参照
- 未取得・actual read未完了・Rへの結び付き不足なら`unresolved`

第一Web Target側でOracle physical factsを補完しません。

## candidate revision

Required Evidenceは一つのRへ固定します。

```text
#310.repositoryRevision = R
#255.repositoryRevision = R
#308.repositoryRevision = R
final.repositoryRevision = R
```

Rを固定するcheckoutはfull checkoutとし、tracked / untracked / ignored local stateなし、tracked Git index entryが通常`H`のみを要求します。

異なるrevisionの成功を寄せ集めません。

source変更が必要ならcycleを停止し、修正をcommitして新Rを選び、三laneとfinal convergenceを最初からやり直します。

mainが進んでもcandidate Rを暗黙に最新mainへ移しません。

## Ready条件

Gate BをReadyとするには、少なくとも次をすべてEvidenceから説明できる必要があります。

- [x] #310 actual EvidenceがRで成立 — `deterministic-repository.json`、`repositoryRevision = e5231839a5dcc62cce1181a2731fcb7c70d16736`、`passed = true`
- [x] #255 actual EvidenceがRで成立 — `generated-artifact-reproducibility.json`、同revision、`passed = true`
- [x] #308 actual EvidenceがRで成立 — `real-postgres-ephemeral-fixture.json`、同revision、`passed = true`
- [x] #387 final convergenceがRで成立 — `evidenceNature = gate-b-three-lane-final-convergence`、`repositoryRevision = e5231839a5dcc62cce1181a2731fcb7c70d16736`
- [x] `allLanesSameRevision = true`
- [x] `childArtifactsRetained = true`
- [x] 三つのsame-run child exact artifactを保存済み（`deterministic-repository.json` / `generated-artifact-reproducibility.json` / `real-postgres-ephemeral-fixture.json`、Repository外owner-only directoryへ保存）
- [x] 各artifact fingerprintがfinal summaryと一致（`evidenceArtifactFingerprint`と`reference-gate-b-readiness-artifact-audit`のlane別fingerprintが一致）
- [x] 独立artifact auditをsame-run artifactsへ実行し `artifactAuditPassed = true` — `finalFingerprint = sha256:3bfcdc459f9e3dd77019e1b9ae368a64bad760f65917c44e1d7d533b6a1f5f97`
- [x] #310 Workspace identity / #310・#255 producer commandが正規contractと一致（`reference-gate-b-readiness-artifact-audit`のcommand-exact-binding checkが`passed = true`）
- [x] child詳細から各lane固有安全条件を再確認できる（`fixtureDatabaseSafety.guardPassed = true`、`fixtureCleanupSafety.verified = true`、`contractQueryTarget = sales_core/sales_core.order_line/find_by_order`ほか）
- [x] final run終了時もR / clean source/index state — `postRunRepository.revisionStable = true` / `workingTreeClean = true`
- [x] Go実行環境fail-closed条件成立 — `goEnvironment.externalWorkspaceActive = false` / `goFlagsEmpty = true` / `goEnvDisabled = true`
- [x] Repository root / source observation条件成立 — Gate B Evidence専用full checkout、`git status --short --untracked-files=all --ignored=matching`空、`git ls-files -v`全`H`を実行前後で確認
- [x] PostgreSQL fixture前後安全性成立 — fixture前`userRelationsBeforeFixture = 0` / `eventTriggersBeforeFixture = 0`、cleanup後`userRelationsAfterCleanup = 0` / `eventTriggersAfterCleanup = 0` / `fixtureObjectAbsentAfterCleanup = true`
- [x] DSN / credential / secret /実Database名を通常Evidenceへ保存していない（`reference-gate-b-readiness-artifact-audit`のsensitive-field拒否checkが`passed = true`、目視でもchild JSONにDSN/secret文字列なしを確認）
- [x] Evidence natureを相互代替していない — 各lane `evidenceNature`が`deterministic-repository` / `generated-artifact-reproducibility` / `real-postgres-ephemeral-fixture`のまま
- [x] Business Result / Execution Error / Observation境界維持 — `#308` child JSONで`executionErrorObserved = true`（missing-table Execution Error）とnormalなrow取得が別fieldとして区別されている
- [x] external未観測事項を推測していない — #310/#255/#308はいずれもsynthetic/PostgreSQL fixture Evidenceのままで、actual Oracleへ格上げしていない
- [x] actual Oracle未完了範囲をHandover `unresolved`へ列挙可能 — 下記「actual Oracle」節、Handover Baseline記録にも反映
- [x] 監査結果をRへ固定可能 — 本ファイルの判定はすべて`R = e5231839a5dcc62cce1181a2731fcb7c70d16736`に紐づく

## 次の行動

Gate B Readyの確定後、Handover Baseline `B`をRepository revision `R = e5231839a5dcc62cce1181a2731fcb7c70d16736`へ固定しました。記録は[`GATE_B_TO_C_HANDOVER_BASELINE_B.md`](GATE_B_TO_C_HANDOVER_BASELINE_B.md)です。

```text
actual full checkout preflight
        ↓
#310 / #255 / #308 pre-check
        ↓ source変更があればcommit
candidate R固定
        ↓
#387 final convergence
        ↓
same-run child exact artifact independent audit
        ↓
Gate B Readiness Audit for R  ← 完了（本ファイル、2026-09-19、Ready）
        ↓ Ready only
Handover Baseline B / revision R  ← 完了（GATE_B_TO_C_HANDOVER_BASELINE_B.md）
        ↓
First Web Target  ← 次のIssue（`ISSUE-000007` / #70）
```

## 非目標

- actual Oracleを第四Required laneへ戻す
- final runner成功だけで自動Ready宣言する
- artifact audit成功だけで自動Ready宣言する
- 同じRの別runをsame-run artifactへ読み替える
- GitHub Actions greenをProduct Evidenceへ格上げする
- 第一Web Target側で不足meaningを補完する