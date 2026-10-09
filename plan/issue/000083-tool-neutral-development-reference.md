# ISSUE-000083: Tool-neutral development Referenceをclean checkoutから実証する

- 状態: In Progress
- 起票日: 2026-10-07
- GitHub Issue: #972
- 関連RQ: RQ-015 / RQ-017 / RQ-021 / RQ-027 / RQ-034
- 関連ADR: ADR-0022 / ADR-0031
- 関連Issue: ISSUE-000027 / ISSUE-000046 / ISSUE-000047 / ISSUE-000079 / ISSUE-000080
- Primary PAS: PAS-06
- Secondary PAS: PAS-01

## このIssueを最初に読むときの要点

このIssueは、TEI内へDevin相当のAgent Runtimeを実装するものではありません。

確認するのは、ChatGPT等のチャット実行環境、Codex、Claude Code、通常のIDE / CLI、CI等が、TEI専用Agent Adapterや常駐Serviceを前提にせず、通常のRepository開発としてTEIを扱えるかです。

```text
clean checkout / task-local worktree
  -> current Source / Canonical contextを取得
  -> file-native sourceを直接編集
  -> build / validation / test
  -> Git commit / branch等として変更を確定
  -> 別のintegration contextからreload
  -> Canonical meaning / Validation / Conformanceを再確認
```

TEIが管理する対象を、開発Tool自身が所有するSession、Workspace、Checkpoint、Plan、一般的なTest状態まで広げません。

## 減らす不確実性

主に次の問いへ答えます。

1. 特定のChat UIやAgent frameworkが無くても、AI / HumanがTEI Sourceを理解して変更できるか。
2. TEI CLIだけで、変更前の意味理解と変更後の再読込・検証に必要な情報へ到達できるか。
3. MCPが必要な場合、CLIと別の意味実装を作らず同じapplication serviceをtransportとして公開できるか。
4. Git commit / branch / Pull Request等を、そのまま第一級の変更成果物として利用できるか。
5. Structured ProposalやTEI独自Change Setを全変更へ必須にしなくても、Shared Acceptance / Validationの意味境界を維持できるか。
6. 開発環境のShell / Filesystem / Git authorityと、TEI Application RuntimeのCapability / Resource authorityを分離できるか。

## E0 — current developer surface audit

current `cmd/tei`には、少なくとも次のread / inspection / validation surfaceがあります。

- `workspace-files`
- `source-index`
- `knowledge-context`
- `knowledge-qualify`
- `validate`
- `world`
- `requirements`
- `index`
- `consumer-impact`
- Database / release / deployment系の各inspection / planning command

RQ-034の最初のReferenceでは、これらのうち必要な最小集合だけを利用します。Agent用の新しい万能commandを先に追加しません。

current RepositoryのArchitectureは、developer-facing HostのMCP / control interfaceと、Application自身が業務Targetとして公開するMCPを区別しています。一方、current Repositoryのcode searchではTEI自身のdeveloper MCP server実装を確認できていません。

したがって、最初のE2EはCLI-onlyで成立させます。CLI-onlyで成立しないGapがある場合は、まず通常のapplication service / CLI Gapとして扱います。MCPは、そのGapを隠す別実装にはしません。

## Development Hostの前提

RQ-034は、すべてのChat UIが同じ能力を持つことを要求しません。

改修からBuild / Testまで一つの環境で完結するには、Host側に少なくとも次の能力が必要です。

```text
repository materialization
  + filesystem read / write
  + process execution
  + Git operations
  + required language toolchain
```

network accessやMCPは常に必須ではありません。必要なdependencyや外部Resourceへ到達する作業でだけ追加前提になります。

この前提はTEIのRuntime CapabilityやCanonical modelへ登録する対象ではありません。開発Hostがそのtaskを完遂できるかを判断する環境条件です。

たとえばGitHub Contents APIだけを利用できるChat環境はSource変更を作成できますが、process executionやRepository checkoutが無ければ、その環境単独ではRQ-034のfull build / test loopを実証できません。その場合は、変更をGitへ渡して別の標準開発環境で継続できます。

