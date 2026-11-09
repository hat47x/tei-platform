# CLI と MCP の結果・失敗・成否の一致（#3420 CAPI-2 の実測）

- 記録日: 2026-11-09
- 対象: `cmd/tei`（CLI）と `tools/reference-mcp-readonly-adapter`（MCP 参照 Adapter）
- 関連: #3420（CAPI-2）、#3419（CAPI-1）、#1158、ADR-0048 §2.3・§5

## 何を確かめたか

#3420 の受入条件のうち、次の4点を**実実行物どうしで**比較しました。

1. 正常系の完全な共通結果（差を許す欄があるなら理由付きで記録）
2. warning・意味エラー・構文エラー・source-root error・重複・load 失敗の全件比較
3. `isError`・CLI exit・`ok` の対応表
4. root 指定の固定（model 入力で別 Workspace へ広げられないこと）

## 対象版

`00-binaries.txt` に、commit（`c79934de6`）、両実行物の SHA-256、`tei version -json` の出力を残しています。Windows 11、go1.27.0、`GOMAXPROCS` 既定。

## 結果

### 正常系（`02-success-parity.json`）

`catalog`・`inspect`・`trace`（summary）の3操作で、CLI の応答文書と MCP の `content[0].text` が**完全一致**しました（`documents_equal: true`）。許容した差はありません（`allowed_differences: []`）。時刻・request ID のような差を許す欄が無いためです。

### validate の失敗分類（`01-validate-matrix.json`）

| 入力 | CLI exit | CLI `ok` | MCP `isError` | 文書一致 | `error.code` 一致 |
| --- | ---: | --- | --- | --- | --- |
| `reference/workspace` | 0 | true | なし | 一致 | — |
| 整形式の合成 Workspace | 0 | true | なし | 一致 | — |
| 構文エラー | 1 | false | true | 一致 | 一致 |
| 存在しない source root | 1 | false | true | 一致 | — |
| 重複宣言 | 1 | false | true | 一致 | 一致 |
| 存在しない Workspace | 2 | false | true | 一致 | 一致 |

すべての行で `cli_exit_matches_ok` と `isError_matches_ok` が `true` です。つまり「検証不合格が成功の空結果になる」経路は残っていません。

**warning のみの場合は、成功（exit 0・`isError` なし・`ok:true`）**です。`reference/workspace` は3件の warning を持ちますが成功として扱われます。

### 封じ込め（`03-confinement.json`）

固定した Workspace の外を指す入力を5通り試し、いずれも**外部のバイト列は応答に現れません**でした。

| 試行 | 結果 |
| --- | --- |
| `arguments.workspace` で別 Workspace | `-32602` |
| `arguments.root` | `-32602` |
| `arguments.path` で `../outside/secret.txt` | `-32602` |
| `subjectId` に `../../../secret` | `SUBJECT_NOT_FOUND` |
| `namespace` に `../../outside` | `NAMESPACE_NOT_FOUND` |

`leaked_outside_bytes` は全件 `false` です。

## 既知の未対応（この記録の範囲外）

- 実 Host・認可・外部送出は NOT RUN。ここで確かめたのはローカル標準入出力の参照経路だけです。
- `-expected-fingerprint` の継続は別 Issue（#3625、main で修正済み）で扱っています。
- JSON-RPC 封筒の不正（`jsonrpc`・`id`・`method`）と `tools/call` の params 検査は別 PR（#3688・#3668）で扱っています。
- 全 MCP protocol 版、実 Harness 接続、本番適格性は扱っていません。

## 再現

`01`〜`03` は、`reference/workspace` と一時ディレクトリに作った合成 Workspace に対して、CLI と MCP Adapter を別プロセスで起動して得た出力です。同じ commit と同じ入力から同じ文書になることを `documents_equal` で示しています。
