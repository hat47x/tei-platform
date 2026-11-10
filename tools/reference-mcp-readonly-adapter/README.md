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
| `tei_model_locate` | `operations.LocateWorkspace` | `tei model locate` |
| `tei_model_overview` | `operations.OverviewWorkspace` | `tei model overview` |
| `tei_inspect` | `operations.InspectSubject` | `tei inspect` |
| `tei_trace` | `operations.TraceWorkspace` | `tei trace` |
| `tei_validate` | `operations.ValidateWorkspace` | `tei validate -workspace ... -json` |

各 tool の結果は `content[0].text` に、CLI と同じ `schemaVersion`・`ok`・`result`／`error` の**JSON 1件**として入ります。`error.code` は CLI と同じ**文字列**です（JSON-RPC の数値 `error` とは別に保ちます）。読取りに失敗したときは `isError: true` を付けます。

`tei_validate` は `result` で包まず、CLI の `validate -workspace ... -json` と同じ最上位の形（`diagnostics`・`worldFingerprint`・`sourceIndexFingerprint`・`declarationCounts`・`workspace`）を返します。検証の不合格も `isError: true` です。文書の組み立ては `internal/operations` の1つの投影を CLI と共有します（[#3420](https://github.com/hat47x/tei-platform/issues/3420)）。

## 入力の検査

実行側でも `arguments` を検査します（`inputSchema` の宣言だけに頼りません）。次は JSON-RPC の `-32602`（invalid params）で拒否し、既定値へ置き換えません。

- tool の入力に宣言のない欄（`workspace`・`path`・`root` を含む）
- 文字列の欄に、文字列以外・`null`・空文字列を渡したとき
- `offset`・`limit` に、整数以外・`null` を渡したとき

欄を省いた場合だけ既定値（`direction=both`、`mode=summary` など）を使います。未指定と不正指定は区別されます（[#3420](https://github.com/hat47x/tei-platform/issues/3420)）。
## 入力の扱い

tool が宣言していない引数は、操作を呼ぶ前に **`-32602` で拒否**します。CLI が未定義の flag を `INVALID_ARGUMENT` で拒否するのに対し、この Adapter が未知の欄を捨てて成功すると、同じ呼出しが入口によって成功と失敗に分かれます（[#3575](https://github.com/hat47x/tei-platform/issues/3575)）。引数の一覧は `main.go` の `toolArguments` が持ち、`dispatch` の先頭で照合します。

型不正（`-32602`）・`null`（`-32602`）・未知の tool（`-32602`）も同じく操作の前に拒否します。意味上の失敗（対象不在・未対応kind）は `result.content` の中の `ok:false` と `error.code`、および `isError:true` で返し、プロトコルエラーと混ぜません。

## 境界

- **Workspace は起動時に固定**します。tool の入力に path を取らないため、モデルが読取り範囲を広げられません。
- 書込み・受理・配備・付与・実行の tool はありません。`tools/list` を試験で固定し、名前にもそれらの語を含めません。
- 認可・開示範囲・秘匿は、この Adapter では判定しません。呼び出し元の Host とTEIの既存の境界に委ねます。
- 期限・同時実行・再接続は扱いません。1プロセス・逐次の参照経路です。
- 1回の要求は1行で、上限は4 MiB です。超えた行は `-32600`（invalid request）で拒否し、その行だけを読み捨てて次の要求の処理を続けます（`bufio.Scanner` の行長超過がプロセス全体を止めていたのを直しました）。
- 要求の封筒（JSON-RPC 2.0）は読込み時に検査します。JSONとして読めない行は `-32700`、JSONでも要求として成り立たない行は `-32600` で1回だけ応答し、次の行の処理を続けます。`-32600` にするのは、オブジェクトでない行（バッチを含む）、`jsonrpc` が文字列 `"2.0"` でない、`method` が空または文字列でない、`params` がオブジェクト・配列・`null` のいずれでもない、`id` が文字列でも数値でもない（MCP は `null` を禁じる）場合です。`id` を返せない応答の `id` は `null` です。`id` のない通知は、不正でも応答しません。
- MCP のプロトコル版は `2024-11-05` を名乗ります。後の版で同じ挙動になるとの主張ではありません。

## 検証

`main_test.go` が次を検査します。

- 公開する tool が上の6つだけで、名前に書込み系の語を含まないこと
- 6つの tool の結果が、同じ Workspace に対する `operations` の戻り値と一致すること（`worldFingerprint`・`sourceIndexFingerprint`・`snapshotFingerprint`・`schemaVersion`）
- `tei_validate` の文書が、実 CLI の `validate -workspace ... -json` と同じ欄と同じ値を持ち、`isError` と `ok` が対応すること（`validate_parity_test.go`）
- 未対応の対象種類が `isError: true` と `UNSUPPORTED_SUBJECT_KIND` で返ること
- `inspect`・`trace` が、実CLIと実MCPの別プロセスで同じ値と同じ `error.code` を返すこと（成功2件・対象不在・未知kind・古い指紋・負のoffset）。`ok` と `isError` の反転一致も含む（`inspect_trace_parity_test.go`）
- 必須引数の欠落が**両方拒否**されること。CLI は `INVALID_ARGUMENT`（#3733 で統一）、MCP は `-32602` で、層が違うだけである。CLI の符号が 1 つであることと、`INVALID_SUBJECT` が「値は渡されたが使えない」に残ることは `cmd/tei/cli_failure_classification_test.go` が固定する（#3420 の層の統一は未着手）
- 広告する6 tool が、実CLIの `commands` と一致すること（対応するコマンドの存在・`effect` が read-only・`writeFlags` が無い・`schemaVersion`・`jsonContract`）。tool の説明が対応する CLI 呼出しを名指しすること（`discovery_parity_test.go`）
- 宣言していない引数が結果を返さず `-32602` で拒否されること（`TestAdapterRejectsUndeclaredArguments`）
- `initialize` の応答と、通知に返答しないこと
- 要求の封筒の不正（不正JSON、非オブジェクト、`jsonrpc`・`method`・`params`・`id` の型不正、`id: null`）を `-32700`／`-32600` で1回だけ拒否して処理を続けること。`id` を返せない応答は `id: null`。不正な通知に応答しないこと。`tools/call` の `params`・`name`・`arguments` 欠落や型不正が `-32602` になること（`envelope_test.go`）
- tool の入力スキーマに `workspace`・`path`・`root`・`file` を含めないこと
- 型不正・`null`・空文字列・宣言のない欄を `-32602` で拒否し、省略時だけ既定値を使うこと（`strict_arguments_test.go`）

実行:

```bash
go test ./tools/reference-mcp-readonly-adapter -count=1
```

## 実測記録

[mcp経路の実測（2026-11-08）](../../plan/evidence/TEI_DOGFOODING_MCP_ROUND_2026-11-08/README.md)と[CLIとMCP参照Adapterの一致と厳密入力（2026-11-09）](../../plan/evidence/TEI_DOGFOODING_ROUND5_CLI_MCP_2026-11-09/README.md)に、CLI との比較の生出力を残しています。**CLIとMCPの一致は、この参照実装に対するローカル実行の結果**であり、製品の正式な MCP 契約、実 Host、認可、外部送出を示しません。
