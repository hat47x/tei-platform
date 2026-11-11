# D9 Relation集合・集計範囲・重複計上の意味境界 Reference（#1706）

**状態: 合成Fixtureの読取り専用Reference。Query計画、SQL、認可、Key正本ではない。**

このパッケージは、導出規則へ Relation 集合を入力として束縛するときの**集合の意味**（形・完全性・業務識別・重複・集計方針）を最小限に固定する。SELECT の Filter/Sort/Aggregate/Limit の実行順と計画決定は D2 #1640、行・集計の開示は D5 #1693、Semantic Relation と CandidateKey の正本は #178/#1126 が所有する。ここで第二のQuery Planner・ORM・Key正本を作らない。

## 入力形と完全性

- `Shape`: `scalar`（1行の属性）、`optional-one`（0..1参照）、`many`（1:N集合）、`aggregate`（集約済み結果）を別の入力形として扱う。同名のSQL列やResultの行順から意味を推測しない。
- `Completeness`: `complete` / `partial`（ページ・上位N件・途中切断）/ `unfetched`。表示ページや途中まで取得した集合を完全集合として集計しない。
- `Anchor`: 集合が属する親Entity Instance。集合の完全性と評価のアンカーを区別する。
- `Key`: 明示のキー束縛を必須とする。暗黙のKey推測をしない。
- `many` / `aggregate` の各行は業務識別（CandidateKey または明示のCollection identity）を必須とする。識別がないまま黙って deduplicate しない。

## 集計の方針

`Sum` は宣言された方針だけに従い、全規則へ同じ暗黙方針を強制しない。

- `EmptySetPolicy`（`null` / `zero`）は空集合のときだけ必要。無宣言で勝手に0へ確定しない。
- `NullPolicy`（`propagate` / `skip`）はNULL要素が実際にあるときだけ必要。`skip` で全要素が消えた集合は空集合として扱い、空集合方針を別に要求する。
- `missing` / `absent-relation` / `multiple-relations` の要素はNULLや0へ潰さず、不完全入力として拒否する。

## 重複計上（fan-out）の検出

`Sum` は同一の業務識別が二度現れた行を fan-out による重複として拒否する。`JoinFanOut` は、独立な二つの `many` 集合を素朴な列一致JOINで一つへ渡したときに、行が乗算されることを検出し、`NaiveRows`（素朴な積）と `DistinctRows`（実の行数）を返す。fan-outした集合を集計へ渡さない。`many` を scalar 引数へ束縛することは `AsScalar` が拒否し、`optional-one` が2件一致した場合は `Bind` が多重度違反として拒否して先頭要素を選ばない。

## 合成Fixture（受注）

- O1: L1(2×300=600), L2(1×500=500)、別Relationに配送履歴2件。
- O2: 明細なし。O3: 金額NULLの明細1件。

| 操作 | 期待 | 試験 |
| --- | --- | --- |
| `SUM(O1.lines.amount)` | 1100 | `TestD9SumOfManySet` |
| `O1.lines × O1.deliveries` の素朴SUM | 2200の危険を検出し1100を変えない | `TestD9JoinFanOutDetectedNotSilentlyDoubled` |
| O2の空集合SUM | 宣言方針に従う。無宣言は拒否 | `TestD9EmptySetNeedsDeclaredPolicy` |
| O3のNULL要素SUM | NULL方針に従い、空集合・未取得と混同しない | `TestD9NullElementFollowsDeclaredPolicy` |
| O1の明細1件だけ取得 | 不完全入力として拒否 | `TestD9IncompleteOrUnfetchedSetIsNotSummed` |
| many Relationをscalar引数へ | 形状不一致 | `TestD9ManyRelationToScalarRejected` |
| one Relationに2件一致 | 多重度違反 | `TestD9OptionalOneCardinalityViolation` |
| 同一業務識別の重複 | fan-outとして拒否 | `TestD9DuplicateBusinessIdentityIsFanOut` |

## 集約済み結果の再集計適合（#2593）

**状態: 合成Fixtureの読取り専用Reference。Measureの正本、Metric DSL、Query Optimizer、BI Engineではない。**

`Sum` は `aggregate`（集約済み）の集合を受け付けない。集約済みの行をもう一度集約してよいかは、その行を作った演算の法則で決まり、`SetRef` はそれを持たないためである。集約済みの入力は、宣言付きの `Reaggregate(Claim)` を通す。判定は数値型や表示値の一致を見ず、宣言された次の情報だけを使う。

| 宣言 | 内容 | 欠けたとき |
| --- | --- | --- |
| `Law` | `sum` / `mean` / `ratio` | `unknown`（演算名や型から推測しない） |
| `Components` | 法則が要求する中間状態。`mean` は `sum` と `count`、`ratio` は `numerator` と `denominator`。各要素は既存の `SetRef`（行の `Identity` はグループの識別）と単位 | `ineligible`（`INTERMEDIATE_STATE_LOST`） |
| `BaseGrain` / `Scope` | 元Fact 1件の識別単位、結果を主張する対象集合 | `unknown` |
| `Axis` | 畳み込む軸と種別（`dimension` / `time`） | `unknown` |
| `Partition` | グループが元Factを重複なく分割するか（`disjoint` / `overlapping`） | 未宣言は `unknown`、`overlapping` は `ineligible` |
| `Time` | `period-flow` / `point-stock` と `AsOf` | 未宣言は `unknown` |
| `Policy` | 既存の `AggregatePolicy`。`mean` では `sum` と `count` が共有するNULLの扱いとして必須 | `unknown` |

