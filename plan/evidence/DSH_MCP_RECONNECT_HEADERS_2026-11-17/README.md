# HTTP 500は`reconnect`の再試行を起こさず、`headers`は失敗した要求にも載る（2026-11-17、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-17（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.3`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [MCPの`headers`](../DSH_MCP_HEADERS_2026-11-10/README.md)（初回接続での送信）。本記録は、そこで未確定とした「**再接続時の`headers`再送**」に取り組んだ。

## 1. 何をしたか

`headers`に**canary（`RECONNECT_HEADER_CANARY_771`）**を入れ、**ハンドシェイクの途中**または**`tools/call`の最中**に**HTTP 500を返す**ようにした。`reconnect`は`maxAttempts: 8`にして**再試行の余地を十分**にした。

## 2. 観測した結果

### 結論を先に: 再送は観測できなかった（再試行が起きないため）

| 実行 | 500の位置 | **再試行** | 結果 |
| --- | --- | --- | --- |
| **A** | ハンドシェイク途中 | **なし** | クライアントが停止。Toolは現れない |
| **B** | `tools/call` の最中 | **なし** | 呼び出しが失敗し、**以降の要求は無い** |
| C（対照） | 500なし | — | 正常に完了 |

- **`maxAttempts: 8` の設定で、HTTP 500では再試行しなかった**。**A・Bとも、500の後に要求が続かない**。**他の`maxAttempts`値は試していない**ので、**「どの値でも再試行しない」とは言えない**。

### `headers`は失敗した要求にも載っている

canaryは**クライアントが送った全要求**に載っていた（**500を受け取った要求を含む**）。

| 要求 | method | 応答 | `Authorization`にcanary |
| --- | --- | --- | --- |
| 1 | `server/discover` | 200 | **あり** |
| 2 | `initialize` | 200 | **あり** |
| 3 | `notifications/initialized` | 200 | **あり** |
| 4 | `tools/list` | 200 | **あり** |
| 5 | `tools/call` | **500** | **あり** |

### HTTP 500 の失敗形

```
Error: Error POSTing to endpoint: {"error":"synthetic outage"}
```

- **これも別の形**である。`fetch failed`（相手が居ない）・`Request timed out`（無応答）・`MCP connection closed`（子が落ちた）とは違う。

## 3. この記録が示すこと

- **`reconnect` は HTTP 500 を再試行しない**。**ハンドシェイク中でも呼び出し中でも同じ**。**`maxAttempts` を増やしても効かない**。
- **`headers` は「接続時に一度」ではなく「毎要求」送られる**（**失敗した要求にも載る**）。
- したがって**「再接続時の再送」は、再接続が起きない以上、この条件では観測できない**。**問いの立て方そのものが条件に依存していた**。

## 4. この記録が示さないこと（重要）

- **トランスポート層の障害（接続拒否・切断）で `reconnect` が働く場合の `headers` 再送**。**本記録はHTTP 500という「応答は返るが失敗」の条件だけ**を測った。**本記録は接続断を試していない**ため、**接続断で`reconnect`が働くかは未確認**である。**「500には効かない」だけが言える**。
- **500が続いた場合に別の経路で復帰するか**（次の起動で繋がるか）。
- **`5xx` と `4xx` の違い**。
- **複数サーバー時の挙動**。
- 実MCPサーバー、サーバー側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `mcp-reconnect-headers.json` | `overlay`、`headerCanary`、`runs`（A: ハンドシェイク中の500／B: `tools/call`中の500／C: 対照）、`finalRunRequests`（5件のmethod・応答・bytes・SHA-256・ヘッダ数・canaryの有無）、`canaryOnEveryRequest`、`canaryOnFailingRequest`、`requestsAfterLastFailure`、`answers`、`failureShape`（4経路の対比）、`caveat` |

生の要求bodyはコミットしていない。**合成のみ**（`reconnectmock`・`rh_echo`・`RECONNECT_HEADER_CANARY_771`）。

再現する場合の要点:

```yaml
- insert:
    - id: mcp-client-reconnect-headers
      name: "@deepseek-ai/dsh-mcp-client"
      config:
        transport: streamable-http
        serverName: reconnectmock
        url: "http://127.0.0.1:8795/mcp/rh"
        headers:
          Authorization: "Bearer RECONNECT_HEADER_CANARY_771"
        reconnect:
          maxAttempts: 8
```

- **サーバー側で要求ごとにヘッダを記録する**。**どの要求に載ったか**が判定の単位になる。
- **500の位置を変えて2通り測る**（ハンドシェイク中・呼び出し中）。**片方だけだと「再試行しない」を見落とす**。
- **対照（500なし）を必ず取る**。これが無いと**「そもそも繋がらない設定だった」**を排除できない。

## 6. `H6-E2`への含意（判断はしない）

**この設定・この条件では、`reconnect` は HTTP 500 を再試行しなかった**。したがって**「`reconnect` を設定しておけば、サーバーが不調でも回復する」とは言えない**（**ただし接続断での挙動は未確認**なので、**効かないのは500に対して**である）。機密ProfileでMCPを統制するなら、**再試行が効く障害と効かない障害を分けて**扱う必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
