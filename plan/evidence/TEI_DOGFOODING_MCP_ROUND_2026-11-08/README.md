# TEIのread-only MCP AdapterによるCLI/MCP一致の実測（2026-11-08）

- 対象課題: [#3149 TEIドッグフーディング](https://github.com/hat47x/tei-platform/issues/3149)、[#3400 MCP入口の欠如](https://github.com/hat47x/tei-platform/issues/3400)。
- 参照実装: [tools/reference-mcp-readonly-adapter](../../../tools/reference-mcp-readonly-adapter/README.md)。
- 実行日: 2026-11-08（Asia/Tokyo）。
- 状態: **ローカルの参照実装に対する read-only の実測**。製品のMCP契約、実Host、認可、外部送出、意味の受理は示さない。

## 1. この実測の位置づけ

第2ラウンド（[TEI_DOGFOODING_ROUND2_2026-11-08](../TEI_DOGFOODING_ROUND2_2026-11-08/README.md)）で、TEIにMCP入口が無いことを確認した。そこで、**「MCP Adapter を作る場合、CLIと同じ結果を返せるか」だけを、参照実装で確かめる**。方針の採否（MCP入口を作るか）は [#3400](https://github.com/hat47x/tei-platform/issues/3400) の判断に委ねる。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `hat47x/tei-platform` commit `20605b79ebb529a9e3706d94bc5f872f4ade8143`（実測開始時点の `origin/main`） |
| CLI実行物 | 同commitから `go build ./cmd/tei`。SHA-256 `8fa2b4e011b531d66eb0d3f33523e3e40b424a073688f11a8f8e70328431cc07`、`vcsRevision: 20605b79e…` |
| MCP Adapter実行物 | 同commitの `go build ./tools/reference-mcp-readonly-adapter`。同一worktree内のソースから作った単一実行ファイル |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（`GOTOOLCHAIN=go1.26.10` も併用） |
| Workspace | `reference/workspace`（Adapter の起動時に固定） |

## 3. 生データ

| ファイル | 内容 |
| --- | --- |
| `cli-01-commands.txt` | CLIの `commands`（19件、MCP関連の名前は無い） |
| `cli-model-catalog.json`・`cli-validate.json`・`cli-inspect.json`・`cli-trace.json` | CLI側の値 |
| `mcp-requests.jsonl` | Adapterへ送った8要求（initialize・通知・tools/list・4つのtool・未対応kind） |
| `mcp-responses.jsonl` | Adapterが返した応答 |

## 4. 観測

### 4.1 CLIとMCPの値一致

同じ Workspace に対して、CLI と MCP Adapter の応答を項目ごとに比べた。

| 比較 | 結果 |
| --- | --- |
| `model catalog` の `worldFingerprint`・`sourceIndexFingerprint`・`declarationCounts` | 一致 |
| `validate` の `worldFingerprint`・`sourceIndexFingerprint` | 一致 |
| `inspect` の `worldFingerprint`・`canonical`・`sourceStatus` | 一致 |
| `trace` の `snapshotFingerprint`・`total`・`authority` | 一致 |
| `schemaVersion`（`tei.cli.model/v0`・`tei.cli.validate/v0`・`tei.cli.inspect/v0`・`tei.cli.trace/v0`） | 一致 |
| 不一致 | **0件** |

### 4.2 MCPの手順と失敗の返し方

- `initialize` は `protocolVersion: 2024-11-05`、`serverInfo.name: tei-reference-mcp-readonly-adapter` を返した。
- `notifications/initialized` には**応答を返さなかった**（8要求に対して7応答）。
- `tools/list` は `tei_model_catalog`・`tei_inspect`・`tei_trace`・`tei_validate` の4つを返した。
- 未対応の対象種類（`not-a-kind`）は `isError: true` で、`error.message` の先頭に `UNSUPPORTED_SUBJECT_KIND` を返した。**成功の空結果にはしなかった**。

### 4.3 Adapter の境界

- tool の入力スキーマに `workspace`・`path`・`root`・`file` を含めない。読取り範囲は起動時の `-workspace` に固定される。
- 書込み・受理・配備・付与・実行の tool は無い。
- 認可・開示範囲は Adapter では判定しない。

## 5. 読み取れること

- TEI の `internal/operations` は transport 非依存のままで、**CLI と同じ操作を MCP から呼んでも値が一致する**。MCP Adapter を足す場合に、第二の正本解釈・検証規則を作らずに済む見込みがある。
- 失敗の返し方（`isError` と `error.code`）も、CLI の `ok/error` と同じ符号で伝えられる。

## 6. 読み取れないこと（NOT RUN）

- 製品としての MCP 契約、後方互換、MCP の版ごとの差。
- 実 Host・実 Harness からの接続、認可、開示範囲、外部送出、期限・再接続・同時実行。
- `go test ./...` 全体、CI の再現。
- Adapter を製品へ採用するかの判断（[#3400](https://github.com/hat47x/tei-platform/issues/3400)）。

## 7. 再実行手順

```bash
git worktree add --detach /tmp/tei-r3 20605b79ebb529a9e3706d94bc5f872f4ade8143
cd /tmp/tei-r3
go build -o /tmp/tei-r3-cli ./cmd/tei
go build -o /tmp/tei-mcp ./tools/reference-mcp-readonly-adapter

# CLI 側
/tmp/tei-r3-cli model catalog -workspace reference/workspace
/tmp/tei-r3-cli validate -workspace reference/workspace -json

# MCP 側（1行1要求）
/tmp/tei-mcp -workspace reference/workspace < plan/evidence/TEI_DOGFOODING_MCP_ROUND_2026-11-08/mcp-requests.jsonl
```

## 8. 位置付け

この記録は、**参照実装に対するローカル実行の観測**である。CLIとMCPの一致は、同じ `internal/operations` を呼ぶ限りで成立するという範囲の主張であり、製品のMCP採用・認可・外部送出・本番適格性を示さない。
