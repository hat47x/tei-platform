# 落ちたMCPサーバーのToolは一覧に残り、呼ぶと通信エラーで失敗する（2026-11-10、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-10（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [サーバーを落とした場合](../DSH_MCP_KILL_MIDRUN_2026-11-10/README.md)（一覧が縮まないこと）。本記録は、そこで**NOT RUN**とした「**呼んだら何が返るか**」を埋める。

## 1. 何をしたか

1. MCPサーバーが生きている状態で開始した。
2. モデルの**1手目で`pwsh`を呼ばせ**、そのポートを listen しているProcessを落とさせた（結果は`killed:45404`）。
3. 同じ実行の**2手目で、残っているMCP Toolを実際に呼ばせた**。

## 2. 観測した結果

| 順 | できたこと |
| --- | --- |
| 1手目 | `pwsh`で`killed:45404`（**サーバー停止**） |
| **2手目** | **`mcp__latemock__header_echo`を呼び、`status: error`・`Error: fetch failed`** |
| 3手目 | モデルは`done`と応答 |

- **Toolは一覧に残っていた**（呼ぶ時点でもTool数28・`hasTargetTool: true`）。
- **呼び出しは通信エラー（`fetch failed`）で失敗**した。**Tool が見つからない旨のエラーではなく、接続できないエラー**である。
- 実行は**exit 0で完了**し、ハーネスは止まらなかった。

## 3. この記録が示すこと

- **落ちたサーバーのToolは、呼べる場所に残っている**。**一覧は当てにならない**（前記録と同じ）。
- **失敗は「呼んだ時点」で初めて分かる**。形は**`fetch failed`という汎用の通信エラー**で、**サーバーが落ちていることを名指す文言ではない**。
- 機密の観点では、**「繋がっていないように見えない」**。**失敗の形から、どのサーバーが落ちているかを読み出すのが難しい**。

## 4. この記録が示さないこと

- **`reconnect`が諦めた後**（`maxAttempts`超過）に一覧から消えるか。**本記録の1回の失敗では消えていない**。
- **時間をおいて再試行した場合**（`fetch failed`の後に復帰するか）。
- **`stdio`の子Processが落ちた場合**のエラー形。
- **エラーがSession Log・Telemetryへどう載るか**。
- 実MCPサーバー、サーバー側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `call-after-kill.json` | `overlay`、`killThenCallTrace`（`pwsh`→`tool_result`→`mcp__...`呼び出し→エラーの行）、`toolCallErrorText`（`Error: fetch failed`）、`callFailed`、`toolStillAdvertisedAtCallTime`、`serverAliveAfterKill`、`modelRequests`（4件のbytes・SHA-256・Tool数・`hasTargetTool`・`hasKillResult`・`hasCallResult`）、`contrast` |

生の要求bodyはコミットしていない。**合成のみ**（`latemock`・`CALL_AFTER_KILL_CANARY`）。

再現する場合の要点:

```sh
# 1) MCPサーバーを起動した状態で開始
# 2) 1手目でポートの持ち主を落とす（Get-NetTCPConnection でPIDを取る）
# 3) 2手目で、まだ一覧にある MCP Tool を呼ぶ
```

- **Mock の手順制御は「Tool一覧を持つ要求だけ」で進める**。Tool一覧の無い要求（タイトル生成など）で段階を進めると、**同じTool呼び出しを無限に繰り返す**（本記録でも最初の試行がこれで暴走し、数千件の要求が出た）。
- 判定は**2手目の`tool_result`の`status`と文言**で行う。

## 6. `H6-E2`への含意（判断はしない）

MCPの障害は**呼んだ時にしか見えない**うえ、**その形は汎用の通信エラー**である。機密ProfileでMCPを統制するなら、**一覧や正常終了では足りず、呼び出しの成否と宛先を結び付けて観測する**必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