反対に、Chat / Coding Agent側がcheckout、shell、Gitを持つ場合は、TEI専用Adapterを追加せず、そのHost内で通常の開発loopを完結できます。

この区別を、`full / partial`等の永続的なTEI状態にはしません。

## E1 — CLI-only round trip

既存Reference Workspaceの小さい変更を一つ選び、専用Agent Adapterなしで次を実行します。

1. clean checkoutまたはtask-local worktreeを作る。
2. `tei source-index` / `knowledge-context`等から変更対象を特定する。
3. file-native authoring sourceを直接編集する。
4. `tei validate`と対象Reference test / Conformanceを実行する。
5. Git diffを確認し、commit / branch等の通常のGit成果物として確定する。
6. 変更元とは別のclean contextでresult revisionを読む。
7. `tei world` / `validate`等からCanonical meaningを再構築する。
8. 編集Tool固有stateが無くても同じ結果へ到達できることを確認する。

Structured Proposalへの事前変換は要求しません。

### 最初の変更候補

新しい機能実装ではなく、既存Reference Workspaceの小さいfile-native semantic changeを優先します。

候補条件は次です。

- 変更前後のCanonical meaningが明確に比較できる。
- Source-to-Canonical traceを確認できる。
- Validation / Conformanceが存在する。
- 外部ServiceやProduction Resourceを必要としない。
- rollbackが通常のGit操作で説明できる。
- AI固有metadataを追加しなくてよい。

ISSUE-000027のShared Acceptance Referenceで既に扱っているValidationやRelationの小変更は候補ですが、本IssueではStructured Proposal経路そのものを再実証することが目的ではありません。

## 2026-10-08 E1 executable candidateの強化

`tools/reference-tool-neutral-development/run.sh`を、E1のExecutable Reference candidateとして強化しました。

初期案は`reference/workspace`だけを一時Git repositoryへ複製していましたが、これは「標準的なTEI Repository checkoutで開発する」というRQ-034を十分にpressureしていませんでした。現在は**current TEI Repository全体をlocal clone**し、そのclone内でbuild / test / CLI実行 / file edit / commitを行います。

```text
clean current TEI checkout
  -> local task clone
  -> build cmd/tei
  -> targeted test
  -> knowledge-context / validate / world
  -> direct file edit
  -> validate / world / targeted test
  -> ordinary Git commit
  -> separate integration clone
  -> rebuild / test / reload
```

task側では`semantic/sales/validation.tei`へ`sales.order_line.unit_price_positive`をfile editで直接追加します。変更前に`workspace-files`でこのfileがmanaged sourceであることを確認し、既存の`sales.order_line.quantity_positive`が`source-index`からこのfileへtraceできることを要求します。変更後も追加Validationの`source-index`が同じauthoring sourceへ戻り、`knowledge-context`から新しい意味へ到達できることを確認します。integration cloneでも同じSource traceを再構築します。

build結果はRepository外の一時directoryへ置きます。関連するknowledge-context testもtask cloneとintegration cloneの双方で実行します。file edit後のGit diffが対象1 file以外へ広がる場合、またbuild / test / CLI利用によってRepository内へuntracked artifactが生じる場合は失敗します。

変更は通常のGit commitへ確定します。integration側はtask repositoryを複製せず、元のsource checkoutから別directoryへlocal cloneし、task result commitを通常Gitのfetch / `--ff-only` mergeで統合します。integration cloneがtaskと同じbase revisionから開始し、統合後HEADがtask result revisionと一致することを検査したうえで、`cmd/tei`をbuildし、targeted test、`tei validate`、`knowledge-context`、`tei world`を再実行します。task側とintegration側のValidation出力とCanonical Worldがそれぞれbyte単位で一致し、read-only操作後もintegration cloneがcleanであることを要求します。

### Decision ledger非依存のpressure

