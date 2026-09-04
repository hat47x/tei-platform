# ISSUE-000039: Gate B最終EvidenceのGo実行環境をcandidate revisionの外へ広げない

- 状態: In Progress — 実装 / pure test / tool README同期済み、実TEI checkoutでの通常Go test待ち
- GitHub Issue: #421
- Gate B最終収束: `ISSUE-000037` / #387
- Gate B監査: `ISSUE-000029` / #273
- 関連Evidence: #310 / #255 / #308

## 目的

Gate Bの三lane最終収束で、cleanなcandidate repository revision `R`の外からGo workspaceやGOFLAGSによってcompile / test入力が差し替わる状態をfail closedで拒否します。

この境界は、三laneを再実行して最終Ready判定へ渡す`tools/reference-gate-b-final-evidence`へ置きます。個別laneのEvidence runnerを重複改修せず、最終収束の入口で同じGo実行条件を要求します。

## 背景

TEI Repositoryは`go.mod`を持ちますが、`go.work`は持ちません。

一方、Goはcheckoutの親directoryにある`go.work`を自動検出できます。またeffective `GOFLAGS`には、例えば次のような指定を含められます。

```text
-overlay=...
-modfile=...
-tags=...
```

これらが有効な場合、`git status --porcelain`がcleanで`repositoryRevision = R`であっても、実際にcompile / testした入力がRだけとは限りません。

Repository source固定とGo実行入力固定は別の観測境界として扱います。

## 実装 — PR #422

final runnerはlaneを開始する前に次を確認します。

### GOWORK

```bash
go env GOWORK
```

許可するのは次だけです。

- 空: workspace未使用
- `off`: workspaceを明示無効化

それ以外のpathが有効な場合は、外部Go workspaceがactiveと判断して停止します。

実pathはEvidence JSONやerrorへ複製しません。

### GOFLAGS

```bash
go env GOFLAGS
```

空だけを許可します。非空の場合は内容を解釈して「安全そうなflag」だけを通すのではなく、Gate B最終Evidenceではfail closedにします。

これにより`-overlay`、`-modfile`、`-tags`等の見落としを個別列挙へ依存しません。

GOFLAGS本文もEvidence JSONやerrorへ複製しません。

## runner自身のcompile前から固定する

runtime guardは、起動済みfinal runnerがeffective Go環境を確認する境界です。

さらにtool READMEの正規実行例では、runner自身を`go run`でcompileする段階から環境を固定します。

```bash
GOWORK=off \
GOFLAGS= \
TEI_TEST_POSTGRES_DSN='...' \
go run ./tools/reference-gate-b-final-evidence \
  > /tmp/tei-gate-b-final-evidence.json
```

この明示指定とruntime guardは代替関係ではありません。外側で実行条件を固定し、内側でも条件を再確認します。

## 非機密なEvidence

最終JSONへ次だけを残します。

```json
{
  "goEnvironment": {
    "externalWorkspaceActive": false,
    "goFlagsEmpty": true
  }
}
```

次は保存しません。

- external `go.work`のpath
- GOFLAGS本文
- module cache path
- credential / secret

## 検証

PR #422のbranch上にあるfinal runner main / Go環境helper / helper testとGit blob SHAが一致するsourceを隔離Go moduleへ配置し、package compileとpure testを実行しました。

```text
ok  tei/tools/reference-gate-b-final-evidence  0.002s
```

`gofmt -d`にも差分はありませんでした。

pure testでは少なくとも次を固定しています。

- GOWORK空を許可する
- `GOWORK=off`を許可する
- external go.work pathを拒否する
- external pathをerrorへ露出しない
- GOFLAGS空を許可する
- 非空GOFLAGSを拒否する
- GOFLAGS本文をerrorへ露出しない
- `go env`を観測できない場合もfail closedにする

この確認は**実TEI Repository checkout上の通常Go testではありません**。また、#310 / #255 / #308 actual Evidenceや#387 actual final convergence Evidenceの代替でもありません。

## 完了条件

- [x] #387 final runnerへGOWORK / GOFLAGS guardを追加する（PR #422）
- [x] 非機密なGo環境EvidenceをJSONへ追加する（PR #422）
- [x] pure testを追加する（PR #422）
- [x] 隔離Go moduleでpackage compile / pure testを実行する
- [x] tool READMEとRepository正本へ同期する
- [ ] 実TEI checkout上の通常Go環境で`go test ./tools/reference-gate-b-final-evidence -count=1`を再確認する
- [ ] actual final runで`goEnvironment.externalWorkspaceActive = false` / `goFlagsEmpty = true`を確認する

actual final run自体は#387の完了条件であり、本Issueだけの成功をGate B Readyへ格上げしません。

## 非目標

- Go toolchainやmodule cacheをRepositoryへvendoringすること
- GOMODCACHE等のlocal pathをEvidenceへ保存すること
- 個別laneのrunnerを置き換えること
- Go環境guardだけでGate B Readyを宣言すること
- GitHub Actionsを再有効化すること

## 関連

- GitHub Issue #421 / #387 / #273
- GitHub Issue #310 / #255 / #308
- PR #422
- `tools/reference-gate-b-final-evidence/`
- `plan/issue/000037-gate-b-three-lane-final-evidence.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
