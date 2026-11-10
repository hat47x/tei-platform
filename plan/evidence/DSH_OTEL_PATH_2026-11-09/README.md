# Telemetry（OTel）経路は既定で載っており、明示的なfeedbackなしには送らない（2026-11-09、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-09（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルMock Endpointのみ**。実Collectorへの送出は**NOT RUN**。
- 先行記録: [台帳](../DSH_HARNESS_EVIDENCE_LEDGER_2026-11-09.md)の§4。本記録は**台帳の記述の訂正**と、Telemetry経路の初回観測を含む。

## 1. 訂正（先に書く）

台帳と[深さ上限を上げた連鎖](../DSH_SUBAGENT_DEPTH_RAISED_2026-11-09/README.md)は「**headlessプロファイルには`dsh-session-telemetry-otel`が載っておらず、`OTEL_EXPORTER_OTLP_ENDPOINT`を設定してもテレメトリの送出は観測されなかった**」と書いていた。**この記述は原因の特定を誤っている。**

- **プラグインは既定で載っている**。`@deepseek-ai/dsh-base`が`session-telemetry-otel`を登録しており、`--dump-config`に次の`!!js`既定として現れる。

```yaml
- id: session-telemetry-otel
  name: '@deepseek-ai/dsh-session-telemetry-otel'
  config:
    mode: !!js process.env.DSH_TELEMETRY_MODE || 'FEEDBACK_ONLY'
    shutdownTimeoutMillis: 3000
    maxRequestBytes: 4000000
    exporter:
      url: !!js >-
        process.env.DSH_TELEMETRY_OTLP_URL ??
        'https://dsh-otel-collector.deepseeksvc.com/v1/logs'
      compression: gzip
      timeoutMillis: 1000
```

- **この版が読むのは`DSH_TELEMETRY_OTLP_URL`と`DSH_TELEMETRY_MODE`**である。標準の`OTEL_EXPORTER_OTLP_ENDPOINT`ではない。前回の試行が無効だったのは**環境変数名が違った**ためで、**プラグインが無かったためではない**。

## 2. 何をしたか

1. `--dump-config`で登録と既定を確認した（上記）。
2. `DSH_TELEMETRY_OTLP_URL`を**ローカルMockの`/v1/logs`**へ向け、モデルも同じMockへ向けて1往復させた。
3. Mockに届いた要求を種別ごとに数えた（`/v1/logs`とモデル呼び出しを別々に記録する）。

## 3. 観測した結果

| 種別 | 件数 |
| --- | --- |
| モデル呼び出し | **2**（route probeと本呼び出し） |
| **`/v1/logs`（OTLP）** | **0** |

- **エンドポイントを差し替えても、OTLPの要求は1件も届かなかった**。合計143,308 bytesのモデル要求だけが届いた。
- 同梱物の説明は`FEEDBACK_ONLY`を「**新しい明示的なfeedbackの後にだけ**正規Logのprefixを解放する」と述べており、`DISABLED`は取得自体を止める。観測はこの説明と一致する。
- **headlessアプリにはfeedbackの面が無い**。`--help`の選択肢は`--json`・`--session-id`・`--permission-mode`・`--approval-policy`・`--add-dir`だけで、**feedbackを送るcommandもflagも無い**。

## 4. この記録が示すこと

- **Telemetry経路は既定で有効に登録されている**。したがって「載っていないから送られない」という説明は**成り立たない**。
- それでも**明示的なfeedbackが無い限り送出は起きない**（本記録の1往復では0件）。**止めているのはプラグインの不在ではなく`FEEDBACK_ONLY`のgate**である。
- **headlessではfeedbackを渡す面が無い**ため、**headlessの実行だけを見て「Telemetry経路は無い」と結論できない**。面が違えばgateが開き得る。
- エンドポイントは**環境変数で差し替え可能**（`DSH_TELEMETRY_OTLP_URL`）。**既定は`https://dsh-otel-collector.deepseeksvc.com/v1/logs`**であり、**gateが開いたときの宛先はこの既定**である。

## 5. この記録が示さないこと

- **feedbackを与えた場合に実際に何が送られるか**（本記録はfeedbackを渡す面を持たないため0件で止まっている）。
- **web・tui等のfeedback面での送出内容**、`DISABLED`にした場合の取得停止の実測。
- `maxRequestBytes`（既定4,000,000）を超える場合の分割・切り捨て。
- **MCP経路**、実Collectorへの到達、Collector側の保存・保持期間。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 6. 生データ

| ファイル | 内容 |
| --- | --- |
| `otel-path.json` | `correction`（誤っていた記述と正しい内容・確認方法）、`defaultsFromComposedConfig`（`mode`・`exporter.url`・`compression`・`timeoutMillis`・`shutdownTimeoutMillis`・`maxRequestBytes`）、`run`（環境変数・profile・permission mode・2要求のbyte数とSHA-256・Session ID・水位・`otelRequests`）、`feedbackSurface`（headlessのflag一覧と`hasFeedbackCommand`） |

生の要求bodyはコミットしていない。**合成のみ**（応答は`ok`、canaryなし）。

再現する場合の要点:

```sh
# この版が読む環境変数は DSH_TELEMETRY_OTLP_URL（標準の OTEL_EXPORTER_OTLP_ENDPOINT ではない）
DSH_TELEMETRY_OTLP_URL=http://127.0.0.1:8799/v1/logs \
  dsh --profile headless --permission-mode read-only --approval-policy never --json "Say ok."
# 既定の登録と宛先は --dump-config で確認できる
dsh --profile headless --dump-config
```

- Mockは`/v1/logs`とモデル呼び出しを**別のカウンタ**で記録する。同じ口に両方を向けると、届いた要求の種別を取り違える。
- **`FEEDBACK_ONLY`のgateを開ける面が無い**ため、headlessだけでは送出内容を測れない。

## 7. `H6-E2`への含意（判断はしない）

Telemetry経路は**既定で登録され、宛先も既定値を持つ**。**gateは`FEEDBACK_ONLY`**であり、**feedbackの面を持つアプリでのみ開く**。したがって機密Profileの判断では、**「headlessで送られない」を根拠にしない**こと、**宛先の既定値を把握しておく**ことが要る。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
