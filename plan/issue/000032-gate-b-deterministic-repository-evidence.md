# ISSUE-000032: Gate Bのdeterministic Repository Evidenceを通常Go runnerで再現する

- 状態: Done — Gate B Ready（R = `e5231839`）のactual #387 final runで確認済み（2026-09-19）
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
- `ISSUE-000025` / #255 generated Reference artifact再生成Evidence

## 正規runner

PR #315で次を`main`へ追加しています。

```text
tools/reference-gate-b-repository-evidence
```

PR #321で、targeted / broad test後にもHEAD不変・working tree cleanを再確認するfail-closed境界を追加しました。

PR #336で、このsource観測規則を#255 / #308と同じ`tools/internal/repositoryevidence`へ収束しました。

PR #346では、共有helperのpure Conformance testを3本目のtargeted checkへ追加し、`-broad`なしでもsource固定判定そのものの回帰を検知できるようにしました。

PR #351では、開始時working treeがcleanでも、それだけで`reproducibleRevision=true`を一時設定しないようにしました。`reproducibleRevision`はpost-runのHEAD同一性とworking tree cleanまで確認できた後にだけtrueになります。

PR #400ではさらに、targeted testまたは明示したbroad testが失敗したrunを、sourceだけが同じclean revisionだったことを理由に`reproducibleRevision=true`へしないようにしました。必須checkの成功とpost-run source固定の両方が揃った後にだけ、完了Evidenceとしてtrueになります。

PR #412では、targeted / broad checkをすべて実行した後だけではなく、**各checkの直後にもRepository sourceを再確認**するようにしました。開始時cleanな通常Evidence runでHEAD変化、source観測不能、新しいworking tree差分を検出した場合は、その時点で後続checkを実行しません。`-allow-dirty`診断では開始時から存在するdirty状態だけを許容しますが、HEAD変化やsource観測不能は同様にfail closedします。

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
7. Repository source固定境界のpure Conformance test
8. 必要に応じたbroad test
9. 各check直後と最終時点でのHEAD / working tree再確認
10. 小さい非Canonical JSON Evidence出力

広いtestは明示指定時だけ追加します。

```bash
go run ./tools/reference-gate-b-repository-evidence -broad
```

## cleanかつ固定されたrevision境界

Gate B完了Evidenceは、後から**実際に実行した同じsource**へ戻れる必要があります。

そのためrunnerは既定で実行前の`git status --porcelain`が空であることを要求し、dirty working treeではtargeted test前にfail closedします。

さらにPR #412以降、各targeted / broad checkの直後に次を再確認します。

- `git rev-parse HEAD`が実行前revisionと同一であること
- Repository source観測が完了していること
- 開始時cleanな通常Evidence runでは`git status --porcelain`が引き続き空であること

開始時cleanだったrunで、test / 並行操作によってtracked・untracked差分が新しく残った場合や、HEADが変化した場合、source観測を完了できなかった場合は、後続checkへ進みません。すべてのcheck終了後にも最終post-run確認を行い、途中と最後の両方でsource固定を確認します。

診断用の`-allow-dirty`では、開始時から存在するdirty working treeそのものは各check間の停止条件にしません。ただしHEAD変化やsource観測不能は診断runでも拒否します。また、最終的にclean revisionへ固定できないため`reproducibleRevision=false`となり、Gate B完了Evidenceには使用できません。

### 必須check成功とpost-run確認が揃うまで再現可能と判定しない

PR #351以降、開始時working treeがcleanでも、開始時点では`reproducibleRevision`をfalseのまま保持します。

PR #400以降はsource固定条件だけでなく、そのrunで要求したcheckの成功も完了条件に含めます。次を最後まで確認できた場合にだけ、`repositoryevidence.IsReproducible`からtrueへします。

- Canonical Workspace Validationが成功している
- 三つのtargeted checkがすべて成功している
- `-broad`指定時はbroad testも成功している
- 各check直後のsource確認を通過している
- 開始revisionが空でない
- 開始時working treeがclean
- post-run状態を確認済み（`postRunRepository.checked = true`）
- post-run revision文字列が開始revisionと同一
- `revisionStable=true`
- 終了時working treeがclean

