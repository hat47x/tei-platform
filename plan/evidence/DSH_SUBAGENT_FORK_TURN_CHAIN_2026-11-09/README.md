# Turn境界をまたぐfork連鎖では、親の完了Turnが深さ3まで運ばれる（2026-11-09、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-09（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルMock Endpointのみ**。実Providerへの送出は**NOT RUN**。
- 先行記録: [Subagent fork](../DSH_SUBAGENT_FORK_2026-11-09/README.md)・[forkの連鎖](../DSH_SUBAGENT_FORK_CHAIN_2026-11-09/README.md)。本記録は、そこで未確定とした「**完了Turnを挟んだforkの連鎖**」を埋める。

## 1. 何をしたか

前記録は**同一Turnの途中**でforkしており、未完了Turnは継承されなかった。本記録は**Turn境界をまたぐ**形を測る。

1. **Turn 1**で`pwsh`から合成canary `PARENT_CANARY_971` を出させ、Turnを完走させた。
2. **Turn 2**で同じSessionを再開し、`subagent_fork`を強制した。子もまた`subagent_fork`を強制され、深さ上限（`maxDepth 3`）まで連鎖する。
3. 各Sessionの**初回要求のメッセージ数**と**canaryの有無**を測った。

## 2. 観測した結果

### canaryは深さ3まで届いた

| Session | 深さ | 要求数 | `catalog` | 初回要求のメッセージ数 | 親canary |
| --- | --- | --- | --- | --- | --- |
| `…0e9c34`（親） | 0 | 5 | **1** | 5（Turn 2） | **あり** |
| `…e764ac` | 1 | 3 | **1** | **5** | **あり** |
| `…f176b3` | 2 | 3 | **1** | **5** | **あり** |
| `…3b5c02` | 3 | 2 | 0 | **5** | **あり** |

- **forkした子の初回要求はどれもメッセージ数5**で、**Turn 1で出したcanaryを含む**。**深さ3まで届いた**。
- 各Sessionの永続Logにもcanaryが**7回**現れる。
- 深さ3は`subagent_fork`を試みて**`Error: subagent depth 4 exceeds maxDepth 3`**（`isError: true`）で止まった。上限はforkでも同じ。

### 同一Turn内との対比

| 形 | Session数 | 要求数 | 合計bytes | 子の初回メッセージ数 | canaryの到達 |
| --- | --- | --- | --- | --- | --- |
| **同一Turn内**（前記録） | 4 | 12 | 765,350 | **1** | **届かない** |
| **Turn境界をまたぐ**（本記録） | 4 | **14** | **786,786** | **5** | **深さ3まで届く** |

- 合計bytesの差は**約2.8%**だが、**中身が違う**。Turn境界をまたぐと、**各段が親の完了Turnを引き継ぐ**。
- 子の初回要求は**約101,500 bytes**で、親のTurn 2初回（58,066 bytes）より大きい。**継承した会話の分だけ増える**。

## 3. この記録が示すこと

- **Turn境界をまたげば、fork連鎖は親の完了Turnを深さ3まで運ぶ**。同一Turn内では運ばれない（[forkの連鎖](../DSH_SUBAGENT_FORK_CHAIN_2026-11-09/README.md)）。
- したがって「forkの連鎖は文脈を積み上げない」は**同一Turn内に限った話**である。**Turnを分けてforkすれば、各段が親の履歴を引き継ぐ**。
- 深さ上限は**forkでも同じ**で、深さ4は拒否される。

## 4. この記録が示さないこと

- **Turn 1が複数ある場合**（履歴が長い場合）の増加量。本記録のTurn 1は1往復だけ。
- `maxDepth`を`4`以上にした場合、`maxActiveSubagents`（既定`8`）との相互作用。
- **Telemetry（OTel）・MCP経路**。headlessプロファイルには`dsh-session-telemetry-otel`が載っていない（[台帳](../DSH_HARNESS_EVIDENCE_LEDGER_2026-11-09.md)§4）。
- 実Providerへの送出、Provider側の保存・保持期間。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `fork-turns.json` | overlay、14要求（Turn 1が3・Turn 2が11）のbyte数・SHA-256・Tool数・メッセージ数・ロール・Session ID・水位・親canary/fork canaryの有無・`exceeds maxDepth`の有無、4 Sessionの永続Log要約（行数・event種別・`catalog`件数・`tool/result`の`isError`と本文・canary回数）、`distinctSessions`・`totalRequestBytes` |

生の要求bodyはコミットしていない。**合成canaryのみ**（`PARENT_CANARY_971`・`FORK_TURN_CANARY_972`）。

再現する場合の要点:

```sh
# Turn 1: pwshでcanaryを出させて完走させる（mode=pwsh）
python mock_provider_fork_turns2.py 8799 <dir1> pwsh
dsh --profile headless --patch <maxDepth 3> ... --json "Emit the canary."
# Turn 2: 同じ Session ID で再開し、forkを連鎖させる（mode=forkchain）
python mock_provider_fork_turns2.py 8799 <dir2> forkchain
dsh --profile headless --patch <maxDepth 3> ... --session-id <id> --json "Fork across the turn boundary."
```

- **Turnを分ける必要がある**。同一Turn内でforkすると未完了Turnは継承されない（[forkの連鎖](../DSH_SUBAGENT_FORK_CHAIN_2026-11-09/README.md)）。
- 継承の有無は**子の初回要求のメッセージ数**で判定できる（`prompt`だけなら1、完了Turnを継承していれば5）。

## 6. `H6-E2`への含意（判断はしない）

forkの連鎖が文脈を運ぶかは**Turn境界に依存する**。**Turnを分ければ親の履歴が深さ3まで運ばれ**、各段の要求は継承分だけ大きくなる。機密Profileでfork連鎖を扱うなら、**深さの上限**と**Turn境界をまたぐforkの可否**を分けて決める必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
