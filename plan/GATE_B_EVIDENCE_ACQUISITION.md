# Gate B Evidence Acquisition

## 目的

この文書は、Gate Bで**何のEvidenceが必要か、各Evidenceが何を証明し、何を証明しないか**を定めます。

具体的なコマンド、実行順、差分確認、fail-closed停止条件は[`GATE_B_EVIDENCE_RUNBOOK.md`](GATE_B_EVIDENCE_RUNBOOK.md)を唯一の実行手順として扱います。

Gate BのReady / Not Ready判定は[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)を正とします。

```text
GATE_B_EVIDENCE_ACQUISITION.md
  └─ 必要なEvidence / Evidence nature / 証明範囲

GATE_B_EVIDENCE_RUNBOOK.md
  └─ 取得順序 / command / fail-closed条件

GATE_B_READINESS_AUDIT.md
  └─ 取得済みEvidenceからReady / Not Readyを判定
```

Upstream traceではPAS-02をprimary、PAS-03をsecondaryとして扱います。

## 3 Required EvidenceとEvidence natureを分ける

TEIではEvidenceの性質を相互代替しません。

Gate B周辺では少なくとも次のEvidence natureを区別します。

| Evidence nature | 代表例 | 主に証明するもの |
| --- | --- | --- |
| deterministic repository | #310 | Repository内のApplication / Driver / Evidence intake境界が固定sourceで再現すること |
| generated artifact reproducibility | #255 | 正規入力から保存generated artifactを固定sourceで再現できること |
| real-provider ephemeral fixture | #308 | 実PostgreSQL provider / driver / native SQLとtool-owned fixture runtime |
| actual external asset | #214等 | TEI非所有の既存外部資産の実物理・provenance・read-only・actual read |

**Evidence natureが四種類あることと、Gate B Readyへ四つのRequired laneが必要であることは別です。**

`ISSUE-000035` / #380以降、Gate B Readyへ必須とするsource-dependent Evidenceは次の三つです。

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL Provider + ephemeral fixture Evidence

actual external asset Evidenceは削除しませんが、Gate Bの第四Required laneにはしません。

## 現在地

| Required Evidence | Repository / GitHub Issue | 現在地 |
| --- | --- | --- |
| #310 | `ISSUE-000032` / #310 | runner / source固定境界 Ready、actual clean checkout run待ち |
| #255 | `ISSUE-000025` / #255 | check / repair境界 Ready、actual clean check-only待ち |
| #308 | `ISSUE-000031` / #308 | runner / DB safety境界 Ready、Evidence専用空PostgreSQLでのactual run待ち |

三つのsource変更が出尽くした後、`ISSUE-000037` / #387のfinal runnerで同じclean candidate revision `R`へ収束します。

```text
#310 ─┐
#255 ─┼─ repositoryRevision = R
#308 ─┘
       ↓
#387 final convergence
       ↓
Gate B Readiness Audit
```

actual run Evidenceがまだ揃っていないため、Gate BはNot Readyです。

## 共通Evidence原則

### 1. natureを相互代替しない

次の読み替えをしません。

- synthetic / deterministic test成功 → actual external asset成功
- generated artifact一致 → Driver Scenario成功
- PostgreSQL ephemeral fixture成功 → production既存Database事実
- PostgreSQL成功 → actual Oracle成功
- acceptance receipt成功 → actual external read成功
- CI green → Product Evidence取得済み

### 2. source-dependent Evidenceはclean revisionへ固定する

Gate B Required Evidenceは、開始時だけでなく各runnerが要求する終了時まで同じclean Repository revisionへ固定します。

少なくとも次を追跡します。

- `repositoryRevision`
- working tree clean
- post-run source確認
- revision stability
- `reproducibleRevision`

異なるrevisionの成功を寄せ集めません。

### 3. source変更後はEvidenceをcarry-forwardしない

candidate Rの途中でsourceまたはgenerated artifact修復が必要になった場合:

1. 変更を行う
2. 意味を確認する
3. commitする
4. 新しいclean revision R'を選ぶ
5. Required EvidenceをR'で取り直す
6. final convergenceをR'で再実行する

「意味は変わっていないはず」で旧EvidenceをR'へ読み替えません。

### 4. EvidenceはCanonical authorityではない

```text
Authoritative Source / Canonical meaning
        ↓
Validation / execution / observation
        ↓
Evidence artifact
```

Evidence artifactからCanonicalへ暗黙に逆書きしません。

external EvidenceもCanonical Worldへ取り込みません。

### 5. secret / private runtime materialを通常Evidenceへ保存しない

少なくとも次を通常Evidenceへ保存しません。

