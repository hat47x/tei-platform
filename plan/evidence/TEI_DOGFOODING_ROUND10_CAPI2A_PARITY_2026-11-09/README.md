# inspect・trace の実プロセス同値と失敗分類（2026-11-09、第10ラウンド）

- 対象: [#3420](https://github.com/hat47x/tei-platform/issues/3420) CAPI-2a、[ADR-0048](../../adr/0048-shared-application-api-and-transport-adapters.md) §5 T01・T04・T06・T07・T08。
- 前提: [第5ラウンド](../TEI_DOGFOODING_ROUND5_CLI_MCP_2026-11-09/README.md)で4操作の値一致を確認したが、実プロセス比較が固定されていたのは `validate` の1操作だけだった。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **read-only の比較実測と、回帰試験の追加**。意味の受理・権限付与・配備は行っていない。

## 1. 目的

CAPI-2a が求める「実CLI Process と実MCP Process の対照」を、`inspect`・`trace` について行う。値だけでなく、失敗の符号と成否の一致を含める。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `5d8c4a564f88ec69305fd740bb890e2391e4bf74`（`origin/main`） |
| CLI実行物 | `sha256:803d3a1c776b7bdd6a43c4986dc9fb54df6093c4a589e7b4627012a5b782ef8f` |
| MCP実行物 | `sha256:3255173b58dc95d3be50332c1bec41156b38a7cf4afc0a984c9e07cd3e995d98` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64` |
| 対象Workspace | `reference/workspace` |

## 3. 生データ

| ファイル | 内容 |
| --- | --- |
| `00-head.txt`・`01-binaries.txt` | 対象 commit と2実行物の識別 |
| `02-failure-parity.txt` | 8ケースの CLI／MCP の成否と符号 |
| `03-mcp-missing-workspace.jsonl`・`.err` | 存在しない Workspace で起動した MCP |
| `04-mcp-broken-workspace.jsonl`・`.err` | manifest が壊れた Workspace で起動した MCP |
| `06-new-tests.txt` | 追加した2試験の PASS |
| `07-package.txt` | adapter パッケージ全体の PASS |
| `08-gofmt.txt`・`09-vet.txt` | 0 byte（整形・vet の指摘なし） |
| `05-mutations.txt` | 符号をわざと変えたときの検出 |

## 4. 観測

### 4.1 値と符号は一致

8ケースのうち7ケースで、CLI と MCP の成否と `error.code` が一致した。

| ケース | CLI | MCP |
| --- | --- | --- |
| `inspect` 対象不在 | `SUBJECT_NOT_FOUND`・exit 1 | `isError:true`・`SUBJECT_NOT_FOUND` |
| `inspect` 未知kind | `UNSUPPORTED_SUBJECT_KIND`・exit 2 | `isError:true`・`UNSUPPORTED_SUBJECT_KIND` |
| `trace` 対象不在 | `SUBJECT_NOT_FOUND`・exit 1 | `isError:true`・`SUBJECT_NOT_FOUND` |
| `trace` 未対応kind | `UNSUPPORTED_SUBJECT_KIND`・exit 2 | `isError:true`・`UNSUPPORTED_SUBJECT_KIND` |
| `trace` 古い指紋 | `STALE_SNAPSHOT`・exit 1 | `isError:true`・`STALE_SNAPSHOT` |
| `trace` 負の offset | `INVALID_ARGUMENT`・exit 2 | `isError:true`・`INVALID_ARGUMENT` |
| `model catalog` 未知namespace | `NAMESPACE_NOT_FOUND`・exit 1 | `isError:true`・`NAMESPACE_NOT_FOUND` |

### 4.2 「Workspace が無い」は入口で表現が違う

`inspect -workspace /no/such/dir` は CLI では `WORKSPACE_LOAD_FAILED`（exit 2）だが、MCP の tool 呼出しには Workspace を指定する引数が無い（起動時に固定する設計）。そのため同じ「読めない Workspace」を、MCP では**起動時の `-workspace`** でしか表現できない。

起動時の Workspace が存在しない場合も manifest が壊れている場合も、MCP は `isError:true` と `WORKSPACE_LOAD_FAILED` を返した（`03`・`04`）。**符号は CLI と同じ**で、差は「いつ分かるか」（呼出し時か起動時か）だけである。

### 4.3 必須引数の欠落だけが層の違い

`inspect` から `-subject-id` を落とすと、CLI は `INVALID_SUBJECT`（exit 2）、MCP は `-32602`（プロトコル層）になる。どちらも操作を実行しない点は同じだが、**文字列の符号が揃わない**。ADR-0048 §2.3「失敗」の未決であり、本ラウンドでは統一しない。

## 5. 追加した回帰試験

`tools/reference-mcp-readonly-adapter/inspect_trace_parity_test.go`。

| 試験 | 内容 |
| --- | --- |
| `TestInspectAndTraceMatchAcrossRealProcesses` | 実CLIと実MCPを別プロセスで起動し、成功2件・失敗5件について、`ok` と `isError` の反転一致、`error.code` の一致、成功時の `result` の byte 一致を検査する |
| `TestMissingRequiredArgumentIsRefusedByBothEntrances` | 必須引数の欠落が、CLI では `INVALID_SUBJECT`・MCP では `-32602` で**両方拒否**されることを固定する（差を広げないための試験） |

### 5.1 試験が効くことの確認

adapter の符号対応をわざと変えた（`05-mutations.txt`）。

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | `trace` の符号を `TRACE_NOT_FOUND` へ | FAIL: `failure code differs: CLI="SUBJECT_NOT_FOUND" MCP="TRACE_NOT_FOUND"` ほか3件 |
| M2 | `inspect` の符号を `OPERATION_FAILED` へ | FAIL: 同様に2件 |

戻すと PASS に戻る。**片側だけを変えても検出できる**。

## 6. 限界

- 比較するのは1つの Workspace（`reference/workspace`）と、そこで実在する subject だけである。大規模Workspace（[#1177](https://github.com/hat47x/tei-platform/issues/1177)）や改行差（[#2399](https://github.com/hat47x/tei-platform/issues/2399)）は未実行。
- `trace` のページ継続（`offset`・`limit` と `nextOffset`）は成功1件でしか見ていない。T06 の継続トークンは未検査。
- `view -execute` など書込みを伴う経路、HTTP・Desktop、実DB・実Provider、権限は未実行。
- 「起動時に固定した Workspace が読めない」場合の符号一致は、MCP の起動引数で確かめた。CLI の同名の失敗と同じ分類かは、CLI の `-workspace` 不在と揃っていることだけを見ている。

## 7. 位置付け

CAPI-2a の「inspect・trace の実プロセス同値」を、成功と主要な失敗について固定した。値と符号は一致し、残る差は「必須引数の欠落を意味層の符号で返すかプロトコル層で返すか」の1点だけである。この差は #3420 の判断事項として残す。
