# discovery と実行の突き合わせ（2026-11-09、第6ラウンド）

- 対象: [#3149](https://github.com/hat47x/tei-platform/issues/3149)、[#3419](https://github.com/hat47x/tei-platform/issues/3419)、[#3420](https://github.com/hat47x/tei-platform/issues/3420)、[ADR-0048](../../adr/0048-shared-application-api-and-transport-adapters.md) §2.6。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **read-only の突き合わせ**。新しい欠陥が見つかった場合は起票し、既存 Issue が持つ場合は重複を作らない。

## 1. 目的

ADR-0048 §2.6 は「CLI catalog・MCP tool schema・HTTP schema・Desktop binding が独立した意味の正本にならないよう、生成または一致試験でずれを検出する」と定める。現行版で、`tei commands` の広告と実際の受理が食い違う箇所があるかを確かめた。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `22131acc8373118804af1ce76cdca5d6153740d5`（`origin/main`） |
| CLI実行物 | `sha256:54c519ceab99883cd7b921b4c98a224627a62ac28a1a73121167478db26f0b3b` |
| MCP実行物 | `sha256:80d5c6b0ff505ee54edeb29fe7330fb10abbf00d1a73f724291d63e060882acc` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64` |
| 対象Workspace | `reference/workspace` |

## 3. 生データ

| ファイル | 内容 |
| --- | --- |
| `00-head.txt` | 対象 commit |
| `01-binaries.txt` | 2実行物の SHA-256 |
| `02-commands.txt` | `tei commands`（19件） |
| `03-tools-list.json` | MCP `tools/list`（4 tool） |
| `04-kind-parity.txt` | 広告した subject-kind ごとの `inspect`・`trace` の実行結果 |
| `05-advertised-vs-actual.txt` | 広告したフラグ条件と実際の受理 |

## 4. 観測

### 4.1 二つの発見面

- CLI `commands` は19件で、各 entry が `effect`・`schemaVersion`・`jsonContract`・`required`・`optional`・`argumentValues`・`argumentDefaults`・`writeFlags` を持つ。
- MCP `tools/list` は4 tool で、`inputSchema` の `properties` と `description` だけを持つ。`effect`・`schemaVersion`・`annotations` は無い。応答の `schemaVersion` は `dispatch` が文字列リテラルで持つ。
- 両者を突き合わせる試験は無い。`TestAdapterAdvertisesOnlyReadOnlyOperations` は tool 名4件の集合だけを固定し、`strict_arguments_test.go` は引数の型・未知欄を実行時に検査する。**広告の一致は検査していない**。

### 4.2 subject-kind の広告と実行

`inspect` は12種、`trace` は10種を広告する。差は `semantic-candidate-key` と `semantic-functional-dependency` の2種で、`trace` は投影しない。

実測では、広告された全種に対して、実在しないIDを与えた結果が `SUBJECT_NOT_FOUND`（exit 1）だった。実在するID（`sales.order`・`order_entry`・`order_entry.recent_orders`）では `inspect`・`trace` とも成功した。**広告と実行の食い違いは検出されなかった**。

### 4.3 フラグ条件の広告と実行

| 広告 | 実行 |
| --- | --- |
| `trace` の `-offset and -limit (full only)` | `-mode summary` に `-offset`／`-limit` を付けると `INVALID_ARGUMENT`・exit 2。`-mode full -offset 5`（limit無し）は成功 |
| `validate` の `-json (workspace only)` | `-world … -json` は `INVALID_ARGUMENT`・exit 2 |
| `view` の `-input and -execute with -command` | `-execute` のみは `INVALID_ARGUMENT`・exit 2 |
| `view` の `at most one of -pattern, -package, -command` | `-pattern` と `-package` の同時指定は `INVALID_ARGUMENT`・exit 2 |
| `inspect` は `-workspace` のみ | `-world` を付けると `INVALID_ARGUMENT`・exit 2 |
| `source-index` の `-subject-kind and -subject-id together` | 両方指定で成功（`mode=subject`） |

**広告と実行は一致した**。`effect` は全19件が `read-only`、`writeFlags` を持つのは `view` だけだった。

### 4.4 既存の試験との関係

`tools/reference-mcp-readonly-adapter/validate_parity_test.go` は、実CLIをビルドして `validate -workspace … -json` と MCP `tei_validate` の文書を byte で比べる。**validate の1操作だけが実プロセスで固定されており、catalog・inspect・trace は `operations` の戻り値との比較**である。ADR-0048 §5 の T01〜T07 を4操作すべてで実プロセス比較する試験はまだ無い。

## 5. 課題の扱い（重複を作らない）

| 観測 | 扱い |
| --- | --- |
| discovery に操作ID・入口の欄が無く、CLI と MCP の広告を突き合わせる試験が無い | [#3419](https://github.com/hat47x/tei-platform/issues/3419) の「対応表」で扱う（新規Issueは作らない） |
| 4操作のうち catalog・inspect・trace が実プロセス比較で固定されていない | [#3420](https://github.com/hat47x/tei-platform/issues/3420) CAPI-2a が同じ範囲を持つ（新規Issueは作らない） |
| 失敗分類の統一（`INVALID_SUBJECT` と `-32602`） | #3420 の未決 |
| D-9（設計文書が意味として読まれない） | [#2621](https://github.com/hat47x/tei-platform/issues/2621) の設計判断が前提 |

今回は新しい欠陥を検出しなかったため、起票は行わない。§4.1 の「突き合わせ試験が無い」は #3419 の受入条件として既に含まれる。

## 6. 読み取れないこと（NOT RUN）

- HTTP・Desktop の入口、実ブラウザー・実WebView。
- 実DB・実Provider・実Host、意味の受理、権限、配備。
- MCP の後続 protocol 版、実Harness からの接続。
- 改行の異なる checkout（[#2399](https://github.com/hat47x/tei-platform/issues/2399)）、大規模Workspace（[#1177](https://github.com/hat47x/tei-platform/issues/1177)）。

## 7. 位置付け

この記録は「広告と実行の突き合わせを行い、欠陥ではなく未検査範囲を特定した」結果である。欠陥が無いことは、全入口の同値性や製品適格性を示さない。
