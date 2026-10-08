# ADR-0031: 標準開発環境とGit-native authoringを第一級の開発経路として扱う

- 状態: 採用
- 日付: 2026-10-08
- 関連RQ: RQ-015 / RQ-017 / RQ-021 / RQ-027 / RQ-034
- 関連ADR: ADR-0022
- 関連Issue: ISSUE-000027 / ISSUE-000046 / ISSUE-000047 / ISSUE-000079 / ISSUE-000083

## 背景

TEIは、生成AIを開発支援へ利用できることを重視しています。一方、生成AIごとに専用のSession、Adapter、Workspace管理をTEIへ持ち込むと、一般的なRepository開発から外れ、将来のTool交代にも弱くなります。

特に、ChatGPT等のチャット実行環境、Codex、Claude Code、通常のIDE / CLI、CIは、それぞれ独立した開発Hostになり得ます。これらがTEIを扱うたびにTEI専用Agent Runtimeや中央Adapterを必要とすると、既存のGit-nativeな開発が二級経路になってしまいます。

また、ADR-0022ではAI ProposalとCanonical受入境界を分離し、Structured Proposalを使う経路を採用しています。この判断は維持します。ただし、それを「すべてのAI編集やfile editは、編集前にStructured Proposalへ変換しなければならない」という一般則には広げません。

## 防ぎたい失敗

防ぎたいのは、AI対応を理由に開発基盤を囲い込むことです。

次の状態は避けます。

- TEIを編集するために特定のChat UIやAgent製品が必須になる
- CodexやClaude Code等の通常のRepository開発がTEI専用Adapterなしでは成立しない
- すべての変更をTEI独自Proposal / Change Setへ変換しないとGitへ反映できない
- Agent Session、Workspace、Checkpoint、Plan、Test実行状態をTEIのProduct stateへ複製する
- GitやCIがすでに所有している変更管理・実行状態をTEIが第二の管理系として持つ
- Chat側でMCPを利用するという理由だけで、TEI自身のdeveloper MCP serverが必須になる
- 開発HostのShell / Filesystem / Git権限を、TEI Application RuntimeのAuthorityとして扱う

## 守る原則

TEIの標準開発経路は、特定のAI製品やAgent frameworkではなく、RepositoryとSourceを中心にします。

```text
Human / Chat / Codex / Claude Code / IDE / CI
                    |
          development host
                    |
      repository / worktree / branch
                    |
              file-native edit
                    |
        build / test / validate
                    |
                  Git
                    |
        integration / acceptance
                    |
          authoritative source
                    |
                  reload
                    v
          Canonical TEI Meaning
```

開発Toolは交換可能であり、変更後のauthoritative sourceから元のChatやAgentなしにCanonical meaningを再構築できなければなりません。

## 判断

### Git-nativeな開発を第一級の経路とする

TEI専用Agent Runtimeや常駐Adapterを使わず、通常のRepository clone、branch、worktree、file edit、Git commit、Pull Request等でTEIを改修できることを標準要件とします。

Gitで十分に表現できる変更へTEI独自Change Setを必須にしません。

Structured ProposalやSemantic Diffを利用する経路は、意味差分・根拠・Preview・Review等に追加価値がある場合の正規経路として維持します。

したがって、標準Architectureは次の二経路を許容します。

```text
A. Structured authoring

AI / Editor / Tool
  -> Proposal / Structured Intent
  -> Preview / Shared Acceptance
  -> authoritative source

B. Git-native development

Human / Chat / Coding Agent / IDE
  -> isolated clone / branch / worktree
  -> direct file edit
  -> build / test / validate
  -> ordinary Git integration
  -> authoritative source
```

両経路はauthoritative sourceへ統合された後、同じloader、Validation、Conformance、Source-to-Canonical Traceabilityへ戻ります。

### Shared Acceptanceを中央Serviceと同一視しない

Shared Acceptance Boundaryは、すべての編集操作を一つのTEI Serviceへ逐次送ることを意味しません。

Structured Proposalを利用する変更ではShared Acceptance Toolを利用できます。一方、通常のGit-native開発ではRepository policy、Pull Request、local merge等でauthoritative sourceへ統合できます。

重要なのは、入口ごとにCanonical解釈やValidation規則を変えないことです。

### Git conflictとSemantic conflictを分ける

file-levelの競合やhistory integrationはGitに任せます。

TEIは独自のGit merge engineを標準要件にはしません。

ただし、Git上では無競合でmergeできても、同じCanonical identityの重複やBinding不整合等の意味上の矛盾が起こることがあります。その場合は、統合後のTEI Validation / Conformanceでfail closedできることを要求します。

```text
Git merge succeeds
        !=
TEI meaning is valid
```

### CLIを標準のdeveloper interfaceとする

TEI CLIは、少なくともSource、Canonical meaning、Source Index、Knowledge Context、Validation、Conformance等へ到達するための標準的な開発interfaceとして扱います。

Chat側がMCPを利用する場合も、Hostが提供するShell / Filesystem / Git MCPから通常の`tei` CLIを呼べれば正規経路として成立します。

TEI自身のdeveloper MCP serverは必須にしません。

将来native developer MCPを追加する場合も、CLIと別のCanonical解釈、Knowledge Store、Validation、Authority modelを作りません。必要なら既存のapplication operationを別transportから利用します。

### Development Hostの能力とTEI Runtime Authorityを分ける

同じHost内で改修からBuild / Testまで完結するには、Host側に少なくとも次が必要です。

