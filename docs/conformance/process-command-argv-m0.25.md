# Conformance — M0.25 process executable / argv

M0.25では、process Pluginの実行指定をshell-likeな1本のcommand lineではなく、単一executableと順序付きargvへ分離します。

この文書は、ADR-0006の判断がauthoringからactual child processまでどこで固定されているかを整理します。

## 不変条件

process runtimeは概念上、次の組として扱います。

```text
executable: string
args: []string
```

TEI Core / Reference Hostは、command文字列に対して空白split、shell quote再解釈、glob展開、environment expansion、`sh -c`等への暗黙委譲を行いません。

`Runtime.Command`というfield名は互換性のため維持しますが、意味は単一executable名/pathです。引数は`Runtime.Args`に順序付きで保持します。

## Authoring grammar

明示的なargsを次のように記述できます。

```tei
runtime process "python3" args "-m" "worker module" "--flag=a b" sandbox required
```

空白を含む各quoted argumentは一つのargv elementのまま保持されます。

旧来の次の記述を、自動splitして互換化しません。

```tei
runtime process "python3 -m worker" sandbox required
```

この形は明示argsへの移行を要求するerrorになります。

主な検証は次のとおりです。

- `internal/syntax/process_args_conformance_test.go`
- `internal/syntax/plugin_parser_test.go`

## Execution identity

argvは実行内容を変えるため、Host-local optionではなくPlugin execution identityの一部です。

同じexecutableでもargsが異なれば、次が変化します。

- Plugin Definition fingerprint
- Environment Plugin Lock fingerprint
- Runtime Policy fingerprint

主な検証は次のとおりです。

- `internal/plugin/process_args_conformance_test.go`
- `internal/plugin/runtime/process_args_policy_test.go`

これにより、descriptor上では同じPlugin versionなのにHost側だけ別argvを使う、といった差異をsilentに扱いません。

## Runtime境界

`PluginRuntime`は`Command`と`Args`を保持し、Process / Sandbox境界では`ProcessCommand{Executable, Args}`へ構造化したまま渡します。

Reference `StdioProcessLauncher`は次の形でchild processを起動します。

```text
exec.CommandContext(ctx, executable, args...)
```

executableとargsを一度command line文字列へ連結しません。

この変更はM0.24で固定した次の境界を変更しません。

- plain process transportはOS sandboxではない
- `sandbox required`はHost Sandbox / policy enforcerを必要とする
- local child PIDをDurable Runtime Handleとはみなさない
- missing / unknown時のno-auto-replay原則を弱めない

## Actual child processでのargv確認

`tools/reference-process-plugin`は、Reference marker

```text
--tei-reference-marker=argv value preserved
```

が**空白を含む一つのargv element**として届いた場合だけ成功Responseを返します。

`examples/process-plugin`は同じmarkerを`PluginRuntime.Args`に設定し、次の縦経路を通します。

```text
DeploymentExecutor
  → Registry
  → PluginRuntime{Command, Args}
  → ProcessCommand{Executable, Args}
  → StdioProcessLauncher
  → actual OS child process
  → os.Args
  → JSON Response
```

常設CIではReference helperをbuildし、この例を実際に実行して`accepted` responseとargv markerを確認します。

## Reference fixture

Reference Workspace内で実際にargsを持つprocess descriptorを一つ保持します。WorkspaceからPlugin Lock / World / Durable Runを再生成し、一致することをCIで確認します。

したがって、argv導入は単体testだけでなく、Workspace authoring、descriptor fingerprint、Plugin Lock、normalized World、Durable execution fixtureまで一続きで再現されます。

## 非目標

M0.25は次を導入しません。

- shell pipeline / redirect
- shell variable expansion
- command substitution
- working directory / environment authoringの一般化
- production-grade PATH resolution policy
- OSごとのshell abstraction
- production-grade sandbox
- cross-host process reattach

必要なshell semanticsがある場合、shell executableとそのargvをdescriptorへ明示し、その選択自体をexecution identityとして扱います。
