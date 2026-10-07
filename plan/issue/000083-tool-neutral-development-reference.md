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

## 2026-10-07 E1 executable candidate

`tools/reference-tool-neutral-development/run.sh`を、E1の最初のExecutable Reference candidateとして追加しました。

runnerはmainの`reference/workspace`を直接変更しません。一時directoryへ複製し、その複製だけを独立Git repositoryとして初期化します。baselineをcommitした後、`semantic/sales/validation.tei`へ`sales.order_line.unit_price_positive`をfile editで直接追加します。

変更前には既存の`sales.order_line.quantity_positive`へ`knowledge-context`から到達し、変更後には追加したValidationへ同じCLIから到達します。変更前後のCanonical Worldが同じなら失敗し、commit前のdiffが対象1 file以外へ広がっても失敗します。

変更は通常のGit commitへ確定し、その一時repositoryを別directoryへlocal cloneします。integration側で`tei validate`、`knowledge-context`、`tei world`を再実行し、task側とintegration側のCanonical Worldがbyte単位で一致すること、read-only操作後にintegration cloneがcleanであることを確認します。

このcandidateはStructured Proposal、Shared Acceptance candidate schema、TEI独自Change Set、Agent Session / Workspace / Checkpointを使用しません。Shared Acceptanceの価値を否定するものではなく、Proposalを使わないfile-native / Git経路も第一級であることだけをpressureします。

### 現在の検証状態

- runnerのBash syntaxは`bash -n`で確認済み。
- この実行環境からGitHubをcloneしてactual Repositoryで実行しようとしたが、`github.com`のDNS解決に失敗した。
- したがって、`go run ./cmd/tei`、actual Git round trip、Repository healthのPASSはまだ主張しない。
- GitHub Actionsは使用していない。
- actual executionが成立するまでは本Issueを完了扱いにせず、RQ-034もDesign Onlyを維持する。

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
