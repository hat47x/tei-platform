# `maxAttempts`を使い切っても、次の起動では繋がる（2026-11-18、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-18（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.3`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [maxAttemptsの消化](../DSH_MCP_MAXATTEMPTS_2026-11-17/README.md)（予算切れは静か）。本記録は、そこで未確定とした「**時間をおいた再試行**」を埋める。

## 1. 何をしたか

**同じoverlay**（`reconnect.maxAttempts: 1`）で**2回起動**した。

| 実行 | MCPサーバー |
| --- | --- |
| **RUN 1** | **落ちている**（ポート未listen） |
| **RUN 2** | **起動済み**（あいだに開始） |

## 2. 観測した結果

| 項目 | RUN 1（サーバー無し） | RUN 2（サーバーあり） |
| --- | --- | --- |
| 起動前のポートlisten | **0** | **1** |
| モデルが見たTool数 | **27 → 0** | **28 → 0** |
| MCP Tool | **現れない** | **`mcp__latermock__later_echo`** |
| サーバーが受けた要求 | **0** | **4** |

- **RUN 1は予算を使い切って諦めた**（`maxAttempts: 1`）。**Toolは現れない**。
- **RUN 2は同じ設定のまま繋がった**。**サーバーが受けたのは4件**（`server/discover`→`initialize`→`notifications/initialized`→`tools/list`）で、**いずれも実行の早い時点**に届いている。**再試行が何回起きたかは測っていない**（**4件は正常なハンドシェイクの並び**であり、**「1回目の試行で成功」とは言えない**）。
- **差はTool数1個**（27 → 28）で、**それがMCPのTool**である。

## 3. この記録が示すこと

- **この1組では、予算切れは「その実行の中」で回復せず**、**次の起動では繋がった**。**一般則として「次の起動なら必ず繋がる」とは言えない**（**overlay・サーバー・予算は各1つ**）。
- **この1組では、`maxAttempts` の小ささが「恒久的な失敗」にならなかった**。**実行をまたいだ状態を持たないらしい**と読めるが、**それは推論**である（**設定値を変えた比較はしていない**）。
- 機密の観点では、**「前回繋がらなかったサーバー」が次の実行で黙って繋がる**。**実行ごとに有効なサーバー集合を数え直す**必要がある。
- **`exit 0` のまま**なので、**前回の失敗は次の実行の手がかりにならない**（[maxAttemptsの消化](../DSH_MCP_MAXATTEMPTS_2026-11-17/README.md)と同じ静かさ）。

## 4. この記録が示さないこと

- **同一プロセス内で長く待てば回復するか**（本記録は**別プロセス**でのみ確認）。
- **`maxAttempts` を大きくした場合**との比較（本記録は1）。
- **サーバーが途中で落ちて戻った場合**（[実行中の再接続](../DSH_MCP_MIDRUN_RECONNECT_2026-11-10/README.md)は別記録）。
- **何回連続で失敗すれば「恒久的に使えない」と判断できるか**。
- 実MCPサーバー、サーバー側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `later-run-recovery.json` | `overlay`（`maxAttempts: 1`）、`run1_serverDown`（listen無し・Tool数27→0・MCP Tool無し・サーバー要求0）、`run2_serverUp`（listen有り・**サーバー起動マーカー**・Tool数28→0・**MCP Tool有り**・要求4件）、`requests`（4件のmethod・bytes・到達時刻）、`modelRequests`、`answers`、`caveat` |

生の要求bodyはコミットしていない。**合成のみ**（`latermock`）。

再現する場合の要点:

```yaml
- insert:
    - id: mcp-client-later
      name: "@deepseek-ai/dsh-mcp-client"
      config:
        transport: streamable-http
        serverName: latermock
        url: "http://127.0.0.1:8792/mcp/later"
        reconnect: { initialDelayMs: 200, maxDelayMs: 400, maxAttempts: 1 }
        failOnStartupError: false
```

- **同じoverlayで2回起動する**。1回目はサーバーを落とし、2回目の前に起動する。
- **起動前後の`listen`状態を確認する**。これを確かめないと「繋がらなかった」と「サーバーが無かった」を混同する。
- **Tool数を比べる**。**27と28の差がMCPの1個**であることを確かめる。

### 手法上の注意

**モデル側Mockの記録ディレクトリをRUN 2の前に消す**。消さないと**RUN 1の要求が混ざり**、どちらの結果か分からなくなる。

## 6. `H6-E2`への含意（判断はしない）

**予算切れは実行単位の失敗**であり、**次の実行では回復する**。したがって**「一度繋がらなかったサーバー」を恒久的に除外した扱いにしない**し、逆に**「前回繋がったから今回も繋がる」とも言えない**。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
