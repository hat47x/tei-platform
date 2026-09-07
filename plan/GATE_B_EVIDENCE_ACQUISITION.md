# Gate B Evidence Acquisition

## 目的

この文書は、Gate Bで**何のEvidenceが必要か、各Evidenceが何を証明し、何を証明しないか**を定めます。

- 具体的なcommand / 実行順 / fail-closed停止条件: [`GATE_B_EVIDENCE_RUNBOOK.md`](GATE_B_EVIDENCE_RUNBOOK.md)
- Ready / Not Ready判定: [`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)
- Gate B → C引渡し: [`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md)

Runbook以外へ第二の実行正本を作りません。本書はEvidence nature、証明範囲、provenance、完了条件を扱います。

Upstream traceではPAS-02をprimary、PAS-03をsecondaryとして扱います。

## 現行Gate B scope

Gate B Readyへ必須のsource-dependent Evidenceは次の三つです。

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL Provider + ephemeral fixture Evidence

三つを同じclean candidate revision `R`へ揃えます。

```text
#310 ─┐
#255 ─┼─ repositoryRevision = R
#308 ─┘
       ↓
#387 final convergence
       ↓
same-run child exact artifacts
       ↓
independent artifact audit
       ↓
Gate B Readiness Audit for R
       ↓ Ready only
Handover Baseline B / revision R
```

actual external asset / Oracle Evidenceは重要なEvidence natureとして継続しますが、第四Required laneではありません。取得済みならHandoverへverified external Evidenceとして参照し、未完了なら`unresolved`として渡します。

actual execution Evidenceがまだ揃っていないため、Gate Bは**Not Ready**です。

## Evidence natureを相互代替しない

Gate B周辺では少なくとも次のEvidence natureを区別します。

| Evidence nature | 代表例 | 主に証明するもの |
| --- | --- | --- |
| deterministic repository | #310 | Repository内Application / Driver / Evidence intake境界が固定sourceで再現すること |
| generated artifact reproducibility | #255 | 正規入力から保存generated artifactを固定sourceで再現できること |
| real-provider ephemeral fixture | #308 | 実PostgreSQL provider / driver / native SQLとtool-owned fixture runtime |
| actual external asset | #214等 | TEI非所有の既存外部資産の実物理・provenance・read-only・actual read |

次を読み替えません。

- deterministic / synthetic成功 → actual external asset成功
- generated artifact一致 → Driver Scenario成功
- PostgreSQL ephemeral fixture成功 → production既存Database事実
- PostgreSQL成功 → actual Oracle成功
- acceptance receipt成功 → actual external read成功
- CI green → Product Evidence取得済み

Business Result / Execution Error / Observation / Diagnosticも相互に読み替えません。

## 共通Evidence原則

### source-dependent Evidenceはclean revisionへ固定する

Gate B Required Evidenceは開始時だけでなく、各runnerが要求する終了時まで同じclean Repository revisionへ固定します。

少なくとも次を追跡します。

- `repositoryRevision`
- `reproducibleRevision`
- working tree / index clean
- post-run source確認
- revision stability

異なるrevisionの成功を寄せ集めません。

candidate `R`の途中でsourceまたはgenerated artifact修復が必要になった場合はcycleを停止します。変更を意味確認してcommitし、新しいclean revision `R'`で#310 / #255 / #308 / #387を最初から取り直します。

mainが先へ進んでも、candidateを暗黙に最新mainへ移しません。

### EvidenceはCanonical authorityではない

```text
Authoritative Source / Canonical meaning
        ↓
Validation / execution / observation
        ↓
Evidence artifact
```

Evidence artifactからCanonicalへ暗黙に逆書きしません。external EvidenceもCanonical Worldへ取り込みません。

### secret / private runtime materialを通常Evidenceへ保存しない

少なくとも次を通常Evidenceへ保存しません。

- DSN
- credential / secret
- original secret file path
- temporary handoff file path
- runtime parameter value
- 実Database名
- 不要な実データ

## Required Evidence A — deterministic Repository / #310

関連: `ISSUE-000032` / GitHub #310

### 目的

Repository checkoutと通常Go runnerから、Application Driverの主要意味境界とexternal Evidence intake Conformanceを再現できることを示します。

正規runnerは `tools/reference-gate-b-repository-evidence` です。具体commandはRunbookを正とします。

### 必要なEvidence

一つのsource-fixed Evidenceへ少なくとも次を束ねます。

- Canonical Workspace Validation
- final UI-less Scenario
- external Evidence intake boundary Conformance
- Repository source observation Conformance
- Go toolchain identity
- run前後のRepository source

final UI-less Scenarioは少なくとも次を接続します。

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

### source / Workspace identity

