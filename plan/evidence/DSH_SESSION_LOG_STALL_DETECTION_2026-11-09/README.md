# 停滞は「受理eventの不在」としてのみ観測できる（2026-11-09、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`／`P0-b`。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-09（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）
- 状態: **ローカルMock Endpointのみ**。実Providerへの送出は**NOT RUN**。
- 先行記録: [停滞の可視性](../DSH_SESSION_LOG_STALL_VISIBILITY_2026-11-09/README.md)。本記録は、そこで残した「**非停滞との同条件比較**」を埋め、**検知できる手掛かりがあるか**を確かめる。

## 1. 何をしたか

同じ4 Tool呼出しのSessionを、`maxBytes`だけ変えて2回実行した。どちらも**stderrをstdoutと混ぜずに別ファイルへ**採取した（先行の非停滞実行は`2>&1`で混ぜていたため、同条件で比較できなかった）。

- **停滞**: `maxBytes=306`（最初のeventが入らない）
- **非停滞**: `maxBytes=4096`（最初のeventが入る）

3面（stdout・stderr・永続Session Log）を同じ手順で解析し、**event種別の内訳**まで比べた。

## 2. 観測した結果

| 面 | 停滞（306） | 非停滞（4096） |
| --- | --- | --- |
| stdout | 6,176 bytes、該当語0件 | 6,176 bytes、該当語0件 |
| stderr | **0 bytes** | **0 bytes** |
| 永続Log | 展開51,196 bytes・**51行** | 展開51,950 bytes・**55行** |
| **`session-log-deepseek/delivery-accepted`** | **0件** | **4件**（`throughSeq` = `9`,`9`,`11`,`12`） |
| 警告種別のevent | なし | なし |
| field送出（捕捉側） | **なし**（6要求すべて） | **あり**（先頭は`-1`→`9`、10 event） |

- **両実行とも、stdoutもstderrも警告を出さない**。同条件で比べても差は無い。
- 差は**永続Logの`delivery-accepted` eventの有無**にだけ現れた。非停滞では4件、停滞では**0件**。
- つまり停滞は「**受理されたことを示すeventが無い**」という**不在**として観測できる。**警告は無い**。

## 3. この記録が示すこと

- 停滞の検知は、**`session-log-deepseek/delivery-accepted`の出現を数える**ことで可能である（この版・この構成で）。
- ただしそれは**「警告が出る」ではなく「成功の記録が無い」**という形である。**沈黙と区別できない**ため、利用者が「送出していない」と気づくには**期待されるeventの有無を照合する仕組み**が要る。
- 先行記録の「3面に警告が出ない」は、**非停滞でも同じく出ない**ことが確認できた。したがって「警告の不在」は停滞の証拠にならず、**対照比較があって初めて意味を持つ**。

## 4. この記録が示さないこと

- **Telemetry・UI・別flag**での警告の有無。本記録は3面だけを見た。
- 単一eventが`maxBytes`を超えた場合に`delivery-accepted`がどう振る舞うか。
- `delivery-accepted`が**他のHarness版・他のSession形式**でも同じ名前・同じ意味で出るか（この版の観測であって契約ではない）。
- 実Providerへの送出、Provider側の保存・保持期間、Crash後の重複送信。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `surfaces-compare.json` | 2実行それぞれの3面のbyte数・SHA-256・event種別内訳・該当語の出現数・**`delivery-accepted`の件数と水位**・`warningEventType` |

生の要求bodyはコミットしていない。**合成データのみ**。

再現する場合の要点:

```sh
# stderrをstdoutと混ぜない
dsh --profile headless --patch <overlay maxBytes=306> ... --json "Run the steps." > out.jsonl 2> err.txt
# 永続Logはframeごとに展開してから event種別を数える
```

**`2>&1`で混ぜると面ごとの可否が分からなくなる**（先行記録の比較ができなかった原因）。

## 6. `H6-E2`への含意（判断はしない）

停滞を検知するには、**成功の記録（`delivery-accepted`）の有無を照合する**必要がある。警告は無いので、「警告が出なかったから送れている」とは読めない。この照合を誰が・どこで行うか（管理側の監視、Harnessの状態表示、捕捉側の欠落検知）は#2515とTAI #232のOwner判断であり、本記録は判定しない。
