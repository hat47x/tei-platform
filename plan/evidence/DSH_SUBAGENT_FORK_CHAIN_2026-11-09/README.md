# `subagent_fork` の連鎖も深さ上限で止まり、同一Turn内では文脈が積み上がらない（2026-11-09、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-09（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルMock Endpointのみ**。実Providerへの送出は**NOT RUN**。
- 先行記録: [Subagent fork](../DSH_SUBAGENT_FORK_2026-11-09/README.md)・[Subagentの連鎖](../DSH_SUBAGENT_CHAIN_2026-11-09/README.md)・[深さ上限を上げた連鎖](../DSH_SUBAGENT_DEPTH_RAISED_2026-11-09/README.md)。本記録は台帳の未確定「**`subagent_fork`の連鎖**」を埋める。

## 1. 何をしたか

`subagent`（継承しない子）の連鎖は測ってある。本記録は**`subagent_fork`（親の完了Turnを継承する子）**で同じことを行う。

- `--patch`で`subagent`の`maxDepth`を**3**にした（[深さ上限を上げた連鎖](../DSH_SUBAGENT_DEPTH_RAISED_2026-11-09/README.md)と同じoverlay）。
- **Sessionごとに1回**`subagent_fork`を強制するMockで連鎖させた。
- 各Sessionの要求の**メッセージ数**を測り、**文脈が段ごとに積み上がるか**を見た。

## 2. 観測した結果

### Sessionは4つで止まり、forkでも深さ上限が効いた

| Session | 深さ | `provider` | 要求数 | `catalog` | 起動の結果 |
| --- | --- | --- | --- | --- | --- |
| `…66f060`（親） | 0 | — | 3 | **1** | `started subagent …`（成功） |
| `…15067e` | 1 | **fork** | 3 | **1** | `started subagent …`（成功） |
| `…afb918` | 2 | **fork** | 3 | **1** | `started subagent …`（成功） |
| `…5adafb` | 3 | **fork** | 2 | 0 | **`Error: subagent depth 4 exceeds maxDepth 3`**（`isError: true`） |

- **forkでも深さ上限は同じように効く**。深さ3で止まり、深さ4は`isError: true`で拒否される。
- 子の`subagent/descriptor`は`provider: fork`。**spawnの連鎖と同じ形**で止まる。

### 文脈は積み上がらなかった

各要求のメッセージ数:

| 要求 | Session | メッセージ数 |
| --- | --- | --- |
| 001（親・初回） | `…66f060` | 1 |
| 003（親・fork後） | `…66f060` | 3 |
| **004（深さ1・初回）** | `…15067e` | **1** |
| **006（深さ2・初回）** | `…afb918` | **1** |
| **008（深さ3・初回）** | `…5adafb` | **1** |

- **forkした子の初回要求はどれもメッセージ数1**。`prompt`だけである。
- [Subagent fork](../DSH_SUBAGENT_FORK_2026-11-09/README.md)では、**完了Turnを持つ親からforkすると子はメッセージ数5**で親の会話を引き継いだ。今回は**同一Turnの途中でforkしている**ため、**そのTurnは未完了で継承されない**。
- 結果として、**深さが増えても文脈は積み上がらない**。各段は`prompt`だけから始まる。

### 送出量はspawnの連鎖とほぼ同じ

| 連鎖 | Session数 | 要求数 | 合計bytes |
| --- | --- | --- | --- |
| `subagent`（spawn） | 4 | 12 | 765,677 |
| **`subagent_fork`** | **4** | **12** | **765,350** |

- **差は327 bytes（0.04%）**。文脈を継承しないため、**同じTurn内で連鎖させる限りforkでも送出量は増えない**。

## 3. この記録が示すこと

- **`subagent_fork`の連鎖も`maxDepth`で止まる**。spawnと同じ上限・同じ拒否の形。
- **forkの継承は「完了Turn」に限られる**。同一Turn内の連鎖では継承されず、**文脈は積み上がらない**。
- したがって「forkは親の会話を運ぶから危険」は**条件付き**である。**完了Turnを挟んでforkした場合**（[Subagent fork](../DSH_SUBAGENT_FORK_2026-11-09/README.md)）に継承が起き、**同一Turn内の連鎖**では起きない。

## 4. この記録が示さないこと

- **完了Turnを挟んだforkの連鎖**（Turnごとにforkして深くする形）。本記録は同一Turn内だけ。
- `maxDepth`を`4`以上にした場合、`maxActiveSubagents`（既定`8`）との相互作用。
- **Telemetry（OTel）・MCP経路**。headlessプロファイルには`dsh-session-telemetry-otel`が載っていない（[台帳](../DSH_HARNESS_EVIDENCE_LEDGER_2026-11-09.md)§4）。
- 実Providerへの送出、Provider側の保存・保持期間。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `fork-chain.json` | overlayの内容、12要求のbyte数・SHA-256・Tool数・メッセージ数・ロール・Session ID・水位・`exceeds maxDepth`の有無・canary出現、4 Sessionの永続Log要約（行数・event種別・`descriptor`・`catalog`件数・`tool/result`の`isError`と本文）、`chainDepthObserved`・`distinctSessions`・`totalRequestBytes` |

生の要求bodyはコミットしていない。**合成canaryのみ**（`FORK_CHAIN_CANARY_96N`）。

再現する場合の要点:

```sh
# maxDepth を上げる overlay（subagent プラグイン）
# - id: subagent / config: { maxDepth: 3 }
python mock_provider_fork_chain.py 8799 <record-dir>
dsh --profile headless --patch <overlay> --permission-mode read-only \
    --approval-policy never --json "Fork a chain."
```

- Mockは**`tools`を伴う最初の要求**で1 Sessionにつき1回だけ`subagent_fork`を強制する。
- **文脈の継承を見るときはメッセージ数**を読む。`prompt`だけなら1、完了Turnを継承していれば5以上（[Subagent fork](../DSH_SUBAGENT_FORK_2026-11-09/README.md)）。

## 6. `H6-E2`への含意（判断はしない）

`subagent_fork`の危険性は**「親の会話を運ぶこと」ではなく「いつ運ぶか」**で決まる。**完了Turnを挟めば継承され**、**同一Turn内なら継承されない**。機密Profileで連鎖を扱うなら、**深さの上限**だけでなく**Turn境界をまたぐforkの可否**も分けて決める必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
