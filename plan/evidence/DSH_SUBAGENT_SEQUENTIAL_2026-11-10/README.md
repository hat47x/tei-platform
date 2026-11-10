# 同時数の上限は「同時」に効き、時間を分けた起動は止まらない（2026-11-10、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-10（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルMock Endpointのみ**。実Providerへの送出は**NOT RUN**。
- 先行記録: [同時起動の上限は設定値に従う](../DSH_SUBAGENT_ACTIVE_CAP_2026-11-10/README.md)。本記録は、そこで未確定とした「**時間を分けた起動**」を埋める。

## 1. 何をしたか

`maxActiveSubagents`を**3**にしたまま、**1ターンにつき`subagent`を1回だけ**呼ばせた。子は**起動直後に完了**させ、**次の要求の前に必ず終わらせた**。これを**4回**繰り返した。

## 2. 観測した結果

| 項目 | 値 |
| --- | --- |
| 上限 | **3** |
| 要求した起動 | **4**（1回ずつ） |
| **起動成功** | **4** |
| **拒否** | **0** |
| 作られた子Session | **4**（すべて別ID） |
| モデル要求 | 12件 |

- **1体ずつなら4体すべて通り、拒否は0件**だった。上限は3なのに4体目も起動している。
- 子SessionのIDは4つとも異なり、**実際に4つの子が作られた**ことを確認した。

### 並列の場合との対比

| 起動のしかた | 上限 | 起動成功 | 拒否 |
| --- | --- | --- | --- |
| **並列**（1メッセージで6体・子を20秒生かす） | 3 | **3** | **3** |
| **順次**（1ターン1体・起動直後に完了） | 3 | **4** | **0** |

## 3. この記録が示すこと

- **`maxActiveSubagents`は「同時に生きている子」の上限**であり、**起動した総数の上限ではない**。
- **時間を分ければ、同じ上限でも何体でも起動できる**。本記録では上限3に対して4体が通った。
- 機密の観点では、**「同時数を絞ったから子の数は増えない」という保証は成り立たない**。並列を止めても、**順に起動すれば総数は増える**。

## 4. この記録が示さないこと

- **上限を大きく超える順次起動**（4体だけで、10体・100体は試していない）。
- **子が完了しないまま次を要求した場合**（本記録は必ず完了させてから次を要求した）。
- **`maxDepth`との同時作用**（本記録は`maxDepth`既定`1`で、子は孫を起動していない）。
- **時間あたりの起動数の制限**（レート制限があるかは未確認）。
- Telemetry・MCP経路、実Providerへの送出、Provider側の保存。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `sequential.json` | overlay、`mode`、`requested`・`started`・`refused`、`childSessions`（4件のID）・`childSessionCount`、`results`（4件の`status`・`started`・`limitError`・本文）、`requests`（12件のbytes・SHA-256・Tool数・メッセージ数・Session ID）、`totalRequestBytes`、`contrast`（並列時の3/3） |

生の要求bodyはコミットしていない。**合成のみ**（`SEQ_CANARY_n`・`SEQ_CHILD_n`）。

再現する場合の要点:

```sh
# 1ターン1体ずつ起動し、子は起動直後に完了させる（子に子を産ませない）
python mock_provider_seq_subagents.py 8799 <record-dir> 4 0
dsh --profile headless --patch <maxActiveSubagents 3> --permission-mode read-only \
    --approval-policy never --json "Start four subagents one at a time."
```

- **子Sessionにも同じTool一覧が渡る**ため、Mockが子にも`subagent`を強制すると孫が増えて結果が読めなくなる。**根のSessionだけを対象にする**。
- 判定は**起動成功/拒否の件数**と**子SessionのID集合の大きさ**の両方で行う。

## 6. `H6-E2`への含意（判断はしない）

同時数の上限は**同時性**に対する統制であり、**総量**を抑えるものではない。機密Profileで子Agentの数を抑えたいなら、**同時数ではなく、起動の総数・頻度・深さ**を別に扱う必要がある。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
