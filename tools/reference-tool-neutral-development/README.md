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
  -> original baseから作ったseparate integration clone
  -> task commitを通常Gitでfetch / fast-forward integration
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

runnerはtask cloneから別の一時Workspaceを作り、そこだけ`.tei/decisions`を削除します。その状態で`validate`と`world`を再実行し、Decision ledgerを持つWorkspaceに対して、Validation出力とCanonical Worldのそれぞれがbyte単位で一致することを要求します。

`.tei/plugin-lock.json`はこのpressureでは削除しません。runnerは元WorkspaceにPlugin Lockが存在することと、一時Workspaceへのcopy後もbyte単位で同一であることを確認します。Plugin resolutionを固定するlockと、Canonical meaningの第二正本ではないDecision ledgerを同じ「Tool state」として扱わないためです。

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
- `.tei/plugin-lock.json`は存在し、Decision ledger除去pressureの前後でbyte単位に保持される
- 直接file edit後も`tei validate`を通る
- 追加したValidationも`source-index`から同じauthoring sourceへtraceできる
- 追加したValidationへ`knowledge-context`から到達できる
- Canonical Worldがbaselineから実際に変わる
- file edit後にも関連targeted testを実行できる
- commit前の変更fileが意図した1 fileだけである
- CLI / build / testの利用によってRepository内へ別のuntracked artifactが生成されない
- ordinary Git commitに変更を確定できる
- integration側は元のsource checkoutから作った別local cloneであり、task repository自体を複製しない
- integration cloneがtaskと同じbase revisionから始まり、task result commitを通常Gitのfetch / fast-forwardで統合する
- integration後のHEADがtask result revisionと一致する
- integration clone内でもCLIをbuildし、関連targeted testを実行できる
- integration cloneでも追加ValidationのSource Indexが同じauthoring sourceへ戻る
- integration cloneで同じValidationへ到達できる
- integration cloneのValidation出力とCanonical Worldがtask側とそれぞれbyte単位で一致する
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

Repositoryの開発ルールでは、code変更を含むcandidateについて、**そのexact headに対するfull `tools/repository-health-check.sh` PASSとclean pre/post state**が必要です。ここでcleanとは、tracked変更とuntracked fileが無いだけでなく、`.gitignore`等で無視されるartifactも残っていない状態を指します。

その取り違えを減らすため、薄いwrapperを用意しています。

```bash
tools/reference-tool-neutral-development/qualify.sh \
  --expected-head <candidate-sha> \
  --expected-main <current-main-sha>
```

このwrapperは次の順でfail closedします。

```text
tracked / untracked / ignored clean pre-state
  -> exact candidate HEAD
  -> optional origin/main binding
  -> expected main is an ancestor of candidate HEAD
  -> full repository-health-check.sh
  -> HEAD / tracked / untracked / ignored clean-state再確認
  -> run.sh
  -> HEAD / tracked / untracked / ignored clean-state再確認
  -> candidate / main bindingを含むPASS receiptをstdoutへ表示
```

`--expected-main`を使う場合は、呼び出し前に通常のpreflightでremoteを再取得し、`origin/main`がcurrent mainであることを確認します。さらに、そのexpected mainがcandidate HEADのancestorであることも要求します。これにより、local `origin/main`のSHAだけは正しくてもcandidate branchがcurrent mainをまだ取り込んでいない状態をqualification PASSへ進めません。wrapper自身はnetwork accessやref更新を行いません。

`--expected-main`はlocal `origin/main`を固定し、同時にそのcommitがcandidate historyへ含まれることを検査します。ただしremote mainをlockしません。qualification後にmainが動いた場合、そのPASSを新しいmainへのmerge authorityとして流用せず、remote stateを再取得し、candidateへ最新mainを統合してから再qualificationします。

wrapper自身がpre/postのtracked / untracked / ignored stateを確認するため、呼び出し側の事前checkだけにclean判定を依存しません。