current Reference Workspaceには`.tei/decisions`が存在します。初期runnerはこれをそのまま複製していたため、「Decision ledgerのようなCanonical meaningの第二正本ではないauthoring stateへ依存しない」という主張を実証できていませんでした。

現在のrunnerではtask cloneから別の一時Workspaceを作り、そこだけ`.tei/decisions`を削除します。その状態で`validate`と`world`を再実行し、Decision ledgerありのWorkspaceに対して、Validation出力とCanonical Worldのそれぞれがbyte単位で一致することを要求します。

`.tei/plugin-lock.json`は削除しません。runnerは元WorkspaceにPlugin Lockが存在することを要求し、Decision ledgerを削除した一時Workspaceでもそのfileがbyte単位で同一であることを検査します。Plugin resolutionを固定するlockと、Canonical meaningの第二正本ではないDecision ledgerを同じstateとして扱わないためです。

このcandidateはStructured Proposal、Shared Acceptance candidate schema、TEI独自Change Set、Agent Session / Agent Workspace / Checkpointを使用しません。Shared Acceptanceを否定するものではなく、Proposalを使わないfile-native / Git経路も第一級であることをpressureします。

### runner orchestrationの合成検証

actual TEI checkoutを取得できない制約とは別に、runner自体のGit / Shell orchestrationを検証するため、local container上に最小の合成Go Repositoryを作成しました。

合成Repositoryは、runnerが利用する`workspace-files`、`source-index`、`knowledge-context`、`validate`、`world`だけを最小実装し、同じ名前のtargeted testとReference Workspaceを持ちます。Semantic conflictについては、二つのfileに`ToolNeutralCollision`が存在すると`DUPLICATE_SEMANTIC_ELEMENT`を返すようにしました。

この合成環境で、最新版runnerは次を最後まで通過しました。

```text
full local clone
  -> build
  -> targeted test
  -> managed source / source trace check
  -> Decision ledger removal equivalence
  -> direct file edit
  -> ordinary commit
  -> independent base integration clone
  -> task commit fetch / fast-forward integration
  -> rebuild / retest / reload
  -> two branch Git merge
  -> semantic conflict rejection
  -> clean post-state
```

runner出力:

```text
integration=<same revision as result>
changed=reference/workspace/semantic/sales/validation.tei
validation-ledger-independent=true
decision-ledger-independent=true
plugin-lock-retained=true
semantic-conflict=DUPLICATE_SEMANTIC_ELEMENT
tool-neutral development reference: PASS
```

最新のstate境界改善後は、GitHub上のrunner blob SHA `0acf8651a21433b431b13716df379e0c64d9ce63`とsynthetic fixture側のGit blob SHAが完全一致することを確認したうえで再実行しました。task clone / integration clone / conflict cloneがsource HEADから始まること、integration cloneが元baseからtask result commitを通常Gitでfetch / fast-forwardして同じrevisionへ到達すること、Validation出力とCanonical Worldの再現、Decision ledger非依存、Plugin Lock保持、Semantic conflict拒否を含め、runnerは最後までPASSしました。

これは**runner orchestrationの検証**であってTEI Product Evidenceではありません。合成CLIがactual TEI semanticsを実装しているわけではないため、RQ-034のEvidence tierは上げません。

### qualification wrapperの合成検証

`qualify.sh`は、actual TEI checkoutとは切り離した最小Git repositoryでも制御フローを実行確認しました。

合成repositoryには、clean stateを確認してPASSする最小`repository-health-check.sh`と`run.sh`を置き、local bare originへ`main`をpushして`origin/main`を作りました。

確認結果:

- exact `--expected-head` / `--expected-main`が一致し、expected mainをcandidateが含む正常系はPASSした
- 出力にcandidate head、origin/main、`candidate-contains-expected-main=true`、repository-health PASS、Reference PASSが含まれる
- 誤った`--expected-head`はexit 1でfail closedした
- dirty pre-stateもexit 1でfail closedした
- `origin/main == --expected-main`でも、そのmainがcandidate HEADのancestorでないsibling historyはexit 1でfail closedした
- current `qualify.sh`は`bash -n`を通過した

