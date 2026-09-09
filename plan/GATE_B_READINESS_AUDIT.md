# Gate B Readiness Audit

- 判定日: 2026-09-05
- 対象: `order_entry` Application Driver / Gate B
- 判定: **Not Ready**
- Product Acceptance Scenario: PAS-02 primary / PAS-03 secondary
- Evidence要求: [`GATE_B_EVIDENCE_ACQUISITION.md`](GATE_B_EVIDENCE_ACQUISITION.md)
- 実行手順: [`GATE_B_EVIDENCE_RUNBOOK.md`](GATE_B_EVIDENCE_RUNBOOK.md)
- Repository Issue: [`issue/000029-gate-b-readiness-audit.md`](issue/000029-gate-b-readiness-audit.md) / GitHub #273
- Handover: [`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md)

## 目的

Gate B「UIなしで主要Scenarioを再現できる」を、Issue close数や設計済みという主張ではなく、**actual source-fixed Evidence**から判定します。

第一Web Targetへ進む前に、Application上の意味をWeb固有コードで補わなくてもよいことと、Required Evidenceを一つのclean revisionへ固定できることを確認します。

## 現行scope

Gate B Readyへ必須のsource-dependent Evidenceは次の三つです。

| Evidence | Repository / GitHub Issue | nature | 現在地 |
| --- | --- | --- | --- |
| #310 | `ISSUE-000032` / #310 | deterministic repository | Ready to run / actual clean checkout待ち |
| #255 | `ISSUE-000025` / #255 | generated artifact reproducibility | Ready to check / actual clean Go environment待ち |
| #308 | `ISSUE-000031` / #308 | real-provider ephemeral fixture | Ready to run / Evidence専用空PostgreSQL待ち |

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

actual Oracle #214 / #371 / #376は重要なactual external asset Evidenceとして継続しますが、第四Required laneではありません。

## 現在の結論

**Gate B: Not Ready**です。

中心設計が未成立だからではありません。Ready判定に必要なactual execution Evidenceがまだ取得されていないためです。

未完了:

- #310 actual run
- #255 actual check-only run
- #308 actual real PostgreSQL run
- 三つのsame-R final convergence
- final convergenceが実際に読んだsame-run child Evidence artifact保持
- same-run child exact artifactへの独立artifact audit
- Rについて本監査を再実施

runner実装、isolated test、文書整備だけではReadyへ格上げしません。

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

- [ ] #310 actual EvidenceがRで成立
- [ ] #255 actual EvidenceがRで成立
- [ ] #308 actual EvidenceがRで成立
- [ ] #387 final convergenceがRで成立
- [ ] `allLanesSameRevision = true`
- [ ] `childArtifactsRetained = true`
- [ ] 三つのsame-run child exact artifactを保存済み
- [ ] 各artifact fingerprintがfinal summaryと一致
- [ ] 独立artifact auditをsame-run artifactsへ実行し `artifactAuditPassed = true`
- [ ] #310 Workspace identity / #310・#255 producer commandが正規contractと一致
- [ ] child詳細から各lane固有安全条件を再確認できる
- [ ] final run終了時もR / clean source/index state
- [ ] Go実行環境fail-closed条件成立
- [ ] Repository root / source observation条件成立
- [ ] PostgreSQL fixture前後安全性成立
- [ ] DSN / credential / secret /実Database名を通常Evidenceへ保存していない
- [ ] Evidence natureを相互代替していない
- [ ] Business Result / Execution Error / Observation境界維持
- [ ] external未観測事項を推測していない
- [ ] actual Oracle未完了範囲をHandover `unresolved`へ列挙可能
- [ ] 監査結果をRへ固定可能

一つでも満たさなければNot Readyです。

## 次の行動

現時点では設計拡張ではなくactual Evidence取得を優先します。

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
Gate B Readiness Audit for R
        ↓ Ready only
Handover Baseline B / revision R
        ↓
First Web Target
```

## 非目標

- actual Oracleを第四Required laneへ戻す
- final runner成功だけで自動Ready宣言する
- artifact audit成功だけで自動Ready宣言する
- 同じRの別runをsame-run artifactへ読み替える
- GitHub Actions greenをProduct Evidenceへ格上げする
- 第一Web Target側で不足meaningを補完する