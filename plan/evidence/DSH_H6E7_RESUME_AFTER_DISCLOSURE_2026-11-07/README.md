# DSH外部Harnessの再開：既に読んだ内容はモデル入力に再送され、Session Logは再送されない（2026-11-07、#2515 H6-E7）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E7`（再開・変更）。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-07（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、`read-only`
- 状態: **合成canaryのみ**。Grant失効後の再開、`Task Packet`の失効、外部ACK不明のreplay、`maxBytes`超過backlogは**NOT RUN**。

`H6-E7`は「Task Packet/Grant失効後のresume、既送出Session Log、予約Tool/外部ACK不明、Rule版更新」について、**新規副作用を現在Grantで検査し、過去開示を遡及取消しない**ことを求める。本記録はそのうち「既送出Session Log」と「再開時のモデル入力」を実測したものである。

## 1. 何をしたか

1. 合成Raw canary `H7_RAW_CANARY_701` を置いたファイルを、`read-only`のSession（`session-7ffdbc3c-70cc-4c44-a120-566236ac91e8`）で`pwsh`から読ませた（Turn 1）。
2. 同じSessionを`--session-id`で再開し、別のコマンドを1つ実行させた（Turn 2）。Session Logの水位はTurn 1の終了時点から進んでいる。
3. 両Turnで、Mock Endpointが受け取った要求bodyの**モデル入力**（`messages`）と**Session Log**（`dsh_session_log`）のそれぞれでcanaryを数えた。

## 2. 観測した結果

### Turn 1（Raw canaryを読む）

| 要求 | bytes | `messages`数 | canary（モデル入力） | canary（Session Log） | Session Logの水位 |
| --- | --- | --- | --- | --- | --- |
| `001`（agent初回） | 66,922 | 1 | 0 | 0 | `-1`→`17` |
| `002`（題名生成） | 40,476 | 1 | 0 | 0 | `-1`→`17`（再送） |
| `003`（Tool結果を返す） | 39,823 | 3 | **1** | **1** | `17`→`27` |

### Turn 2（同じSessionを再開）

| 要求 | bytes | `messages`数 | canary（モデル入力） | canary（Session Log） | Session Logの水位 |
| --- | --- | --- | --- | --- | --- |
| `001`（再開・agent） | 58,073 | **5** | **1** | **0** | `27`→`41` |
| `002`（題名生成） | 41,100 | **7** | **1** | **0** | `41`→`49` |

- **再開すると、過去の会話（`user`・`assistant`の交互7件）がモデル入力に再構成され、Turn 1で読んだRaw canaryが再び載る。** 新しいTurnでToolを呼ぶ前の要求にも、既に載っている。
- **Session Logは、既に受理された区間を再送しない**。Turn 2の水位は`27`から進んでおり、canaryは`dsh_session_log`に現れない。Turn 1で送出済みの内容は、再開時に送り直されない。
- 再開後も**新しいコマンドは実行できた**（`resumed=true`・cwd・Session IDを返した）。読み取り専用の作業は局所継続する。

## 3. この記録が示すこと

- `H6-E7`の「**過去開示を遡及取消しない**」は、この構成で観測どおりである。Session Logの水位（`afterSeq`/`throughSeq`）が受理済み区間を表し、再開時に送り直さない。
- 一方で、**モデル入力には過去の会話が再構成される**ため、既に外部へ出たRaw canaryは「新しい要求でも毎回出る」。Session Logの非再送は「もう出ない」ことを意味しない。
- 機密Profileで「一度読んだRawが以後の要求に載らない」ことを求める場合、**会話履歴の再構成そのものを制御する必要がある**。Session Logの`enabled`だけでは足りない。

## 4. この記録が示さないこと

- Grant失効後の再開、Task Packetの失効・更新後の権限検査。
- 外部ACK不明時のreplay（`#1174`/`#1129`の担当）。
- Rule版の更新、`session-checkpoint`からの復元、Crash後の重複送信。
- `maxBytes`を超えるbacklogの分割送出。
- 実Providerへの送出、Provider側の保存、Egress制御、別Harness。
- `H6-E7`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `observations.json` | 両TurnのローカルTool結果、要求ごとのbyte数・SHA-256・`messages`数・canaryの出現・Session Logの水位 |

生の要求bodyはコミットしていない。**canaryは合成であり、実secretではない**。

再現する場合の要点:

```sh
python mock_provider_cmd_seq.py 8799 <record-dir-1> <turn1-commands.json>
DEEPSEEK_API_KEY=dummy DEEPSEEK_BASE_URL=http://127.0.0.1:8799 \
  dsh --profile headless --permission-mode read-only --approval-policy never --json "Investigate."
# Session IDを控えてから、同じSessionを別Mockで再開する
python mock_provider_cmd_seq.py 8799 <record-dir-2> <turn2-commands.json>
DEEPSEEK_API_KEY=dummy DEEPSEEK_BASE_URL=http://127.0.0.1:8799 \
  dsh --profile headless --permission-mode read-only --approval-policy never \
      --session-id <session> --json "Continue."
```

## 6. `H6-E7`への含意（判断はしない）

「既送出Session Log」は水位で管理され、再開時に再送されない。しかし**モデル入力は会話履歴を再構成する**ため、既に読んだ内容は新しい要求にも載る。機密Profileの設計では、この2つを別々に扱う必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
