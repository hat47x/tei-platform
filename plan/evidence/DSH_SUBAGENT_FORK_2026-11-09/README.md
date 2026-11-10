# `subagent_fork` は親の完了Turnを子へ運ぶ（2026-11-09、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-09（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）
- 状態: **ローカルMock Endpointのみ**。実Providerへの送出は**NOT RUN**。
- 先行記録: [Subagent経路](../DSH_SUBAGENT_EGRESS_2026-11-09/README.md)。本記録は、そこで未確定として残した**`subagent_fork`**を測る。

台帳の未確定は「`subagent_fork`（会話を継承する子。親の会話が子へ載る可能性があり別に測る必要がある）」。本記録はその1点を確かめる。

## 1. 何をしたか

**Turnを分けて2回実行した**。継承の対象は「完了したTurn」であり、実行中のTurnは含まれないと説明されているためである。

1. **Turn 1**（`pwsh`で`PARENT_CANARY_941`を出力させる）を実行し、完走させた。Session IDを控えた。
2. **Turn 2**（同じSessionを再開し、`subagent_fork`を1回強制）を実行した。`prompt`に`FORK_PROMPT_CANARY_942`を入れた。
3. 親・子の要求と永続Logを、**canaryの出現位置**で比べた。

## 2. 観測した結果

### 要求

| 実行 | 要求 | Tool数 | メッセージ数 | 水位 | 親canary | fork canary | Session |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Turn 1 | 001 | 24 | 1 | `-1`→`17` | なし | なし | 親 |
| Turn 1 | 002（題名） | 0 | 1 | `-1`→`17` | なし | なし | 親 |
| Turn 1 | 003 | 24 | 3 | `17`→`27` | **あり** | なし | 親 |
| Turn 2 | 001 | 24 | 5 | `27`→`41` | **あり** | なし | 親 |
| Turn 2 | 002 | 24 | 7 | `41`→`48` | **あり** | **あり** | 親 |
| Turn 2 | **003（子）** | 24 | **5** | **`-1`→`42`** | **あり** | **あり** | **子** |
| Turn 2 | 004 | 24 | 9 | `48`→`57` | **あり** | **あり** | 親 |

- **子の要求は`-1`から始まる別Session**（`…aa0046686f15`）だが、**親のTurn 1で出たcanary `PARENT_CANARY_941` を本文に含む**。
- **子のメッセージ数は5**で、ロールは`user`→`assistant`→`user`→`assistant`→`user`。先頭の`user`は**Turn 1の最初の指示**であり、子は**親の完了Turnの会話を引き継いでいる**。
- `subagent`（継承しない子）では子のメッセージ数が1だった（[先行記録](../DSH_SUBAGENT_EGRESS_2026-11-09/README.md)）のと対照的である。

### 永続Log

| | 親 | 子 |
| --- | --- | --- |
| 圧縮 | 本記録では展開のみ比較 | 同左 |
| 展開bytes / 行 | 69,114 / 63行 | **67,489 / 48行** |
| seq範囲 | `0`..`61` | `0`..`46`（**自分のseqで0から**） |
| 親canaryの出現 | 7回 | **7回** |
| fork canaryの出現 | 4回 | 2回 |
| `subagent/descriptor` | 無し | **あり** |

- **子のLogは自分のseqで0から始まる**（親のseqを引き継がない）が、**親canaryが7回現れる**。子のLogに親の会話内容が入っている。
- 子のLogは**親と同じディレクトリ構成の別ディレクトリ**に保存される。

## 3. この記録が示すこと

- `subagent_fork`は**別Session**を作るが、その内容は**親の完了Turnを引き継ぐ**。したがって**親で読んだ内容（Toolの出力を含む）が子のモデル入力にも載る**。
- `subagent`（継承しない）と`subagent_fork`（継承する）は、**送出される文脈の量が違う**。同じ「Subagent起動」として扱うと、fork側の送出を過小評価する。
- 子の永続Logにも親の内容が入るため、**子のLogを見ても親の開示内容が分かる**（逆に、親のLogだけでは子の追加分を数えられない）。

## 4. この記録が示さないこと

- **実行中のTurn**（未完了）が子に載るか。本記録は完了Turnだけを対象にした。
- 子が**さらに`subagent_fork`**を起動した場合の連鎖。
- Telemetry（OTel）・MCP経路。本記録はSubagent forkの1経路だけ。
- 実Providerへの送出、Provider側の保存・保持期間、Crash後の重複送信。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `fork-inheritance.json` | Turn 1（3要求）とTurn 2（4要求）のbyte数・SHA-256・Tool数・メッセージ数・ロール・Session ID・水位・canary出現 |
| `fork-logs.json` | 親と子の永続Logの圧縮/展開bytes・行数・seq範囲・event種別内訳・親canary/fork canaryの出現回数・`subagent/descriptor`の有無 |

生の要求bodyはコミットしていない。**合成canaryのみ**。

再現する場合の要点:

```sh
# Turn 1: 親にcanaryを出させる（pwshモード）
python mock_provider_fork2.py 8799 <dir1> pwsh
dsh ... --json "Emit the canary."
# Turn 2: 同じ Session ID で再開し、forkモードで起動する
python mock_provider_fork2.py 8799 <dir2> fork
dsh ... --session-id <id> --json "Fork the assistant."
```

**Turnを分ける必要がある**。1 Turn内で`pwsh`の直後に`subagent_fork`を要求すると、そのTurnは未完了のため継承されず、`prompt`だけの子になる（実際にその形を先に試して確認した）。

## 6. `H6-E2`への含意（判断はしない）

「Subagentを起動したときの送出量」は、`subagent`（文脈を継承しない）と`subagent_fork`（親の完了Turnを継承する）で**別に数える**必要がある。forkでは親が読んだ内容が子のモデル入力にも載るため、**親のLogだけを見て「子は独立だから影響なし」と読むのは誤り**である。どの経路を機密Profileの対象にするかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
