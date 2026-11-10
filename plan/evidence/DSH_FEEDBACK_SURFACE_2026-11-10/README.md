# feedback面が無いのでTelemetryの送出は実測できない（2026-11-10、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-10（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルMock Endpointのみ**。実Providerへの送出は**NOT RUN**。
- 先行記録: [Telemetry経路](../DSH_OTEL_PATH_2026-11-09/README.md)。本記録は、そこで未確定とした「**feedbackを与えた場合の送出内容**」に着手し、**この環境では実測できない**ことを確かめたうえで、**実装から条件を記録する**。

## 1. 何をしたか

1. **`/feedback`をheadlessで打ってみた**（打った文：`/feedback FBOTEL_CANARY_998 telemetry probe`）。
2. **OTLPの宛先をローカルのMockへ向けた**（`DSH_TELEMETRY_OTLP_URL`・`DSH_TELEMETRY_MODE=FEEDBACK_ONLY`）。
3. **正本のSession Logを展開し、`feedback/**` eventが出たかを数えた**。
4. あわせて**同梱の`session-telemetry-otel`の実装を読み**、送出の条件を確認した。

## 2. 観測した結果

| 項目 | 値 |
| --- | --- |
| `/feedback ...` の扱い | **通常のプロンプトとしてモデルへ渡った**（モデルは`ok`と応答） |
| モデル要求 | 2件（`FBOTEL_CANARY_998`が**要求に入っている**） |
| **OTLP要求** | **0件** |
| 正本Logの`feedback/**` event | **0件** |
| 正本Logに出たevent種別 | `session`・`user/message`・`assistant/message`・`request/header`・`request/context`・`step/*`・`turn/*`・`session-log-deepseek/delivery-accepted` など**17種** |

- **`/feedback`はコマンドとして解釈されていない**。**`/`で始まる行も普通のメッセージ**になる。
- そのため**`feedback/record`が記録されず**、**OTLPへは1件も送られない**。

## 3. 実装が示す送出の条件

同梱の`dsh/node_modules/@deepseek-ai/dsh-session-telemetry-otel/lib/index.js`を読むと、送出の可否は`isFeedback()`で決まる。

| 条件 | 内容 |
| --- | --- |
| **受け付けるevent種別** | **`feedback/record`**、`feedback/message-put`、`feedback/message-delete` |
| **Session一致** | `message-put`／`message-delete`は`event.data.sessionId === session.id`が必要 |
| **順序** | `event.seq >= session.inheritedEventCount`（**forkで引き継いだ分は不可**） |
| **それ以外** | すべて対象外 |
| **mode** | `FEEDBACK_ONLY`（既定）／`DISABLED` |

feedback surface側（同梱の`dsh-command-feedback`のREADME）は、`/feedback <text>`が**コマンド**で、Webのダイアログは`sessionFeedback.record`で同じ`feedback/record`を追記するとしている。**同READMEは「headless・ACP・JSON-RPCの入口はslash commandを提供しない」と明記している。**

## 4. この記録が示すこと

- **このInstallationでは、feedback-gatedなTelemetry送出を実測できない**。理由は**feedback面が無いこと**であり、**gateの不備ではない**。
- **headlessに`/feedback`と書いても、コマンドではなくプロンプトになる**。**利用者が意図してfeedbackを書いたつもりでも、Telemetryの対象にはならない**。
- **導入済みのProfileは`desktop`と`headless`の2つだけ**で、**TUI/Webの入口をCLIから作る手立ても見つからなかった**。
- したがって**「OTLPが0件」はgateが閉じていることの確認**であって、**送出内容の測定ではない**。

## 5. この記録が示さないこと

- **feedbackを与えた場合の送出内容**（本記録では**feedbackを発生させられていない**）。実装を読む限り、**正本Session Logのevent列が対象**になるが、**その実物は見ていない**。
- **Webクライアントで`/feedback`やダイアログを使った場合**の挙動。
- `DISABLED`との差、`shutdownTimeoutMillis`・`maxRequestBytes`の効果。
- **forkで引き継いだSession**で`inheritedEventCount`がどう効くか。
- 実Providerの保存・保持・二次利用。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 6. 生データ

| ファイル | 内容 |
| --- | --- |
| `feedback-surface.json` | `probePrompt`、`env`（OTLP宛先・mode）、`sessionId`、`modelRequests`（2件のbytes・SHA-256・メッセージ数・canaryの有無・Session ID）、`otelRequestCount`・`otelFiles`、`canaryReachedModel`、`canonicalEventTypes`（17種の件数）、`feedbackEventCount`、`egressTriggerFromSource`（受け付ける種別・順序条件・mode）、`feedbackSurfaceFromSource`（slash command・dialog・利用可否）、`profilesInstalled`、`conclusion` |

生の要求bodyはコミットしていない。**合成のみ**（`FBOTEL_CANARY_998`）。

再現する場合の要点:

```sh
# OTLPの宛先をローカルへ向け、headlessで /feedback を打つ
DSH_TELEMETRY_OTLP_URL=http://127.0.0.1:8799/v1/logs \
DSH_TELEMETRY_MODE=FEEDBACK_ONLY \
dsh --profile headless --permission-mode read-only --approval-policy never \
    --json "/feedback FBOTEL_CANARY_998 telemetry probe"
```

- 判定は**OTLP側で受けた件数**と、**正本Session Logの`feedback/**` event数**の両方で行う。
- **`/feedback`がコマンドになっているか**は、**正本Logに`feedback/record`が現れるか**で見る。モデルの応答内容では判定できない。

## 7. `H6-E2`への含意（判断はしない）

Telemetryの送出は**feedback eventを契機**にしており、**feedback面が無い入口では原理的に動かない**。逆に言えば、**feedback面を持つ入口（Web）では、会話Logを伴うeventが送出対象になり得る**（同梱のREADMEは「the conversation log travels with the recorded event」と書いている）。この版のheadlessだけを見て「Telemetryは出ない」と一般化できない。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
