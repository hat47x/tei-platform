# Session Logのbacklog分割送出と停滞、および上限判定の単位（2026-11-08、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`／`P0-b`。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-08（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルMock Endpointのみ**。実Providerへの送出、Provider側の保存、OS/Network境界の拒否は**NOT RUN**。
- 先行記録: [maxBytes上限の初回計測](../DSH_SESSION_LOG_MAXBYTES_2026-11-08/README.md)。本記録はその**未確定点を埋め、1つの読み違いを訂正する**。

## 1. 何をしたか

`session-log-deepseek.maxBytes`を**1024**と**256**に設定し、いずれも**複数Turn（4回のTool呼出し）**を1 Sessionで実行した。宛先は出荷既定の`deepseek-official`経路を`DEEPSEEK_BASE_URL`でローカルMockへ向けた。Credentialはダミー。

計測方法も変えた。先行記録では復号したJSONを`json.dumps`し直してbyte数を数えていた。これは**送ったbyte列と別物**になり得るので、本記録では**元bodyから`dsh_session_log`の値の範囲を切り出し、その生byte数**を数える。

## 2. 観測した結果

### `maxBytes=1024`（4 Turn）

| 要求 | field 生byte数 | 上限超過 | 水位 | event数 |
| --- | --- | --- | --- | --- |
| 001（agent） | **975** | いいえ | `-1`→`6` | 7 |
| 002（題名生成） | 975 | いいえ | `-1`→`6`（再送） | 7 |
| 003（agent・次のTurn） | **490** | いいえ | `6`→`9` | 3 |
| 004（agent・さらに次） | 無し | — | — | — |
| 005・006 | 無し | — | — | — |

- **backlogが後続の受理済み要求で分割送出された**。001で`6`まで運び、003で`afterSeq 6`から`9`まで運んだ。先行記録で未確定だった点がここで確認できた。
- 003のfieldは490 bytesで、001の975 bytesより小さい。**新しい区間だけを運ぶため、fieldは毎回同じ大きさにならない**。
- 004以降はfieldが載らない。新しいeventが無い、または次の塊が上限に入らない状態と読めるが、**どちらかは本記録だけでは確定しない**。

### `maxBytes=256`（4 Turn）

| 要求 | field 生byte数 | 水位 |
| --- | --- | --- |
| 001〜006（すべて） | **無し** | — |

- **上限256では、初回から一度もfieldが載らなかった**。ヘッダとenvelopeだけで上限を超えるため、**event本文が0でもfieldを作れない**。
- この間もTool呼出しは6要求すべて実行され、Sessionは`completed`で終わった。**Session Logの送出だけが止まり、作業は止まらない**。

### 先行記録の読み違いを訂正する

先行記録は「`maxBytes=512`でfieldが**518 bytes**載り、上限を超えた」と書いた。しかし518は**復号して`json.dumps`し直した値**であり、実際に送られたbyte列は**474 bytes**だった（上限512以下）。

| 見方 | 値 | 上限512との関係 |
| --- | --- | --- |
| 生のbyte span | **474** | **以下**（超過ではない） |
| `json.dumps`で作り直した値 | 518 | 超過（**この比較は誤り**） |

同じcap（2048）でも、生1263 bytesに対し作り直すと1393 bytesで、**差は一貫して生の方が小さい**（キーの順序・空白・escapeが変わるため）。**上限判定は必ず生byteで行う**。先行記録の「event本文の合計が上限未満でもfield全体は上限を超え得る」という一般論は成り立つが、**512の実測をその例として引くのは取り消す**。

## 3. この記録が示すこと

- **backlogは後続の受理済み要求へ分割送出される**。`afterSeq`/`throughSeq`の水位がそのまま分割の境界になる。
- **1回に運ぶ量は上限で決まり、fieldの大きさは毎回変わる**（1024の例で975→490）。
- **上限がヘッダ＋envelopeより小さいと、fieldは一度も作られない**（256の例）。Session Logの送出だけが止まり、Session自体は完了する。
- **上限判定の単位は生byte**。復号して作り直した値で判定すると、超過の有無を誤る（512の例）。

## 4. この記録が示さないこと

- 004以降でfieldが載らなかった理由（新eventが無いのか、次の塊が上限に入らないのか）。
- 単一eventが`maxBytes`を超えた場合の**警告文と停滞**（プラグインのREADMEが述べる挙動）。本記録では警告を観測していない。
- 実Providerへの送出、Provider側の保存・保持期間、`enabled`のin-flight変更、Crash後の重複送信。
- Telemetry、MCP、Subagent、`llm-pi-ai`他経路。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `split.json` | `unitCorrection`（先行記録の訂正値）、2実行の要求ごとのbody byte数・SHA-256・field生byte数・上限超過判定・水位・event数 |

生の要求bodyはコミットしていない。

再現する場合の要点:

```sh
python mock_provider_cmd_seq.py 8799 <record-dir> <turn-commands.json>
DEEPSEEK_API_KEY=dummy DEEPSEEK_BASE_URL=http://127.0.0.1:8799 \
  dsh --profile headless --patch <overlay with maxBytes> \
      --permission-mode read-only --approval-policy never --json "Run the steps."
```

`<overlay>`は`session-log-deepseek`の`enabled: true`と`maxBytes`だけを指定する。**provider経路を差し替えるoverlayと併用しない**（`DEEPSEEK_BASE_URL`は`deepseek-official`経路だけが読む）。上限判定は**元bodyのbyte span**で行い、復号後の再直列化で代用しない。

## 6. `H6-E2`への含意（判断はしない）

`maxBytes`は**1要求の送出量の上限**であり、総開示量の上限ではない。backlogは後続要求へ分割されるため、機密Sourceの見積りは「1要求あたり」ではなく、**セッション全体で何要求に分割されるか**で数える必要がある。加えて、**上限が小さすぎると送出が丸ごと止まる**（止まるのはLogだけで作業は続く）ため、「設定を厳しくすれば安全」とは読めない。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
