# MCPは設定形を合わせれば繋がり、Toolが増え、ハンドシェイクが外へ出る（2026-11-10、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-10（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルMock MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [MCP経路](../DSH_MCP_PATH_2026-11-09/README.md)。本記録は、そこで「**推測した設定では届かなかった**」と留保した点を、**プラグインの実装から設定形を特定して解消**する。

## 1. 訂正（先に書く）

前記録は「`mcp-client`を`servers.synthetic { transport: http, url }`で挿すと`--dump-config`は受理するが、MCP Toolも要求も現れない」と書いた。**その設定形はschemaではなかった**ため、**何も接続していなかった**。

同梱物から実装を読むと、`dsh-mcp-client`の`Config`は**1インスタンス1サーバー**で、`transport`は次の2択である。

```js
z.union([
  z.object({ transport: z.const("stdio"), serverName, command, args, env, cwd, ... }),
  z.object({ transport: z.const("streamable-http"), serverName, url, headers, ... }),
])
```

- **`transport: "http"`は無い**（`streamable-http`）。
- **`servers`という入れ子も無い**（複数サーバーは**エントリを複数挿す**）。
- `serverName`と`url`が必須。

## 2. 何をしたか

1. 同梱の`app.asar`から`dsh/node_modules/@deepseek-ai/dsh-mcp-client/lib/index.js`を取り出し、`Config`を読んだ。
2. その形どおりに`--patch`で挿した（`transport: streamable-http`、`serverName: synthetic`、`url: http://127.0.0.1:8799/mcp`）。
3. ローカルの合成MCPサーバー（JSON-RPCの`initialize`・`tools/list`・`resources/list`に応答）へ向けて1往復させ、**届いた要求**と**モデルのTool一覧**を数えた。

## 3. 観測した結果

### ハンドシェイクが走った

| # | method | 要求bytes |
| --- | --- | --- |
| 1 | **`server/discover`** | 280 |
| 2 | **`initialize`**（`protocolVersion: 2025-11-25`） | 163 |
| 3 | **`notifications/initialized`** | 54 |
| 4 | **`tools/list`** | 46 |

- **4件のJSON-RPCが届いた**。`server/discover`→`initialize`→`notifications/initialized`→`tools/list`の順で、**クライアント名は`dsh-mcp-client`**。

### モデルのToolが24から28へ増えた

| 追加されたTool | 出どころ |
| --- | --- |
| **`mcp__synthetic__synthetic_echo`** | MCPサーバーのTool（**`mcp__<serverName>__<toolName>`**で名前空間化） |
| `list_mcp_resources` | `mcp-resources` |
| `list_mcp_resource_templates` | `mcp-resources` |
| `read_mcp_resource` | `mcp-resources` |

- 既存24個に**4個が加わって28個**。
- モデル要求のbytesも**99,797**（既定時の90,627より**約9,170増**）。Tool定義の分だけ増えている。

## 4. この記録が示すこと

- **MCP経路は設定次第で開く**。既定の面に出ないのは**サーバーが未設定だから**で、経路自体が無いわけではない（前記録の留保が確かめられた）。
- 設定形は**`transport: streamable-http`／`serverName`／`url`**で、**1インスタンス1サーバー**。
- 開くと**起動時にハンドシェイク（4件）が外へ出る**。`initialize`には**クライアント名と版**が載る。
- **モデルのTool一覧が増える**ため、**MCPのTool定義もモデル入力として外へ出る**（要求bytesが約9,170増）。

## 5. この記録が示さないこと

- **Toolを実際に呼んだ場合**の送信内容（本記録は`tools/list`までで、`tools/call`は実行していない）。
- **stdio transport**、認証ヘッダ、再接続（`reconnect`）。
- **複数サーバー**を同時に挿した場合の相互作用。
- `mcp-resources`が実際に何を読みに行くか。
- 実MCPサーバーへの接続、サーバー側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 6. 生データ

| ファイル | 内容 |
| --- | --- |
| `mcp-configured.json` | `configShapeFromSource`（`union`の2形・必須/任意・`oneServerPerInstance`）、`overlay`、`mcpRequests`（4件のmethod・bytes・SHA-256）、`methodsSeen`、`modelRequests`（2件のbytes・Tool名一覧・MCP Tool・Session ID・水位）、`toolCountBefore`/`toolCountAfter`、`mcpToolNames`、`toolNamespacePattern`、`correction` |

生の要求bodyはコミットしていない。**合成のみ**（サーバー名`synthetic`・Tool名`synthetic_echo`・canary `MCP_CANARY_993`）。

再現する場合の要点:

```yaml
# 1インスタンス1サーバー。transport は stdio か streamable-http のみ。
- insert:
    - id: mcp-client
      name: "@deepseek-ai/dsh-mcp-client"
      config:
        transport: streamable-http
        serverName: synthetic
        url: "http://127.0.0.1:8799/mcp"
        toolCallTimeoutMs: 5000
        failOnStartupError: false
```

```sh
dsh --profile headless --patch <overlay> --permission-mode read-only \
    --approval-policy never --json "Say ok."
```

- 設定形は**同梱の`dsh-mcp-client/lib/index.js`の`Config`**が正本。`--dump-config-schema`では`unknownConfig`（投影されたschemaが無い）なので、**実装を読む必要がある**。
- **`--dump-config`が受理しても接続するとは限らない**。判定は**MCP要求が届いたか**と**モデルのTool一覧が増えたか**の両方で行う。
- `app.asar`からファイルを取り出すには、asarのヘッダ（8+headerSizeのオフセット）を読む小さなスクリプトで足りる。

## 7. `H6-E2`への含意（判断はしない）

MCPは**設定すれば開く経路**であり、開くと**起動時にハンドシェイクが外へ出て、サーバーのTool定義がモデル入力に載る**。機密ProfileでMCPを扱うなら、**「既定で開いていない」を安全の根拠にせず**、**どのサーバーを許可するか**と**Tool定義・引数が外部へ出ることをどう扱うか**を分けて決める必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
