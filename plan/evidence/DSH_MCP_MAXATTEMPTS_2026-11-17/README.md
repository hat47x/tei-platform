# `maxAttempts`を使い切ると静かに諦める（警告も出ず、Toolは現れない）（2026-11-17、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-17（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [reconnect無効](../DSH_MCP_RECONNECT_DISABLED_2026-11-16/README.md)（試行の有無）、[実行中の再接続](../DSH_MCP_MIDRUN_RECONNECT_2026-11-10/README.md)（既定で復帰）。本記録は、そこで未確定とした「**`maxAttempts`を超えて諦めた後**」を埋める。

## 1. 何をしたか

**常に500を返すMCPエンドポイント**を立て、`reconnect`を**小さい値**（`initialDelayMs 200`・`maxDelayMs 800`・**`maxAttempts 3`**）にした。再試行の**回数と間隔**をサーバー側で時刻つきに記録した。

## 2. 観測した結果

### 試行は4回で止まった

| # | 時刻 | 前回からの間隔 | method |
| --- | --- | --- | --- |
| 1 | 14.828s | — | `server/discover` |
| 2 | 15.063s | **+0.235s** | `server/discover` |
| 3 | 15.469s | **+0.406s** | `server/discover` |
| 4 | 16.281s | **+0.812s** | `server/discover` |

- **間隔は約235ms → 406ms → 812ms**。**おおむね倍々だが、厳密な倍ではない**（406/235 ≒ **1.73倍**、812/406 = **2.00倍**）。**設定値の200/400/800msに対する上振れは35ms・6ms・12ms**で、**一律ではない**。
- **`maxAttempts: 3` で試行は4回**（初回1回＋再試行3回）。
- **4回目の後に、記録した範囲では来ない**（この実行の観測窓で、それ以上の要求は無い）。

### 諦めたあとが静か

| 項目 | 結果 |
| --- | --- |
| モデルが見たTool数（要求ごと） | **27 → 0**（MCP Toolは**どちらも0**） |
| MCP Toolが現れたか | **最後まで現れない** |
| **警告** | **出ない**（`did not activate` も `warning` も無し） |
| **終了コード** | **0** |

- **[同じserverNameの重複](../DSH_MCP_DUPLICATE_NAME_2026-11-12/README.md)では `serverName "..." is already in use` が出た**が、**ここでは何も出ない**。
- **設定の誤り（名前の重複）は声を上げ、接続の失敗（予算切れ）は黙る**。

## 3. この記録が示すこと

- **`maxAttempts` は「初回＋N回の再試行」**として効く（3 → 4試行）。
- **間隔は `initialDelayMs` から伸びる**。**設定は200→400→800ms**だが、**実測は235→406→812ms**で、**おおむね倍々で、上振れは35ms・6ms・12msと一律ではない**。**`maxDelayMs: 800` に対して実測812msなので、頭打ちの効き方は本記録では確かめられない**。
- **予算を使い切った後の失敗は観測できない**。**終了コード0・警告なし・Tool一覧に変化なし**なので、**外から見ると「そのサーバーを設定していない」のと区別がつかない**。

## 4. この記録が示さないこと

- **他の`maxAttempts`値**（本記録は3のみ。掃引していない）。
- **`maxDelayMs` の頭打ち**が実際に効く領域（本記録は800msで止まる前）。
- **諦めた後に時間をおいて再試行するか**（同一プロセス内では来なかったが、次の起動は未確認）。
- **諦めたことを状態として取得する方法**（警告が無いことしか分かっていない）。
- **再接続時の`headers`再送**。
- **`stdio`の子Process**。
- 実MCPサーバー、サーバー側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `maxattempts-exhausted.json` | `overlay`（`serverName`・`url`・`reconnect`の3値）、`server`（常に500・起動マーカー）、`attempts`（4件の時刻・間隔・method・bytes）、`attemptCount`・`configuredMaxAttempts`、`observedDelays`、`backoffPattern`、`mcpToolEverReachedModel`、`modelToolCounts`、`activationWarningPrinted`・`warningLines`、`runExitCode`、`modelRequests`、`contrast`、`caveat` |

生の要求bodyはコミットしていない。**合成のみ**（`failmock`）。

再現する場合の要点:

```yaml
- insert:
    - id: mcp-client-500
      name: "@deepseek-ai/dsh-mcp-client"
      config:
        transport: streamable-http
        serverName: failmock
        url: "http://127.0.0.1:8796/mcp/fail"
        failOnStartupError: false
        reconnect:
          initialDelayMs: 200
          maxDelayMs: 800
          maxAttempts: 3
```

- **サーバー側で時刻を記録する**。間隔はクライアント側のログに出ないため、**受け手で測る**必要がある。
- **`dsh`の標準エラーも捕捉する**（`2>&1`）。警告の有無が判定対象なので、捨てると確かめられない。

## 6. `H6-E2`への含意（判断はしない）

`maxAttempts` は**回数と間隔の統制としては効く**が、**使い切った後の失敗は静か**である。機密ProfileでMCPを統制するなら、**「繋がらなかった」を終了コードや警告で検知できない**前提に立ち、**有効化されたサーバー数を別に数える**必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
