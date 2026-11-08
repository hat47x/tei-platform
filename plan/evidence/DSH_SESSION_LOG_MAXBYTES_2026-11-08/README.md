# Session Log拡張の`maxBytes`上限：backlogの分割送出と停滞（2026-11-08、#2515 H6-E2 / P0-b）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`P0-b`。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-08（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルMock Endpointのみ**。実Providerへの送出、Provider側の保存、OS/Network境界の拒否は**NOT RUN**。

先行記録（[送出計測](../DSH_SESSION_LOG_EGRESS_MOCK_2026-11-07/README.md)、[既定・再試行・切断](../DSH_SESSION_LOG_RETRY_AND_DEFAULT_2026-11-07/README.md)）では、既定の`maxBytes`（8 MiB）に達しなかったため、**上限に当たったときの振る舞い**は未観測だった。本記録は上限を意図的に下げて、その振る舞いを短いSessionで観測したものである。

## 1. 何をしたか

`session-log-deepseek`の`maxBytes`を**2048**と**512**に下げ、出荷既定の`deepseek-official`経路の宛先を`DEEPSEEK_BASE_URL`でローカルMockへ向けて実行した。Credentialはダミー。

- `maxBytes=2048`: 新規Sessionで1往復、さらに同じSessionを再開して1往復
- `maxBytes=512`: 新規Sessionで1往復（上限が1 eventの塊より小さい場合）

## 2. 観測した結果

### `maxBytes=2048`

| 要求 | body bytes | Tool数 | `dsh_session_log` |
| --- | --- | --- | --- |
| 1回目（agent） | 34,384 | 24 | **1,393 bytes**、`-1`→`9`、10 event |
| 2回目（題名生成） | 7,946 | 0 | 同じ1,393 bytes、同じ水位 |

- 上限2,048に対し、fieldは**1,393 bytes**で収まった。`throughSeq`は`9`で、**その時点で送れる範囲だけ**が載った。
- 題名生成の要求は**直前の水位のまま再送**した（先行記録と同じ）。

### `maxBytes=512`

| 要求 | body bytes | Tool数 | `dsh_session_log` |
| --- | --- | --- | --- |
| 1回目 | 33,595 | 24 | **518 bytes**、`-1`→`2`、**3 event** |
| 2回目（題名生成） | 7,157 | 0 | 同じ518 bytes、同じ水位 |

- 上限512に対し、**field 518 bytes**が載った。ヘッダとenvelopeを含む直列化後の値で判定されるため、**event本文の総量が上限以下でも、field全体が上限を超え得る**。
- 水位は`2`で止まり、`3 event`だけが送られた。**以降のeventは送られていない**。

### 再開（`maxBytes=2048`、受理済みSession）

| 要求 | body bytes | Tool数 | `dsh_session_log` |
| --- | --- | --- | --- |
| 再開1回目 | 33,230 | 24 | **field無し** |

- 既に受理された水位（`9`）から先のeventが、**この要求では載らなかった**。先行記録で観測した「受理済み区間は再送されない」と整合する。
- この要求では新しいfieldが付かなかったため、**未送信分が残っている可能性を、本記録だけでは確定できない**（`maxBytes`を上げた再実行が必要）。

## 3. この記録が示すこと

- `maxBytes`は**1要求が運ぶfield全体の上限**であり、`afterSeq`/`throughSeq`の水位となって現れる。上限に達すると、**送れる範囲だけが載り、残りは次の受理済み要求へ持ち越される**。
- **event本文の合計が上限未満でも、field全体は上限を超え得る**。512の例では518 bytesが載った。上限を「イベント本文の予算」と読むと誤る。
- 上限が小さいと、**1要求で運べるevent数が減る**（2048で10 event、512で3 event）。その分、同じ内容を運ぶのに必要な要求数が増える。

## 4. この記録が示さないこと

- **上限を超えるbacklogが後続要求で分割送出されること**の実測。本記録のSessionは短く、未送信分を次要求で観測できていない（再開時のfieldが無かったため）。
- 単一eventが`maxBytes`を超えて**停滞する**条件（プラグインの警告文と挙動）。本記録では警告を観測していない。
- 実Providerへの送出、Provider側の保存・保持期間、`enabled`のin-flight変更、Crash後の重複送信。
- Telemetry、MCP、Subagent、`llm-pi-ai`他経路。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `maxbytes.json` | 3実行の要求ごとのbody byte数・SHA-256・Tool数・field byte数・水位・event数 |

生の要求bodyはコミットしていない。

再現する場合の要点:

```sh
# provider routeは出荷既定のまま（DEEPSEEK_BASE_URLでMockへ向ける）
python mock_provider.py 8799 <record-dir>
DEEPSEEK_API_KEY=dummy DEEPSEEK_BASE_URL=http://127.0.0.1:8799 \
  dsh --profile headless --patch <overlay with maxBytes> \
      --permission-mode read-only --approval-policy never --json "<task>"
```

`<overlay>`は`session-log-deepseek`の`enabled: true`と`maxBytes`だけを指定する。**provider経路を差し替えるoverlayと併用しない**（`DEEPSEEK_BASE_URL`は`deepseek-official`経路だけが読む）。

## 6. `H6-E2`への含意（判断はしない）

`maxBytes`は「1要求の送出量の上限」であり、**総開示量の上限ではない**。上限に当たった分は次の要求へ持ち越されるため、機密Sourceを扱う場合の見積りは「1要求あたり」ではなく、**セッション全体で何要求に分割されるか**で数える必要がある。分割送出と停滞の条件は本記録では確定できておらず、追加の実測が必要である。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
