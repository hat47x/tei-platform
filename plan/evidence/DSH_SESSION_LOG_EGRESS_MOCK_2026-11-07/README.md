# DSHのセッションログ拡張が実送出する内容の局所計測（2026-11-07、#2509 / TAI #232）

- 対象Issue: [#2509](https://github.com/hat47x/tei-platform/issues/2509)の「送出」項目。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-07（Asia/Tokyo）
- 実行環境: Windows 10.0.26100、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルのMock Endpointだけで計測**。実Providerへの送出、実Egress経路の制御、OS/Network隔離は**NOT RUN**。

この記録は、DSHの拡張field（`dsh_session_log`）が**HTTP要求へ何を載せるか**を、外部へ出さずに測ったものである。DSHの設計意図や公開資料の記述を検証するものではなく、この版・このProfileでの実測に限る。

## 1. 何をしたか

1. `127.0.0.1`で動くMock Endpoint（Anthropic Messages形式のSSEを返す小さなHTTP Server）を用意し、`DEEPSEEK_BASE_URL`でDSHの`deepseek-official`経路の宛先をそこへ向けた。Credentialはダミー値を環境変数で渡した。
2. DSH headlessを次の4通りで実行し、受け取った要求bodyを保存した。
   - `session-log-deepseek.enabled: true`で新規Session（2要求: agent呼出しと題名生成）
   - 同じSessionのまま`false`へ切り替えて再開
   - 同じSessionのまま`true`へ戻して再開
   - 別の隔離Rootで、`read` ToolをMock側から強制し、Tool結果が次の要求に載るかを観測（実行と再開で2要求）
3. 要求bodyを**構造だけ**（鍵名・byte数・digest・拡張fieldの水位）へ落とし、生bodyは保存していない。生bodyにはagent instructions全文とSession Logが入るため、リポジトリへ入れない。

## 2. 観測した結果

| 実行 | 要求body bytes | Tool数 | モデル入力 bytes | `dsh_session_log` |
| --- | --- | --- | --- | --- |
| enabled・新規Session・1回目 | 87,474 | 24 | 37,626 | 45,487 bytes、`afterSeq -1` → `throughSeq 18`、19 event |
| enabled・新規Session・2回目（題名生成） | 51,152 | 0 | 707 | 同じ45,487 bytes、同じ水位（**送り直し**） |
| 同じSessionを`false`で再開 | 43,121 | 24 | 37,777 | **field無し** |
| 同じSessionを`true`へ戻して再開 | 90,179 | 24 | 37,928 | 48,800 bytes、`afterSeq 18` → `throughSeq 47`、29 event |

- 拡張fieldの鍵は`afterSeq`・`throughSeq`・`events`・`session`・`sessionFormatVersion`・`version`。`session`は`id`・`createdAt`・`cwd`を持つ。
- `events`には`permission/preset`・`sandbox/mode`・`approval/policy`（実行Profile）、`user/message`（**指示本文を含む**）、`system/message`（**system promptを含む**）、`request/header`・`request/context`、`session/title`・`session/title-llm-request`、`turn/start`・`step/start`などが入る。
- `false`へ切り替えた時点でfieldは消え、`true`へ戻すと**止めた間の未送信分だけ**が`afterSeq 18`から再開して送られた。再有効化で全会話を送り直す挙動は観測していない。
- 2回目の要求（題名生成）は同じ水位のfieldを再度載せていた。同じ内容が2回送られる。

## 3. Tool結果がどこへ載るか

`read` Toolを強制し、ローカルのファイルへ合成marker `WIRE_MARKER_20261107105334_DO_NOT_EGRESS`（45 bytes）を置いて読ませた。

| 要求 | markerの出現 |
| --- | --- |
| Tool実行前の要求 | 0 |
| Tool実行後（結果を返す要求） | **モデル入力に1回**（`tool_result`として。これは通常のTool往復）**＋ `dsh_session_log`に2回** |
| 同じSessionの再開後 | 0（水位が進んでおり、その区間は再送されない） |

`dsh_session_log`の2回は、同じ`tool/result` eventの`data.message`（Tool結果本文の392 bytes）と`data.meta`（176 bytes）に分かれて載っている。つまり**Tool結果はモデル入力とは別に、Session Logとしても外部へ送られる**。この要求1件の中で、同じ内容が3箇所（モデル入力1・Session Log 2）に現れた。

これはTool結果の全byteが常に送られるという主張ではない。この計測では、読み取った本文がそのまま載ることを観測した。

## 4. この記録が示すこと

- DSH `0.2.2-rc.1` headless・`deepseek-official`経路で、`dsh_session_log`拡張がHTTP要求に載る。ここでは`enabled: true`を明示した。DSH付属のREADMEは既定`true`と述べているが、この版の既定値そのものは本記録では単独確認していない（既定のProfileで実Providerへ出る計測をしていない）。
- `enabled: false`でfieldは消え、`true`へ戻すと止めた区間から再開する（`afterSeq`/`throughSeq`の水位で確認）。
- Tool結果はモデル入力に加えてSession Logにも載り、外部へ送出される。
- Session Logには利用者の指示本文・system prompt・実行Profile（sandbox mode・approval policy）・cwdが入る。

## 5. この記録が示さないこと

- 実Provider（`api.deepseek.com`）への送出、Provider側の保存・保持期間・二次利用。
- Telemetry、MCP、Subagent、別Provider（`llm-pi-ai`の他経路）の送出。`llm-pi-ai`経路（OpenRouter）では同じ拡張fieldを載せなかったが、これは経路ごとの差の一例にすぎず、全経路の有無を網羅した結果ではない。
- Egress制御の実効性（OS Firewall・Proxy・Network名前空間での拒否）。
- 再有効化時のbacklogが上限`maxBytes`を超えた場合の分割送出（本計測のfieldは最大48,800 bytesで、8 MiB上限に達していない）。
- 要求のin-flight中に設定を変えた場合の扱い（本計測は要求完了後に切り替えた）。
- TAI #232の機密Profile適合、Owner受理、#1119のGate。

## 6. 生データ

| ファイル | 内容 |
| --- | --- |
| `wire-shape.json` | 8要求の構造（鍵名・byte数・digest・拡張fieldの水位・event種別） |
| `marker-egress.json` | markerの出現位置と、載ったeventのfield別byte数 |

生の要求bodyは**リポジトリへ入れない**。各要求のSHA-256は`wire-shape.json`の`requestSha256`に記録しており、同じMock Endpointと同じProfileで再実行すれば突合できる。Mock Endpointは`127.0.0.1`だけで待ち受け、要求bodyをローカルのディレクトリへ保存し、Anthropic Messages形式の固定SSEを返す。

再現する場合の要点:

```sh
# Mock Endpoint（ローカルのみ、要求bodyを保存）
python mock_provider.py 8799 <record-dir>
# DSHをMockへ向けて実行（Credentialはダミー、実Providerへは出ない）
DEEPSEEK_API_KEY=dummy DEEPSEEK_BASE_URL=http://127.0.0.1:8799 \
  dsh --profile headless --patch <overlay.yml> --permission-mode read-only --approval-policy never --json "<task>"
```

`<overlay.yml>`は`session-log-deepseek.enabled`だけを切り替える。`--session-id`で同じSessionを再開できる。

## 7. 機密境界への含意（判断はしない）

TAI #232は、機密Rawを外部Harnessへ渡さないこと、`read-only`やAgentの自己抑制を保護の根拠にしないことを定める。この計測では、**Harnessが読んだTool結果が、モデル入力とは別のSession Log経路でも外部へ出る**ことを確認した。機密Sourceを読ませた場合の送出量は「モデルのコンテキストに入った分」ではなく、**Session Logの複製を含めて数える必要がある**。どの範囲なら許容できるかはOwnerの判断であり、本記録は判定しない。
