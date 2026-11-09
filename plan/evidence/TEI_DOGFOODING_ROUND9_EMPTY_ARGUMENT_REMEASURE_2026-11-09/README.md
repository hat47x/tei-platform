# 統合後の再実測: 空文字列の引数（2026-11-09、第9ラウンド）

- 対象: [#3605](https://github.com/hat47x/tei-platform/issues/3605)、[#3420](https://github.com/hat47x/tei-platform/issues/3420)。
- 前段: [第8ラウンド](../TEI_DOGFOODING_ROUND8_EMPTY_ARGUMENT_2026-11-09/README.md)。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **read-only の再実測**。意味の受理・権限付与・配備は行っていない。

## 1. 目的

第8ラウンドの修正（PR [#3607](https://github.com/hat47x/tei-platform/pull/3607)）が `main` に統合された後、同じCLI操作とMCP呼出しで解消を確かめる。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `0067d6a1e9a657915cc8943b6f4fa784879c35dc`（PR #3607 の統合コミット、`origin/main` と同一） |
| CLI実行物 | `sha256:b72dc7e692208bc2246311f78f5b8dc614440bc598b02cbaefc816e41fdaa8ff` |
| MCP実行物 | `sha256:b80c6062836da2054d2d8d8206ca17febeaceceac6784d3b07a96864101bf16a` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64` |
| 対象Workspace | `reference/workspace` |

## 3. 生データ

| ファイル | 内容 |
| --- | --- |
| `00-head.txt` | 対象 commit |
| `01-binaries.txt` | 2実行物の SHA-256 |
| `02-empty-parity-after.txt` | 5ケースの CLI と MCP の結果 |

## 4. 観測

| 入力 | CLI | MCP |
| --- | --- | --- |
| `-mode ""` | `INVALID_ARGUMENT`・exit 2 | `-32602` |
| `-direction ""` | `INVALID_ARGUMENT`・exit 2 | `-32602` |
| `-expected-fingerprint ""` | `INVALID_ARGUMENT`・exit 2 | `-32602` |
| 3引数を省略 | `ok:true`・`mode=summary` | 成功・既定どおり |
| `-mode full` | `ok:true`・`mode=full` | 成功 |

空文字列は両入口で拒否され、省略時だけ既定（`summary`）が適用される。**「空文字列を未指定として別の値へ落とす」経路は塞がった**。

残る差は符号だけで、CLI は `INVALID_ARGUMENT`（exit 2）、MCP は `-32602`（プロトコル層）。これは #3420 の失敗分類の統一で扱う未決であり、本ラウンドでは変更していない。

## 5. 読み取れないこと（NOT RUN）

- `model catalog -namespace ""` と `inspect -subject-kind ""`（第8ラウンドで意図的に残した範囲）。
- HTTP・Desktop の入口、実DB・実Provider・実Host、意味の受理・権限・配備。
- MCP 後続 protocol 版、実Harness 接続。

## 6. 位置付け

#3605 の解消を、統合後の版でCLIとMCPの両方から観測した記録である。失敗分類の統一は未達のまま残る。
