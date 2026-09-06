# ISSUE-000045: Gate B final runnerのcompile段階からPostgreSQL DSN本文を隔離する

- 状態: Ready to verify — secret-file入力 / stable-read / pre-secret clean-source hardening実装済み、actual checkout / final run待ち
- GitHub Issue: #441
- 親: `ISSUE-000037` / #387
- 関連: `ISSUE-000043` / #433、`ISSUE-000044` / #436、`ISSUE-000042` / #430
- Gate B監査: `ISSUE-000029` / #273

## 目的

final runner自身を`go run`でcompileする段階から、credentialを含み得るPostgreSQL DSN本文をGo compiler / linker等へ継承しません。

launch environmentにはRepository外secret file pathだけを渡し、runner起動後も**clean Repository sourceを確認できるまでDSN fileをopen / readしません**。

## 現行入力契約

final runnerのlaunch environmentではdirect `TEI_TEST_POSTGRES_DSN`を受理しません。

```text
TEI_TEST_POSTGRES_DSN = absent
TEI_TEST_POSTGRES_DSN_FILE = Repository外existing regular file
```

具体的な正規commandは`plan/GATE_B_EVIDENCE_RUNBOOK.md`を唯一の正本とします。

standalone #308 toolの`TEI_TEST_POSTGRES_DSN`入力契約は変更しません。

## secret read前のsource preflight

final runnerはDSN本文をmemoryへ読み込む前に、PostgreSQL secretを除いたenvironmentでRepository sourceを観測します。

```text
HEAD before
  ↓
git status --porcelain
  ↓
HEAD after
  ↓
HEAD stable && working tree clean
  ↓ only then
open / read DSN secret file
```

次の場合はsecret loaderへ進みません。

- Repository sourceを観測できない
- source観測中にHEADが変化する
- working treeがdirty

これは後段の`runWith`開始時source確認を置き換えません。secret読込後も、lane開始前に改めてcandidate sourceを確認するため、preflight後のsource driftは従来どおりfail closedです。

## secret file guard

secret fileは少なくとも次を要求します。

- pathが設定されている
- existing regular file
- symlink解決後もRepository外
- emptyではない
- 一行DSN
- optionalな末尾単一LF / CRLFのみ許容
- 残存CR/LF拒否
- NUL拒否

Repository外pathからRepository内fileへ向くsymlinkを受理しません。

file pathやunderlying read errorのlocal pathを通常stderr / Evidenceへ複製しません。

## stable single-open read

従来は、Repository外regular fileであることをpathから確認したあと、別の`os.ReadFile`で再openしていました。

この間にpath / symlink targetが差し替わると、確認したfileと実際に読んだfileが異なるTOCTOU余地がありました。

現行runnerではsecret fileを**一度だけopen**し、そのopen済みfile descriptorからDSNを読みます。

読込前と読込後に、少なくとも次を確認します。

```text
TEI_TEST_POSTGRES_DSN_FILE path
        ↓ resolve symlink
Repository外
        ↓
regular file
        ↓
os.SameFile(opened file, currently resolved file)
        ↓
read exact bytes from already-open descriptor
        ↓
resolve / same-file check again
```

次の場合はfail closedします。

- openしたfileがregular fileではない
- pathがRepository内へ解決される
- pathが別file identityへ差し替わる
- symlink targetがRepository内へ変わる
- 読込後に現在pathがopen済みfileと一致しない

Repository外fileへ向くRepository外symlinkは、読込前後とも同じexternal file identityを指す場合に限り許容します。

これはOS filesystem全体をtransaction化するものではありません。重要なのは、DSN bytesを**検証済みの一つのopen file identityから読む**ことです。

## subprocess exposure

```text
outer go run / compiler / linker
  -> DSN本文なし
  -> secret file pathのみ

final runner runtime
  -> secretを除いたenvironmentでclean source preflight
  -> external secret fileをsingle-open
  -> identityを読込前後に確認
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

通常Evidenceへ保存しません。

- PostgreSQL DSN本文
- secret file path
- credential
- secret file内容
- 実Database名

secret取得方式自体を新しいProduct Evidence natureへしません。Gate B actual runの安全な入力前提です。

## 成立済み

- [x] Repository外secret file入力を実装（PR #442）
- [x] direct DSN launchをfail closedで拒否
- [x] DSN本文をparent environmentへ再設定せず#308だけへ注入
- [x] Go utility / Git / #310 / #255からDSN本文とsecret file pathを除去
- [x] tool README / Runbook / Readiness Auditへsecret-file契約を同期
- [x] PR #458でRepository外 / symlink解決後Repository外 / regular-file guardを追加
- [x] PR #465でsecret fileをsingle-openし、読込前後のfile identityを再確認
- [x] external regular file / external symlink受理、Repository内symlink拒否、path replacement拒否のpure testを追加
- [x] dirty / source観測失敗時にsecret loaderを呼ばないclean-source preflightを実装
- [x] clean / dirty / source観測失敗のpreflight pure testを追加

## 残作業

- [ ] 実TEI checkoutでfinal runner package test

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
go test ./tools/reference-gate-b-final-evidence -count=1
```

- [ ] actual #387 final runでRepository外secret file経路を利用
- [ ] actual run中にpre-secret clean-source / stable-read guardが成立したことを確認
- [ ] same-run child Evidenceにsecret本文 / file pathが混入していないことを#273で監査

actual final runは#387の完了条件です。本Issue単独の成功をGate B Readyへ格上げしません。

## 非目標

- #308 standalone input contractの変更
- DSN secret fileのRepository保存
- secret manager製品の導入
- secret file permission policyのOS横断標準化
- OS filesystem全体のtransactional snapshot化
- OS process全体のsandbox化
- Gate B lane構成の変更
- actual #308 Provider Evidenceの代替
- GitHub Actions再有効化