PASS時には`candidate-head`、`origin-main`、`candidate-contains-expected-main=true`、Repository health / Reference / qualification結果をstdoutへ出します。これは永続的なTEI Product stateではなく、その時点のexact Git relationとGate結果を示すreceiptです。必要なGate handoffで、通常のIssue / PR Evidenceとして記録できます。

### machine-readable qualification receipt

通常Git mergeによる統合でもGateを取り違えないため、`--receipt-file`を指定すると、すべてのGateが成功した後だけmachine-readable receiptをRepository外へ出力します。

```bash
CANDIDATE_HEAD="$(git rev-parse HEAD)"
TESTED_MAIN="$(git rev-parse origin/main)"
RECEIPT="$(mktemp)"

tools/reference-tool-neutral-development/qualify.sh \
  --expected-head "$CANDIDATE_HEAD" \
  --expected-main "$TESTED_MAIN" \
  --receipt-file "$RECEIPT"
```

receiptは少なくとも次を固定します。

- exact candidate HEAD
- exact `origin/main` / expected main
- candidateがexpected mainを含むこと
- tracked / untracked / ignored clean pre/post
- full Repository Health PASS
- RQ-034 Reference PASS
- `repository-health-check.sh` / `qualify.sh` / `run.sh` / `integration-preflight.sh` のcandidate上のGit blob SHA
- Git / Go / Python version

receiptはRepository内へ書けません。receipt出力自体によってqualification checkoutをdirtyにしないためです。

### integration preflight

統合直前には、**資格確認済みcandidateのclean checkout**と、receiptが固定したexact mainをHEADに持つ**別のclean integration checkout**を用意します。preflight自身がcandidateで初めて追加・変更される場合でも自己適用できるよう、preflightはcandidate側のfileを実行し、`--integration-root`で統合側checkoutを明示します。

```bash
CANDIDATE_CHECKOUT=/path/to/qualified-candidate
INTEGRATION_CHECKOUT=/path/to/clean-integration-main

git -C "$INTEGRATION_CHECKOUT" fetch --prune origin main
git -C "$INTEGRATION_CHECKOUT" switch --detach "$TESTED_MAIN"

"$CANDIDATE_CHECKOUT/tools/reference-tool-neutral-development/integration-preflight.sh" \
  --receipt-file "$RECEIPT" \
  --candidate "$CANDIDATE_HEAD" \
  --expected-main "$TESTED_MAIN" \
  --integration-root "$INTEGRATION_CHECKOUT"
```

preflightはmerge / commit / push / ref更新を行いません。次をfail closedで照合します。

- integration checkoutのHEAD == expected main
- local `origin/main` == expected main
- candidate HEAD == receiptのcandidate
- candidateがexpected mainを含む
- receiptの全PASS条件
- receiptに記録された4つのrunner blob == candidate上の現在のblob
- 実行中の`integration-preflight.sh`自身 == candidate上のpreflight blob
- candidate checkoutとintegration checkoutの双方がtracked / untracked / ignored clean
- preflightの開始から終了までcandidate HEAD / candidate ref / integration HEAD / `origin/main`が不変

mainがreceipt取得後に動いた場合、candidateやrunnerが変わった場合、preflight自身がreceipt取得時と異なる場合、またreceiptを書き換えた場合は失敗します。その場合は新しいmainをcandidateへ取り込み、**新しいexact candidateでqualificationから取り直します**。

このreceiptは暗号学的な署名やGitHub branch protectionの代替ではありません。人間・AIが通常Gitで統合するときに、別revisionの成功記録を誤って流用しないための機械的なhandoffです。TEI Product state、Agent Session、独自Change Setへ昇格させません。

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

このReferenceが示すのは、**Agent / Acceptance固有の非正本stateをCanonical meaningの第二正本にせず、SourceとGitを境界にして開発環境を交換できるか**という最小の一往復です。現時点で除去して実証する対象はDecision ledgerであり、Plugin Lock等の別責務のstateまで不要だとは主張しません。