- DSN
- credential / secret
- secret file path
- runtime parameter value
- 実Database名
- 不要な実データ

## Required Evidence A — deterministic Repository / #310

関連: `ISSUE-000032` / GitHub #310

### 目的

Repository checkoutと通常Go runnerから、Application Driverの主要意味境界とexternal Evidence intake Conformanceを再現できることを示します。

### 必要なEvidence

正規runnerは:

```text
tools/reference-gate-b-repository-evidence
```

少なくとも次を一つのsource-fixed Evidenceへ束ねます。

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

### 完了条件

Gate Bへ採用する#310 Evidenceには少なくとも次を要求します。

- `evidenceNature = deterministic-repository`
- `repositoryRevision = R`
- `reproducibleRevision = true`
- Workspace Validation成功
- `final-ui-less-scenario`成功
- `external-evidence-intake`成功
- `repository-evidence-source-boundary`成功
- post-run source確認済み
- 実行前後HEAD同一
- 終了時working tree clean

### 証明しないもの

- actual Oracle physical facts
- actual Oracle read
- PostgreSQL real-provider Evidence
- generated artifact reproducibility
- Browser / Web Target Conformance

## Required Evidence B — generated artifact reproducibility / #255

関連: `ISSUE-000025` / GitHub #255

### 目的

Canonical Workspace等の正規入力から、保存generated artifactを同じclean revisionで再生成・照合できることを示します。

対象:

- `reference/world.json`
- `reference/workspace/.tei/plugin-lock.json`
- `reference/order_rollout.durable-run.m0.21.start.json`

### 正本と生成物

正本はauthoring sourceです。

generated artifactを正本としてCanonicalへ逆書きしません。

次を行いません。

- generated JSONの手編集
- fingerprintだけの置換
- fixed observed fixtureを生成結果へ合わせて書き換える

### checkとrepairを分ける

Gate B最終Evidence入口はcheck-onlyです。

```text
tools/reference-generated-artifact-evidence
```

staleの場合だけ、個別golden toolの`-write`を修復操作として利用します。

```text
check-only
  ↓ stale
必要な -write
  ↓
git diff
  ↓
意味確認
  ↓
commit
  ↓
新しいclean revision
  ↓
check-only再実行
```

`-write`直後のdirty treeやsafe-replace成功を完了Evidenceへ格上げしません。

### 完了条件

少なくとも次を要求します。

- World / Plugin Lock golden一致
- Durable Run golden一致
- `repositoryRevision = R`
- `reproducibleRevision = true`
- post-run source確認済み
- 実行前後HEAD同一
- 終了時working tree clean

### 証明しないもの

- UI-less Driver Scenarioそのもの
- actual PostgreSQL runtime
- actual Oracle asset
- external observed physical facts

## Required Evidence C — real PostgreSQL ephemeral fixture / #308

関連: `ISSUE-000031` / GitHub #308

### 目的

`Order -> OrderLine[]`について、成立済みSemantic Relation、Relation Storage、Contract Query realization、Driver runtimeが**実PostgreSQL engine**上でも同じ意味境界を保つことを示します。

これはproduction既存PostgreSQL Databaseの事実を観測するEvidenceではありません。

### Evidence専用Database

#308はtool-owned ephemeral fixtureを使います。

fixture作成前に少なくとも次をfail closedで確認します。

- Canonical Workspace Validation成功
- default Databaseではない
- user relation 0件
- Event Trigger 0件
- `public.order_lines`不在

既存relationやEvent Triggerを削除・無効化してEvidence用Databaseへ転用しません。

### DB-side fixture ownership

fixture作成時にDB側ownership markerを付与します。

cleanupではtableをlockした後、作成時markerとDB上markerがexact matchしたobjectだけをDROPします。

同名objectが別主体へ置換された疑いがある場合、名前だけを根拠に削除しません。

cleanup不能でtool fixtureが残る可能性は、他主体のobjectを誤削除することより安全側として許容します。そのDatabaseは次回empty-DB guardを通りません。

### runtime Evidence

少なくとも次を同じrunで確認します。

- `order_id=1001`で2件
- `order_id=9999`で0件を正常なempty collectionとして扱う
- fixture削除後のphysical table欠落をExecution Errorとして扱う
- raw result → Semantic OrderLine mappingを明示する
- support column `order_id`をSemantic attributeへ漏らさない
- parameter nameを追跡し、valueを通常Evidenceへ保存しない

### cleanup Evidence

成功runでは少なくとも:

- cleanup後user relation 0件
- cleanup後Event Trigger 0件
- fixture object不在
- `fixtureCleanupSafety.verified = true`

を確認します。

### source-fixed完了条件

