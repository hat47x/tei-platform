# ADR-0006: process commandをexecutableとargvへ分離する

- 状態: 採用
- 日付: 2026-08-31
- 関連Issue: #33
- 前提: M0.24実process Plugin runtime

## 背景

M0.24では、`BoundaryProcess`をactual OS child processへ接続するReference `StdioProcessLauncher`を追加しました。このlauncherはsecurity boundaryを曖昧にしないため、Plugin descriptorのcommandをshellへ渡さず、単一のexecutable名またはpathとして`exec.CommandContext`へ渡します。

しかし、M0.24以前のPlugin authoring grammarは、`runtime process "..."`のquoted string全体を`Runtime.Command`へ保存していました。そのため、たとえば次のdescriptorはparser上は有効です。

```tei
runtime process "python3 -m worker" sandbox required
```

一方、M0.24のruntime contractではこの文字列全体を一つのexecutable名として扱うため、通常は実行できません。

この差を埋める方法として、command文字列を空白でsplitしたり、`sh -c`等へ渡したりすることもできます。しかし、その場合はquote、escape、pipeline、redirect、environment expansion、platform固有shell semanticsが暗黙にTEIのruntime contractへ入り込みます。Plugin descriptorから実行同一性を追跡する上でも、文字列の再解釈をHostごとに変えられる状態は避けるべきです。

## 判断

process Pluginの実行指定を、**単一executableと順序付きargvの組**として扱います。

概念上は次です。

```text
Process Command
  executable: string
  args: []string
```

既存の`Runtime.Command` fieldは互換性を保つため名称を維持しますが、その意味を「shell command line」ではなく**単一executable名/path**へ明確化します。引数は`Runtime.Args []string`として別に保持します。

Authoring grammarは次の形を採用します。

```tei
runtime process "python3" args "-m" "worker" sandbox required
```

引数が不要な既存descriptorはそのまま有効です。

```tei
runtime process "atlas" sandbox required
```

## shell semanticsを導入しない

TEI Core / Reference Hostは、次を暗黙には行いません。

- command文字列の空白split
- shell quote / escapeの再解釈
- glob展開
- pipeline / redirect
- command substitution
- environment variable expansion
- `sh -c`、`cmd.exe /c`、PowerShell等への自動委譲

必要な引数はdescriptor上で明示します。

この判断は「shellを禁止する」という意味ではありません。shell自体を実行したいPluginは、shell executableとそのargvを明示できます。たとえばLinux hostで意図的にshell scriptを使うなら、概念上は`/bin/sh`と`-c`およびscript本文を別argsとして指定します。その場合、shell semanticsを選んだこと自体がdescriptorへ明示され、fingerprintにも入ります。

## legacyな空白入りcommand

`runtime process "python3 -m worker"`のような空白入りcommandを、互換性の名目で自動splitしません。

これは一見便利ですが、`"a b"`が一つのpathなのか、`a`と`b`なのかをTEIが推測することになるためです。

M0.25では、process commandに空白が含まれる場合は明示的なparse / validation errorとし、`args`構文への移行を要求します。

既存Reference Workspaceの`"atlas"`等、空白を含まないcommandは変更不要です。

## execution identity

argvは実行内容を変え得るため、単なるHost-local optionではありません。

`Args`はPlugin Definitionの一部として扱い、既存のDefinition fingerprint、Plugin Lock fingerprint、Runtime Policy fingerprintへ自然に含めます。したがって、同じexecutableでもargsが異なれば異なる実行同一性として検出されます。

例:

```text
python3 -m worker-a
python3 -m worker-b
```

は同じprocess runtimeとはみなしません。

## Host/runtime境界

Host adapterへ渡す際も、executableとargvを一度shell-like stringへ連結しません。

```text
Plugin descriptor
  ↓
Runtime{Command, Args}
  ↓
Environment Plan / PluginRuntime
  ↓
ProcessCommand{Executable, Args}
  ↓
ProcessLauncher / SandboxedProcessRunner
  ↓
exec.CommandContext(ctx, executable, args...)
```

Sandbox境界でも同じ構造を維持します。Sandbox policyやResource Leaseとは別の次元ですが、どのprogramをどのargvで起動したかはRuntime Policy fingerprintの対象です。

## 採用しなかった案

### `Command string`をshell command lineとして維持する

記述は短いですが、platformやshellによって解釈が変わり、Hostが同一descriptorを異なるprocessへ解決し得るため採用しません。

### Host側で`strings.Fields`する

quoteやescapeを表せず、空白を含むpathやargumentを壊します。見かけ上の互換性のために曖昧な意味を導入するので採用しません。

### 常に`sh -c`へ渡す

Unix shellをTEI標準runtimeへ固定し、Windows等との意味差も大きくなるため採用しません。またshell injection surfaceを不要に広げます。

### `Command`を直ちに`Executable`へrenameする

意味としては明瞭ですが、M0.24直後の段階でJSON shapeや既存コードへの変更量を増やす効果が大きくありません。M0.25では`Command`名を維持し、契約上の意味を単一executableへ固定します。将来public API compatibilityを設計する段階でrenameの価値を再評価できます。

## 影響

- Plugin descriptorが実process executionへ直接対応するようになる
- shell semanticsが暗黙のHost依存にならない
- args差分がPlugin Lock / Runtime Policy差分として検出される
- `ProcessLauncher` / `SandboxedProcessRunner` interfaceは構造化commandを受け取る形へ更新される
- Reference process exampleはargvが実際のchild processまで保存されることをConformanceで確認する

この変更はproduction-grade process isolation、PATH resolution policy、cross-host reattachを導入するものではありません。M0.24で分離したplain process / sandbox / durable runtimeの境界は維持します。
