# MCPサーバーを5つ繋ぐとハンドシェイク20件・Tool 32個になり、増え方は線形のまま（2026-11-11、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-11（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [MCPの2サーバー](../DSH_MCP_TWO_SERVERS_2026-11-10/README.md)・[3サーバー](../DSH_MCP_THREE_SERVERS_2026-11-11/README.md)。本記録は**5サーバー**へ増やし、**比例が崩れないか**を確かめる。

## 1. 何をしたか

`id`と`serverName`を分けた**5エントリ**を挿した。Mockは**パスで5サーバーを分けて**、それぞれ別名のToolを返す。

## 2. 観測した結果

### 5サーバーがそれぞれ独立にハンドシェイクした

| サーバー | ハンドシェイク |
| --- | --- |
| **server1**〜**server5** | **各1回（4件ずつ）** |

- **MCP要求は合計20件**（4件×5）。**5つとも通り、取りこぼしは無い**。

### Toolは32個

| 追加されたTool | 出どころ |
| --- | --- |
| **`mcp__server1__tool_server1`** 〜 **`mcp__server5__tool_server5`** | 各サーバー |
| `list_mcp_resources`／`list_mcp_resource_templates`／`read_mcp_resource` | `mcp-resources` |

### サーバー数とTool数の関係（0・2・3・5の4点）

| サーバー数 | ハンドシェイク | Tool数 | 出典 |
| --- | --- | --- | --- |
| **0** | 0 | **27** | MCPサーバー未設定の既定面（24＋`mcp-resources` 系3） |
| 2 | 8 | 29 | [MCPの2サーバー](../DSH_MCP_TWO_SERVERS_2026-11-10/README.md) |
| 3 | 12 | 30 | [3サーバー](../DSH_MCP_THREE_SERVERS_2026-11-11/README.md) |
| **5** | **20** | **32** | **本記録** |

- **`Tool数 = 27 + サーバー数`** が4点すべてで成り立つ。
- **ハンドシェイク = 4 × サーバー数** も成り立つ。

## 3. この記録が示すこと

- **MCPの接続数は、この範囲では線形に効く**。**5サーバーでも比例は崩れていない**。
- **Tool数の増え方が予測できる**（`27 + N`）。**「いくつ繋いだか」からTool数を計算できる**。
- 機密の観点では、**サーバーを1つ増やすごとに、起動時の要求4件とTool定義1つが確実に増える**。**増分を数で見積もれる**。

## 4. この記録が示さないこと

- **上限**（本記録は5まで。**線形性は0・2・3・5の4点からの外挿**であり、**どこかで頭打ちになるかは未確認**）。
- **`serverName`を重複させた場合**、**同じサーバーを二重に挿した場合**。
- **一部だけ落ちた場合**（[MCPの障害分離](../DSH_MCP_FAILURE_ISOLATION_2026-11-10/README.md)は2サーバーで確認済み）。
- **多数接続時の起動時間・メモリ**。
- 実MCPサーバー、サーバー側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `mcp-five.json` | `overlay`（5エントリ）、`serverCount`・`mcpRequestCount`、`methodsPerServer`、`handshakesPerServer`、`requests`（20件のserver・path・method・bytes・SHA-256）、`modelRequests`（2件のbytes・SHA-256・Tool数・MCP Tool・`mcp-resources`系）、`toolCountAfter`、`mcpTools`、`resourceTools`、`toolCountFormula`、`trend`（0・2・3・5の4点）、`namespacePattern` |

生の要求bodyはコミットしていない。**合成のみ**（`server1`〜`server5`）。

再現する場合の要点:

```yaml
# 5エントリ。idとserverNameをそれぞれ分ける（4番目・5番目も同じ形）
- insert:
    - id: mcp-client-1
      name: "@deepseek-ai/dsh-mcp-client"
      config: { transport: streamable-http, serverName: server1, url: "http://127.0.0.1:8799/mcp/1" }
    # ... 5 まで
```

- **`transport: streamable-http`** と **`toolCallTimeoutMs`** を各エントリに書く（省略しても動くが、既定値に依存しない形にした）。
- 判定は**サーバーごとのmethod列**と**モデルのTool名**の両方で行う。

## 6. `H6-E2`への含意（判断はしない）

MCPの接続数は**サーバー数に対して線形**で、**Tool数は `27 + N` で予測できる**（この範囲では）。機密ProfileでMCPを許すなら、**許可するサーバー数を上限として決めれば、起動時の要求数とTool数を事前に見積もれる**。ただし**上限そのもの（頭打ち）は未確認**であり、**「決めた数だけ必ず繋がる」と「それ以上は繋がらない」は別に確かめる**必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
