# stdioの子Processが落ちてもToolは一覧に残り、呼ぶと`unknown tool`になる（2026-11-17、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-17（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH **`0.2.2-rc.3`**（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [MCPのstdio経路](../DSH_MCP_STDIO_2026-11-10/README.md)（接続と環境の継承のみ）。本記録は、そこで未確定とした「**stdioの子Processが落ちた場合のエラー形**」を埋める。

## 1. 何をしたか

**`tools/list`を返した直後に`exit 0`する**stdioのMCPサーバーを子Processとして起動し、モデルにそのToolを**呼ばせた**。

## 2. 観測した結果

### 子Processは起動し直された

| 項目 | 値 |
| --- | --- |
| 子Processの起動回数 | **4** |
| 子Processの終了回数 | **2** |
| 終了理由 | `after-list`（`tools/list`の直後） |

- **落ちた子は起動し直される**。1回目と2回目で、**同じハンドシェイクをやり直している**（`initialize` → `notifications/initialized` → `tools/list` → 終了）。
- つまり**「落ちたまま」にはならず、**毎回作り直して同じところで落ちる**。

### Toolは残るが、呼ぶと別のエラーになる

| 要求 | モデルが見たTool数 | `mcp__stdiomock__dying_echo` |
| --- | --- | --- |
| 1 | **28** | **あり** |
| 2 | 0 | なし |
| 3 | **28** | **あり** |

呼び出しの結果:

```
status: error
result: Error: unknown tool "mcp__stdiomock__dying_echo"
error: { name: ToolNotFoundError, code: UNKNOWN_TOOL }
```

### httpで落とした場合との違い

| 経路 | 一覧 | 呼んだとき |
| --- | --- | --- |
| **streamable-http**（[サーバーを落とした場合](../DSH_MCP_KILL_MIDRUN_2026-11-10/README.md)） | 残る | **`Error: fetch failed`**（通信エラー） |
| **stdio**（本記録） | **残る** | **`Error: unknown tool`**（`UNKNOWN_TOOL`） |

- **同じ「落ちた」でも、外から見えるエラーの種類が違う**。**stdioは通信エラーにならず、Toolが見つからない扱いになる**。
- 機密の観点では、**「呼べば通信エラーになる」を検知の根拠にできない**。**経路によって現れ方が変わる**。

## 3. この記録が示すこと

- **stdioの子Processが落ちても、Toolは一覧に残り続ける**（httpと同じ）。
- **呼ぶと `UNKNOWN_TOOL` になる**。**`fetch failed` ではない**ので、**httpの観測をstdioへそのまま当てはめられない**。
- **子Processは自動で起動し直される**。**同じ設定なら同じところで落ち、一覧は元に戻る**。

## 4. この記録が示さないこと

- **`tools/call`の途中で落ちた場合**（本記録は`tools/list`の直後に落ちる子）。
- **異常終了（非0）で落ちた場合**（本記録は`exit 0`）。
- **落ち続けた場合に再起動が止まるか**（回数の上限は未確認）。
- **`env`設定で値を明示した場合**、**`cwd`設定**。
- 実MCPサーバー、サーバー側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 版による違い（重要）

本記録の実行環境は **`0.2.2-rc.3`** で、先行記録の **`0.2.2-rc.1`** とは**版が違う**。そのため、**同じstdio経路でも観測が一致しない点がある**。

| 観測 | `rc.1`（先行記録） | `rc.3`（本記録） |
| --- | --- | --- |
| stdioでの`server/discover` | **送られない**（`stdioHasServerDiscover: false`） | **送られる**（`server-discover-probe-1`・280 bytes、子Processごとに1回） |

- **`server/discover`の有無は、版で変わりうる**。**先行記録の「stdioは`server/discover`を送らない」は`rc.1`の事実**であり、**現在の版にも成り立つとは限らない**。
- 本記録は**版を明記**し、**先行記録の値をそのまま現在の事実として引用しない**。

## 6. 生データ

| ファイル | 内容 |
| --- | --- |
| `stdio-child-death.json` | `overlay`（stdio・`serverName`・`command`・`args`）、`childBehaviour`（応答するmethod列・終了条件・子側の全ログ）、`childSpawnCount`・`childExitCount`・`exitReasons`、`serverDiscoverProbes`、`modelRequests`（3件のbytes・SHA-256・Tool数・MCP Tool）、`toolAdvertisedAt`、`toolCallOutcome`（`callId`・`status`・`result`・`error`）、`failureShape`（httpとの対比）、`recoveredByRespawn`、`caveat` |

生の要求bodyはコミットしていない。**合成のみ**（`stdiomock`・`dying_echo`）。

再現する場合の要点:

```yaml
- insert:
    - id: mcp-client-stdio-dying
      name: "@deepseek-ai/dsh-mcp-client"
      config:
        transport: stdio
        serverName: stdiomock
        command: python
        args: ["<script>", "<record-dir>", "after-list"]
        failOnStartupError: false
```

- 子Process側で**受け取った行を全て記録する**。**起動回数と終了回数は子側でしか数えられない**。
- **モデルにToolを呼ばせる**（一覧に載るだけでは、呼んだときのエラー形が分からない）。
- **`--version`で版を記録する**。**版が違うと経路の観測が変わる**。

## 7. `H6-E2`への含意（判断はしない）

stdioの子Processが落ちても**Toolは一覧に残り**、**呼ぶと `UNKNOWN_TOOL`** になる。**httpの`fetch failed`とは別の形**なので、**「落ちたことを検知する」手段を経路ごとに確かめる**必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
