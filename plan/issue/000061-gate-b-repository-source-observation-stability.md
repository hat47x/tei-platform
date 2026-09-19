# ISSUE-000061: Gate BのRepository source観測中HEAD driftをfail closedにする

- 状態: In Progress — HEAD drift / Git environment / untracked / ignored / Git index local state guardまで実装・pure test済み、実TEI checkoutでの通常Go test待ち
- GitHub Issue: #418
- Gate B最終収束: `ISSUE-000037` / #387
- Gate B監査: `ISSUE-000029` / #273
- 関連Evidence: #255 / #308 / #310

## 目的

Gate BのEvidence取得で共有するRepository source観測について、`git status`を取得している途中のHEAD切替や、ambient Git environment・R外local file・非標準Git index stateによるclean誤観測をfail closedで拒否します。

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

さらに、`git status`の観測意味はambient Git configで変えられ、untracked fileやignored fileは通常のstatusだけではcandidate revision `R`外の入力として見落とし得ます。特にignored Go sourceはGit上cleanに見えたまま`go list` / `go test`へ入るため、Gate B Evidenceのsource固定と矛盾します。

PR #483では、tracked fileでも`assume-unchanged` / `skip-worktree`をGit indexへ設定すると、変更後sourceがGo入力へ入る一方で`git status --porcelain --untracked-files=all --ignored=matching`がemptyのままになる経路も実測しました。

actual Gate B runでは専用の固定checkoutを使う運用が前提ですが、その運用だけに依存せず、共有helperでも観測可能なsource drift / local source state / index source stateを拒否します。

## 採用する観測境界

PR #419以降、開始時・終了時ともHEADをworking tree観測の前後で確認します。PR #477でuntracked fileを明示観測し、PR #479でignored local stateも同じclean判定へ含め、PR #483でtracked entryのGit index stateも観測対象へ加えました。

```text
HEAD before
  ↓
git status --porcelain --untracked-files=all --ignored=matching
  ↓
git ls-files -v -z
  ↓
HEAD after
```

`HEAD before`と`HEAD after`が一致しない場合、その観測を一つのRepository stateとは扱いません。

status出力が空、かつ`git ls-files -v -z`の全tracked entryが通常の`H` tagである場合だけ`WorkingTreeClean=true`です。したがって次はいずれもGate Bのclean candidateとして扱いません。

- tracked modification / staged change
- `??` untracked local state
- `!!` ignored local state
- `h` assume-unchanged tracked entry
- `S` skip-worktree tracked entry
- `s` assume-unchanged + skip-worktree tracked entry
- malformed / 想定外の非`H` index tag

### 開始時

`InspectStart`は、working tree / index state観測前後のHEADが同じ場合だけ`Source`を返します。

観測中にHEADが変わった場合はerrorとし、切替前revisionと切替後のlocal source stateを組み合わせた`Source`を返しません。

HEADが安定している場合は、tracked / untracked / ignored local stateとGit index flagの有無を`WorkingTreeClean`へ反映します。dirtyであること自体を観測不能とはしません。各runnerが自身の用途に応じて、通常Evidenceでは拒否し、診断モードがある場合はその契約に従います。

### 終了時・中間確認

`InspectPost`も同じ観測を行います。

`postRunRepository.checked = true`にするのは、次をすべて完了した場合だけです。

1. 最初のHEADを取得できた
2. `--untracked-files=all --ignored=matching`を明示したworking tree状態を取得できた
3. `git ls-files -v -z`でtracked index stateを取得できた
4. 二回目のHEADを取得できた
5. 二つのHEADが一致した

観測中にHEADが変わった場合はerrorとし、`Checked=false`のままにします。最終的に観測したHEADを`RepositoryRevision`へ残し、`RevisionStable=false`とします。

これにより、#255 / #308 / #310だけでなく、#387 final runnerが各lane後に行うsource再確認でも、観測中driftやR外local source/index stateを「確認済みclean」と誤認しません。

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

## ignored local stateもR外入力として拒否する

PR #479では、Git ignore対象がactual Go inputへ入るケースを一時repositoryで実測しました。

