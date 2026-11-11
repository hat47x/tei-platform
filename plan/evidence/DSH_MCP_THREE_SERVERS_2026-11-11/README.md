# MCPサーバーを3つ繋ぐとハンドシェイクは12件になり、Toolは30個になる（2026-11-11、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-11（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [MCPの複数サーバー](../DSH_MCP_TWO_SERVERS_2026-11-10/README.md)（2サーバー・Tool 29）。本記録は**3サーバー**へ増やして、**増え方が線形か**を確かめる。

## 1. 何をしたか

`id`と`serverName`を分けた**3エントリ**を挿し、Mockは**パスで3サーバーを分けて**それぞれ別名のToolを返すようにした。

## 2. 観測した結果

### 各サーバーが独立にハンドシェイクした

| サーバー | 受け取った method（順） | 件数 |
| --- | --- | --- |
| **server1** | `server/discover` → `initialize` → `notifications/initialized` → `tools/list` | **4** |
| **server2** | 同じ | **4** |
| **server3** | 同じ | **4** |

- **MCP要求は合計12件**。**3つとも最後まで通った**。

### Toolは24→30

| 追加されたTool | 出どころ |
| --- | --- |
| **`mcp__server1__tool_server1`** | サーバー1 |
| **`mcp__server2__tool_server2`** | サーバー2 |
| **`mcp__server3__tool_server3`** | サーバー3 |
| `list_mcp_resources`／`list_mcp_resource_templates`／`read_mcp_resource` | `mcp-resources` |

### 2サーバーとの比較

| サーバー数 | ハンドシェイク | Tool数 | MCP Tool数 |
| --- | --- | --- | --- |
| **2** | 8 | **29** | 2 |
| **3** | **12** | **30** | **3** |

## 3. この記録が示すこと

- **ハンドシェイクはサーバー数に比例して増える**（2→8件、3→12件、いずれも**4件×N**）。
- **MCPのToolもサーバー数に比例して増える**（2→2個、3→3個）。**ベースの24に、サーバー数だけ単純に足される**（`mcp-resources` 系3つは1回だけ）。
- **`serverName`を分ければ衝突しない**。**3つ同時でも順序や取りこぼしの問題は起きていない**。

## 4. この記録が示さないこと

- **4つ以上**（本記録は3まで）。**線形性は3点（0・2・3）からの外挿**であり、上限は未確認。
- **`serverName`を重複させた場合**の挙動。
- **片方だけ落ちた場合**の切り分け（[MCPの障害分離](../DSH_MCP_FAILURE_ISOLATION_2026-11-10/README.md)は2サーバーで確認済み）。
- **同じサーバーを2回挿した場合**。
- 実MCPサーバー、サーバー側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `mcp-three.json` | `overlay`（3エントリ）、`serverCount`・`mcpRequestCount`、`methodsPerServer`、`handshakesPerServer`、`requests`（12件のserver・path・method・bytes・SHA-256）、`modelRequests`（2件のbytes・SHA-256・Tool数・MCP Tool）、`toolCountBefore`／`toolCountAfter`、`mcpTools`、`namespacePattern`、`comparison`（2サーバーの記録との対比） |

生の要求bodyはコミットしていない。**合成のみ**（`server1`〜`server3`）。

再現する場合の要点:

```yaml
# 3エントリ。idとserverNameをそれぞれ分ける
- insert:
    - id: mcp-client-1
      name: "@deepseek-ai/dsh-mcp-client"
      config: { transport: streamable-http, serverName: server1, url: "http://127.0.0.1:8799/mcp/1" }
    - id: mcp-client-2
      config: { transport: streamable-http, serverName: server2, url: "http://127.0.0.1:8799/mcp/2" }
    - id: mcp-client-3
      config: { transport: streamable-http, serverName: server3, url: "http://127.0.0.1:8799/mcp/3" }
```

- Mockは**パスでサーバーを判別**し、**別名のTool**を返すようにする。
- 判定は**サーバーごとのmethod列**と**モデルのTool名**の両方で行う。

## 6. `H6-E2`への含意（判断はしない）

MCPは**エントリを増やせば増える**（[MCPの複数サーバー](../DSH_MCP_TWO_SERVERS_2026-11-10/README.md)）。**3つでは、ハンドシェイクもToolも線形に増えた**。機密ProfileでMCPを許すなら、**「いくつまで許すか」を数で決め、実際の数（ハンドシェイク件数・Tool数）で確かめる**必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
