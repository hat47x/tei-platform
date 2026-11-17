# 行スコープのフックを browse（一覧）で使えるようにするか（設計判断、2026-11-17）

## 対象

- `internal/validation/application_pattern.go` の `APPLICATION_PATTERN_INTERACTION_NOT_ALLOWED`
- `internal/application/model.go` の `Interaction`（`Items`／`Navigations`／`Commands`／`Hooks`）
- `internal/hook` の `Runner`・`RunRows`・`decode`（`Interaction` からフック宣言を読む箇所）
- `internal/driver` の browse 実行（`ExecuteBrowseQuery`、`MappedResult`）
- 受容設計 D1（`plan/design/XTEA_LOWCODE_RECEPTION_ARCHITECTURE_2026-11.md`）

## 見つけたこと

行スコープ（`scope row`）のフックは、`RunRows`・`RunReachable`・入口・警告・driver結果からの変換まで実装した。**しかし、実運用では一度も走らない。**

理由は構造的な二重の閉塞である。

1. **`interaction` を持てるのは record Pattern だけ**（`APPLICATION_PATTERN_INTERACTION_NOT_ALLOWED`、「the first interaction contract is limited to record patterns」）。したがって **browse（一覧）は行フックを宣言できない**。
2. record Pattern は `relation` も `query` も拒否されるため、**行の出所を書く欄が `Hook` に無い**。

つまり「行ごとに判定したい」という最も自然な使い先（一覧の各行）に、**宣言する構文が無い**。record 上で `scope row` を宣言することはできるが、record は1レコードを扱う Pattern であり、行集合を供給する経路が無い。

`interaction` の消費者は広い（`application_pattern.go` 9箇所、`calls.go` 6、`sequence.go` 5、`reach.go` 4、`rows.go` 4、`run.go` 4、`view.go` 3 ほか）。`hook` 側は `Interaction.Hooks` から宣言を読むので、**宣言できる Pattern の種類を変えると hook の適用範囲も変わる**。

## 選択肢

### A. `interaction` を browse へ広げる

browse も `Interaction` を持てるようにし、`scope row` のフックを一覧の行に適用する。

- **利点**: 行フックの自然な使い先が開く。`MappedResult` → `RowsFromColumns` → `RunRows` の経路が実際に繋がる。
- **欠点**: 受容設計 D1 の「最初の対話契約は record 限定」という**判断を変更する**。`interaction` の `Items`・`Commands`・`Navigations` が browse でも意味を持つか（項目の並びは一覧の列、遷移は行選択、起動は行操作）を、それぞれ決める必要がある。既存の検査・試験・投影に波及する。
- **コスト**: 中〜大。`Items` の意味が record（単票の項目）と browse（一覧の列）で異なるため、**同じ型で2つの意味を持つ**ことになる危険がある。

### B. `Hook` に行の出所を書く欄を足す（record のまま）

`hook ... { source <query|pattern> ... }` のように、行の出所を宣言に書く。record Pattern 上で「この Query の結果の各行について判定する」と述べる。

- **利点**: `interaction` の拡張をせずに済む。`Items` の意味の二重化を避けられる。行の出所が**明示**になる（現状は「供給源が無い」）。
- **欠点**: record Pattern が query を参照することになり、record が「1レコード」という性格を保てるか要検討。`Hook` の宣言欄が増える。
- **コスト**: 小〜中。検証（出所の解決）と、供給経路（Query 実行 → `RowsFromColumns` → `RunRows`）の実装が要る。

### C. browse 専用の行フック宣言を別に持つ

`interaction` とは別に、browse が `rowHook` のような独立した宣言を持つ。

- **利点**: 意味の二重化を避けつつ browse で行フックを書ける。
- **欠点**: フックの宣言が2箇所（`interaction.Hooks` と `rowHook`）に分かれ、`hook` 側が両方を読む必要がある。**宣言の正本が2つになる**危険。
- **コスト**: 中。

### D. 何もしない（record の行フックは「供給源が来たら使う」まま）

