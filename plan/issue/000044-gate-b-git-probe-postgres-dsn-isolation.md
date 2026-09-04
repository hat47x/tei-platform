# ISSUE-000044: Gate B final runnerのGit source probeからPostgreSQL DSNを隔離する

- 状態: Ready to verify
- GitHub Issue: #436
- 親Issue: `ISSUE-000037` / GitHub Issue #387
- 関連Issue: `ISSUE-000043` / #433、`ISSUE-000031` / #308、`ISSUE-000029` / #273

## 目的

Gate B final runner自身がRepository sourceを観測するために起動するGit subprocessへ、不要な`TEI_TEST_POSTGRES_DSN`を渡さないようにします。

#433ではGo環境probe、Go version、#310 / #255 childからDSNを除去しました。本Issueは、runner開始時・各lane後・最終source確認で使う`git rev-parse` / `git status`まで同じleast-exposure境界へ揃えます。

## 背景

`tools/internal/repositoryevidence`は、Evidence sourceを一つのclean revisionへ固定するためにGit subprocessを起動します。

```text
git rev-parse HEAD
git status --porcelain
git rev-parse HEAD
```

PR #437以前、この共有packageの標準`commandOutput`は親process environmentを通常継承していました。

final runnerの親processには#308用`TEI_TEST_POSTGRES_DSN`が必要なため、Git subprocessにもDSNが渡り得ます。GitはPostgreSQLへ接続しないため、この継承は不要です。

## 設計方針

共有`repositoryevidence`へPostgreSQL固有の変数名やsecret規則を持ち込みません。

代わりにcallerが明示environmentを渡せるgeneric入口を追加します。

```go
InspectStartWithEnv(env []string)
InspectPostWithEnv(initialRevision string, env []string)
```

既存APIは維持します。

```go
InspectStart()
InspectPost(initialRevision string)
```

source同一性の意味は変えず、Git subprocessへどのenvironmentを渡すかだけをcallerが選べるようにします。

## final runnerでの利用

final runnerは、既存#433 helperでDSNを除去したenvironmentを渡します。

```go
repositoryevidence.InspectStartWithEnv(
    environmentWithoutPostgresDSN(os.Environ()),
)
```

実行後観測も同じです。

```go
repositoryevidence.InspectPostWithEnv(
    initialRevision,
    environmentWithoutPostgresDSN(os.Environ()),
)
```

これにより次の境界になります。

```text
final runner process (DSNあり)
  ├─ go env / go version probe -> DSNなし
  ├─ git source probe          -> DSNなし
  ├─ #310 child                -> DSNなし
  ├─ #255 child                -> DSNなし
  └─ #308 child                -> DSNあり
```

## source観測契約は変更しない

`InspectStartWithEnv` / `InspectPostWithEnv`は既存の`inspectStart` / `inspectPost`へ接続します。

したがって次の意味は従来どおりです。

- HEADをworking tree観測の前後で確認する
- 観測中のHEAD driftをfail closedにする
- post-runで`Checked=true`になるのはHEAD / working tree / 二度目のHEAD確認がすべて成功した場合だけ
- `IsReproducible`は同一clean revisionを要求する

secret exposureを狭めるためにsource Evidenceの意味を変えません。

## test

共有packageでは、親test processへsecret候補を設定したうえで、明示environmentからそのkeyを除外して現在のtest binaryをchild起動するtestを追加しました。

child側では次を確認します。

- 親secret候補が存在しない
- callerが明示したmarkerは存在する
- callerが渡したenvironment sliceを破壊しない

standard-libraryだけの隔離module結果:

```text
ok  tei-check/repositoryevidence  0.002s
```

これは通常TEI Repository checkout上のpackage testやactual #387 Evidenceではありません。

## #433との関係

```text
#433 / ISSUE-000043
  go env / go version / #310 / #255からDSNを除去

#436 / ISSUE-000044
  final runner自身のgit source probeからDSNを除去
```

#308だけは実PostgreSQLへ接続するため、DSNを保持します。

## Evidenceへの影響

新しいEvidence fieldは追加しません。

次は引き続き通常Evidenceへ保存しません。

- PostgreSQL DSN
- credential / secret
- Database名

本Issueは「保存しない」だけではなく「利用しないprocessへ渡さない」境界を補強します。

## 完了条件

- [x] `repositoryevidence`へgeneric environment注入口を追加する（PR #437）
- [x] 既存`InspectStart` / `InspectPost`を維持する
- [x] final runnerの開始時source probeへDSN除去済みenvironmentを渡す
- [x] 各lane後 / 最終source probeへ同じenvironmentを渡す
- [x] actual subprocess testを追加する
- [x] Repository正本を追加する（本書）
- [ ] tool READMEへGit source probeのleast-exposureを同期する
- [ ] 実TEI checkout上で`go test ./tools/internal/repositoryevidence ./tools/reference-gate-b-final-evidence -count=1`を再確認する
- [ ] actual #387 final runで使用する

最後のactual run自体は#387の完了条件です。本Issue単独の成功をGate B Readyへ格上げしません。

## 非目標

- `repositoryevidence`へPostgreSQL固有keyを埋め込むこと
- #308からDSNを除去すること
- source同一性判定を変更すること
- OS process全体のsecret sandbox化
- GitHub Actionsの再有効化
- 本変更をGate B Evidenceそのものへ格上げすること
