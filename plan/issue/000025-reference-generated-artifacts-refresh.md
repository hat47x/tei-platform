# ISSUE-000025: Reference生成物を最新Canonical Worldから再生成する

- 状態: Done — Gate B Ready（R = `e5231839`）のactual #387 final runで確認済み（2026-09-19）
- GitHub Issue: #255
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- 実PostgreSQL Evidence: `ISSUE-000031` / GitHub Issue #308
- actual Oracle Evidence: `ISSUE-000022` / GitHub Issue #214
- deterministic Repository Evidence: `ISSUE-000032` / GitHub Issue #310
- Gate B監査: `ISSUE-000029` / GitHub Issue #273
- 関連RQ: RQ-009 / RQ-012 / RQ-014

## 背景

Reference data access、header-detail、Relation Storage、Contract Query executor等の追加により、`reference/workspace`のCanonical authoringは更新されています。

このIssueでは、**Canonical Workspaceから生成されるReference artifactの再現性**だけを扱います。生成物を形式的に再作成すること自体を目的にはしません。まず保存済みartifactと現在の正規生成結果が一致するかを確認し、staleなものだけを正規経路から再生成します。

実PostgreSQL runtime Evidence、actual external Oracle Evidence、deterministic Repository / normal runner Evidenceは、それぞれ別Issueへ分離します。

## 依存関係

主要なgenerated Reference artifactは次です。

```text
reference/workspace/                 Canonical authoring
  ├─> reference/world.json           generated golden
  └─> .tei/plugin-lock.json          generated golden
          │
          └─ Plugin Lock / Runtime Policy inputs

reference/observed/production.deployment-state.json   fixed observed fixture
reference/workspace/ + Plugin Lock + observed fixture
  └─> reference/order_rollout.durable-run.m0.21.start.json
                                                  generated golden
```

次はCanonical Workspaceから自動導出する対象ではありません。

```text
reference/observed/*
reference/legacy-auth/*
```

外部状態やlegacy認証条件を模した固定fixtureなので、Canonical変更だけを理由に機械的に更新しません。

## 正本と生成物

正本は`reference/workspace`のauthoring sourceです。

生成物はTEI自身の正規経路から確認・再生成し、差分を現在のCanonical authoring、Plugin Lock、fixed observed fixtureから説明できることを確認してから更新します。

次を行いません。

- generated JSONの手編集
- fingerprintだけの置換
- 保存artifactを正本としてCanonicalへ逆書き
- fixed observed fixtureを生成結果へ合わせて変更

## Evidenceと修復を分離する

`-write`はstaleなgenerated artifactを正規状態へ戻すための**修復操作**です。`-write`が成功し、その直後のdirty working treeでcheckが通ったことだけを、Gate Bの完了Evidenceにはしません。

Gate Bへ採用する#255 Evidenceは、後から同じsourceへ戻れる**cleanなrepository revision**へ固定します。

PR #331で、この完了条件を一つのcheck-only runnerへ集約しました。

```text
tools/reference-generated-artifact-evidence
```

runnerは`-write`を呼ばず、次を一つのrunで確認します。

1. 実行前repository revision
2. 実行前working tree clean
3. Go toolchain identity
4. `reference-golden`一致
5. `reference-durable-run-golden`一致
6. 実行後repository revision
7. 実行前後のHEAD同一性
8. 実行後working tree clean
9. `reproducibleRevision`判定

PR #398では、`reproducibleRevision`をrepository sourceだけの状態表示ではなく、**このrunnerの必須golden checkも成立した完了Evidence条件**としてfail closedに揃えました。

- 二つのgolden checkが両方成功する
- post-run repository source確認が完了している（`postRunRepository.checked = true`）
- 実行前後でHEADが同一
- 実行後もworking treeがclean

これらを確認した後にだけ`reproducibleRevision = true`にします。どちらかのgolden checkが失敗した場合、sourceが同じclean revisionのままでも、その失敗JSONではfalseを維持します。

したがって、artifactがstaleだった場合は次の二段階で扱います。

```text
check-only runnerでstale検出
  ↓
個別golden toolの診断を確認
  ↓
必要なtoolだけ-writeで修復
  ↓
git diffで意味上の差分を確認
  ↓
意図したgenerated artifactをcommit
  ↓
新しいclean revision
  ↓
check-only runnerを再実行
  ↓
両golden check成功 + post-run source確認済み
  ↓
reproducibleRevision = true
  ↓
#255完了Evidence候補
```

