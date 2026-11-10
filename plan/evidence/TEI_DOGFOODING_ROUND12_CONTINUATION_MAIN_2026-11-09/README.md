# main の trace 継続トークンを実測（2026-11-09、第12ラウンド）

- 対象: [#3625](https://github.com/hat47x/tei-platform/issues/3625)、[#3420](https://github.com/hat47x/tei-platform/issues/3420) T06。
- 前段: [第11ラウンド](../TEI_DOGFOODING_ROUND11_TRACE_CONTINUATION_2026-11-09/README.md)（同じ問題を独立に検出）。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **read-only の再実測**。

## 1. 目的

`main` で #3625 が解消された後、`trace -mode full` の応答と継続の挙動を確かめる。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `5d5d5de6b0e43b677b1ffb87217654ce5c2b9aad`（`origin/main`） |
| CLI実行物 | `sha256:fc83a50671af2eed9645d3f551f484a99275de975e2930b6fddb9ec67f2ace36` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64` |
| 対象Workspace | `reference/workspace`（`semantic-data:sales.order`） |

## 3. 生データ

| ファイル | 内容 |
| --- | --- |
| `00-head.txt`・`01-binary-sha256.txt` | 対象 commit と実行物 |
| `02-page1.json`・`03-token.txt` | 1ページ目と、`continuation.fingerprint` |
| `04-page2.json` | その値で継続した2ページ目 |
| `05-page2-graph.json` | `graphFingerprint` で継続しようとした場合 |

## 4. 観測

| 位置 | 値（先頭24文字） |
| --- | --- |
| `result.snapshotFingerprint` | `sha256:420e7a2002dcc093f…` |
| `result.trace.graphFingerprint` | `sha256:706c056c5b13b34ed…` |
| `result.trace.snapshotFingerprint`（非推奨の別名） | 同上 |
| `result.trace.continuation.fingerprint` | `sha256:420e7a2002dcc093f…` |
| `result.trace.continuation.field` | `result.snapshotFingerprint` |
| `result.trace.nextOffset` | `2` |

| `-expected-fingerprint` に渡した値 | 結果 |
| --- | --- |
| `continuation.fingerprint` | `ok:true`・rows=2 |
| `graphFingerprint` | `STALE_SNAPSHOT`・exit 1 |
| 存在しない指紋 | `STALE_SNAPSHOT`（第11ラウンドで確認） |

`graphFingerprint` と別名 `snapshotFingerprint` は同じ値で、どちらも継続には使えない。
`continuation` は `nextOffset` があるページにだけ付く。

## 5. 読み取れないこと（NOT RUN）

- `-scope`・`-direction` を変えた場合の拒否。
- MCP の `tei_trace` が `continuation` を返すか。
- 別名 `snapshotFingerprint` の削除（既存の読み手の移行が前提）。
- HTTP・Desktop、実DB・実Provider、意味の受理・権限・配備。

## 6. 位置付け

#3625 の解消を、統合後の版でCLIから観測した記録である。`continuation` を選べば継続でき、
`graphFingerprint` は継続に使えないことが、応答の中の名前と `field` で分かる。
