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

Result Aは現在のProduct revisionを観測しています。Result Bは意図的に古いrevisionを持ち、producerの`actorRef`へ`admin`という自己申告値を設定します。どちらも`canonicalAccepted=false`、`runtimeAuthorized=false`のままです。

Workを`completed`へ更新した後も、Product repositoryのHEADとclean stateが変わっていないことを確認します。最後にCoordination repositoryを削除し、削除後もProduct repositoryが変化しないことを確認します。

## 検査内容

runnerは、Product repositoryがcleanなGit checkoutであることを最初に確認します。Workの`origin.kind`は`human`として保持し、一つのWorkへ二つの独立したAgent Resultを保存します。base revisionからcurrent resultとstale resultを区別し、stale resultを自動適用しません。`actorRef=admin`という自己申告もAuthorityとして扱いません。

Result保存やWork completionの前後でProduct Sourceが変化しないことも確認します。Artifactは本文へ埋め込まず、SHA-256 digest付き参照で保持します。token budget、quota、credential、Agent Session、Checkpointは交換文書へ含めません。最後にCoordination repositoryを削除し、Product repositoryのrevisionとclean stateが変化しないことを検査します。

## Run

```bash
bash tools/reference-external-coordination/run.sh
```

runnerはremote Git、GitHub API、外部SaaS、Go toolchain、TEI CLIを必要としません。標準Git、Bash、Python 3だけを利用します。

## このReferenceが主張しないこと

E1の結果だけで、Git repositoryをproductionの標準Coordination Backendには採用しません。Work / Result JSONもproduction wire formatとして固定せず、Human / Agent actor identityが認証済みだともみなしません。stale ResultのSemantic Rebase、Coordination stateからのCanonical Acceptance、Broker、Queue、distributed leaseの必要性も未確定です。ChatGPT、Copilot、Backlog、Box等のAdapter完成も主張しません。

Coordination Store削除後のCanonical reloadは、このrunner単独では検証しません。Canonical reloadとGit-nativeな開発経路はRQ-034 / ISSUE-000083が所有します。本Referenceは、Coordination層がProduct Sourceの第二正本やAuthority sourceにならないことを先に検証します。
