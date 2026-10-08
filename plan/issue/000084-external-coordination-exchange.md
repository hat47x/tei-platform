# ISSUE-000084: External Coordination Exchangeを既存基盤で実証する

- 状態: In Progress
- 起票日: 2026-10-08
- GitHub Issue: #977
- 関連RQ: RQ-034
- 関連ADR: ADR-0022
- 関連Issue: ISSUE-000027 / ISSUE-000080 / ISSUE-000082 / ISSUE-000083

## このIssueの目的

このIssueでは、複数のHuman / AI Workerが同じTEIプロジェクトへ参加するときの作業受け渡しを扱います。ChatGPT等のチャット、Coding Agent、API Agent、外部システムを調査するAgent、人間を、TEI専用Agent Runtimeへ従属させることは目的にしません。

RQ-034 / ISSUE-000083は、各開発環境がclean checkoutやworktreeからTEIを通常のGit開発として扱えることを検証します。本Issueはその外側にあり、複数のWorkerがWork、Evidence、Result等を共有する任意のCoordination層を検証します。

Coordination層を利用しなくてもRQ-034の直接Git / CLI経路は成立しなければなりません。

## Coreへ持ち込まない状態

Coordinationのためだけに、Agent Session、Agent Workspace、Checkpoint、一般的なPlan / progress、Agent fleet state、token / quota / subscription capacityをTEI Product stateへ追加しません。

モデル選択、余剰API quota、定額Chatの利用量、local compute等の割り当てはTEIの責務外です。外部Scheduler、Fleet Manager、各Agent基盤、または利用者が判断します。

## Work exchangeの境界

Referenceでは、利用可能なWorkの発見から始め、目的、対象Source、base revision、制約、期待成果を取得できることを確認します。必要な場合は着手を宣言し、Observation、Evidence、Proposal、変更候補、Verification summaryを提出します。作業中に新しい論点が見つかった場合はfollow-up workを提案でき、conflict、blocked、insufficient informationも結果として報告します。最後にcompletionを記録できます。

これらの項目をproduction schemaとして先に固定しません。Git text profileと性質の異なる第二consumerで共通して必要になった情報だけを残します。

## Backendを固定しない

Coordination Backendは一製品へ固定しません。候補にはGit repository、GitHub Issues、Backlog、Jira、Azure DevOps Boards、Notion、Microsoft Lists / SharePoint、Box等があります。

一つのBackendにすべての役割を要求しません。TaskをBacklog、ArtifactをBox、SourceをGitHubで管理する構成も許容します。

企業がすでに利用しているサービスを転用できることを重要な導入条件とします。ただし、契約中であることだけでは利用可能と判断しません。API利用可否、Service Identity、管理者承認、Read / Write scope、Webhook、API制限、監査、Retention等はBackend qualificationとして分離して確認します。

## Authority境界

Coordination BackendへWorkやResultが保存されても、Canonical Acceptance、Semantic validity、Runtime Capability authority、Resource / Secret authorityは得ません。

正式なSource変更はADR-0022 / ISSUE-000027のAcceptance原則へ戻します。Runtimeや外部Resourceへの副作用は既存のAuthority境界へ戻します。

外部システムを調査するAgentが取得した文章は命令ではなくObservation / Evidenceです。文脈、Provenance、鮮度はISSUE-000082の境界へ接続します。

## E0: 最小交換契約

最初のReferenceでは、WorkとResultをBackend非依存の小さいenvelopeとして扱います。Work / Resultの参照はBackend種別と局所IDの組に固定せず、**Adapterが一意に解決できるbackend-instance-scoped opaque locator**として扱います。複数Repository、tenant、workspace、projectを同時利用しても衝突しないscopeをAdapter側で持たせます。TEI Coreがlocatorをparseしてprovider固有意味を理解する必要はありません。

Workのoriginやactor referenceもCoordination上のprovenance / descriptive metadataであり、本人性、Canonical identity、Acceptance authorityではありません。Resultには、対象Work locator、任意のresult locator、producer、観測したbase revision、summary、Artifact参照等を持たせます。model名、token budget、quota、credential、Agent Session、Checkpointは最小交換契約へ含めません。Canonical AcceptanceやRuntime Authorityも交換文書のfieldとして表現せず、既存の別契約へ戻します。

大型Artifactは本文へ埋めず、backend instanceを含めて解決可能なlocator、optional digest、media type等の参照で扱えるようにします。digestはcontent integrityの補助であり、Canonical identityやAcceptanceの証明ではありません。Backend stateは共通contractへ先に正規化しません。Git textの`ready / completed`はE1 Adapter内の状態であり、GitHub Issues等のnative stateとは別に扱います。

## E1: Git text profile

最初のExecutable Referenceは`tools/reference-external-coordination/`に置きます。外部SaaSやTEI Coreの変更を必要とせず、標準GitとJSONだけを利用します。

runnerは一時的なCoordination repositoryを作り、Human-originのWorkを1件保存します。同じWorkに対して二つのAgent Resultを保存し、一方はcurrent base revision、もう一方は意図的にstaleなbase revisionと自己申告の`admin` actor labelを持たせます。current側のResultからはAgent-originのfollow-up Workを1件生成し、Human-originとAgent-originを同じStore上で区別します。

