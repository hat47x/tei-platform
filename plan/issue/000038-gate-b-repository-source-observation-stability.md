# ISSUE-000038: Gate BのRepository source観測中HEAD driftをfail closedにする

- 状態: In Progress — HEAD drift / Git environment / untracked / ignored / Git index非標準state guardまで実装・pure test済み、実TEI checkoutでの通常Go test待ち
- GitHub Issue: #418
- Gate B最終収束: `ISSUE-000037` / #387
- Gate B監査: `ISSUE-000029` / #273
- 関連Evidence: #255 / #308 / #310

## 目的

Gate BのEvidence取得で共有するRepository source観測について、HEAD切替、ambient Git environment、R外local file、Git indexの非標準flagによるclean誤観測をfail closedで拒否します。

この境界は`tools/internal/repositoryevidence`へ置き、#255 / #308 / #310と#387 final runner自身のsource再確認が共通利用します。Canonical modelやProvider固有意味へRepository観測概念を追加しません。

## 現行source観測

PR #419 / #477 / #479 / #483を経て、開始時・終了時とも概念上次の順で観測します。

```text
HEAD before
  ↓
git status --porcelain --untracked-files=all --ignored=matching
  ↓
git ls-files -v -z
  ↓
HEAD after
```

`HEAD before != HEAD after`なら一つのRepository stateとして採用しません。

`WorkingTreeClean=true`にするには、status出力が空であることに加えて、`git ls-files -v -z`のtracked entryがすべて通常`H` tagであることを要求します。

したがって次はいずれもclean candidate `R`へ格上げしません。

- tracked modification / staged change
- `??` untracked local state
- `!!` ignored local state
- `h` assume-unchanged
- `S` skip-worktree
- `s` assume-unchanged + skip-worktree
- malformed / 想定外のindex tag

index state取得自体に失敗した場合もsource観測失敗です。

### 開始時

`InspectStart`はHEAD before → status → index state → HEAD afterを完了し、HEADが同一の場合だけ`Source`を返します。HEADが安定していてもlocal stateまたは非標準index stateがあれば`WorkingTreeClean=false`です。

### 終了時・中間確認

`InspectPost`も同じ観測を行います。`postRunRepository.checked=true`にするには、HEAD / status / index state / second HEADの全観測が完了し、観測中HEAD driftがないことを要求します。

これにより#255 / #308 / #310だけでなく、#387 final runnerのsecret read前preflight、各lane後、final source確認でもR外入力を「確認済みclean」と誤認しません。

## Git environmentによる観測意味の差し替えを拒否する

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

さらにPR #477以降、command-scope config injectionも大小文字違いを含め拒否します。

```text
GIT_CONFIG_COUNT
GIT_CONFIG_PARAMETERS
GIT_CONFIG_KEY_*
GIT_CONFIG_VALUE_*
```

untracked fileがある一時repositoryで`status.showUntrackedFiles=no`をambient `GIT_CONFIG_*`から注入すると通常の`git status --porcelain`がemptyになることを実測済みです。そのためhelper側でconfig injectionを拒否し、status command側でも`--untracked-files=all`を明示します。

`GIT_PAGER` / `GIT_TERMINAL_PROMPT`等、source identityやstatus意味を変更しないGit environmentまで一律禁止しません。

## ignored local stateもR外入力として拒否する

PR #479では、Git ignore対象がactual Go inputへ入るケースを実測しました。

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

このため`.gitignore` / `.git/info/exclude`等でignoredになっているfileもcandidate `R`外のlocal inputとして扱います。Gate B Evidence専用checkoutでは`.idea` / `.vscode` / `.DS_Store` / `.tei` cache等を含むignored local stateも残しません。

`.gitignore`自体の意味は変更せず、「Gate B Evidence sourceとしてcleanと呼べる範囲」を厳しくします。

## Git indexの非標準stateもR外入力として拒否する

PR #483では、tracked fileでもindex flagによりR外source変更を通常statusから隠せることを実測しました。

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

両flagが同時なら`s`です。したがってEvidence専用checkoutではtracked entryが通常`H`以外のindex stateを持つ場合もclean candidateとして受理しません。

