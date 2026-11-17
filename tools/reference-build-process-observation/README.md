# reference-build-process-observation

一回のBuildについて、**宣言したこと・Build自身が申告したこと・OSが見たこと**を別々の記録として持ち、突き合わせる参照ツールです（[#2831](https://github.com/hat47x/tei-platform/issues/2831) SB-P0.1、[#2600](https://github.com/hat47x/tei-platform/issues/2600)）。

Buildを実行する仕組みではありません。工程を起動せず、受理や資格も決めません。観測して、照合するだけです。

## なぜ要るか

`tei self-build`の記録は、entryが自分で読み直した値と、builderが申告した値でできています。どちらもBuildの内側から書いた記録なので、次のことは確かめられません。

- 申告にない実行物が動いたか
- 名指ししたファイルが、起動後に別の実行物へ処理を渡したか
- 二回目のBuildが、一回目のstateを読んだか
- 宣言した入力の外を読んだか

このツールは、Buildの外側からプロセス木を追跡し、実行されたプログラムと、パスを指定して開いたファイルを記録します。記録を書くのはBuildに参加しない第三者です。追跡できない呼出しに出会った場合は、その事実を記録に残します。

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
| `limits` | 観測器が出会ったが読めなかったもの。一件でもあれば、記録は不完全である |
| `notObserved` | 観測していない範囲。**常に空でない** |

実行ファイルのdigestは、パスからではなく`/proc/<pid>/exe`から読みます。同じパスに別のファイルが置かれていても、実際に動いたファイルを指します。

開いたファイルのパスも、要求された文字列ではなく、**カーネルが実際に開いたファイル**のパスです（返ってきた記述子を`/proc/<pid>/fd/<n>`で読みます）。シンボリックリンク、`..`、`/proc/self/root`を経由した要求は、すべて実体のパスとして記録されます。

追跡する呼出しは、`open`・`openat`・`openat2`・`creat`と、`rename`・`link`・`symlink`・`unlink`・`truncate`の各系統です。名前の変更や削除は、対象パスへの書込みとして数えます。

| `limits`の欄 | 数えるもの |
| --- | --- |
| `foreignAbiSyscalls` | 32bitなど別のABIで発行されたシステムコール。番号が違うので解読しない |
| `unobservableCalls` | パスを追えない開き方（`open_by_handle_at`、`io_uring_setup`）、パスの意味を変える操作（`unshare`、`setns`、`mount`、`chroot`、`pivot_root`）、追跡から外れる`clone`（`CLONE_UNTRACED`） |
| `traceFailures` | カーネルが問合せを拒否した停止、読めなかったパス引数 |
| `unattributedTasks` | 起動元を特定できなかったタスク |

## 宣言範囲と、その出どころ

宣言範囲は、宣言の側から作ります。照合する相手の観測記録からは作りません。項目ごとに、誰の言明かを`source`に持ちます。

| `source` | 意味 |
| --- | --- |
| `plan` | 計画が、Buildの前に名指した |
| `entry-report` | entryが、自分で渡した値として報告した |
| `builder-report` | builderが、Buildの後に申告した |
| `derived-from-report` | 報告にない値を、報告の値から規約で導いた |
| `caller` | 呼び手が名指した。entryは知らされていない |

宣言範囲は、Buildの前に書かれた宣言だけからは作れません。報告は同じ実行の後に出るものだからです。そこで、項目ごとに出どころを残し、**Build自身の申告だけに基づく項目は、範囲外が無いことの根拠に使いません。**

計画がtoolchainを名指していない場合や、名指したものと一致しなかった場合、builderが申告したtoolchainは`toolchain-reported-not-declared`という役割になります。その下で動いたプログラムは役割ごとの集計には現れますが、`boundedAbsence`は`holds`になりません。builderは任意のディレクトリを申告できるからです。計画が名指すのはtoolchainのdigestで、パスではありません。パスを宣言済みとして扱うのは、entryが「申告されたtoolchainは計画が名指したものである」と確かめた場合だけです。

ファイルシステムの根や相対パスは、範囲の根として受け付けません。

OSと言語処理系が必ず触るパスは`os-interface`として扱います。対象は個別に列挙しています（`/proc/self/maps`、`/proc/self/cgroup`、`/proc/self/mountinfo`、cgroupの設定、共有ライブラリの置き場など）。`/proc/self/`全体は入れていません。`/etc/localtime`と`/tmp`も入れていません。時刻帯と共有の一時領域は環境から来る入力であり、Buildが読んだなら見えるべきだからです。

toolchainなどの閉包と`os-interface`は読取り専用として扱います。そこへの書込みは`writeToReadOnly`に出ます（`/dev/null`を除く）。

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

`boundedAbsence`が`holds`になるのは、次のすべてが成り立つ場合だけです。

- パスを追跡した観測であり、少なくとも一件の読み書きが記録されている
- 範囲外の実行、範囲外の読み書き、step越え、読取り専用領域への書込みが一件もない
- Build自身の申告だけに基づく閉包の下で動いたプログラムがない
- `limits`のどの欄も0である

## 言えること、言えないこと

`holds`が言うのは、「**このホストでのこの一回の実行で、観測したコマンドが起動したプロセス木の中では、宣言範囲の外で何も実行されず、何も開かれなかった**」ことです。

次は言いません。

- **宣言済み閉包の内部。** toolchainの下のプログラムは、閉包の一員として範囲に入ります。compilerが内部で何をしたかは記述しません
- **開いた後の読み書き。** 記録するのは開いたことと、その向きです。読んだbyteや書いたbyteは見ていません
- **開かずに得る情報。** `stat`や`readlink`で得たファイルの有無・大きさ・リンク先は記録しません。`mkdir`も記録しません
- **領域を持ち主で分けていない場合の共有。** step越えを検出できるのは、持ち主のstepを宣言した領域だけです。呼び手が`-write`で名指した領域には持ち主がありません。`rename`による持ち出しは、両方のパスへの書込みとして現れます
- **残った常駐プロセスを介した共有。** 先のstepが残したプロセスへ後のstepがソケットでつないだ場合、ファイルを開くのは先のstepのプロセスです
- **stepの番号。** stepは、観測したコマンドが直接起動した子の順番です。コマンド自身が別のプログラムへ`exec`する場合や、builderの前に別の子が起動される場合は番号がずれ、step越えとして現れます（誤って`holds`になる方向ではありません）
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

試験していない範囲があります。`observe`の試験の対象はGoのプログラムで、絶対パスの`openat`だけを発行します。`open`、相対パスと記述子からの解決、`openat2`、32bitの呼出し、`limits`の各欄を実際に発生させる試験はありません。`scope`の試験は手で書いた報告を使っており、実際の`tei self-build`の報告との対応は、実測記録の実行で確かめたものです。
