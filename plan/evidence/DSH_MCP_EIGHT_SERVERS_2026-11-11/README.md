# MCPサーバーを8つ繋いでも線形は崩れない（Toolは35個、ハンドシェイク32件）（2026-11-11、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-11（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [5サーバー](../DSH_MCP_FIVE_SERVERS_2026-11-11/README.md)。本記録は**8サーバー**へ増やし、**線形がどこまで続くか**を確かめる。

## 1. 何をしたか

`id`と`serverName`を分けた**8エントリ**を挿した。Mockは**パスで8サーバーを分けて**、それぞれ別名のToolを返す。

## 2. 観測した結果

| 項目 | 値 |
| --- | --- |
| 繋がったサーバー | **8** |
| **MCP要求（ハンドシェイク）** | **32件（4件×8）** |
| **モデルが見たTool数** | **35** |
| **MCP Tool数** | **8** |

- **8サーバーすべてが4件ずつ受け取り、取りこぼしは無い**。
- **`Tool数 = 27 + 8 = 35`** で、**式どおり**。

### 0・2・3・5・8 の5点

| サーバー数 | ハンドシェイク | Tool数 |
| --- | --- | --- |
| **0** | 0 | **27** |
| 2 | 8 | 29 |
| 3 | 12 | 30 |
| 5 | 20 | 32 |
| **8** | **32** | **35** |

- **`Tool数 = 27 + N`** と **`ハンドシェイク = 4×N`** が**5点すべてで成り立つ**。

## 3. この記録が示すこと

- **MCPの接続数は、少なくとも8まで線形に効く**。**比例は崩れていない**。
- **Tool数は `27 + N` で予測できる**。**8サーバーでも式が合う**。
- **この版に「サーバー数の上限」は見えていない**（8までは通る）。ただし**上限が無いとは言えない**（下記）。

## 4. この記録が示さないこと

- **上限（頭打ち）があるか**。本記録は8までで、**「8が上限」でも「上限が無い」でもない**。**さらに増やした場合**は未確認。
- **多数接続時の起動時間・メモリ・同時接続の限界**。
- **`serverName`を重複させた場合**、**同じサーバーを二重に挿した場合**。
- **一部だけ落ちた場合**（[MCPの障害分離](../DSH_MCP_FAILURE_ISOLATION_2026-11-10/README.md)は2サーバーで確認済み）。
- 実MCPサーバー、サーバー側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `mcp-eight.json` | `overlay`（8エントリ・サーバー数）、`serverCount`・`mcpRequestCount`、`methodsPerServer`、`handshakesPerServer`、`requests`（32件のserver・path・method・bytes・SHA-256）、`modelRequests`（2件のbytes・SHA-256・Tool数・MCP Tool・`mcp-resources`系）、`toolCountAfter`、`mcpTools`、`toolCountFormula`、`trend`（0・2・3・5・8の5点）、`namespacePattern` |

生の要求bodyはコミットしていない。**合成のみ**（`server1`〜`server8`）。

再現する場合の要点:

```yaml
# 8エントリ。idとserverNameをそれぞれ分ける（同じ形を8回）
- insert:
    - id: mcp-client-1
      name: "@deepseek-ai/dsh-mcp-client"
      config: { transport: streamable-http, serverName: server1, url: "http://127.0.0.1:8799/mcp/1" }
    # ... 8 まで
```

- Mockは**パスでサーバーを判別**し、**別名のTool**を返す。
- **エントリを増やすときは `id` と `serverName` の両方**を分ける（`serverName` は名前空間になる）。

## 6. `H6-E2`への含意（判断はしない）

MCPの接続数は**8まで線形**で、**Tool数は `27 + N` で予測できる**。機密ProfileでMCPを許すなら、**許可するサーバー数を上限として決めれば、起動時の要求数とTool数を事前に見積もれる**。ただし**上限そのもの（頭打ち）は未確認**であり、**「決めた数だけ必ず繋がる」と「それ以上は繋がらない」は別に確かめる**必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