```text
.git/info/exclude: *.local.go
p/hidden.local.go を作成

git status --porcelain --untracked-files=all
=> empty

git status --porcelain --untracked-files=all --ignored=matching
=> !! p/hidden.local.go

go list -f '{{join .GoFiles ","}}' ./p
=> base.go,hidden.local.go
```

このため`.gitignore` / `.git/info/exclude`等でignoredになっているfileも、Gate B candidate `R`外のlocal inputとして扱います。

Gate B final Evidenceはdedicated checkout前提なので、`.idea` / `.vscode` / `.DS_Store` / `.tei` cache等を含め、ignored local stateが存在するcheckoutはclean candidateとして受理しません。必要なlocal stateはEvidence専用checkoutの外へ置くか、専用checkoutから除去してから実行します。

これは`.gitignore`自体の意味を変更するものではなく、「Gate B Evidence sourceとしてcleanと呼べる範囲」を厳しくするものです。

## Git indexの非標準source stateも拒否する

PR #483では、tracked sourceであってもGit index flagによって変更を通常statusから隠せることを一時repositoryで実測しました。

### assume-unchanged

```text
git update-index --assume-unchanged main.go
# main.goを変更

git status --porcelain --untracked-files=all --ignored=matching
=> empty

git ls-files -v
=> h main.go

go test ./...
=> 変更後sourceを入力として実行
```

### skip-worktree

```text
git update-index --skip-worktree main.go
# main.goを変更

git status --porcelain --untracked-files=all --ignored=matching
=> empty

git ls-files -v
=> S main.go

go test ./...
=> 変更後sourceを入力として実行
```

両flagを同時に持つ場合は`git ls-files -v`で`s`になります。

PR #483以降、共有helperはstart / post-runとも`git ls-files -v -z`を実行し、全tracked entryが通常の`H` tagである場合だけindex stateをcleanと扱います。`h` / `S` / `s`だけでなく、malformedまたは想定外の非`H` tagもfail closedで`WorkingTreeClean=false`です。index state自体を取得できない場合はsource観測失敗です。

Gate B Evidence専用checkoutでは`assume-unchanged`を使わず、sparse checkout由来を含む`skip-worktree`も残さないfull checkoutを用います。

## 再現可能判定との関係

`IsReproducible`の上位契約は変更しません。

引き続き、少なくとも次をすべて要求します。

- 開始revisionが空でない
- 開始時working treeがclean（tracked / untracked / ignored local stateなし、tracked index entryが通常`H`のみ）
- post-run確認済み
- post-run revisionが開始revisionと同一
- `RevisionStable=true`
- 終了時working treeがclean（tracked / untracked / ignored local stateなし、tracked index entryが通常`H`のみ）

PR #419 / #477 / #479 / #483は、この判定へ渡す開始時 / post-run観測自体の整合性を強くする変更です。

## 検証

PR #419では、branch上の共有sourceを隔離Go moduleへ配置し、HEAD drift境界のpackage testを実行しました。元の検証で固定したGit blob SHAも履歴として残します。

```text
repository.go       5a69ec380c45feb41e1fb14732f4e396bebb18ba
repository_test.go  4f0484fef3089ad2961cfec0c4d44ad37c4ff10c
```

結果:

```text
ok  tei/tools/internal/repositoryevidence  0.002s
```

`gofmt -d`にも差分はありませんでした。

PR #477では、変更した`repository.go`と新しい`git_environment_test.go`を隔離Go moduleへ同内容で配置し、次を確認しました。

```text
ok  isolated/repositoryevidence  0.002s
```

PR #479でも同じ共有helper / testを隔離moduleへ配置し、ignored-state guardを含めて次を確認しました。

```text
ok  isolated/repositoryevidence  0.003s
```

PR #483では、branch上の`repository_test.go` / `git_environment_test.go`相当を含む共有helperを隔離moduleへ配置し、`gofmt -d`差分なしと次を確認しました。

```text
ok  isolated/repositoryevidence  0.003s
```

pure testでは少なくとも次を固定しています。

