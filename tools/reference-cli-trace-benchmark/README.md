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

## 変更前の基準値（main cb5dd852相当、`BuildSourceIndex`変更前）

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

## 出力が変わらないことの確認

`golden_test.go`は、変更前の実装で出力した次のSHA-256を`testdata/golden.json`に固定し、変更後も一致することを調べます。small・mediumの両方で、5種の処理（`TraceWorkspace`の完全なJSON出力）、`BuildSourceIndex`の結果、ページ継続（700件ずつ取得して連結した行が一括`full`と一致）を含みます。あわせて、古い継続tokenとWorkspace更新後の継続tokenが`STALE_SNAPSHOT`で拒否されることを確認します。largeは変更前の実装で約2分かかるため、goldenには含めず手動計測に限ります。

## 実測していないこと

- 実案件のWorkspaceは計測していません。合成Workspaceは1種類の構造（全attributeが1つのTypeを参照）で、実案件のファイルの大きさ、Subjectの種類、edgeの分布を代表しません。
- WindowsやmacOSでの計測、ディスクが遅い環境、ネットワーク上のWorkspaceは計測していません。
- `tei` CLI全体（プロセス起動、フラグ解析）は含めていません。`TraceWorkspace`とJSON符号化までです。
- 全体テキスト生成後の外部ソート、オンディスクスプール、`limit`で行の並べ替えを省く方式、Revisionに束縛したキャッシュは、実装も評価もしていません。#1177の完了条件のうち、これらの評価とキャッシュ失効のfail-closed再現は未完了です。