```text
repository materialization
+ filesystem read / write
+ process execution
+ Git operations
+ required toolchain
```

これはTEI Application RuntimeのCapability modelではなく、開発Hostの環境条件です。

開発HostがRepositoryへ書き込めることから、実Database、Secret、外部Service、Production相当Resource等へのRuntime Authorityを推論してはいけません。

### 開発Toolの一般状態をTEIへ複製しない

RQ-034のためだけに次をTEIのProduct stateへ追加しません。

- Agent Session
- Agent Workspace
- Checkpoint
- internal Plan
- tool trajectory
- Chat transcript
- Test実行回数
- CI job state
- 一般的な `verified / unverified`

Git、CI、開発Tool等がすでに所有する状態は、その仕組みに残します。

TEIへ永続化するのは、Canonical meaning、Source、Decision、Authority、Conformance等としてTEI自身に独立したProduct上の必要性がある情報に限ります。

## ADR-0022との関係

ADR-0022は、Structured Proposalを利用するAI authoringで、ProposalとCanonical authorityを分離する判断として維持します。

本ADRはADR-0022を置き換えません。適用範囲を次のように分けます。

```text
ADR-0022
  Structured Proposal / Shared Acceptance path

ADR-0031
  Tool-neutral / Git-native development path
  + both paths converge after source integration
```

Structured Proposalを利用する場合はADR-0022のstale detection、Preview / Reject / Accept、authority分離等を適用します。

Proposalを使わない通常のGit-native開発については、本ADRに従います。

## 具体例

### Chat実行環境が完全な開発Hostを持つ場合

Chat側でRepositoryをcheckoutでき、shell、Filesystem、Git、Go toolchainを使えるなら、その環境内でTEI CLIを使って改修、Build、Testまで完結できます。

TEI専用Agent SessionやAdapterを追加する必要はありません。

### Chat実行環境がRepositoryを書き換えられるがprocessを実行できない場合

その環境でSource変更やGit差分を作ることはできますが、full Build / Test loopは完結しません。

その変更をGit等へ渡し、process executionを持つ別の標準開発環境で継続します。

この違いをTEIへ`partial`等の永続状態として記録することは要求しません。

### Codex / Claude Code / Human local development

通常のRepository clone / worktreeを利用し、

```text
inspect
-> edit
-> build
-> test
-> git commit / PR
```

という既存の開発プロセスをそのまま利用できます。

TEI対応Adapterが導入されていなくても正規経路です。

## 代替案

### すべての変更をStructured Proposalへ変換する

Semantic DiffやReviewを統一しやすい利点はあります。

しかし、通常のfile-native / Git開発へ独自中間形式を強制し、CodexやClaude Code等の標準的な開発processを不必要に囲い込むため採用しません。

### TEIがAgent Session / Workspaceを中央管理する

長時間Agentのresumeや監査には便利です。

しかし、Agent製品ごとの実行modelをTEIのProduct modelへ持ち込み、Git / CI / Tool stateと重複しやすいため標準経路には採用しません。

必要な環境ではpluginや外部serviceとして追加できます。

### TEI native MCPを必須入口にする

typedなoperationやdiscoverabilityには価値があります。

ただし、Shell / Filesystem / Git MCPからCLIを呼ぶだけでも同じ意味操作へ到達できるため、native MCPを標準開発の前提にはしません。

## 影響

良くなる点は、現在広く使われるRepository開発とTEIが連続し、AI Toolを交換しても開発方法が崩れにくくなることです。

また、Agent execution stateとTEIのCanonical stateを分離するため、TEIが変更管理やCIの第二正本になることを避けられます。

一方で、すべての変更を一つのProposal lifecycleへ押し込まないため、Git-native経路ではreviewやconflict解消をRepository policyへ委ねる必要があります。TEIはその代わり、Source-to-Canonical Traceability、Validation、Conformanceを統合後にも再実行できなければなりません。

## 検証

ISSUE-000083 / GitHub #972で、少なくとも次をExecutable Referenceとして確認します。

- clean Repository cloneからCLIをbuildできる
- managed sourceをCLIから特定できる
- Source Indexから変更対象へ追跡できる
- file-native sourceを直接編集できる
- Build / Test / Validationを実行できる
- ordinary Git commitへ変更を確定できる
- 別のintegration cloneから同じCanonical meaningを再構築できる
- Decision ledger等のsupporting stateを除去してもCanonical meaningが変わらない
- Git上はmerge可能でもSemantic上不正な変更をValidationが拒否できる
- Agent Adapter、TEI独自Change Set、native developer MCPなしでも一往復が成立する

## 未解決 / 非主張範囲

このADRは、次を保証しません。

- すべてのChat製品がRepository checkoutやprocess executionを持つ
- offline hermetic buildが完成している
- native developer MCPが不要であり続ける
- すべての変更が直接file editだけで十分である
- Shared Acceptance / Structured Proposalが不要である
- GitだけでSemantic conflictを検出できる
- local Git write authorityがProduction Runtime authorityを与える

native MCP、Agent Adapter、remote execution service等に追加価値が見つかった場合も、本ADRの「標準Git-native経路を必須依存にしない」という境界を維持した上で追加します。

## 関連文書

- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `docs/architecture.md`
- `plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md`
- `plan/issue/000027-ai-proposal-acceptance-boundary.md`
- `plan/issue/000083-tool-neutral-development-reference.md`
