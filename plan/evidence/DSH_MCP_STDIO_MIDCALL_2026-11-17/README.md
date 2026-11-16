# stdioの子Processが`tools/call`の途中で死ぬと`MCP connection closed`になる（2026-11-17、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-17（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.3`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [stdioの子Processの死](../DSH_MCP_STDIO_CHILD_DEATH_2026-11-17/README.md)（`tools/list`直後に落ちる場合）。本記録は、そこで未確認とした「**`tools/call`の途中で落ちた場合**」を埋める。

## 1. 何をしたか

stdioのMCPサーバーが**ハンドシェイクと`tools/list`に答え**、**`tools/call`を受け取った直後に`exit(2)`**するようにした。**応答は書かずに落ちる**。モデルにそのToolを呼ばせた。

## 2. 観測した結果

### 子は`tools/call`を受けてから落ちた

```
started → server/discover → started → initialize → notifications/initialized
        → tools/list → tools/call → exiting(on-call)
```

- **`tools/call`は92 bytesで届いている**。**子はそれに答えずに終了**した（`exit 2`）。
- **起動は2回、終了は1回**（`childSpawnCount: 2`・`childExitCount: 1`）。

### 呼び出しの結果

```
status: error
result: Error: Error: MCP connection closed
```

- **`MCP connection closed`** になった。
- **実行全体は`exit 0`**で終わり、モデルのTool数は **28 → 0 → 28**。

## 3. 落ち方で3種類のエラーになる（この記録の要点）

| 落ち方 | 出るエラー | 出典 |
| --- | --- | --- |
| **stdio・呼ぶ前**に死ぬ | **`Error: unknown tool`**（`UNKNOWN_TOOL`） | [stdioの子Processの死](../DSH_MCP_STDIO_CHILD_DEATH_2026-11-17/README.md) |
| **stdio・呼び出し中**に死ぬ | **`Error: MCP connection closed`** | **本記録** |
| **http・サーバーを落とす** | **`Error: fetch failed`** | [サーバーを落とした場合](../DSH_MCP_KILL_MIDRUN_2026-11-10/README.md) |

- **同じ「落ちた」でも、3つとも別の文言になる**。
- **「繋がらない」を1つの文字列で判定できない**。**経路とタイミングの両方で変わる**。
- 機密の観点では、**監視や検知をエラー文言に依存させると取りこぼす**。

## 4. この記録が示すこと

- **`tools/call`の途中で子が死ぬと `MCP connection closed`** になる。**`unknown tool` とは別**。
- **呼び出しは失敗するが、実行は止まらない**（`exit 0`・Toolは次の要求で戻る）。
- **失敗の現れ方は「経路」と「タイミング」の2軸で変わる**。**片方だけ測って一般化できない**。

## 5. この記録が示さないこと

- **子が`tools/call`に**部分的に応答してから**落ちる場合**（本記録は応答前に落ちる）。
- **子がハングする場合**（応答も終了もしない）。
- **`toolCallTimeoutMs` を超えた場合**の文言（本記録は即時終了）。
- **非stdio経路で呼び出し中に落ちる場合**。
- **再起動の回数上限**。
- 実MCPサーバー、サーバー側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 6. 生データ

| ファイル | 内容 |
| --- | --- |
| `stdio-midcall-death.json` | `overlay`、`childBehaviour`（応答するmethod列・`tools/call`受信後に`exit(2)`・子側の全ログ）、`childSpawnCount`・`childExitCount`、`toolCall`、`toolResult`／`toolResultText`、`modelToolCounts`、`failureShapes`（3経路の対比）、`threeDistinctShapes`、`runExitCode`、`modelRequests`、`caveat` |

生の要求bodyはコミットしていない。**合成のみ**（`stdiomock`・`dying_echo`）。

再現する場合の要点:

```yaml
- insert:
    - id: mcp-client-stdio-midcall
      name: "@deepseek-ai/dsh-mcp-client"
      config:
        transport: stdio
        serverName: stdiomock
        command: python
        args: ["<script>", "<record-dir>", "on-call"]
        toolCallTimeoutMs: 5000
        failOnStartupError: false
```

- 子は**`tools/call`を受け取ってから**落とす（**応答を書かせない**）。
- **モデルにToolを呼ばせる**。呼ばないと、この失敗形は出ない。
- **3経路を並べて比べる**。1つだけ見ると、その文言が一般形に見えてしまう。

## 7. `H6-E2`への含意（判断はしない）

**落ちたことを示す文言は経路とタイミングで変わる**（`unknown tool`／`MCP connection closed`／`fetch failed`）。したがって**エラー文言を検知の根拠にする統制は、測った組み合わせの外で取りこぼす**。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