最初から両goldenが一致している場合は、形式的な`-write`を行わず、そのclean revision上でrunnerが成功した結果をEvidence候補とします。

## 正規toolの責務

### `tools/reference-generated-artifact-evidence`

#255の**最終完了Evidence入口**です。

```bash
go run ./tools/reference-generated-artifact-evidence \
  > /tmp/tei-gate-b-generated-artifact-evidence.json
```

Repository内の新規fileへ直接redirectしません。shellがcommand開始前にfileを作成し、runnerの開始時clean条件を壊すためです。

このrunnerには`-allow-dirty`を設けません。dirty treeでの診断は個別golden toolを利用し、完了Evidenceと混同しません。

runner自身のConformanceでは、check-only commandに`-write`を含めないこと、golden check失敗runで`reproducibleRevision=false`を維持すること、post-run source確認済みの同一clean revisionだけを成功Evidence候補にできることを固定しています。

### `tools/reference-golden`

WorldとPlugin Lockを同じCanonical Workspaceから扱います。

```bash
go run ./tools/reference-golden
```

既定はread-only checkです。

確認対象:

- `reference/world.json`
- `reference/workspace/.tei/plugin-lock.json`
- 保存Durable RunのPlugin Lock依存

staleで、差分を現在のauthoringから説明できる場合だけ更新します。

```bash
go run ./tools/reference-golden -write
```

PR #367以降、World / Plugin Lockの修復writeも既存goldenを先にtruncateしません。同じdirectoryの一時fileへ全内容を書き込み、mode設定、Sync、Closeを完了した後にRenameし、一時fileは成功・失敗のどちらでもcleanupします。

この境界の目的は、生成途中のwrite / sync / close失敗で既存artifactを空または途中内容へ壊さないことです。OS横断のRename atomicityや、World / Plugin Lock / Durable Runの複数artifact全体を一つのtransactionとして扱うことまでは契約にしません。

safe replaceが成功しても、それは修復操作の安全性を示すだけです。`-write`直後のdirty treeを#255完了Evidenceへ格上げせず、修復差分をcommitした新しいclean revisionでcheck-only Evidence runnerを再実行します。

### `tools/reference-durable-run-golden`

PR #318で、Durable Run全体を`tei deployment-start`と同じinternal API経路からcheck/writeする専用toolを追加しました。

```bash
go run ./tools/reference-durable-run-golden
```

生成経路:

```text
Reference Workspace Validation
  ↓
Solution Release exact resolution
  ↓
fixed observed deployment state Validation
  ↓
releaseplan.CompileWithDeployment
  ↓
pluginruntime.BuildPlan
  ↓
Plugin Lock / Runtime Policy fingerprint
  ↓
deploymentdurable.StartWithRuntimePolicy
  ↓
saved Durable Runとの完全一致check
```

staleの場合だけ次で全体を更新します。

```bash
go run ./tools/reference-durable-run-golden -write
```

保存pathを先にtruncateせず、同じdirectoryの一時fileへ全JSONを書き、Sync / Close後にRenameします。安全境界は**生成途中の失敗で既存Durable Runを空にしないこと**です。OS横断のRename atomicityそのものは契約にしません。

PR #326では、`-solution`のCanonical IDを`core.ParseCanonicalID`で検証し、不正な外部入力をpanicではなく診断可能な通常errorとしてfail closedするようにしました。

## 対象artifact

### Canonical World snapshot

```text
reference/world.json
```

Canonical Workspaceから導出します。

### Plugin Lock

```text
reference/workspace/.tei/plugin-lock.json
```

現在のPlugin / Bundle / Environment authoringから生成します。

### Durable Run

```text
reference/order_rollout.durable-run.m0.21.start.json
```

第一Referenceのidentity:

```text
solution: order_rollout
version: 2026.8.27
run-id: order-rollout-prod-m0.21
environment: production
```

Plugin Lock fingerprintだけでなくRuntime PolicyとRun全体の正規生成結果を専用toolで確認します。

## 2026-09-04 静的依存監査

通常Go runnerを使わず確認できる範囲では、保存済みproduction Plugin Lockから計算したEnvironment fingerprintと、保存済みDurable Runの`pluginLockFingerprint`は一致していました。

したがって、保存済み二artifact間のPlugin Lock依存だけが明らかに壊れている状態ではありません。

一方、これは次を証明しません。

