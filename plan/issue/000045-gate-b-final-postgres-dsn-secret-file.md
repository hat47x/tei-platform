# ISSUE-000045: Gate B final runnerのcompile段階からPostgreSQL DSN本文を隔離する

- 状態: In Progress
- GitHub Issue: #441
- 親Issue: `ISSUE-000037` / GitHub Issue #387
- 関連Issue: `ISSUE-000043` / #433、`ISSUE-000044` / #436、`ISSUE-000042` / #430、`ISSUE-000031` / #308
- Gate B監査: `ISSUE-000029` / #273

## 目的

Gate B final runner自身を`go run`でcompileする段階から、credentialを含み得るPostgreSQL DSN本文をGo compiler / linker等へ継承しないようにします。

PR #434 / #437以降、runner起動後のGo probe、Git source probe、#310、#255からDSN本文は除去されています。しかし、final runnerのlaunch environmentへ`TEI_TEST_POSTGRES_DSN`本文を置く方式では、runner binaryが起動する前の`go run` processとそのchildにもDSNが渡ります。

このIssueは、そのcompile-before-runtime gapだけを閉じます。

## 現行境界 — PR #442

final runnerではdirect `TEI_TEST_POSTGRES_DSN` launchを正規経路として受理しません。

正規入力はRepository外のsecret file pathです。

```text
TEI_TEST_POSTGRES_DSN_FILE=/secure/path/to/dsn
```

runner起動後にfileを読み、DSN本文はGo valueとしてmemoryへ保持します。親final runner processのenvironmentへ`TEI_TEST_POSTGRES_DSN`を設定し直しません。

standaloneの#308 toolは従来どおり`TEI_TEST_POSTGRES_DSN`を受け取ります。本Issueは#308単体の入力契約を変更しません。

## fail-closed入力条件

### direct DSN launchを拒否する

launch environmentに`TEI_TEST_POSTGRES_DSN` keyが存在する場合、値が空でも正規final-run経路から外れているため停止します。

```text
TEI_TEST_POSTGRES_DSN=<anything>
  ↓
reject
```

DSN本文はerrorへ複製しません。

### secret fileを一行に限定する

secret fileは次を許容します。

- 改行なしの一行DSN
- 末尾に単一LFがある一行DSN
- 末尾に単一CRLFがある一行DSN

末尾の一つのLF / CRLFだけを除去した後、次は拒否します。

- empty
- 複数行
- 残存CR / LF
- NUL

file read失敗時もunderlying errorをwrapせず、local secret file pathを通常stderrへ複製しません。

## subprocessへのsecret exposure

PR #442以降、production final runnerはDSN本文とsecret file pathの双方をbase environmentから除去した後にchild environmentを組み立てます。

```text
final runner process
  │ secret file pathのみlaunch時に保持
  │ DSN本文はmemoryのみ
  │
  ├─ go env / go version probe -> DSNなし / file pathなし
  ├─ git source probe          -> DSNなし / file pathなし
  ├─ #310                      -> DSNなし / file pathなし
  ├─ #255                      -> DSNなし / file pathなし
  └─ #308                      -> DSN本文だけ / file pathなし
```

#308 childへ渡すときだけ、明示environmentへ次を追加します。

```text
TEI_TEST_POSTGRES_DSN=<in-memory secret>
```

secret file pathを#308へ渡してchild側で再読込する方式にはしません。secret取得責務はfinal runner側へ一度だけ置きます。

## #433 / #436との役割分離

`ISSUE-000043` / #433:

- runner起動後のGo utility subprocessからDSN本文を除去する

`ISSUE-000044` / #436:

- runner起動後のGit source subprocessからDSN本文を除去する

本Issue:

- **runner起動前のcompile段階へDSN本文を持ち込まない**
- runner起動後もsecret file pathを不要subprocessへ渡さない

三つは同じleast-exposure方針の別境界です。

## test

PR #442では、standard-library中心の隔離packageで次を確認しました。

- direct DSN launchを拒否する
- direct DSN reject前にsecret fileを読まない
- DSN本文をerrorへ複製しない
- file read失敗でsecret file pathをerrorへ複製しない
- 一行DSN / single final LF / single final CRLFを受理する
- empty / multiline / NUL / bare CRを拒否する
- DSN本文とfile pathをcase-insensitiveにenvironmentから除去する
- callerのenvironment sliceを破壊しない
- #310 / #255 childへDSN本文もfile pathも渡さない
- #308 childへDSN本文だけを渡す
- 実test binary child processでも同じ継承境界になる

```text
ok  tei/tools/reference-gate-b-final-evidence  0.006s
```

これは通常TEI Repository checkout上のpackage testでもactual #387 Evidenceでもありません。

## Evidenceへ保存しないもの

- PostgreSQL DSN本文
- secret file path
- credential
- secret file内容

本IssueではEvidence JSON shapeを増やしません。secretをどの入力経路で取得したかを新しいProduct Evidenceへ格上げせず、安全な実行前提として扱います。

## 正規実行の方向

具体的なコマンドの唯一の正本は`plan/GATE_B_EVIDENCE_RUNBOOK.md`です。

最終的な正規起動は、#430のGo環境条件と本IssueのDSN-file条件を合わせ、概念上次の形になります。

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
TEI_TEST_POSTGRES_DSN_FILE='/secure/path/to/dsn' \
go run ./tools/reference-gate-b-final-evidence
```

Runbookへ安全に同期するまでは、本Issueの文書同期を完了扱いにしません。

## 完了条件

- [x] secret file入力を実装した（PR #442）
- [x] direct DSN launchをfail closedで拒否した（PR #442）
- [x] DSN本文を親process environmentへ設定し直さず#308 childだけへ注入した（PR #442）
- [x] utility / Git / #310 / #255からDSN本文とsecret file pathを除去した（PR #442）
- [x] pure / actual child subprocess testを追加した（PR #442）
- [x] Repository正本を追加した（本Issue）
- [ ] final runner tool READMEへ同期する
- [ ] `plan/GATE_B_EVIDENCE_RUNBOOK.md`の正規起動例へ安全に同期する
- [ ] 実TEI checkout上で`go test ./tools/reference-gate-b-final-evidence -count=1`を再確認する
- [ ] actual #387 final runで本入力経路を使う

最後のactual run自体は#387の完了条件です。本Issue単独の成功をGate B Readyへ格上げしません。

## 非目標

- #308 standalone toolの`TEI_TEST_POSTGRES_DSN`契約を変更すること
- DSN secret fileをRepositoryへ保存すること
- secret manager製品を導入すること
- OS process全体をsandbox化すること
- Gate B lane構成やEvidence JSON shapeを変更すること
- GitHub Actionsを再有効化すること
- actual #308 Provider Evidenceを代替すること

関連PR: #442
