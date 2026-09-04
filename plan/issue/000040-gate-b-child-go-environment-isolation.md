# ISSUE-000040: Gate B child Go環境をGOENVまで固定する

- 状態: In Progress — 実装 / 隔離test済み、実TEI checkout test待ち
- GitHub Issue: #424
- 親: `ISSUE-000037` / #387
- 関連: `ISSUE-000039` / #421、#310、#255、#308、#273

## 目的

Gate B三lane最終収束runnerから起動するchild processと、そのchildが通常の継承で起動するnested Go commandについて、Repository外のGo environment fileから実行条件が後付けで変化しないようにします。

## 背景

`ISSUE-000039` / #421では、final runner開始時にeffective `GOWORK` / `GOFLAGS`を確認し、外部workspaceや非空GOFLAGSをfail closedで拒否する境界を追加しました。

さらに正規起動例では、runner自身をcompileする前から次を明示します。

```bash
GOWORK=off \
GOFLAGS= \
...
```

一方、user-level Go environment fileはRepository外にあります。final runner開始後にそのfileが変更された場合、後続のchild Go commandが新しい設定を読み直す余地を、Git HEAD / working tree確認だけでは捉えられません。

## 実装 — PR #425

ambient Go環境が安全であることを確認した**後だけ**、final runner自身のprocess environmentを次へ固定します。

```text
GOWORK=off
GOFLAGS=
GOENV=off
```

`GOENV=off`によって、childや通常のnested Go commandがuser-level Go environment fileを既定経路として読み直さない状態にします。

重要なのは順序です。

```text
ambient GOWORK / GOFLAGSを観測
  ↓
unsafeなら停止
  ↓
safeならprocess environmentを固定
  ↓
#310 child
  ↓
#255 child
  ↓
#308 child
```

ambient環境を検査する前に安全値へ書き換え、もともと不安全だった状態を隠して成功させる実装にはしません。

## fail-closed境界

process environmentの固定には`os.Setenv`を利用します。

次のいずれかを満たせない場合、laneを一つも開始しません。

- `GOWORK=off`を設定できる
- `GOFLAGS=`を設定できる
- `GOENV=off`を設定できる

途中まで設定した後に失敗した場合も、そのprocessはerrorで終了するため成功Evidenceへは進みません。

## #308 DSNとの関係

この変更はGo実行環境だけを扱います。

`TEI_TEST_POSTGRES_DSN`については既存のleast-exposure境界を維持します。

- #310へは渡さない
- #255へは渡さない
- #308へだけ渡す

Go環境固定を理由にDSNの露出範囲を広げません。

## 隔離test

PR #425 branch上の変更production fileと関連test fileを、Git blob SHAが完全一致する隔離moduleで検証しました。

```text
541d57bfdadbe775ef5cfdaf10b582274d086e3d  go_environment.go
b4c4431a3a477f9cab24d2f99c4b8a5bd2d52619  go_environment_test.go
0502ea4946584a00fe0de5fec74c560b5650e43e  go_environment_enforcement_test.go
00dc97f3bdf654a0e654e90416b3fdd07fa361eb  go_environment_inheritance_test.go
```

結果:

```text
ok  tei-goenv-exact  0.007s
```

`gofmt -d`も差分なしでした。

### 実subprocess継承

外部GOENV fileへ意図的に次を書きました。

```text
GOFLAGS=-tags=should_not_leak
```

固定処理後に実`go env` subprocessを起動し、次を確認しました。

- process environment上の`GOWORK=off`
- process environment上の`GOFLAGS=`
- process environment上の`GOENV=off`
- child `go env GOFLAGS`は空
- child `go env GOWORK`は`off`

Go 1.23.2では`GOENV=off`時の`go env GOENV`表示は空になるため、無効化確認を文字列`off`の表示だけに依存させません。

## 限界

これはOS process全体のsandboxではありません。

child program自身が意図的に自分のenvironmentを書き換えた後でnested commandを起動するケースまで、親processから強制するものではありません。

本境界は、final runnerから通常の環境継承で進む正規Evidence経路について、Repository外のGo environment fileを暗黙入力へしないための補強です。

## 完了条件

- [x] ambient Go環境検査後にprocess environmentを固定する（PR #425）
- [x] `GOWORK=off / GOFLAGS= / GOENV=off`を固定する
- [x] unsafe ambient環境では固定処理へ進まないpure testを追加する
- [x] environment固定失敗をfail closedにするtestを追加する
- [x] 外部GOENV fileをchild Go commandが無視する実継承testを追加する
- [x] Git blob SHA一致の隔離moduleでtestする
- [ ] tool READMEへ境界を同期する
- [ ] 実TEI checkout上で`go test ./tools/reference-gate-b-final-evidence -count=1`を再確認する
- [ ] actual final runで三laneが同じcandidate revisionへ収束することを確認する

## 非目標

- OS sandboxを実装すること
- Go module cacheやtoolchainをRepositoryへvendoringすること
- GitHub Actionsを再有効化すること
- #310 / #255 / #308 actual Evidenceを代替すること
- 本IssueだけでGate B Readyを宣言すること

## 関連

- #424
- PR #425
- `ISSUE-000037` / #387
- `ISSUE-000039` / #421
- #273 / #310 / #255 / #308
