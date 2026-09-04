# Gate B Evidence Runbook

## 目的

Gate Bで残っている作業を、設計議論へ戻さず**再現可能なEvidence取得**として実行するための正規手順です。

Evidenceの必要性・証明範囲・完了条件は[`GATE_B_EVIDENCE_ACQUISITION.md`](GATE_B_EVIDENCE_ACQUISITION.md)、Gate BのReady / Not Ready判定は[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)を正とします。

**具体的なコマンド、実行順、差分確認、fail-closedの停止条件については本書を唯一の正本とします。** 他文書に同じコマンドが残っている場合、本書と矛盾する手順を採用しません。

Gate Bを直接止めているEvidence laneは次の四つです。

1. `ISSUE-000022` / #214: actual Oracle `master_data`物理・provenance・read-only Evidence
2. `ISSUE-000031` / #308: real PostgreSQL engine + ephemeral fixture Evidence
3. `ISSUE-000025` / #255: World / Plugin Lock / Durable Runのgenerated artifact再現Evidence
4. `ISSUE-000032` / #310: deterministic Repository / normal Go runner Evidence

GitHub Actionsが別判断で再開された場合の継続的regression確認は重要ですが、この四laneのProduct Evidenceそのものとは分けます。

## Evidenceの種類

### A. deterministic repository Evidence

Repository checkoutとGo toolchainだけで再現できるApplication Driver / Evidence intakeのEvidenceです。

正規入口:

```text
tools/reference-gate-b-repository-evidence
```

主な対象:

- Canonical Workspace Validation
- 最終UI-less synthetic Scenario
- external Evidence intake tests
- 必要に応じたRepository全体のGo test
- 実行前後のsource revision不変性

### B. generated artifact reproducibility Evidence

Canonical Workspace等の正規入力から、保存generated artifactを同じclean revisionで再現できることのEvidenceです。

正規完了Evidence入口:

```text
tools/reference-generated-artifact-evidence
```

個別check / repair tool:

```text
tools/reference-golden
tools/reference-durable-run-golden
```

`-write`は修復操作であり、完了Evidenceそのものではありません。

### C. real-provider ephemeral fixture Evidence

実provider engineを使いますが、toolが作るephemeral fixtureに対するEvidenceです。

```text
tools/reference-postgres-header-detail-runtime
```

これはproduction既存資産の物理事実を観測したEvidenceではありません。

### D. actual external asset Evidence

TEIが所有していない既存資産の観測事実です。

第一Referenceでは#214のOracle `master_data`が該当します。synthetic / generated artifact / provider fixtureで代替しません。

## 共通原則

- 未観測の外部事実を名前や慣習から補完しない
- Evidence実行前にCanonical WorkspaceのValidityを確認する
- generated artifactを手編集しない
- fingerprintを人手で置換しない
- runtime credential / DSN / parameter値を通常Evidenceへ保存しない
- synthetic / generated / provider fixture / actual external Evidenceを相互に格上げしない
- 失敗を0件・NotSatisfied・denied等の正常業務結果へ変換しない
- 一致済みgenerated artifactを形式的に書き換えない
- `-write`をGate B完了Evidenceそのものとして扱わない
- Gate Bへ採用するEvidenceは、後から同じsourceへ戻れるclean revisionへ固定する
- Evidence runの前後でsource revisionが変化した場合は完了Evidenceとして採用しない
- GitHub Actionsをこの手順から再有効化しない

## 0. 実行前確認

Repository rootで作業します。

```bash
git status --short
git rev-parse HEAD
go version
```

意図しない未commit差分がある場合は先に識別します。既存差分を機械的に破棄しません。

#310と#255の完了Evidenceは、いずれも実行前後で**同じHEADかつclean working tree**であることを要求します。

## 1. Canonical WorkspaceをValidationする

```bash
go run ./cmd/tei validate -workspace reference/workspace
```

期待値:

```text
ok
```

Validation Errorがある場合は、その後のgolden check / repair、provider Evidence、external Evidenceへ進みません。生成物側を手編集して回避しません。

