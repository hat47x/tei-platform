# MCPの複数サーバーはエントリを分ければ同時に繋がり、Toolは名前空間で分かれる（2026-11-10、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-10（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [MCPの設定形](../DSH_MCP_CONFIGURED_2026-11-10/README.md)・[MCPの`headers`](../DSH_MCP_HEADERS_2026-11-10/README.md)。本記録は、そこで未確定とした「**複数サーバーの相互作用**」を埋める。

## 1. 何をしたか

`Config`が**1インスタンス1サーバー**なので、**`id`を分けたエントリを2つ**挿した。

```yaml
- id: mcp-client-a
  name: "@deepseek-ai/dsh-mcp-client"
  config: { transport: streamable-http, serverName: servera, url: "http://127.0.0.1:8799/mcp/a" }
- id: mcp-client-b
  name: "@deepseek-ai/dsh-mcp-client"
  config: { transport: streamable-http, serverName: serverb, url: "http://127.0.0.1:8799/mcp/b" }
```

Mockはパスで2つのサーバーを分け、**それぞれ別名のTool**を返す。

## 2. 観測した結果

### 2つのサーバーがそれぞれ独立にハンドシェイクした

| サーバー | 受け取った method（順） |
| --- | --- |
| **a** | `server/discover` → `initialize` → `notifications/initialized` → `tools/list` |
| **b** | `server/discover` → `initialize` → `notifications/initialized` → `tools/list` |

- **MCP要求は合計8件**（4件×2サーバー）。**両方とも最後まで通った**。
- 到着順は**交互**（a,b,a,b,…）で、**どちらかが失敗して他方だけ生き残る形ではない**。

### モデルのToolは24→29

| 追加されたTool | 出どころ |
| --- | --- |
| **`mcp__servera__tool_a`** | サーバーa |
| **`mcp__serverb__tool_b`** | サーバーb |
| `list_mcp_resources`／`list_mcp_resource_templates`／`read_mcp_resource` | `mcp-resources` |

- **名前空間が`mcp__<serverName>__<toolName>`なので、2つのサーバーのToolが衝突しない**。
- **両方のToolが同時にモデルへ渡る**。

## 3. この記録が示すこと

- **MCPは複数サーバーを同時に繋げられる**。`Config`が1サーバーでも、**エントリを分ければ並ぶ**。
- **サーバーごとに独立したハンドシェイクが走る**。**N個繋げば N×4 の要求が起動時に出る**。
- **`serverName`が名前空間になる**ので、**サーバー名が衝突すると Tool 名も衝突し得る**（本記録は別名で回避）。
- **モデルは全サーバーのToolを同時に見る**。あるサーバーと話している最中でも、**別サーバーのTool定義は文脈に載っている**。

## 4. この記録が示さないこと

- **`serverName`を同じにした場合**（衝突時の挙動）。
- **一方のサーバーが落ちた場合**（`failOnStartupError`・`reconnect`との関係）。
- **片方だけに`headers`を付けた場合**の切り分け（[MCPの`headers`](../DSH_MCP_HEADERS_2026-11-10/README.md)は1サーバーで確認済み）。
- **stdioとstreamable-httpの混在**。
- 実MCPサーバーへの接続、サーバー側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `mcp-two.json` | `overlay`（2エントリ）、`mcpRequestCount`・`serverCount`、`methodsPerServer`（a/bそれぞれ4件）、`requests`（8件のserver・path・method・bytes・SHA-256）、`modelRequests`（2件のbytes・SHA-256・Tool数・MCP Tool・`mcp-resources`系）、`toolCountBefore`／`toolCountAfter`、`mcpTools`、`namespacePattern` |

生の要求bodyはコミットしていない。**合成のみ**（`servera`／`serverb`・`tool_a`／`tool_b`）。

再現する場合の要点:

```sh
# パスで2サーバーを分け、それぞれ別名のToolを返すMock
python mock_provider_mcp_two.py 8799 <record-dir>
dsh --profile headless --patch <2エントリのoverlay> --permission-mode read-only \
    --approval-policy never --json "Say ok."
```

- **`id`を分ける**（同じ`id`では2つ目が置き換わる）。**`serverName`も分ける**（名前空間になる）。
- 判定は**サーバーごとのmethod列**と**モデルのTool名**の両方で行う。

## 6. `H6-E2`への含意（判断はしない）

MCPを1つ許す判断は、**エントリを増やせばいくつでも増やせる**ことを意味する。**サーバーごとに独立してハンドシェイクし、`headers`を送り、Toolを文脈に載せる**ので、**「1つだけ許可した」は構成の制約にならない**。機密ProfileでMCPを扱うなら、**許可するサーバーの集合**と、**その集合が増えないことをどう確かめるか**を分けて決める必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
