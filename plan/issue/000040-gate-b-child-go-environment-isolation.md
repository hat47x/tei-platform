# ISSUE-000040: Gate B child Go環境をGOENVまで固定する

- 状態: In Progress — 実装 / 隔離test / tool README / Runbook同期済み、実TEI checkout test待ち
- GitHub Issue: #424
- 親: `ISSUE-000037` / #387
- 関連: `ISSUE-000039` / #421、`ISSUE-000042` / #430

## 目的

final runnerから起動する#310 / #255 / #308 childと、そのchildが通常継承で起動するnested Go commandについて、Repository外のuser-level Go environment fileから実行条件が後付けで変化しないようにします。

## 現行境界

まず`ISSUE-000039` / #421のambient検査で外部GOWORK /非空GOFLAGSを拒否し、`ISSUE-000042` / #430でfinal runner自身がcompile前からexact `GOENV=off`で起動されたことを確認します。

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

現行final runnerではPostgreSQL secretについて別途least-exposure境界を適用します。

- Go / Git probe: DSN本文なし / secret file pathなし
- #310: DSN本文なし / secret file pathなし
- #255: DSN本文なし / secret file pathなし
- #308: in-memory DSN本文のみ / secret file pathなし

Go環境固定を理由にsecret exposureを広げません。

## 成立済み

- [x] ambient検査後だけprocess environmentを固定（PR #425）
- [x] `GOWORK=off / GOFLAGS= / GOENV=off`を固定
- [x] unsafe ambient状態では固定処理へ進まないpure test
- [x] environment固定失敗のfail-closed test
- [x] external GOENV fileをchild Go commandが無視する継承test
- [x] 隔離module test
- [x] final runner tool READMEへ同期
- [x] `plan/GATE_B_EVIDENCE_RUNBOOK.md`へ同期

## 残作業

- [ ] 実TEI checkoutで次を再確認する

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
go test ./tools/reference-gate-b-final-evidence -count=1
```

- [ ] actual #387 final runで三laneが同じcandidate Rへ収束することを確認する

actual runは#387の完了条件です。本IssueだけでGate B Readyを宣言しません。

## 非目標

- OS process全体のsandbox化
- Go toolchain / module cacheのvendoring
- child programが意図的にenvironmentを書き換える全ケースの強制防止
- #310 / #255 / #308 actual Evidenceの代替
- GitHub Actions再有効化
