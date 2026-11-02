# 期待状態の比較による更新（実データベース）

- 状態: **参照用の検証（合成データ）**。製品機能ではありません。
- 仮説: [PoCの仮説 H1](../../plan/workstreams/POC_HYPOTHESES_2026-11.md)（同時更新の検出を期待状態の比較で行う）
- 課題: [#2483](https://github.com/hat47x/tei-platform/issues/2483)

## 何を確かめるか

更新を「読んだときの値と一致した場合だけ書く」形にすると、別の書き手が先に状態を変えていた場合に**後勝ちで上書きしない**ことを、実際のPostgreSQLで確かめます。

| 試験 | 確かめること |
| --- | --- |
| `TestExpectedStateStopsTheSecondWriter` | 二つのトランザクションが同じ状態を読み、**同時に**更新を試みる。片方だけが1行を更新し、もう片方は**0行**。勝った側の値が残る |
| `TestExpectedStateAllowsTheSecondWriterAfterAReRead` | 現在の状態を読み直せば、第二の書き手は正しく1行を更新できる |

```sql
UPDATE orders SET status = $1 WHERE order_id = $2 AND status = $3
```

## 実行

```sh
TEI_TEST_POSTGRES_DSN='postgres://...' go test ./tools/reference-expected-state-concurrency -count=1 -v
```

環境変数が無い場合は**スキップ**します。スキップは成功ではありません。実行した場合は、試験ごとに使い捨てのデータベースを作り、終了時に削除します。

## 限界

- 扱うのは一つの行と、`READ COMMITTED` の分離水準です。直列化異常（`SERIALIZABLE` の書き込み衝突）は扱いません。
- 二つのトランザクションは同時に更新を試みますが、実際に並ぶ時刻はPostgreSQLの排他制御に依存します。確かめているのは「古い期待状態での更新が成立しない」ことです。
- 実際のアプリケーションの実行経路（能力の呼出し・Driver）を通していません。SQLの意味だけを確かめる参照試験です。
