# DSHのセッションログ拡張：既定Profile・再試行・stream切断の追測（2026-11-07、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`（DSH副次送出）と[#2509](https://github.com/hat47x/tei-platform/issues/2509)。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 先行記録: [DSHセッションログ拡張の実送出計測](../DSH_SESSION_LOG_EGRESS_MOCK_2026-11-07/README.md)（本記録はその未実施項目を補う）
- 実行日: 2026-11-07（Asia/Tokyo）
- 実行環境: Windows 10.0.26100、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`、Go `go1.27.0 windows/amd64`
- 状態: **ローカルMock Endpointのみ**。実Providerへの送出・OS/Network境界での拒否・機密Profile適合は**NOT RUN**。

先行記録では`session-log-deepseek.enabled`を明示指定して測った。本記録は、その既定値・失敗時の再送・stream切断時の扱いを同じMockで確かめたものである。

## 1. 何をしたか

先行記録と同じMock Endpoint（`127.0.0.1:8799`）を、次の3種の応答で使い分けた。DSHの`deepseek-official`経路の宛先は`DEEPSEEK_BASE_URL`で向け、Credentialはダミー値を環境変数で渡した。

| 実行 | Mockの応答 | 目的 |
| --- | --- | --- |
| 既定Profile | すべて200（正常なSSE） | **overlayを当てず**、出荷既定で拡張fieldが載るか |
| 503→再試行 | 最初のagent要求だけ503、以降200 | 非2xxで受理されない場合、再試行が同じ内容を再送するか |
| stream切断 | 200だがbodyが空（`message_stop`無し） | 2xxでもSSEが途中で終わった場合の受理と再送 |

## 2. 観測した結果

| 実行 | 応答 | 要求body bytes | Tool数 | `dsh_session_log` |
| --- | --- | --- | --- | --- |
| 既定Profile・1回目 | 200 | 87,479 | 24 | 45,491 bytes、`afterSeq -1`→`throughSeq 18`、19 event |
| 既定Profile・2回目（題名生成） | 200 | 51,157 | 0 | 同じ45,491 bytes・同じ水位（**再送**） |
| 503→再試行・1回目 | **503** | 87,469 | 24 | 45,483 bytes、`-1`→`18`、19 event |
| 503→再試行・題名生成 | 200 | 51,147 | 0 | 45,483 bytes、`-1`→`18`（**同じ水位で再送**） |
| 503→再試行・再試行後 | 200 | 44,484 | 24 | **1,609 bytes、`18`→`23`、5 event**（水位が進んだ） |
| stream切断・1回目 | 200（body空） | 87,459 | 24 | 45,475 bytes、`-1`→`18`、19 event |
| stream切断・2回目（題名生成） | 200（body空） | 51,137 | 0 | 同じ45,475 bytes、同じ水位 |

### 読み取れること

- **出荷既定のProfileでも拡張fieldが載る**。overlayで`enabled: true`を与えていない実行で、`dsh_session_log`が45,491 bytes載った。DSH付属READMEの「既定`true`」と整合する。
- **非2xx（503）では受理されず、水位が進まない**。503を受けた同じ内容が、次の要求（題名生成）でも`-1`→`18`のまま**そのまま再送**された。先行記録の「2xx後に受理」という記述と整合する。ただし再送されたのは**題名生成の要求**であり、agent要求の再試行そのものを観測したわけではない。3番目の要求で水位が`18`→`23`へ進んだことから、再試行後の要求で新規分だけが送られたことが分かる。
- **200でもSSEが空なら受理されない**。stream切断では2要求とも同じ水位のままで、`message_stop`欠落によりDSHは`STREAM_CLOSED`で終了した。
- 先行記録と同じく、**同じ水位のfieldが複数要求に現れる**（題名生成は毎回、直前の水位を再送する）。

## 3. この記録が示すこと

- DSH `0.2.2-rc.1`の**出荷既定で`dsh_session_log`が有効**であり、overlayなしでもHTTP要求に載る。
- 拡張fieldの受理は**2xxかつ完全なstream**に依存する。503・空streamでは水位が進まず、後続要求が同じ内容を再送する。
- 再送は「新規Session Eventの追加」ではなく、**未受理区間の丸ごと再送**である（`afterSeq`が動かないことで確認）。

## 4. この記録が示さないこと

- 実Provider（`api.deepseek.com`）への送出、Provider側の保存・保持期間・二次利用。
- 503を受けたagent要求そのものの自動再試行（`llm-retry`の挙動）。本計測では再試行を観測していない。
- `maxBytes`（8 MiB）を超えるbacklogの分割送出。今回のfieldは最大45,491 bytesで上限に達していない。
- in-flight要求中の設定変更、Crash後の重複送信（受理Watermarkの永続化タイミング）。
- Telemetry、MCP、Subagent、`llm-pi-ai`他経路。
- Egress制御の実効性、TAI #232の機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `wire-shape-2.json` | 7要求の構造（応答status・byte数・SHA-256・拡張fieldの水位・event種別） |

生の要求bodyは先行記録と同じ理由で**コミットしていない**。各要求のSHA-256を`wire-shape-2.json`に残しており、同じMockで再実行すれば突合できる。

再現する場合の要点:

```sh
python mock_provider.py 8799 <record-dir>              # 全200
python mock_provider_503_once.py 8799 <record-dir>     # 最初のagent要求だけ503
python mock_provider_cut.py 8799 <record-dir>          # 200 + 空body

DEEPSEEK_API_KEY=dummy DEEPSEEK_BASE_URL=http://127.0.0.1:8799 \
  dsh --profile headless --permission-mode read-only --approval-policy never --json "<task>"
```

既定Profileの確認では`--patch`を付けない。503と切断の確認でも同じ（`enabled`は既定のまま）。

## 6. 機密境界への含意（判断はしない）

`#2515`の`H6-E2`は「`enabled:false`を送出禁止の証明にしない」「`false`→`true`のbacklogをMockで対照する」ことを求める。本記録は、**既定ON・非2xxで未受理・未受理区間の再送**という3点を実測で押さえた。したがって、機密Sourceを扱う際の統制は「設定で止める」だけでは足りず、**通信経路そのものの拒否**と、**未受理分が後から再送され得ることの見積り**が必要になる。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