各Evidence tool自身も通常Validation Errorでfail closedしますが、最初の明示Validationは前提不整合を早期に分離するため残します。

## 2. #310 deterministic Repository Evidenceを取得する

clean revisionで次を実行します。

```bash
go run ./tools/reference-gate-b-repository-evidence \
  > /tmp/tei-gate-b-repository-evidence.json
```

Repository内の新規fileへ直接redirectしません。shellがcommand開始前にfileを作成し、開始時clean条件を壊すためです。

runnerは一つのrunで次を記録します。

1. 実行前repository revision
2. 実行前working tree clean
3. Go toolchain identity
4. Canonical Workspace Validation
5. 最終UI-less Scenario
6. external Evidence intake boundary tests
7. 実行後repository revision
8. 実行前後のHEAD同一性
9. 実行後working tree clean

`reproducibleRevision = true`には、実行前後でHEADが同一で、両時点のworking treeがcleanであることを要求します。

### broad test

横断回帰も同じrunで確認する場合は明示指定します。

```bash
go run ./tools/reference-gate-b-repository-evidence -broad \
  > /tmp/tei-gate-b-repository-evidence.json
```

`go test ./...`はtargeted Evidenceとは別checkとして記録します。未実行をgreenとは扱いません。

## 3. #255 generated artifactのcheck-only Evidenceを試す

まず修復せず、clean revisionから正規完了Evidence runnerを実行します。

```bash
go run ./tools/reference-generated-artifact-evidence \
  > /tmp/tei-gate-b-generated-artifact-evidence.json
```

このrunnerは**`-write`を呼びません**。

一つのrunで次を扱います。

1. 実行前repository revision
2. 実行前working tree clean
3. Go toolchain identity
4. `go run ./tools/reference-golden`
5. `go run ./tools/reference-durable-run-golden`
6. 実行後repository revision
7. 実行前後のHEAD同一性
8. 実行後working tree clean
9. 小さいJSON Evidence出力

成功条件:

- `reference-golden`一致
- `reference-durable-run-golden`一致
- 実行前後でHEAD同一
- 実行前後でworking tree clean
- `reproducibleRevision = true`

最初から成功した場合、形式的な`-write`は行いません。このJSONを#255 Evidence候補として扱います。

不一致の場合は#255完了ではなく、staleを検出した診断結果です。次の修復手順へ進みます。

## 4. #255 World / Plugin Lockを修復する — staleの場合だけ

まず個別checkの診断を確認します。

```bash
go run ./tools/reference-golden
```

このtoolはCanonical Workspaceから次を生成して保存goldenと比較します。

```text
reference/world.json
reference/workspace/.tei/plugin-lock.json
```

保存Durable RunのPlugin Lock依存も確認します。

不一致があり、修復が必要な場合だけ実行します。

```bash
go run ./tools/reference-golden -write
```

その後:

```bash
git diff -- \
  reference/world.json \
  reference/workspace/.tei/plugin-lock.json
```

確認観点:

- 現在のauthoring変更から説明できる差分だけか
- 追加済みCapability等が期待どおり反映されたか
- 未観測のexternal physical情報が混入していないか
- 並び順やfingerprintだけを人手で合わせていないか

**この`-write`は修復操作です。** dirty tree上で個別checkが通っても、#255完了Evidenceにはしません。

## 5. #255 Durable Run全体を修復する — staleの場合だけ

対象:

```text
reference/order_rollout.durable-run.m0.21.start.json
```

第一Reference identity:

```text
solution: order_rollout
version: 2026.8.27
run-id: order-rollout-prod-m0.21
environment: production
```

個別check:

```bash
go run ./tools/reference-durable-run-golden
```

内部経路:

```text
Reference Workspace Validation
  -> Solution Release exact resolution
  -> fixed observed deployment state Validation
  -> releaseplan.CompileWithDeployment
  -> pluginruntime.BuildPlan
  -> Plugin Lock / Runtime Policy fingerprint
  -> deploymentdurable.StartWithRuntimePolicy
  -> saved Durable Runとの完全一致check
```

