# 同時起動の上限は設定値に従い、3にすると3体で止まる（2026-11-10、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-10（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルMock Endpointのみ**。実Providerへの送出は**NOT RUN**。
- 先行記録: [同時起動の上限](../DSH_SUBAGENT_CONCURRENT_LIMIT_2026-11-09/README.md)（既定`8`で8/4）。本記録は、そこで未確定とした「**`maxActiveSubagents`を変えた場合**」を埋める。

## 1. 何をしたか

`--patch`で`subagent`の`maxActiveSubagents`を**3**にし（`--dump-config`で適用を確認）、**1メッセージで6体の`subagent`を並列要求**した。子は応答前に20秒待たせ、同時に生きている状態を作った。

## 2. 観測した結果

| 項目 | 値 |
| --- | --- |
| 要求した同時数 | **6** |
| **起動成功** | **3** |
| **拒否** | **3** |
| 作られた子Session | **3** |
| 本文に出た上限値 | **`active child limit: 3`** |

- **設定した3で止まった**。既定（8）のときと同じ形で、**値だけが設定に従う**。
- 拒否の本文は次のとおり（上限値の部分だけが変わる）。

```
Error: subagent limit reached (active child limit: 3); wait for an existing child to finish or complete this work with the current agents
```

- 実行は正常終了し（exit 0）、警告やエラーはstdoutのイベントに出なかった。**止まったことは拒否された呼び出しの`status: error`としてのみ現れる**。

## 3. この記録が示すこと

- **`maxActiveSubagents`は実際に効く設定**である。`3`にすれば**3体で止まり**、既定の`8`にすれば8体で止まる（[同時起動の上限](../DSH_SUBAGENT_CONCURRENT_LIMIT_2026-11-09/README.md)）。
- **上限値は拒否の本文に現れる**（`active child limit: N`）。**観測した値がそのまま制限値**であることを、記録から読み取れる。
- したがって**「既定の8で止まる」は既定値の話**であり、**設定を変えれば同じ構成で変わる**。

## 4. この記録が示さないこと

- **時間を分けた起動**（1体ずつ順に完了させる形）。本記録は同一メッセージ内の並列要求だけ。
- **`maxDepth`との同時作用**（本記録は`maxDepth`既定`1`で、子は孫を起動していない）。
- 上限を**0や1**にした場合の挙動。
- **Telemetry（OTel）・MCP経路**、実Providerへの送出、Provider側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `active-cap.json` | overlay（`maxActiveSubagents: 3`）、`requestedConcurrent`・`started`・`refused`・`limitValueInMessage`、`results`（6件の`status`・`started`・`limitError`・本文）、`requests`（byte数・SHA-256・Tool数・メッセージ数・Session ID・水位）、`distinctSessions`・`totalRequestBytes`、`defaultCapFromEarlierRecord` |

生の要求bodyはコミットしていない。**合成のみ**（`MANY_CANARY_98N`）。

再現する場合の要点:

```yaml
# 上限を下げる（キーは subagent プラグインの maxActiveSubagents）
- id: subagent
  name: "@deepseek-ai/dsh-subagent"
  config:
    maxActiveSubagents: 3
```

```sh
# 1メッセージで6体を並列に要求し、子は20秒待たせる
python mock_provider_many_subagents.py 8799 <record-dir> 6 20
dsh --profile headless --patch <overlay> --permission-mode read-only \
    --approval-policy never --json "Start six subagents at once."
```

- 拒否の本文に**`active child limit: N`**が入る。**Nが設定値と一致するか**を確認する。
- 子を待たせないと上限が効かない場合があるため、**子を生かしておく**。

## 6. `H6-E2`への含意（判断はしない）

同時起動数の上限は**設定できる統制**であり、**既定値に依存した「増えない」を保証として書けない**。深さ上限（[深さ上限を上げた連鎖](../DSH_SUBAGENT_DEPTH_RAISED_2026-11-09/README.md)）と同じ性質である。機密Profileで並列起動を扱うなら、**深さと同時数の両方を明示して固定し、拒否の本文で実際の値が効いていることを確かめる**必要がある。どの値を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
