# 空文字列の引数が既定値と食い違う（2026-11-09、第8ラウンド）

- 対象: [#3605](https://github.com/hat47x/tei-platform/issues/3605)、[#3420](https://github.com/hat47x/tei-platform/issues/3420)、[ADR-0048](../../adr/0048-shared-application-api-and-transport-adapters.md) §2.3・§2.4・§5 T05。
- 前提: [第7ラウンド](../TEI_DOGFOODING_ROUND7_CROSS_SURFACE_2026-11-09/README.md)までで、discovery と実行の整合を確認した。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **read-only の入出力比較と、1操作の最小修正**。

## 1. 目的

ADR-0048 §2.4 は「既定値の適用は共通契約で定め、正常系と同じく入口間で比較する」と定める。CLI と MCP で、空文字列の引数が同じ意味になるかを確かめた。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source（比較の基準） | `a338d3d048e1afb78b574c2b195b49676cdbafa2`（`origin/main`） |
| CLI実行物 | `sha256:0dd2a8bb2546ad64e6c3132cc81a798707b151f5dc1497a0d4e74ffe66973966` |
| MCP実行物 | `sha256:a5e94cbee8b7082619b55331579aabf1b53070f70683d37cac74d2cac8a4f5b2` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64` |
| 対象Workspace | `reference/workspace` |

## 3. 生データ

| ファイル | 内容 |
| --- | --- |
| `00-head.txt`・`01-binaries.txt` | 対象 commit と2実行物の識別 |
| `02-commands.txt` | `tei commands`（`argumentDefaults` を含む） |
| `03-enum-parity.txt` | `direction`・`mode` の値集合と既定値の比較 |
| `04-defaults.txt` | 省略時の `direction`・`mode` |
| `05-schema-enums.json` | MCP の `inputSchema` にある `enum` |
| `06-cli-empty-mode.txt`・`07-cli-empty-direction.txt`・`08-cli-empty-kind.txt`・`09-cli-empty-scope.txt` | 空文字列を渡した実行 |
| `10-empty-string-parity.txt` | 5つの引数について CLI と MCP を並べた結果 |
| `11-omitted-mode.txt` | `-mode` を省略した実行 |

## 4. 観測（修正前）

### 4.1 値集合と既定値は一致

`direction`（`outgoing`・`incoming`・`both`）と `mode`（`summary`・`full`）は、MCP の `inputSchema.enum` と CLI の `argumentValues` が同じ値を持つ。省略時の既定も両方 `direction=both`・`mode=summary` で一致した（`03-enum-parity.txt`・`04-defaults.txt`）。

### 4.2 空文字列だけが食い違う

| 入力 | CLI | MCP |
| --- | --- | --- |
| `-mode ""` | `ok:true`・結果の `mode` は **`full`**・rows=4 | `-32602`・`mode must not be an empty string` |
| `-direction ""` | `ok:true`・`direction=both` | `-32602`・`direction must not be an empty string` |
| `-expected-fingerprint ""` | `ok:true`・照合を省略 | `-32602`・`expectedFingerprint must not be an empty string` |
| `model catalog -namespace ""` | `ok:true` | `-32602`・`namespace must not be an empty string` |
| `inspect -subject-kind ""` | `INVALID_SUBJECT`・exit 2 | `-32602`・`subjectKind must not be an empty string` |

**`-mode ""` が最も危険**である。CLI の flag 既定は `summary`（`cmd/tei/main.go`）だが、`internal/modelref/trace.go` の `normalizeTraceRequest` が `options.Mode == ""` を `full` に置き換える。空文字列を渡した呼出し元は「既定で動く」と読むが、実際は行を返す `full` で実行される。省略時は `summary`（rows=0）で、**同じ「指定なし」のつもりが別の結果になる**。

## 5. 修正

`cmd/tei/main.go` に `setStringFlag` を入れ、`-direction`・`-mode`・`-expected-fingerprint` について「フラグを書いたか」を保持するようにした。書いたうえで値が空なら、操作を呼ぶ前に `INVALID_ARGUMENT`（exit 2）で拒否する。省略時は従来どおり既定（`both`・`summary`）を使う。

修正後の実測:

| 入力 | 結果 |
| --- | --- |
| `-mode ""` | `INVALID_ARGUMENT`・exit 2 |
| `-direction ""` | `INVALID_ARGUMENT`・exit 2 |
| `-expected-fingerprint ""` | `INVALID_ARGUMENT`・exit 2 |
| `-mode` 省略 | `ok:true`・`mode=summary` |
| `-mode full` | `ok:true`・`mode=full` |
| `-mode summary` | `ok:true`・`mode=summary` |

回帰試験は `cmd/tei/cli_trace_empty_argument_test.go` の `TestTraceRejectsExplicitlyEmptyArguments` と `TestTraceOmittedArgumentsUseTheAdvertisedDefaults`。`go test ./cmd/tei -count=1` は ok。

## 6. 残した範囲

- `model catalog -namespace ""` と `inspect -subject-kind ""` は**変更していない**。前者は「名前空間の絞り込み無し」という意味があり、CLI の既定と一致するため、拒否へ変えるかは既存の読み手への影響を伴う判断になる。後者は CLI が `INVALID_SUBJECT` で拒否しており、「拒否する」点は MCP と同じで、符号だけが違う（#3420 の失敗分類）。
- `-mode ""` を明示的に許す設計にする可能性は残る。その場合は「既定と実適用値を一致させ、規則を `commands` に書く」ことが条件になる。

## 7. 読み取れないこと（NOT RUN）

- HTTP・Desktop の入口、実ブラウザー・実WebView。
- 実DB・実Provider・実Host、意味の受理、権限、配備。
- MCP の後続 protocol 版、実Harness からの接続。

## 8. 位置付け

「空文字列を未指定として既定へ落とす」規則が、入口と操作で揃っていないことを確認した。最も結果の意味を変える `trace` の3つの引数だけを拒否へ揃え、残りは既存の担当（#3420・#3605）へ残した。この修正は意味の受理や権限を与えない。
