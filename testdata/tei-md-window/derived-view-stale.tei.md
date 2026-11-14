# 系列別Window平均：派生表示の不一致

これはCTK-13〜17の**正式Reader導入前の入力候補**です。通常のMarkdownとして読めますが、現行`*.tei.md` Loaderで構文・意味が受理されたわけではありません。定義元は観測表と計算条件の表であり、説明文や派生表示は実装の命令ではありません。

## 観測データ（定義候補）

| obs_id | grp | seq | value |
| --- | --- | ---: | ---: |
| A1 | A | 1 | 10 |
| A2 | A | 2 | 20 |
| A3 | A | 3 | 30 |
| A4 | A | 4 | 40 |
| B1 | B | 1 | 5 |
| B2 | B | 2 | NULL |
| B3 | B | 2 | 15 |

## 計算条件（定義候補）

| operator_id | aggregate | partition | order | frame |
| --- | --- | --- | --- | --- |
| moving | AVG(value) | grp | seq, obs_id | ROWS BETWEEN 1 PRECEDING AND CURRENT ROW |
| cumulative_rows | AVG(value) | grp | seq, obs_id | ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW |
| cumulative_range | AVG(value) | grp | seq | RANGE BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW |

## 入力と結果の扱い

- `obs_id`は行の安定ID候補です。同じIDの二つの観測を勝手に統合しません。
- `B2`行の`NULL`は、行そのものの欠落ではなく**明示的な値の欠損**です。`AVG`ではNULLを数えません。
- `ROWS`は`seq, obs_id`で行順を確定し、`RANGE`は`seq`だけで同順位の`B2`／`B3`を含めます。
- 全七行が必要な場合は入力閉包を検査します。表示の`LIMIT`で先に計算用の行を切り捨てません。
- 独立Expectedは隣接する候補JSONに保持し、生成SQLやその結果から作りません。

## 派生表示（意図的に古い結果）

この表示は直前三行の旧計算を反映したもので、直前二行に変更した上の計算定義と一致しません。**表示結果は入力の意味の正本ではなく、不一致を診断するための例**です。勝手に定義を書き換えません。

| obs_id | 表示されたmoving平均 |
| --- | ---: |
| A3 | 20 |
| A4 | 30 |
| B3 | 10 |