- 保存済みWorldが現在のCanonical Workspaceからの正規生成結果と一致すること
- 保存済みPlugin Lockが現在のCanonical Workspaceからの正規生成結果と一致すること
- 保存済みDurable Runが現在のRuntime Policyを含む正規`deployment-start`結果全体と一致すること

これらは通常Go runnerで正規toolを実行して確認します。

## 正規実行手順

具体的な順序・停止条件は`plan/GATE_B_EVIDENCE_RUNBOOK.md`を唯一の正本とします。

### 1. Workspace Validation

```bash
go run ./cmd/tei validate -workspace reference/workspace
```

Validation Errorがあれば生成物確認・更新へ進みません。

### 2. clean revisionからcheck-only Evidence runnerを実行する

```bash
go run ./tools/reference-generated-artifact-evidence \
  > /tmp/tei-gate-b-generated-artifact-evidence.json
```

成功して`reproducibleRevision = true`なら、そのrevisionを#255 Evidence候補にできます。形式的な`-write`は行いません。

失敗した場合は、runnerが実行した個別golden checkの診断を確認し、必要なartifactだけを修復します。PR #398以降、失敗runではsourceが同じclean revisionであっても`reproducibleRevision=false`のままなので、そのJSONを完了Evidenceへ読み替えません。

### 3. World / Plugin Lock check

```bash
go run ./tools/reference-golden
```

一致する場合は書き換えません。

### 4. World / Plugin Lock write — staleの場合だけ

```bash
go run ./tools/reference-golden -write

git diff -- \
  reference/world.json \
  reference/workspace/.tei/plugin-lock.json
```

PR #367以降、このwriteも一時fileへの全量write / Sync / Close / Renameによるsafe replaceです。修復途中の失敗で既存goldenを先にtruncateしません。

### 5. Durable Run full check

```bash
go run ./tools/reference-durable-run-golden
```

Plugin Lockが変わっていなくても、Runtime Policyを含むRun全体の再現性を確認するため実行します。

### 6. Durable Run write — staleの場合だけ

```bash
go run ./tools/reference-durable-run-golden -write

git diff -- reference/order_rollout.durable-run.m0.21.start.json
```

### 7. staleがあった場合は生成物をcommitする

`-write`で差分が生じた場合、差分が正規入力から説明できることを確認し、意図したgenerated artifactだけをcommitします。

DSN、credential、一時Evidence file、fixed observed fixture等を混入させません。

### 8. 新しいclean revisionでEvidence runnerを再実行する

```bash
go run ./tools/reference-generated-artifact-evidence \
  > /tmp/tei-gate-b-generated-artifact-evidence.json
```

両golden checkが成功し、`postRunRepository.checked = true`、実行前後で同じHEAD、両時点のworking treeがcleanで、`reproducibleRevision = true`になったrunだけを#255の完了Evidence候補とします。

`-write`直後のdirty treeで個別checkが成功しても、それだけでは#255を完了にしません。

## 再生成前に解消済みの事項

OrderLine Relation Storage / SQL realizationの重複authoringは再生成前に正本を一本化済みです。

現在の正本:

```text
reference/workspace/databases/sales-core/persistence/order-line.tei
reference/workspace/databases/sales-core/relation-storage/order-line.tei
reference/workspace/databases/sales-core/query-realizations/order-line-find-by-order.tei
reference/workspace/databases/sales-core/query-realizations/order-line-find-by-order.sql
```

`tools/reference-postgres-integration`は独立fixtureを利用しており、OrderLine Persistence Mapping追加だけを理由に既存期待値を変更する必要がないことも確認済みです。

## 他Evidenceとの責務分離

### real PostgreSQL engine Evidence

`ISSUE-000031` / #308が、`tools/reference-postgres-header-detail-runtime`によるreal PostgreSQL engine + ephemeral fixture Evidenceを扱います。

### actual external Oracle Evidence

`ISSUE-000022` / #214が、`master_data`のactual Oracle物理情報 / provenance / read-only Evidenceを扱います。

### deterministic Repository Evidence

`ISSUE-000032` / #310が、clean revisionに対するWorkspace Validation、UI-less Scenario、external Evidence intake等のnormal Go runner Evidenceを扱います。

#255でもWorkspace Validationとclean revision固定を要求しますが、#310のEvidence全体を代替しません。

## 完了条件

