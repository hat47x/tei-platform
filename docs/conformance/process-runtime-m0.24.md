# Conformance — M0.24 実process Plugin runtime

M0.24では、`BoundaryProcess`をinterface上の概念だけでなく、Repository内で実際のOS child processとして実行できるところまで縦に接続します。

このConformanceが確認するのは、**TEI Host → child process → JSON response**という最小のprocess trust boundaryです。production-grade sandbox、HA process manager、cross-host reattachまでを実装済みとは主張しません。

## 実行境界

`StdioProcessLauncher`は`ProcessLauncher`のReference実装です。

```text
Registry
  ↓ ProcessLauncher.Invoke
StdioProcessLauncher
  ↓ exec.CommandContext
actual child process
  stdin  ← TEI Request JSON
  stdout → TEI Response JSON
  stderr → content is not copied into TEI error text
```

`PluginRuntime.Command`は現段階では一つのexecutable名またはpathとして扱います。shellへ渡さず、空白分割してargvを推測しません。

したがって、次のようなshell semanticsをTEI runtime contractへ暗黙に持ち込みません。

- pipe / redirect
- variable expansion
- glob
- quote interpretation
- `sh -c` / `cmd /c`相当の解釈

将来plugin commandに引数が必要な場合は、文字列のshell parseを追加するのではなく、descriptor側へ明示的なargv構造を導入することを基本方針とします。

## Request / Response identity

child processには既存の`runtime.Request`をJSONで渡します。

Responseを受け取った後のidentity validationはRegistry側の既存契約を維持します。

- protocol version
- request ID
- Plugin identity
- Capability ID
- descriptor fingerprint
- response status

Reference integration testでは、実processが偽のPlugin identityを返した場合にRegistryが拒否することを確認します。

## process failure

Reference launcherは少なくとも次を区別します。

- executable start / exit failure
- context cancellation
- malformed JSON response
- 複数JSON valueを返すprotocol violation
- response size超過

context cancellationには`exec.CommandContext`を使い、直接起動したchild processの終了を待ってから返ります。

ただし、childがさらに作成した孫process treeを全OSで完全に掃除するproduction process supervisorまでを、このReference adapterは保証しません。

## stdout boundary

external processはHostから見ればuntrustedです。stdoutを無制限にメモリへ保持しません。

`StdioProcessLauncher.MaxResponseBytes`でResponse JSONサイズを制限し、0の場合もReference defaultの8 MiBを使います。上限超過はResponse decode前に明示的に拒否します。

これは最終的なproduction protocol上限を固定するものではなく、Reference Hostがunbounded external outputをそのまま受け入れないことをConformance化するものです。

## stderrと秘密値

process Pluginは、Secret Injector等を通じてHost外の具体的なsecret materialへ到達する可能性があります。そのprocessが誤ってsecretをstderrへ書く可能性を前提にします。

そのためReference launcherはstderr本文をTEI error textへコピーしません。

異常終了時には、診断のため次だけを返します。

- stderr byte count
- stderr SHA-256

成功時のstderrもResponse protocolへ混ぜません。

これは外部process自身のログ出力を安全化する仕組みではありません。Host側がTEI error / provenanceへ不用意に再転記しない境界です。

## Sandboxとの関係

`StdioProcessLauncher`は`ProcessLauncher`と`SandboxedProcessRunner`の双方を実装しますが、**それ自体はsandboxではありません**。

`InvokeSandboxed`はprocess transportを再利用するだけです。実際のdeny-by-default network / filesystem policyやsecret injectionは、既存`HostSandboxLauncher`がHost-provided enforcerを適用した後で呼び出す必要があります。

したがって、

- `sandbox required`なのに`SandboxLauncher`がない場合は従来どおり拒否
- plain `Registry.Processes`経由の実行を`sandboxed=true`とは記録しない
- `StdioProcessLauncher`の存在だけでOS sandbox実装済みとは主張しない

という境界を維持します。

## Resource Leaseとの順序

Registryの既存順序は変更しません。

```text
resolve capability
  ↓
permission check
  ↓
artifact verification
  ↓
boundary / sandbox availability preflight
  ↓
resource lease acquisition
  ↓
process invocation
  ↓
lease / ownership cleanup
```

つまり、verification不成立やrequired sandbox不在のprocessへ、先にsecret/network/filesystem leaseを渡す構造にはしません。

## Reference DeploymentExecutor

`tools/reference-process-plugin`はstdio protocolを実装する最小の実process helperです。

`examples/process-plugin`はこのhelperを`StdioProcessLauncher`経由で起動し、既存`DeploymentExecutor` adapterを通してprovider-neutralな`executor.Command`を渡します。

Reference processが返す`accepted`はexecutor acknowledgementであり、外部stateが意図どおり完成した証明ではありません。既存どおり、durable completionはObservation / Evidence / Reconcile側の責務です。

例ではさらに`ValidateDeploymentInvocation`を呼び、process boundaryとPlugin provenanceがEnvironment Planと整合することを確認します。

## Durable Invocation Handleとの関係

このM0.24 vertical sliceはone-shot local child processです。

OS PIDを`DurableInvocationHandle.RuntimeHandle`へ自動的に昇格させません。PIDは再利用され得て、別Hostが同一runtime identityを安全に証明する一般的なhandleではないためです。

したがって、このConformanceは次を意味しません。

- Host restart後のPID reattach
- cross-host process takeover
- long-lived process manager durability
- process lifecycleのHA保証

Durable reattachには、M0.22 / ADR-0005で定めたopaque durable runtime handleとauthority-bearing reattach契約を満たす外部runtime adapterが必要です。

process invocation中にHostが失われ、副作用の有無を確定できない場合も、missing / unknownから自動replayしない原則を弱めません。

## Repositoryでの検証

単体・integration testでは実際のtest binaryをchild processとして起動し、次を確認します。

- valid Request / Response round trip
- successful stderrがprotocolへ混ざらない
- malformed response拒否
- oversized stdout拒否
- non-zero exit時にstderr本文をHost errorへコピーしない
- context cancellation
- forged response identityをRegistryが拒否

常設CIではさらに、

1. `tools/reference-process-plugin`を実binaryへbuild
2. `examples/process-plugin`を起動
3. DeploymentExecutor経由で実processへcommandを渡す
4. `accepted` acknowledgementを取得
5. provenance上の`boundary=process`を確認

という経路を毎回実行します。

## 非目標

M0.24では次を実装済みとはしません。

- production-grade OS sandbox
- seccomp / namespace / Windows Job Object等のgeneric abstraction
- Kubernetes / containerd等のdurable runtime integration
- cross-host reattach
- long-lived plugin daemon protocol
- streaming RPC
- process pool
- arbitrary shell command execution
- plugin SDKの最終public API

M0.24の目的は、processを抽象interfaceの向こう側に置いたままにせず、TEIの既存trust / identity / provenance境界を実際のOS processでも壊さないことを確認することです。
