# MCPの引数は4.1MBでも切られず、Telemetryの上限とは無関係に外へ出る（2026-11-10、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-10（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [MCPの引数の大きさ](../DSH_MCP_ARG_SIZE_2026-11-10/README.md)（200 KB と 1 MB を確認）。本記録は、そこで未確定とした「**1 MBを超える場合**」を埋める。

## 1. 何をしたか

モデルに`mcp__synthetic__synthetic_echo`を1回呼ばせ、引数`text`に**合成canary + 4,100,000 bytesの`z`**を入れた。**Telemetry側`maxRequestBytes`の既定4,000,000を超える大きさ**を選んでいる。

## 2. 観測した結果

| 項目 | 値 |
| --- | --- |
| 要求した引数 | **4,100,000 bytes** |
| **受け取った引数** | **4,100,018 bytes** |
| canary | **あり** |
| 末尾100文字が全て`z` | **はい** |
| `tools/call`全体 | **4,100,123 bytes** |
| **結果後のモデル要求** | **4,149,793 bytes** |

- **要求した大きさ＋canary接頭辞（18 bytes）がそのまま届いた**。**切り詰めも拒否も無い**。
- **`maxRequestBytes`の既定4,000,000を超えている**。つまり**Telemetryの上限はMCPの引数を制限しない**。
- 結果が返った後のモデル要求は**約4.15 MB**になった（引数が文脈に残るため）。

### 3つの大きさでの比較

| 引数 | 受け取った引数 | 切り詰め |
| --- | --- | --- |
| 200,000 bytes | 200,018 bytes | 無し |
| 1,000,000 bytes | 1,000,018 bytes | 無し |
| **4,100,000 bytes** | **4,100,018 bytes** | **無し** |

## 3. この記録が示すこと

- **MCPの引数に、少なくとも4.1 MBまでの上限は無い**（この版の観測）。**DSHは引数を切らずに外部のMCPサーバーへ渡す**。
- **Telemetryの`maxRequestBytes`（既定4,000,000）は、MCPの引数を制限しない**。**別の経路なので、片方の上限をもう片方の歯止めにできない**。
- **引数は次のモデル入力にも残る**ため、**1回の大きな呼び出しが以後の要求を4 MB級にする**。

## 4. この記録が示さないこと

- **4.1 MBを超える場合**。本記録はこの大きさまでしか試していない。**「上限が無い」とは言えない**。
- **実Providerが4.15 MBの要求を受け付けるか**（ローカルMockでのみ確認）。
- **MCPサーバー側の受信上限**（本記録のMockは制限していない）。
- **引数の内容の検査**（禁止項目・秘密情報の検出など）。本記録は大きさだけを見ている。
- **`maxInstructionBytes`**（MCPクライアントの設定）との関係。
- `stdio` transport、認証、複数サーバー、実MCPサーバー。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `arg-size-4m.json` | `requestedArgBytes`、`argumentCanary`／`resultCanary`、`mcpRequests`（5件のbytes・SHA-256・method・argTextBytes・canary・末尾判定）、`toolsCall`、`modelRequests`（5件のbytes・SHA-256・canaryの有無）、`largestModelRequestBytes`、`telemetryMaxRequestBytesDefault`、`exceedsTelemetryDefault`、`comparison`（3つの大きさ・切り詰め無し） |

生の要求bodyはコミットしていない（4 MB級のため保存もしない）。**合成のみ**（`BIGARG_CANARY_996`・`BIGARG_RESULT_997`・`z`の繰り返し）。

再現する場合の要点:

```sh
# 引数の大きさを4.1MBにして1回呼ばせる（TelemetryのmaxRequestBytes既定4,000,000を超える）
python mock_provider_mcp_bigarg.py 8799 <record-dir> mcp__synthetic__synthetic_echo 4100000
dsh --profile headless --patch <mcp-client overlay> --permission-mode read-only \
    --approval-policy never --json "Call the MCP tool with a very large argument."
```

- **サーバー側で受け取った引数のバイト数と末尾**を必ず記録する。全体のbytesだけでは切り詰めを見落とす。
- **結果後のモデル要求**も測る。引数はそこにも載る。

## 6. `H6-E2`への含意（判断はしない）

MCPの引数は**Telemetryの上限とは独立**に、**少なくとも4.1 MBまで外へ出る**。機密Profileで量を抑えたいなら、**経路ごとに別の統制が要る**（Telemetryの`maxRequestBytes`はMCPには効かない）。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
