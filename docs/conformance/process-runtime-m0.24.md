# M0.24 別プロセス型プラグイン実行の準拠確認

M0.24では、`BoundaryProcess`をインターフェース上の概念にとどめず、リポジトリからOSの子プロセスを実際に起動できるところまで接続します。

この準拠確認の対象は、**TEIのホストから子プロセスを呼び出し、JSONの応答を受け取る**最小限の信頼境界です。本番運用向けの隔離環境、高可用性のプロセス管理、別ホストからの再接続まで実装したことを意味しません。

## 実行境界

`StdioProcessLauncher`は、`ProcessLauncher`の参照実装です。

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

`PluginRuntime.Command`は、この段階では単一の実行ファイル名またはパスを表します。シェルには渡さず、文字列を空白で分割して引数を推測することもしません。

したがって、次のようなシェル固有の解釈を、TEIの実行時の契約へ暗黙に持ち込みません。

- pipe / redirect
- variable expansion
- glob
- quote interpretation
- `sh -c` / `cmd /c`相当の解釈

プラグインの起動に引数が必要になった場合は、シェルの構文解析を追加するのではなく、定義情報に引数を明示する構造を設ける方針です。

## 要求と応答の識別

子プロセスには、既存の`runtime.Request`をJSON形式で渡します。

応答を受け取った後の識別情報の検査には、登録管理側の既存の契約を使います。

- protocol version
- request ID
- Plugin identity
- Capability ID
- descriptor fingerprint
- response status

参照実装の結合テストでは、実際の子プロセスが偽のプラグイン識別情報を返した場合、登録管理側が拒否することを確かめます。

## プロセスの異常終了

参照用の起動機構では、少なくとも次の状態を区別します。

- executable start / exit failure
- context cancellation
- malformed JSON response
- 複数JSON valueを返すprotocol violation
- response size超過

処理の取り消しには`exec.CommandContext`を使い、直接起動した子プロセスが終了してから呼び出し元へ戻ります。

ただし、子プロセスがさらに起動したプロセス群を、すべてのOSで完全に終了させる本番用の監督機構までは、この参照実装では保証しません。

## 標準出力の境界

外部プロセスは、ホストから見れば信頼できない実行主体です。標準出力を無制限にメモリへ取り込むことはしません。

`StdioProcessLauncher.MaxResponseBytes`でJSON応答のサイズを制限します。値が0の場合も参照実装の既定値である8 MiBを適用し、上限を超える応答は解析前に拒否します。

これは本番環境の通信規約における最終的な上限を定めるものではありません。参照用のホストが無制限の外部出力を受け入れないことを確かめるものです。

## 標準エラー出力と秘密情報

別プロセスで動くプラグインは、秘密情報を渡す仕組みなどを通じて、ホストの外にある秘密情報の実際の値にアクセスする可能性があります。その値が誤って標準エラー出力に書かれることを想定します。

そのため、参照用の起動機構は標準エラー出力の本文をTEIのエラー文面へコピーしません。

異常終了時には、診断のため次だけを返します。

- stderr byte count
- stderr SHA-256

成功時のstderrもResponse protocolへ混ぜません。

これは外部プロセス自身のログを安全にする仕組みではありません。ホスト側で、エラーや来歴の記録に内容を不用意に転記しないための境界です。

## 隔離環境との関係

`StdioProcessLauncher`は`ProcessLauncher`と`SandboxedProcessRunner`の両方を実装しますが、**それ自体が隔離環境になるわけではありません**。

`InvokeSandboxed`は、プロセス間通信の仕組みを再利用するだけです。ネットワークやファイルシステムへのアクセスを既定で拒否する方針、秘密情報の受け渡しについては、既存の`HostSandboxLauncher`がホスト側の強制機構を適用した後で呼び出す必要があります。

したがって、

- `sandbox required`なのに`SandboxLauncher`がない場合は従来どおり拒否
- plain `Registry.Processes`経由の実行を`sandboxed=true`とは記録しない
- `StdioProcessLauncher`の存在だけでOS sandbox実装済みとは主張しない

という境界を維持します。

## 資源の貸出しとの順序

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

つまり、成果物を検証できない場合や必要な隔離環境がない場合に、秘密情報・ネットワーク・ファイルシステムの利用権を先に渡すことはしません。

## 参照用の配備実行器

`tools/reference-process-plugin`は、標準入出力で通信する規約を実装した最小限の実行用プログラムです。

`examples/process-plugin`は、このプログラムを`StdioProcessLauncher`から起動します。既存の`DeploymentExecutor`接続部を通じて、提供元に依存しない`executor.Command`を渡します。

参照用のプロセスが返す`accepted`は、実行器が要求を受け付けたことを示すだけで、外部の状態が意図どおりになった証明ではありません。最終的な完了状態の確認は、従来どおり観測・検証根拠・整合確認の仕組みが担います。

この例ではさらに`ValidateDeploymentInvocation`を呼び出し、プロセスの境界とプラグインの来歴が環境の計画と整合することを確かめます。

## 永続化する実行識別子との関係

M0.24で実証するのは、ローカルで一度だけ起動する子プロセスです。

OSのプロセス識別子を`DurableInvocationHandle.RuntimeHandle`として自動採用しません。プロセス識別子は再利用されることがあり、別のホストから同じ実行状態を安全に確認できる汎用的な識別情報ではないためです。

そのため、この準拠確認は次の機能の成立を意味しません。

- Host restart後のPID reattach
- cross-host process takeover
- long-lived process manager durability
- process lifecycleのHA保証

実行状態へ永続的に再接続するには、M0.22とADR-0005で定めた、内部構造を公開しない永続的な実行識別情報と、権限を伴う再接続の契約を満たす外部実行環境の接続部が必要です。

プロセスの呼び出し中にホストが失われ、外部への変更が起きたか判断できない場合でも、見つからない状態や不明な状態から自動的に再実行しない原則を守ります。

## リポジトリ内での検証

単体テストと結合テストでは、実際のテスト用実行ファイルを子プロセスとして起動し、次を確認します。

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

M0.24では、次の機能を実装済みとは扱いません。

- production-grade OS sandbox
- seccomp / namespace / Windows Job Object等のgeneric abstraction
- Kubernetes / containerd等のdurable runtime integration
- cross-host reattach
- long-lived plugin daemon protocol
- streaming RPC
- process pool
- arbitrary shell command execution
- plugin SDKの最終public API

M0.24の目的は、別プロセスでの実行を抽象的なインターフェースだけで終わらせず、実際のOSプロセスでも、TEIの信頼・識別・来歴の境界が守られることを確かめることです。
