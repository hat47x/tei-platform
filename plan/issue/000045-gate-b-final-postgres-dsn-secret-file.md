# ISSUE-000045: Gate B final runnerのcompile段階からPostgreSQL DSN本文を隔離する

- 状態: Ready to verify — secret-file入力 / stable-read / pre-secret source guard / #308 path-only handoff実装済み、actual checkout / final run待ち
- GitHub Issue: #441
- 親: `ISSUE-000037` / #387
- 関連: `ISSUE-000043` / #433、`ISSUE-000044` / #436、`ISSUE-000042` / #430、GitHub #444
- Gate B監査: `ISSUE-000029` / #273

## 目的

final runner自身を`go run`でcompileする段階から、credentialを含み得るPostgreSQL DSN本文をGo compiler / linker等へ継承しません。

launch environmentにはRepository外secret file pathだけを渡します。runner起動後もclean Repository sourceを確認できるまでoriginal secret fileをopen / readせず、読込後もDSN本文をGo / Git probe、#310、#255、#308のcompile段階へ広げません。

## 現行入力契約

final runnerのlaunch environmentではdirect `TEI_TEST_POSTGRES_DSN`を受理しません。

```text
TEI_TEST_POSTGRES_DSN = absent
TEI_TEST_POSTGRES_DSN_FILE = Repository外existing regular file
```

具体的な正規commandは`plan/GATE_B_EVIDENCE_RUNBOOK.md`を唯一の正本とします。

standalone #308 toolはdirect DSNまたはsecret file入力を利用できますが、#387 final convergenceではfinal runnerがoriginal secret fileを読み、Repository外owner-only temporary handoff fileを新規作成して#308へpath-onlyで渡します。

## secret read前のsource preflight

final runnerはoriginal DSN secret fileをopen / readする前に、PostgreSQL secretを除いたenvironmentでRepository sourceを観測します。

```text
HEAD before
  ↓
git status --porcelain --untracked-files=all --ignored=matching
  ↓
git ls-files -v -z
  ↓
HEAD after
  ↓
HEAD stable && tracked/untracked/ignored/index state clean
  ↓ only then
open / read original DSN secret file
```

次の場合はsecret loaderへ進みません。

- Repository sourceを観測できない
- source観測中にHEADが変化する
- tracked / untracked / ignored local stateがある
- tracked index entryが通常`H`ではない
- Git source identity / command-scope configをambient environmentで差し替え得る

これは後段のlane開始時source確認を置き換えません。secret読込後にsourceが変化した場合もfail closedです。

## original secret file guard

original secret fileは少なくとも次を要求します。

- pathが設定されている
- existing regular file
- symlink解決後もRepository外
- emptyではない
- 一行DSN
- optionalな末尾単一LF / CRLFのみ許容
- 残存CR/LF拒否
- NUL拒否
- 大小文字違いを含むduplicate secret-file keyなし

Repository外pathからRepository内fileへ向くsymlinkを受理しません。

file pathやunderlying read errorのlocal pathを通常stderr / Evidenceへ複製しません。

## stable single-open read

original secret fileは検証済みresolved pathを一度だけopenし、そのopen済みfile descriptorからexact DSN bytesを読みます。

読込前後に少なくとも次を確認します。

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

これはfilesystem全体をtransaction化するものではありません。DSN bytesを検証済みの一つのopen file identityから読むための境界です。

## normalized path-only handoff

PR #473以降、final runnerが読んだnormalized DSN本文を#308 child processのenvironmentへ直接設定しません。

final runner runtimeがRepository外にowner-only temporary scopeを作ります。

```text
Repository外 temporary directory: 0700
  └─ normalized DSN handoff file: 0600
```

#308の`go run` / compiler / linkerにはtemporary handoff file pathだけを渡します。DSN本文、original secret file pathは渡しません。

#308 runtimeはhandoff fileからDSN本文を読んだ後、Repository source probeより前にDSN / file path environmentを消去し、それ以後はDSN本文をmemoryだけで保持します。

normal final-runner return時にtemporary handoff directoryをcleanupします。process強制kill / host停止等に対するOS-level完全消去までは保証せず、handoff存在中はRepository外owner-only temporary scopeへ限定します。

## subprocess exposure

```text
outer go run / compiler / linker
  -> DSN本文なし
  -> original secret file pathのみ

final runner runtime
  -> secret-free environmentでclean-source preflight
  -> original external secret fileをstable single-open
  -> 読込前後で同一file identityを再確認
  -> normalized DSN本文をmemoryへ保持
  -> Repository外0700 temporary directory / 0600 handoff fileを作成

Go environment / version probe
Git source probe
#310
#255
  -> DSN本文なし
  -> original secret file pathなし
  -> handoff file pathなし

#308 go run / compiler / linker
  -> DSN本文なし
  -> original secret file pathなし
  -> handoff file pathのみ

#308 runtime
  -> handoff fileからDSN本文を読込
  -> DSN / file path environmentをsource probe前に消去
  -> 以後DSN本文はmemoryのみ

final runner return
  -> temporary handoff directoryをcleanup
```

parent final runner processへ`TEI_TEST_POSTGRES_DSN`を設定し直しません。

## Evidence

通常Evidenceへ保存しません。

- PostgreSQL DSN本文
- original secret file path
- temporary handoff file path
- credential
- secret file内容
- 実Database名

secret取得・handoff方式自体を新しいProduct Evidence natureへしません。Gate B actual runの安全な入力前提です。

## 成立済み

- [x] Repository外secret file入力を実装（PR #442）
- [x] direct DSN final launchをfail closedで拒否
- [x] Go utility / Git / #310 / #255からDSN本文とsecret file pathを除去
- [x] tool README / Runbook / Readiness Auditへsecret-file契約を同期
- [x] PR #458でRepository外 / symlink解決後Repository外 / regular-file guardを追加
- [x] PR #465でsecret fileをsingle-openし、読込前後のfile identityを再確認
- [x] dirty / source観測失敗時にsecret loaderを呼ばないclean-source preflightを実装
- [x] PR #477 / #479 / #483でGit config injection、ignored local state、non-standard index stateをpreflightへ反映
- [x] PR #473でnormalized DSNをRepository外owner-only temporary handoffへ移し、#308 compile段階をpath-only transportへ変更
- [x] #308 runtimeでhandoff読込後にDSN / file path environmentをsource probe前に消去
- [x] #310 / #255 / Go / Git probeへhandoff pathを伝播しない

## 残作業

- [ ] 実TEI full checkoutでfinal runner package test

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
go test ./tools/reference-gate-b-final-evidence -count=1
```

- [ ] actual #387 final runでRepository外secret-file + temporary path-only handoff経路を利用
- [ ] actual run中にpre-secret source guard / stable-read / handoff cleanupが成立したことを確認
- [ ] #308 compile段階にDSN本文が存在しないことをactual executionで確認
- [ ] same-run child EvidenceにDSN本文 / original secret path / handoff pathが混入していないことを#273で監査

actual final runは#387の完了条件です。本Issue単独の成功をGate B Readyへ格上げしません。

## 非目標

- DSN secret fileのRepository保存
- secret manager製品の導入
- secret file permission policyのOS横断標準化
- OS filesystem全体のtransactional snapshot化
- OS process全体のsandbox化
- Gate B lane構成の変更
- actual #308 Provider Evidenceの代替
- GitHub Actions greenをGate B Product Evidenceへ格上げすること