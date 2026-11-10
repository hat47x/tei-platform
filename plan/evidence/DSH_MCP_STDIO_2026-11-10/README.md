# MCPはstdioでも繋がり、子Processは環境を引き継ぐがAPIキーは渡らない（2026-11-10、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-10（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成MCPサーバーのみ**。実MCPサーバーへの接続は**NOT RUN**。
- 先行記録: [MCPの設定形](../DSH_MCP_CONFIGURED_2026-11-10/README.md)・[MCPの`tools/call`](../DSH_MCP_TOOL_CALL_2026-11-10/README.md)。本記録は、そこで未確定とした「**stdio transport**」を埋める。

## 1. 何をしたか

`mcp-client`を**`transport: stdio`**で挿し、`command: python`で自作の合成MCPサーバー（改行区切りのJSON-RPC）を起動した。サーバーは**受け取った行**と**自分に渡された環境変数**を記録する。モデルはローカルのMock。

## 2. 観測した結果

### ハンドシェイクは3行（`server/discover`が無い）

| # | method | 備考 |
| --- | --- | --- |
| 1 | **`initialize`** | `protocolVersion: 2025-11-25`、`clientInfo: dsh-mcp-client` |
| 2 | **`notifications/initialized`** | — |
| 3 | **`tools/list`** | — |

- **`server/discover`は送られていない**。streamable-httpでは最初に送られていた（[MCPの設定形](../DSH_MCP_CONFIGURED_2026-11-10/README.md)）ので、**transportで手順が違う**。

### モデルのToolは24→28

| 追加されたTool | 出どころ |
| --- | --- |
| **`mcp__stdiomock__stdio_echo`** | stdioのMCPサーバー |
| `list_mcp_resources`／`list_mcp_resource_templates`／`read_mcp_resource` | `mcp-resources` |

**名前空間の形は`mcp__<serverName>__<toolName>`で、streamable-httpと同じ**。

### 子Processの環境

| 項目 | 値 |
| --- | --- |
| **環境変数の数** | **49** |
| 作業ディレクトリ | **親と同じ**（`D:\GIT\hat47x\.tmp_h2`） |
| 合成canary `STDIO_ENV_CANARY_997` | **引き継がれた** |
| `DEEPSEEK_BASE_URL` | **引き継がれた** |
| **`DEEPSEEK_API_KEY`** | **渡っていない** |

- 引き継がれた中身は`USERPROFILE`・`APPDATA`・`PATH`・`SYSTEMROOT`など**通常のユーザー環境**で、そこに**任意の変数（canary）もそのまま載る**。
- **`DEEPSEEK_API_KEY`は子の環境に無い**。ハーネスは**Provider資格情報だけを外して**子を起動している。

## 3. この記録が示すこと

- **MCPはstdioでも繋がる**。`tools/list`まで通り、**モデルのTool一覧が増える**。
- **transportで手順が違う**（stdioは`server/discover`を送らない）。**同じ「MCPが繋がった」でも、外へ出る要求の数と順序は同じではない**。
- **stdioは「ローカルの任意のコマンドを起動する」経路**である。`command`に何を書くかで**実行される内容が変わる**。
- **子Processはハーネスの環境を引き継ぐ**。したがって**ハーネスの環境に置いた任意の値は、MCPサーバーから見える**。
- **ただし`DEEPSEEK_API_KEY`は渡らない**。**少なくともこの版では、Provider資格情報は子へ漏れていない**。

## 4. この記録が示さないこと

- **`env`設定で明示的に渡した場合**（本記録は`env`未指定で、継承のみを見ている）。
- **`cwd`設定**（本記録は既定）。
- **認証ヘッダ、再接続（`reconnect`）**、複数サーバー、`tools/call`のstdio経路。
- **`DEEPSEEK_API_KEY`が渡らない理由**（実装を読んでいない。観測事実のみ）。
- **他の資格情報**（Keychain・Credential Manager・SSOトークン等）が子から見えるか。
- 実MCPサーバーへの接続、サーバー側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `mcp-stdio.json` | `overlay`、`stdioLines`（3件のmethod・bytes・SHA-256）、`stdioMethods`、`stdioHasServerDiscover`、`envRunLines`、`childEnvironment`（`cwd`・`envKeyCount`・`envKeys`49件・`sensitive`の要約・`canaryPresent`／`baseUrlPresent`／`apiKeyPresent`）、`modelRequests`（Tool数・MCP Tool）、`toolCountBefore`／`toolCountAfter`、`mcpToolNames` |

**環境変数の値そのものは保存していない**（キー名・長さ・SHA-256の先頭16桁・判定フラグのみ）。生の要求bodyもコミットしていない。**合成のみ**（`STDIO_ENV_CANARY_997`・`STDIO_RESULT_CANARY_999`）。

再現する場合の要点:

```yaml
- insert:
    - id: mcp-client
      name: "@deepseek-ai/dsh-mcp-client"
      config:
        transport: stdio
        serverName: stdiomock
        command: python
        args: ["<server.py>", "<record-dir>"]
        failOnStartupError: false
```

```sh
DEEPSEEK_API_KEY=dummy STDIO_ENV_CANARY_997=canary dsh --profile headless \
    --patch <overlay> --permission-mode read-only --approval-policy never --json "Say ok."
```

- サーバー側で**受け取った行**と**自分の環境**を必ず記録する。**キー名だけ残し、値はハッシュと判定フラグに留める**。
- **`server/discover`の有無**でtransportを判別できる。

## 6. `H6-E2`への含意（判断はしない）

stdioのMCPは**ローカルで任意のコマンドを走らせる経路**であり、**子はハーネスの環境を引き継ぐ**。したがってMCPを許す判断は、**「どのサーバーに繋ぐか」だけでなく「どの環境を与えるか」**でもある。この版では**Provider資格情報は渡っていない**が、**他の値は渡る**ため、**ハーネスの環境に秘密を置かない**ことが前提になる。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
