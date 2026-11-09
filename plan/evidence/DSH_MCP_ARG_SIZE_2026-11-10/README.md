# MCPの引数は200KBでも1MBでも切られず、そのまま外部へ出る（2026-11-10、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-10（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルMock MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [MCPの`tools/call`](../DSH_MCP_TOOL_CALL_2026-11-10/README.md)。本記録は、そこで未確定とした「**引数の検閲・上限**」に着手した結果である。

## 1. 何をしたか

モデルに`mcp__synthetic__synthetic_echo`を1回呼ばせ、引数`text`に**合成canary + 指定バイト数の`z`**を入れた。**2条件**を測った。

| 条件 | 要求した引数 |
| --- | --- |
| A | **200,000 bytes** |
| B | **1,000,000 bytes** |

MCPサーバー側で**受け取った引数のバイト数**と**末尾が切られていないか**を記録した。

## 2. 観測した結果

| 条件 | 要求した引数 | **受け取った引数** | canary | 末尾が全て`z` | `tools/call`全体 |
| --- | --- | --- | --- | --- | --- |
| A | 200,000 | **200,018** | あり | **はい** | 200,123 bytes |
| B | 1,000,000 | **1,000,018** | あり | **はい** | 1,000,123 bytes |

- **どちらも要求した大きさ＋canary接頭辞（18 bytes）がそのまま届いた**。**切り詰めも拒否も無い**。
- 末尾100文字がすべて`z`であることを確認した（後ろが削られていない）。
- `tools/call`の要求全体は引数＋約105 bytes（JSON-RPCの枠）。

### 次のモデル要求が膨らむ

| 条件 | モデル要求1 | モデル要求2 | **モデル要求3** |
| --- | --- | --- | --- |
| A（200KB） | 99,992 | 57,464 | **1,052,647** |
| B（1MB） | 100,012 | 57,484 | **5,052,652** |

- **Tool結果が返った後の要求が、引数の分だけ大きくなる**。Bでは**約5 MB**になった。
- つまり**1回の大きな引数が、以後のモデル入力にも積み上がる**。

## 3. この記録が示すこと

- **MCPの引数に、少なくとも1 MBまでの上限は無い**（この版の観測）。**DSHは引数を切らずに外部のMCPサーバーへ渡す**。
- **引数は次のモデル入力にも残る**ため、**1回の呼び出しが以後の要求を大きくする**。
- 機密の観点では、**モデルが作った内容を、上限なしで外部のMCPサーバーへ送れる**。**「小さいはず」という前提は置けない**。

## 4. この記録が示さないこと

- **1 MBを超える場合**。本記録は1 MBまでしか試していない。**「上限が無い」とは言えない**。
- **Provider側の上限**（モデル要求が5 MBになったが、実Providerが受け付けるかは未確認）。
- `maxInstructionBytes`（MCPクライアントの設定）や`maxRequestBytes`（Telemetry側）との関係。
- **引数の内容の検査**（禁止項目・秘密情報の検出など）。本記録は大きさだけを見ている。
- `stdio` transport、認証、複数サーバー、実MCPサーバー。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `arg-size.json` | `argumentCanary`／`resultCanary`、`runs`（2条件の`requestedArgBytes`・`mcpRequests`・`toolsCall`（bytes・argTextBytes・canary・末尾判定）・`modelRequests`（3件のbytes・SHA-256・canaryの有無）） |

生の要求bodyはコミットしていない（1 MB級のため保存もしない）。**合成のみ**（`BIGARG_CANARY_996`・`BIGARG_RESULT_997`・`z`の繰り返し）。

再現する場合の要点:

```sh
# 引数の大きさを変えて2回測る（200 KB / 1 MB）
python mock_provider_mcp_bigarg.py 8799 <record-dir> mcp__synthetic__synthetic_echo 200000
dsh --profile headless --patch <mcp-client overlay> --permission-mode read-only \
    --approval-policy never --json "Call the MCP tool with a large argument."
```

- サーバー側で**受け取った引数のバイト数**と**末尾**を必ず記録する。全体のbytesだけでは切り詰めを見落とす。
- **Tool結果が返った後のモデル要求**も測る。引数はそこにも載る。

## 6. `H6-E2`への含意（判断はしない）

MCPを開くと、**モデルが作った引数が上限なしで外部へ出る**（本記録の範囲では1 MBまで）。しかも**その内容は以後のモデル入力にも残る**。機密ProfileでMCPを許すなら、**どのサーバーを許すか**だけでなく、**引数の大きさ・内容をどこで検査するか**を決める必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
