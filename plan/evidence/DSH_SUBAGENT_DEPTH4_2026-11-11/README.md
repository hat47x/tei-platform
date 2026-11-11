# `maxDepth 4`では深さ4まで伸び、5段目が`exceeds maxDepth 4`で止まる（2026-11-11、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-11（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルMock Endpointのみ**。実Providerへの送出は**NOT RUN**。
- 先行記録: [深さ上限を上げた連鎖](../DSH_SUBAGENT_DEPTH_RAISED_2026-11-09/README.md)（`maxDepth 3`で深さ3）。本記録は、そこで未確定とした「**`maxDepth`を`4`以上にした場合**」を埋める。

## 1. 何をしたか

`maxDepth: 4`を`--patch`で設定し（`--dump-config`で適用を確認）、**1セッションにつき1回だけ`subagent`を起動**させた。子も同じTool一覧を持つため、**連鎖が自然に伸びる**。

## 2. 観測した結果

**深さ0〜4の5セッションができ、深さ4の子の次の起動が拒否された。**

| 深さ | Session | 最初の要求 |
| --- | --- | --- |
| **0**（根） | `session-0ddce688…` | `002-request.json` |
| 1 | `b5a36daf…` | `004-request.json` |
| 2 | `50169df9…` | `006-request.json` |
| 3 | `d0853fe8…` | `008-request.json` |
| **4** | `8fd60ba4…` | `010-request.json` |

- **到達した最大深さは4**。`maxDepth 3`のときは3だったので、**設定値に従って1段伸びた**。
- 深さ4の子が次を起動しようとした要求（`011-request.json`）に、**`exceeds maxDepth 4`**が現れた。
- 根のTool結果は**`started subagent` が1件**（根の1回だけが表に出る。孫以降の起動は子の側で起きるため、根のstdoutには出ない）。

## 3. この記録が示すこと

- **`maxDepth`は設定値どおりに効く**。`3`→深さ3、`4`→深さ4で、**1段ずつ伸びる**。
- **拒否の本文に設定値がそのまま現れる**（`exceeds maxDepth 4`）。**観測した値が制限値**であることを記録から読み取れる。
- **深さは根のstdoutだけでは分からない**。**子Sessionの要求を突き合わせて初めて連鎖が見える**。

## 4. この記録が示さないこと

- **`maxDepth 5`以上**。本記録は4まで。
- **1メッセージで複数の子を同時に起動した場合**の深さ（本記録は1セッション1回）。
- **`maxActiveSubagents`との同時作用**（[2つの上限の同時作用](../DSH_SUBAGENT_TREE_CAP_2026-11-10/README.md)は深さ3・同時2で確認済み）。
- **`subagent_fork`の連鎖**（本記録は`subagent`のみ）。
- **履歴が長い場合の各段の増加量**。
- 実Providerの保存、`H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `depth4.json` | `overlay`（`maxDepth: 4`）、`sessionsObserved`・`edgesObserved`、`maxDepthReached`、`depthBySession`（5件）、`rootSessions`、`depthRefusalText`（`exceeds maxDepth 4`）・`depthRefusalSession`、`toolResults`、`requests`（各要求のbytes・SHA-256・Session・Tool数）、`comparison`（`maxDepth 3`の記録との対比） |

生の要求bodyはコミットしていない。**合成のみ**（`CHAIN_PROMPT_CANARY_95n`）。

再現する場合の要点:

```yaml
# 深さ上限を4にする（キーは subagent プラグインの maxDepth）
- id: subagent
  name: "@deepseek-ai/dsh-subagent"
  config:
    maxDepth: 4
```

```sh
# 1セッションにつき1回だけ subagent を起動させる（子も同じTool一覧を持つ）
python mock_provider_subagent_chain.py 8799 <record-dir> 1
dsh --profile headless --patch <overlay> --permission-mode read-only \
    --approval-policy never --json "Start a subagent chain."
```

- **Mockの「1セッション1回」を守る**。同じセッションで何度も起動させると、**同時数の上限に先に当たって深さが見えなくなる**（本記録でも最初の試行がこれで`active child limit: 8`になった）。
- 深さは**`started subagent <id>`の辺を根からたどって**求める。**根のTool結果だけでは分からない**。

## 6. `H6-E2`への含意（判断はしない）

深さ上限は**設定できる統制**であり、**既定値に依存した「深くならない」を保証として書けない**（同時数上限と同じ性質）。機密ProfileでAgentの広がりを抑えるなら、**深さと同時数の両方を明示して固定し、拒否の本文で実際の値が効いていることを確かめる**必要がある。どの値を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
