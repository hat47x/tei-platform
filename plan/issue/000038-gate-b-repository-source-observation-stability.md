# ISSUE-000038: Gate BのRepository source観測中HEAD driftをfail closedにする

- 状態: In Progress — 実装 / pure test済み、実TEI checkoutでの通常Go test待ち
- GitHub Issue: #418
- Gate B最終収束: `ISSUE-000037` / #387
- Gate B監査: `ISSUE-000029` / #273
- 関連Evidence: #255 / #308 / #310

## 目的

Gate BのEvidence取得で共有するRepository source観測について、`git status`を取得している途中にHEADが切り替わった場合の混成観測をfail closedで拒否します。

この境界は、次のEvidence runnerが共通利用する`tools/internal/repositoryevidence`へ置きます。

- #255 generated artifact reproducibility Evidence
- #308 real PostgreSQL Provider + ephemeral fixture Evidence
- #310 deterministic Repository Evidence
- #387 三lane最終収束runner自身のsource再確認

Canonical modelやProvider固有の意味へRepository観測概念を追加しません。

## 背景

PR #419以前の共有helperは、開始時・終了時とも概ね次の順でsource状態を観測していました。

```text
git rev-parse HEAD
  ↓
git status --porcelain
```

この二つのcommandの間に同じworking treeでcheckout等が行われると、理論上は「切替前のrevision」と「切替後のworking tree状態」を一つの観測として組み合わせる余地がありました。

actual Gate B runでは専用の固定checkoutを使う運用が前提ですが、その運用だけに依存せず、共有helperでも明確に観測できるHEAD driftを拒否します。

## 採用する観測境界

PR #419以降、開始時・終了時とも次の順で観測します。

```text
HEAD before
  ↓
git status --porcelain
  ↓
HEAD after
```

`HEAD before`と`HEAD after`が一致しない場合、その三点を一つのRepository stateとは扱いません。

### 開始時

`InspectStart`は、working tree観測前後のHEADが同じ場合だけ`Source`を返します。

観測中にHEADが変わった場合はerrorとし、切替前revisionと切替後statusを組み合わせた`Source`を返しません。

HEADが安定している場合は、従来どおりclean / dirtyを観測できます。dirtyであること自体を観測不能とはしません。各runnerが自身の用途に応じて、通常Evidenceでは拒否し、診断モードがある場合はその契約に従います。

### 終了時・中間確認

`InspectPost`も同じ三点観測を行います。

`postRunRepository.checked = true`にするのは、次をすべて完了した場合だけです。

1. 最初のHEADを取得できた
2. working tree状態を取得できた
3. 二回目のHEADを取得できた
4. 二つのHEADが一致した

観測中にHEADが変わった場合はerrorとし、`Checked=false`のままにします。最終的に観測したHEADを`RepositoryRevision`へ残し、`RevisionStable=false`とします。

これにより、#255 / #308 / #310だけでなく、#387 final runnerが各lane後に行うsource再確認でも、観測中driftを「確認済み」と誤認しません。

## 再現可能判定との関係

`IsReproducible`の上位契約は変更しません。

引き続き、少なくとも次をすべて要求します。

- 開始revisionが空でない
- 開始時working treeがclean
- post-run確認済み
- post-run revisionが開始revisionと同一
- `RevisionStable=true`
- 終了時working treeがclean

PR #419は、この判定へ渡す開始時 / post-run観測自体の整合性を一段強くする変更です。

## 検証

PR #419のbranch上にある次の二fileとGit blob SHAが一致するsourceを隔離Go moduleへ配置し、共有package testを実行しました。

```text
repository.go       5a69ec380c45feb41e1fb14732f4e396bebb18ba
repository_test.go  4f0484fef3089ad2961cfec0c4d44ad37c4ff10c
```

結果:

```text
ok  tei/tools/internal/repositoryevidence  0.002s
```

`gofmt -d`にも差分はありませんでした。

pure testでは少なくとも次を固定しています。

- start観測中にHEADが変わればerror
- post-run観測中にHEADが変わればerrorかつ`Checked=false`
- working tree観測後の二回目HEAD取得に失敗した場合も`Checked=true`にしない
- HEADが前後同一ならclean / dirtyを従来どおり観測できる
- status取得失敗を確認済み状態へ昇格させない

これは**実TEI Repository checkout上の通常Go testではありません**。また、#255 / #308 / #310のactual Evidenceや#387 final convergence Evidenceの代替でもありません。

## この境界の限界

この補強はGit working tree全体をtransactional snapshotとして観測するものではありません。

例えば、観測中にfileが一時的に変更され、その後元の内容へ戻ったケースをすべて検出できるわけではありません。また、三command全体をGitの一つのtransactionへするものでもありません。

したがってactual Gate B runでは、引き続き次を運用上の前提とします。

- 専用のcheckoutを使う
- candidate revisionを実行中に切り替えない
- 並行操作でworking treeを変更しない
- Evidence出力をRepository内へ直接redirectしない

PR #419はこの運用を置き換えるのではなく、明確に観測できるHEAD切替TOCTOUをhelper側でもfail closedにする補強です。

## 完了条件

- [x] `InspectStart`をHEAD二重確認へ変更する（PR #419）
- [x] `InspectPost`をHEAD二重確認へ変更する（PR #419）
- [x] 観測中HEAD driftのpure testを追加する（PR #419）
- [x] 隔離Go moduleで共有package testを実行する
- [x] #387とRepository正本へ境界を同期する
- [ ] 実TEI Repository checkoutの通常Go環境で`go test ./tools/internal/repositoryevidence -count=1`を再確認する

最後の通常checkout testが完了するまで、GitHub #418は完了扱いにしません。

## 非目標

- Git working tree全体のtransactional snapshot化
- 瞬間的なfile変更と復元を含むあらゆる並行変更の検出
- Repository source観測概念をCanonical modelへ追加すること
- #255 / #308 / #310 / #387の個別Evidence契約を置き換えること
- 本pure testをactual Gate B Evidenceへ格上げすること
- GitHub Actionsを再有効化すること

## 関連

- GitHub Issue #418
- GitHub Issue #255 / #308 / #310 / #387 / #273
- PR #336 / #355 / #409 / #412 / #419
- `tools/internal/repositoryevidence/`
- `plan/issue/000037-gate-b-three-lane-final-evidence.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_READINESS_AUDIT.md`
