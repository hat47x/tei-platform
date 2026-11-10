# 同時に起動できるSubagentは8体までで、9体目以降はToolエラーで拒否される（2026-11-09、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-09（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルMock Endpointのみ**。実Providerへの送出は**NOT RUN**。
- 先行記録: [Subagentの連鎖](../DSH_SUBAGENT_CHAIN_2026-11-09/README.md)・[深さ上限を上げた連鎖](../DSH_SUBAGENT_DEPTH_RAISED_2026-11-09/README.md)。本記録は台帳の未確定「**`maxActiveSubagents`（既定`8`）との相互作用**」を埋める。

## 1. 何をしたか

`--dump-config-schema`で`subagent`プラグインの設定に`maxActiveSubagents`（既定`8`）があることは分かっていた（[深さ上限を上げた連鎖](../DSH_SUBAGENT_DEPTH_RAISED_2026-11-09/README.md)）。この上限が**実際に効くか**、効くなら**どこでどう拒否されるか**を測った。

- **1つのassistantメッセージで12個の`subagent`呼び出しを並列に要求**した。
- 子Sessionは応答前に**25秒待つ**ようにし、子が同時に生きている状態を作った。
- **対照**として、子が**待たずに即応答**する条件でも同じ12個を要求した。

## 2. 観測した結果

### 8体が起動し、4体が拒否された

| 条件 | 要求数 | 起動成功 | 拒否 | 子Session |
| --- | --- | --- | --- | --- |
| 子が25秒待つ | 12 | **8** | **4** | **8** |
| **対照**: 子が即応答 | 12 | **8** | **4** | **8** |

- **子の応答速度を変えても8/4で変わらない**。したがって上限は**「同時に生きている子の数」を起動時に見る**もので、**子の完了を待って順に通す形ではない**。
- 拒否は**`isError: true`のTool結果**で、本文は次のとおり。

```
Error: subagent limit reached (active child limit: 8); wait for an existing child to finish or complete this work with the current agents
```

- 起動した8体はすべて`started subagent <id>`（`isError: false`）を返し、**8つの子Sessionが実際に作られた**（`~/.dsh/sessions/`に親1＋子8）。
- 親の永続Logの`subagent/catalog`は**8件**。

### 送出量

| 条件 | 要求数 | 合計bytes |
| --- | --- | --- |
| 子が25秒待つ | 13 | 1,074,677 |
| 対照（即応答） | 13 | 1,074,754 |

- 親1＋子8で**13要求**。子1体あたり**約93,300 bytes**の要求が発生する。

## 3. この記録が示すこと

- **`maxActiveSubagents`（既定`8`）は実際に効く**。12体を同時に要求すると**8体で止まり、残り4体は拒否される**。
- 拒否は**Toolエラー**で、**モデルに「待つか今の子らで完結しろ」と伝わる**。止め方は深さ上限と同じく**Toolレベルのエラー**である。
- **子の速さに依存しない**ため、**「すぐ終わるから大丈夫」という前提は成り立たない**。同じメッセージで9体以上を要求すると、**その回のバッチは8体で切られる**。

## 4. この記録が示さないこと

- **時間を分けて起動した場合**（1体ずつ順に起動し、その都度完了させる形）。本記録は**同一メッセージ内の並列要求**だけ。
- `maxActiveSubagents`を**変えた場合**の挙動（本記録は既定値のまま）。
- **`maxDepth`との同時作用**（本記録は`maxDepth`既定`1`で、子は孫を起動していない）。
- **Telemetry（OTel）・MCP経路**。headlessプロファイルには`dsh-session-telemetry-otel`が載っていない（[台帳](../DSH_HARNESS_EVIDENCE_LEDGER_2026-11-09.md)§4）。
- 実Providerへの送出、Provider側の保存・保持期間。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `concurrent-limit.json` | 要求数・起動数・拒否数・`activeChildLimitInMessage`・子Session数、親Logの要約（行数・event種別・`catalog`件数・`tool/result`12件の`isError`と本文）、13要求のbyte数・SHA-256・Tool数・メッセージ数・Session ID・水位、対照条件（`controlInstantChildren`）の同項目 |

生の要求bodyはコミットしていない。**合成canaryのみ**（`MANY_CANARY_98N`）。

再現する場合の要点:

```sh
# 1メッセージで12個の subagent 呼び出しを返し、子は25秒待つ
python mock_provider_many_subagents.py 8799 <record-dir> 12 25
dsh --profile headless --permission-mode read-only --approval-policy never \
    --json "Start twelve subagents at once."
```

- **子を待たせない対照**（`<child-sleep>`を`0`）でも結果が変わらないことを確かめる。変わらなければ上限は**同時数**を見ている。
- 拒否の本文に**`active child limit: 8`**が入る。`isError`は**`data.message.isError`**にある。

## 6. `H6-E2`への含意（判断はしない）

同時起動数には**既定で8体の上限**があり、超えた分は**Toolエラーで拒否**される。深さ上限と同じく、**上限値は設定できる**（`maxActiveSubagents`）ため、**既定値に依存した「増えない」を保証として書けない**。機密Profileで並列起動を扱うなら、**深さ**と**同時数**を別々に固定して検証する必要がある。どの値を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
