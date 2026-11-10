# `trace`／`inspect` の費用を段別に測り、`-limit` が処理量を減らさないことを確認（#1177）

- 記録日: 2026-11-09
- 対象: `tools/reference-cli-trace-benchmark` の既存計測と、現行 main の実 CLI 実行
- 関連: #1177（CLI C2）、#1165（C1）、#1996（並行化）

## 何を確かめたか

#1177 の未決は「`-limit`／`-offset` が**出力**を絞るだけで、内部処理量（Load・Source Index・TraceDirect・Resolver）を絞らない」ことの実測と、段ごとの寄与です。既存の `tools/reference-cli-trace-benchmark` に加えて、現行 main の実 CLI を同じホストで測り直しました。

## 実測1: 段別（in-process、`medium`、3回の中央値、Windows 11 / go1.27.0 / GOMAXPROCS=12）

`go run ./tools/reference-cli-trace-benchmark -sizes medium -repeat 3`

| op | trace ms | encode ms | output bytes | rows |
| --- | ---: | ---: | ---: | ---: |
| summary | 116.5 | 0.0 | 1,650 | 0 |
| full | 88.6 | 9.3 | 1,461,273 | 1,602 |
| full-limit1 | 103.4 | 0.0 | 2,239 | 1 |

| 段 | ms | alloc MiB |
| --- | ---: | ---: |
| `workspace.Load` | 76.0 | 3.7 |
| `index.Build`+`derivedbridge` | 10.8 | 5.2 |
| `TraceDirect` summary | 43.6 | 12.4 |
| `TraceDirect` full | 42.5 | 13.9 |
| `TraceDirect` full limit=1 | 35.4 | 13.9 |
| `workspace.BuildSourceIndex` | 24.6 | 7.0 |
| `sourcebridge.NewResolver` | 12.1 | 2.5 |

**`full -limit 1` は `full` と同じ段をすべて通り、費用もほぼ同じ**（出力は 1,461,273 → 2,239 バイトに減る）。`-limit` は出力の絞りであり、走査予算の別名ではない。これは仕様どおり（`summary` は常に全件の `total/unresolved` を出す）で、隠れた省略をしないことの裏付けです。

## 実測2: 実 CLI（現行 main `6f3ab9cec`、同一ホスト、各3回）

| 実行 | 経過 ms | 出力 bytes |
| --- | ---: | ---: |
| `tei version`（プロセス起動の下限） | 1,019〜1,208 | 小 |
| `inspect -subject-kind semantic-data -subject-id sales.order`（`reference/workspace`） | 1,017〜1,023 | 1,841 |
| `trace … -mode summary`（同上） | 1,010〜1,014 | 2,716 |
| `trace … -mode full`（同上） | 1,009〜1,028 | 4,677 |
| `source-index`（200型の合成Workspace） | 1,015 | — |
| `trace`（同合成Workspace） | 1,272 | — |

`reference/workspace` 規模では、**CLI の経過時間はプロセス起動（約1.0秒）でほぼ決まり**、段の差（summary と full の差）は出力バイト数に見えるだけで経過時間には出ません。合成 200 型の Workspace でも同じ床に埋もれます。つまり現行ホストでは、#1177 の改善は `medium` 以上の規模（段の合計が起動コストを超える領域）でのみ観測できます。

## 段の寄与から見える優先順位

1. `workspace.Load`（76 ms / 全体の約65%）。`Load` と `BuildSourceIndex` は**同じファイルを2回解析**しており、`Load` の解析結果を共有すれば二重解析を消せる。ただし `Load` だけを使う操作の負担が増えるため設計判断が要る（既存 README が「未評価」と記録している項目）。
2. `TraceDirect`（42〜44 ms）。`traceGraphSnapshotBytes` の符号化が支配的で、指紋は全バイトから求めるため省略できない。
3. `BuildSourceIndex`（24.6 ms）＋`NewResolver`（12.1 ms）。`-limit` では減らない。

## 決定していないこと（この記録では変えない）

- 二重解析の共有、ファイル単位の並列解析、オンディスクスプールのいずれも**実装していない**。採用には実案件の規模とメモリ上限の計測が要る。
- この計測は合成 Workspace と `reference/workspace` に対するローカル実行で、実案件のデータ配布・I/O・CPU を代表しない。
- `-limit 1` の費用が `full` と同等であることは仕様として維持する（走査予算と出力予算を混同しない）。変更する場合は契約の変更として別に扱う。
