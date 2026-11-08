# TEI read-only MCP Adapter（参照実装）

MCP のクライアントから、既存のTEIの読み取り操作を使えるようにする**薄い Adapter** です。`internal/operations` を CLI と同じ形で呼び、**第二の正本解釈・知識の保存先・検証規則・権限モデルを作りません**（[全体構成](../../docs/architecture.md)の制約、[#3400](https://github.com/hat47x/tei-platform/issues/3400)）。

## なぜ置くか

[#3400](https://github.com/hat47x/tei-platform/issues/3400) で、TEIには MCP 入口が無く、`CLI/MCP` を主体とするドッグフーディング（[#3149](https://github.com/hat47x/tei-platform/issues/3149)）のうち MCP 経路を実測できないことを確認しました。この参照実装は、**入口を作る場合の最小形**を実際に動かして確かめるための材料です。製品採用を決めたものではありません。

## 動かす

```bash
go build -o /tmp/tei-mcp ./tools/reference-mcp-readonly-adapter
/tmp/tei-mcp -workspace reference/workspace
```

標準入出力で JSON-RPC（1行1要求）を扱います。

```text
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}
{"jsonrpc":"2.0","method":"notifications/initialized"}
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"tei_validate","arguments":{}}}
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"tei_inspect","arguments":{"subjectKind":"semantic-data","subjectId":"sales.order"}}}
```

## 公開する操作

| tool | 呼ぶ操作 | CLI の対応 |
| --- | --- | --- |
| `tei_model_catalog` | `operations.CatalogWorkspace` | `tei model catalog` |
| `tei_inspect` | `operations.InspectSubject` | `tei inspect` |
| `tei_trace` | `operations.TraceWorkspace` | `tei trace` |
| `tei_validate` | `operations.ValidateWorkspace` | `tei validate -workspace ... -json` |

各 tool の結果は `content[0].text` に、CLI と同じ `schemaVersion`・`ok`・`result`／`error` の**JSON 1件**として入ります。読取りに失敗したときは `isError: true` を付け、`error.message` の先頭に `error.code` を置きます。

## 境界

- **Workspace は起動時に固定**します。tool の入力に path を取らないため、モデルが読取り範囲を広げられません。
- 書込み・受理・配備・付与・実行の tool はありません。`tools/list` を試験で固定し、名前にもそれらの語を含めません。
- 認可・開示範囲・秘匿は、この Adapter では判定しません。呼び出し元の Host とTEIの既存の境界に委ねます。
- 期限・同時実行・再接続は扱いません。1プロセス・逐次の参照経路です。
- MCP のプロトコル版は `2024-11-05` を名乗ります。後の版で同じ挙動になるとの主張ではありません。

## 検証

`main_test.go` が次を検査します。

- 公開する tool が上の4つだけで、名前に書込み系の語を含まないこと
- 4つの tool の結果が、同じ Workspace に対する `operations` の戻り値と一致すること（`worldFingerprint`・`sourceIndexFingerprint`・`snapshotFingerprint`・`schemaVersion`）
- 未対応の対象種類が `isError: true` と `UNSUPPORTED_SUBJECT_KIND` で返ること
- `initialize` の応答と、通知に返答しないこと
- tool の入力スキーマに `workspace`・`path`・`root`・`file` を含めないこと

実行:

```bash
go test ./tools/reference-mcp-readonly-adapter -count=1
```

## 実測記録

[mcp経路の実測（2026-11-08）](../../plan/evidence/TEI_DOGFOODING_MCP_ROUND_2026-11-08/README.md)に、CLI との値一致の生出力を残しています。**CLIとMCPの一致は、この参照実装に対するローカル実行の結果**であり、製品の正式な MCP 契約、実 Host、認可、外部送出を示しません。
