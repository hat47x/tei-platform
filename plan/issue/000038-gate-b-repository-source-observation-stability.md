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

PR #419以降、開始時・終了時ともHEADをworking tree観測の前後で確認します。PR #477以降、working tree観測はuntracked fileをambient Git configで隠せないよう明示します。

```text
HEAD before
  ↓
git status --porcelain --untracked-files=all
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
2. `--untracked-files=all`を明示したworking tree状態を取得できた
3. 二回目のHEADを取得できた
4. 二つのHEADが一致した

観測中にHEADが変わった場合はerrorとし、`Checked=false`のままにします。最終的に観測したHEADを`RepositoryRevision`へ残し、`RevisionStable=false`とします。

これにより、#255 / #308 / #310だけでなく、#387 final runnerが各lane後に行うsource再確認でも、観測中driftを「確認済み」と誤認しません。

## Git environmentによる観測意味の差し替えを拒否する

HEAD二重確認だけでは、Git subprocessへ渡るambient environment自体が観測対象を差し替える場合を閉じられません。

共有helperは、実行checkoutとは別のrepository / work tree / index等を観測し得る次のGit identity overrideをfail closedで拒否します。

```text
GIT_DIR
GIT_WORK_TREE
GIT_COMMON_DIR
GIT_INDEX_FILE
GIT_OBJECT_DIRECTORY
GIT_ALTERNATE_OBJECT_DIRECTORIES
GIT_NAMESPACE
```

さらにPR #477では、Gitのcommand-scope configをenvironmentから注入する次のkeyも、大小文字違いを含め拒否します。

```text
GIT_CONFIG_COUNT
GIT_CONFIG_PARAMETERS
GIT_CONFIG_KEY_*
GIT_CONFIG_VALUE_*
```

一時Git repositoryでuntracked fileだけを作った状態について、次を実測しました。

```text
git status --porcelain
=> ?? untracked.txt
```

一方、ambient environmentへ:

```text
GIT_CONFIG_COUNT=1
GIT_CONFIG_KEY_0=status.showUntrackedFiles
GIT_CONFIG_VALUE_0=no
```

を設定すると、同じ`git status --porcelain`がemptyになりました。

したがってGate Bのsource-fixed Evidenceでは、runtime config injectionを単に「通常は設定されない」と仮定せず、helperで拒否します。同時にstatus command自身も`--untracked-files=all`を明示し、`status.showUntrackedFiles`へ依存しません。

`GIT_PAGER`や`GIT_TERMINAL_PROMPT`等、source identity / command-scope configを変更しないGit environmentまで一律禁止するものではありません。

## 再現可能判定との関係

`IsReproducible`の上位契約は変更しません。

引き続き、少なくとも次をすべて要求します。

- 開始revisionが空でない
- 開始時working treeがclean
- post-run確認済み
- post-run revisionが開始revisionと同一
- `RevisionStable=true`
- 終了時working treeがclean

PR #419 / #477は、この判定へ渡す開始時 / post-run観測自体の整合性を強くする変更です。

## 検証

PR #419では、branch上の共有sourceを隔離Go moduleへ配置し、HEAD drift境界のpackage testを実行しました。

```text
ok  tei/tools/internal/repositoryevidence  0.002s
```

PR #477では、変更した`repository.go`と新しい`git_environment_test.go`を隔離Go moduleへ同内容で配置し、次を確認しました。

```text
ok  isolated/repositoryevidence  0.002s
```

PR #477のpure testでは少なくとも次を固定しています。

- `GIT_CONFIG_COUNT` / `GIT_CONFIG_PARAMETERS` / `GIT_CONFIG_KEY_*` / `GIT_CONFIG_VALUE_*`を大小文字違いも含め拒否する
- source観測意味を変更しない`GIT_PAGER` / `GIT_TERMINAL_PROMPT`は一律拒否しない
- start観測のstatus commandが`--untracked-files=all`を明示する
- post-run観測のstatus commandも`--untracked-files=all`を明示する
- 明示観測したuntracked fileをcleanへ格上げしない

既存pure testでは引き続き次を固定しています。

- start観測中にHEADが変わればerror
- post-run観測中にHEADが変わればerrorかつ`Checked=false`
- working tree観測後の二回目HEAD取得に失敗した場合も`Checked=true`にしない
- HEADが前後同一ならclean / dirtyを従来どおり観測できる
- status取得失敗を確認済み状態へ昇格させない

これらは**実TEI Repository checkout上の通常Go testではありません**。また、#255 / #308 / #310のactual Evidenceや#387 final convergence Evidenceの代替でもありません。

## この境界の限界

この補強はGit working tree全体をtransactional snapshotとして観測するものではありません。

例えば、観測中にfileが一時的に変更され、その後元の内容へ戻ったケースをすべて検出できるわけではありません。また、三command全体をGitの一つのtransactionへするものでもありません。

したがってactual Gate B runでは、引き続き次を運用上の前提とします。

- 専用のcheckoutを使う
- candidate revisionを実行中に切り替えない
- 並行操作でworking treeを変更しない
- Evidence出力をRepository内へ直接redirectしない

PR #419 / #477はこの運用を置き換えるのではなく、観測可能なHEAD切替やambient Git environmentによる誤観測をhelper側でもfail closedにする補強です。

## 完了条件

- [x] `InspectStart`をHEAD二重確認へ変更する（PR #419）
- [x] `InspectPost`をHEAD二重確認へ変更する（PR #419）
- [x] 観測中HEAD driftのpure testを追加する（PR #419）
- [x] Git identity overrideを共有source helperで拒否する
- [x] `git status --porcelain --untracked-files=all`へ固定する（PR #477）
- [x] `GIT_CONFIG_*` command-scope config injectionを拒否する（PR #477）
- [x] Git runtime config guardのpure testを追加する（PR #477）
- [x] 隔離Go moduleで共有package / pure testを実行する
- [x] #418へPR #477の再現条件と隔離test結果を記録する
- [ ] 実TEI Repository checkoutの通常Go環境で`go test ./tools/internal/repositoryevidence -count=1`を再確認する

最後の通常checkout testが完了するまで、GitHub #418は完了扱いにしません。

## 非目標

- Git working tree全体のtransactional snapshot化
- 瞬間的なfile変更と復元を含むあらゆる並行変更の検出
- Gitの全config sourceを独自sandboxへ置き換えること
- Repository source観測概念をCanonical modelへ追加すること
- #255 / #308 / #310 / #387の個別Evidence契約を置き換えること
- 本pure testをactual Gate B Evidenceへ格上げすること
- GitHub Actionsを再有効化すること

## 関連

- GitHub Issue #418
- GitHub Issue #255 / #308 / #310 / #387 / #273
- PR #336 / #355 / #409 / #412 / #419 / #477
- `tools/internal/repositoryevidence/`
- `plan/issue/000037-gate-b-three-lane-final-evidence.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_READINESS_AUDIT.md`