- `GIT_CONFIG_COUNT` / `GIT_CONFIG_PARAMETERS` / `GIT_CONFIG_KEY_*` / `GIT_CONFIG_VALUE_*`を大小文字違いも含め拒否する
- source identity / command-scope configを変更しない`GIT_PAGER` / `GIT_TERMINAL_PROMPT`は一律拒否しない
- start / post-run観測のstatus commandが`--untracked-files=all --ignored=matching`を明示する
- start / post-run観測が`git ls-files -v -z`を実行する
- `??` untracked local stateをcleanへ格上げしない
- `!!` ignored local stateをcleanへ格上げしない
- `h` assume-unchanged / `S` skip-worktree / `s`両方をcleanへ格上げしない
- malformed / 想定外の非`H` index tagをcleanへ格上げしない
- index state取得失敗を確認済みcleanへ格上げしない
- start観測中にHEADが変わればerror
- post-run観測中にHEADが変わればerrorかつ`Checked=false`
- local source/index state観測後の二回目HEAD取得に失敗した場合も`Checked=true`にしない
- HEADが前後同一ならclean / dirtyを観測できる
- status取得失敗を確認済み状態へ昇格させない

これらは**実TEI Repository checkout上の通常Go testではありません**。また、#255 / #308 / #310のactual Evidenceや#387 final convergence Evidenceの代替でもありません。

## この境界の限界

この補強はGit working tree / index全体をtransactional snapshotとして観測するものではありません。

例えば、観測中にfileやindex flagが一時的に変更され、その後元の内容へ戻ったケースをすべて検出できるわけではありません。また、四command全体をGitの一つのtransactionへするものでもありません。

したがってactual Gate B runでは、引き続き次を運用上の前提とします。

- 専用のfull checkoutを使う
- `assume-unchanged` / `skip-worktree`を使わない
- candidate revisionを実行中に切り替えない
- 並行操作でworking tree / indexを変更しない
- tracked / untracked / ignored local stateを置かない
- Evidence出力をRepository内へ直接redirectしない

PR #419 / #477 / #479 / #483はこの運用を置き換えるのではなく、観測可能なHEAD切替やambient Git environment、R外local source/index stateによる誤観測をhelper側でもfail closedにする補強です。

## 完了条件

- [x] `InspectStart`をHEAD二重確認へ変更する（PR #419）
- [x] `InspectPost`をHEAD二重確認へ変更する（PR #419）
- [x] 観測中HEAD driftのpure testを追加する（PR #419）
- [x] Git identity overrideを共有source helperで拒否する
- [x] `git status --porcelain --untracked-files=all`へ固定する（PR #477）
- [x] `GIT_CONFIG_*` command-scope config injectionを拒否する（PR #477）
- [x] ignored local stateを`--ignored=matching`で明示観測する（PR #479）
- [x] untracked / ignored local stateをcleanへ格上げしないpure testを追加する
- [x] `git ls-files -v -z`でGit index source stateを観測する（PR #483）
- [x] `assume-unchanged` / `skip-worktree` / 想定外index tagをcleanへ格上げしないpure testを追加する（PR #483）
- [x] 隔離Go moduleで共有package / pure testを実行する
- [x] #387とRepository正本へHEAD drift境界を同期する（PR #420）
- [x] #418へPR #477 / #479の再現条件と隔離test結果を記録する
- [ ] 実TEI Repository checkoutの通常Go環境で`go test ./tools/internal/repositoryevidence -count=1`を再確認する

最後の通常checkout testが完了するまで、GitHub #418は完了扱いにしません。

## 非目標

- Git working tree / index全体のtransactional snapshot化
- 瞬間的なfile / index変更と復元を含むあらゆる並行変更の検出
- Gitの全config sourceを独自sandboxへ置き換えること
- `.gitignore`内容そのものを変更すること
- Repository source観測概念をCanonical modelへ追加すること
- #255 / #308 / #310 / #387の個別Evidence契約を置き換えること
- 本pure testをactual Gate B Evidenceへ格上げすること
- GitHub Actionsを再有効化すること

## 関連

- GitHub Issue #418
- GitHub Issue #255 / #308 / #310 / #387 / #273
- PR #336 / #355 / #409 / #412 / #419 / #420 / #477 / #479 / #483
- `tools/internal/repositoryevidence/`
- `plan/issue/000037-gate-b-three-lane-final-evidence.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_READINESS_AUDIT.md`