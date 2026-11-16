# scoped driver の3テストが、宣言と違う型の引数を渡していた（2026-11-17、第86ラウンド）

- 対象: `reference/web-api-contract` の `TestScoped*` / `TestRealDriver*` 3件。
- 前段: [第85ラウンド](../TEI_DOGFOODING_ROUND85_GOFMT_GATE_2026-11-17/README.md)
- 状態: **3件すべて解消。production コードは変更していない。**

## 1. 目的

`repository-health` に残る赤のうち、3件が同じ `reference/web-api-contract` に集中していたので、**1つの原因かを確かめる。**

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `024b3f60a`（`origin/main`） |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 3件は同じメッセージで落ちていた

```text
--- FAIL: TestRealDriverRequiresScopedProviderAndProjectsOnlyPublishedColumns
    scoped query execution unavailable
--- FAIL: TestScopedProviderCannotMutateAuthorizedResponseProjection
    scoped query execution unavailable
--- FAIL: TestScopedDriverRejectsProviderReturningMoreThanAuthorizedTop
    Provider exceeded Top but its result escaped: ... err=scoped query execution unavailable
```

## 4. メッセージが原因を隠していた

`scoped_driver_bridge.go` は**意図的に**プロバイダの文字列を外へ出さない設計で、`default:` で `ErrScopedQueryUnavailable` に潰している。**その設計は正しいが、テストからは原因が見えない。**

一時的に `fmt.Errorf("DIAGNOSIS %w", err)` に差し替えて原因を出した（**診断後に即座に元へ戻した**）。

```text
DIAGNOSIS Query order_entry.recent_orders input from requires std.datetime (time.Time)
DIAGNOSIS Query order_entry.recent_orders requires input from
```

**2種類の原因が1つのメッセージに隠れていた。**

## 5. 本当の原因: フィクスチャが宣言と違う型を渡していた

クエリの宣言はこうなっている。

```text
query order_entry.recent_orders {
    input from std.datetime
```

`internal/driver/query_execution.go` は `std.datetime` に対して **`time.Time` 以外を拒否する**。ところがテストは:

| テスト | 渡していたもの | 正しい形 |
| --- | --- | --- |
| `TestRealDriver...`・`TestScopedProvider...` | 文字列 `"fixture-query-parameter"` | `time.Time` |
| `TestScopedDriverRejects...` | `nil`（入力ごと欠落） | `time.Time` |

**production コードは正しく、フィクスチャが誤っていた。**

## 6. 修正

`fixtureFromTime`（`time.Date(2026,10,1,...)`）を1つ定義し、**引数と期待値が同じ値を使う**ようにした。

`TestScopedDriverRejects...` は **Top 上限の検査**が目的なので、入力を満たしてから Top の判定に到達させる必要があった。`nil` のままだと**入力検査で先に落ち、Top の assert が一度も走っていなかった。**

## 7. 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 文字列引数に戻す | FAIL `scoped query execution unavailable`（**元の症状を再現**） |
| M2 | Top テストの入力を `nil` に戻す | FAIL `Provider exceeded Top but its result escaped`（**元の症状を再現**） |

## 8. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./reference/web-api-contract` | **ok**（修正前は3件 FAIL） |
| CI gate（`--fast`） | **6ステップすべて PASS** |
| production コード | **変更なし**（`git diff` で `_test.go` 1ファイルのみ） |

## 9. 読み取れないこと（NOT RUN）

- `ErrScopedQueryUnavailable` に潰す設計自体の妥当性（情報を漏らさないため意図的。**テスト側の可観測性**は別の課題）。
- 他の残存赤（`TestPeriodsAdvance*`・`TestREAIntake*`・`TestOrgCase*`）。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 10. 位置付け

**「1つのメッセージが2つの原因を隠していた」回**である。3件が同じ文言で落ちていたので同因と見えたが、**実際は「型違い」と「入力欠落」の2種類**だった。潰す設計は正しいので、**隠れた原因は一時的な計装で取り出してから直した。**
