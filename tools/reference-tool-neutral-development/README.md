# reference-tool-neutral-development

RQ-034 / ISSUE-000083（GitHub #972）の最初のExecutable Referenceです。

目的は、TEI専用Agent Runtime、Agent Adapter、Structured Proposal、TEI独自Change Set、Agent Session / Workspace / Checkpoint等を使わずに、通常のRepository開発だけで次の一往復が成立するかを確認することです。

```text
clean TEI Repository checkout
  -> local task clone
  -> TEI CLIでcurrent meaningを読む
  -> file-native sourceを直接編集
  -> build / targeted test / validate / Canonical reload
  -> ordinary Git commit
  -> separate integration clone
  -> rebuild / test / validate / Canonical reload
```

## Scenario

runnerは、**現在のTEI Repository checkout全体**を`git clone --local`で一時directoryへ複製します。元Repositoryがdirtyなら開始せず、mainや現在のworktreeを変更しません。

task cloneでは`cmd/tei`をbuildし、既存のknowledge-context testを実行したうえで、Reference WorkspaceをCLIから読みます。

baselineには既存の次のruleがあります。

```tei
validation sales.order_line.quantity_positive positive {
    attribute sales.order_line.quantity
}
```

task側では、同じ既知のValidation意味を使って次をfileへ直接追加します。

```tei
validation sales.order_line.unit_price_positive positive {
    attribute sales.order_line.unit_price
}
```

この変更はShared Acceptanceのcandidate形式を経由しません。目的はProposal経路を否定することではなく、Proposalを使わない通常のfile-native / Git開発も第一級の経路として成立することを確認することです。

## Tool固有stateを前提にしないpressure

current Reference Workspaceには`.tei/decisions`が存在します。これはShared AcceptanceのDecision ledgerであり、Canonical meaningの第二正本ではありません。

runnerはtask cloneから別の一時Workspaceを作り、そこだけ`.tei/decisions`を削除します。その状態で`validate`と`world`を再実行し、Decision ledgerを持つWorkspaceとCanonical Worldがbyte単位で一致することを要求します。

`.tei/plugin-lock.json`はこのpressureでは削除しません。Plugin resolutionを固定するlockと、削除可能なDecision ledgerを同じ「Tool state」として扱わないためです。

## 検査内容

runnerは少なくとも次を検査します。

- 元RepositoryがGit checkoutでありcleanである
- task環境がcurrent Repository全体のlocal cloneである
- task clone内で`cmd/tei`をbuildできる
- knowledge-contextの関連targeted testを実行できる
- `workspace-files`が`semantic/sales/validation.tei`をmanaged sourceとして列挙する
- baseline Workspaceが`tei validate`を通る
- 既存Validationが`source-index`から`semantic/sales/validation.tei`へtraceできる
- 既存Validationへ`knowledge-context`から到達できる
- `.tei/decisions`を除去してもValidationとCanonical Worldが変わらない
- 直接file edit後も`tei validate`を通る
- 追加したValidationも`source-index`から同じauthoring sourceへtraceできる
- 追加したValidationへ`knowledge-context`から到達できる
- Canonical Worldがbaselineから実際に変わる
- file edit後にも関連targeted testを実行できる
- commit前の変更fileが意図した1 fileだけである
- CLI / build / testの利用によってRepository内へ別のuntracked artifactが生成されない
- ordinary Git commitに変更を確定できる
- integration側はtask repositoryから作った別local cloneである
- integration clone内でもCLIをbuildし、関連targeted testを実行できる
- integration cloneでも追加ValidationのSource Indexが同じauthoring sourceへ戻る
- integration cloneで同じValidationへ到達できる
- integration cloneのCanonical Worldがtask側とbyte単位で一致する
- read-only reload後もintegration cloneがcleanである

## Git mergeとSemantic conflict

別のnegative caseでは、current Repositoryから独立したcloneを作り、二つのbranchが別fileへ同じCanonical Data identityを追加します。

