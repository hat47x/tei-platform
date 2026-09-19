# ISSUE-000063: Gate B child Go環境をGOENVまで固定する

- 状態: Done — Gate B Ready（R = `e5231839`）のactual #387 final runで確認済み（2026-09-20）
- GitHub Issue: #424
- 親: `ISSUE-000037` / #387
- 関連: `ISSUE-000062` / #421、`ISSUE-000065` / #430、`ISSUE-000045` / #441、GitHub #444

## 目的

final runnerから起動する#310 / #255 / #308 childと、そのchildが通常継承で起動するnested Go commandについて、Repository外のuser-level Go environment fileから実行条件が後付けで変化しないようにします。

## 現行境界

まず`ISSUE-000062` / #421のambient検査で外部GOWORK /非空GOFLAGSを拒否し、`ISSUE-000065` / #430でfinal runner自身がcompile前からexact `GOENV=off`で起動されたことを確認します。

その確認に成功した**後だけ**、final runner processを次へ固定します。

```text
GOWORK=off
GOFLAGS=
GOENV=off
```

これを#310 / #255 / #308 childと通常のnested Go commandへ継承します。

```text
ambient / launch guard
        ↓ unsafeなら停止
safeならprocess environment固定
        ↓
#310
        ↓
#255
        ↓
#308
```

検査前に安全値へ書き換えて、もともと不安全だった状態を隠しません。

## fail closed

次のいずれかを設定できなければlaneを開始しません。

- `GOWORK=off`
- `GOFLAGS=`
- `GOENV=off`

途中まで設定した後に失敗した場合も成功Evidenceへ進みません。

## PostgreSQL secretとの分離

このIssueはGo環境だけを扱います。

現行final runnerではPostgreSQL secretについて別途least-exposure / path-only handoff境界を適用します。

```text
Go environment / version probe
Git source probe
#310
#255
  -> DSN本文なし
  -> original secret file pathなし
  -> temporary handoff pathなし

#308 go run / compiler / linker
  -> DSN本文なし
  -> original secret file pathなし
  -> temporary handoff pathのみ

#308 runtime
  -> handoff fileからDSN本文を読込
  -> DSN / file path environmentをsource probe前に消去
  -> 以後DSN本文はmemoryのみ
```

Go環境固定を理由にsecret exposureを広げません。`GOWORK / GOFLAGS / GOENV`の固定とPostgreSQL secret transportは別の安全境界です。

## 成立済み

- [x] ambient検査後だけprocess environmentを固定（PR #425）
- [x] `GOWORK=off / GOFLAGS= / GOENV=off`を固定
- [x] unsafe ambient状態では固定処理へ進まないpure test
- [x] environment固定失敗のfail-closed test
- [x] external GOENV fileをchild Go commandが無視する継承test
- [x] 隔離module test
- [x] final runner tool READMEへ同期
- [x] `plan/GATE_B_EVIDENCE_RUNBOOK.md`へ同期
- [x] #308 compile環境のpath-only secret handoffと両立

## 残作業

- [x] 実TEI full checkoutで次を再確認する

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
go test ./tools/reference-gate-b-final-evidence -count=1
```

- [x] actual #387 final runで三laneが同じcandidate Rへ収束することを確認する
- [x] actual #308 childが同じGo環境固定とpath-only secret handoffを同時に満たすことを確認する

actual runは#387の完了条件です。本IssueだけでGate B Readyを宣言しません。

## 非目標

- OS process全体のsandbox化
- Go toolchain / module cacheのvendoring
- child programが意図的にenvironmentを書き換える全ケースの強制防止
- #310 / #255 / #308 actual Evidenceの代替
- CI greenをGate B Product Evidenceへ格上げすること