これはqualification wrapperの制御ロジックに対するsynthetic Evidenceです。actual TEIのRepository health、CLI、Canonical meaning、Semantic Validationが成立したことは示しません。

### 現在の検証状態

- full local clone方式へ書き直したrunnerは`bash -n`で構文確認済み。
- `qualify.sh`のcurrent contentも`bash -n`で構文確認済み。
- E2で期待する`DUPLICATE_SEMANTIC_ELEMENT`がcurrent Validation実装に存在することはsourceから確認済み。
- current execution environmentにはTEI Repositoryのlocal checkoutが無く、GitHubからcloneしようとすると`github.com`のDNS解決に失敗する。
- したがって、actual `go build` / `go test` / CLI round trip / Git integration / Repository healthのPASSはまだ主張しない。
- GitHub Actionsは使用していない。
- actual executionが成立するまでは本Issueを完了扱いにせず、RQ-034もDesign Onlyを維持する。
- runner自身はremote Gitを要求しないが、Go module cacheが空の環境では通常のGo buildと同様にdependency取得が必要になる可能性がある。offline hermetic buildはRQ-034の成立条件にはしない。

実行入口:

```bash
bash tools/reference-tool-neutral-development/run.sh
```

## E2 — Git integration boundary

変更元と統合側を別contextとして扱います。

```text
base revision R0
  -> task branch / commit R1
  -> integration context
  -> resulting revision R2
  -> reload / validation
```

base / head / integration revisionはGitから取得します。Agent Session IDやTEI独自Task IDを必須にしません。

### Git conflictとSemantic conflictを分ける

Git conflictはGitに任せます。

そのうえで、Git上ではmerge可能でもTEIの意味上は成立しない変更を一つpressureします。

候補例:

- 別fileから同じlogical identityへ互換性のない変更を加える。
- BindingとCapability requirementを別々に変更し、file-level conflictなしで不整合を作る。
- Data / Relation変更とApplication側の参照変更を別branchで行い、片方だけ統合するとValidationが失敗する。

このnegative caseで必要なのは「TEI独自merge engine」ではなく、通常のGit統合後にTEIの既存Validation / Conformanceが意味上の問題を検出できることです。

## 2026-10-07 E2 semantic conflict candidate

同じrunnerへ、Gitのfile-level mergeとTEIのSemantic validationを分離するnegative caseを追加しました。

一つのbaselineから二つのbranchを作り、それぞれ別fileへ同じCanonical Data identity `sales.tool_neutral_collision` を追加します。

```text
branch A
  semantic/sales/tool-neutral-collision-a.tei
        \
         +-- git merge succeeds
        /
branch B
  semantic/sales/tool-neutral-collision-b.tei

merged workspace
  -> tei validate
  -> DUPLICATE_SEMANTIC_ELEMENT
```

各branch単独では`tei validate`を通すことを要求します。二つのfile pathは異なるため、Gitのtext mergeではconflictしません。統合後だけ既存`validation.Validate`が同じCanonical IDの重複を検出し、`DUPLICATE_SEMANTIC_ELEMENT`で拒否することを期待します。

このcaseの目的はTEI独自merge engineを作ることではありません。Gitが所有するtext / history integrationを尊重したまま、その後のSemantic boundaryで意味上の不整合を止められることを確認します。

E2のstatic auditでは、current `loadSemantics`が複数fileの同一namespaceを一つのSemantic Namespaceへ統合し、Data定義をappendしてからValidationへ渡すことを確認しました。そのため、別fileに同じ`sales.tool_neutral_collision`を置くcaseはloader段階のnamespace errorではなく、`internal/validation.Validate`の`DUPLICATE_SEMANTIC_ELEMENT`まで到達する設計です。actual TEI実行は、current execution environmentのGitHub DNS制約により未確認です。

## E3 — MCP利用とCLI parity

RQ-034は、TEI自身のdeveloper MCP serverを必須にはしません。

チャット実行環境がMCPを利用する場合、最初の正規経路は次で十分です。

