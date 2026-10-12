# reference-external-coordination

ISSUE-000084 / GitHub #977のE1 Executable Referenceです。

目的は、複数のHuman / AI WorkerがWorkとResultを共有するために、TEI CoreへAgent Session、Workspace、Checkpoint、Fleet state等を追加しなくてもよいことをpressureすることです。最初のBackendは外部SaaSを使わず、標準GitとJSONだけで構成します。

## 確認済みの範囲（2026-10-12）

Git textを用いるE1と、GitHub Issuesを別のTask Trackerとして観測するE2-Aは、[Issue #977](https://github.com/hat47x/tei-platform/issues/977)・[Issue #983](https://github.com/hat47x/tei-platform/issues/983)で参照例の検証を完了しました。E2-Aの比較結果は[GitHub Issues profile](github-issues-profile.md)に記録しています。

[PR #1298](https://github.com/hat47x/tei-platform/pull/1298)の統合後、[GitHub Actions #38240846308](https://github.com/hat47x/tei-platform/actions/runs/38240846308)ではmainのコミット`d540fcf20c5a2a7a2e31f1081da49803712569b7`の完全なGit作業領域で、リポジトリ全体の健全性確認4項目、E1回帰18ケース、E1本体、実行前後の版・変更状態の確認がすべて成功しました。この記録は**検証したコミットに限定**され、以後のmainの全版が自動的に合格したことは意味しません。

本検証は外部との作業受渡しに関するものです。Canonical Sourceの再読込やGitを用いた開発工程全体の検証を含みません。独立した開発工程の検証を扱っていた[Issue #972](https://github.com/hat47x/tei-platform/issues/972)は不要として取り下げられており、その未実施項目を本参照例の合格として扱いません。本番用のGitHub Issues Adapter、第三のBackend、Brokerの実装も対象外です。


```text
product repository
  |
  | base revisionを参照するだけ
  v
separate coordination repository
  -> human-origin work
  -> agent result A
  -> agent result B
  -> completed
  -> repository deletion

product repository remains unchanged
```

## Scenario

runnerは、現在のProduct repositoryとは別に一時Git repositoryを作ります。そこへHuman-originのWorkを1件保存し、作成経路は`createdVia=automation/reference-runner`として別に記録します。同じWorkへ二つのAgent Resultを追加します。

Result Aは現在のProduct revisionを観測し、Agent-originのfollow-up Workを1件生成します。Result Bは意図的に古いrevisionを持ち、producerの`actorRef`へ`admin`という自己申告値を設定します。Human-origin WorkとAgent-origin Workを同じStoreへ保存しても、`origin`と`createdVia`は別に保持します。Work / Result / Artifactの参照は、`git-text://reference-store/...`というReference用のbackend-instance-scoped locatorで表し、単なるBackend種別 + 局所IDへ依存しません。Canonical AcceptanceやRuntime Authorityを表すfieldは交換文書へ置きません。

Git text profileの`ready / completed`は共通Work envelopeへ置かず、Adapter-localな`native-state` sidecarへ保存します。completionではnative-stateだけを変更し、Work envelopeのblobが変わらないことを確認します。その後もProduct repositoryのHEADとclean stateが変わっていないことを確認し、最後にCoordination repositoryを削除してもProduct repositoryが変化しないことを確認します。

## 検査内容

runnerは、Product repositoryがcleanなGit checkoutであることを最初に確認します。通常のtracked / untracked変更だけでなく、ignored artifactと、Git indexの`assume-unchanged` / `skip-worktree`によって隠れた状態も拒否します。`GIT_WORK_TREE`、`GIT_DIR`、`GIT_INDEX_FILE`、`GIT_CONFIG_COUNT`などのGit環境変数で対象checkoutをすり替えられる状態も、検査開始前にexit 2で拒否します。環境変数を使わない`.git/config`の`core.worktree`による参照先の変更にも対応し、runner自身の物理checkoutルートとGitが報告するworktreeルートの一致を実行前・Coordination処理後・Store削除後のそれぞれで確認します。Coordination処理後とStore削除後にも同じclean判定を行い、Source不変の主張を通常の`git status`のみに依存させません。最初のWorkは`origin.kind=human`として保持し、作成経路を`createdVia`へ分離します。一つのWorkへ二つの独立したAgent Resultを保存し、Result Aからは`origin.kind=agent`のfollow-up Workを生成します。base revisionからcurrent resultとstale resultを区別し、stale resultを自動適用しません。`actorRef=admin`という自己申告もAuthorityとして扱いません。

Result保存やBackend-native completionの前後でProduct Sourceが変化しないことも確認します。Workのpathと内容はcompletionで変更せず、Git text固有のstateは`native-state/<work>.json`だけに保持します。Work / Result locatorがbackend instanceを含むopaque stringであることに加え、同じ局所IDでも別instanceならlocatorが異なることを検査します。Artifactは本文へ埋め込まず、backend-instance-scoped locatorとSHA-256 digestを分けて保持します。token budget、quota、credential、Agent Session、Checkpoint、Acceptance / Runtime Authorityのfieldは交換文書へ含めません。最後にCoordination repositoryを削除し、Product repositoryのrevisionとclean stateが変化しないことを検査します。

## Run

```bash
bash tools/reference-external-coordination/run.sh
```

runnerはremote Git、GitHub API、外部SaaS、Go toolchain、TEI CLIを必要としません。標準Git、Bash、Python 3だけを利用します。 PythonのJSON・意味検証では`python3 -I`（isolated mode）を使用します。これにより呼び出し元の`PYTHONOPTIMIZE`で`assert`が無効化されることを防ぎます。

合成Git repositoryでのclean判定回帰を個別に実行する場合は、次を用います。

```bash
bash tools/reference-external-coordination/clean-check-test.sh
```

この試験は正常なcheckoutでのE1 round tripと、tracked / untracked / ignored artifact、`assume-unchanged`、`skip-worktree`、non-Git directoryの拒否を確認します。さらに`GIT_WORK_TREE`で参照先を別のclean treeへ切り替え、実際のProduct checkoutのtracked変更が`git status`から消えるケースを再現して拒否します。`GIT_INDEX_FILE`、`GIT_DIR`、`GIT_CONFIG_COUNT`も拒否を確認します。さらに環境変数を使用せず`.git/config`の`core.worktree`を別のclean copyに変更するケースと、Coordination実行途中で同様に変更するケースを追加し、両方でroot mismatchを検出することを確認します。さらに、Coordination Storeへの最初のcommit直前にignored artifactまたはhidden-index状態を挿入し、処理後のclean判定がexit 1で拒否することを確認します。Store削除直後にもignored artifactを挿入し、削除後のclean判定が同様に拒否することを確認します。さらに、`PYTHONOPTIMIZE=2`の条件下でWork envelopeのoriginを一時的に改ざんし、Python側の意味検証が失敗することも確認します。注入は合成fixture内でのみ行い、元のProduct checkoutは変更しません。合成環境での成功は、trustedな実TEI checkout上のfull Repository HealthやCanonical reloadのEvidenceには含めません。

## このReferenceが主張しないこと

E1の結果だけで、Git repositoryをproductionの標準Coordination Backendには採用しません。Work / Result JSONもproduction wire formatとして固定せず、Human / Agent actor identityが認証済みだともみなしません。stale ResultのSemantic Rebase、Coordination stateからのCanonical Acceptance、Broker、Queue、distributed leaseの必要性も未確定です。ChatGPT、Copilot、Backlog、Box等のAdapter完成も主張しません。

Coordination Store削除後のCanonical Source再読込は、このrunnerでは検証しません。Gitを用いた独立の開発工程を扱ったIssue #972は不要として取り下げ済みであり、同Issueの検証を完了したことにはしません。本参照例は、外部の協調用StoreがProduct Sourceの第二正本や受理・実行の権限元にならないことを確認します。

GitHub Issuesを第二consumerとして比較した結果は`github-issues-profile.md`に記録しています。Backend stateを共通lifecycleへ黙って正規化せず、`origin`、`createdVia`、Backend actor、Authorityを分離します。locatorのschemeや文字列表現はReference例であり、TEI Coreがprovider固有URIを解釈するproduction wire formatとしては固定しません。

## Backend-native lifecycle境界

共通Work envelopeには`status`や`coordination.status`を置きません。Git text Referenceでは`native-state/<work-id>.json`をAdapter-local sidecarとして保持し、completion時にはこのsidecarのみ変更します。runnerはWork本文のGit blobが不変であることを検査します。同じ局所Work IDでもBackend instanceが異なればlocatorは異なります。これはReferenceの例示形式であり、production URI仕様の確定ではありません。
