# ISSUE-000083: Tool-neutral development Referenceをclean checkoutから実証する

- 状態: In Progress
- 起票日: 2026-10-07
- GitHub Issue: #972
- 関連RQ: RQ-015 / RQ-017 / RQ-021 / RQ-027 / RQ-034
- 関連ADR: ADR-0022
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

task側では`semantic/sales/validation.tei`へ`sales.order_line.unit_price_positive`をfile editで直接追加します。変更前には既存の`sales.order_line.quantity_positive`へ`knowledge-context`から到達し、変更後には追加したValidationへ同じCLIから到達します。

build結果はRepository外の一時directoryへ置きます。関連するknowledge-context testもtask cloneとintegration cloneの双方で実行します。file edit後のGit diffが対象1 file以外へ広がる場合、またbuild / test / CLI利用によってRepository内へuntracked artifactが生じる場合は失敗します。

変更は通常のGit commitへ確定し、そのtask repositoryをさらに別directoryへlocal cloneします。integration側でも`cmd/tei`をbuildし、targeted test、`tei validate`、`knowledge-context`、`tei world`を再実行します。task側とintegration側のCanonical Worldがbyte単位で一致し、read-only操作後もintegration cloneがcleanであることを要求します。

### Decision ledger非依存のpressure

current Reference Workspaceには`.tei/decisions`が存在します。初期runnerはこれをそのまま複製していたため、「Tool固有stateなしでもCanonical meaningを再構築できる」という主張が弱い状態でした。

現在のrunnerではtask cloneから別の一時Workspaceを作り、そこだけ`.tei/decisions`を削除します。その状態で`validate`と`world`を再実行し、Decision ledgerありのWorkspaceとCanonical Worldがbyte単位で一致することを要求します。

`.tei/plugin-lock.json`は削除しません。Plugin resolutionを固定するlockと、Canonical meaningの第二正本ではないDecision ledgerを同じstateとして扱わないためです。

このcandidateはStructured Proposal、Shared Acceptance candidate schema、TEI独自Change Set、Agent Session / Agent Workspace / Checkpointを使用しません。Shared Acceptanceを否定するものではなく、Proposalを使わないfile-native / Git経路も第一級であることをpressureします。

### 現在の検証状態

- full local clone方式へ書き直したrunnerは`bash -n`で構文確認済み。
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

E2追加後のrunnerも`bash -n`でBash syntaxを再確認しました。また、current `internal/validation.Validate`が同一Canonical IDを`DUPLICATE_SEMANTIC_ELEMENT`として拒否することをsourceから確認しています。actual TEI実行は、前節と同じくcurrent execution environmentのGitHub DNS制約により未確認です。

## E3 — MCP parity

CLI-only E2Eが成立した後で、developer-facing MCPがProduct valueを増す箇所を限定して検討します。

候補は、既存のapplication service / read model / validation operationをMCP transportから呼び出す薄いprojectionです。

```text
CLI ----┐
        ├-> same application service / semantic operation
MCP ----┘
```

MCP用にCanonical解釈、Validation、Knowledge qualification、Authority判定を再実装しません。

MCP経由であることは、次の根拠にはなりません。

- Semantic authority
- Acceptance authority
- Runtime authority
- Resource authority

MCP serverが無い環境でもCLI-only pathが成立することを維持します。

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
- Tool固有stateなしで再構築できたこと
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
- [ ] Tool固有stateを除去してもauthoritative sourceとCanonical meaningが変わらない
- [ ] Git上はmerge可能だがSemantic上は不整合となるnegative caseを少なくとも一つ再現できる
- [ ] CLI-only pathがdeveloper MCPなしでも成立する
- [ ] MCPを追加する場合、CLIと同じapplication service / semantic contractを利用できる
- [ ] CLI / MCPの入口差が意味・Authority差にならない
- [ ] 開発環境のShell / Filesystem / Git authorityとApplication Runtime authorityを混同しない
- [ ] Referenceで見つかった不足を通常CLI / application serviceのGapとして説明できる
- [ ] Agent専用subsystemを追加しなくても一往復が成立する

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
- `plan/issue/000027-ai-proposal-acceptance-boundary.md`
- `plan/issue/000046-studio-shared-acceptance-authoring-loop.md`
- `plan/issue/000047-knowledge-first-semantic-exploration.md`
- `plan/issue/000079-plugin-self-build.md`
- `plan/issue/000080-extensible-plugin-capability-contract.md`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md`
- `docs/architecture.md`