- **利点**: 変更ゼロ。受容設計 D1 を維持。
- **欠点**: 行スコープの機能が**実運用で一度も走らないまま**残る。警告（`..._HAS_NO_SUPPLIER`）は出るので無信号ではないが、実装した意味が無い。

## 既定案: B（`Hook` に行の出所を書く欄を足す）

理由:

1. **`interaction` の `Items` は record と browse で意味が違う。** record の `Items` は「単票の項目の並び」、browse の列は「一覧の列」。同じ型に2つの意味を持たせると、`validation`・`view`・`hook` の各消費者が**どちらの意味かを判別できなくなる**。A はこの危険を避けられない。
2. **行の出所が明示されることが、そもそも不足していた。** 現状は「record 上で row を宣言できるが、行がどこから来るかを誰も言っていない」。B はその宣言を足すので、**供給経路の設計と宣言が一致する**。
3. C は宣言の正本が2つになる。フックは1つの語彙（位相・範囲・reads・uses・produces・on-failure）を持つべきで、入口を2つに分けると片方だけが検査される事故（過去に何度も起きた形）を招く。

覆す条件:

- record Pattern が query を参照することを、`Pattern` の意味（1レコード）を壊すと判断したら、A または C へ。
- 一覧の列と record の項目が**同じ意味**であると確認できたら（たとえば列が単票の項目の部分集合だと示せたら）、A の危険は下がる。

## 実施（2026-11-17）

**B案を実装した。** `Hook.Rows`（`rows from query <query-id>`）を宣言欄として足し、検証・変換・ホスト入口まで通した。詳細と限界は [行スコープのフックが行の出所を宣言する](XTEA_LOWCODE_HOOK_ROW_SOURCE_2026-11.md) にある。

- `interaction` を browse へ広げていない（A 案は採らない）。`Items` の意味の二重化を避ける判断は維持する。
- `rows from relation` は語彙に入れていない。Relation Query の結果をこの境界へ供給する経路が無く、宣言できるのに供給できない語を増やすと「通るのに走らない」宣言ができるため。
- 出所を宣言した `scope row` は警告 `APPLICATION_PATTERN_INTERACTION_HOOK_ROW_SCOPE_HAS_NO_SUPPLIER` を出さない。出所が解決できない場合は警告ではなく**エラー**で拒否する。
- **既定の参照実行は Query を実行しない。** したがって既定経路では `scope row` は依然として理由つきで skip される。実装したのは「行を渡す層」までであり、Query 実行から行供給への配線は未実装。

## いま明記しておくこと（実装前の状態の記録）

- （実装前）行スコープのフックは、**record Pattern 上で宣言できるが、行の出所を書く欄が無いため供給経路が存在しない**。
- （実装前）したがって**既定の参照実行では必ず skip する**。警告 `APPLICATION_PATTERN_INTERACTION_HOOK_ROW_SCOPE_HAS_NO_SUPPLIER` がそのことを示す。
- この状態は暫定であり、**B（またはその変種）を実装するまで機能は動かない**。

上の3点は実装前の記録として残す。現在は `Hook.Rows` が出所を宣言でき、**出所を宣言した宣言には警告が出ない**。既定経路が Query を実行しないため skip は残るが、その理由は「出所が無い」ではなく「ホストがまだ Query を実行していない」である。


## 読み取れないこと

- browse の `interaction` が持つべき `Items` の意味（一覧の列か、行選択の項目か）。
- `Navigations`（`From` のみ）を browse で使う意味があるか。
- `MappedResult` の `Coverage`（全件か一部か）を行フックへ渡す方法。一覧は page/filter で本質的に partial。
- X-TEA の一覧の `BR`/`AR`（行ごとの読取前後）が、どのタイミングで行の値を評価可能にしていたか（`EventR` の段階宣言は script 文字列に埋まっていた）。

## 位置付け

**B案を実装した**（2026-11-17）。設計ノートとしての判断は B のままで、実装は [行の出所の宣言](XTEA_LOWCODE_HOOK_ROW_SOURCE_2026-11.md) に記録した。受容設計 D1 の「最初の対話契約は record 限定」は**変更していない**（`interaction` を browse へ広げていない）。仮承認。