この状態でも、どちらのResultにもCanonical AcceptanceやRuntime authorityを与えません。Authorityを表すfield自体をWork / Result envelopeへ置かないことで、Coordination metadataからAuthorityへ昇格する経路を作りません。Workは固定pathのまま`coordination.status`だけを`completed`へ更新し、directory名とfieldによる状態の二重管理を避けます。最後にCoordination repository自体を削除し、元のProduct repositoryのrevisionとclean stateが変化しないことを確認します。

### E1で確認するnegative pressure

E1では、一つのWorkへ複数Resultを保存できることに加え、Human-origin WorkとAgent-origin follow-up Workを区別できることを確認します。stale Resultは履歴として保持しても自動適用せず、actor labelの自己申告もAuthorityへ昇格させません。Authorityを否定するbooleanを置くのではなく、Authority fieldを交換契約から除外します。Workを`completed`へ変更してもProduct Sourceの変更とはみなしません。Agent数とGit branch数は1:1にせず、token、quota、credential、Agent Sessionも交換文書へ要求しません。ArtifactはWork本文へ埋め込まず、digest付き参照として扱います。最後にCoordination Storeを削除し、Product repositoryが変化しないことを確かめます。

E1はCanonical Worldを実際にreloadする試験ではありません。その部分はRQ-034 / ISSUE-000083が所有します。本Issueでは、Coordination層がProduct Sourceの第二正本にならないことを先にpressureします。

## Optional Broker

多数のAgentから同じBackendへ接続する必要がある場合は、薄いBrokerを任意で配置できます。BrokerはTEI Coreの必須Runtimeにはしません。

Brokerの候補責務は、認証情報の隔離、Protocol正規化、Backend Adapter、Idempotency、Rate control、必要最小限の短時間Leaseです。AI推論、長時間Task、AI capacity scheduling、Canonical state ownershipはBrokerへ持たせません。

AWS API Gateway + Lambda等は低コストなReference候補ですが、AWSをArchitecture Requirementにはしません。

## E2-A: GitHub Issuesを実Task Tracker consumerとしてpressureする

GitHub Issue #977自体を、Git textとは性質の異なる最初のTask Tracker consumerとして観測します。#977の目的は利用者の依頼から生じていますが、Issue作成はAgentがConnector経由で実行しました。GitHub上のauthorは接続アカウントとして記録されるため、author fieldだけでは「誰の意図から生じたWorkか」と「どの経路で作成されたか」を区別できません。

Reference上は次のように分けます。

```text
workRef      = github-issue://hat47x/tei-platform/977
origin.kind  = human
createdVia   = agent-connector
backendActor = connected GitHub account

resultRef    = github-pr://hat47x/tei-platform/978
resultKind   = reference.evidence

Canonical Acceptance / Runtime Authority
  = exchange documentには表現しない
```

このpressureから、`origin`と`createdVia`を同じfieldへ畳まないことをE2の要件候補とします。また、`workRef` / `resultRef`はGitHubという種別名とnumberの組ではなく、Repository instanceまで含めてAdapterが一意に解決できるlocatorとして扱います。例示URIはwire formatの固定ではありません。Backendのnative authorは監査上の事実として保持できますが、Human-origin、Agent-origin、Acceptance authorityのいずれもauthorだけから推定しません。

#978がDraftであることもCoordination上の進捗であり、それ自体をCanonical Acceptanceにはしません。GitHub Issuesを標準Backendへ昇格させる判断も、このconsumerだけでは行いません。

## E2以降

E1のGit text profile成立後、Gitとは性質の異なるBackendを第二consumerとして接続します。Task Tracker、Structured Knowledge、File / Artifact Storeのいずれかから、既存環境で実証しやすいものを選びます。

第二consumerで共通して必要になったfieldだけを最小契約候補へ残し、Backend固有metadataはAdapter側へ戻します。GitHub Issues profileは`tools/reference-external-coordination/github-issues-profile.md`へ記録します。

その後に必要性が確認できた場合だけ、HTTP / MCPから同じ契約を利用するthin Brokerをpressureします。BrokerなしのGit / CLI経路を壊しません。

## Product Requirementへの昇格条件

現時点では本Issueを新しいRQへ固定しません。最初にGit text profileでBrokerなしの最小round tripを成立させ、次にGitとは性質の異なる第二Backendで同じ最小契約を試します。そのうえで、Coordination dataを削除してもauthoritative Source / Canonical meaningの成立条件が変わらないこと、ISSUE-000083のdirect Git / CLI pathがCoordination層なしで成立し続けることを確認します。Human起票とAgent起票をBackend固有の無理な表現なしに区別でき、transient progressやchain-of-thoughtを永続化しなくてもhandoffが成立することまで確認した後、共通部分だけをProduct Requirement候補へ昇格します。

## 非目標

本Issueでは、Devin相当のAgent orchestration platformをTEI内へ実装しません。AI token、quota、subscription、Provider / model routing、Chat Sessionも管理対象にしません。Git branchをAgent schedulerとして使うこと、GitHub Issuesや特定SaaSを唯一の正本にすること、Agentのchain-of-thoughtを保存することも対象外です。Coordination BackendをCanonical meaningの第二正本にはせず、MCPを唯一のTransportにも、Broker常駐をTEI利用の前提にもしません。

## 関連文書

- `plan/issue/000027-ai-proposal-acceptance-boundary.md`
- `plan/issue/000080-extensible-plugin-capability-contract.md`
- `plan/issue/000082-context-scoped-epistemic-knowledge.md`
- `plan/issue/000083-tool-neutral-development-reference.md`
- `plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
