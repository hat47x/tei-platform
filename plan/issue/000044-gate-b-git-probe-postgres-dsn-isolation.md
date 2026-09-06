# ISSUE-000044: Gate B final runnerのGit source probeからPostgreSQL secretを隔離する

- 状態: Ready to verify — 実装 / isolated test / tool README / Runbook同期済み、actual checkout / final run待ち
- GitHub Issue: #436
- 親: `ISSUE-000037` / #387
- 関連: `ISSUE-000043` / #433、`ISSUE-000045` / #441

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

final runnerはDSN本文とsecret file pathを除去したenvironmentをGit probeへ渡します。

```text
final runner
  ├─ Go environment / version probe -> DSN本文なし / file pathなし
  ├─ Git source probe               -> DSN本文なし / file pathなし
  ├─ #310                           -> DSN本文なし / file pathなし
  ├─ #255                           -> DSN本文なし / file pathなし
  └─ #308                           -> in-memory DSN本文のみ / file pathなし
```

Git probeはPostgreSQLへ接続しないためsecretを必要としません。

## source観測契約

secret exposureを狭めるためにsource Evidenceの意味は変えません。

`repositoryevidence`は概念上次を観測します。

```text
HEAD before
  ↓
git status --porcelain
  ↓
HEAD after
```

- 観測中HEAD driftをfail closedで拒否
- post-runの`Checked=true`は全観測成功時のみ
- `IsReproducible`は同一clean revisionを要求

この境界は`ISSUE-000038` / #418と共通です。

## 成立済み

- [x] `repositoryevidence`へgeneric environment注入口追加（PR #437）
- [x] 既存API維持
- [x] runner開始時source probeへsecret除去済みenvironmentを利用
- [x] 各lane後 / final source probeも同じenvironmentを利用
- [x] actual child subprocess test
- [x] `ISSUE-000045`でsecret file pathも除去対象へ拡張
- [x] tool READMEへGit source probe least-exposureを同期
- [x] `plan/GATE_B_EVIDENCE_RUNBOOK.md`へ同期

## 残作業

- [ ] 実TEI checkoutで次を再確認する

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
go test \
  ./tools/internal/repositoryevidence \
  ./tools/reference-gate-b-final-evidence \
  -count=1
```

- [ ] actual #387 final runで同じGit probe境界を使用する

本Issue単独の成功をGate B Readyへ格上げしません。

## 非目標

- `repositoryevidence`へPostgreSQL固有keyを埋め込むこと
- source同一性判定の変更
- #308から必要なDSN本文を除去すること
- OS process全体のsecret sandbox化
- GitHub Actions再有効化