この境界は一般開発checkoutのGit運用を禁止するものではありません。Gate B actual Evidenceを取得する専用checkoutに限り、R以外のlocal source入力を残さないためのfail-closed条件です。

## 再現可能判定との関係

`IsReproducible`の上位契約は変更しません。引き続き少なくとも次を要求します。

- 開始revisionが非empty
- 開始時sourceがclean（tracked / untracked / ignored local stateなし、非標準index stateなし）
- post-run確認済み
- post-run revisionが開始revisionと同一
- `RevisionStable=true`
- 終了時sourceがclean

PR #419 / #477 / #479 / #483は、この判定へ渡すsource観測自体の整合性を強くする変更です。

## 検証

PR #419:

```text
ok  tei/tools/internal/repositoryevidence  0.002s
```

PR #477:

```text
ok  isolated/repositoryevidence  0.002s
```

PR #479:

```text
ok  isolated/repositoryevidence  0.003s
```

PR #483:

```text
ok  isolated/repositoryevidence  0.002s
```

PR #483では通常checkout取得を試みましたがDNS解決失敗で実行できませんでした。上記は**実TEI Repository checkout上の通常Go testではなく**、#255 / #308 / #310 / #387 actual Evidenceの代替でもありません。

pure / isolated testでは少なくとも次を固定しています。

- start / post-run観測中のHEAD drift拒否
- post-run drift時`Checked=false`
- second HEAD取得失敗を確認済みにしない
- Git identity override拒否
- `GIT_CONFIG_*` command-scope injection拒否
- `--untracked-files=all --ignored=matching`の明示
- untracked / ignored local stateをcleanへ格上げしない
- index stateを`git ls-files -v -z`で観測する
- `H`以外のindex tagをcleanへ格上げしない
- malformed / index観測失敗をfail closedで扱う

## この境界の限界

この補強はGit working tree全体をtransactional snapshotとして観測するものではありません。観測中にfileが一時変更され、その後元へ戻る全ケースを検出するものでもありません。

actual Gate B runでは引き続き次を前提とします。

- 専用checkoutを使う
- candidate revisionを実行中に切り替えない
- 並行操作でworking treeを変更しない
- tracked / untracked / ignored local stateを置かない
- assume-unchanged / skip-worktree等の非標準index flagを残さない
- Evidence出力をRepository内へredirectしない

## 完了条件

- [x] `InspectStart`をHEAD二重確認へ変更（PR #419）
- [x] `InspectPost`をHEAD二重確認へ変更（PR #419）
- [x] 観測中HEAD drift pure test
- [x] Git identity override拒否
- [x] `--untracked-files=all`固定（PR #477）
- [x] `GIT_CONFIG_*` injection拒否（PR #477）
- [x] ignored local stateを`--ignored=matching`で明示観測（PR #479）
- [x] untracked / ignored local stateをcleanへ格上げしないpure test
- [x] Git index stateを`git ls-files -v -z`で観測（PR #483）
- [x] assume-unchanged / skip-worktree / malformed index stateをcleanへ格上げしないpure test
- [x] 隔離Go moduleで共有package / pure test実行
- [ ] 実TEI Repository checkoutの通常Go環境で`go test ./tools/internal/repositoryevidence -count=1`を再確認

最後のactual checkout package testが完了するまで、GitHub #418は完了扱いにしません。

## 非目標

- Git working tree全体のtransactional snapshot化
- 瞬間的file変更と復元を含む全並行変更の検知
- Gitの全config sourceを独自sandboxへ置き換えること
- `.gitignore`内容の変更
- 一般開発checkoutでassume-unchanged / skip-worktreeを禁止すること
- Repository source観測概念をCanonical modelへ追加すること
- #255 / #308 / #310 / #387 Evidence契約の代替
- isolated testをProduct Evidenceへ格上げすること
- GitHub Actionsを再有効化すること

## 関連

- GitHub Issue #418
- GitHub Issue #255 / #308 / #310 / #387 / #273
- PR #336 / #355 / #409 / #412 / #419 / #420 / #477 / #479 / #481 / #483
- `tools/internal/repositoryevidence/`
- `plan/issue/000037-gate-b-three-lane-final-evidence.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_READINESS_AUDIT.md`