```text
Chat / Agent
  -> Host-provided Shell / Filesystem / Git MCP
  -> tei CLI
  -> existing workspace / knowledgecontext / validation implementation
```

つまり、「MCPを使うこと」と「TEIがMCP serverを実装すること」を分けます。

### E3-A — Host-mediated CLI

最初に確認するのは、汎用MCPやComputer / Shell capabilityから通常の`tei` CLIを呼ぶだけで、E1 / E2と同じ開発loopを実行できることです。

この経路ではTEI側に新しいtransport、Session model、Adapter stateを追加しません。MCPはHost capabilityであり、TEIのCanonical modelには入りません。

### E3-B — optional native MCP projection

将来、structured request / response、discoverability、latency、typed schema等に明確なProduct valueがある場合だけ、TEI固有のdeveloper MCP projectionを検討します。

current CLI実装を読むと、主要処理は次へ既に分かれています。

- `workspace.Load`
- `workspace.BuildSourceIndex`
- `knowledgecontext.Build`
- `knowledgecontext.QualifyWithSources`
- `validation.Validate`
- `workspace.ProjectDiagnostics`

`cmd/tei`側は、これらを組み合わせ、引数検査や出力整形を行う薄い境界です。

そのためnative MCPを追加する場合も、MCP専用parser、MCP専用Knowledge Store、MCP専用Validationを作りません。必要ならCLIとMCPが共有するapplication-level operationを抽出しますが、second frontendが本当に必要になる前から新しいservice layerを先行実装しません。

```text
                       -> CLI formatter / exit code
workspace + operation -|
                       -> optional MCP transport
```

移行初期にMCP側がCLI subprocessを呼ぶ薄いbridgeを使うことも禁止しません。長期的に共有serviceへ切り出すかは、実測したlatency、error projection、typed transportの必要性から判断します。

### Authority

Host-mediated MCPでもnative MCPでも、MCP経由であることは次の根拠にはなりません。

- Semantic authority
- Acceptance authority
- Runtime authority
- Resource authority

developer MCPが存在しない環境でもCLI-only pathが成立することを維持します。

## E4 — chat-hosted / coding-agent consumer

E1〜E3のRepository contractを、実際の外部開発環境から利用できることを確認します。

候補:

- ChatGPT等のチャット実行環境
- Codex
- Claude Code
- human + IDE / terminal
- CI

個別製品のSession model、prompt schema、memory形式等をTEI Product Requirementへ取り込みません。

consumerごとの差は、利用可能なfilesystem、shell、Git、network、MCP等のHost capabilityです。TEIのCanonical meaningはconsumerによって変えません。

## Authority境界

### 開発環境のauthority

独立開発環境が自分のcheckout / worktreeへ書き込めることは、通常の開発権限です。

```text
local filesystem / Git authority
  !=
TEI Application Runtime authority
```

Repository内のBuild、Test、Driver、Validation等を実行することも、それだけでProduction相当Resourceへのauthorityを与えません。

実Database、Secret、外部Service、Production相当Resource、その他のside effectへ到達する場合はRQ-018〜RQ-020を維持します。

### AuthoringとCanonical

file-native sourceを直接編集できることと、その内容が自動的にAccepted Canonicalになることも同じではありません。

Repository policyや対象変更に応じて、Git review、Shared Acceptance、Validation、Conformance等の既存境界を利用します。

このIssueは「全file editをShared Acceptance Toolへ通す」ことを要求しません。入口によってCanonical解釈が変わらず、正式統合後のSourceから再構築できることを要求します。

## 保存しない状態

RQ-034のためだけに次をTEIのProduct stateへ追加しません。

- Agent Session
- Agent Workspace
- Checkpoint
- Chain-of-thought / internal plan
- tool trajectory
- Test実行回数
- 一般的な `verified / unverified`
- CI job state
- editor selection / cursor / temporary layout
- Chat transcript

これらに独立したProduct上の必要性が後から見つかった場合は、その必要性自体を別に評価します。

