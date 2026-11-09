# DSH外部Harnessが読んだ内容の保存先（sink）棚卸し（2026-11-07、#2515 H6-E6）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E6`（保存面と出力拡張）。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-07（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`、`read-only`
- 状態: **合成canaryのみ**。実secretは使っていない。`Issue`・`PR`・CI log・外部Providerへの送出は**本記録の対象外（NOT RUN）**。

`H6-E6`は「CLI/CI失敗、debug trace、生成Artifact、Issue出力、Assistant Finalへ合成Raw markerを混入させ、**資料本文を公開せず、許可されたsinkにのみ結果が記録されること**を確認する」ことを求める。本記録は、**1 Session内で読んだ合成canaryが、どのローカル保存先に残るか**を列挙したものである。

## 1. 何をしたか

1. 合成canary `E6_SINK_CANARY_801` を置いたファイルを`pwsh`から読ませた（`read-only`、Mock Endpointは`127.0.0.1`）。
2. Session ID `session-68e678a4-3880-47c9-bac5-9aa0b9fb3910` を控えた。
3. 実行の前後で`%USERPROFILE%\.dsh\sessions`のスナップショットを取り、**新規／更新されたファイル**を特定した。
4. 永続化されたSession Log（`session.v4.jsonl.zstd`）は**連結zstd frame**（9 frame）だったため、frame単位で展開して内容を確認した。単一frameとして展開すると先頭frameしか得られず、canaryを見落とす。
5. `storages`・`cache`・`attachments`を走査し、canaryの有無を数えた。

## 2. 観測した結果

| sink | 実体 | bytes | canary |
| --- | --- | --- | --- |
| headlessのstdout stream | `--json`出力（capture） | 2,035 | **1** |
| 永続化Session Log（展開後） | `session.v4.jsonl.zstd` → 9 frame → 40,136 bytes / 34行 | 40,136 | **1** |
| 永続化Session Log（圧縮時） | 同ファイル | 14,373 | （展開して確認） |
| `\.dsh\storages`（84ファイル） | Session投影cache等 | — | **0** |
| `\.dsh\cache`（5ファイル） | — | — | **0** |
| `\.dsh\attachments`（73ファイル） | — | — | **0** |

- canaryは永続化Session Logの`tool/result`event（27行目、`data.message.role = tool`）に載っていた。**Tool結果がそのままディスクへ残る**。
- stdout streamにも載る（Tool resultを`--json`で出力しているため）。
- `storages`・`cache`・`attachments`には**残らなかった**。

## 3. この記録が示すこと

- `H6-E6`の「許可されたsinkにのみ結果が記録されること」に対して、**この構成では少なくとも2つのローカルsink（stdoutと永続Session Log）に読取内容が残る**。どちらも「許可されたsink」かはPolicyの判断であり、本記録は判定しない。
- 永続Session Logは**圧縮**されているが、**展開すれば内容が読める**。ディスク上で「圧縮されているから保護されている」とは扱えない。
- 永続Session Logは連結zstd frameであるため、**単一frameとして展開すると内容の大半を見落とす**。監査で「canaryが無い」と結論する前に、frame単位の展開が必要である（本計測では単一frame展開で0件、frame単位で1件だった）。

## 4. この記録が示さないこと

- `Issue`・`PR`本文・CI log・Telemetry・外部Providerへの送出。本記録はローカルのみ。
- debug trace、生成Artifact、Assistant Final（最終回答本文）への混入。
- 保持期間、削除、ローテーション、暗号化。
- session投影cacheが**別Session**の内容を持つ場合の挙動（本計測は1 Session）。
- `storages`の全ファイルを意味解析したわけではない（canary文字列の有無のみ）。
- `H6-E6`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `sinks.json` | canary、Session ID、入力ファイルのdigest、sinkごとのbytes・SHA-256・canary出現、走査したrootごとのファイル数と一致数 |

展開後のSession Log本文は**コミットしていない**（Tool結果やsystem promptを含むため）。圧縮ファイルのSHA-256とbyte数のみを記録した。

再現する場合の要点:

```sh
# 1) 合成canaryを置き、Mock Endpointから pwsh を強制して読ませる
python mock_provider_cmd_seq.py 8799 <record-dir> <commands.json>
DEEPSEEK_API_KEY=dummy DEEPSEEK_BASE_URL=http://127.0.0.1:8799 \
  dsh --profile headless --permission-mode read-only --approval-policy never --json "Investigate."
# 2) 永続Session Logは frame ごとに展開する（単一frame展開では足りない）
node decompress_frames.js <session.v4.jsonl.zstd> <out.jsonl> <canary>
```

## 6. `H6-E6`への含意（判断はしない）

`H6-E6`は「全log/backupを検査できなければunknown」と定める。本記録は、**ローカルで検査可能なsinkを列挙し、2つに内容が残ることを確認**した。`Issue`・PR・CI・外部Providerは別途の検査が必要であり、本記録だけでは`H6-E6`を満たさない。どのsinkを許可するかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
