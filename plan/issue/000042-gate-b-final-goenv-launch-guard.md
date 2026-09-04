# ISSUE-000042: Gate B final runner自身のcompile前からGOENVを無効化する

- 状態: Ready to verify
- GitHub Issue: #430
- 親Issue: `ISSUE-000037` / GitHub Issue #387
- 関連Issue: `ISSUE-000039` / #421、`ISSUE-000040` / #424、`ISSUE-000029` / #273

## 目的

Gate B三lane最終収束runner自身を`go run`でcompileする段階から、Repository外のuser-level Go environment fileを暗黙入力へしないようにします。

#421ではexternal Go workspaceとeffective GOFLAGSを、#424ではrunner起動後のchild / nested Go environmentを閉じました。本Issueは、そのさらに一段手前である**final runner自身のcompile時入力**を閉じます。

## 背景

PR #423以降の正規起動例では、runner自身のcompile前から次を明示していました。

```text
GOWORK=off
GOFLAGS=
```

PR #425では、runner起動後にambient Go環境を確認してからprocess environmentを次へ固定しました。

```text
GOWORK=off
GOFLAGS=
GOENV=off
```

ただし、`GOENV=off`はrunner起動後に設定されるため、その時点では`go run ./tools/reference-gate-b-final-evidence`のcompileは既に終わっています。user-level Go environment fileに`GOTOOLCHAIN`等が設定されていれば、Repository HEAD / working treeでは観測できないcompile入力になり得ます。

## 正規起動境界

final Evidenceの正規起動は、runner自身のcompile前から次を明示します。

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
TEI_TEST_POSTGRES_DSN='...' \
go run ./tools/reference-gate-b-final-evidence \
  > /tmp/tei-gate-b-final-evidence.json
```

`-broad`を使う場合も同じです。

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
TEI_TEST_POSTGRES_DSN='...' \
go run ./tools/reference-gate-b-final-evidence -broad \
  > /tmp/tei-gate-b-final-evidence.json
```

## exact `GOENV=off`

PR #431の実装では、process environmentの`GOENV`が**完全一致で`off`**である場合だけ継続します。

```text
GOENV == "off"
```

次は受理しません。

- 未設定
- 空文字
- filesystem path
- `OFF`
- ` off`
- `off `
- その他の値

これは表記を厳しくするためではなく、Go tool側の意味とruntime guardを一致させるためです。

Go 1.23.2で確認したところ、`GOENV=off`はuser-level Go environment fileを無効化します。一方、別pathを指定すれば、そのfileに保存された`GOFLAGS`等が有効になります。大文字・空白をruntime側だけで正規化して受理すると、compile時には外部GOENVが有効だったrunを後から安全と誤判定し得るため、緩和しません。

## 実装順序

```text
shell / process environmentで exact GOENV=off
        ↓
final runner自身をcompile
        ↓
runner開始
        ↓
GOENV launch guard
        ↓
GOWORK / GOFLAGS ambient確認
        ↓
runner processを GOWORK=off / GOFLAGS= / GOENV=off へ固定
        ↓
#310 → #255 → #308
```

unsafeな起動を安全値へ書き換えてから検査しません。

## Evidence

`goEnvironment`へ次を追加します。

```json
{
  "externalWorkspaceActive": false,
  "goFlagsEmpty": true,
  "goEnvDisabled": true
}
```

`goEnvDisabled = true`は、runner runtimeがprocess environmentでexact `GOENV=off`を確認したことを示します。

次は保存しません。

- GOENV path
- GOENVの拒否値
- external Go workspace path
- GOFLAGS本文

## #421 / #424との役割分離

```text
#421 / ISSUE-000039
  external GOWORK / effective GOFLAGSをreject

#430 / ISSUE-000042
  final runner compile前からexact GOENV=offを要求

#424 / ISSUE-000040
  ambient確認後のrunner processを
  GOWORK=off / GOFLAGS= / GOENV=offへ固定し、child / nested Goへ継承
```

三つは代替関係ではありません。

## test

PR #431では次を固定しました。

- exact `GOENV=off`を受理する
- unset / empty / path / uppercase / 前後空白 / その他値を拒否する
- reject時にprocess environmentを書き換えない
- 拒否したGOENV値をerrorへ保存しない
- 既存#421 testsは`GOENV=off`を明示したうえで本来のGOWORK / GOFLAGS条件を検証する
- 成功時に`goEnvDisabled = true`
- #424のchild environment固定を維持する

隔離packageで#421 / #424 / #430のGo環境testsをまとめて実行しました。

```text
ok  tei-goenv-430  0.007s
```

これは通常TEI Repository checkout上のpackage test、#310 Evidence、#387 final Evidenceではありません。

## 文書の正本関係

具体的なGate Bの実行コマンド・順序の唯一の正本は引き続き`plan/GATE_B_EVIDENCE_RUNBOOK.md`です。

ただしPR #431直後の時点では、巨大なRunbookを安全な部分patchなしに全再生成して履歴を壊すことを避けるため、本Issueとtool READMEを先に同期します。Runbookの正規final-runner起動例へ`GOENV=off`が反映されるまで、文書同期は完了扱いにしません。

Runbookが古い起動例を残していても、それを根拠に`GOENV`未指定のrunを成功Evidenceへ採用しません。runtime guardがfail closedで拒否します。

## 完了条件

- [x] final runnerへGOENV launch guardを追加する（PR #431）
- [x] exact lowercase `off`だけを受理する
- [x] `goEnvironment.goEnvDisabled`を追加する
- [x] pure / isolated testを追加する
- [x] 既存#421 test前提を新launch guardへ同期する
- [x] Repository正本を追加する（本書）
- [ ] tool READMEの正規起動例を`GOENV=off`付きへ同期する
- [ ] `plan/GATE_B_EVIDENCE_RUNBOOK.md`の正規起動例を安全に同期する
- [ ] 実TEI checkout上で`go test ./tools/reference-gate-b-final-evidence -count=1`を再確認する
- [ ] actual #387 final runで`goEnvDisabled=true`を確認する

最後のactual run自体は#387の完了条件です。本Issue単独の成功をGate B Readyへ格上げしません。

## 非目標

- Go toolchain versionの完全固定
- module cache / toolchainのvendoring
- OS process全体のsandbox化
- GitHub Actionsの再有効化
- #310 / #255 / #308個別Evidenceの代替
- actual Oracle Evidenceの代替
