# 単一eventが`maxBytes`を超えると拡張が丸ごと消え、`delivery-accepted`も増えない（2026-11-09、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-09（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルMock Endpointのみ**。実Providerへの送出は**NOT RUN**。
- 先行記録: [停滞の可視性](../DSH_SESSION_LOG_STALL_VISIBILITY_2026-11-09/README.md)・[停滞の検知](../DSH_SESSION_LOG_STALL_DETECTION_2026-11-09/README.md)。本記録は台帳§4の未確定「**単一eventが上限を超えた場合の`delivery-accepted`の振る舞い**」を埋める。

## 1. 何をしたか

停滞（`maxBytes`が小さく、送るものが溜まる）では**拡張は付いたまま水位が止まる**ことを既に測っていた。本記録は**単一のeventが上限を超える**場合を分けて測る。

1. `maxBytes`を**4096**にした。
2. **約8,000 bytesの標準出力**を出す`pwsh`を1回強制し、その**後に小さな呼び出しを5回**強制した（大きい結果がbacklogに残ったまま次の要求が出るようにするため）。
3. **同じ`maxBytes`で、大きい結果を出さない対照**も走らせた。

## 2. 観測した結果

### 大きいeventが入ると拡張が消える

| 要求 | 拡張 | event数 | 水位 |
| --- | --- | --- | --- |
| 001 | **あり** | 10 | `-1`→`9` |
| 002 | **あり** | 10 | `-1`→`9` |
| 003 | **あり** | 2 | `9`→`11` |
| 004 | **あり** | 1 | `11`→`12` |
| **005〜011** | **なし** | — | — |

- **005以降は`dsh_session_log`というfield自体が無い**。空の配列でも、切り詰めたeventでもない。
- 水位は**`12`で凍った**まま、作業は進んだ（永続Logは**96 event・maxSeq 94**）。

### `delivery-accepted`は増えない

- 永続Logの`session-log-deepseek/delivery-accepted`は**4件**で止まった。**拡張が消えた後の要求では1件も増えていない**。
- つまり「拒否された」記録も「成功した」記録も残らず、**成功記録が途切れる**形でしか観測できない。

### 何が上限を超えていたか

永続Logの大きいevent:

| seq | type | bytes |
| --- | --- | --- |
| 13 | `user/message` | **5,792** |
| 14 | `request/header` | **20,639** |
| 25 | `tool/result` | **9,407** |
| 48 | `tool/result` | **9,407** |

- 水位が`12`で止まった直後の**seq 13がすでに5,792 bytes**で、上限4,096を超えている。
- **送れない最小単位が1つあると、それ以降まとめて送れなくなる**。分割して送る形にはなっていない。

### 対照（同じ上限・小さいeventのみ）

| 条件 | 要求数 | 拡張あり | 水位 |
| --- | --- | --- | --- |
| `maxBytes 4096`・**小さい結果** | 3 | **3** | 通常に進む（`-1`→`9`→`11`） |
| `maxBytes 4096`・**大きい結果** | 11 | **4** | `12`で凍る |

- **同じ上限でも、大きいeventが入るかどうかで結果が変わる**。上限値だけでは説明できない。

### 参考: 256は最小単位より小さい

- `maxBytes 256`では**小さい結果でも拡張が付かなかった**（3要求すべてで不在）。この版では**256が送れる最小単位を下回る**ためで、**「大きいevent」の問題とは別**である。

## 3. この記録が示すこと

- **単一eventが`maxBytes`を超えると、拡張は丸ごと送られなくなる**。**切り詰めでも部分送信でもない**。
- **`delivery-accepted`は増えない**。拒否の記録も残らないため、**成功記録の不在**としてしか分からない。
- **作業は止まらない**（永続Logは96 eventまで進む）。**送出だけが止まる**。
- **上限値だけを見て「送れる」と判断できない**。**1件の大きさ**が効く。

## 4. この記録が示さないこと

- **Telemetry（OTel）・UI・別flag**での検知。3面に警告が出ないことは[停滞の可視性](../DSH_SESSION_LOG_STALL_VISIBILITY_2026-11-09/README.md)で確認したが、本記録の「拡張が消える」形が他の面に出るかは未確認。
- **上限を戻した再実行で回復するか**。停滞では回復した（[停滞後の再開](../DSH_SESSION_LOG_STALL_DETECTION_2026-11-09/README.md)）が、本記録の形では未確認。
- **`maxRequestBytes`**（Telemetry側、既定4,000,000）との関係。
- **他版で同じ挙動になるか**（この版の観測であって契約ではない）。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `oversize-event.json` | overlay、`capped`（11要求のbyte数・SHA-256・Tool数・メッセージ数・拡張の有無・event数・水位・最大event bytes、永続Logのevent数・maxSeq・`delivery-accepted`件数・大きいeventの一覧）、`control`（同じ上限で小さい結果のみの3要求）、`contrast256`（256では小さい結果でも拡張が付かなかったこと） |

生の要求bodyはコミットしていない。**合成のみ**（`BIGEVENT_START`/`x`の繰り返し/`BIGEVENT_END`・`SMALL_EVENT_TAIL`）。

再現する場合の要点:

```sh
# maxBytes を 4096 にする overlay（- id: session-log-deepseek / config: { enabled: true, maxBytes: 4096 }）
# 約8,000 bytes を出す pwsh を1回、その後に小さな呼び出しを5回強制する
python mock_provider_bigevent2.py 8799 <record-dir> 8000
dsh --profile headless --patch <overlay> --permission-mode read-only \
    --approval-policy never --json "Big then several small."
```

- **大きい結果の後に必ずもう1回要求を出す**。1回で終わると、大きいeventがbacklogに入る前にrunが終わり、現象が見えない（最初の試行はこれで空振りした）。
- 判定は**`dsh_session_log` fieldの有無**で行う。空配列かどうかではない。
- `delivery-accepted`は**`data.message.isError`ではなく**、永続Logのevent種別として数える。

## 6. `H6-E2`への含意（判断はしない）

`maxBytes`は**「合計の上限」としてだけでなく「1件の上限」としても効く**。**1件が超えると全部止まり、しかも成功記録が増えないだけなので、出力からは「送っていない」と「設定が効いている」を区別できない**。機密Profileで送出を抑える設計を扱うなら、**合計と単一eventの両方**を確かめ、**止まったことを観測できる面**を別に用意する必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
