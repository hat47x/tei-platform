# Subagentの連鎖は`maxDepth 1`で止まる（2026-11-09、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-09（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルMock Endpointのみ**。実Providerへの送出は**NOT RUN**。
- 先行記録: [Subagent経路](../DSH_SUBAGENT_EGRESS_2026-11-09/README.md)・[Subagent fork](../DSH_SUBAGENT_FORK_2026-11-09/README.md)。本記録は台帳の未確定「**子からの連鎖起動**」を埋める。

## 1. 何をしたか

**Sessionごとに1回**`subagent`を強制するMockを立てた。子Sessionも同じMockに当たるため、子が子を起動すれば連鎖する。上限を設けずに走らせ、**いくつSessionができるか**と**どこで止まるか**を見た。

各Sessionについて、永続Logの`subagent/descriptor`（自分が子であること）と`subagent/catalog`（子を起動したこと）を数え、`tool/result`の`isError`と本文を読んだ。

## 2. 観測した結果

### Sessionは3つで止まった

| Session | 役割 | 要求数 | 水位 | `descriptor` | `catalog` | 深さエラー |
| --- | --- | --- | --- | --- | --- | --- |
| `…995b3c96` | **親** | 6 | `-1`→`52` | 0 | **2** | 0 |
| `…eb2a52a6` | **子1** | 3 | `-1`→`26` | **1** | 0 | **2** |
| `…5c7fd11b` | **子2** | 3 | `-1`→`26` | **1** | 0 | **2** |

- **親は2回の`subagent`起動に成功**した（`started subagent …`、`isError: false`）。
- **子はどちらも2回ずつ`subagent`を要求し、2回とも拒否された**。
- **孫Sessionは1つも作られなかった**（`~/.dsh/sessions/<cwd-slug>/` に3ディレクトリのみ）。

### 拒否の形

子の`tool/result`（永続Log、`data.message`）:

```json
{"role":"tool","toolCallId":"toolu_ch_4",
 "content":[{"type":"text","text":"Error: subagent depth 2 exceeds maxDepth 1"}],
 "isError": true}
```

- **拒否は`isError: true`のTool結果**として返る。**Policy拒否でもSandbox拒否でもない**（Tool自身の深さ検査）。
- 拒否文は**子の次のモデル入力にも載る**（要求005・007のbodyに`exceeds maxDepth`を含む）。つまり**拒否の事実と文言も外部へ送られる**。
- 拒否後も子は作業を続け、**要求数3**（初回＋拒否2回ぶんの再試行）を使った。

### 設定の手掛かり

同梱物に`subagentMaxDepth`（表示名 "Maximum recursion depth"）という設定キーがある。本記録では**値を変えていない**（`1`のまま）。したがって「**この既定値では1**」が観測であり、「**常に1**」ではない。

## 3. この記録が示すこと

- **子から孫を起動する連鎖は、既定の`maxDepth 1`で止まる**。暴走して無限にSessionが増える形にはならない。
- 止め方は**Toolレベルのエラー**であり、**子は「起動できなかった」ことを知らされ、その文言がモデル入力にも載る**。
- 連鎖が止まっても、**子は拒否された回数だけ要求を消費する**（本記録では各子3要求）。「止まったから送出が増えない」とは言えない。
- Session Log拡張は**3 Sessionすべてが送る**（親も子も`sessionFormatVersion 4`の拡張を持つ）。子2つ分のLogも別ディレクトリに残る。

## 4. この記録が示さないこと

- `subagentMaxDepth`を**上げた場合**の連鎖と、そのときの送出量（本記録は既定値のまま）。
- **`subagent_fork`の連鎖**（本記録は継承しない`subagent`だけ）。
- **Telemetry（OTel）・MCP経路**。headlessプロファイルには`dsh-session-telemetry-otel`が載っていない（[台帳](../DSH_HARNESS_EVIDENCE_LEDGER_2026-11-09.md)§4）。
- 実Providerへの送出、Provider側の保存・保持期間。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `chain.json` | 12要求のbyte数・SHA-256・Tool数・メッセージ数・Session ID・水位・event数・`exceeds maxDepth`の有無、3 Sessionの永続Log要約（行数・event種別・`descriptor`/`catalog`件数・`tool/result`の`isError`と本文）、`chainDepthObserved`・`grandchildSessions`・`configKey` |

生の要求bodyはコミットしていない。**合成canaryのみ**（`CHAIN_PROMPT_CANARY_95N`）。

再現する場合の要点:

```sh
# Sessionごとに1回 subagent を強制する（子も同じMockに当たる）
python mock_provider_subagent_chain.py 8799 <record-dir> 2
DEEPSEEK_API_KEY=dummy DEEPSEEK_BASE_URL=http://127.0.0.1:8799 \
  dsh --profile headless --permission-mode read-only --approval-policy never --json "Launch a chain of subagents."
```

- 子のSessionは`~/.dsh/sessions/<cwd-slug>/<child-id>/session.v4.jsonl.zstd`に**別ディレクトリ**で置かれる。親だけを見ると連鎖の有無を誤る。
- `isError`は**`data.message.isError`**にある（`data.isError`ではない）。初回の解析でこのパスを取り違え、拒否を「成功」と読み違えた。

## 6. `H6-E2`への含意（判断はしない）

連鎖起動は**既定で深さ1に制限**されており、無制限の増殖にはならない。ただし止め方は**Toolエラー**であって**送出の停止ではない**ため、拒否された試行も**モデル入力とSession Logとして外部へ出る**。機密Profileで「連鎖を許すか」を扱うなら、**深さの上限値**と**拒否の通知方法**を分けて決める必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