Gate B Evidence専用full checkoutを使用し、tracked / untracked / ignored local stateを許容しません。tracked index entryは通常`H`だけを許容し、`assume-unchanged` / `skip-worktree` / sparse checkout等の非標準index stateを成功Evidenceへ持ち込みません。

Workspace identityは `reference/workspace` を正規contractへexact bindingします。

### 完了条件

Gate Bへ採用する#310 Evidenceには少なくとも次を要求します。

- `evidenceNature = deterministic-repository`
- `repositoryRevision = R`
- `reproducibleRevision = true`
- Workspace Validation成功
- final UI-less Scenario成功
- external Evidence intake成功
- Repository source-boundary Conformance成功
- producer checkの`name / command / passed`が正規contractと一致
- post-run source確認済み
- 実行前後HEAD同一
- 終了時clean source / index state

synthetic Evidenceをactual external Evidenceへ格上げしません。

## Required Evidence B — generated artifact reproducibility / #255

関連: `ISSUE-000025` / GitHub #255

### 目的

Canonical Workspace等の正規入力から、保存generated artifactを同じclean revisionで再生成・照合できることを示します。

対象:

- `reference/world.json`
- `reference/workspace/.tei/plugin-lock.json`
- `reference/order_rollout.durable-run.m0.21.start.json`

正本はauthoring sourceです。generated artifactを正本としてCanonicalへ逆書きしません。

### checkとrepairを分ける

Gate B最終Evidence入口はcheck-onlyの `tools/reference-generated-artifact-evidence` です。

staleの場合だけ正規generatorを修復操作として使い、diffを意味上確認してcommitします。`-write`直後のdirty treeやsafe-replace成功は完了Evidenceではありません。

### 完了条件

少なくとも次を要求します。

- World / Plugin Lock golden一致
- Durable Run golden一致
- producer checkの`name / command / passed`が正規contractと一致
- `repositoryRevision = R`
- `reproducibleRevision = true`
- post-run source確認済み
- 実行前後HEAD同一
- 終了時clean source / index state

## Required Evidence C — real PostgreSQL ephemeral fixture / #308

関連: `ISSUE-000031` / GitHub #308

### 目的

`Order -> OrderLine[]`について、成立済みSemantic Relation、Relation Storage、Contract Query realization、Driver runtimeが**実PostgreSQL engine**上でも同じ意味境界を保つことを示します。

これはproduction既存PostgreSQL Databaseの事実を観測するEvidenceではありません。

### Evidence専用Database

#308はtool-owned ephemeral fixtureを使います。fixture作成前に少なくとも次をfail closedで確認します。

- Canonical Workspace Validation成功
- default Databaseではない
- user relation 0件
- Event Trigger 0件
- fixture object不在

既存relationやEvent Triggerを削除・無効化してEvidence用Databaseへ転用しません。

### DB-side fixture ownership / cleanup

fixture作成時にDB側ownership markerを付与します。

cleanupではtableをlockし、DB上markerが作成時markerとexact matchしたobjectだけをDROPします。同名objectが別主体へ置換された疑いがある場合、名前だけを根拠に削除しません。

cleanup不能でtool fixtureが残る可能性は、他主体のobjectを誤削除することより安全側として許容します。そのDatabaseは次回empty-DB guardを通りません。

### runtime Evidence

同じrunで少なくとも次を確認します。

- 2件取得
- 正常0件
- missing-tableをExecution Errorとして扱う
- explicit raw result → Semantic mapping
- support column非露出
- parameter nameを追跡し、valueを通常Evidenceへ保存しない

### 完了条件

Gate Bへ採用する#308 Evidenceには少なくとも次を要求します。

- `evidenceNature = real-postgres-ephemeral-fixture`
- `repositoryRevision = R`
- `reproducibleRevision = true`
- Canonical target identity / parameter / deterministic Semantic attributesが正規contractと一致
- `fixtureDatabaseSafety.guardPassed = true`
- DB-side ownership marker exact一致
- `fixtureCleanupSafety.verified = true`
- cleanup後user relation 0 / Event Trigger 0 / fixture不在
- post-run source確認済み
- 実行前後HEAD同一
- 終了時clean source / index state
- DSN / credential / parameter値 / 実Database名非保存

## Final convergence Evidence / #387

関連: `ISSUE-000037` / GitHub #387

### 目的

三つのRequired Evidenceが個別に再現可能なだけでなく、**同じcandidate revision R**へ収束したことを示します。

final runnerは `tools/reference-gate-b-final-evidence` です。具体起動条件はRunbookを正とします。

### lane順序

```text
#310
  ↓ source check
#255
  ↓ source check
#308
  ↓ source check
final source check
```

#310または#255が失敗した場合、不要なPostgreSQL fixture操作へ進みません。

