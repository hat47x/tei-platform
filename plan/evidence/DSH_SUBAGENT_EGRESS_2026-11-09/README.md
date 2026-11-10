# Subagent起動は別Sessionを作り、そのSessionも同じSession Log拡張を送る（2026-11-09、#2515 H6-E2 / #2509）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-09（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）
- 状態: **ローカルMock Endpointのみ**。実Providerへの送出は**NOT RUN**。台帳の未確定「Telemetry・MCP・Subagent経路の網羅」のうち、**Subagent経路の一部**を埋める。

これまでの記録は、Harness本体・Tool・子Process・Session Logの経路を測ってきたが、**Subagent**は未観測だった。台帳は「Telemetry・MCP・Subagent経路の網羅」を未確定として挙げている。本記録はそのうち**Subagent起動の1経路**を測る。

## 1. 何をしたか

`subagent` Toolを1回だけ強制するMockを立て、4 Tool呼出しのSessionの代わりに**1回のSubagent起動**を行わせた。親Sessionと子Sessionの**両方**について、送出された要求と永続Logを採取した。合成canary `SUBAGENT_PROMPT_CANARY_931` を`prompt`に入れ、どこまで運ばれるかを追った。

## 2. 観測した結果

### 親Sessionの要求（5件）

| 要求 | Tool数 | メッセージ数 | `system` bytes | `afterSeq`→`throughSeq` | canary |
| --- | --- | --- | --- | --- | --- |
| 001 | **24** | 1 | 2,800 | `-1`→`17` | なし |
| 002（題名生成） | 0 | 1 | **487** | `-1`→`17` | なし |
| 003 | 24 | 3 | 2,800 | `17`→`26` | **あり** |
| 004（**子Session**） | 24 | 1 | 2,800 | `-1`→`13` | **あり** |
| 005 | 24 | 5 | 2,800 | `26`→`35` | **あり** |

- **Session IDは2種類**。親 `…459fbf754114` と子 `…1a51e62c3f8b`。004だけが子の要求で、`afterSeq`が`-1`から始まる＝**別Sessionとして最初から送っている**。
- **子Sessionも`dsh_session_log`拡張を送る**（`sessionFormatVersion 4`、14 event）。
- **Canaryは親の003・005と子の004の両方に載る**。つまり`prompt`は**親の要求にも子の要求にも運ばれる**。

### 永続Log（親と子）

| | 親 | 子 |
| --- | --- | --- |
| 圧縮bytes | 15,135 | 12,728 |
| 展開bytes / 行 | 41,644 / 41行 | **36,774 / 19行** |
| `delivery-accepted` | **4件** | **1件** |
| `subagent`系event | `subagent/catalog` | `subagent/descriptor` |
| Canary | あり | あり |

- **子は独立した永続Logを持つ**。親のLogに子の会話は畳み込まれず、子のLogに親の会話も無い（別ディレクトリ）。
- 子にも`delivery-accepted`がある＝**子のSession Log拡張も送出される**。

## 3. この記録が示すこと

- `subagent`起動は**別Sessionを作り**、そのSessionも**同じSession Log拡張を送出する**。したがって「1回のTask」で外へ出る量は、親Sessionの分だけではなく**子Sessionの分も加算される**。
- `prompt`は**親と子の両方**に載る。同じcanaryが2つのSessionで観測された。
- 子のLogは親と別に保存される。**子の分を親のLogだけ見て数えると過小評価になる**。

## 4. この記録が示さないこと

- **Telemetry（OTel）** 経路、**MCP**経路。本記録はSubagent Toolの1経路だけ。
- **`subagent_fork`**（会話を継承する子）。本記録は`subagent`（継承しない子）だけ。forkでは親の会話が子へ載る可能性があり、**別に測る必要がある**。
- 子が**さらに子を起動**した場合の連鎖。
- 実Providerへの送出、Provider側の保存・保持期間、Crash後の重複送信。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `subagent-surface.json` | 5要求のbyte数・SHA-256・Tool数・メッセージ数・`system` bytes・Session ID・水位・canary出現。`distinctSessionIds`と`sessionOrder`。 |
| `subagent-sessions.json` | 親と子の永続Logの圧縮/展開bytes・行数・event種別内訳・canary出現・`delivery-accepted`の有無。 |

生の要求bodyはコミットしていない。**合成canaryのみ**。

再現する場合の要点:

```sh
python mock_provider_subagent.py 8799 <record-dir>
DEEPSEEK_API_KEY=dummy DEEPSEEK_BASE_URL=http://127.0.0.1:8799 \
  dsh --profile headless --permission-mode read-only --approval-policy never --json "Launch a subagent."
```

Mockは**`tools`を伴う最初の要求**で`subagent`を強制する。最初の要求は`tools`を伴わない（経路の事前確認）ため、単純な「1回目」判定では発火しない。子の永続Logは`~/.dsh/sessions/<cwd-slug>/<child-id>/session.v4.jsonl.zstd`に別ディレクトリで置かれる。

## 6. `H6-E2`への含意（判断はしない）

「Taskあたりの送出量」を見積もるときは、**親Sessionだけでなく子Sessionの分も数える**必要がある。子は独立したLogを持つため、親のLogを見るだけでは**子の送出を見落とす**。Telemetry・MCP・`subagent_fork`は未観測であり、経路の網羅はまだ完了していない。どの経路までを機密Profileの対象にするかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
