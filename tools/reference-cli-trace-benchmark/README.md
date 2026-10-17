# CLI traceの処理段別計測

`tei trace`が大規模Workspaceで時間とメモリをどの処理段に使うかを、合成Workspaceで計測する道具です。Issue #1177（CLI C2）の根拠として使います。製品の動作は定めず、実案件のデータも使いません。

## 実行方法

```sh
go run ./tools/reference-cli-trace-benchmark                # small, medium, large を各5回
go run ./tools/reference-cli-trace-benchmark -sizes small,medium -repeat 3
go test ./tools/reference-cli-trace-benchmark               # 出力不変の確認（goldenと失敗応答）
```

各処理は子プロセスで実行し、ピーク常駐メモリ（`/proc/self/status`のVmHWM、Linuxのみ）と経過時間の中央値を報告します。段別の表は1回の実行で、各段の前にGCして基準ヒープを取り、段中のヒープ最大値を1ミリ秒ごとに標本化します。段別の表は`TraceWorkspace`の内部段を同じ順に個別実行した再現であり、`TraceWorkspace`本体の計測は上の処理別の表を正とします。段別のSource Index検証に使う読取り関数は単純な`os.ReadFile`で、本体の`readWorkspaceSource`（symlink封じ込め検査付き）とは異なります。

## 合成Workspaceの寸法

`fixture.go`が乱数を使わずに生成します。全attributeのうち偶数番目が1つのType（`hub.hub_id`）を参照するため、そのTypeを対象にした`trace`は参照元の数だけ行を返します。

| 寸法 | ファイル数 | attribute数 | ハブへの宣言済みedge | Workspace総byte数 |
| --- | ---: | ---: | ---: | ---: |
| small | 5 | 80 | 40 | 3,337 |
| medium | 41 | 3,200 | 1,600 | 119,229 |
| large | 201 | 40,000 | 20,000 | 1,467,549 |

処理は次の5種です。`summary`、`full`、`full-limit1`はハブを対象にし、`leaf-summary`と`leaf-full`は出向きedgeが1本のattributeを対象にします。後者は行数に依らない固定費を表します。

## 計測環境

- WSL2（Linux 6.18、amd64）、Go 1.26.9、13th Gen Intel Core i7-1365U（12論理CPU）、メモリ約7.8GiB
- `GOMAXPROCS=12`。Windowsホストの負荷による揺れがあり、同じ処理でも十数％から数十％ずれます。数値は桁と傾向を見るためのものです。

## 変更前の基準値（変更前のコミットbc4b673e、基点65f83eee。largeの値は同じ処理を含む旧基点での計測）

medium（5回の中央値）とlarge（3回の中央値）の処理別の時間です。

| 寸法 | summary | full | full -limit 1 | leaf-summary | leaf-full | 出力byte（full / limit1） |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| small | 9.6 ms | 12.6 ms | 12.9 ms | 10.8 ms | 12.7 ms | 39,302 / 2,052 |
| medium | 1,036 ms | 1,075 ms | 1,199 ms | 920 ms | 873 ms | 1,461,088 / 2,054 |
| large | 73,035 ms | 83,473 ms | 123,090 ms | 51,321 ms | 69,224 ms | 18,260,529 / 2,055 |

largeの段別（1回）では、`BuildSourceIndex`が113,802 msで全体のほぼすべてを占め、確保量は8,508 MiB、確保回数は5,900万回でした。その他の段は、Load 281 ms、`index.Build`と`derivedbridge` 634 ms、`TraceDirect` 480〜1,602 ms、`NewResolver` 1,625 msです。ピーク常駐メモリはlargeで約127〜162 MiBでした。

## 支配的な費用と変更

`workspace.BuildSourceIndex`は、意味定義のSubjectごとに、そのSubjectを含むファイル全体を構文解析し直していました（`exactSubjectRange`）。解析量が「1ファイルのSubject数 × ファイルの大きさ」に比例して増えるため、largeでは時間の約97%を占めました。`summary`でも`full -limit 1`でも同じ費用がかかるのは、`TraceWorkspace`が出力行の数に依らず全Workspaceの出所を検証する設計だからです。

変更は、`BuildSourceIndex`の1回の呼出しの中に限って、ファイルごとの解析結果（Subjectごとの先頭anchor）を共有するだけです。解析や検証の内容、出力する行、診断の文言、fingerprintは変えません。呼出しをまたぐキャッシュは作らないので、古いWorldやSource Indexを参照する経路は増えません。

## 変更後の計測（同じ環境、5回の中央値）

| 寸法 | summary | full | full -limit 1 | leaf-summary | leaf-full |
| --- | ---: | ---: | ---: | ---: | ---: |
| small | 3.0 ms | 3.9 ms | 3.8 ms | 2.8 ms | 3.0 ms |
| medium | 85 ms | 117 ms | 125 ms | 207 ms | 104 ms |
| large | 3,533 ms | 2,190 ms | 2,277 ms | 2,006 ms | 2,370 ms |