### final success

final成功には少なくとも次を要求します。

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

各laneには少なくとも:

- `passed = true`
- `repositoryRevision = R`
- `reproducibleRevision = true`
- `artifactFile`
- `evidenceArtifactFingerprint`

を要求します。

final runnerが実際に読んだchild stdout exact bytesをRepository外へ保持し、そのSHA-256を`evidenceArtifactFingerprint`として記録します。fingerprintだけでEvidence本文や合否を代替しません。

## Final runのexecution Evidence境界

### Repository source observation

共有source observationは概念上次を同じHEAD観測窓で取得します。

```text
HEAD before
  ↓
git status --porcelain --untracked-files=all --ignored=matching
  ↓
git ls-files -v -z
  ↓
HEAD after
```

少なくとも次をfail closedで要求します。

- HEAD stable
- tracked / untracked / ignored local stateなし
- tracked index entryが通常`H`のみ
- full checkout
- Repository / work tree / common dir / index / object store / namespace等のGit identity overrideなし
- command-scope Git config injectionなし

source observation不能、HEAD drift、ignored local state、非標準index stateを成功Evidenceへ持ち込みません。

### Go environment

final runnerのcompile前からRepository外のGo environment / workspace / flagsを暗黙入力へしません。

actual final JSONでは少なくとも:

```text
goEnvironment.externalWorkspaceActive = false
goEnvironment.goFlagsEmpty = true
goEnvironment.goEnvDisabled = true
```

を確認します。

### PostgreSQL secret exposure

final launch environmentへdirect `TEI_TEST_POSTGRES_DSN`を置きません。

Repository外の一行secret fileを指定し、final runnerはsecret-free clean-source preflight成功後にoriginal secret fileをstable single-openします。normalized DSN本文はmemoryに保持したうえで、Repository外owner-only temporary handoff fileへ複製します。

```text
outer go run / compiler / linker
  -> DSN本文なし
  -> original secret file pathのみ

Go / Git probe / #310 / #255
  -> DSN本文なし
  -> original secret file pathなし
  -> handoff file pathなし

#308 go run / compiler / linker
  -> DSN本文なし
  -> original secret file pathなし
  -> temporary handoff file pathのみ

#308 runtime
  -> handoff fileからDSN本文を読込
  -> DSN / file path environmentをsource probe前に消去
  -> 以後DSN本文はmemoryのみ
```

normal final runner return時にtemporary handoff directoryをcleanupします。process強制killやhost停止に対するOS-level完全消去までは保証せず、Repository外owner-only temporary scopeへ限定します。

DSN本文、credential、original secret path、handoff pathを通常Evidenceへ保存しません。

これらのenvironment / secret境界はEvidence natureを増やすものではなく、Required Evidence取得の再現性・least exposure条件です。

## Same-run child exact artifact independent audit

final convergence成功だけでGate B Readyにはしません。

#387が実際に保持したsame-run child exact artifactsを、read-only補助tool `tools/reference-gate-b-readiness-artifact-audit` で独立に再監査します。具体command / 順序はRunbookを正とします。

このartifact auditは第四Required laneではありません。`artifactAuditPassed = true`だけでReadyを宣言しません。

独立監査では少なくとも次を確認します。

- fixed 3 lane / fixed artifact filename
- final `artifactFile`との対応
- child exact bytes SHA-256 == final `evidenceArtifactFingerprint`
- same `repositoryRevision = R`
- child / final lane envelopeのnature / revision / reproducible一致
- #310 Workspace identity
- #310 / #255 producer checkの`name / command / passed` exact一致
- #308 Canonical target identity / parameter / deterministic Semantic attributes
- #308 fixture前後安全性 / ownership / cleanup
- post-run clean source
- secret / credential / DSN / physical identity等の禁止field非保存

同じRで後から再実行した別artifactを、final convergenceが読んだsame-run artifactへ読み替えません。

## Optional / unresolved Evidence — actual external asset / Oracle

関連:

- `ISSUE-000022` / #214
- `ISSUE-000033` / #371
- `ISSUE-000034` / #376

### 性質

これは**actual external asset Evidence**です。

TEIが所有していない既存Oracle資産について、実際に観測したphysical facts、provenance、effective read-only、concrete driver behavior、actual readを扱います。

Evidence natureとして重要ですが、Gate B Readyの第四Required laneではありません。

### 未観測事実を補完しない

少なくとも次を観測前に推測しません。

- physical object kind / identity
- physical parameter / raw columns
- physical type / NULL semantics
- native Oracle SQL
- concrete driver constraints
- effective read-only mechanism

### Evidence intakeとactual readを分ける

external Evidence artifactは非Canonicalのまま、明示authoring済みrealizationと照合します。

