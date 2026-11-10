# MCP経路は既定の面に出ておらず、`mcp-client`を挿しても推測した設定では届かない（2026-11-09、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-09（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルMock Endpointのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [Telemetry経路](../DSH_OTEL_PATH_2026-11-09/README.md)。本記録は台帳の未確定「**MCP経路の網羅**」に着手した結果である。

## 1. 何をしたか

1. **既定の面**を測った。headlessの1要求に載るTool名を列挙し、MCPのToolがあるかを数えた。
2. **登録状況**を`--dump-config`と`--dump-config-schema`で確認した。
3. **`mcp-client`を`--patch`で挿し**、ローカルの合成MCPサーバー（JSON-RPCの`initialize`・`tools/list`・`resources/list`に応答するMock）へ向けて実行し、**MCPの要求が届くか**を数えた。

## 2. 観測した結果

### 既定の面にMCPのToolは無い

- 1要求に載るToolは**24個**で、内訳は`create_goal`・`edit`・`get_goal`・`glob`・`grep`・`interrupt_agent`・`job_kill`・`job_list`・`job_output`・`list_agents`・`pwsh`・`read`・`read_image`・`send_message`・`skill`・`subagent`・`subagent_fork`・`todo_write`・`update_goal`・`web_fetch`・`web_search`・`workflow`・`working_directory`・`write`。
- **名前に`mcp`を含むToolは0**。

### 登録はされているがサーバーが無い

- `mcp-resources`（`@deepseek-ai/dsh-mcp-resources`）は`@deepseek-ai/dsh-base`が**登録している**。
- ただし`--dump-config-schema`では**`configRef`が`unknownConfig`**で、**投影された設定schemaが無い**。既定の合成後の木にも**サーバー定義が無い**。

### `mcp-client`を挿しても届かなかった

| 項目 | 結果 |
| --- | --- |
| overlay（`servers.synthetic { transport: http, url: http://127.0.0.1:8799/mcp }`） | `--dump-config`は**受理** |
| 実行後のTool数 | **24**（変化なし） |
| 実行後のMCP Tool | **0** |
| Mockが受けた**MCP要求** | **0** |
| Mockが受けたモデル要求 | **2**（計143,308 bytes） |
| 親の永続Logに現れる`mcp`の回数 | **0** |

- **`--dump-config`が受理しても、実行時にMCPのToolも要求も現れなかった**。
- 警告・エラーはstdoutのイベントに出なかった（`status`の4件と`text`・`final`のみ）。

## 3. この記録が示すこと

- **既定のheadlessの面にMCP経路は出ていない**。モデルはMCPのToolを持たず、**MCP宛の要求も起きない**。
- `mcp-resources`は**登録済みだがサーバー未設定**で、**設定schemaが公開されていない**（`unknownConfig`）。
- **推測した設定では経路を開けられなかった**。したがって**「MCP経路が無い」とは言えない**が、**「既定では開いていない」とは言える**。

## 4. この記録が示さないこと

- **正しい設定を与えた場合にMCPが何を送るか**。`mcp-client`の設定形は**profile schemaに現れない**ため、本記録の0件は**「この推測が届かなかった」**ことを示すだけで、**MCPがデータを運べないことの証拠ではない**。
- MCPサーバー側の保存・保持、stdio transport、認証。
- `mcp-resources`が実際に何をするか（resource discovery/read）の実測。
- Telemetryのfeedback時の送出内容、web・tui面のfeedback経路。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `mcp-path.json` | `defaultSurface`（Tool数24・Tool名一覧・MCP Tool 0）、`mountedWithoutServers`（プラグイン名・登録元・`configSchema: unknownConfig`）、`clientInsertAttempt`（overlay内容・`acceptedByDumpConfig`・実行後のTool数とMCP Tool・`mcpRequestsReceived`）、`run`（2要求のbyte数・SHA-256・Tool名・Session ID・水位）、`parentLog`（bytes・SHA-256・event種別・`mcp`出現0）、`limits` |

生の要求bodyはコミットしていない。**合成のみ**（MCPのcanaryは`MCP_CANARY_993`、ただし**送信は0件**）。

再現する場合の要点:

```sh
# 既定の面を数える
dsh --profile headless --permission-mode read-only --approval-policy never --json "Say ok."
#   -> tools 24個、mcp を含む名前は0

# 登録と設定schemaを見る
dsh --profile headless --dump-config | grep -i mcp
dsh --profile headless --dump-config-schema   # mcp-resources は unknownConfig

# クライアントを挿して合成サーバーへ向ける（この形では届かなかった）
dsh --profile headless --patch <mcp-client overlay> --dump-config   # 受理される
dsh --profile headless --patch <mcp-client overlay> ... --json "Say ok."   # MCP要求は0
```

- **`--dump-config`の受理は、実行時に経路が開くことを意味しない**。Tool一覧と宛先の受信件数の**両方**を見る。
- `unknownConfig`は「設定が禁止」ではなく**「投影されたschemaが無い」**である。正しい形は別途特定が要る。

## 6. `H6-E2`への含意（判断はしない）

既定のheadlessの面では**MCPは開いていない**。`mcp-resources`は登録済みで、**サーバーを設定すれば経路が開き得る**が、**その設定形は公開されたschemaに無い**。機密ProfileでMCPを扱うなら、**経路の有無を既定のTool一覧で確認する**ことと、**設定形を特定したうえで送出を測る**ことを分けて行う必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
