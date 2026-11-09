# 送出できなかったbacklogは、上限を上げた再実行でまとめて流れる（2026-11-09、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`／`P0-b`。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-09（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）
- 状態: **ローカルMock Endpointのみ**。実Providerへの送出、Provider側の保存は**NOT RUN**。
- 先行記録: [しきい値の特定](../DSH_SESSION_LOG_MAXBYTES_THRESHOLD_2026-11-09/README.md)。本記録はそこで残った「**同じSessionで上限を上げると停滞分は流れるか**」を確かめる。

## 1. 何をしたか

1. `maxBytes=306`（最初のeventが入らない上限）で4 Tool呼出しを実行し、**Session IDを控えた**。6要求すべてでfieldが載らないことを確認した。
2. **同じSession IDを指定して**、`maxBytes=4096`で再実行した。
3. 両方の捕捉について、bodyから切り出した生byte数でfieldの有無・水位・event数を比べた。

## 2. 観測した結果

### 1回目（`maxBytes=306`）

| 要求 | field |
| --- | --- |
| 001〜006（すべて） | **無し** |

- 6要求すべてでfieldが載らず、水位も進まなかった。

### 2回目（同じSession、`maxBytes=4096`）

| 要求 | field 生byte数 | 水位 | event数 |
| --- | --- | --- | --- |
| 001（agent） | **1,273** | **`-1`→`9`** | **10** |
| 002（題名生成） | 3,498 | `9`→`11` | 2 |
| 003（agent） | 1,745 | `11`→`12` | 1 |
| 004〜005 | 無し | — | — |

- **停滞していた分がまとめて流れた**。最初の要求が`afterSeq -1`から`9`までを運び、**10 event**を載せた。1回目が一度も送らなかった区間である。
- 続く2要求で`9`→`11`→`12`と進み、**合計13 event**が載った。
- **上限を上げると、その場で溜まっていた分が1要求にまとまって出る**。分割送出（[先行記録](../DSH_SESSION_LOG_SPLIT_AND_STALL_2026-11-08/README.md)）は「上限に収まる範囲」で切れるため、上限が大きいほど**1要求あたりの開示量が増える**。

## 3. この記録が示すこと

- `maxBytes`未満で送れなかったeventは**失われない**。後から上限を上げれば、**その区間がまとめて送出される**。
- したがって「上限を小さくして止めた」は**恒久的な抑止ではない**。設定を戻す（または別の実行が既定値で走る）と、**溜まっていた分が1要求で出る**。
- 止めた期間が長いほど、**再開時に1要求へ載る量が大きくなる**。機密Sourceを扱う場合、`maxBytes`は「平均の送出量」ではなく**再開時の最大burst**として見積もる必要がある。

## 4. この記録が示さないこと

- **上限を上げた瞬間に必ず流れるか**（本記録は新しいTurnを1回実行しており、`maxBytes`の変更だけで自動的に流れるとは限らない）。
- 停滞中に**どのeventが詰まっているか**を外部から知る手段（警告文・状態表示）。本記録でも警告は観測していない。
- 実Providerへの送出、Provider側の保存・保持期間、`enabled`のin-flight変更、Crash後の重複送信。
- Telemetry、MCP、Subagent、`llm-pi-ai`他経路。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `stall-grow.json` | 2実行の要求ごとのbody byte数・SHA-256・field生byte数・水位・event数、`anyFieldSent`・`firstAfterSeq`・`maxWatermark`・`eventsCarried`、および`stall`（1回目の条件とSession ID） |

生の要求bodyはコミットしていない。**合成データのみ**。

再現する場合の要点:

```sh
# 1回目: 上限を最初のeventより小さくする
dsh --profile headless --patch <overlay maxBytes=306> ... --json "Run the steps."
# 2回目: 同じ Session ID を指定し、上限を上げて再実行する
dsh --profile headless --patch <overlay maxBytes=4096> ... --session-id <id> --json "Run the steps."
```

上限判定は**元bodyのbyte span**で行う。Session IDは1回目の実行ログ（`{"type":"session"}`）から取る。

## 6. `H6-E2`への含意（判断はしない）

`maxBytes`は**平均の送出量を抑える設定**としては働くが、**恒久的な抑止ではない**。止めていた分は上限を戻した再実行で**1要求にまとまって出る**ため、機密Sourceを扱う場合の見積りは「1要求あたりの上限」ではなく、**止めていた期間に溜まった分が再開時に何byteになるか**で行う必要がある。この点をどう統制へ反映するかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