acceptance receiptが成功してもactual read成功にはしません。

```text
external observed artifact
      ↓
source-fixed acceptance
      ↓
actual read-only runtime
      ↓
raw result observation
      ↓
explicit Semantic mapping
```

external observed artifact自体の`observedAt`がcandidate Rの時点である必要はありません。ただし、そのartifactをRに対するverified claimとしてHandoverへ載せる場合は、Rとのacceptance / runtime / actual read provenanceを明示します。

旧revisionのacceptanceやactual readをRへ黙ってcarry-forwardしません。

### Gate B通過時の扱い

actual Oracle Evidenceが取得済みでRに対するverified claimとして十分なら、Handoverから追加のverified external asset Evidenceとして参照できます。

未取得、actual read未完了、またはRへの結び付きが不足する場合は、未検証範囲を`unresolved`としてHandoverします。

第一Web Target側で不足Oracle factsを補完しません。

## Gate B Required Evidence Acquisitionの完了条件

少なくとも次が必要です。

- [ ] #310 actual Evidenceがcandidate Rで成立
- [ ] #255 actual Evidenceがcandidate Rで成立
- [ ] #308 actual Evidenceがcandidate Rで成立
- [ ] #387 final convergenceがRで成立
- [ ] Required Evidenceすべて`reproducibleRevision = true`
- [ ] final `allLanesSameRevision = true`
- [ ] `childArtifactsRetained = true`
- [ ] 三つのsame-run child exact artifactsを保存済み
- [ ] child SHA-256とfinal fingerprint一致
- [ ] independent artifact auditをsame-run artifactsへ実行し`artifactAuditPassed = true`
- [ ] #310 Workspace identity / #310・#255 producer commandが正規contractと一致
- [ ] #308 lane固有Semantic / fixture safety / cleanup条件成立
- [ ] final run終了時もR / clean source / index state
- [ ] PostgreSQL fixture前後安全性がactual runで成立
- [ ] Go environment / source observation / secret exposure条件がactual runで成立
- [ ] Evidence natureを相互代替していない
- [ ] secret / credential / DSN本文 / original secret path / handoff path / 実Database名を通常Evidenceへ保存していない

これらを取得した後、`GATE_B_READINESS_AUDIT.md`がRをReady / Not Ready判定します。一つでも成立しなければNot Readyです。

## Handoverとの関係

Gate BがRについてReadyになった場合、同じRを`GATE_B_TO_C_HANDOVER_CONTRACT.md`へ渡します。

```text
Handover Baseline R
  ├─ Required Evidence
  │    ├─ #310
  │    ├─ #255
  │    └─ #308
  │
  ├─ same-run Evidence provenance
  │    ├─ #387 final
  │    └─ retained child exact artifacts
  │
  └─ external asset status
       ├─ verified actual Evidence
       └─ unresolved
```

Handover recordは第二のCanonical authorityではありません。

## 非目標

- actual OracleをGate B第四Required laneへ戻すこと
- artifact auditを第四Required laneへすること
- artifact audit成功だけでReadyを宣言すること
- Evidence natureを一つへ畳むこと
- synthetic / generated / provider fixture / actual external Evidenceを相互代替すること
- external EvidenceからCanonical realizationを自動生成すること
- generated artifactを手編集すること
- credential / secretをEvidenceへ保存すること
- Runbook以外に実行commandの第二正本を作ること
- CIをProduct Evidenceへ読み替えること

## 関連

- `GATE_B_EVIDENCE_RUNBOOK.md`
- `GATE_B_READINESS_AUDIT.md`
- `GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `tools/reference-gate-b-readiness-artifact-audit/README.md`
- `issue/000022-external-master-data-physical-query-evidence.md`
- `issue/000025-reference-generated-artifacts-refresh.md`
- `issue/000029-gate-b-readiness-audit.md`
- `issue/000030-gate-b-to-c-handover-baseline.md`
- `issue/000031-gate-b-postgres-header-detail-evidence.md`
- `issue/000032-gate-b-deterministic-repository-evidence.md`
- `issue/000035-gate-b-external-provider-scope-audit.md`
- `issue/000037-gate-b-three-lane-final-evidence.md`
- `issue/000038-gate-b-repository-source-observation-stability.md`
- `issue/000039-gate-b-final-go-environment-isolation.md`
- `issue/000040-gate-b-child-go-environment-isolation.md`
- `issue/000041-gate-b-postgres-fixture-ownership-marker.md`
- `issue/000042-gate-b-final-goenv-launch-guard.md`
- `issue/000043-gate-b-probe-postgres-dsn-isolation.md`
- `issue/000044-gate-b-git-probe-postgres-dsn-isolation.md`
- `issue/000045-gate-b-final-postgres-dsn-secret-file.md`
