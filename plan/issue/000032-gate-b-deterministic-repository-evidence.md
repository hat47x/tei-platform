# ISSUE-000032: Gate Bのdeterministic Repository Evidenceを通常Go runnerで再現する

- 状態: Ready
- GitHub Issue: #310
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- Gate B監査: `ISSUE-000029` / GitHub Issue #273
- 関連Issue: `ISSUE-000022` / #214、`ISSUE-000025` / #255、`ISSUE-000031` / #308
- 関連RQ: RQ-009 / RQ-012 / RQ-014 / RQ-018

## 目的

Repository checkoutとGo toolchainだけで再現できるGate Bのdeterministic Evidenceを、cleanかつ実行中に不変なrepository revision上で一つの正規入口から実行し、現在の`main`に成立しているApplication Driver / external Evidence intake / Repository source固定境界を再現可能なEvidenceとして確認します。

このIssueはCIサービスのgreenを要求するためのものではありません。特定CIサービスとは独立した**normal Go runner Evidence**を取得し、後から実際に実行した同じsource revisionへ戻れることを重視します。

## Evidenceの性質

このIssueで扱うのは**deterministic repository Evidence**です。

主な対象:

- Canonical Workspace Validation
- 最終UI-less Scenario
- external Evidence intake / fail-closed境界
- Repository source固定境界のpure Conformance
- 必要に応じた広いRepository test
- Evidence実行前後のrepository source identity / cleanliness

成功しても次の代替にはしません。

- `ISSUE-000022` / #214 actual external Oracle asset Evidence
- `ISSUE-000031` / #308 real PostgreSQL engine + ephemeral fixture Evidence
- `ISSUE-000025` / #255 generated Reference artifact reproducibility Evidence

## 正規runner

PR #315で次を`main`へ追加しています。

```text
tools/reference-gate-b-repository-evidence
```

PR #321で、targeted / broad test後にもHEAD不変・working tree cleanを再確認するfail-closed境界を追加しました。

PR #336で、source状態の観測規則を#255 / #308と共通の`tools/internal/repositoryevidence`へ収束しました。

PR #346で、その共有source境界のpure Conformanceをtargeted runnerの独立checkへ追加しました。

既定実行:

```bash
go run ./tools/reference-gate-b-repository-evidence
```

このrunnerは一つのrunで次を扱います。

1. 実行前repository revision取得
2. 実行前working tree clean確認
3. Go toolchain identity取得
4. Canonical Workspace Validation
5. 最終UI-less Scenario test
6. external Evidence intake test
7. Repository source境界Conformance test
8. 必要に応じたbroad test
9. 実行後HEAD / working treeの再確認
10. 小さい非Canonical JSON Evidence出力

広いtestは明示指定時だけ追加します。

```bash
go run ./tools/reference-gate-b-repository-evidence -broad
```

## targeted Evidence

PR #346以降、targeted runnerは次の三つを**別々のcheck**として扱います。

### 1. 最終UI-less Scenario

等価コマンド:

```bash
go test ./internal/driver \
  -run '^TestOrderEntryGateBScenarioConnectsMajorBoundariesWithoutUI$' \
  -count=1
```

同じ`order_entry` Application / Semantic Order identityから、少なくとも次の境界を一続きに確認します。

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

synthetic adapterを含むため、actual external state Evidenceへ格上げしません。

### 2. external Evidence intake

等価コマンド:

```bash
go test ./internal/database/externalqueryevidence \
  ./tools/reference-external-query-evidence \
  -count=1
```

少なくとも次を確認します。

- external EvidenceとCanonical Worldの分離
- target / provider / ownership照合
- explicit input → physical parameter照合
- raw result → Semantic mapping照合
- native source fingerprint stale検知
- provenance必須
- effective read-only mechanism / Evidence参照必須
- EvidenceからSQL realizationを生成・補完しない
- Invalid Workspaceを受け入れない
- source-fixed acceptance receiptのConformance
- Evidence artifact fingerprintのConformance

このcheckが成功しても#214 actual Oracle Evidenceにはしません。

### 3. Repository source境界Conformance

等価コマンド:

```bash
go test ./tools/internal/repositoryevidence -count=1
```

JSONでは`repository-evidence-source-boundary`として独立checkに残します。

少なくとも次のpure契約を検査します。

- cleanな同一revisionの開始 / 終了だけを再現可能と判定する
- dirty startを拒否する
- post-run未確認を拒否する
- HEAD変化を拒否する
- post-run dirtyを拒否する
- revision文字列と`RevisionStable`の矛盾を受理しない

runner自身のpackage `./tools/reference-gate-b-repository-evidence`はtargeted `go test`へ含めません。将来runner testが`run()`を呼ぶようになっても自己再帰の経路を作らず、共有helperのpure契約だけを検査します。