## 2026-10-08〜09 未資格統合の記録

PR #967は2026-10-08T06:06:01Zに、head `425b5743fab37ec087892d67d7bbf09e6496bacd`からmerge commit `6307950c53f0c7879d9231c5077e2410cff71e9a`としてmainへ統合されました。

しかし、統合時点のPR本文は、exact headに対するactual `tools/repository-health-check.sh`とRQ-034 ReferenceのPASSが未取得であり、RQ-034をDesign Only、PRをDraftのまま保持する、と明記していました。Repositoryの既存開発ルールも、code変更を含むDraft PRについてexact-head full Repository Healthとclean pre/post stateが無い状態でのmergeを認めていませんでした。

したがって、**#967がmerge済みであること自体をqualification Evidenceとして扱いません。** synthetic runner PASS、shell構文確認、GitHub上のmergeable / merged状態も代替Evidenceにはしません。この記録のために共有済みGit履歴を巻き戻さず、forward-onlyで扱います。

さらに、#967の是正用Draft PR #1027も、actual qualification receipt未取得を本文に明記したまま2026-10-09T06:50:05Zにmainへ統合されました。merge commitは`474c1b545267afad99b8320d676b6f02251809fb`、messageは`Merge remote-tracking branch 'origin/work/rq-034-post-merge-qualification'`です。これは通常Git merge経由の統合であり、Draft flagだけでは防止できないことを示します。#1027のmergeもqualification Evidenceには数えません。

現在のforward stopは、trustedなactual TEI full checkoutで、その時点のcurrent main exact HEADを固定し、次を同じcheckoutで確認することです。

1. tracked / untracked / ignored artifactに加え、`assume-unchanged` / `skip-worktree`等のtracked内容を隠せるindex flagも無いclean pre-state。
2. full `tools/repository-health-check.sh` PASS。
3. RQ-034 Executable Reference PASS。
4. 実行中にHEADと対象mainが変わっていないこと。
5. tracked / untracked / ignored artifactとtracked内容を隠せるindex flagの双方が無いclean post-state。

current execution environmentではactual TEI checkoutを実行できないため、このforward stopは未完了です。RQ-034はDesign Only、本IssueはIn Progressのままとします。2026-10-09の是正作業はmain `60b8de4718c495dfbcbb1f20b98427f399d6598c`を起点に開始しましたが、このSHAは固定の完了条件ではありません。actual qualification時にmainが進んでいれば、新しいcurrent mainから最初から取り直します。

また、qualification wrapperとReference runnerのclean判定は、通常の`git status`だけでなく`git ls-files --others --ignored --exclude-standard`も確認するように強化します。これにより、ignored build artifactが残ったcheckoutからPASS receiptを出しません。この強化自体も、actual Repository Health / RQ-034 PASSの代替には数えません。

## Merge qualificationとの関係

本IssueのExecutable ReferenceがPASSしても、それだけでcode-changing PRをmainへ統合してよいとは判断しません。

Repository全体のGateは`AGENTS.md` / `CONTRIBUTING.md`を正本とし、exact candidate headに対するfull `tools/repository-health-check.sh` PASSとclean pre/post stateを別に要求します。

`tools/reference-tool-neutral-development/qualify.sh`は、そのfull health Gateと本Referenceを同じclean checkoutで順番に実行し、途中でHEADやworking treeが変わったらfail closedする補助です。

```text
candidate exact head
  -> full Repository health
  -> still same clean head
  -> RQ-034 executable reference
  -> still same clean head
  -> qualification output
```

これはTEIへ新しい`verified`状態を追加する仕組みではありません。Gate実行時の一時的なEvidenceを取り違えにくくするためのRepository Toolです。

#1027の未資格統合では、文書上の停止条件が存在していても通常Git mergeでmainへ入れられることが実際に起きました。このため`qualify.sh --receipt-file`は、成功時にRepository外へmachine-readable receiptを出せるようにします。receiptはcandidate HEAD / tree、expected main、clean pre/post、Repository Health / RQ-034 Reference PASS、およびhealth / qualify / Reference / integration-preflight runnerのGit blob SHAを固定します。既存receiptは上書きせず、同じdirectory内の一時fileからhard-linkで原子的に公開します。