判定は三値である。`eligible` は宣言された入力が法則に対して十分であること、`ineligible` は宣言された内容から再集計が成立しないと分かること、`unknown` は宣言が足りないことを表す。`ineligible` と `unknown` は値を返さない。理由の分類は、操作面の導出（`internal/operations`）が使う `missing-input` / `missing-semantics` / `conflict` / `out-of-scope` と同じ語を使い、別の語彙を作らない。

結果は丸めない。`Exact` に有理数（`3/4`、`18`）を返し、有限小数で表せるときだけ `Result` に小数を入れる。丸めと単位換算は D8 #1705 が所有する。

| 例（期待値は元Factからの手計算で固定） | 期待 | 試験 |
| --- | --- | --- |
| MC-02 A: 2件・合計20、B: 8件・合計160 | `eligible`、18 | `TestMC02MeanIsRecomposedFromSumAndCount` |
| MC-02 局所平均10と20だけが残っている | `ineligible`。15も30も返さない | `TestMC02MeanOfMeansIsNotAdoptedWhenStateIsLost` |
| MC-03 A: 1/2、B: 8/10 | `eligible`、3/4（0.75） | `TestMC03RatioIsRecomposedFromNumeratorAndDenominator` |
| MC-03 局所比率だけ、または分子と分母の対応が崩れている | `ineligible` | `TestMC03MeanOfRatiosIsNotAdoptedWhenStateIsLost` |
| MC-01 別の明細が同じ金額100 | 200（金額で重複除去しない） | `TestMC01JoinOccurrenceIsNotASecondFact` |
| MC-01 JOINで同じ明細が3回出現／先集約した合計を再JOIN | `ineligible`。600も100も返さない | 同上、`TestMC01PreAggregatedTotalRejoinedIsNotSummedAgain` |
| MC-04 部門別人数2と2、兼務者あり | `ineligible`（4を全体人数にしない） | `TestMC04GroupCountsAreNotAddedWhenGroupsOverlap` |
| MC-05 同じ日の口座残高110と190／日別残高300と300 | 300／`ineligible` | `TestMC05StockIsCombinedAtOnePointNotAcrossTime` |
| Scope・粒度・軸・分割・時間・単位・NULL方針・完全性の欠落 | `unknown`（部分取得は `ineligible`） | `TestMissingDeclarationsStayUnknown`、`TestCompletenessAndNullAreNotGuessed` |
| MC-06 行なし／NULLのみ／部分取得／未取得 | 四つを別の状態として扱い、宣言なしに0へ寄せない | `TestMC06EmptyNullPartialAndUnfetchedStayDistinct` |
| 丸めの段階（0.004が1000件） | 渡された値を正確に合成するだけで丸めない（厳密値なら4、丸め済みなら0） | `TestRoundingStageIsNeverChosenHere` |
| `count` が整数でない、負 | `unknown` | `TestMeanCountMustBeAWholeNumberOfFacts` |

MC-01〜05の期待値は [`plan/fixtures/metric-aggregation/expected.json`](../../plan/fixtures/metric-aggregation/expected.json) に手計算の値として置き、このReference（`TestHandFixedExpectedMatchesSharedFixture`）と同じディレクトリの `verify_sqlite.py` の両方が照合する。どちらの出力もこのファイルの値として使わない。

JSONでの入出力は [`tools/reference-relation-aggregate`](../../tools/reference-relation-aggregate/README.md) の `-mode reaggregate` を使う。

この判定が示すのは、合成入力について宣言が法則に対して十分かどうかだけである。実SQLの生成、実Providerの計算結果、Hostの認可、Ownerによる業務意味の受理は別に確認する。次は扱わない。

- MC-04の正しい人数3の合成。人物の識別集合が必要で、このReferenceは持たない。
- 中央値・分位点、時間方向のStockの演算（期末残高、平均残高など）。`unknown` を返す。
- `Partition` と `Time` の宣言が正しいことの検証。呼び出し側が渡す根拠であり、行の値からは推測しない。
- 丸めをどの段階に置くかの宣言と検査。入力が丸め済みかどうかは区別できない（D8 #1705）。
- グループごとに単位が異なる入力。

## 焦点試験

```sh
gofmt -l reference/derivation-relationsets/*.go
go test -count=1 ./reference/derivation-relationsets
go vet ./reference/derivation-relationsets
```

## 未実装・依存

- 実際の Filter/Sort/Aggregate/Limit の順序、Count/Limit、SQL/Go Lowering は D2 #1640 が所有する。このReferenceはクエリを解析せず、行を絞り込まず、Providerから集合を取得しない。
- 業務上の集合の identity/Key は既存 Semantic Relation と #1126 を再利用する前提であり、ここでは入力として受け取るだけで、KeyやRelationを新設しない。
- 行レベル権限で可視の部分集合と Order 全明細の違い、実行・開示範囲は #1693 が所有する。
- 0件・NULL・未取得・多重度違反の判定は合成Fixtureに限る。実Providerや外部DBでの意味同等性、X-TEA #940 の実動作との一致は未検証。正しい「受注合計1100」の合成試験の成功を、実SQL/Providerの正しさの証明とは扱わない。

実行の有無、exact SHA、成功／失敗／未検証を Issue #1706 と対象PRへ記録する。