Plugin Lockが変わっていなくても、Durable Run全体を比較します。

不一致があり、修復が必要な場合だけ実行します。

```bash
go run ./tools/reference-durable-run-golden -write
```

保存pathを先にtruncateせず、同じdirectoryの一時fileへ生成結果全体を書き、Sync / Close後にRenameします。

更新後:

```bash
git diff -- reference/order_rollout.durable-run.m0.21.start.json
```

確認観点:

- `pluginLockFingerprint`が現在Plugin Lockと整合する
- `runtimePolicyFingerprint`等の差分を正規生成入力から説明できる
- fixed observed deployment stateを生成結果へ合わせて書き換えていない
- fingerprintやJSON断片だけを手編集していない

PR #326以降、`-solution`の不正Canonical IDもpanicではなく通常errorでfail closedします。

## 6. #255 repair差分をcommitし、final check-onlyを再実行する

World / Plugin Lock / Durable Runのいずれかを修復した場合、dirty treeのままGate B監査へ進みません。

1. `git diff`で差分が現在のCanonical authoring、Plugin Lock、fixed observed deployment stateから説明できることを確認する
2. 意図したgenerated artifactだけをcommitする
3. DSN、credential、一時Evidence file、fixed observed fixture等が混入していないことを確認する
4. 新しいclean revisionへ固定する

そのうえで、#255の正規完了Evidence runnerを再実行します。

```bash
go run ./tools/reference-generated-artifact-evidence \
  > /tmp/tei-gate-b-generated-artifact-evidence.json
```

このrunnerが成功して`reproducibleRevision = true`になったrunだけを#255完了Evidence候補とします。

`-write`直後のdirty treeで個別toolが一致しただけでは#255を完了にしません。

## 7. #308 real PostgreSQL engineでheader-detailを確認する

これは**real PostgreSQL engine + ephemeral fixture runtime Evidence**です。production既存Databaseの実データEvidenceではありません。

### fail-closed前提

Evidence toolは次の順で確認します。

```text
Workspace Load
  ↓
Canonical Validation
  ↓ Errorなら停止
PostgreSQL接続
  ↓
current_database()でDatabase identity確認
  ↓
postgres / template0 / template1なら停止
  ↓
system / temporary領域を除くuser relation件数確認
  ↓ 1件以上なら停止
public.order_lines既存確認
  ↓ 既存なら停止
Ephemeral fixture作成
  ↓
Runtime Evidence
```

`TEI_TEST_POSTGRES_DSN`には、Evidence専用に作成した空PostgreSQL Databaseを指定します。

```bash
export TEI_TEST_POSTGRES_DSN='postgres://...'
```

DSN、credential、接続先の実Database名をEvidenceファイルへ保存しません。

### Database側の安全条件

次の既定Databaseはfixture先として拒否します。

```text
postgres
template0
template1
```

さらに、`pg_catalog` / `information_schema` / PostgreSQL内部・temporary領域を除き、table / partitioned table / view / materialized view / foreign table / sequenceのいずれかが一つでも存在すれば停止します。

空Database検査後も`public.order_lines`の既存有無を明示確認します。

既存relationを削除して続行するoverrideはありません。

### 実行

```bash
go run ./tools/reference-postgres-header-detail-runtime \
  | tee /tmp/tei-gate-b-postgres-header-detail.json
```

期待するEvidence:

- `evidenceNature = real-postgres-ephemeral-fixture`
- `fixtureObject = public.order_lines`
- `preexistingFixtureProtected = true`
- `fixtureOwnedByTool = true`
- Pattern = `order_entry.order_lines`
- Relation = `sales.order.lines`
- Environment = `development`
- Database instance = `sales_core.dev`
- executor Capability = `database.query.postgres`
- parameter names = `order_id`
- parameter values persisted = false
- rows = 2
- zeroRows = 0
- Execution Error observed = true
- `sales.order_line.order_id`がSemantic attributeへ追加されていない
- Relation Storage column = `order_id`
- Semantic mapping applied = true

