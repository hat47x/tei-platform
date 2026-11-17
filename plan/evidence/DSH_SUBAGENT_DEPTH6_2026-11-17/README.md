# `maxDepth 6`でも深さ6まで伸び、7段目が`subagent depth 7 exceeds maxDepth 6`で止まる（2026-11-17、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-17（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.3`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成モデルのみ**。実Providerへの接続は**NOT RUN**。
- 先行記録: [`maxDepth 5`の連鎖](../DSH_SUBAGENT_DEPTH5_2026-11-17/README.md)（深さ5まで）。本記録は、そこで未確定とした「**`maxDepth` を `6` 以上にした場合**」の**`6`**を埋める。

## 1. 何をしたか

`subagent`の**既存エントリ`id`に`config`を上書き**して**`maxDepth: 6`**にし、**1セッションにつき1回だけ**起動させた。

## 2. 観測した結果

### 7セッション、深さ0〜6

| 深さ | Session | 最初の要求 |
| --- | --- | --- |
| **0**（根） | `session-ddd23def…` | `001-request.json` |
| 1 | `d89e2354…` | `004-request.json` |
| 2 | `02ffdc8e…` | `006-request.json` |
| 3 | `d562ff1b…` | `009-request.json` |
| 4 | `3a9d80ed…` | `011-request.json` |
| 5 | `5bfff3e5…` | `013-request.json` |
| **6** | `5ca05d28…` | `015-request.json` |

- **要求は合計16件**。**起動の試行は7回、成功6回、拒否1回**。
- **`maxDepth 5`では6セッション**だったので、**設定値どおり1段伸びた**。

### 拒否の文言

深さ6のセッション（**1つだけ**）がさらにもう1回起動しようとした結果（`016-request.json`）:

```
Error: subagent depth 7 exceeds maxDepth 6
```

- **試みた深さ（7）と上限（6）の両方**が本文に現れる。
- **拒否されたセッションは深さ6**で、**8セッション目はできない**。

## 3. この記録が示すこと

- **`maxDepth 6` は深さ6まで通す**。**7段目は拒否**される。
- **本記録が測ったのは `6` の1点**である。**3・4・5 は別記録の値**（下の比較表）で、**本記録の生データには含まれない**。**4点を並べると設定値どおりに1段ずつ伸びている**。
- **拒否の本文から上限を読み取れる**（**試みた深さと上限の両方**が出る）。

### 比較（3・4・5 は別記録）

| 設定 | 到達した深さ | セッション | 出典 |
| --- | --- | --- | --- |
| `3` | 3 | 4 | 先行記録 |
| `4` | 4 | 5 | [depth4](../DSH_SUBAGENT_DEPTH4_2026-11-11/README.md) |
| `5` | 5 | 6 | [depth5](../DSH_SUBAGENT_DEPTH5_2026-11-17/README.md) |
| **`6`** | **6** | **7** | **本記録** |

### 深さの求め方（解釈であること）

生データに**明示的な親子リンクは無い**。深さは**セッションIDの初出順**と**各セッションが1回だけ起動した事実**から求めた**解釈**である。**線形の連鎖では正しい**が、**分岐すると順序だけでは決まらない**。

## 4. この記録が示さないこと

- **`maxDepth 7`以上**。**上限そのもの（プラグイン固有の頭打ちがあるか）は未確認**。
- **1メッセージで複数の子を同時に起動した場合**（本記録は**1セッション1回**）。
- **`maxActiveSubagents` との同時作用**（別記録で深さ3・同時2は確認済み。**本記録では測っていない**）。
- **`subagent_fork` の連鎖**、**実Providerの動作**、**深さ判定がクライアント側だけかどうか**。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `depth6.json` | `overlay`（`maxDepth: 6`）、`configuredMaxDepth`、`sessionsObserved`・`maxDepthReached`、`depthBySession`、`rootSession`、`requests`（16件のbytes・SHA-256・sessionId・深さ・Tool数）、`refusalText`・`refusalFile`・`refusalSession`・`refusalSessionDepth`・`refusalAttemptedDepth`・`refusalLimit`、`spawnAccounting`（**試行7・成功6・拒否1**）、`comparison`（3・4・5・6の4点）、`caveat` |

生の要求bodyはコミットしていない。**合成のみ**（`CHAIN_PROMPT_CANARY_95n`）。

再現する場合の要点:

```yaml
- id: subagent
  config:
    maxDepth: 6
```

- **既存エントリの`id`に`config`を上書きする**。`insert`で新しいエントリを足すと**`service "subagents" has been registered`で有効化に失敗**する（先行記録で踏んだ）。
- **Mockは1セッションにつき1回だけ起動させる**。
- **深さは到着順で求める**（**線形の連鎖でのみ正しい**）。

## 6. `H6-E2`への含意（判断はしない）

`maxDepth` は**6でも設定値どおりに効く**。**3〜6を並べると1段ずつ伸びている**（**ただし本記録が測ったのは`6`のみ**）ので、**委任の深さを制限する統制として期待できる**。ただし**7以上と、プラグイン固有の上限は未確認**である。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
