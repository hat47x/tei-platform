# `maxDepth 5`では深さ5まで伸び、6段目が`subagent depth 6 exceeds maxDepth 5`で止まる（2026-11-17、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-17（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.3`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成モデルのみ**。実Providerへの接続は**NOT RUN**。
- 先行記録: [`maxDepth 4`の連鎖](../DSH_SUBAGENT_DEPTH4_2026-11-11/README.md)（深さ4まで）。本記録は、そこで未確定とした「**`maxDepth`を`5`以上にした場合**」を埋める。

## 1. 何をしたか

`subagent`の設定を**`maxDepth: 5`**にし、**1セッションにつき1回だけ**`subagent`を起動させた。**線形の連鎖**を作り、**各セッションの要求を突き合わせて深さ**を求めた。

## 2. 観測した結果

### 6セッション、深さ0〜5

| 深さ | Session | 最初の要求 |
| --- | --- | --- |
| **0**（根） | `session-cba8f9f1…` | `007-request.json` |
| 1 | `1ac26f7f…` | `010-request.json` |
| 2 | `e3c92864…` | `013-request.json` |
| 3 | `6d449b37…` | `015-request.json` |
| 4 | `6d3fdca2…` | `017-request.json` |
| **5** | `42bd4fec…` | `019-request.json` |

- **要求は合計14件**。**6セッションすべてが1回ずつ起動を試みた**が、**成功したのは5回**（深さ0〜4）。**深さ5の試行は拒否**され、**7セッション目はできない**。
- **`maxDepth 4`では5セッション（深さ0〜4）**だったので、**設定値どおり1段伸びた**。

### 拒否の文言が両方の数を名指しする

深さ5のセッションが**さらにもう1回起動しようとした**とき、次の本文が返った（`020-request.json`）。

```
Error: subagent depth 6 exceeds maxDepth 5
```

- **試みた深さ（6）と上限（5）の両方**が本文に現れる。
- **`maxDepth 4`の記録では `exceeds maxDepth 4` だけ**で、**試みた深さは書かれていなかった**。**この版では情報が増えている**。
- 拒否された**セッションは深さ5**で、**その後は起動していない**。

## 3. この記録が示すこと

- **`maxDepth 5` は深さ5まで通す**（**設定値どおり**）。**6段目は拒否**される。
- **拒否の本文に「試みた深さ」と「上限」の両方が出る**ので、**記録から上限を読み取れる**。
- **3点（3・4・5）で `maxDepth N → 深さ N まで`** が成り立つ。**設定は上限として一貫して効く**。
- **深さは根のstdoutだけでは分からない**。**子セッションの要求を突き合わせて初めて見える**（先行記録と同じ）。

## 4. この記録が示さないこと

- **`maxDepth 6`以上**。**上限そのもの（頭打ち）は未確認**。
- **1メッセージで複数の子を同時に起動した場合**（本記録は**1セッション1回**）。
- **`maxActiveSubagents` との同時作用**（[2つの上限の同時作用](../DSH_SUBAGENT_ACTIVE_CAP_2026-11-10/README.md)は別記録）。
- **`subagent_fork` の連鎖**。
- **実Providerの動作**、**履歴が長い場合の増加量**。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `depth5.json` | `overlay`（`maxDepth: 5`）、`configuredMaxDepth`、`sessionsObserved`・`maxDepthReached`、`depthBySession`、`rootSession`、`requests`（14件のbytes・SHA-256・sessionId・深さ・Tool数）、`refusalText`・`refusalFile`・`refusalSession`・`refusalSessionDepth`・`refusalAttemptedDepth`・`refusalLimit`・`refusalFormat`、`comparison`（3・4・5の3点）、`caveat` |

生の要求bodyはコミットしていない。**合成のみ**（`CHAIN_PROMPT_CANARY_95n`）。

再現する場合の要点:

```yaml
- id: subagent
  config:
    maxDepth: 5
```

- **既存エントリの`id`を指定して`config`を上書きする**。`insert`で新しいエントリを足すと**「service "subagents" has been registered」で有効化に失敗**する（実際に踏んだ）。
- **Mockは1セッションにつき1回だけ起動させる**。何度も起動させると**同時数の上限に先に当たる**。
- **深さは到着順で求める**。**線形の連鎖でのみ正しい**（分岐すると順序だけでは決まらない）。

### 手法上の注意（実際に踏んだ）

- **`insert`で足そうとして失敗**した。**`--dump-config`で既存の`id`を確認**してから**上書き**する必要がある。
- **モデルの応答を遅らせる等でリクエストが挟まると、深さの対応が崩れる**。**セッションIDで束ねて**数える。

## 6. `H6-E2`への含意（判断はしない）

`maxDepth` は**設定値どおりに効く上限**で、**拒否の文言から上限を読み取れる**。機密Profileで委任の深さを制限するなら、**この設定が統制として機能する**。ただし**上限そのもの（どこまで許すか）は版と設定に依存**し、**`6`以上は未確認**である。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
