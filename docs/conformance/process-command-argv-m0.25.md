# M0.25 別プロセスの実行形式と引数の準拠確認

M0.25では、別プロセス型プラグインの起動方法を、シェル風の一つの命令文字列ではなく、単一の実行ファイルと順序付きの引数に分けます。

この文書では、ADR-0006の設計判断が、定義の記述から実際の子プロセスの起動まで、どこで守られているかを整理します。

## 不変条件

別プロセスの実行指定は、次の組として扱います。

```text
executable: string
args: []string
```

TEIの中核部分と参照用ホストは、起動する命令の文字列を空白で分割したり、シェルの引用符・ワイルドカード・環境変数を解釈したり、`sh -c`などへ暗黙に処理を委ねたりしません。

互換性のため`Runtime.Command`という項目名は維持しますが、値は単一の実行ファイル名またはパスです。引数は`Runtime.Args`に順序を保って記録します。

## 記述文法

引数は、次のように明示して記述できます。

```tei
runtime process "python3" args "-m" "worker module" "--flag=a b" sandbox required
```

引用符で囲んだ引数は、空白を含んでいても一つの引数として保持されます。

従来の次の書き方を、空白で自動分割して互換扱いにはしません。

```tei
runtime process "python3 -m worker" sandbox required
```

この書き方はエラーとなり、引数を明示する形式への変更が必要です。

主な検証は次のとおりです。

- `internal/syntax/process_args_conformance_test.go`
- `internal/syntax/plugin_parser_test.go`

## 実行対象の識別

引数は実行内容を変えるため、ホストだけの設定ではなく、プラグインの実行対象を識別する情報の一部として扱います。

同じ実行ファイルでも、引数が異なれば次の照合値が変わります。

- Plugin Definition fingerprint
- Environment Plugin Lock fingerprint
- Runtime Policy fingerprint

主な検証は次のとおりです。

- `internal/plugin/process_args_conformance_test.go`
- `internal/plugin/runtime/process_args_policy_test.go`

これにより、定義情報上は同じプラグインの版なのに、ホスト側だけ別の引数を使うような違いを、検出せずに見過ごすことはありません。

## 実行時の境界

`PluginRuntime`は`Command`と`Args`を保持し、別プロセス実行や隔離実行の境界へは`ProcessCommand{Executable, Args}`として構造を保ったまま渡します。

参照実装の`StdioProcessLauncher`は、次の形で子プロセスを起動します。

```text
exec.CommandContext(ctx, executable, args...)
```

実行ファイル名と引数を、一つの命令文字列に連結することはしません。

この変更によって、M0.24で定めた次の境界は変わりません。

- plain process transportはOS sandboxではない
- `sandbox required`はHost Sandbox / policy enforcerを必要とする
- local child PIDをDurable Runtime Handleとはみなさない
- missing / unknown時のno-auto-replay原則を弱めない

## 実際の子プロセスで引数を確認する

`tools/reference-process-plugin`は、参照用の識別文字列

```text
--tei-reference-marker=argv value preserved
```

が**空白を含む一つの引数**として届いた場合にだけ、成功を示す応答を返します。

`examples/process-plugin`は同じ識別文字列を`PluginRuntime.Args`に設定し、次の一連の経路を通します。

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

常設の自動検査では参照用プログラムをビルドし、この例を実際に動かして、`accepted`の応答と引数の識別文字列を確認します。

## 参照用の試験データ

参照用の作業領域には、引数を持つ別プロセス型プラグインの定義を一つ置きます。そこからプラグインの固定情報、全体モデル、永続化する実行記録を再生成し、保存された内容と一致することを自動検査で確かめます。

つまり、引数の導入は単体テストだけでなく、作業領域での記述、定義情報の照合値、プラグインの固定情報、正規化された全体モデル、保存済みの実行記録までを通して再現できます。

## 非目標

M0.25では、次の機能を追加しません。

- shell pipeline / redirect
- shell variable expansion
- command substitution
- working directory / environment authoringの一般化
- production-grade PATH resolution policy
- OSごとのshell abstraction
- production-grade sandbox
- cross-host process reattach

シェル固有の処理が必要な場合は、使うシェルの実行ファイルと引数を定義情報に明記し、その選択自体を実行対象を識別する情報として扱います。
