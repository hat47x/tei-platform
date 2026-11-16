# 接続拒否は`reconnect`が再試行して復帰する（HTTP 500との違いは先行記録）（2026-11-17、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-17（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.3`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [500と再試行](../DSH_MCP_RECONNECT_HEADERS_2026-11-17/README.md)（HTTP 500は再試行しない）。本記録は、そこで残した「**トランスポート層の障害**」を埋める。

## 1. 何をしたか

**起動時はポートを開かず（接続拒否）**、**6.0秒後に listen を開始する**サーバーを用意した。`reconnect` は `initialDelayMs 300`・`maxDelayMs 1500`・**`maxAttempts 12`**。`headers` に canary を入れた。

## 2. 観測した結果

### 再試行して、繋がった

| 項目 | 値 |
| --- | --- |
| サーバーが listen を開始 | **t=6.0s** |
| クライアントの**最初の要求** | **t=19.797s** |
| 差 | **13.797s** |
| 得られたTool数 | **28**（`mcp__faultmock__fault_echo`。**2件目の要求はTool一覧が空**） |
| canary | **全4要求にあり** |

- **サーバーが上がる前に諦めていない**。**上がった後に繋がっている**。
- **`failOnStartupError: false` でも、実行は正常に完了**した。

### 再試行の回数は測れていない（当初の逆算は誤りだった）

失敗した再試行は**サーバー側に痕跡を残さない**（そもそも listen していない）。**回数は直接測れない**。

**当初「13.797s ≒ 11回分」と書いたが、これは誤りだった。** 理由は、**backoffの起点を取り違えていた**ことにある。

| 項目 | 値 |
| --- | --- |
| backoff の起点 | **クライアントの起動時**（サーバーの listen 開始時ではない） |
| 全12回の累計 | **15.6s** |
| 最初の要求の到達 | **19.797s** |
| 言えること | **15.6s < 19.797s** であり、**クライアントの起動時刻が不明** |

- **12回分の累計15.6sは、要求が届いた19.797sより短い**。**クライアントがいつ起動したか分からない**ので、**回数は復元できない**。
- 正しく言えるのは **「6.0秒の停止を越えて再試行し、繋がった」** までで、**回数は未測定**である。

## 3. 500との違い（500側は先行記録の観測で、本記録は測っていない）

| 障害 | 再試行 | 結果 |
| --- | --- | --- |
| **接続拒否**（応答が返らない） | **する** | **復帰した**（28 Tool） |
| **HTTP 500**（応答は返る） | **しない** | 停止／呼び出し失敗のみ |

- **同じ「繋がらない」でも、`reconnect` が効くものと効かないものがある**。
- **「`reconnect` を設定しておけば不調から回復する」とは言えない**。**回復するのは接続できない場合**で、**サーバーがエラーを返す場合は回復しない**。
- 機密の観点では、**「サーバーが落ちた」と「サーバーが拒否した」を区別して監視する**必要がある。

## 4. この記録が示すこと

- **接続拒否は再試行され、復帰する**（**この1つの timing で**）。
- **HTTP 500 は再試行されない**（[先行記録](../DSH_MCP_RECONNECT_HEADERS_2026-11-17/README.md)）。
- **`headers` は復帰後の要求にも載っている**（**canary が全要求にあり**）。

## 5. この記録が示さないこと

- **失敗した再試行そのものに `headers` が載ったか**。**サーバーが listen していないので観測できない**。**「再送」を直接証明したわけではない**。
- **他の timing**（ダウン時間・backoff・`maxAttempts`）。**本記録は1組のみ**。
- **接続が途中で切れた場合**（**開始時に落ちている場合だけ**を測った）。
- **`4xx` の場合**、**複数サーバー**。
- 実MCPサーバー、サーバー側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 6. 生データ

| ファイル | 内容 |
| --- | --- |
| `mcp-transport-fault.json` | `overlay`、`headerCanary`、`fault`（接続拒否・6.0秒後に listen）、`requests`（4件のmethod・到達時刻・bytes・SHA-256・ヘッダ数・canaryの有無）、`firstRequestAtSeconds`、`secondsAfterServerCameUp`、`canaryOnEveryRequest`、`recovered`、`toolCountSeen`、`modelRequests`、`contrastWithHttp500`、`answersForTheOpenQuestion`、`retryInference`（backoff との整合計算）、`caveat` |

生の要求bodyはコミットしていない。**合成のみ**（`faultmock`・`TRANSPORT_FAULT_CANARY_881`）。

再現する場合の要点:

```yaml
- insert:
    - id: mcp-client-fault
      name: "@deepseek-ai/dsh-mcp-client"
      config:
        transport: streamable-http
        serverName: faultmock
        url: "http://127.0.0.1:8794/mcp/fault"
        reconnect: { initialDelayMs: 300, maxDelayMs: 1500, maxAttempts: 12 }
```

- **サーバーは「起動時に listen しない」状態から始める**。**接続拒否**を作るのが目的。
- **サーバー側に「listen を開始した時刻」を書かせる**。これが無いと**「遅れて繋がった」を言えない**。
- **失敗した再試行は痕跡を残さない**ので、**間隔と backoff の算術で整合を確認**する。**直接数えたと書かない**。

## 7. `H6-E2`への含意（判断はしない）

**`reconnect` が効くのは「接続できない」場合**で、**「サーバーがエラーを返す」場合は効かない**。したがって**可用性の担保を `reconnect` の設定だけに委ねることはできない**。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