したがって、Go toolchain取得、Workspace Load / Validation等でpost-run確認前に失敗したJSONはもちろん、targeted testやbroad testが失敗したJSONも、sourceが同じclean revisionのままであっても`reproducibleRevision=false`のままです。

これにより、「sourceは変わらなかった」という一事実や「開始時点ではcleanだった」という一事実を、「そのrunの必須Evidenceが成立した」という完了Evidenceへ誤昇格させません。

Evidence JSONを保存する場合も、Repository内へ直接redirectして先にdirty状態を作らないよう、まずworking tree外へ保存します。

```bash
go run ./tools/reference-gate-b-repository-evidence \
  > /tmp/tei-gate-b-repository-evidence.json
```

## targeted Evidence

### 1. 最終UI-less Scenario

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

等価コマンド:

```bash
go test ./internal/driver \
  -run '^TestOrderEntryGateBScenarioConnectsMajorBoundariesWithoutUI$' \
  -count=1
```

### 2. external Evidence intake

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
- source-fixed acceptance receiptのpure contractを壊していない

等価コマンド:

```bash
go test ./internal/database/externalqueryevidence \
  ./tools/reference-external-query-evidence \
  -count=1
```

このtest成功を#214 actual Oracle facts / actual readのEvidenceへ格上げしません。

### 3. Repository source固定境界

PR #346以降、共有helper自体のpure Conformanceをtargeted checkとして独立実行します。

```bash
go test ./tools/internal/repositoryevidence \
  -count=1
```

少なくとも次を固定します。

- 開始revisionが空でない
- 開始時working treeがclean
- post-run状態が確認済み
- post-run revision文字列が開始revisionと同一
- `revisionStable`がtrue
- 終了時working treeがclean
- どれか一つでも欠ける状態を再現可能と判定しない

runner自身のpackageはこのtargeted testへ含めません。将来runner側のtestが`run()`を呼ぶようになっても自己再帰の経路を作らず、共有helperの契約だけを独立して検証します。

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
- `final-ui-less-scenario`のcommand / pass-fail
- `external-evidence-intake`のcommand / pass-fail
- `repository-evidence-source-boundary`のcommand / pass-fail
- broad testを要求したか

`reproducibleRevision = true`になるのは、Workspace Validation、三つのtargeted check、明示した場合のbroad checkが成功し、各check直後のsource確認を通過し、さらに実行前後で同じHEADを保持し、両時点のworking treeがcleanであることを**最終post-run確認まで完了した場合だけ**です。check失敗や早期失敗ではfalseのままです。

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
- [x] source revision観測をtool-only共有境界へ収束した（PR #336）
- [x] Repository source固定境界のpure Conformanceをtargeted checkへ追加した（PR #346）
- [x] post-run確認前の早期失敗で`reproducibleRevision=true`を残さない（PR #351）
- [x] targeted / broad check失敗時にも`reproducibleRevision=false`を維持する（PR #400）
- [x] targeted / broad checkごとにRepository sourceを再確認し、通常Evidence runでsource変化を検出したら後続checkを止める（PR #412）
- [x] cleanな対象revisionでrunnerを実行する
- [x] Canonical Workspace Validation成功Evidenceを取得する（`workspaceValidation.passed = true`）
- [x] 最終UI-less Scenarioを通常Go runnerで成功させる（check `final-ui-less-scenario`）
- [x] external Evidence intake testを通常Go runnerで成功させる（check `external-evidence-intake`）
- [x] Repository source固定境界testを通常Go runnerで成功させる（check `repository-evidence-source-boundary`）
- [x] 実行前後の同一revision / clean状態をJSON Evidenceで再確認する（`postRunRepository.revisionStable = true` / `workingTreeClean = true`）
- [x] repository revisionを含むJSON Evidenceを保存・再確認する（`repositoryRevision = R`。Repository外へ保存）
- 任意: `-broad`は実施していない（`broadRequested = false`）。Gate B必須Evidenceではなく、通常の`go test ./...`は日次のhealth checkが担う
- [x] 結果を`ISSUE-000029` / #273のGate B監査へ対応付ける（Gate B Readiness Audit「Required Evidence 1」）