```text
branch A: semantic/sales/tool-neutral-collision-a.tei
        \
         +-- Git merge succeeds
        /
branch B: semantic/sales/tool-neutral-collision-b.tei

merged workspace
  -> tei validate
  -> DUPLICATE_SEMANTIC_ELEMENT
```

各branch単独ではValidationを通ることを要求します。二つのfile pathは異なるため、Gitのtext mergeではconflictしません。統合後だけ同じCanonical IDの重複を既存Semantic Validationが拒否することを確認します。

TEIがGit mergeを置き換えるのではなく、**Gitが正常に統合した後でも意味上の不整合をTEIが止められること**がこのpressureの目的です。

## Qualification

`run.sh`はRQ-034のExecutable Referenceです。ただし、code-changing PRをmainへ統合してよいというmerge authorityは、これだけでは得られません。

Repositoryの開発ルールでは、code変更を含むcandidateについて、**そのexact headに対するfull `tools/repository-health-check.sh` PASSとclean pre/post state**が必要です。

その取り違えを減らすため、薄いwrapperを用意しています。

```bash
tools/reference-tool-neutral-development/qualify.sh \
  --expected-head <candidate-sha> \
  --expected-main <current-main-sha>
```

このwrapperは次の順でfail closedします。

```text
clean pre-state
  -> exact candidate HEAD
  -> optional origin/main binding
  -> expected main is an ancestor of candidate HEAD
  -> full repository-health-check.sh
  -> HEAD / clean-state再確認
  -> run.sh
  -> HEAD / clean-state再確認
  -> PASS receiptをstdoutへ表示
```

`--expected-main`を使う場合は、呼び出し前に通常のpreflightでremoteを再取得し、`origin/main`がcurrent mainであることを確認します。さらに、そのexpected mainがcandidate HEADのancestorであることも要求します。これにより、local `origin/main`のSHAだけは正しくてもcandidate branchがcurrent mainをまだ取り込んでいない状態をqualification PASSへ進めません。wrapper自身はnetwork accessやref更新を行いません。

`--expected-main`はlocal `origin/main`を固定し、同時にそのcommitがcandidate historyへ含まれることを検査します。ただしremote mainをlockしません。qualification後にmainが動いた場合、そのPASSを新しいmainへのmerge authorityとして流用せず、remote stateを再取得し、candidateへ最新mainを統合してから再qualificationします。

出力は永続的なTEI Product stateではありません。必要なGate handoffで、通常のIssue / PR Evidenceとして記録できます。

## Run

```bash
bash tools/reference-tool-neutral-development/run.sh
```

runner自身はremote Git operation、GitHub Actions、developer-facing MCP serverを必要としません。Repository内のlocal Git clone、Go toolchain、通常CLIだけを利用します。

ただし、Go moduleがlocal module cacheに存在しない環境では、通常の`go build` / `go test`と同様にdependency取得が必要になる可能性があります。offline hermetic buildをRQ-034の成立条件とはしません。

MCP parityはISSUE-000083 E3で後続検証します。MCPを追加する場合も、CLIと別のCanonical解釈やAuthority modelを作りません。

## Non-claims

このReferenceだけでは次を主張しません。

- ChatGPT / Codex / Claude Code固有integrationが完成した
- developer-facing MCP serverが完成した
- すべてのTEI変更が直接file editだけで十分である
- Shared Acceptance / Semantic Diff / Reviewが不要である
- local Git write authorityがApplication Runtime authorityを与える
- production-grade CI / sandboxが完成した
- `run.sh`単体のPASSだけでcode-changing PRのmerge authorityが得られる
- offline hermetic Go buildが完成した
- TEIが独自のGit merge engineを必要とする
- Decision ledgerが不要である

このReferenceが示すのは、**Tool固有stateをCanonical meaningの成立条件へ持ち込まなくても、SourceとGitを境界にして開発環境を交換できるか**という最小の一往復です。
