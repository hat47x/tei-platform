# ISSUE-000062: Gate B最終EvidenceのGo実行環境をcandidate revisionの外へ広げない

- 状態: Done — Gate B Ready（R = `e5231839`）のactual #387 final runで確認済み（2026-09-19）
- GitHub Issue: #421
- Gate B最終収束: `ISSUE-000037` / #387
- Gate B監査: `ISSUE-000029` / #273
- 関連: `ISSUE-000063` / #424、`ISSUE-000065` / #430

## 目的

Gate Bの三lane最終収束で、clean candidate revision `R`の外にあるGo workspace、GOFLAGS、user-level Go environment fileをcompile / testの暗黙入力へしません。

Repository source固定とGo実行入力固定は別の境界として扱います。

## 現行境界

final runnerはlane開始前に少なくとも次を確認します。

```text
GOENV = exact "off"
go env GOWORK = empty または off
go env GOFLAGS = empty
```

外部`go.work` pathや非空GOFLAGSを検出した場合はfail closedで停止します。

値そのものは通常Evidenceへ保存しません。

- external workspace pathを保存しない
- GOFLAGS本文を保存しない
- GOENV path /拒否値を保存しない

## compile前とruntimeを分ける

具体的な正規起動commandは`plan/GATE_B_EVIDENCE_RUNBOOK.md`を唯一の正本とします。

現行contractではrunner自身のcompile前から概念上次を明示します。

```text
GOENV=off
GOWORK=off
GOFLAGS=
```

runtime guardだけで安全値へ書き換えて、compile時に不安全だった環境を後から安全とみなしません。

ambient確認後のchild / nested Go環境固定は`ISSUE-000063`、compile前のexact `GOENV=off` guardは`ISSUE-000065`が担当します。

## final Evidence

成功JSONでは少なくとも次を確認します。

```text
goEnvironment.externalWorkspaceActive = false
goEnvironment.goFlagsEmpty = true
goEnvironment.goEnvDisabled = true
```

これはGo toolchainやmodule cacheをRepositoryへvendoringしたことを意味しません。正規Evidence経路へ明示していないGo設定を持ち込まないための境界です。

## secret境界との分離

PostgreSQL secretのleast-exposureは`ISSUE-000043`〜`ISSUE-000045`で扱います。

Go環境固定を理由にDSN本文やsecret file pathの露出範囲を広げません。

現行final runnerではGo environment probeへPostgreSQL DSN本文もsecret file pathも渡しません。

## 成立済み

- [x] GOWORK / GOFLAGS runtime guardを実装（PR #422）
- [x] external workspace / non-empty GOFLAGSをfail closedで拒否
- [x] path / flags本文を通常Evidenceへ保存しない
- [x] pure / isolated testを追加
- [x] child Go環境固定を`ISSUE-000063`で実装
- [x] exact `GOENV=off` compile-before-runtime guardを`ISSUE-000065`で実装
- [x] final runner tool READMEへ同期
- [x] `plan/GATE_B_EVIDENCE_RUNBOOK.md`へ同期

## 残作業

- [x] 実TEI checkoutで次を再確認する（2026-09-21にmain 9f4fcd5でok）

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
go test ./tools/reference-gate-b-final-evidence -count=1
```

- [x] actual #387 final runで`goEnvironment`三条件を確認する（`externalWorkspaceActive = false` / `goFlagsEmpty = true` / `goEnvDisabled = true`）

actual run自体は#387の完了条件です。本Issue単独の成功をGate B Readyへ格上げしません。

## 非目標

- Go toolchain / module cacheのRepository vendoring
- OS process全体のsandbox化
- 個別lane runnerの置換
- Go環境guardだけでGate B Readyを宣言すること
- GitHub Actions再有効化