- [x] World / Plugin Lockを同じCanonical Workspaceからcheck/writeできるtoolがある
- [x] OrderLine Relation Storage / SQL realizationの重複authoringを解消した
- [x] Durable RunをPlugin Lock / Runtime Policy依存artifactとして特定した
- [x] World / Plugin Lockに依存するReference artifactを洗い出した
- [x] `reference-golden`にDurable Run Plugin Lock stale検知を追加した
- [x] 実PostgreSQL Evidenceを`ISSUE-000031` / #308へ分離した
- [x] actual Oracle Evidenceを`ISSUE-000022` / #214へ分離した
- [x] deterministic Repository Evidenceを`ISSUE-000032` / #310へ分離した
- [x] 保存済みPlugin Lockと保存済みDurable Runのproduction fingerprint相互一致を静的確認した
- [x] Durable Run全体を正規internal APIからcheck/writeする専用toolを追加した（PR #318）
- [x] World / Plugin Lock writeで既存fileを先にtruncateしないsafe replace境界を実装した（PR #367）
- [x] Durable Run writeで既存fileを先にtruncateしないsafe replace境界を実装した
- [x] Runbookからshell `mktemp` / direct `deployment-start`の重複手順を外し、専用toolへ一本化した
- [x] `-write`を修復操作、clean revision上のcheck-onlyを完了Evidenceとして分離した
- [x] Durable Run goldenの不正Solution Canonical IDをpanicではなくfail-closed errorにした（PR #326）
- [x] #255完了条件を一つのcheck-only Evidence runnerへ集約した（PR #331）
- [x] Evidence runnerから`-write`を呼ばないConformanceを固定した
- [x] Evidence runnerで実行前後のHEAD / working treeを確認する境界を固定した
- [x] golden check失敗時に`reproducibleRevision=false`を維持し、post-run source確認済みの成功runだけを完了候補にする（PR #398）
- [x] cleanな通常Go環境でWorkspace Validationを成功させる（#310 childの`workspaceValidation.passed = true`）
- [x] `go run ./tools/reference-generated-artifact-evidence`をclean revisionで実行する（Rでcheck-only成功。`world-plugin-lock-golden` / `durable-run-golden`とも`passed = true`）
- [x] staleの場合だけWorld / Plugin Lockを正規生成し、差分を説明する（該当なし: Rでstale無し、修復不要）
- [x] staleの場合だけDurable Run全体を正規生成し、差分を説明する（該当なし: Rでstale無し、修復不要）
- [x] stale修復が生じた場合、意図したgenerated artifactをcommitする（該当なし: 修復が生じなかった）
- [x] final clean revisionでEvidence runnerを再実行し`reproducibleRevision = true`を確認する（`generated-artifact-reproducibility.json`）
- [x] JSON Evidenceを再確認する（独立artifact auditでfingerprint一致）
- [x] 必要なReference artifactを手編集せず更新する（更新不要で、手編集も無し）
- [x] 結果を`ISSUE-000029` / #273のGate B監査へ対応付ける（Gate B Readiness Audit「Required Evidence 2」）

## 非目標

- snapshotをCanonical authoringの正本として扱うこと
- fingerprintを人手で計算・置換すること
- observed fixtureをCanonical変更だけで更新すること
- external-observed `master_data`の未観測物理情報を生成物へ埋めること
- 実PostgreSQL Provider Evidenceを本Issueへ抱え込むこと
- deterministic Repository Evidence全体を本Issueへ抱え込むこと
- 一致済み生成物を形式的に書き換えること
- dirty working treeのcheck成功をGate B完了Evidenceとして採用すること
- safe replaceの成功だけをGate B完了Evidenceとして採用すること
- golden check失敗runをsource固定だけで完了Evidenceとして採用すること
- Evidence runnerから修復操作を行うこと
- CIをこのIssueから独断で再有効化すること

## 関連

- GitHub Issue #69 / #214 / #255 / #273 / #308 / #310
- GitHub PR #280 / #294 / #299 / #306 / #315 / #318 / #319 / #321 / #323 / #326 / #331 / #367 / #369 / #398
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/issue/000022-external-master-data-physical-query-evidence.md`
- `plan/issue/000029-gate-b-readiness-audit.md`
- `plan/issue/000031-gate-b-postgres-header-detail-evidence.md`
- `plan/issue/000032-gate-b-deterministic-repository-evidence.md`
- `tools/reference-generated-artifact-evidence/`
- `tools/reference-golden/`
- `tools/reference-durable-run-golden/`
- `reference/README.md`
- `reference/world.json`
- `reference/workspace/.tei/plugin-lock.json`
- `reference/order_rollout.durable-run.m0.21.start.json`
