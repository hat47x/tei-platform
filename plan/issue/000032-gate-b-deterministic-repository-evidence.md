# ISSUE-000032: Gate Bのdeterministic Repository Evidenceを通常Go runnerで再現する

- 状態: Ready
- GitHub Issue: #310
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- Gate B監査: `ISSUE-000029` / GitHub Issue #273
- 関連Issue: `ISSUE-000022` / #214、`ISSUE-000025` / #255、`ISSUE-000031` / #308
- 関連RQ: RQ-009 / RQ-012 / RQ-014 / RQ-018

## 目的

Repository checkoutとGo toolchainだけで再現できるGate Bのdeterministic Evidenceを、cleanかつ実行中に不変なrepository revision上で一つの正規入口から実行し、現在の`main`に成立しているApplication Driver / external Evidence intake境界を再現可能なEvidenceとして確認します。

このIssueはCIサービスのgreenを要求するためのものではありません。特定CIサービスとは独立した**normal Go runner Evidence**を取得し、後から実際に実行した同じsource revisionへ戻れることを重視します。

## Evidenceの性質

このIssueで扱うのは**deterministic repository Evidence**です。

主な対象:

- Canonical Workspace Validation
- 最終UI-less Scenario
- external Evidence intake / fail-closed境界
- 必要に応じた広いRepository test
- Evidence実行前後のrepository source identity / cleanliness

成功しても次の代替にはしません。

- `ISSUE-000022` / #214 actual external Oracle asset Evidence
- `ISSUE-000031` / #308 real PostgreSQL engine + ephemeral fixture Evidence
- `ISSUE-000025` / #255 generated Reference artifact再生成Evidence

## 正規runner

PR #315で次を`main`へ追加しています。

```text
tools/reference-gate-b-repository-evidence
```

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
7. 必要に応じたbroad test
8. 実行後HEAD / working treeの再確認
9. 小さい非Canonical JSON Evidence出力

広いtestは明示指定時だけ追加します。

```bash
go run ./tools/reference-gate-b-repository-evidence -broad
```

## cleanかつ固定されたrevision境界

Gate B完了Evidenceは、後から**実際に実行した同じsource**へ戻れる必要があります。

そのためrunnerは既定で実行前の`git status --porcelain`が空であることを要求し、dirty working treeではtargeted test前にfail closedします。

さらにtargeted / broad test後にも、次を再確認します。

- `git rev-parse HEAD`が実行前revisionと同一であること
- `git status --porcelain`が空であること

実行中にHEADが変化した場合や、test / 並行操作によってtracked・untracked差分が残った場合は、testが成功していても単一revisionのGate B完了Evidenceとして採用しません。

診断用の`-allow-dirty`もGate B完了Evidenceには使用しません。`reproducibleRevision=false`として非0終了するためです。

Evidence JSONを保存する場合も、Repository内へ直接redirectして先にdirty状態を作らないよう、まずworking tree外へ保存します。

```bash
go run ./tools/reference-gate-b-repository-evidence \
  > /tmp/tei-gate-b-repository-evidence.json
```

## targeted Evidence

### 最終UI-less Scenario

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

### external Evidence intake

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
- targeted checkごとのcommandとpass / fail
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
- [ ] runner実装で実行後HEAD不変・working tree cleanをfail-closed確認する
- [ ] cleanな対象revisionでrunnerを実行する
- [ ] Canonical Workspace Validation成功Evidenceを取得する
- [ ] 最終UI-less Scenarioを通常Go runnerで成功させる
- [ ] external Evidence intake testを通常Go runnerで成功させる
- [ ] 実行前後の同一revision / clean状態をJSON Evidenceで再確認する
- [ ] repository revisionを含むJSON Evidenceを保存・再確認する
- [ ] 可能なら`-broad`を実行し結果を記録する
- [ ] 結果を`ISSUE-000029` / #273のGate B監査へ対応付ける

## 非目標

- GitHub Actionsの再有効化
- CIサービスのgreenを唯一のEvidenceにすること
- actual Oracle物理情報の推測
- PostgreSQL fixture Evidenceの代替
- generated Reference artifactの手編集
- synthetic Scenarioをactual external Evidenceへ格上げすること
- dirty working treeの成功をGate B完了Evidenceとして採用すること
- 実行中にsource identityが変化したrunを元revisionのEvidenceとして採用すること

## 関連

- GitHub Issue #69 / #214 / #255 / #273 / #308 / #310
- PR #281 / #288 / #300 / #304 / #306 / #315
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `internal/driver/final_gate_b_scenario_test.go`
- `internal/database/externalqueryevidence/`
- `tools/reference-external-query-evidence/`
- `tools/reference-gate-b-repository-evidence/README.md`
