# 落ちたMCPサーバーは同じ実行の中で復帰し、次の要求からToolが使える（2026-11-10、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-10（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [落ちていたサーバーの復帰](../DSH_MCP_LATE_START_2026-11-10/README.md)（プロセスをまたぐ形）。本記録は、そこで**NOT RUN**とした「**1つのプロセス内での再接続**」を埋める。

## 1. 何をしたか

1. **MCPサーバーを止めたまま**実行を開始した（ポート`8797`は閉）。
2. モデルの**1手目で`pwsh`を呼ばせ**、その中で**MCPサーバーを起動**して12秒待たせた。
3. **同じプロセスの中で**次の要求を出し、**Tool一覧にMCP Toolが現れたか**を見た。

## 2. 観測した結果

| モデル要求 | Tool数 | MCP Tool |
| --- | --- | --- |
| **001（起動直後）** | **27** | **なし** |
| 002（内部） | 0 | — |
| **003（サーバー起動後）** | **28** | **`mcp__latemock__header_echo`** |

- **同じ実行の中で、27→28に増えた**。**プロセスを再起動していない**。
- MCPサーバーが受けたのは**4件＝ハンドシェイク1回分**（`server/discover`・`initialize`・`notifications/initialized`・`tools/list`）。
- **再試行の設定は既定**（同梱実装の`RECONNECT_DEFAULTS`：`initialDelayMs 500`・`maxDelayMs 30000`・`maxAttempts 10`、遅延は倍々）。

## 3. この記録が示すこと

- **`reconnect`は実際に効く**。**起動時に繋がらなくても、後から立ち上がったサーバーを同じ実行の中で拾う**。
- **Toolは再接続後にモデルの一覧へ載る**。**「起動時に無かったToolは最後まで無い」わけではない**。
- 機密の観点では、**実行中に送出先が増え得る**。**開始時の構成を最後まで前提にできない**。

## 4. この記録が示さないこと

- **実行中にサーバーが落ちた場合**（本記録は「落ちている→起動する」の向きだけ）。**逆向きは未確認**。
- **`maxAttempts`を超えて諦めた後**の挙動、**警告の出方**。
- **`reconnect`を明示的に切った場合**（`enabled: false`）。
- **再接続時に`headers`が再送されるか**（[MCPの`headers`](../DSH_MCP_HEADERS_2026-11-10/README.md)は初回接続のみ確認済み）。
- **`failOnStartupError: true`との組み合わせ**。
- 実MCPサーバーへの接続、サーバー側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `midrun-reconnect.json` | `overlay`、`reconnectDefaultsFromSource`（`initialDelayMs`・`maxDelayMs`・`maxAttempts`・挙動）、`firstModelRequest`／`lastModelRequest`（Tool数・MCP Tool）、`toolAppearedMidRun`、`mcpRequests`（4件のmethod・bytes）、`mcpRequestCount`・`handshakesObserved`、`modelRequests`（3件のbytes・SHA-256・Tool数・MCP Tool） |

生の要求bodyはコミットしていない。**合成のみ**（`latemock`）。

再現する場合の要点:

```sh
# 1) MCPサーバーを止めたまま起動し、1手目で pwsh から起動させる
#    （pwsh でプロセスを起こすため permission-mode は danger-full-access）
dsh --profile headless --patch <url to a closed port> \
    --permission-mode danger-full-access --approval-policy never \
    --json "Start the MCP server and check the tools."
```

- **モデルの1手目に起動コマンドを出させる**必要がある。手で起動すると「プロセスをまたぐ」形になり、この記録の条件にならない。
- 判定は**要求ごとのTool数**と**MCPサーバーが受けたハンドシェイク回数**の両方で行う。

## 6. `H6-E2`への含意（判断はしない）

MCPの接続状態は**実行中にも変わる**。**開始時に無かった送出先が、途中から加わり得る**。機密ProfileでMCPを統制するなら、**起動時の点検だけでは足りない**。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
