# CLI discovery と MCP tool 宣言を突き合わせる回帰試験（2026-11-09）

- 対象: [#3419](https://github.com/hat47x/tei-platform/issues/3419)、[#3420](https://github.com/hat47x/tei-platform/issues/3420)、[ADR-0048](../../adr/0048-shared-application-api-and-transport-adapters.md) §2.6。
- 前提: [第6ラウンド](../TEI_DOGFOODING_ROUND6_DISCOVERY_2026-11-09/README.md)で、CLI discovery と MCP `tools/list` を突き合わせる試験が無いことを確認した。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **read-only**。新しい規約や権限は作らない。試験のみを追加する。

## 1. 追加した試験

`tools/reference-mcp-readonly-adapter/discovery_parity_test.go`。

| 試験 | 内容 |
| --- | --- |
| `TestToolListAgreesWithTheCLICatalog` | 実CLIをビルドして `commands` を実行し、広告された tool と突き合わせる |
| `TestToolDescriptionsNameTheCLICommand` | 各 tool の説明が、対応する CLI 呼出し（`tei <command>`）を名指ししていることを検査する |

突き合わせるのは次の5点である。

1. tool が対応する CLI コマンドが `commands` に広告されていること。
2. そのコマンドの `effect` が `read-only` であること。
3. `writeFlags` を持たないこと（書込みを伴う操作を MCP から露出しない）。
4. `schemaVersion` が adapter が応答に書く版と一致すること。
5. `jsonContract` が `enveloped` であること。

対応付け（`tei_model_catalog` → `model catalog` など）は、文字列変換で推測せず表で宣言する。CLI と MCP は操作を別名で広告するため、変換規則に頼ると改名のときに静かにずれる。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `60d14dcc1027eb1b0a46041e0254fb6d62d7307b`（`origin/main`） |
| CLI実行物 | `sha256:4d6653dfe22fc885901e7faa3d2d41f76e03186868179755fbed45ba72ff2829` |
| MCP実行物 | `sha256:80d5c6b0ff505ee54edeb29fe7330fb10abbf00d1a73f724291d63e060882acc` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`、gofmt は `1.26.10` |

## 3. 生データ

| ファイル | 内容 |
| --- | --- |
| `00-head.txt` | 対象 commit |
| `01-cli-sha256.txt` | CLI 実行物の識別 |
| `02-commands.txt` | `tei commands`（19件） |
| `03-tools-list.json` | MCP `tools/list`（4 tool） |
| `04-baseline.txt` | 新しい2試験の PASS |
| `05-package.txt` | adapter パッケージ全体の PASS |
| `06-gofmt.txt` | `gofmt -l` が空であること（0 byte） |
| `07-mutations.txt` | 故意のずれ6種と、その検出結果 |

## 4. 観測

### 4.1 現行版では一致

新しい2試験は現行 `main` で PASS した。4 tool すべてが対応する CLI コマンドを持ち、`effect` は `read-only`、`writeFlags` は無く、応答版は `tei.cli.{model,inspect,trace,validate}/v0`、`jsonContract` は `enveloped` である。

**この一致は「ずれが無い」ことの証明ではない**。固定しているのは、上に挙げた5点が両面で同じであることだけである。

### 4.2 ずれを検出できることの確認

試験が実際に効くかを、故意のずれ6種で確かめた（`07-mutations.txt`）。

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | adapter の対応表を存在しないコマンド名へ | FAIL: `tool "tei_inspect" maps to "inspectt", which the CLI catalog does not advertise` |
| M2 | adapter が固定する応答版を別版へ | FAIL: `CLI schemaVersion="tei.cli.validate/v0", adapter pins "tei.cli.validate/v1"` |
| M3 | adapter に対応先の無い tool を追加 | FAIL: `adapter advertises 4 tools, the mapping names 5` |
| M4 | CLI discovery の応答版を別版へ | FAIL: 同じ不一致を CLI 側から検出 |
| M5 | CLI discovery の `effect` を `write` へ | FAIL: `"inspect" is advertised as effect="write"` |
| M6 | CLI discovery の `trace` を別名へ | FAIL: `tool "tei_trace" maps to "trace", which the CLI catalog does not advertise` |

6件すべてが FAIL し、変異を戻すと PASS に戻った。**両面のどちらを変えても検出できる**ことを確認した。

## 5. 限界

- 突き合わせるのは discovery の一致だけである。同じ入力で同じ結果が返ることは、`validate` の byte 比較と `catalog` の namespace 失敗比較が別に持つ。`inspect`・`trace` の実プロセス比較は #3420 CAPI-2a が持つ。
- 引数の値集合（`argumentValues`）や既定値（`argumentDefaults`）は、MCP の `inputSchema` と比べていない。`enum` を持つ `direction`・`mode` は一致するが、`subject-kind` の12種・10種は adapter の schema に列挙が無い。この差は広告の粒度の違いであり、実行の受理は第6ラウンドで一致を確認した。
- HTTP・Desktop の discovery は無い。

## 6. 位置付け

ADR-0048 §2.6 が求める「生成または一致試験でずれを検出する」に対して、CLI と MCP の discovery についてだけ最小の一致試験を置いた。全入口の discovery を1つの正本から生成する仕組みではなく、**ずれたら気づく**ための回帰試験である。
