# MCPサーバーが落ちてもHarnessは止まらず、落ちた側だけが外れる（2026-11-10、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-10（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [MCPの複数サーバー](../DSH_MCP_TWO_SERVERS_2026-11-10/README.md)。本記録は、そこで未確定とした「**片方のサーバーが落ちた場合**」を埋める。

## 1. 何をしたか

**元気なサーバー1つ**（`127.0.0.1:8799`）と、**閉じたポートを指すサーバー1つ**（`127.0.0.1:8798`）を同時に挿し、**`failOnStartupError`を false / true の2通り**で動かした。

## 2. 観測した結果

| 条件 | 終了コード | 警告 | モデルが見たTool | MCP Tool |
| --- | --- | --- | --- | --- |
| **`failOnStartupError: false`** | **0** | **なし** | **28個** | **`mcp__serverok__header_echo`** |
| **`failOnStartupError: true`** | **0** | **`1 entry did not activate`** | **28個** | **`mcp__serverok__header_echo`** |

- **どちらもHarnessは起動し、処理を完了した**（exit 0）。
- **落ちたサーバーのToolは現れず、元気なサーバーのToolは現れた**。
- `failOnStartupError: true`では**stderrに警告と当該エントリのエラー**が出た。

```
dsh: warning: 1 entry did not activate
mcp-client-dead (@deepseek-ai/dsh-mcp-client): Error: mcp-client(serverdead): initial connection or tool synchronization failed
```

- **落ちたサーバーへの要求は0件**（元気なサーバーは4件のハンドシェイクを完了）。

## 3. この記録が示すこと

- **MCPサーバーの障害はHarness全体を止めない**。**落ちた側のエントリだけが外れ、他は動き続ける**。
- **`failOnStartupError: true` は「プロセスを止める」ではなく「そのエントリを有効化しない」**。**警告は出るが実行は続く**。
- したがって**「設定したサーバーが繋がっている」を、終了コードでは判定できない**。**exit 0でも、外れたエントリがあり得る**。
- 機密の観点では、**静かに外れたサーバーがあると「送出先は設定どおり」と思い込む危険がある**。**警告を見ないと構成の実態と食い違う**。

## 4. この記録が示さないこと

- **`reconnect`による復帰**（一時的な切断から戻るか）。本記録は起動時の失敗のみ。
- **起動後に落ちた場合**（本記録は最初から繋がらないケース）。
- **全サーバーが落ちた場合**（本記録は片方だけ）。
- **`failOnStartupError`の既定値**（本記録は明示指定のみ。実装の既定は未確認）。
- **警告が機械可読な出力（`--json`のイベント）にも出るか**（本記録ではstderrの人間向け文言として観測）。
- 実MCPサーバーへの接続、サーバー側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `mcp-failure-isolation.json` | `softRun`（`failOnStartupError`・exit・Tool数・MCP Tool・`warningSeen`）、`hardRun`（同項目＋`entryErrorSeen`・`entryErrorLines`）、`healthyServerRequests`（4件のmethod）、`deadServerRequestCount`、`modelRequests`（4件のbytes・SHA-256・Tool数・MCP Tool） |

生の要求bodyはコミットしていない。**合成のみ**（`serverok`／`serverdead`）。

再現する場合の要点:

```yaml
- insert:
    - id: mcp-client-ok
      name: "@deepseek-ai/dsh-mcp-client"
      config: { transport: streamable-http, serverName: serverok, url: "http://127.0.0.1:8799/mcp/ok" }
    - id: mcp-client-dead
      name: "@deepseek-ai/dsh-mcp-client"
      config: { transport: streamable-http, serverName: serverdead, url: "http://127.0.0.1:8798/mcp/dead",
                failOnStartupError: <false|true> }
```

- **閉じたポートを指す**だけで「落ちたサーバー」を作れる。
- 判定は**終了コードではなく**、**stderrの`did not activate`**と**モデルが見たTool一覧**の両方で行う。

## 6. `H6-E2`への含意（判断はしない）

MCPの障害は**部分的に吸収される**。これは運用上は望ましいが、**「設定したのに繋がっていない」が静かに起きる**ことを意味する。機密ProfileでMCPの送出先を統制するなら、**設定の内容ではなく、実際に有効化されたエントリ**を観測する必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
