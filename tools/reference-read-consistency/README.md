# Data Derivation D11：読取り整合性・Snapshot保証 Reference（#1708）

本ディレクトリは **一回の導出評価の入力読取りの整合保証を読取り専用で判定する小さな参照** であり、TEIの正式な分散トランザクション制御、全Provider共通snapshot API、汎用キャッシュ、認可判定、受入済みAPIではありません。実装主体は `readconsistency.go`、受入は `readconsistency_test.go`、起動点は `cmd/tei-read-consistency/` です。`正本`を書き換える経路はありません。

## 責務と分界

- **#1641（D3）**：入力鮮度・キャッシュ・確定業務値の区別は委譲先です。本参照は「一回の評価中の入力読取りの整合保証レベル」だけを判定します。
- **#179（Database Contract Query）／#1121（Target Realization・Lock）**：既存Driver／Database Contractの保証を参照し、新しい分散TX基盤や全Provider共通snapshot APIを先行実装しません。
- **#1693（D5）**：行・テナント・項目の認可は委譲します。**#1696（D7）**：評価根拠の開示は委譲します。Providerの資格情報を規則本文やEvidenceへ埋め込みません。
- **#1637（D1）**：規則と入力を前提にします。`正本`を書き換えません。

## 区別する一致条件（評価候補の用語）

| レベル | 意味 |
| --- | --- |
| `same-record` | 同一rowの取得値 |
| `same-transaction-snapshot` | Providerが保証するスナップショット |
| `version-aligned` | 入力版一致を検証可能 |
| `best-effort` | 同一時点保証なし（案件の明示受容が必要） |

これらは恒久的Core enumではなく評価候補の用語です。Providerの自己申告やtimestampが近いことだけで同一スナップショットとはみなしません。

## 判定する意味（採用前）

| 条件 | 期待 | 結果コード |
| --- | --- | --- |
| 固定したRevisionで読取り | そのRevisionの値 | `RESULT_OK` |
| 強い要求でRevision混在 | どのRevisionにもない値を確定値にしない | `MIXED_SNAPSHOT`（`mixedValue` を併記） |
| best-effort受容でRevision混在 | 非原子のResult（強い整合を偽装しない） | `NON_ATOMIC_READ` |
| best-effort未受容 | 拒否 | `BEST_EFFORT_NOT_ACCEPTED` |
| 異なるProviderで強い同時点保証 | 保証不在なら不成立 | `UNSUPPORTED_CONSISTENCY` |
| Providerが要求保証を持たない | 不成立 | `UNSUPPORTED_CONSISTENCY` |
| snapshot token失効／入力未取得 | NULLや0としない | `INPUT_UNAVAILABLE` |
| 入力不在／NULL | 個別診断 | `INPUT_MISSING`／`NULL_INPUT` |
| 参照先に一致する行が無い | 入力不在と区別する | `ABSENT_RELATION` |
| Providerが途中でsnapshotを失効 | NULLや0にしない | `INPUT_UNAVAILABLE` |
| 同じ入力集合で要求レベルだけが違う | 強い要求は確定値を返さず、受容されたbest-effortだけが非原子Resultを返す | `MIXED_SNAPSHOT`／`NON_ATOMIC_READ` |
| 再試行で認可版が変化 | 再認可要求 | `RETRY_REQUIRES_REAUTHORIZATION` |
| 権限拒否 | #1693 へ委譲 | `AUTHORIZATION_DELEGATED` |

## 合成Fixture（同一DBでも発生し得る競合）

在庫・引当済を一つのDBトランザクションで同時更新する。

- R0: 在庫120、引当済100、引当可能20
- R1: 在庫110、引当済90、引当可能20

| 条件 | 本参照の結果 |
| --- | --- |
| R0で固定した読取り | `RESULT_OK`、`result=20` |
| R1で固定した読取り | `RESULT_OK`、`result=20` |
| R0在庫120 と R1引当済90 を混合、`same-transaction-snapshot` | `MIXED_SNAPSHOT`、`mixedValue=30`（確定値として返さない） |
| 同じ混合、`best-effort` を案件が受容 | `NON_ATOMIC_READ`、`result=30`（非原子的状態を明示） |
| 異なるProvider A/B で強い同時点保証 | `UNSUPPORTED_CONSISTENCY` |
| Providerがsnapshot tokenを失効 | `INPUT_UNAVAILABLE` |

## 実行方法

```sh
go run ./tools/reference-read-consistency/cmd/tei-read-consistency -input request.json
```

`-input -`（既定）は標準入力から読みます。入力例：

```json
{
  "requiredLevel": "same-transaction-snapshot",
  "providerGuarantee": "same-transaction-snapshot",
  "operation": "subtract",
  "reads": [
    { "inputId": "在庫", "revision": "R0", "state": "present", "value": "120" },
    { "inputId": "引当済", "revision": "R1", "state": "present", "value": "90" }
  ]
}
```

## 限界

- 合成入力に対する静的な判定です。実Providerの保証、実DBのIsolation Level、実際の同時更新下での挙動の証拠ではありません。
- 合成fixtureとactual Providerの検証結果を区別し、後者は本PRでは未実施です。
- `-race` は cgo を伴わない可搬Goでは実行していません。
- M2/M3（実DBのread snapshot境界、外部Providerの非原子例）は本PRの範囲外であり、未実証として残します。
