# 落ちていたMCPサーバーは次回起動で繋がり、失敗は覚えられていない（2026-11-10、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-10（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [MCPの障害分離](../DSH_MCP_FAILURE_ISOLATION_2026-11-10/README.md)。本記録は、そこで未確定とした「**再接続**」に、**プロセスをまたぐ形で**着手した結果である。

## 1. 何をしたか

1. **MCPサーバーを止めたまま**（ポート`8797`は閉）Turn 1を実行した。
2. **MCPサーバーを起動**した。
3. **同じSessionを再開**してTurn 2を実行した（同じ`--patch`、`reconnect`は既定）。

## 2. 観測した結果

| Turn | MCPサーバー | モデルが見たTool数 | MCP Tool |
| --- | --- | --- | --- |
| **1** | **落ちている** | **27** | **なし** |
| **2** | **起動済み** | **28** | **`mcp__latemock__header_echo`** |

- **Turn 1ではMCP Toolが1つも無く、Turn 2では現れた**（27→28）。
- MCPサーバーが受けた要求は**8件＝ハンドシェイク2回分**（`server/discover`・`initialize`・`notifications/initialized`・`tools/list`が2周）。
- **落ちていたことは次の起動に持ち越されていない**。**起動し直せば繋がる**。

## 3. この記録が示すこと

- **「1回繋がらなかったサーバー」は、状態として残らない**。**次の起動で普通に繋がる**。
- したがって**構成の実態は実行のたびに変わり得る**。**前回の観測（繋がった／繋がらなかった）を今回の根拠にできない**。
- 機密の観点では、**「昨日は繋がらなかったから今日も出ていない」とは言えない**。**送出先は毎回確かめる必要がある**。

## 4. この記録が示さないこと

- **1つのプロセス内での再接続**（実行中にサーバーが落ちて復帰した場合）。本記録は**プロセスをまたぐ**形だけを見ており、`reconnect`の設定は既定のまま**中身を検証していない**。**この点はNOT RUN**。
- **起動後にサーバーが落ちた場合**の再試行回数・待ち時間。
- **`failOnStartupError: true`との組み合わせ**（[MCPの障害分離](../DSH_MCP_FAILURE_ISOLATION_2026-11-10/README.md)は起動時の片側障害のみ）。
- 実MCPサーバーへの接続、サーバー側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `late-start.json` | `overlay`、`turn1`（`serverState`・Tool数・MCP Tool）、`turn2`（同項目）、`toolCountDelta`、`mcpRequests`（8件のmethod・bytes）、`mcpRequestCount`、`handshakesObserved`、`modelRequests`（3件のbytes・SHA-256・Tool数・MCP Tool）、`reconnectWithinOneProcess`（`NOT RUN`） |

生の要求bodyはコミットしていない。**合成のみ**（`latemock`）。

再現する場合の要点:

```sh
# 1) MCPサーバーを止めたままTurn 1
dsh --profile headless --patch <overlay with url to a closed port> ... --json "Turn one."
# 2) MCPサーバーを起動する
# 3) 同じSessionを再開してTurn 2（必ずTurn 1と同じ作業ディレクトリで）
dsh --profile headless --patch <同じ overlay> --session-id <id> ... --json "Turn two."
```

- **Turn 2はTurn 1と同じ作業ディレクトリで実行する**。別ディレクトリだと`session ... was recorded in ...`で拒否される（本記録でも1回発生した）。
- 判定は**各Turnのモデル要求のTool一覧**と**MCPサーバーが受けたハンドシェイク回数**の両方で行う。

## 6. `H6-E2`への含意（判断はしない）

MCPの接続状態は**実行ごとに決まる**。**前回の観測を現在の構成の証拠にできない**ため、送出先の統制は**設定の固定**ではなく**毎回の確認**に向ける必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
