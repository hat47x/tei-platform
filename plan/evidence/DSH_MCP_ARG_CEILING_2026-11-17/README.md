# MCPの引数は12MBでも切られない。4.1MBの上限は無い（2026-11-17、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-17（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.3`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [MCPの引数4.1MB](../DSH_MCP_ARG_SIZE_4M_2026-11-10/README.md)（4.1MBまで切り詰めなし）。本記録は、そこで未確定とした「**上限の有無**」を**12MBで**確かめる。

## 1. 何をしたか

モデルに`mcp__bigargmock__bigarg_echo`を1回呼ばせ、引数`text`に**canary + 12,000,000 bytesの`z`**を入れた。**先行記録の4.1MBの約3倍**、**Telemetry既定`maxRequestBytes` 4,000,000の3倍**。

## 2. 観測した結果

| 項目 | 値 |
| --- | --- |
| 要求した引数 | **12,000,000 bytes** |
| **受け取った引数** | **12,000,016 bytes**（canary接頭辞16 bytes込み） |
| canary（先頭） | **あり** |
| **末尾100文字が全て`z`** | **はい** |
| **切り詰め** | **なし** |
| `tools/call`全体（**サーバーで実測**） | **12,000,118 bytes** |
| 引数が入った後の**モデル要求** | **12,052,319 bytes** |
| 実行の所要時間（**計測はしていない**） | 5.3s |

- **要求した大きさがそのまま届いた**。**先頭も末尾も無傷**である。
- **Telemetryの既定上限（4,000,000）を約8MB上回っている**。**Telemetryの上限はMCPの引数を制限しない**。
- **文脈にも残る**。引数が入った後のモデル要求は**12,052,319 bytes**になった（**`tools/call`より大きい**のは、**Tool定義と結果も同じ要求に載る**ため）。

### 大きさの推移

| 引数 | 記録 |
| --- | --- |
| 200 KB | [MCPの引数の大きさ](../DSH_MCP_ARG_SIZE_2026-11-10/README.md) |
| 1 MB | 同上 |
| 4.1 MB | [MCPの引数4.1MB](../DSH_MCP_ARG_SIZE_4M_2026-11-10/README.md) |
| **12 MB** | **本記録** |

**4点とも切り詰めなし**。**12MBまで上限は見えていない**。

## 3. この記録が示すこと

- **MCPの引数に4.1MBの上限は無い**。**12MBがそのまま通る**。
- **Telemetryの`maxRequestBytes`は、MCPの引数の大きさに関係しない**（**別の統制**である）。
- 機密の観点では、**1回のTool呼び出しで12MBを外部へ出せる**。**「大きすぎるから止まる」という期待は置けない**。
- 同時に**文脈も12MB膨らむ**。**コストと遅延は上限ではなく別の場所で効く**。

### 訂正（2026-11-17）

初版は `tools/call` の大きさを **12,052,319 bytes** と書いていたが、これは
**引数が入った後のモデル要求の大きさを取り違えて転記したもの**だった。
**サーバーで実測した `tools/call` は 12,000,118 bytes** である。
**2つは別の測定値**なので分けて載せた。

## 4. この記録が示さないこと

- **12MBを超える上限の有無**。**本記録は1点だけ**で、**「上限が無い」とは言えない**。
- **実Provider側の上限**（本記録はモックのみ）。
- **MCPサーバー側の受信上限**（本記録のサーバーは無条件に受ける）。
- **時間・メモリの限界**（5.3sだったが計測していない）。
- **複数の大きな引数を同時に送った場合**。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `mcp-arg-ceiling.json` | `overlay`、`argCanary`、`requestedArgBytes`／`receivedArgBytes`、`canaryPrefixPresent`、`tailUnchanged`、`truncated`、`toolsCallRequestBytes`、`telemetryDefaultMaxRequestBytes`、`exceedsTelemetryDefaultBy`、`modelRequests`、`largestModelRequestBytes`、`contextGrowth`、`priorRecords`、`answers`、`caveat` |

生の要求bodyはコミットしていない。**合成のみ**（`BIGARG_CANARY_5X`）。

再現する場合の要点:

```yaml
- insert:
    - id: mcp-client-bigarg
      name: "@deepseek-ai/dsh-mcp-client"
      config:
        transport: streamable-http
        serverName: bigargmock
        url: "http://127.0.0.1:8793/mcp/big"
        toolCallTimeoutMs: 60000
        failOnStartupError: false
```

- **サーバー側で受け取った引数の長さ・先頭canary・末尾**を記録する。**長さだけでは「途中で切れた」を区別できない**。
- **`toolCallTimeoutMs`を大きくする**。既定では12MBの往復に足りない可能性がある。
- **モデル側は引数を分割してSSEで流す**。1フレームに12MBを載せると別の問題を作る。

## 6. `H6-E2`への含意（判断はしない）

**引数の大きさは統制されていない**（12MBまで）。したがって**「大きすぎる引数は止まる」前提を置けない**。外部Effectを持つToolでは、**1回の呼び出しで持ち出せる量**を別に制限する必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
