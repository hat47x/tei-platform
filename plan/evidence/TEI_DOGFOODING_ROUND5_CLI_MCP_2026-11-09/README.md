# CLI と MCP 参照Adapterの一致と、入力検証の追随（2026-11-09）
# CLI と MCP 参照Adapterの一致と、厳密入力の差（2026-11-09）

- 対象: [#3400](https://github.com/hat47x/tei-platform/issues/3400)、[#1158](https://github.com/hat47x/tei-platform/issues/1158) CAPI-2（[#3420](https://github.com/hat47x/tei-platform/issues/3420)）、[ADR-0048](../../adr/0048-shared-application-api-and-transport-adapters.md) §5 の T01・T04・T05・T07。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **read-only の比較実測**。意味の受理・権限付与・配備は行っていない。

## 1. 目的

第2ラウンドでは、MCP 参照Adapterが `tei.cli.*` を名乗るのに validate の包み方が CLI と違うことを指摘した。その後 `main` に共通投影の修正（#3458）が入ったため、現行版で次を確かめる。
第2ラウンドでは、MCP 参照Adapterの `tei.cli.*` 名乗りと validate の包み方が CLI と違うことを指摘した。その後 `main` に共通投影の修正（#3458）が入ったため、現行版で次を確かめる。

1. 4操作の**値**が CLI と MCP で一致するか。
2. **失敗**（対象不在・未対応kind）の符号と `ok`/`isError` が一致するか。
3. **厳密入力**（未知欄・型不正・null・未知tool）の扱いが両入口で同じか。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source（比較の基準） | `4f97c86cf00de0a8e82d27eefd5398f983eae78b`（当時の `origin/main`） |
| CLI実行物 | `go build ./cmd/tei`、SHA-256 `72d32c7e8d826522bf3f9247e2120cdc43176b8804dee028ae86fd8bcc36d66c` |
| MCP実行物（基準） | `go build ./tools/reference-mcp-readonly-adapter`、SHA-256 `cdebdbd19346e2c2bee03f4dd6789558636a208fcacd2951050ac2f279f5de9f` |
| MCP実行物（再確認） | `be5f832fbaa588dd6efd6876ab22a8910238e784` からビルド、SHA-256 `01431eaddd47f5c513c09a04cf4e8f7e291c0ad1ac7f5f14644f36a507c52573` |
| Source | `hat47x/tei-platform` commit `4f97c86cf00de0a8e82d27eefd5398f983eae78b`（`origin/main`） |
| CLI実行物 | `go build ./cmd/tei`、SHA-256 `72d32c7e8d826522bf3f9247e2120cdc43176b8804dee028ae86fd8bcc36d66c` |
| MCP実行物 | `go build ./tools/reference-mcp-readonly-adapter`、SHA-256 `cdebdbd19346e2c2bee03f4dd6789558636a208fcacd2951050ac2f279f5de9f` |
| MCP protocolVersion | `2024-11-05` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64` |
| 対象Workspace | `reference/workspace`（両実行物とも同じパスを `-workspace` で指定） |

## 3. 生データ

| ファイル | 内容 |
| --- | --- |
| `00-head.txt` | 比較の基準 commit |
| `00-head.txt` | 対象 commit |
| `01-binary-sha256.txt`・`10-mcp-binary-sha256.txt` | 実行物の識別 |
| `02-commands.txt` | `commands`（`-format json` は存在せず、この呼出しは `commands does not accept arguments` を返す） |
| `03-commands-json.txt` | 上記の失敗出力 |
| `12-mcp-requests.jsonl`・`11-mcp-responses.jsonl` | initialize・tools/list・4操作＋2失敗 |
| `13-cli-*.txt` | 同じ6件のCLI応答 |
| `14-mcp-strict-probes.jsonl` | 未知欄・型不正・null・未知tool・必須欠落（基準の実行物） |
| `15-cli-*.txt` | 同じ意図のCLI応答 |
| `b88759643-00-head.txt`・`b88759643-01-binary-sha256.txt`・`b88759643-02-probes.jsonl` | 現行 `main`（`be5f832fb`）の実行物による再確認 |
| `14-mcp-strict-probes.jsonl` | 未知欄・型不正・null・未知tool・必須欠落 |
| `15-cli-*.txt` | 同じ意図のCLI応答 |

## 4. 観測

### 4.1 成功応答の一致（T01）

`validate`・`model catalog`・`inspect`・`trace` の4件で、MCPの `content[0].text` を復号したJSONとCLIの標準出力を、パスの深さ3までで比較した。**キー集合と型に差は0件**だった。`trace` の `snapshotFingerprint`・`sourceIndexFingerprint`・`worldFingerprint` も一致した。
`validate`・`model catalog`・`inspect`・`trace` の4件で、MCPの `content[0].text` を復号したJSONとCLIの標準出力を、パスの深さ3までで比較した。**キー集合と型に差は0件**だった（`schemaVersion`・`ok`・`result` の各下位を含む）。`trace` の `snapshotFingerprint`・`sourceIndexFingerprint`・`worldFingerprint` も一致した。

### 4.2 失敗の一致（T04・T07）

| ケース | CLI | MCP |
| --- | --- | --- |
| 不在のsubject | `ok:false`・`SUBJECT_NOT_FOUND`・exit 1 | `ok:false`・`SUBJECT_NOT_FOUND`・`isError:true` |
| 未対応kind（`semantic-candidate-key`） | `ok:false`・`UNSUPPORTED_SUBJECT_KIND`・exit 2 | `ok:false`・`UNSUPPORTED_SUBJECT_KIND`・`isError:true` |

### 4.3 厳密入力（T05）

基準の実行物では、**未知の引数だけが不一致**だった。

| ケース | CLI | MCP（基準） |
MCPの意味上の失敗は `isError:true` と `ok:false` の両方で示され、文字列の `failure.code` はCLIと同じだった。JSON-RPCのプロトコルエラー（数値code）とは別の欄に分かれている。

### 4.3 厳密入力の差（T05）

| ケース | CLI | MCP |
| --- | --- | --- |
| 未知の引数 | `INVALID_ARGUMENT`・`flag provided but not defined: -extra`・exit 2 | **無視して成功**（`ok:true`） |
| `-limit` に非整数 | `INVALID_ARGUMENT`・`invalid value "many"`・exit 2 | `-32602`・`limit must be an integer` |
| `subjectId` が `null` | （該当なし。空文字は `INVALID_SUBJECT`） | `-32602`・`subjectKind and subjectId are required` |
| 未知のtool | （該当なし） | `-32602`・`unknown tool` |
| 必須欠落 | `INVALID_SUBJECT` | `-32602` |

この差は #3575 として起票した。ただし **`main` では別の変更が同時期に同じ問題を直していた**。`b88759643`（`fix(mcp): 不正な引数を既定値へ置き換えず -32602 で拒否する (#3420 CAPI-2b)`）が `rejectUnknownArguments` を追加しており、基準 `4f97c86cf` には含まれていなかったが、その後の `main` に入った。

### 4.4 現行 main での再確認

`be5f832fb` の実行物 `sha256:01431eadd…` で、基準の実行物が受け入れていた入力を再試行した（`b88759643-02-probes.jsonl`）。

| ケース | 結果 |
| --- | --- |
| 未知の `extra` 欄 | `-32602`・`unknown argument "extra"`。成功しない |
| `workspace` 欄 | `-32602`・`unknown argument "workspace"`。読取り範囲を広げられない |
| `subjectId` が `null` | `-32602`・`subjectId must be a string` |
| `-limit` に非整数 | `-32602`・`limit must be an integer` |
| 既存の `tei_validate`・`tei_inspect` | 変更前と同じく成功 |

未指定と不正指定が区別され、不正値が既定値へ置き換わらないことを確認した。

### 4.5 発見情報

`commands` は19件で、いずれも `name`・`effect`・`schemaVersion`・`jsonContract` を持つ。ADR-0048 §2.6 が求める「操作ID・利用可能な入口」の欄は無く、入口ごとの対応は文書とIssueでしか辿れない。この点は #3419 の対応表で扱う。

## 5. 読み取れないこと（NOT RUN）

- MCP準拠の網羅（`2025-06-18` 等の後続版、通知、取消、進捗、rate limit）。
- 実Harness（外部AI）からの接続。ここではローカルのstdio往復のみ。
- HTTP・Desktopの入口（#3421）。`view -execute` など書込みを伴う経路。
- 実DB・実Provider・権限・本番適格性。

## 6. 課題の扱い

- 未知の引数の差は **#3575**。起票時点では基準の版で再現したが、`b88759643` が同じ問題を扱っており、現行 `main` では再現しない。重複として閉じ、元の担当である #3420 へ戻す。
- 必須欠落の符号（CLI の `INVALID_SUBJECT` と MCP の `-32602`）は #3420 の範囲。今回の変更で「不正値を成功にしない」点は揃ったが、失敗分類の統一は未達である。

## 7. 位置付け

4操作の値と主要な失敗は CLI と MCP で一致した。入力の厳密さも、基準の版では未知欄だけが緩く、現行 `main` では揃っている。**この記録は基準の版と現行 `main` の両方を残し、起票から解消までの経緯を含む**。入口ごとの失敗分類の統一、HTTP・Desktop の同値性は未達である。
| 必須欠落 | `INVALID_SUBJECT` | `-32602`（プロトコルエラー） |

**未知の引数だけが不一致**である。MCP Adapterは `json.Unmarshal` を引数ごとに行い、宣言していない欄を捨てるため、CLI が `INVALID_ARGUMENT` で拒否する入力を成功として扱う。`dispatch` の `optionalString` も、型不正を既定値へ落とさないことは確認できたが、**未知欄の拒否は未実装**である。

また、必須欠落は CLI では `INVALID_SUBJECT`（意味層の符号）、MCP では `-32602`（プロトコル層）になる。これは入口の表現差として説明できるが、**同じ失敗分類へ寄せるかどうかはADR-0048 §2.3「失敗」の未決**である。

### 4.4 発見情報

`models` の `commands` は19件で、いずれも `name`・`effect`・`schemaVersion`・`jsonContract` を持つ。ADR-0048 §2.6 が求める「操作ID・利用可能な入口」の欄は無く、**入口ごとの対応は文書とIssueでしか辿れない**（4.1の一致はテストで固定されているが、discovery からは見えない）。

- MCP準拠の網羅（`2025-06-18` 等の後続版、通知、取消、進捗、課金・rate limit）。
- 実Harness（外部AI）からの接続。ここではローカルのstdio往復のみ。
- HTTP・Desktopの入口。`#3421` の担当。
- `view -execute` など書込みを伴う経路。MCPは4読取り操作のみを公開する。
- 実DB・実Provider・権限・本番適格性。

## 8. 修正後の確認（2026-11-09 追記）

#3575 の修正として、参照Adapterの `dispatch` の先頭に `checkArguments` を置き、tool が宣言していない引数を操作の前に `-32602` で拒否するようにした。修正後の実行物 `sha256:2f919c7e…` で次を確認した（`20-mcp2-binary-sha256.txt`・`21-mcp2-strict-probes.jsonl`）。

- 未知の `extra` 欄: `-32602`・`unknown argument "extra" for tool "tei_inspect"`。成功しない。
- 既存の4操作（`tei_validate`・`tei_inspect`・`tei_trace`・`tei_model_catalog`）: 変更前と同じく `ok:true`。
- 必須欠落: `-32602`（従来どおりプロトコル層の表現）。

回帰試験は `TestAdapterRejectsUndeclaredArguments`（`tools/reference-mcp-readonly-adapter/main_test.go`）で固定した。`go test ./tools/reference-mcp-readonly-adapter -count=1` は ok。

CLI の `INVALID_ARGUMENT`（exit 2）と MCP の `-32602` は、まだ別の層の符号である。同じ失敗分類へ寄せるかは [#3420](https://github.com/hat47x/tei-platform/issues/3420) の未決として残る。今回の修正は「未知の引数を成功として扱わない」という一点を揃えたもので、分類の統一ではない。

## 9. 起票する課題

- **未知の引数をMCPだけが無視する**（T05の不一致）。CLIと同じ厳密さを持たせるか、逆にCLIを緩めるかを決め、負例で固定する。→ 起票。
- 必須欠落の符号を意味層へ寄せるか（`INVALID_SUBJECT` と `-32602`）。→ 既存 #3420 の範囲として整理する（新規Issueは作らない）。
- discovery に操作ID・入口の欄が無い。→ 既存 #3419 の「対応表」で扱う。

4操作の値と主要な失敗は CLI と MCP で一致した。ただし厳密入力の未知欄だけが不一致で、これは「MCPは同じ操作を呼ぶが、入力の検証規則は別」という状態を示す。ADR-0048 が禁じた「入口ごとの意味の再実装」には当たらないが、**入力契約の共通化は未達**である。
