# reference-external-coordination

ISSUE-000084 / GitHub #977のE1 Executable Referenceです。

目的は、複数のHuman / AI WorkerがWorkとResultを共有するために、TEI CoreへAgent Session、Workspace、Checkpoint、Fleet state等を追加しなくてもよいことをpressureすることです。最初のBackendは外部SaaSを使わず、標準GitとJSONだけで構成します。

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

runnerは、Product repositoryがcleanなGit checkoutであることを最初に確認します。最初のWorkは`origin.kind=human`として保持し、作成経路を`createdVia`へ分離します。一つのWorkへ二つの独立したAgent Resultを保存し、Result Aからは`origin.kind=agent`のfollow-up Workを生成します。base revisionからcurrent resultとstale resultを区別し、stale resultを自動適用しません。`actorRef=admin`という自己申告もAuthorityとして扱いません。

Result保存やBackend-native completionの前後でProduct Sourceが変化しないことも確認します。Workのpathと内容はcompletionで変更せず、Git text固有のstateは`native-state/<work>.json`だけに保持します。Work / Result locatorがbackend instanceを含むopaque stringであることに加え、同じ局所IDでも別instanceならlocatorが異なることを検査します。Artifactは本文へ埋め込まず、backend-instance-scoped locatorとSHA-256 digestを分けて保持します。token budget、quota、credential、Agent Session、Checkpoint、Acceptance / Runtime Authorityのfieldは交換文書へ含めません。最後にCoordination repositoryを削除し、Product repositoryのrevisionとclean stateが変化しないことを検査します。

## Run

```bash
bash tools/reference-external-coordination/run.sh
```

runnerはremote Git、GitHub API、外部SaaS、Go toolchain、TEI CLIを必要としません。標準Git、Bash、Python 3だけを利用します。

## このReferenceが主張しないこと

E1の結果だけで、Git repositoryをproductionの標準Coordination Backendには採用しません。Work / Result JSONもproduction wire formatとして固定せず、Human / Agent actor identityが認証済みだともみなしません。stale ResultのSemantic Rebase、Coordination stateからのCanonical Acceptance、Broker、Queue、distributed leaseの必要性も未確定です。ChatGPT、Copilot、Backlog、Box等のAdapter完成も主張しません。

Coordination Store削除後のCanonical reloadは、このrunner単独では検証しません。Canonical reloadとGit-nativeな開発経路はRQ-034 / ISSUE-000083が所有します。本Referenceは、Coordination層がProduct Sourceの第二正本やAuthority sourceにならないことを先に検証します。

GitHub Issuesを第二consumerとして比較した結果は`github-issues-profile.md`に記録しています。Backend stateを共通lifecycleへ黙って正規化せず、`origin`、`createdVia`、Backend actor、Authorityを分離します。locatorのschemeや文字列表現はReference例であり、TEI Coreがprovider固有URIを解釈するproduction wire formatとしては固定しません。

## Backend-native lifecycle境界

共通Work envelopeには`status`や`coordination.status`を置きません。Git text Referenceでは`native-state/<work-id>.json`をAdapter-local sidecarとして保持し、completion時にはこのsidecarのみ変更します。runnerはWork本文のGit blobが不変であることを検査します。同じ局所Work IDでもBackend instanceが異なればlocatorは異なります。これはReferenceの例示形式であり、production URI仕様の確定ではありません。
