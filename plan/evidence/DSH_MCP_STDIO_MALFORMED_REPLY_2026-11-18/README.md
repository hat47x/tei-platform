# 壊れたstdio応答は「子が終了するか」で別のエラーになる（2026-11-18、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-18（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.3`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [stdioの呼び出し中の死](../DSH_MCP_STDIO_MIDCALL_2026-11-17/README.md)（途中で落ちる場合）。本記録は、そこで未確認とした「**部分的に応答してから落ちる場合**」を埋める。

## 1. 何をしたか

`tools/call`に対して**壊れた応答**を書く子を用意し、**書いた後に終了する場合**と**生き続ける場合**を分けて測った。

| モード | 書く内容 | その後 |
| --- | --- | --- |
| **truncate-json** | JSON応答の**前半だけ**（改行なし） | **`exit(2)`** |
| **no-newline** | **完全なJSON**だが**末尾改行なし** | **`exit(0)`** |
| **garbage** | **非JSON**（`NOT JSON AT ALL {{{`） | **`exit(2)`** |
| **keepalive-garbage** | 同じ非JSON | **30秒生き続ける** |
| **keepalive-truncate** | 同じ前半だけのJSON | **30秒生き続ける** |

`toolCallTimeoutMs` は **4000**。

## 2. 観測した結果

### 子が終了すると `MCP connection closed`

| モード | 結果 |
| --- | --- |
| truncate-json | **`Error: Error: MCP connection closed`** |
| no-newline | **`Error: Error: MCP connection closed`** |
| garbage | **`Error: Error: MCP connection closed`** |

- **3つとも同じ文言**である。**「切り詰め」も「改行なし」も「非JSON」も区別されない**。

### 子が生きていると `Request timed out`

| モード | 結果 |
| --- | --- |
| keepalive-garbage | **`Error: Request timed out`** |
| keepalive-truncate | **`Error: Request timed out`** |

- **壊れたバイトで呼び出しが即座に終わらない**。**タイムアウトまで待ってから失敗する**。**「解析して拒否した」という記録は無い**ので、**捨てられたとは推論**である（観測は**待ってからタイムアウトした**ことまで）。
- **子は生きているのでEOFが来ない**。それでも**応答は永遠に返らない**。

## 3. この記録が示すこと

- **`MCP connection closed` は「解析に失敗した」ではなく「パイプが閉じた」を報告している**。
- **同じ「壊れた応答」でも、子が終了するかで別のエラーになる**（`connection closed` と `Request timed out`）。
- **どちらの文言も本当の原因（応答が壊れている）を名指ししない**。**診断は「接続断」か「遅い」と読み違える**。
- 機密の観点では、**壊れた応答を返すサーバーは「落ちた」と「遅い」の両方に見える**。**どちらも「不正なデータが来た」とは報告されない**。

### 所要時間について（訂正）

初版は**18.3秒・38.0秒**と書いたが、これは**外側のシェルのストップウォッチ**で、**DSHの起動と自分の処理時間を含む**。**Tool呼び出しの所要時間ではない**。**38秒はkeepaliveの30秒を超えており、矛盾する**。**この2つの数値は削除した**（読み替えもしていない）。

**記録しているのは子側の時刻だけ**（`tools/call`が届いた時刻・書いた時刻）である。**クライアント側の「エラーまでの時間」は測っていない**。

## 4. この記録が示さないこと

- **タイムアウト後にバイトが解析されるか**。
- **整形は正しいが形が違う応答**（例: `result` の代わりに `data`）。
- **改行の後に切り詰めた場合**（本記録は改行の前に切っている）。
- **他のタイムアウト値**、**keepaliveの長さ**。
- 実MCPサーバー、サーバー側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `stdio-malformed-reply.json` | `overlay`、`modes`（5通り）、`runs`（各モードの子ログ・書いた内容・`toolResult`・`tools/call`到達有無・**終了したか**）、`childExitsThenObservation`、`childStaysAliveThenObservation`、`answers`、`implication`、`caveat` |

生の要求bodyはコミットしていない。**合成のみ**（`PARTIAL_CANARY_441`）。

再現する場合の要点:

```yaml
- insert:
    - id: mcp-client-partial
      name: "@deepseek-ai/dsh-mcp-client"
      config:
        transport: stdio
        serverName: partialmock
        command: python
        args: ["<script>", "<record-dir>", "truncate-json"]
        toolCallTimeoutMs: 4000
        failOnStartupError: false
```

- **「書いた後に終了する」と「書いた後に生き続ける」を必ず分けて測る**。**片方だけだと、原因を取り違える**（実際、最初は「壊れた応答＝接続断」と読みかけた）。
- **子側で「何を書いたか」と「その後どうしたか」を記録する**。
- **`toolCallTimeoutMs` を短くしすぎない**。生存側はタイムアウトまで待つので、その時間が必要になる。

### 手法上の注意（実際に踏んだ）

**モデル側Mockの「1回だけToolを呼ぶ」フラグが実行をまたいで残る**ため、2回目のモードで`tools/call`が発火せず空振りした。**モードごとにMockを再起動**して測り直している。

## 6. `H6-E2`への含意（判断はしない）

**壊れた応答は、子の生存状態によって「接続断」にも「タイムアウト」にも見える**。したがって**エラー文言から「相手が不正なデータを返した」を読み取れない**。外部Effectを持つToolでは、**失敗の理由を取り違えると再試行の判断も誤る**。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
