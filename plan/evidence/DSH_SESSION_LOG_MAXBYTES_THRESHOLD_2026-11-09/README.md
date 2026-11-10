# `maxBytes` の停止しきい値は「最初のeventのfield直列化サイズ」（2026-11-09、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`／`P0-b`。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-09（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）
- 状態: **ローカルMock Endpointのみ**。実Providerへの送出、Provider側の保存は**NOT RUN**。
- 先行記録: [maxBytes上限](../DSH_SESSION_LOG_MAXBYTES_2026-11-08/README.md)・[backlog分割と停滞](../DSH_SESSION_LOG_SPLIT_AND_STALL_2026-11-08/README.md)。本記録はその「上限過小時にfieldが一度も作られない」を**1byte単位で特定する**。

## 1. 何をしたか

同じ4 Tool呼出しのSessionを、`maxBytes`だけを変えて繰り返した。**320 → 307 → 306 → 300 → 288** と下げ、各回で6要求（agent 5＋題名生成1）を採取した。上限判定は**元bodyから切り出した生byte数**で行う（復号して作り直した値は使わない。[先行記録の訂正](../DSH_SESSION_LOG_SPLIT_AND_STALL_2026-11-08/README.md)を参照）。

## 2. 観測した結果

| `maxBytes` | fieldが載ったか | 最初のfield生byte数 | 到達した最大水位 |
| --- | --- | --- | --- |
| 320 | **はい** | 307 | `4` |
| 307 | **はい** | 307 | `4` |
| 306 | **いいえ** | — | — |
| 300 | **いいえ** | — | — |
| 288 | **いいえ** | — | — |

- **307 と 306 の1byte差で結果が反転した**。307ではfieldが載り、306では**6要求すべてで載らなかった**。
- 載るときの最初のfieldは**307 bytes**で、上限とちょうど同じ。**上限＝直列化後の実サイズなら送られ、1byte下回ると送られない**。
- 載る場合、水位は`0`→`1`→`2`→`3`→`4`と**1要求あたり1eventずつ**進んだ（[分割送出の記録](../DSH_SESSION_LOG_SPLIT_AND_STALL_2026-11-08/README.md)で観測した「上限が小さいほど1要求で運べるevent数が減る」の極端な形）。
- 載らない場合も**Sessionは`completed`で終わった**。Logの送出だけが止まり、作業は続く。

## 3. この記録が示すこと

- `maxBytes`の下限は「**最初の未送信eventを包んだfieldの直列化サイズ**」。その値以上なら送られ、1byte下回ると**そのSessionでは一度も送られない**。
- 「上限を小さくすれば開示量が減る」は、**ある点までは成り立つが、超えると開示量が0になる**。0になること自体は安全側に見えるが、**送出が止まったことを検知する手段**（警告・失敗）は本記録では観測していない。
- 上限に近い領域では、**1要求で1eventしか運べない**。同じ内容を運ぶのに要求数がevent数だけ必要になる。

## 4. この記録が示さないこと

- **停滞したeventが後から送られる条件**（上限を上げた再実行での確認）。本記録では各回を新しいSessionとして実行しており、同じSessionで上限を変えていない。
- プラグインが述べる**警告文の有無**。6要求すべてで非JSON行は0件、実行ログにも`maxBytes`関連の警告は現れなかった。
- 実Providerへの送出、Provider側の保存・保持期間、`enabled`のin-flight変更、Crash後の重複送信。
- Telemetry、MCP、Subagent、`llm-pi-ai`他経路。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `threshold.json` | 5実行 × 6要求のbody byte数・SHA-256・field生byte数・上限超過判定・水位・event数、および`anyFieldSent`・`firstFieldRawBytes`・`maxWatermark` |

生の要求bodyはコミットしていない。**合成データのみ**。

再現する場合の要点:

```sh
python mock_provider_cmd_seq.py 8799 <record-dir> <turn-commands.json>
DEEPSEEK_API_KEY=dummy DEEPSEEK_BASE_URL=http://127.0.0.1:8799 \
  dsh --profile headless --patch <overlay with maxBytes> \
      --permission-mode read-only --approval-policy never --json "Run the steps."
```

`<overlay>`は`session-log-deepseek`の`enabled: true`と`maxBytes`だけを指定する。**provider経路を差し替えるoverlayと併用しない**。上限判定は**元bodyのbyte span**で行う。

## 6. `H6-E2`への含意（判断はしない）

`maxBytes`を厳しくすると開示量は減るが、**ある1byteを境に「減る」から「送出されない」へ変わる**。これは統制として使える一方、**止まったことが利用者に見えない**なら「送出していない」と「設定が効いている」を取り違える余地がある。検知手段の要否と、機密Profileで許す`maxBytes`の下限は#2515とTAI #232のOwner判断であり、本記録は判定しない。