largeの`BuildSourceIndex`は113,802 msから1,177 msに、確保量は8,508 MiBから70 MiBになりました。largeのピーク常駐メモリは変更前後でほぼ同じ（約125〜153 MiB）です。largeの`summary`が`full`より遅い値になっているのは計測の揺れで、傾向の差ではありません。

残る費用は、largeで`Load`約0.16〜0.5秒、`index.Build`と`derivedbridge`約0.3〜0.4秒、`TraceDirect`約0.3〜0.6秒、`BuildSourceIndex`約1.2〜1.6秒（行・列位置を数え直す`PositionAt`の走査が中心と見られます）、`NewResolver`約0.4〜0.9秒（Source Indexの検証を2回目に行う）です。これらは別の候補として未着手です。

## 位置の計算の改善（#1177の続き）

CPUプロファイル（largeの`full`を繰り返し実行、`go test -bench`）では、`sourceindex.PositionAt`が単独で全サンプルの約29%（3.96秒／13.6秒）を占めていました。`PositionAt`は呼出しのたびにファイルの先頭からoffsetまでを1バイトずつ数えており、`exactSubjectRange`・`RangeFromOffsets`・`Range.Validate`から多数回呼ばれるため、1ファイルに多数のSubjectがあると費用が二乗になります。

行を`bytes.Count`、列を`bytes.LastIndexByte`で求める形に変えました。結果は1バイトずつ数える従来の定義と同一です。`internal/sourceindex/position_test.go`が、あらゆるoffsetで素朴な実装と照合し、`OffsetAt`との往復も確かめます。出力byte数と件数は変わっていません。

変更前（main `36169dd6`）と変更後（`703e3845`）を、同じWSL2・Go 1.26.9・同じ生成Workspaceで測りました（largeの3回の中央値）。

| 処理 | 変更前 | 変更後 |
| --- | ---: | ---: |
| summary | 1,935 ms | 823 ms |
| full | 1,741 ms | 1,070 ms |
| full -limit 1 | 2,059 ms | 1,034 ms |
| leaf-summary | 2,361 ms | 739 ms |
| leaf-full | 2,267 ms | 719 ms |

段別（1回）では、`BuildSourceIndex`が1,109 msから255 ms、`NewResolver`が386 msから57 msになりました。他の段は変わっていません（`Load`・`index.Build`・`TraceDirect`の差は計測の揺れの範囲です）。

変更後に残る主な費用は、`TraceDirect`の`full`で行キーを作る文字列整形（約0.4秒）、`index.Build`と`derivedbridge`（約0.2秒）、`Load`（約0.13秒）、およびメモリの確保に伴う回収です。これらは未着手です。

## 行キー生成の改善（#1177の続き）

`PositionAt`の改善後も、largeの`full`と`full -limit 1`は`summary`より約0.3秒遅く、`TraceDirect`の`full`の段が約0.5秒かかっていました。CPUプロファイルでは`traceRowKey`が全体の約18%（1.99秒／11.1秒）、そのうち`fmt.Sprintf`が約1.2秒でした。原因は、並べ替えの比較関数の中で、比較のたびに両側の行キー（11個の欄を整形し、証拠をJSON符号化する）を作り直していたことです。n行の整列でO(n log n)回のキー生成が起きます。

行キーを行ごとに一度だけ作り、行とキーを並べて整列する形に変えました（`sortTraceRows`）。あわせて、キー生成の`fmt.Sprintf`の連結を`strconv.AppendInt`とバイト列の追記へ置き換えました。キーの文字列と整列後の行は、書き換え前の定義と一致します（`internal/modelref/trace_sort_test.go`が、区切り文字を含むID・多バイト文字・空文字・重複行を混ぜた乱数列で照合します）。

変更前（main `775f5c2c`）と変更後（`df45adb6`）を、同じ環境で交互に2回ずつ測りました（largeの5回の中央値、単位はms）。

| 処理 | 変更前（1回目 / 2回目） | 変更後（1回目 / 2回目） |
| --- | ---: | ---: |
| summary | 823 / 738 | 725 / 722 |
| full | 1,046 / 1,039 | 715 / 754 |
| full -limit 1 | 1,101 / 989 | 735 / 754 |
| leaf-summary | 779 / 706 | 722 / 752 |
| leaf-full | 753 / 726 | 711 / 748 |

`summary`と`leaf-*`は整列を通らないため、差は計測の揺れの範囲です。段別（1回）では、`TraceDirect`の`full`が494 msから196 ms、確保回数が380万回から36万回、確保量が248 MiBから130 MiBになりました。出力byte数（18,260,529 / 2,055）と件数（20,002）は変わっていません。

変更後に残る主な費用は、`BuildSourceIndex`（約0.25秒）、`index.Build`と`derivedbridge`（約0.19秒）、`Load`（約0.14秒）、`TraceDirect`の`summary`（約0.2秒）です。

## 出力が変わらないことの確認

`golden_test.go`は、変更前の実装で出力した次のSHA-256を`testdata/golden.json`に固定し、変更後も一致することを調べます。small・mediumの両方で、5種の処理（`TraceWorkspace`の完全なJSON出力）、`BuildSourceIndex`の結果、ページ継続（700件ずつ取得して連結した行が一括`full`と一致）を含みます。あわせて、古い継続tokenとWorkspace更新後の継続tokenが`STALE_SNAPSHOT`で拒否されることを確認します。largeは変更前の実装で約2分かかるため、goldenには含めず手動計測に限ります。