このConformanceがrunnerへ組み込まれていることと、normal Go runnerで#310実Evidenceを取得済みであることは別です。

## cleanかつ固定されたrevision境界

Gate B完了Evidenceは、後から**実際に実行した同じsource**へ戻れる必要があります。

runnerは既定で実行前の`git status --porcelain`が空であることを要求し、dirty working treeではtargeted test前にfail closedします。

さらにtargeted / broad test後にも次を再確認します。

- `git rev-parse HEAD`が実行前revisionと同一であること
- `git status --porcelain`が空であること

実行中にHEADが変化した場合や、test / 並行操作によってtracked・untracked差分が残った場合は、testが成功していても単一revisionのGate B完了Evidenceとして採用しません。

診断用の`-allow-dirty`もGate B完了Evidenceには使用しません。`reproducibleRevision=false`として非0終了するためです。

Evidence JSONを保存する場合も、Repository内へ直接redirectして先にdirty状態を作らないよう、working tree外へ保存します。

```bash
go run ./tools/reference-gate-b-repository-evidence \
  > /tmp/tei-gate-b-repository-evidence.json
```

## broad test

横断回帰も同じrunで確認できる環境では`-broad`を明示します。

```bash
go run ./tools/reference-gate-b-repository-evidence -broad \
  > /tmp/tei-gate-b-repository-evidence.json
```

`-broad`は三つのtargeted check成功後に`go test ./...`を追加checkとして実行します。

broad testの失敗をtargeted check成功へ畳み込まず、逆にbroad未実行をgreenとも扱いません。

## Evidence JSON

通常Evidenceには少なくとも次を残します。

- `evidenceNature = deterministic-repository`
- 実行前repository revision
- 実行前working tree clean
- `reproducibleRevision`
- 実行後repository revision
- 実行前後のrevision同一性
- 実行後working tree clean
- Go version
- Workspace Validation結果
- `final-ui-less-scenario` check
- `external-evidence-intake` check
- `repository-evidence-source-boundary` check
- broad testを要求したか

`reproducibleRevision = true`になるのは、実行前後で同じHEADを保持し、両時点のworking treeがcleanである場合だけです。

次は保存しません。

- test stdout / stderr
- credential / secret
- DSN
- runtime parameter value

## 完了条件

- [x] 専用runnerを`main`へ用意した（PR #315）
- [x] 実行前clean revisionを既定のfail-closed条件にした
- [x] Workspace Validationをtargeted testより先に実行する
- [x] 最終Scenarioとexternal Evidence intakeを別checkとして保持する
- [x] broad testを明示オプションへ分離した
- [x] Evidence JSONへsecret / test stdoutを取り込まない境界を定めた
- [x] runner実装で実行後HEAD不変・working tree cleanをfail-closed確認する（PR #321）
- [x] source状態観測を#255 / #308と同じtool-only packageへ収束した（PR #336）
- [x] Repository source境界のpure Conformanceを第三targeted checkへ追加した（PR #346）
- [x] runner自身のpackageをtargeted source-boundary testへ含めず自己再帰を避ける（PR #346）
- [ ] cleanな対象revisionでrunnerを実行する
- [ ] Canonical Workspace Validation成功Evidenceを取得する
- [ ] 最終UI-less Scenarioを通常Go runnerで成功させる
- [ ] external Evidence intake testを通常Go runnerで成功させる
- [ ] Repository source境界Conformanceを通常Go runnerで成功させる
- [ ] 実行前後の同一revision / clean状態をJSON Evidenceで再確認する
- [ ] repository revisionを含むJSON Evidenceを保存・再確認する
- [ ] 可能なら`-broad`を実行し結果を記録する
- [ ] 結果を`ISSUE-000029` / #273のGate B監査へ対応付ける

## 非目標

- GitHub Actionsの再有効化
- CIサービスのgreenを唯一のEvidenceにすること
- actual Oracle物理情報の推測
- PostgreSQL fixture Evidenceの代替
- generated Reference artifact Evidenceの代替
- synthetic Scenarioをactual external Evidenceへ格上げすること
- source-boundary Conformance成功を#214 / #308 / #255の代替にすること
- dirty working treeの成功をGate B完了Evidenceとして採用すること
- 実行中にsource identityが変化したrunを元revisionのEvidenceとして採用すること

## 関連

- GitHub Issue #69 / #214 / #255 / #273 / #308 / #310
- PR #281 / #288 / #300 / #304 / #306 / #315 / #321 / #336 / #346 / #347
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `internal/driver/final_gate_b_scenario_test.go`
- `internal/database/externalqueryevidence/`
- `tools/internal/repositoryevidence/`
- `tools/reference-external-query-evidence/`
- `tools/reference-gate-b-repository-evidence/README.md`
