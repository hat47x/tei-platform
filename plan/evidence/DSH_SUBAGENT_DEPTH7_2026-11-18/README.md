# `maxDepth 7`でも深さ7まで伸び、8段目が`subagent depth 8 exceeds maxDepth 7`で止まる（2026-11-18、#2515 H6-E2）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E2`と`#2509`の経路網羅。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-18（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.3`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **ローカルの合成モデルのみ**。実Providerへの接続は**NOT RUN**。
- 先行記録: [`maxDepth 6`の連鎖](../DSH_SUBAGENT_DEPTH6_2026-11-17/README.md)（深さ6まで）。本記録は、そこで未確定とした「**`maxDepth` を `7` 以上にした場合**」の**`7`**を埋める。

## 1. 何をしたか

`subagent`の**既存エントリ`id`に`config`を上書き**して**`maxDepth: 7`**にした。**Mockを「1セッションにつき1回だけ起動する」ように作った**ので、**各深さの起動は1回ずつ**である（**これは実験の作り方であって、「2回目以降が起きない」という観測ではない**）。

## 2. 観測した結果

### 8セッション、深さ0〜7

| 深さ | Session | 最初の要求 |
| --- | --- | --- |
| **0**（根） | `session-d1b8d0a4…` | `001-request.json` |
| 1 | `c5a16803…` | `004-request.json` |
| 2 | `386a2725…` | `006-request.json` |
| 3 | `5b1fb902…` | `009-request.json` |
| 4 | `f8bc4d62…` | `011-request.json` |
| 5 | `8be79fa4…` | `013-request.json` |
| 6 | `6dc21001…` | `015-request.json` |
| **7** | `5fa5cc2f…` | `017-request.json` |

- **要求は合計18件**。**起動の試行は8回、成功7回、拒否1回**。
- **`maxDepth 6`では7セッション**だったので、**設定値どおり1段伸びた**。

### 拒否の文言

深さ7のセッション（**1つだけ**）がさらにもう1回起動しようとした結果（`018-request.json`）:

```
Error: subagent depth 8 exceeds maxDepth 7
```

- **試みた深さ（8）と上限（7）の両方**が本文に現れる。
- **拒否されたセッションは深さ7**で、**9セッション目はできない**。

## 3. この記録が示すこと

- **`maxDepth 7` は深さ7まで通す**。**8段目は拒否**される。
- **本記録が測ったのは `7` の1点**である。**3〜6 は別記録の値**（下の比較表）。
- **5点（3・4・5・6・7）を並べると、設定値どおりに1段ずつ伸びている**。**上限としての効き方は7まで一貫**している。

### 比較（3〜6 は別記録）

| 設定 | 到達した深さ | セッション | 出典 |
| --- | --- | --- | --- |
| `3` | 3 | 4 | 先行記録 |
| `4` | 4 | 5 | [depth4](../DSH_SUBAGENT_DEPTH4_2026-11-11/README.md) |
| `5` | 5 | 6 | [depth5](../DSH_SUBAGENT_DEPTH5_2026-11-17/README.md) |
| `6` | 6 | 7 | [depth6](../DSH_SUBAGENT_DEPTH6_2026-11-17/README.md) |
| **`7`** | **7** | **8** | **本記録** |

### 深さの求め方（解釈であること）

生データに**明示的な親子リンクは無い**。深さは**セッションIDの初出順**と**各セッションが1回だけ起動した事実**から求めた**解釈**である。**線形の連鎖では正しい**が、**分岐すると順序だけでは決まらない**。

## 4. この記録が示さないこと

- **`maxDepth 8` 以上**と**プラグイン固有の頭打ちがあるか**。**7点では頭打ちは見えていないが、上限が無いとは言えない**。
- **1メッセージで複数の子を同時に起動した場合**（本記録は**1セッション1回**）。
- **`maxActiveSubagents` との同時作用**（別記録で深さ3・同時2は確認済み）。
- **`subagent_fork` の連鎖**、**実Providerの動作**。
- **時間・リソースの消費**（8セッションで6.6秒だったが、計測していない）。
- `H6-E2`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `depth7.json` | `overlay`（`maxDepth: 7`）、`configuredMaxDepth`、`sessionsObserved`・`maxDepthReached`、`depthBySession`、`rootSession`、`requests`（18件のbytes・SHA-256・sessionId・深さ・Tool数）、`refusalText`・`refusalFile`・`refusalSession`・`refusalSessionDepth`・`refusalAttemptedDepth`・`refusalLimit`、`spawnAccounting`（**試行8・成功7・拒否1**）、`comparison`（3〜7の5点）、`caveat` |

生の要求bodyはコミットしていない。**合成のみ**（`CHAIN_PROMPT_CANARY_95n`）。

再現する場合の要点:

```yaml
- id: subagent
  config:
    maxDepth: 7
```

- **既存エントリの`id`に`config`を上書きする**。`insert`で足すと**有効化に失敗**する（先行記録で踏んだ）。
- **Mockは1セッションにつき1回だけ起動させる**。
- **深さは到着順で求める**（**線形の連鎖でのみ正しい**）。

## 6. `H6-E2`への含意（判断はしない）

`maxDepth` は**7でも設定値どおりに効く**。**3〜7を並べると1段ずつ伸びている**（**本記録が測ったのは`7`のみ**）ので、**委任の深さを制限する統制として7段までは期待できる**。ただし**8以上と、プラグイン固有の上限は未確認**である。どの統制を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