統合直前は`integration-preflight.sh`をread-onlyで実行します。preflight自身がcandidateで初めて追加・変更される場合にも自己適用できるよう、**exact candidateのclean checkoutからcandidate版preflightを実行し、`--integration-root`でexact expected mainの別clean checkoutを指定します。** candidate checkoutとintegration checkoutを同一にせず、実行中のpreflight file自身もcandidate上のblobおよびreceiptへ結び付けます。integration checkout HEADとlocal `origin/main`、candidate HEAD / ref / tree、4つのrunner blob、双方のclean stateが一致し続けることを確認します。さらに`git ls-remote`で指定remoteのlive `main`を開始時と終了時に確認し、remote-tracking refが古いままでもfail closedします。main移動、candidate変更、receipt書換え、runner変更、preflight自身の差替えでは失敗します。receiptは開始時と終了時のblobも比較し、preflight実行中の差替えも拒否します。このpreflight自体はfetch / merge / commit / push / ref更新を行わず、remoteにはread-onlyで問い合わせます。

preflightがPASSした後は、branch名を再解決せず、preflightが出力したexact `candidate-head` SHAをそのまま通常Gitの統合対象へ引き継ぎます。これによりpreflight後にbranch refが進んでも別commitを誤統合しません。

receiptは署名済みAuthorityやTEI Product stateではありません。通常Gitを使うHuman / AI間で、別revisionのPASSをmerge authorityとして誤流用しないためのhandoff Evidenceです。強いwitnessが必要な環境では別Providerを重ねられますが、RQ-034の通常開発経路に必須化しません。

`--expected-main`がqualification時に固定するのは、事前取得済みのlocal `origin/main`です。wrapperはそのSHA一致だけでなく、expected mainがcandidate HEADのancestorであることも確認します。統合直前のpreflightではlocal refに加えてlive remote `main`も直接確認します。ただしremoteをlockするものではないため、preflight後にmainが進んだ場合はnon-force pushのnon-fast-forward拒否を安全網とし、最新mainをcandidateへ取り込んでqualificationからやり直します。

### qualification receipt / integration preflightの合成検証

actual TEI checkoutを使えない制約とは別に、receipt handoff自体のGit mechanicsをlocal synthetic Repositoryで実行しました。synthetic Repositoryには最小のhealth runnerとReference runnerを置き、candidateをcurrent mainから分岐させ、`qualify.sh`と同じreceipt field / runner blob binding、`integration-preflight.sh`と同じ検査条件を使っています。

確認結果:

```text
normal=PASS
tamper=PASS
runner_mutation=PASS
main_movement=PASS
bootstrap_separate_integration=PASS
preflight_self_mutation=PASS
```

- 正常系: exact candidate / mainからreceiptを作成し、integration checkoutをexact mainへ置いたpreflightがPASS。
- receipt改ざん: `qualification=PASS`を書き換えたreceiptを拒否。
- candidate runner変更: receipt取得後にReference runnerを変更したcandidateをrunner blob不一致で拒否。
- main移動: receipt取得後にlocal `origin/main`またはlive remote `main`が進んだ状態をstaleとして拒否。
- bootstrap: candidate版preflightをexact candidate checkoutから実行し、別のexact-main integration checkoutを検査できる。
- preflight自己変更: receipt取得後に実行中preflightのbytesを変えた場合、candidate blobとの不一致で拒否。

また、detached HEADのsource checkoutから`git clone --local --no-hardlinks`した場合、cloneも同じdetached commitから始まることをlocal Gitで確認しました。source checkoutにignored / untracked artifactが存在してもclone側へは複製されないため、source側のignored-clean検査は省略できません。

追加のindex-state pressureでは、`git status --porcelain`が空のままになる`assume-unchanged`と`skip-worktree`をそれぞれ設定し、`git ls-files -v`由来のfail-closed検査が両方を検出することを確認しました。