Gate Bへ採用する#308 Evidenceには少なくとも次を要求します。

- `evidenceNature = real-postgres-ephemeral-fixture`
- `repositoryRevision = R`
- `reproducibleRevision = true`
- `fixtureDatabaseSafety.guardPassed = true`
- `fixtureCleanupSafety.verified = true`
- post-run source確認済み
- 実行前後HEAD同一
- 終了時working tree clean
- DSN / credential /実Database名非保存

### 証明しないもの

- production既存PostgreSQL schema
- actual Oracle physical facts
- actual external read-only Evidence
- generated artifact reproducibility

## Final convergence Evidence / #387

関連: `ISSUE-000037` / GitHub #387

### 目的

三つのRequired Evidenceが、個別に再現可能なだけでなく、**同じcandidate revision R**へ収束したことを示します。

final runner:

```text
tools/reference-gate-b-final-evidence
```

具体起動条件はRunbookを正とします。

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

### 照合項目

各child Evidenceの少なくとも:

- `evidenceNature`
- `repositoryRevision`
- `reproducibleRevision`

を照合します。

final成功には少なくとも次を要求します。

```text
evidenceNature = gate-b-three-lane-final-convergence
repositoryRevision = R
reproducibleRevision = true
allLanesSameRevision = true
postRunRepository.checked = true
postRunRepository.revisionStable = true
postRunRepository.workingTreeClean = true
```

child Evidence exact bytesの`evidenceArtifactFingerprint`は、final runnerが実際に読み込んだartifactとの照合に利用します。fingerprintだけでEvidence本文や合否を代替しません。

## Final runのexecution Evidence境界

### Repository source observation

共有source helperは概念上:

```text
HEAD before
  ↓
git status --porcelain
  ↓
HEAD after
```

で観測します。

観測中HEAD driftをfail closedで拒否します。

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

final runnerへDSN本文をlaunch environmentから直接渡しません。

Repository外の一行secret fileをrunner起動後に読み、DSN本文はmemoryで保持します。

```text
outer go run / compiler / linker -> DSN本文なし
Go / Git probe                  -> DSN本文 / file pathなし
#310                            -> DSN本文 / file pathなし
#255                            -> DSN本文 / file pathなし
#308                            -> in-memory DSN本文のみ
```

secret file pathも通常Evidenceへ保存しません。

これらのenvironment / secret境界はEvidence natureを増やすものではなく、Required Evidence取得の再現性・least exposure条件です。

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

### provenance

external observed artifact自体の`observedAt`がcandidate Rの時点である必要はありません。

ただし、そのartifactをRに対するverified claimとしてHandoverへ載せる場合は、Rとのacceptance / runtime / actual read provenanceを明示します。

旧revisionのacceptanceやactual readをRへ黙ってcarry-forwardしません。

### Gate B通過時の扱い

actual Oracle Evidenceが取得済みでRに対するverified claimとして十分なら、Handoverから追加のverified external asset Evidenceとして参照できます。

未取得、actual read未完了、またはRへの結び付きが不足する場合は、未検証範囲を`unresolved`としてHandoverします。

第一Web Target側で不足Oracle factsを補完しません。

## Gate B Required Evidenceの完了条件

Gate B Required Evidence Acquisitionが完了したと言えるには、少なくとも次が必要です。

- [ ] #310 actual Evidenceがcandidate Rで成立
- [ ] #255 actual Evidenceがcandidate Rで成立
- [ ] #308 actual Evidenceがcandidate Rで成立
- [ ] #387 final convergenceがRで成立
- [ ] Required Evidenceすべて`reproducibleRevision = true`
- [ ] final `allLanesSameRevision = true`
- [ ] final run終了時もR / clean source
- [ ] PostgreSQL fixture前後安全性がactual runで成立
- [ ] Go environment / source observation / secret exposure条件がactual runで成立
- [ ] Evidence natureを相互代替していない
- [ ] secret / credential / DSN本文 /実Database名を通常Evidenceへ保存していない

これらを取得した後、`GATE_B_READINESS_AUDIT.md`がRをReady / Not Ready判定します。

## Handoverとの関係

Gate BがRについてReadyになった場合、同じRを`GATE_B_TO_C_HANDOVER_CONTRACT.md`へ渡します。

HandoverではRequired Evidenceとexternal Evidenceを区別します。

```text
Handover Baseline R
  ├─ Required Evidence
  │    ├─ #310
  │    ├─ #255
  │    └─ #308
  │
  └─ external asset status
       ├─ verified actual Evidence
       └─ unresolved
```

Handover recordは第二のCanonical authorityではありません。

## 非目標

- actual OracleをGate B第四Required laneへ戻すこと
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
