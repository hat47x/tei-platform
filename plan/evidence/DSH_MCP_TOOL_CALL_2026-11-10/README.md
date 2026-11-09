# MCPの`tools/call`はモデルが作った引数をそのまま外へ出し、結果を次の入力へ戻す（2026-11-10、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-10（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルMock MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [MCPの設定形](../DSH_MCP_CONFIGURED_2026-11-10/README.md)。本記録は、そこで未確定とした「**`tools/call`を実行した場合の送出**」を埋める。

## 1. 何をしたか

1. [MCPの設定形](../DSH_MCP_CONFIGURED_2026-11-10/README.md)で特定した形（`transport: streamable-http`・1インスタンス1サーバー）で`mcp-client`を挿した。
2. モデルに**`mcp__synthetic__synthetic_echo`を1回呼ばせ**、引数に合成canary `MCP_ARG_CANARY_994` を入れた。
3. MockのMCPサーバーは`tools/call`に**`MCP_RESULT_CANARY_995`**を返す。
4. **MCPサーバーが受け取った要求**と、**その後のモデル要求**を数えた。

## 2. 観測した結果

### MCPサーバーが受けた要求は5件

| # | method | bytes | 引数のcanary |
| --- | --- | --- | --- |
| 1 | `server/discover` | 280 | — |
| 2 | `initialize` | 163 | — |
| 3 | `notifications/initialized` | 54 | — |
| 4 | `tools/list` | 46 | — |
| 5 | **`tools/call`** | **123** | **あり** |

### `tools/call` の中身

```json
{"method":"tools/call",
 "params":{"name":"synthetic_echo","arguments":{"text":"MCP_ARG_CANARY_994"}},
 "jsonrpc":"2.0","id":2}
```

- **モデルが作った引数が、そのまま外へ出ている**（`arguments.text` にcanaryが入った）。
- **ワイヤ上のTool名は`mcp__synthetic__synthetic_echo`ではなく`synthetic_echo`**。名前空間（`mcp__<serverName>__`）は**モデル向けの見え方**で、MCPサーバーへは元の名前で送られる。

### 結果は次のモデル入力へ戻る

| モデル要求 | bytes | 引数canary | 結果canary |
| --- | --- | --- | --- |
| 001（初回） | 99,900 | — | — |
| 002 | 57,372 | — | — |
| **003** | 52,633 | **あり** | **あり** |

- **3回目のモデル要求に、送った引数と受け取った結果の両方が載っている**。Tool結果として会話に入り、次の入力の一部になる。

## 3. この記録が示すこと

- **MCPの`tools/call`は、モデルが生成した引数をそのまま外部のMCPサーバーへ送る**。DSH側で引数を加工・検閲している形にはなっていない。
- **MCPサーバーの応答はTool結果としてモデル入力に戻る**。つまり**外部から持ち込んだ内容が次の推論の文脈になる**。
- **名前空間はモデル向けの見た目**で、ワイヤ上は元のTool名。**監査や許可判定を名前で行うなら、どちらの名前を見るかを決める必要がある**。

## 4. この記録が示さないこと

- **引数の検閲・上限**（大きさ・禁止項目）。本記録は1件の小さな引数のみ。
- **`stdio` transport**、認証ヘッダ、再接続、**複数サーバー**の相互作用。
- **`mcp-resources`の`read_mcp_resource`**など、他のMCP系Toolの送出。
- 実MCPサーバーへの接続、サーバー側の保存・保持。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `mcp-call.json` | `toolNameModelFacing`／`toolNameOnWire`、`mcpRequests`（5件のmethod・bytes・SHA-256・paramsキー・引数canaryの有無）、`methodsSeen`、`toolsCall`、`modelRequests`（3件のbytes・Tool数・MCP Tool・引数/結果canaryの有無・水位）、`argumentCanary`／`resultCanary`、`argumentLeft`・`resultReturned` |

生の要求bodyはコミットしていない。**合成のみ**（`MCP_ARG_CANARY_994`・`MCP_RESULT_CANARY_995`）。

再現する場合の要点:

```sh
# MCPクライアントを正しい形で挿し、モデルにMCP Toolを1回呼ばせる
python mock_provider_mcp_call.py 8799 <record-dir> mcp__synthetic__synthetic_echo
dsh --profile headless --patch <mcp-client overlay> --permission-mode read-only \
    --approval-policy never --json "Call the MCP tool."
```

- モデル向けのTool名は**`mcp__<serverName>__<toolName>`**。Mockはその名前が`tools`に現れたときだけ強制する。
- **ワイヤ上の名前は`<toolName>`だけ**。両方を記録しておくと取り違えない。

## 6. `H6-E2`への含意（判断はしない）

MCPを開くと、**モデルが作った引数がそのまま外部へ出て、外部の応答が文脈に戻る**。したがってMCPは**「読む経路」ではなく「往復する経路」**として扱う必要がある。機密ProfileでMCPを許すなら、**どのサーバーを許すか**に加えて、**引数の検査をどこで行うか**と**応答を文脈に入れてよいか**を分けて決める必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
