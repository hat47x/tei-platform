# reference-tool-neutral-development

RQ-034 / ISSUE-000083（GitHub #972）の最初のExecutable Referenceです。

目的は、TEI専用Agent Runtime、Agent Adapter、Structured Proposal、TEI独自Change Set、Agent Session / Workspace / Checkpoint等を使わずに、通常のRepository開発だけで次の一往復が成立するかを確認することです。

```text
clean Git repository
  -> TEI CLIでcurrent meaningを読む
  -> file-native sourceを直接編集
  -> validate / Canonical reload
  -> ordinary Git commit
  -> separate clone
  -> validate / Canonical reload
```

## Scenario

runnerはcurrent `reference/workspace`を一時directoryへ複製し、その一時directoryだけを独立Git repositoryにします。mainのReference Workspaceは変更しません。

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

runnerは次を検査します。

- baseline Workspaceが`tei validate`を通る
- 既存Validationへ`knowledge-context`から到達できる
- 直接file edit後も`tei validate`を通る
- 追加したValidationへ`knowledge-context`から到達できる
- Canonical Worldがbaselineから実際に変わる
- commit前の変更fileが意図した1 fileだけである
- TEI CLIの利用によってAgent stateや別のuntracked fileが生成されない
- ordinary Git commitに変更を確定できる
- 別cloneで同じValidationへ到達できる
- 別cloneのCanonical Worldがtask側とbyte単位で一致する
- read-only reload後も別cloneがcleanである

## Run

```bash
bash tools/reference-tool-neutral-development/run.sh
```

このrunnerはlocal Git、Go toolchain、Repository内の通常CLIだけを使います。network、GitHub Actions、developer-facing MCP serverを必要としません。

MCP parityはISSUE-000083 E3で後続検証します。MCPを追加する場合も、CLIと別のCanonical解釈やAuthority modelを作りません。

## Non-claims

このReferenceだけでは次を主張しません。

- ChatGPT / Codex / Claude Code固有integrationが完成した
- developer-facing MCP serverが完成した
- すべてのTEI変更が直接file editだけで十分である
- Shared Acceptance / Semantic Diff / Reviewが不要である
- local Git write authorityがApplication Runtime authorityを与える
- production-grade CI / sandboxが完成した

このReferenceが示すのは、**Tool固有stateをTEIへ持ち込まなくても、SourceとGitを境界にして開発環境を交換できるか**という最小の一往復です。
