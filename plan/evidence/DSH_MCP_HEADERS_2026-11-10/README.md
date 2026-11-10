# MCPの`headers`は毎回そのまま送られ、信頼確立前の最初の要求にも載る（2026-11-10、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-10（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [MCPの設定形](../DSH_MCP_CONFIGURED_2026-11-10/README.md)・[MCPのstdio経路](../DSH_MCP_STDIO_2026-11-10/README.md)。本記録は、そこで未確定とした「**認証ヘッダ**」を埋める。

## 1. 何をしたか

`mcp-client`を`transport: streamable-http`で挿し、**`headers`に合成canaryを入れた**。

```yaml
config:
  transport: streamable-http
  serverName: headermock
  url: "http://127.0.0.1:8799/mcp"
  headers:
    Authorization: "Bearer MCP_HEADER_CANARY_1001"
    X-Synthetic-Trace: "trace-1001"
```

サーバー側で**受け取ったHTTPヘッダ**を記録し、**どの要求に載ったか**を数えた。

## 2. 観測した結果

| 要求 | method | `Authorization`にcanary | ヘッダ数 |
| --- | --- | --- | --- |
| 1 | **`server/discover`** | **あり** | 11 |
| 2 | `initialize` | あり | 11 |
| 3 | `notifications/initialized` | あり | 11 |
| 4 | `tools/list` | あり | 11 |

- **4件すべてに載っている**。**最初の`server/discover`（サーバーの素性を知る前の要求）にも載る**。
- 受け取ったヘッダの集合は**11個**で、毎回同じ。

```
Authorization: <canary>
X-Synthetic-Trace: trace-1001
host / connection / content-type / content-length / accept / accept-encoding /
accept-language / sec-fetch-mode / user-agent: node
```

- **設定した2つ以外は、HTTPクライアントの標準ヘッダだけ**。Cookieや環境の資格情報が勝手に付く形ではない。
- **`user-agent`は`node`**で、HTTP層では`dsh`を名乗っていない。
- **canaryはモデル要求には現れない**（`headers`はモデル入力に載らない）。

## 3. この記録が示すこと

- **`headers`は毎回そのまま外部へ出る**。**「必要なときだけ送る」形ではない**。
- **信頼確立前の最初の要求にも載る**。サーバーが本物か確かめる前から、**設定した資格情報は渡っている**。
- したがって**`headers`に置く値は、そのサーバーに常時渡してよいものに限られる**。
- 一方で**余計なヘッダは付かない**（Cookie・環境変数・OS資格情報は載らない）。**渡した分だけが出る**。

## 4. この記録が示さないこと

- **`reconnect`時の再送**（本記録は1回の接続のみ）。
- **`stdio`の`env`設定**（[MCPのstdio経路](../DSH_MCP_STDIO_2026-11-10/README.md)は継承のみを見ており、`env`明示は未確認）。
- **複数サーバー**を同時に挿した場合の相互作用。
- **TLS**（本記録はHTTP）。**実サーバーの証明書検証**。
- **`headers`の値がログ・Session Logへ残るか**（本記録はモデル要求に載らないことのみ確認）。
- 実MCPサーバーへの接続、サーバー側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `mcp-headers.json` | `configuredHeaders`（canaryを含む設定値）、`headerCanary`、`mcpRequestCount`、`canarySentOnEveryRequest`、`requests`（4件のmethod・bytes・SHA-256・canaryの有無・ヘッダ名一覧・ヘッダ数）、`observedHeaderSet`（11個）、`modelRequests`（2件のbytes・SHA-256・canaryの有無）、`canaryInModelRequests` |

生の要求bodyはコミットしていない。**合成のみ**（`MCP_HEADER_CANARY_1001`）。

再現する場合の要点:

```yaml
- insert:
    - id: mcp-client
      name: "@deepseek-ai/dsh-mcp-client"
      config:
        transport: streamable-http
        serverName: headermock
        url: "http://127.0.0.1:8799/mcp"
        headers:
          Authorization: "Bearer <canary>"
          X-Synthetic-Trace: "<canary>"
```

- サーバー側で**受け取ったヘッダ名と、canaryがどのヘッダに載ったか**を記録する。**値をそのまま残すと秘密が入る**ため、canaryは伏せる。
- **最初の要求（`server/discover`）に載るか**を必ず見る。ここが「信頼確立前」の判定点になる。

## 6. `H6-E2`への含意（判断はしない）

MCPの`headers`は**常時送出**であり、**相手を確かめる前から渡る**。したがって**「必要になってから渡す」前提の資格情報を`headers`に置けない**。機密ProfileでMCPを許すなら、**どの値をどのサーバーへ常時渡してよいか**を先に決める必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
