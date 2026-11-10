# `subagent.maxDepth` を上げると連鎖が深くなり、Sessionと送出が段ごとに増える（2026-11-09、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-09（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルMock Endpointのみ**。実Providerへの送出は**NOT RUN**。
- 先行記録: [Subagentの連鎖](../DSH_SUBAGENT_CHAIN_2026-11-09/README.md)。本記録は、そこで未確定とした「**`subagentMaxDepth`を上げた場合の連鎖と送出量**」を埋める。

## 1. 何をしたか

既定の深さ上限は`1`だった（[連鎖の記録](../DSH_SUBAGENT_CHAIN_2026-11-09/README.md)）。この上限が**設定で緩められるか**、緩めたときに**連鎖と送出がどう変わるか**を測った。

1. `--dump-config-schema`で上限の在処を特定した。`maxDepth`（既定`1`、`volatile`）は**`subagent`プラグインの設定**で、同じschemaに`maxActiveSubagents`（既定`8`）もある。
2. `--patch`で`subagent`の`maxDepth`を**3**にした（`--dump-config`で適用を確認）。
3. **Sessionごとに1回**だけ`subagent`を強制するMockで連鎖させ、Session数・深さ・拒否の位置・送出量を測った。

## 2. 観測した結果

### Sessionは4つになり、深さ3まで到達した

| Session | 深さ | 要求数 | `descriptor` | `catalog` | 起動の結果 |
| --- | --- | --- | --- | --- | --- |
| `…3dacf9d2`（親） | 0 | 3 | 0 | **1** | `started subagent …`（成功） |
| `…937c2d20` | 1 | 3 | **1** | **1** | `started subagent …`（成功） |
| `…ebcf19c1` | 2 | 3 | **1** | **1** | `started subagent …`（成功） |
| `…2c5a04a8` | 3 | 2 | **1** | 0 | **`Error: subagent depth 4 exceeds maxDepth 3`**（`isError: true`） |

- **既定（`1`）では深さ1で止まった**のに対し、`3`では**深さ3まで連鎖し、深さ4で止まった**。上限は**設定で緩められる**。
- 拒否の形は既定のときと同じで、**`isError: true`のTool結果**として返り、**その文言が当該Sessionのモデル入力にも載る**。
- 深さ3のSessionは**要求2**（起動1回＋拒否後の1回）で終わった。深さ1・2のSessionは**要求3**。

### 送出量

- **12要求・合計765,677 bytes**（要求bodyの実byte数の合計）。
- **Sessionごとに約52,000〜54,000 bytes**の永続Log（展開後）を持つ。深さが1つ増えるたびに、**その深さのSessionの分だけ**Logと要求が増える。
- 既定（深さ1）の測定は**Mockの発火回数が異なる**（1 Sessionあたり2回）ため、**byte数の直接比較はしていない**（下記§4）。

## 3. この記録が示すこと

- **`subagent.maxDepth`は設定できる統制**であり、既定は`1`。上げると**連鎖が深くなり、Session数と送出量が段ごとに増える**。
- 上限に当たったときの止め方は**Toolエラー**で、**拒否の文言もモデル入力に載る**（[連鎖の記録](../DSH_SUBAGENT_CHAIN_2026-11-09/README.md)と同じ）。
- 「深さ1で止まるから安全」は**既定値の話**であり、**設定を変えれば同じ構成で深くなる**。機密Profileで連鎖を制限するなら、**既定値に依存せず上限値を明示して固定する**必要がある。

## 4. この記録が示さないこと

- **既定（深さ1）との正確なbyte比較**。本記録のMockは1 Sessionあたり1回発火、先行記録は2回発火で、**発火方針が違う**ため要求数も総量も同条件ではない。比較は**構造（深さ・Session数）**に限る。
- `maxDepth`を**さらに上げた場合**（`4`以上）や、`maxActiveSubagents`（既定`8`）との相互作用。
- **`subagent_fork`の連鎖**。本記録は継承しない`subagent`だけ。
- **Telemetry（OTel）・MCP経路**。headlessプロファイルには`dsh-session-telemetry-otel`が載っていない（[台帳](../DSH_HARNESS_EVIDENCE_LEDGER_2026-11-09.md)§4）。
- 実Providerへの送出、Provider側の保存・保持期間。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `depth3.json` | overlayの内容、12要求のbyte数・SHA-256・Tool数・メッセージ数・Session ID・水位・`exceeds maxDepth`の有無、4 Sessionの永続Log要約（行数・event種別・`descriptor`/`catalog`件数・`tool/result`の`isError`と本文）、`chainDepthObserved`・`distinctSessions`・`totalRequestBytes` |

生の要求bodyはコミットしていない。**合成canaryのみ**（`CHAIN_PROMPT_CANARY_95N`）。

再現する場合の要点:

```yaml
# overlay: 深さ上限を上げる（キーは subagent プラグインの maxDepth）
- id: subagent
  name: "@deepseek-ai/dsh-subagent"
  config:
    maxDepth: 3
```

```sh
# 適用の確認（合成後の木に出る）
dsh --profile headless --patch <overlay> --dump-config
# Sessionごとに1回 subagent を強制する
python mock_provider_subagent_chain_once.py 8799 <record-dir> 3
dsh --profile headless --patch <overlay> --permission-mode read-only \
    --approval-policy never --json "Launch a chain of subagents."
```

- 上限の在処は`--dump-config-schema`で確認できる（`maxDepth`の既定は`1`、同じschemaに`maxActiveSubagents`の既定`8`がある）。
- 子Sessionの永続Logは`~/.dsh/sessions/<cwd-slug>/<child-id>/`に**別ディレクトリ**で置かれる。親だけを見ると連鎖を誤る。

## 6. `H6-E2`への含意（判断はしない）

連鎖の深さは**既定値であって固定仕様ではない**。機密Profileで「連鎖を許さない」を主張するなら、**既定値に依存せず上限値を明示して固定し、その値を検証で確かめる**必要がある。また上限を上げれば**段ごとにSessionと送出が増える**ため、深さの上限は**送出量の統制**としても扱える。どの値を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
