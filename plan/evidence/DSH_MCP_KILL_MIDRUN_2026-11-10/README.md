# MCPサーバーを実行中に落としてもToolは一覧に残り続ける（2026-11-10、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-10（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [実行中の再接続](../DSH_MCP_MIDRUN_RECONNECT_2026-11-10/README.md)（落ちている→起動する）。本記録は**逆向き**（起動済み→落ちる）を埋める。

## 1. 何をしたか

1. **MCPサーバーを起動した状態**で実行を開始した。
2. モデルの**1手目で`pwsh`を呼ばせ**、その中で**MCPサーバーを停止**して10秒待たせた。
3. 同じ実行の中で次の要求を出し、**Tool一覧がどうなるか**を見た。

## 2. 観測した結果

| モデル要求 | MCPサーバー | Tool数 | MCP Tool |
| --- | --- | --- | --- |
| **001（開始時）** | 生きている | **28** | `mcp__latemock__header_echo` |
| 002（内部） | — | 0 | — |
| **003（停止後）** | **落ちている** | **28** | **`mcp__latemock__header_echo`** |

- `pwsh`の結果は`killed:36536`で、**実行後に対象ポートを listen しているProcessは0**だった（確かに落ちている）。
- **それでも003のTool一覧は28個のままで、落ちたサーバーのToolが残っている**。

## 3. この記録が示すこと

- **サーバーが落ちても、そのサーバーのToolは一覧から消えない**。**広告された面は縮まない**。
- したがって**「Tool一覧に在る」は「繋がっている」の証拠にならない**。呼んだ時点で初めて失敗する。
- 前記録（落ちている→起動する）と合わせると、**一覧は増えるが減らない**。**実行中の構成変化は一方向にしか観測できない**。

## 4. この記録が示さないこと

- **落ちた後にToolを実際に呼んだ場合**の挙動（エラーの形・待ち時間・再試行）。本記録は**呼んでいない**。
- **`reconnect`が諦めた後**（`maxAttempts`超過）に一覧から消えるか。本記録の10秒では諦めていない可能性がある。
- **サーバーが復帰した場合**に同じToolが戻るか（前記録の逆向きの続き）。
- `stdio`の子Processが落ちた場合。
- 実MCPサーバー、サーバー側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `kill-midrun.json` | `overlay`、`killEvidenceLines`（`killed:<pid>`を含む行）、`serverAliveAfterKill`、`firstModelRequest`／`lastModelRequest`、`toolSurvivedServerDeath`、`handshakeRequestsBeforeKill`・`mcpRequests`、`modelRequests`、`contrast`（前記録との向きの対比） |

生の要求bodyはコミットしていない。**合成のみ**（`latemock`）。

再現する場合の要点:

```sh
# 1) MCPサーバーを起動した状態で開始する
# 2) モデルの1手目で、そのポートを listen しているProcessを落とさせる
#    $p = (Get-NetTCPConnection -LocalPort <port> -State Listen).OwningProcess
#    Stop-Process -Id $p -Force ; Start-Sleep -Seconds 10
# 3) 同じ実行の次の要求でTool一覧を見る
```

- **Process一覧を`CommandLine`で全走査しない**。`Get-CimInstance Win32_Process`の全件走査は実行環境によって**job runnerごと落ちる**（本記録でも1回失敗した）。**`Get-NetTCPConnection`でポートの持ち主を取る**方が安定する。
- 判定は**停止後にポートを listen しているProcessが0であること**と、**その後のTool一覧**の両方で行う。

## 6. `H6-E2`への含意（判断はしない）

MCPの**広告された面は、サーバーが死んでも残る**。**「今のTool一覧」を「今の到達先」と同一視できない**。機密ProfileでMCPを統制するなら、**一覧ではなく実際の接続を観測する**必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
