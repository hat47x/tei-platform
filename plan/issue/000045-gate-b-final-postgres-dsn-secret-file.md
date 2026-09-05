# ISSUE-000045: Gate B final runnerのcompile段階からPostgreSQL DSN本文を隔離する

- 状態: Ready to verify — 実装 / isolated test / tool README / Runbook同期済み、actual checkout / final run待ち
- GitHub Issue: #441
- 親: `ISSUE-000037` / #387
- 関連: `ISSUE-000043` / #433、`ISSUE-000044` / #436、`ISSUE-000042` / #430
- Gate B監査: `ISSUE-000029` / #273

## 目的

final runner自身を`go run`でcompileする段階から、credentialを含み得るPostgreSQL DSN本文をGo compiler / linker等へ継承しません。

runner起動後だけDSNを除去しても、outer `go run` / compiler / linkerへsecretが渡る余地を残すためです。

## 現行入力契約

final runnerのlaunch environmentではdirect `TEI_TEST_POSTGRES_DSN`を受理しません。

```text
TEI_TEST_POSTGRES_DSN = absent
TEI_TEST_POSTGRES_DSN_FILE = Repository外existing regular file
```

具体的な正規commandは`plan/GATE_B_EVIDENCE_RUNBOOK.md`を唯一の正本とします。

概念上:

```text
GOENV=off
GOWORK=off
GOFLAGS=
TEI_TEST_POSTGRES_DSN_FILE=/external/path/to/dsn
go run ./tools/reference-gate-b-final-evidence ...
```

です。

standalone #308 toolの`TEI_TEST_POSTGRES_DSN`入力契約は変更しません。

## secret file guard

secret fileはrunner起動後に一度だけ読み、DSN本文をmemoryへ保持します。

現行runnerではPR #458の補強も含め、次を要求します。

- pathが設定されている
- existing regular file
- symlink解決後もRepository外
- emptyではない
- 一行DSN
- optionalな末尾単一LF / CRLFのみ許容
- 残存CR/LF拒否
- NUL拒否

file pathやread errorのunderlying local pathを通常stderr / Evidenceへ複製しません。

Repository内fileや、Repository外pathからRepository内fileへ向くsymlinkをsecret inputとして受理しません。

## subprocess exposure

runner memory上のDSN本文とlaunch時secret file pathは、必要なprocessだけへ限定します。

```text
outer go run / compiler / linker
  -> DSN本文なし
  -> secret file pathのみlaunch environmentに存在

final runner runtime
  -> secret fileを一度読む
  -> DSN本文はmemoryのみ

Go environment / version probe
Git source probe
#310
#255
  -> DSN本文なし
  -> secret file pathなし

#308
  -> in-memory DSN本文のみ
  -> secret file pathなし
```

parent final runner processへ`TEI_TEST_POSTGRES_DSN`を設定し直しません。

## Evidence

次を通常Evidenceへ保存しません。

- PostgreSQL DSN本文
- secret file path
- credential
- secret file内容
- 実Database名

secret取得方式自体を新しいProduct Evidence natureへしません。Gate B actual runの安全な入力前提として扱います。

## 成立済み

- [x] Repository外secret file入力を実装（PR #442）
- [x] direct DSN launchをfail closedで拒否
- [x] DSN本文をparent environmentへ再設定せず#308だけへ注入
- [x] Go utility / Git / #310 / #255からDSN本文とsecret file pathを除去
- [x] pure / actual child subprocess test
- [x] tool READMEへ同期
- [x] `plan/GATE_B_EVIDENCE_RUNBOOK.md`へ同期
- [x] `plan/GATE_B_READINESS_AUDIT.md`へ同期
- [x] PR #458でsecret fileのRepository外 / symlink解決後Repository外 / regular-file guardを追加

## 残作業

- [ ] 実TEI checkoutで次を再確認する

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
go test ./tools/reference-gate-b-final-evidence -count=1
```

- [ ] actual #387 final runでRepository外secret file経路を利用する
- [ ] same-run child Evidenceにsecret本文 / file pathが混入していないことをReadiness Auditで確認する

actual final runは#387の完了条件です。本Issue単独の成功をGate B Readyへ格上げしません。

## 非目標

- #308 standalone input contractの変更
- DSN secret fileのRepository保存
- secret manager製品の導入
- OS process全体のsandbox化
- Gate B lane構成の変更
- actual #308 Provider Evidenceの代替
- GitHub Actions再有効化
