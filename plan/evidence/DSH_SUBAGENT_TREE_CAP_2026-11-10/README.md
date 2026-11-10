# 同時数の上限は木全体に効き、深さ上限より先に効く（2026-11-10、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-10（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルMock Endpointのみ**。実Providerへの送出は**NOT RUN**。
- 先行記録: [深さ上限を上げた連鎖](../DSH_SUBAGENT_DEPTH_RAISED_2026-11-09/README.md)・[同時起動の上限は設定値に従う](../DSH_SUBAGENT_ACTIVE_CAP_2026-11-10/README.md)。本記録は、そこで未確定とした「**`maxDepth`と`maxActiveSubagents`の同時作用**」を埋める。

## 1. 何をしたか

`--patch`で**`maxDepth: 3`**と**`maxActiveSubagents: 2`**を同時に設定し（`--dump-config`で適用を確認）、**2段の木**を作った。

1. 根が**子を2体並列**で起動する（子は25秒生きて枠を占有する）。
2. 各子が**孫を2体並列**で起動しようとする（合計4体）。

深さ上限は3なので、**孫（深さ2）は深さの面では許可されている**。

## 2. 観測した結果

| 項目 | 値 |
| --- | --- |
| `maxDepth` | **3** |
| `maxActiveSubagents` | **2** |
| 根が起動した子 | **2**（**2体とも成功**） |
| 子が起動しようとした孫 | **4** |
| **起動できた孫** | **0** |
| 作られたSession | **3**（根＋子2） |
| 深さ上限による拒否 | **0** |
| **同時数上限による拒否** | **2要求**（子2体の各要求） |

- 子が受け取った拒否の本文は次のとおり。

```
subagent limit reached (active child limit: 2); wait for an existing child to finish or complete this work with the current agents
```

- **孫のSessionは1つも作られていない**（記録されたSessionは根と子2体の3つだけ）。
- **深さ上限の拒否（`exceeds maxDepth`）は0件**。深さの面では止まっていない。

## 3. この記録が示すこと

- **`maxActiveSubagents`は木全体で数える**。根が枠を埋めると、**その下の段は起動できない**。
- **深さ上限を上げても、同時数の枠が空かない限り深い段は始まらない**。本記録では`maxDepth 3`でも孫は0体だった。
- 2つの上限は**独立に効くのではなく、先に埋まった方が勝つ**。機密Profileで「深さだけ絞る」「同時数だけ絞る」のどちらか一方を決めると、**意図と違う側で止まる**。

## 4. この記録が示さないこと

- **子が完了してから孫を起動した場合**（本記録は子が生きている間に孫を要求した）。時間を分ければ孫は起動できると[時間を分けた起動](../DSH_SUBAGENT_SEQUENTIAL_2026-11-10/README.md)から推測できるが、**この木では未確認**。
- **`maxActiveSubagents`を4以上にした場合**（本記録は2のみ）。
- **3段以上の木**、`maxDepth`を2以下にした場合との比較。
- `subagent_fork`の木（本記録は`subagent`のみ）。
- Telemetry・MCP経路、実Providerへの送出、Provider側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `tree-cap.json` | overlay（`maxDepth: 3`・`maxActiveSubagents: 2`）、`tree`（子2・孫4・孫起動0）、`rootStarted`／`rootRefused`、`grandchildRefusalText`、`grandchildRefusals`／`depthRefusals`、`sessionsSeen`（3件）、`requests`（9件のbytes・SHA-256・Session ID・水準マーカー・拒否の有無）、`totalRequestBytes`、`conclusion` |

生の要求bodyはコミットしていない。**合成のみ**（`TREE_CANARY_L<n>_<m>`・`TREE_CHILD_L<n>_<m>`）。

再現する場合の要点:

```sh
# 根が子2体を並列起動し、各子が孫2体を並列起動しようとする2段の木
python mock_provider_tree_cap.py 8799 <record-dir> 2 2
dsh --profile headless --patch <maxDepth 3 / maxActiveSubagents 2> \
    --permission-mode read-only --approval-policy never --json "Build a two-level tree."
```

- **子を十分長く生かす**（本記録は25秒）。すぐ完了すると枠が空いて孫が起動でき、結果が変わる。
- 水準は**子のプロンプトに埋めた`LEVEL=n`**から読み取る。ただし**親の文脈にも子のプロンプトが入る**ため、水準の判定は記録側で慎重に行う。
- 判定は**作られたSessionの数**と**拒否の本文**の両方で行う。根のstdoutには子の拒否は出ない。

## 6. `H6-E2`への含意（判断はしない）

深さ上限と同時数上限は**別々の統制ではなく、同じ資源を奪い合う**。機密ProfileでAgentの広がりを抑えるなら、**両方を明示し、どちらが先に効いたかを記録で確かめる**必要がある。片方だけを決めると、**もう片方が意図せず先に効いて、深さの制御が効いていないように見える**。どの値を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
