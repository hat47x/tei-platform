# `reconnect: { enabled: false }` では、後から起動したMCPサーバーを拾わない（2026-11-16、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-16（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [実行中の再接続](../DSH_MCP_MIDRUN_RECONNECT_2026-11-10/README.md)（既定の`reconnect`で27→28）。本記録は、そこで未確定とした「**`reconnect: { enabled: false }`**」を埋める。

## 1. 何をしたか

**同じ実験を2回**行った。MCPサーバーは**起動時は落ちており**、実行の**1手目の`pwsh`がサーバーを起動**する。違いは**`reconnect`の設定だけ**。

| 実行 | `reconnect` |
| --- | --- |
| **無効** | `enabled: false` |
| **既定** | 未指定（`enabled`・`initialDelayMs 500`・`maxDelayMs 30000`・`maxAttempts 10`） |

## 2. 観測した結果

### 無効にすると、サーバーが動いても1件も繋がない

| 項目 | 無効 | 既定 |
| --- | --- | --- |
| **MCP要求** | **0件** | **4件**（ハンドシェイク1回） |
| **モデルが見たTool数** | **27・0・27** | **27・0・28** |
| **MCP Toolが現れたか** | **最後まで現れない** | **現れた** |
| 現れた要求 | — | **6番目** |

- **サーバーは確かに起動している**。サーバー自身が`server-started.json`（`port: 8797`・`pid`）を書き、プロセスも生存していた。
- それでも**無効側はMCP要求が0件**で、**`mcp__latemock__header_echo`は一度も現れない**。
- **既定側は同じ条件で4件の要求が届き、6番目の要求でToolが現れた**。

### 差は設定だけ

`transport`・`url`・`serverName`・`failOnStartupError`・起動コマンドは**両方で同一**。**`reconnect.enabled`だけが違う**。

## 3. この記録が示すこと

- **`reconnect: { enabled: false }` は「再接続しない」として実際に効く**。**後から立ち上がったサーバーは拾わない**。
- **既定との差は「試行回数」ではなく「試行の有無」**。無効側は**1回も試さない**（0件）。
- 機密Profileの観点では、**`enabled: false` は「その実行では繋がらない」を意味するが、「サーバーが存在しない」ではない**。**次の起動では既定に戻せば繋がる**（[落ちていたサーバーの復帰](../DSH_MCP_LATE_START_2026-11-10/README.md)）。

## 4. この記録が示さないこと

- **`maxAttempts`を超えて諦めた後**の挙動（本記録は既定の1回の成功のみ）。
- **`enabled: true` で `maxAttempts` を小さくした場合**。
- **再接続時に`headers`が再送されるか**（[MCPの`headers`](../DSH_MCP_HEADERS_2026-11-10/README.md)は初回接続のみ確認済み）。
- **起動後にサーバーが落ちた場合**に`enabled: false`がどう振る舞うか。
- **`stdio`の子Process**の場合。
- 実MCPサーバー、サーバー側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `reconnect-disabled.json` | `experiment`（URL・`serverName`・起動方法・サーバーの起動マーカー）、`disabledRun`／`defaultRun`（要求番号・Tool数・MCP Toolの有無・受けたMCP要求数）、`contrast`、`mcpMethodsSeen`、`modelRequests`（6件のbytes・SHA-256・Tool数・MCP Tool）、`caveat` |

生の要求bodyはコミットしていない。**合成のみ**（`latemock`）。

再現する場合の要点:

```yaml
# 無効側。既定側はこの reconnect ブロックを外すだけ。
- insert:
    - id: mcp-client-late
      name: "@deepseek-ai/dsh-mcp-client"
      config:
        transport: streamable-http
        serverName: latemock
        url: "http://127.0.0.1:8797/mcp/late"
        failOnStartupError: false
        reconnect:
          enabled: false
```

- **サーバーは1手目の`pwsh`で起動させる**（`--permission-mode danger-full-access`が必要）。
- **サーバー側に起動マーカーを書かせる**。これが無いと「起動しなかった」と「繋がなかった」を区別できない。実際、初回はサーバースクリプトが無いまま実行してしまい、**結果を無効と誤読しかけた**（作り直して再実行した）。
- 判定は**MCPサーバーが受けた要求数**と**モデルのTool一覧**の両方で行う。

## 6. `H6-E2`への含意（判断はしない）

`reconnect.enabled` は**実際に効く統制**である。無効にすれば**その実行では繋がない**が、**サーバーが無いことにはならない**。機密ProfileでMCPを扱うなら、**「繋がない」を再試行の設定で作るのではなく**、**宛先そのものを制御する**必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
