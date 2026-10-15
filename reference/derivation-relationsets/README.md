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