## 参照グラフ生成の改善（#1177の続き）

段別の`index.Build`と`derivedbridge`は約0.19秒、確保113万回でした。largeを`go test -bench`で繰り返し測ると、`index.Build`単体は約27 ms・57回の確保で、費用の大半は`derivedbridge.FromIndex`でした。確保したオブジェクトの約77%が、`FromIndex`の整列の比較関数が比較のたびに作る「種別/ID」などの文字列連結でした（`#1177`のtraceの整列と同じ形です）。

整列キーを要素ごとに一度だけ作る形（`sortByKey`）に変え、`semanticKind`の文字列連結を定数に、Nodes・Edges・ノード表の容量を入力の件数から事前確保する形にしました。キーの文字列が同じで、同じ`sort.Slice`で比較するため、キーが等しい要素の並びも含めて結果は従来と同一です。`internal/modelref/derivedbridge/projection_order_test.go`が、書き換え前の`FromIndex`を定義として保持し、`Projection`全体の一致を照合します。

変更前（main `905c5fa8`）と変更後（`190f2372`）を、同じ環境で交互に2回ずつ測りました（largeの段別、1回ずつ）。

| 段 | 変更前（1回目 / 2回目） | 変更後（1回目 / 2回目） |
| --- | ---: | ---: |
| 2 index.Build+derivedbridge | 202 / 187 ms | 110 / 94 ms |
| 同・確保回数 | 1,130,978 / 1,130,956 | 128,965 / 128,939 |
| 同・確保量 | 149.4 MiB | 67.2 MiB |

largeの`TraceWorkspace`全体（5回の中央値）は、`summary`が826 / 815 msから812 / 759 ms、`leaf-summary`が829 / 807 msから674 / 657 msでした。段の差（約0.09秒）は全体の揺れ（±10%）に近く、全体の値は傾向としてだけ読んでください。出力byte数と件数は変わっていません。

変更後に残る主な費用は、`BuildSourceIndex`（約0.25秒）、`TraceDirect`の`summary`（約0.2秒）、`Load`（約0.14秒）、`index.Build`と`derivedbridge`（約0.1秒）です。

## 識別子の検証の改善（#1177の続き）

参照グラフ生成の改善後にlargeの`summary`を再度プロファイルすると、`core/identity.ParseCanonicalID`が`TraceWorkspace`全体の約9%（1.53秒／16.4秒）を占め、その大半は正規表現の照合（`regexp.MatchString`は約11%）でした。`workspace.Load`と、`BuildSourceIndex`内の再解析の双方が、属性ごとにこの検証を呼びます。

`CanonicalID`（`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*$`）と`LocalName`（`^[a-z][a-z0-9_]*$`）の検証を、同じ言語を1回の走査で判定する形に変えました。周囲の空白を取り除く処理、エラー文言、`SymbolToSegment`の結果は変えていません。`core/identity/id_equivalence_test.go`が、正規表現を定義として保持し、英小文字・数字・`_`・`.`・大文字・空白・多バイト文字の全組合せ（長さ6まで）と乱数列20万件で、受理の可否・値・エラー文言の一致を照合します。

1件の検証は約367 nsから約25 ns（確保はどちらも0回）になりました。

変更前（main `af03b2c3`）と変更後（この変更のコード。試験ファイルの整形だけが計測後に加わりました）を、同じ環境で交互に2回ずつ測りました（largeの5回の中央値、単位はms。段別は1回）。

| 処理 | 変更前（1回目 / 2回目） | 変更後（1回目 / 2回目） |
| --- | ---: | ---: |
| summary | 689 / 704 | 630 / 618 |
| full | 754 / 855 | 660 / 665 |
| full -limit 1 | 744 / 897 | 629 / 636 |
| leaf-summary | 696 / 828 | 604 / 627 |
| leaf-full | 671 / 720 | 617 / 609 |
| 段 1 `workspace.Load` | 119 / 118 | 97 / 86 |
| 段 4 `workspace.BuildSourceIndex` | 219 / 225 | 159 / 156 |

出力byte数と件数は変わっていません。Windowsホストの負荷による揺れ（±10〜20%）があるため、全体の数値は傾向として読んでください。

## 実測していないこと

- 実案件のWorkspaceは計測していません。合成Workspaceは1種類の構造（全attributeが1つのTypeを参照）で、実案件のファイルの大きさ、Subjectの種類、edgeの分布を代表しません。
- WindowsやmacOSでの計測、ディスクが遅い環境、ネットワーク上のWorkspaceは計測していません。
- `tei` CLI全体（プロセス起動、フラグ解析）は含めていません。`TraceWorkspace`とJSON符号化までです。
- 全体テキスト生成後の外部ソート、オンディスクスプール、`limit`で行の並べ替えを省く方式、Revisionに束縛したキャッシュは、実装も評価もしていません。#1177の完了条件のうち、これらの評価とキャッシュ失効のfail-closed再現は未完了です。
