# ISSUE-000065: Gate B final runner自身のcompile前からGOENVを無効化する

- 状態: Done — Gate B Ready（R = `e5231839`）のactual #387 final runで確認済み（2026-09-20）
- GitHub Issue: #430
- 親: `ISSUE-000037` / #387
- 関連: `ISSUE-000062` / #421、`ISSUE-000063` / #424

## 目的

final runner自身を`go run`でcompileする段階から、Repository外のuser-level Go environment fileを暗黙入力へしません。

runtime開始後に`GOENV=off`へ設定するだけでは、runner自身のcompile時入力を閉じられないためです。

## 現行launch guard

具体的な正規commandは`plan/GATE_B_EVIDENCE_RUNBOOK.md`を唯一の正本とします。

概念上、final runner自身のcompile前から次を要求します。

```text
GOENV=off
GOWORK=off
GOFLAGS=
TEI_TEST_POSTGRES_DSN_FILE=Repository外secret file
TEI_TEST_POSTGRES_DSN=absent
```

旧direct `TEI_TEST_POSTGRES_DSN`起動は現行final runではありません。

## exact `GOENV=off`

process environmentの`GOENV`が完全一致で`off`の場合だけ継続します。

受理しない例:

- unset
- empty
- filesystem path
- `OFF`
- 前後空白付き
- その他の値

runtime側だけで大文字小文字や空白を正規化して受理しません。compile時のGo tool解釈とruntime guardを一致させます。

## 実行順

```text
shell environmentで exact GOENV=off
        ↓
final runner compile
        ↓
runtime launch guard
        ↓
GOWORK / GOFLAGS ambient確認
        ↓
processを GOWORK=off / GOFLAGS= / GOENV=off へ固定
        ↓
#310 → #255 → #308
```

unsafeなlaunchを安全値へ書き換えてから検査しません。

## Evidence

成功JSONでは次を含みます。

```text
goEnvironment.goEnvDisabled = true
```

これはruntimeがexact `GOENV=off`を確認したことを示します。

GOENV pathや拒否値を通常Evidenceへ保存しません。

## 成立済み

- [x] final runnerへGOENV launch guard追加（PR #431）
- [x] exact lowercase `off`のみ受理
- [x] `goEnvironment.goEnvDisabled`追加
- [x] pure / isolated test
- [x] #421 test前提をlaunch guardへ同期
- [x] tool READMEへ同期
- [x] `plan/GATE_B_EVIDENCE_RUNBOOK.md`へ同期
- [x] direct DSNではなくsecret-file起動へ現行文書を同期

## 残作業

- [x] 実TEI checkoutで次を再確認する

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
go test ./tools/reference-gate-b-final-evidence -count=1
```

- [x] actual #387 final runで`goEnvironment.goEnvDisabled = true`を確認する

本Issue単独の成功をGate B Readyへ格上げしません。

## 非目標

- Go toolchain versionの完全固定
- module cache / toolchain vendoring
- OS process全体のsandbox化
- 個別Required Evidenceの代替
- actual Oracle Evidenceの代替
- GitHub Actions再有効化
