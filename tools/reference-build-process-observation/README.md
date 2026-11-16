# reference-build-process-observation

一回のBuildについて、**宣言したこと・Build自身が申告したこと・OSが見たこと**を別々の記録として持ち、突き合わせる参照ツールです（[#2831](https://github.com/hat47x/tei-platform/issues/2831) SB-P0.1、[#2600](https://github.com/hat47x/tei-platform/issues/2600)）。

Buildを実行する仕組みではありません。工程を起動せず、受理や資格も決めません。観測して、照合するだけです。

## なぜ要るか

`tei self-build`の記録は、entryが自分で読み直した値と、builderが申告した値でできています。どちらもBuildの内側から書いた記録なので、次のことは確かめられません。

- 申告にない実行物が動いたか
- 名指ししたファイルが、起動後に別の実行物へ処理を渡したか
- 二回目のBuildが、一回目のstateを読んだか
- 宣言した入力の外を読んだか

このツールは、Buildの外側からプロセス木を追跡し、実行されたすべてのプログラムと開かれたすべてのパスを記録します。記録を書くのはBuildに参加しない第三者です。

## 三つの操作

| 操作 | 入力 | 出力 |
| --- | --- | --- |
| `observe` | 観測するコマンド | 観測記録（`tei.reference.build-process-observation/v0`） |
| `scope` | `tei self-build`の報告 | 宣言範囲（`tei.reference.build-observation-scope/v0`） |
| `reconcile` | 観測記録と宣言範囲 | 照合結果（`tei.reference.build-observation-ledger/v0`） |

```bash
go build -o observe ./tools/reference-build-process-observation

# 1. コマンドを観測する。-files を付けると、開いたパスも記録する
./observe observe -files -out observation.json -- tei self-build ... > report.json

# 2. 報告から宣言範囲を作る。entryが知らされていない入力は呼び手が名指す
./observe scope -self-build-report report.json \
  -read source-root=/path/to/checkout -read intent=intent.json -out scope.json

# 3. 照合する。範囲外の実行や読み書きがあれば終了コード1
./observe reconcile -observation observation.json -scope scope.json -out ledger.json
```

`observe`は観測対象の終了コードをそのまま返します。観測そのものに失敗した場合は125です。

## 観測記録

| 欄 | 内容 |
| --- | --- |
| `execs[]` | 成功した`execve`の列。実行ファイルのパス、**そのファイルのsha256**、引数、作業ディレクトリ、環境変数の名前 |
| `execs[].parentSeq` | そのプロセスが直前に動かしていたプログラム。観測対象のコマンド自身は`seq 0` |
| `execs[].step` | コマンドが直接起動した子のうち、どの子の配下か。一つの子が起動したものは同じstepになる |
| `files[]` | 開いたパスごとの回数。読取り、書込み、失敗を分け、開いたプログラムとstepを持つ |
| `notObserved` | 観測していない範囲。**常に空でない** |

実行ファイルのdigestは、パスからではなく`/proc/<pid>/exe`から読みます。同じパスに別のファイルが置かれていても、実際に動いたファイルを指します。

## 宣言範囲と、その出どころ

宣言範囲は、宣言の側から作ります。照合する相手の観測記録からは作りません。項目ごとに、誰の言明かを`source`に持ちます。

| `source` | 意味 |
| --- | --- |
| `plan` | 計画が、Buildの前に名指した |
| `entry-report` | entryが、自分で渡した値として報告した |
| `builder-report` | builderが、Buildの後に申告した |
| `derived-from-report` | 報告にない値を、報告の値から規約で導いた |
| `caller` | 呼び手が名指した。entryは知らされていない |

計画がtoolchainを名指していない場合、builderが申告したtoolchainは`toolchain-reported-not-declared`という役割になります。範囲には入りますが、宣言済みとは表示しません。

OSと言語処理系が必ず触るパス（`/proc/self/`、cgroup、動的リンカの検索先など）は`os-interface`として扱います。`/etc/localtime`と`/tmp`は入れていません。時刻帯と共有の一時領域は環境から来る入力であり、Buildが読んだなら見えるべきだからです。

## 照合結果

| 欄 | 内容 |
| --- | --- |
| `execs.byRole` | 実行を役割ごとに数えたもの |
| `execs.undeclared` | どの宣言済みdigestとも一致せず、どの宣言済み閉包の下にもない実行 |
| `files.byRole` | 開いたパスを役割ごとに数えたもの |
| `files.undeclared` | 範囲の外で成功した読み書き |
| `files.crossStep` | 別のstepが持つ領域への読み書き。二回のBuildがstateを共有した場合はここに出る |
| `files.writeToReadOnly` | 読取り用に宣言した領域への書込み |
| `files.probesOutsideScope` | 範囲の外で失敗した読取り。入力ではないが、後からそこへファイルが置かれると入力になる |
| `boundedAbsence.state` | `holds`または`not-proven` |

`boundedAbsence`が`holds`になるのは、パスを追跡した観測で、範囲外の実行・読み書き・step越えが一件もない場合だけです。パスを追跡していない記録や、実行が一件もない記録では、決して`holds`になりません。

## 言えること、言えないこと

`holds`が言うのは、「**このホストでのこの一回の実行で、観測したコマンドが起動したプロセス木の中では、宣言範囲の外で何も実行されず、何も開かれなかった**」ことです。

次は言いません。

- **宣言済み閉包の内部。** toolchainの下のプログラムは、閉包の一員として範囲に入ります。compilerが内部で何をしたかは記述しません
- **開いた後の読み書き。** 記録するのは開いたことと、その向きです。読んだbyteや書いたbyteは見ていません
- **木の外。** 既に動いていたdaemonやserverとの通信、ネットワーク、時計、乱数
- **環境変数の値。** 名前だけを記録します
- **Seed Trust、受理、Runtime Authority。** 観測は実行の事実であり、資格ではありません
- **ほかのホスト。** `observe`はlinux/amd64だけで動きます。ほかのホストでは記録を書かずに終了コード125で止まります

追跡には費用がかかります。`./cmd/tei`の全鎖（二回のBuild）は、追跡なしで44秒、`-files`なしの追跡でほぼ同じ、`-files`付きで約215秒でした（1台での実測）。

## 試験

```bash
go test ./tools/reference-build-process-observation -count=1
```

照合の期待値は、上の規則から手で書いたものです。`reconcile`の出力から作っていません。`observe`の試験はlinux/amd64で実際にプロセスを追跡します。