## 8. #214 actual Oracle Evidenceを受け取る

この工程は、実外部事実が得られるまで**Blockedのまま保持**して構いません。

最低限必要な観測事実:

- physical object kind / identity
- Contract input `id`に対応するphysical parameter
- parameter type / NULL上の扱い
- raw result columns
- raw column type / NULL上の扱い
- raw → `customer.customer` Semantic mapping
- native Oracle SQLまたは他realization source
- source / provenance
- 匿名化・縮約等を行った場合の加工内容
- read-only principal / DB permission / read-only session等の実効的根拠

未観測の`CUSTOMER`、`CUSTOMER_ID`等を仮定しません。

Evidence取得後:

1. 観測事実をEvidence JSONへ記録する
2. 観測事実を根拠に`ContractQuerySQLRealization`を明示authoringする
3. native sourceを通常Workspaceから追跡可能にする
4. Evidenceとauthoringを照合する

```bash
go run ./tools/reference-external-query-evidence \
  -workspace reference/workspace \
  -evidence <observed-evidence.json>
```

SQL realizationがない場合はEvidenceから生成せず停止します。

5. read-onlyを実効強制したOracle connectionでruntime Executorを構成する
6. Workspace → actual read → Semantic resultを一続きに確認する

## 9. 最終差分を確認する

```bash
git status --short

git diff -- \
  reference/world.json \
  reference/workspace/.tei/plugin-lock.json \
  reference/order_rollout.durable-run.m0.21.start.json
```

commit対象は、正規生成されたartifactと意味上説明可能なsource変更だけにします。

DSN、credential、runtime parameter値、一時Evidence fileを誤ってcommitしません。

#255 repairが生じた場合は、dirty差分を残したままGate B監査へ進まず、意図したartifactをcommitしてclean revisionへ固定し、`reference-generated-artifact-evidence`を再実行します。

## 10. Gate B監査へ戻す

取得したEvidenceを[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)へ対応付けます。

| Evidence lane | 正規Evidence | 判定対象 |
| --- | --- | --- |
| #310 | `reference-gate-b-repository-evidence` JSON | UI-less Scenario / external Evidence intake / normal Go runner / source revision不変性 |
| #255 | `reference-generated-artifact-evidence` JSON | World / Plugin Lock / Durable Runのclean revision再現性 |
| #308 | PostgreSQL runtime Evidence JSON | real PostgreSQL engine上の専用空Database / fixture runtime |
| #214 | observed Oracle Evidence + actual runtime Evidence | external-observed assetの実物理事実とread-only境界 |

#255の個別`reference-golden` / `reference-durable-run-golden`はcheck / repairの正規toolですが、Gate Bへ対応付ける最終Evidenceは`reference-generated-artifact-evidence`のclean check-only runとします。

一つのEvidenceが成功したことを別laneのEvidenceの代替にしません。

## 停止条件

次の場合はfail closedで止めます。

- Workspace Validation Error
- #310完了Evidenceをdirty working treeから採用しようとしている
- #310 runner実行前後でHEADが変わった、または実行後working treeがdirtyになった
- #255完了Evidenceを`-write`直後のdirty treeから採用しようとしている
- #255 `reference-generated-artifact-evidence`でgolden不一致がある
- #255 runner実行前後でHEADが変わった、またはworking treeがdirtyになった
- #308で`postgres` / `template0` / `template1`へ接続している
- #308の接続先にsystem / temporary領域以外のuser relationが1件以上存在する
- `public.order_lines`が既に存在するPostgreSQL接続先
- #214でprovenanceのない物理情報しかない
- read-only保証をSQL文字列だけで主張している
- generated artifactがstaleなのにfingerprintやJSON断片だけを手編集しようとしている
- fixed observed deployment stateを生成結果へ合わせて機械的に変更しようとしている
- EvidenceとSQL realizationが一致しない
- 実行失敗を0件・正常Business Resultへ変換しないと先へ進めない

これらは「作業が止まった」のではなく、Gate Bが意図どおり不足を露出させたEvidenceとして扱います。