```text
assume_unchanged_detected=PASS
skip_worktree_detected=PASS
normal_index=PASS
```

これは「working treeで実行した内容」と「receiptが指すcommit内容」がずれる経路を閉じるためのRepository mechanicsです。sparse checkout等で`skip-worktree`が必要なcheckoutはRQ-034 qualificationには使わず、full clean checkoutへ切り替えます。

これはqualification toolingのmechanics Evidenceであり、actual TEIのRepository Health、CLI、Canonical meaning、RQ-034 Reference成立Evidenceではありません。

## Evidence

最初のReferenceでは少なくとも次を残します。

- base Git revision
- clean pre-state
- 利用したCLI command
- 対象Source identity / locator
- 変更したfile
- Git diff
- Build / Validation / Test commandとactual result
- result commit / revision
- clean post-state
- integration context
- integration後のreload / Validation result
- 削除対象とした非正本state（最初のReferenceではDecision ledger）に依存せず再構築できたこと
- Semantic conflict negative case
- 使用した開発環境の制約。ただしCanonical identityには含めない

このEvidenceを、TEI独自の永続Task stateへ変換することは要求しません。Reference EvidenceとしてRepositoryに残すか、通常のGit / test outputへ結び付けます。

## 完了条件

- [ ] clean checkout / task-local worktreeだけで対象TEI Sourceを理解できる
- [ ] 専用Agent Adapterなしにfile-native sourceを編集できる
- [ ] CLI / Repository Toolから必要なBuild / Validation / Testを完結できる
- [ ] Git commit / branch等を第一級の変更成果物として利用できる
- [ ] Structured Proposal / TEI独自Change Setを必須にしない
- [ ] 別contextから変更後SourceをreloadしCanonical meaningを再構築できる
- [ ] Agent Session / Workspace / Checkpoint / Plan / 一般的な検証状態をTEI Product stateへ追加しない
- [ ] Decision ledger等、明示的に非正本と分類したstateを除去してもauthoritative sourceとCanonical meaningが変わらない
- [ ] Git上はmerge可能だがSemantic上は不整合となるnegative caseを少なくとも一つ再現できる
- [ ] CLI-only pathがdeveloper MCPなしでも成立する
- [ ] MCPを追加する場合、CLIと同じapplication service / semantic contractを利用できる
- [ ] CLI / MCPの入口差が意味・Authority差にならない
- [ ] 開発環境のShell / Filesystem / Git authorityとApplication Runtime authorityを混同しない
- [ ] Referenceで見つかった不足を通常CLI / application serviceのGapとして説明できる
- [ ] Agent専用subsystemを追加しなくても一往復が成立する
- [ ] exact candidate headがcurrent mainを含む状態で、full `tools/repository-health-check.sh`と本Referenceをclean pre/post state付きでPASSできる
- [ ] qualification receiptとread-only integration preflightで、exact candidate HEAD / tree、expected main、live remote main、runner blobの一致を再確認してから通常Git統合へ進める

## 非目標

- Devin相当のAgent Runtime
- Agent Session / Checkpoint管理
- chat-first IDE
- Gitの置換
- 全変更へのTEI独自Change Set強制
- MCP serverをTEI開発の必須入口にすること
- CI状態のTEI内複製
- 開発ToolのShell権限をTEI Runtime Capabilityへ変換すること
- Tool trajectoryの監査基盤
- 特定のAI provider / modelの標準化

## 関連文書

- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `requirements/README.md`
- `plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md`
- `plan/adr/0031-tool-neutral-git-native-development.md`
- `plan/issue/000027-ai-proposal-acceptance-boundary.md`
- `plan/issue/000046-studio-shared-acceptance-authoring-loop.md`
- `plan/issue/000047-knowledge-first-semantic-exploration.md`
- `plan/issue/000079-plugin-self-build.md`
- `plan/issue/000080-extensible-plugin-capability-contract.md`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md`
- `docs/architecture.md`
