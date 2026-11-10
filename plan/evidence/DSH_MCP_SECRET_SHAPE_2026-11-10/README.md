# MCPの引数は秘密らしい内容でも検査されず、そのまま外へ出る（2026-11-10、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-10（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [MCPの`tools/call`](../DSH_MCP_TOOL_CALL_2026-11-10/README.md)・[MCPの引数の大きさ](../DSH_MCP_ARG_SIZE_2026-11-10/README.md)。本記録は、そこで未確定とした「**引数の内容の検査**」を埋める。

## 1. 何をしたか

モデルに`mcp__synthetic__synthetic_echo`を1回呼ばせ、引数`text`に**秘密らしい形の文字列**を7種入れた。MCPサーバーが受け取った内容をそのまま比較した。

使ったのは**すべて合成の見本**で、実Credentialではない。

| 種類 | 入れた文字列の形 |
| --- | --- |
| AWSアクセスキーの形 | `AKIAIOSFODNN7EXAMPLE`（AWSの文書にある見本） |
| GitHubトークンの形 | `ghp_` + 36文字 |
| OpenSSH秘密鍵のヘッダ | `-----BEGIN OPENSSH PRIVATE KEY-----` |
| APIキーの形 | `sk-live-...` |
| パスワード代入 | `password=Hunter2Example` |
| Bearerトークンの形 | `Authorization: Bearer eyJ...` |
| 目印 | `SECRETSHAPE_CANARY_1002` |

## 2. 観測した結果

**7種すべてが、MCPサーバーに一字も変わらず届いた。**

| 種類 | サーバーで見つかった |
| --- | --- |
| AWSアクセスキーの形 | **あり** |
| GitHubトークンの形 | **あり** |
| OpenSSH秘密鍵のヘッダ | **あり** |
| APIキーの形 | **あり** |
| パスワード代入 | **あり** |
| Bearerトークンの形 | **あり** |
| 目印 | **あり** |

- 届いた`tools/call`は**318 bytes**で、**マスク・伏字・拒否のいずれも無い**。
- **同じ文字列が、結果後のモデル要求にもそのまま載っている**（7種中6種を確認。目印を含めると全種）。

## 3. この記録が示すこと

- **DSHはMCPの引数を内容で検査しない**。**秘密らしい形でも、そのまま外部のMCPサーバーへ渡る**。
- **伏字・切り詰め・警告も無い**。**「危ない内容は止まる」という前提は置けない**。
- **引数はモデル入力にも残る**ので、**一度出した内容は、以後の要求にも載り続ける**。
- したがって**機密の扱いは、経路側の検査ではなく、入力を作る側（モデルに何を渡すか）と宛先（どのサーバーを許すか）で決める必要がある**。

## 4. この記録が示さないこと

- **DSH以外の層（MCPサーバー側・ネットワーク側・DLP）の検査**。本記録はハーネスの手元だけを見ている。
- **実際のCredential**（本記録はすべて合成の見本）。
- **`headers`や`env`に入れた場合**の検査（[MCPの`headers`](../DSH_MCP_HEADERS_2026-11-10/README.md)は送られることのみ確認）。
- **Session Log・Telemetry側での伏字**（本記録はMCP経路のみ）。
- **他版での挙動**（この版の観測であって契約ではない）。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `secret-shape.json` | `shapeCanary`、`noteOnSafety`、`patternsSent`、`patternsUnchangedAtServer`（7種の真偽）、`allPatternsUnchanged`、`toolsCallBytes`・`toolsCallSha256`、`mcpRequests`（5件）、`modelRequests`（3件のbytes・SHA-256・一致した形の数）、`patternsInLatestModelRequest` |

**値そのものは記録に残していない**（種類名と真偽のみ）。生の要求bodyもコミットしていない。**合成のみ**。

再現する場合の要点:

```sh
# 引数に秘密らしい形を入れて1回呼ばせる（必ず合成の見本を使う）
python mock_provider_mcp_secret.py 8799 <record-dir> mcp__synthetic__synthetic_echo
dsh --profile headless --patch <mcp-client overlay> --permission-mode read-only \
    --approval-policy never --json "Call the MCP tool."
```

- **実Credentialを使わない**。AWSの文書にある見本のような**公知のダミー**で足りる。
- 判定は**サーバーが受け取った本文に形が残っているか**で行う。**マスクされていれば文字列は一致しない**。

## 6. `H6-E2`への含意（判断はしない）

MCP経路には**内容の検査が無い**。機密ProfileでMCPを許すなら、**「秘密は検出されて止まる」を前提にできない**。統制は**宛先の限定**と**入力の作り方**に置く必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
