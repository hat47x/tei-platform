# 継続トークンの失効条件を実測（2026-11-09、第16ラウンド）

- 対象: [#3420](https://github.com/hat47x/tei-platform/issues/3420) CAPI-2c、[#3625](https://github.com/hat47x/tei-platform/issues/3625)、[ADR-0048](../../adr/0048-shared-application-api-and-transport-adapters.md) §5 T06。
- 前段: [第11・12ラウンド](../TEI_DOGFOODING_ROUND11_TRACE_CONTINUATION_2026-11-09/README.md)。そこでは「どの値が継続トークンか」を扱い、失効条件は未検査と記録した。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **read-only の実測**。新しい欠陥は検出しなかった。

## 1. 目的

第11ラウンドで「`-scope`・`-direction` を変えた場合の拒否は未確認」「MCP が `continuation` を返すかは未実測」と残した。このラウンドで埋める。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `381ce9dcb64dfe20cfc9f45f02953c3f6b2fc84b`（`origin/main`） |
| CLI実行物 | `sha256:3a8d57b68d474eab856a5d0f3bb7c6788e14cde690d8b7af45e81764b4d96c04` |
| MCP実行物 | `sha256:f3db03ff83410dc1e97241c78c905d6e390539016ae9f0cc8683837f6dc5c5f4` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |
| 対象Workspace | `reference/workspace` の複製（`semantic-data:sales.order`、全4行） |

## 3. 生データ

| ファイル | 内容 |
| --- | --- |
| `00-head.txt`・`01-binary.txt` | 対象 commit と CLI 実行物 |
| `10-page1.json`・`11-token.txt`・`12-offset.txt` | 1ページ目と継続トークン |
| `13-comment-continue.json` | コメントだけの変更後に同じトークンで継続 |
| `14-semantic-continue.json` | 属性追加（意味の変更）後に同じトークンで継続 |
| `20-page1.json`・`21-token.txt`・`22-offset.txt` | 選択子テスト用の1ページ目 |
| `23-control.json`〜`26-mode.json` | 同じ選択子／`-scope`／`-direction`／`-mode` の変更 |
| `30-mcp-binary.txt`・`31-mcp-token.txt`・`32-mcp-offset.txt` | MCP 実行物とトークン |
| `33-mcp-page1.json`・`34-mcp-page2.json` | MCP の1ページ目と継続 |
| `40-summary.json`・`41-summary-token.txt`・`42-summary-to-full.json`・`43-full-to-summary.json` | `summary` ⇄ `full` の相互継続 |

## 4. 観測

### 4.1 Source の変更は種類を問わず失効させる

| 変更 | 同じトークンでの継続 |
| --- | --- |
| コメント行を1行追加（意味は不変） | `STALE_SNAPSHOT`・exit 1 |
| 属性を1つ追加（意味が変化） | `STALE_SNAPSHOT`・exit 1 |

`plan/design/CLI_FIRST_INTERFACE_DESIGN.md` §159 は「非意味的な原文変更は拒否する」と明記しており、**実測はそれと一致した**。Source の生バイトが `sourceIndexFingerprint` に入るため、コメント1行でも版が変わる。安全側の設計である。

### 4.2 選択子の変更は失効させる

| 変更 | 結果 |
| --- | --- |
| 同じ選択子（対照） | `ok:true` |
| `-scope` を `c2c` → `other` | `STALE_SNAPSHOT`・exit 1 |
| `-direction` を `both` → `outgoing` | `STALE_SNAPSHOT`・exit 1 |
| `-mode full` → `summary` | `INVALID_ARGUMENT`・exit 2（`summary` は `-offset` を受け付けない） |

第11ラウンドで未確認だった `-scope`・`-direction` の拒否を確認した。`scope` は `modelref` のダイジェストに含まれないが、`operations` 側の外側の値が World と Source Index を含むため、実測では拒否される。

### 4.3 `summary` ⇄ `full` は相互に許容される

| 手順 | 結果 |
| --- | --- |
| `summary` のトークンで `mode=full -offset 0 -limit 2` | `ok:true`・rows 2件・`nextOffset: 2` |
| 同じトークンで `mode=summary` | `ok:true`・total 4 |

`plan/design/CLI_FIRST_INTERFACE_DESIGN.md` §159 の「`summary→full` や offset/limit だけの変更は同じWorkspace状態であれば許容」と一致した。`mode`・`offset`・`limit` はトークンの対象外である。

### 4.4 MCP は CLI と同じ結果になる

| 操作 | MCP |
| --- | --- |
| 1ページ目 | `continuation.fingerprint` と `field=result.snapshotFingerprint` を返す |
| `continuation.fingerprint` で継続 | `ok:true`・rows 1件 |
| `graphFingerprint` で継続 | `STALE_SNAPSHOT` |
| `-scope` を変えて継続 | `STALE_SNAPSHOT` |

第11ラウンドで未実測だった「MCP が `continuation` を返すか」を確認した。**CLI と MCP で継続の可否が一致する。**

## 5. 起票しない理由

新しい欠陥を検出しなかった。すべて文書（`plan/design/CLI_FIRST_INTERFACE_DESIGN.md` §159、`docs/getting-started.md`）に書かれた挙動と一致している。重複Issueを作らない方針に従い、**CAPI-2c の実測記録として残す**。

## 6. 読み取れないこと（NOT RUN）

- [main #3477](https://github.com/hat47x/tei-platform/pull/3477) の World 同一/SourceIndex 改版、[#3479](https://github.com/hat47x/tei-platform/pull/3479) の Consumer 二FP不変／外部Provider raw改版（#3420 CAPI-2c が挙げる別の故障モード）。
- 読取り中の原子的スナップショット（読取前後の独立 Witness）。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 7. 位置付け

継続トークンの失効条件を、Source の変更・選択子の変更・`mode` の相互変換・MCP の4軸で確認した。すべて文書どおりで、**この領域に新しい欠陥は無い**。CAPI-2c のうち「参照実装の継続トークン」はここで一区切りとする。