## 非目標

- GitHub Actionsの再有効化
- CIサービスのgreenを唯一のEvidenceにすること
- actual Oracle物理情報の推測
- PostgreSQL fixture Evidenceの代替
- generated Reference artifactの手編集
- synthetic Scenarioをactual external Evidenceへ格上げすること
- dirty working treeの成功をGate B完了Evidenceとして採用すること
- 実行中にsource identityが変化したrunを元revisionのEvidenceとして採用すること
- 開始時cleanだけでrun全体を再現可能と判定すること
- test失敗runをsource固定だけで完了Evidenceへ採用すること

## 関連

- GitHub Issue #69 / #214 / #255 / #273 / #308 / #310
- PR #281 / #288 / #300 / #304 / #306 / #315 / #321 / #336 / #340 / #346 / #351 / #352 / #400 / #412
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `internal/driver/final_gate_b_scenario_test.go`
- `internal/database/externalqueryevidence/`
- `tools/internal/repositoryevidence/`
- `tools/reference-external-query-evidence/`
- `tools/reference-gate-b-repository-evidence/README.md`

## 2026-09-05 通常Go環境での修復・検証記録

計画整理commit `89eb739` から通常Repositoryの検証を開始したところ、統合済みのSemantic Validation、Relation key、SourceError、CLIに重複実装があり、Evidence入口のcompileが失敗しました。修復commitは `3b92850` です。

- 現行Referenceの `positive` / `relation-unique` と、Relation直後の `key source -> target` 構文へ定義を一本化しました。
- ApplicationのValidation参照とCapability Outcome診断を共通Validateへ接続し、Query result mappingの二重検査を解消しました。
- CLIのsource診断、Contract Query row属性の所属検査、external Evidence intakeの重複処理、既存テストの型名・fixture参照・構文エラーを修復しました。

検証環境はWSL Ubuntu / Go 1.26.5 linux/amd64です。`89eb739` の独立Linux checkoutへ修復差分を適用して検証しました。これは修復中のdirty checkoutでの確認であり、clean candidate revision Rの最終Evidenceではありません。

| 検証 | 結果 |
| --- | --- |
| `go test ./... -run '^$'` | 全packageのcompile成功。テスト本体は未実行 |
| `go test ./internal/semantic/... ./internal/syntax ./internal/validation ./internal/security/legacyauth ./internal/database/providers/postgres ./tools/reference-external-query-evidence` | 8 package成功 |
| `go test ./...` | 失敗。`cmd/tei`、`internal/driver`、`internal/workspace` に残件 |

次の作業はこのIssueと既存Issueの範囲で継続します。

1. Source IndexのSemantic anchorがRelation直後の `key` を扱えない問題、およびContract内のQueryをトップレベル宣言として探索している問題を修復する。
2. Referenceに明示的なOrderLine Persistence / SQL realization / executorが追加された後も「未定義」を期待するテストを、現行Referenceの肯定確認と、明示的に実現を除いたfixtureのfail-closed確認へ整理する。認証input名とexecutor件数の古い期待値も現行契約と照合する。
3. `TestReferenceWorkspaceNormalizesToGoldenWorld` の不一致は [ISSUE-000025](000025-reference-generated-artifacts-refresh.md) で正規生成経路から解決する。手修正や検査の無効化で通さない。
4. clean revisionでRepository Evidenceを再取得し、[ISSUE-000037](000037-gate-b-three-lane-final-evidence.md) の三lane収束へ進む。

検証logは実行端末の `/tmp/tei-gate-b-build-check.log`、`/tmp/tei-gate-b-targeted-check.log`、`/tmp/tei-gate-b-repair-final-test.log` に保存しました。これらは一時的な修復logであり、配布可能な最終Evidenceではありません。実PostgreSQL / actual OracleのEvidenceはこの検証では取得しておらず、Gate B Ready判定は変更しません。
