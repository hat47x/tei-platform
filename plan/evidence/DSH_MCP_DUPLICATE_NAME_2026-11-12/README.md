# 同じ`serverName`で2つ挿すと2つ目は有効化されず、名前の衝突は起きない（2026-11-12、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-12（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [MCPの複数サーバー](../DSH_MCP_TWO_SERVERS_2026-11-10/README.md)・[8サーバー](../DSH_MCP_EIGHT_SERVERS_2026-11-11/README.md)。本記録は、そこで未確定とした「**`serverName`を重複させた場合**」を埋める。

## 1. 何をしたか

**`serverName`を同じ**（`dupserver`）にした2エントリを挿し、**URLは別**（`/mcp/1`と`/mcp/2`）にした。両方のサーバーは**同じ名前のTool**（`echo`）を返す。**モデル向けの名前が衝突する条件**を作った。

## 2. 観測した結果

| 項目 | 値 |
| --- | --- |
| 有効化されたエントリ | **1**（`mcp-client-dup-a`） |
| 拒否されたエントリ | **1**（`mcp-client-dup-b`） |
| **ハンドシェイクしたサーバー** | **1**（`/mcp/1`のみ） |
| モデルが見たTool数 | **28** |
| MCP Tool | **`mcp__dupserver__echo`（1個）** |

拒否の文言:

```
serverName "dupserver" is already in use by another mcp-client instance
```

- **`/mcp/2` には1件も要求が行っていない**。
- **Tool名の衝突はモデルに届く前に止まった**（`mcp__dupserver__echo` は1個だけ）。
- 実行は**exit 0**で、`dsh: warning: 1 entry did not activate` が stderr に出た。

## 3. この記録が示すこと

- **`serverName`は一意でなければならない**。**重複は「2つ目が黙って動く」ではなく、明示的に拒否される**。
- **拒否は起動時**（ハンドシェイクの前）に起きる。**`/mcp/2` には一度も接続しない**。
- **Tool名の衝突は起きない**。**同じ名前で別サーバーを指す状態は作れない**。
- 機密の観点では、**「名前を揃えて2つ挿せば二重に繋がる」は成り立たない**。**設定の誤りは片方の欠落として現れる**（[MCPの障害分離](../DSH_MCP_FAILURE_ISOLATION_2026-11-10/README.md)と同じく、**exit 0 のまま**）。

## 4. この記録が示さないこと

- **同じURLを2回挿した場合**（`serverName`は別）。本記録は**同じ名前・別URL**のみ。
- **`serverName`の書式制約**（長さ・使える文字）。
- **拒否された後、そのエントリが再試行されるか**（`reconnect`との関係）。
- **`failOnStartupError: true` を付けた場合**の挙動。
- 実MCPサーバー、サーバー側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `mcp-duplicate-name.json` | `overlay`（2エントリ・同一`serverName`・別URL）、`sameServerName`・`distinctUrls`、`refusalText`、`serversThatHandshook`・`pathsTouched`、`requests`（4件のpath・method・bytes・SHA-256）、`modelRequests`（2件のbytes・SHA-256・Tool数・MCP Tool）、`toolCountAfter`、`mcpToolsInModel`、`collisionRejected`、`contrast` |

生の要求bodyはコミットしていない。**合成のみ**（`dupserver`・`echo`）。

再現する場合の要点:

```yaml
# 同じ serverName・別 URL。両方のサーバーは同名の tool を返すようにする
- insert:
    - id: mcp-client-dup-a
      name: "@deepseek-ai/dsh-mcp-client"
      config: { transport: streamable-http, serverName: dupserver, url: "http://127.0.0.1:8799/mcp/1" }
    - id: mcp-client-dup-b
      name: "@deepseek-ai/dsh-mcp-client"
      config: { transport: streamable-http, serverName: dupserver, url: "http://127.0.0.1:8799/mcp/2" }
```

- 判定は**どちらのパスに要求が行ったか**（Mockの`path`記録）と、**モデルのTool名**の両方で行う。
- **拒否はstderrの`did not activate`**に現れる。**終了コードでは分からない**。

## 6. `H6-E2`への含意（判断はしない）

`serverName`は**一意性の制約**であり、**重複は起動時に拒否される**。したがって**「名前を揃えて二重に繋ぐ」経路は無い**が、**設定の誤りは片方の欠落として静かに現れる**（exit 0 のまま）。機密ProfileでMCPを統制するなら、**設定の重複を作らないこと**と、**有効化されたエントリを数えること**を分けて扱う必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
