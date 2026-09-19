# ISSUE-000044: Gate B final runnerのGit source probeからPostgreSQL secretを隔離する

- 状態: Done — Gate B Ready（R = `e5231839`）のactual #387 final runで確認済み（2026-09-20）
- GitHub Issue: #436
- 親: `ISSUE-000037` / #387
- 関連: `ISSUE-000043` / #433、`ISSUE-000045` / #441、`ISSUE-000061` / #418、GitHub #444

## 目的

final runner自身がRepository sourceを観測するために起動するGit subprocessへ、PostgreSQL接続に不要なsecretを渡しません。

Go utility probeだけでなく、runner開始時・各lane後・final確認のGit source probeまで同じleast-exposure境界へ揃えます。

## 共有source helper

`tools/internal/repositoryevidence`にはProvider固有のsecret名を持ち込みません。

callerが明示environmentを渡せるgeneric入口を使います。

```text
InspectStartWithEnv(env)
InspectPostWithEnv(initialRevision, env)
```

既存`InspectStart` / `InspectPost`のsource同一性契約は維持します。

## 現行final runnerでの利用

final runnerはDSN本文とDSN file pathを除去したenvironmentをGit probeへ渡します。temporary handoff pathもGit probeへ追加しません。

```text
final runner
  ├─ Go environment / version probe
  │    -> DSN本文なし / original pathなし / handoff pathなし
  ├─ Git source probe
  │    -> DSN本文なし / original pathなし / handoff pathなし
  ├─ #310
  │    -> DSN本文なし / original pathなし / handoff pathなし
  ├─ #255
  │    -> DSN本文なし / original pathなし / handoff pathなし
  └─ #308 go run / compiler / linker
       -> DSN本文なし / original pathなし / handoff pathのみ
```

#308 runtimeはhandoff fileを読んだ後、DSN / file path environmentをsource probe前に消去します。

Git probeはPostgreSQLへ接続しないため、いずれのPostgreSQL secret transportも必要としません。

## source観測契約

secret exposureを狭めるためにsource Evidenceの意味は変えません。

PR #419 / #477 / #479 / #483以降、`repositoryevidence`は開始時・終了時とも概念上次を観測します。

```text
HEAD before
  ↓
git status --porcelain --untracked-files=all --ignored=matching
  ↓
git ls-files -v -z
  ↓
HEAD after
```

- 観測中HEAD driftをfail closedで拒否
- post-runの`Checked=true`は全観測成功時のみ
- tracked / untracked / ignored local stateをcleanへ格上げしない
- tracked entryは通常`H` tagだけをcleanとする
- `assume-unchanged` / `skip-worktree` / malformed index stateを拒否
- Git identity override / `GIT_CONFIG_*` command-scope injectionを観測前に拒否
- `IsReproducible`は同一clean revisionを要求
- Gate B Evidenceはfull checkoutを前提とし、sparse由来のskip-worktreeを許容しない

この境界は`ISSUE-000061` / #418と共通です。

## 成立済み

- [x] `repositoryevidence`へgeneric environment注入口追加（PR #437）
- [x] 既存API維持
- [x] runner開始時source probeへsecret除去済みenvironmentを利用
- [x] 各lane後 / final source probeも同じenvironmentを利用
- [x] actual child subprocess test
- [x] `ISSUE-000045`でoriginal secret file pathも除去対象へ拡張
- [x] PR #473で#308 compile環境をtemporary path-only handoffへ変更
- [x] Git probeへtemporary handoff pathを伝播しない
- [x] PR #477 / #479 / #483でGit config injection / ignored local state / index state guardをcurrent source contractへ追加
- [x] tool READMEへGit source probe least-exposureを同期
- [x] `plan/GATE_B_EVIDENCE_RUNBOOK.md`へ同期

## 残作業

- [x] 実TEI full checkoutで次を再確認する

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
go test \
  ./tools/internal/repositoryevidence \
  ./tools/reference-gate-b-final-evidence \
  -count=1
```

- [x] actual #387 final runで同じGit probe境界を使用する
- [x] actual runでGit probeへDSN本文 / original secret path / handoff pathが渡っていないことを確認する

本Issue単独の成功をGate B Readyへ格上げしません。

## 非目標

- `repositoryevidence`へPostgreSQL固有keyを埋め込むこと
- source同一性判定の再設計
- #308 runtimeから必要なDSN利用を除去すること
- OS process全体のsecret sandbox化
- CI greenをGate B Product Evidenceへ格上げすること
