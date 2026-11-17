# 為替の粒度宣言と採番例の行データの設計判断（2026-11）

## 対象

- 設計: [X-TEA標準業務ライブラリの受容設計](../design/XTEA_STANDARD_BUSINESS_SEMANTICS_2026-11.md) §6.1 の「為替の粒度を宣言必須にするか」「採番の例での規則行データ」
- 為替: `internal/currencyrate/currencyrate.go`、`binding.go`、`granularity.go`、`tables.go`、`fallback.go`、`fallback_declaration.go`
- 採番: `internal/numbering/numbering.go`、`binding.go`、`rules.go`、`rule_binding.go`、`reservation.go`、`transaction.go`
- 構文: `internal/syntax/lexer.go`、`internal/syntax/parser.go`
- 例: `examples/xtea-lowcode/semantic/master/standard.tei`、`examples/xtea-lowcode/README.md`
- 試験: `internal/currencyrate/*_test.go`、`internal/numbering/*_test.go`、`cmd/tei/xtea_standard_business_test.go`
- 対象版: `main` の `429a1673b` から分岐した `docs/xtea-open-decisions-r104`
- 環境: Windows、Go `1.27.0`
- 位置付け: 設計判断（決めるべきこと）の材料。**仮承認**であり、受容の採否・実装は別。コードは変更していない。

## 見つけたこと

### 1. 為替の粒度は「行にしか値が無い」任意の宣言

#### 1.1 宣言が無くても解決は成功する

- `ResolveGranularity` は Data の属性から `RoleGranularity`（`currency-rate-granularity`）を探し、無ければ `GranularityDeclaration{DataID: dataID}`（属性は空）を返して成功する（`internal/currencyrate/granularity.go:32-47`）。
- その後の `GranularityRows` は属性が空なら `("", false, nil)` を返す（同 54-56）。行の値を読みに行かない。
- `ResolveTable` は `Resolve` → `Bind` → `ResolveGranularity` → `GranularityRows` の順に呼び、宣言があるときだけ `CheckDeclared` で行の推定と照合する（`tables.go:76-107`）。宣言が無ければ `CheckDeclared` は呼ばれない。
- 最終的に `Table.Granularity` へ入るのは**行から推定した値**であり、宣言値ではない（同 99、108）。宣言は「行と矛盾しないことの検査」にだけ使われ、値の正本にはならない。
- 空の表は、宣言の有無にかかわらず `has no rows, so its granularity is unknown` で拒否される（同 93-98）。役割は「値を持つ列」を名指すだけで、空の表の値を与えない。これは先行記録の訂正（[為替の粒度の宣言語彙の確認](XTEA_LOWCODE_CURRENCY_GRANULARITY_2026-11.md)）と同じ結論である。

#### 1.2 行だけに頼るときの解釈

`Declaration.Granularity`（`tables.go:44-65`）の推定は次のとおり。

| 行の月順序 | 推定 |
| --- | --- |
| すべて `AnnualMonthSequence`（`0`） | `annual` |
| 1..12 が1つでもある | `monthly` |
| 0 と 1..12 が混在 | 拒否（`mixes annual and monthly rows`） |
| 行が無い | 拒否（`has no rows...`） |

`AnnualMonthSequence` は `0`（`fallback.go:22-24`）。`Bind` は 0 を年次行の目印として受理し、1..12 以外の非0を拒否する（`binding.go:109-114`）。

宣言が無い表で起きること:

1. **混在**は `currency rate cannot answer: <data> mixes annual and monthly rows` で止まる。宣言があっても同じ。
2. **空**は `currency rate cannot answer: <data> has no rows, so its granularity is unknown` で止まる。宣言があっても同じ。
3. **表を取り違えた場合**は `NewPair` が `... is not a monthly table` / `... is not an annual table` で止まる（`tables.go:129-135`）。位置が推定と合わなければ、宣言が無くても止まる。
4. **単独利用で月が無い場合**は `no rate for USD in fiscal year 2026 month 2`（`currencyrate.go:156`）で止まる。0で埋めない。
5. **年次のはずの行に月番号（例えば 1）を書いた場合**だけは、推定が `monthly` になり、宣言が無ければ止まらない。その表を `Set` で月1に引くと年次の値を返し、月2以降は `no rate` になる。`Pair` の年次側に置けば `is not an annual table` で止まるが、メッセージは「粒度宣言が無い」とは言わない。
6. **宣言がある場合**は 5 が `the table declares annual but its rows are monthly`（`granularity.go:86-93`）で止まる。これが宣言の実際の効き目である。

つまり、宣言の欠落はそれ自体として報告されず、「行が定義する粒度」をそのまま受け入れる。誤った粒度で解釈され得るのは、行の書き方が意図とずれ、かつその表が推定と一致する位置で使われた場合に限られる。エラーは Data 名・通貨・年度・月を名指すが、原因が「宣言の欠落」であることは示さない。

#### 1.3 宣言の位置付けと、必須にした場合に変わること

- 必須の役割は通貨・年度・月順序・中値の4つで、粒度は入っていない（`binding.go:70-85`）。検証は役割名として `currency-rate-granularity` を知っているが（`internal/validation/validation.go:64`、`attribute_role_type.go:30`）、必須にはしていない。
- 例の `data ExchangeRate` にも粒度の属性は無い（`examples/xtea-lowcode/semantic/master/standard.tei:21-28`）。
- `ResolveTable` の呼出し元は参照実装の試験だけである（`internal/currencyrate/tables_test.go`、`internal/standardbusiness/rq_c3_verification_test.go`）。
- 設計文書 §3 の役割表には `currency-rate-granularity` が載っていない（表は「17個」と書くが列挙は20個で、`setting-*` も載っていない）。必須にするなら、この表と §3.4 の記述を同じ変更で直す必要がある。

### 2. 採番の例に行データを置く構文は無い

- 意味ファイルのトップレベルは `type`・`data`・`capability`・`validation`・`calendar`・`period`・`effect-rule`・`run-plan` だけで（`internal/syntax/parser.go:1870-1926`）、`data` の本体は `attribute`（オプション: `description`・`display`・`value`・`required`・`immutable`・`role`）・`description`・`display`・`script`・`relation` だけ（同 2006-2165）。行・インスタンスを宣言するキーワードは無い。
- 字句は識別子・文字列・波括弧・矢印・キャレットだけで（`internal/syntax/lexer.go:11-19`）、行の区切りや値の並びを表すトークンは無い。
- 例の `data NumberingRule` は役割付きの属性6つだけで、行は無い（`examples/xtea-lowcode/semantic/master/standard.tei:30-37`）。例の README も「税率・為替・採番の行データは含みません」と明記している（`examples/xtea-lowcode/README.md:67`）。
- 行は試験が `[]map[core.CanonicalID]string` で与えている（`cmd/tei/xtea_standard_business_test.go:149-167`、`internal/numbering/rules_test.go`）。キーは正規化された属性IDである。
- ワークスペースのローダーは `sources` 配下の `.tei` を読む。`.tei` 以外は管理対象には数えるが読まず、未読の警告も出さない（`internal/workspace/source_roots.go:101-124`）。サイドカーを置いても、今のローダーは読まない。
- 採番の現在値は行ではなく **CounterStore** が正本である。`Rules.ReserveRule` は行から読んだ `State` を捨て、`ReserveInTransaction` が Store から現在値を読む（`internal/numbering/rules.go:56-68`、`reservation.go:67-93`）。行の `counter` は `BindRule` + `Contract.Next` の初期状態としてだけ効く。永続 Store（SQLite）は再起動を越えて現在値を保つ（[採番の永続CounterStoreの確認](XTEA_LOWCODE_NUMBERING_SQLITE_STORE_2026-11.md)）。
- 設計の三段は「宣言 → 行の結合 → 照会」で、行は保存行（実行時のデータ）として扱う（設計文書 §5）。行を `.tei` へ置くと、この段の意味が変わる。

## 選択肢

### (1) 為替の粒度を宣言必須にするか

#### A. 必須にする

変更の形は、`Resolve` か `ResolveTable` のどちらかで、役割が無い Data を拒否する形になる。`Resolve` に置くと、粒度を必要としない `Bind` の呼出しも巻き込む。`ResolveTable` に置くと、`Resolve` + `Bind` を直接使う経路は素通りする。

- **帰結**: 宣言と行が矛盾する表（1.2 の5）を、照会の前に止められる。粒度が Data の宣言面に現れる。一方、行を持つすべての表で、各行に `monthly`/`annual` を書く必要がある。表の性質をデータの各行へ複写する形になり、1行でも空なら `exchange rate row N: ... is empty`、行間で違えば `exchange rate rows disagree...` で止まる。
- **移行影響**: 例の `ExchangeRate` に属性を足す。`twoDataWorld` と `granularityWorld(false)` を使う試験の前提を変える。将来、実DBから行を読むローダーを作るときは、粒度列（または定数）を各行へ供給する必要がある。空の表の粒度は決まらないままで、これは必須化では解けない。
- **観測方法**: `ResolveTable` の拒否メッセージで Data を名指せる。受容前に、為替の役割を持つ Data のうち粒度の役割を持たないものを数える調査ができる。検証段階で必須にするなら新しい診断コードが要り、その試験か理由付き登録が必要になる。
- **コスト**: コードは小さい。データとスキーマの移行は、実ローダーを作る時点で中程度。宣言の形（表の性質を毎行に複写）は変わらない。

#### B. 任意のままにする（欠落の観測を足す）

- **帰結**: 現状のまま。`Table.Granularity` は行の推定値で、宣言は矛盾検査にだけ効く。混在・空・位置の取り違えは止まり、一様に誤った月番号の表だけが静かに `monthly` と解釈される。宣言の欠落は報告されない。
- **移行影響**: 無し。例・試験・将来のローダーに列を強制しない。
- **観測方法**: World を走査して、為替の役割を持ち粒度の役割を持たない Data を数える。`ResolveTable` に通して推定値を記録する。欠落を警告にしたくなったら、拒否ではなく警告として足す（その時点で診断コードの扱いを決める）。`Pair` を組む境界でだけ欠落を報告する形も考えられる。
- **コスト**: ほぼゼロ。警告を足す場合は小さい。

#### 既定案: B（任意のまま）

理由:

1. **値の正本が行にある**。宣言は行の複写を照合するだけで、空の表を記述できず、`Table.Granularity` も行の推定のまま。必須にしても「宣言が粒度を決める」形にはならない。
2. **表の性質を毎行に複写する**。粒度は Data（表）の性質なのに、現在の役割は行の値としてしか表現できない。必須にすると、この形を先に固定してしまう。
3. **静かに誤る範囲が狭い**。1.2 の5（年次意図の行に月番号）だけであり、それは行の書き方の誤りである。宣言を必須にしても、書く人が行に合わせて `monthly` と書けば止まらない。矛盾検査であって、意図の保証ではない。
4. **任意のフォールバック役割と揃う**。`currency-rate-fallback` も任意で、既定 `none` を持つ（`fallback_declaration.go:22-26`）。粒度には安全な既定が無く、行へ委ねる点だけが違う。
5. **後から表レベルの宣言へ移せる**。`data ExchangeRate { granularity monthly }` のような宣言が言語に載れば、そこで必須にする方が重複が無い。今の必須化はその移行を難しくする。

覆す条件:

- Data レベルの粒度宣言が受理されるようになったら、そちらを必須にし、行ごとの役割は非推奨にする。
- 行のローダー（DB取り込み）を作り、取り込み境界で列を供給・検査できるようになったら、境界で必須にする（`Resolve` 全体ではなく）。
- 実際の例・DBで、年次の行が月として解釈され、誤った率で換算された事象が観測されたら、`ResolveTable` で必須にして止める。
- 月次表と年次表の対が製品の経路になり、宣言時に両者を区別する必要が出たら、`Pair` の境界で必須にする。

### (2) 採番の例に規則行データを置くか

#### A. 言語に行リテラルを足す

- **帰結**: 行が World・受理済み意味の一部になる。採番の現在値は発行のたびに変わるため、番号を1つ発行するたびに意味の変更になる。行の値が属性の型・役割に合うかの検証、行の同一性（どの行がどの規則か）、行の更新の意味、形式版の扱いを決める必要がある。
- **受理との関係**: 行が意味の一部になれば、行の変更も受理の対象になる。固定版の `.tei` では、行を足すたびに受理経路が要る。
- **実装コスト**: 高い。字句・構文・AST・`ir.World`・検証・Source Index・形式版・診断コード・ラチェットに触れる。
- **`.tei` が宣言専用である設計との整合**: 低い。モデルファイルにデータと可変のカウンタが混ざる。

#### B. サイドカー＋ローダー

- **帰結**: 行は別の成果物になる。初期値（seed）か現在値かを決めないと、Store と二重の正本になる。ファイル形式・ローダー・例の配線・試験・文書が要る。`.tei` 以外を読む経路は今は無いので、置いても誰も読まない。
- **受理との関係**: 受理の対象外にできるが、その場合は「受理されていないデータ」を実行が読むことになり、正本性の説明が要る。
- **実装コスト**: 中程度。ローダーと例の配線が要る。読むようにすると、Store の現在値とサイドカーの初期値がずれるため、「seed」と明記する必要がある。
- **`.tei` が宣言専用である設計との整合**: 中程度。宣言ファイルは汚さないが、データの置き場所と権威を新しく決める。

#### C. 行はテストフィクスチャに置く

- **帰結**: 現状の運用を正式化する。行は試験入力であり、意味を主張しない。現在値の正本は Store のまま。例は宣言専用のままなので、例を実行して番号を発行する実演はできない。
- **受理との関係**: 受理の対象外。試験の失敗としてだけ現れる。
- **実装コスト**: 低い。既にその運用で、文書化と必要なら共通ヘルパーだけ。
- **`.tei` が宣言専用である設計との整合**: 高い。宣言は `.tei`、行は Store か試験、という分担を保つ。

#### 比較

| 観点 | A. 言語に行リテラル | B. サイドカー | C. テストフィクスチャ |
| --- | --- | --- | --- |
| 意味の正本性 | 行が受理済み意味になる。カウンタの発行が意味の変更になる | 別成果物。seed か現在値かを決めないと Store と二重になる | 行は意味を主張しない。Store が現在値の正本 |
| 受理との関係 | 行の変更が受理の対象 | 対象外にできるが、未受理データを実行が読む説明が要る | 対象外。試験の失敗としてだけ現れる |
| 実装コスト | 高い | 中程度 | 低い |
| `.tei` 宣言専用との整合 | 低い | 中程度 | 高い |

#### 既定案: C（行はテストフィクスチャに置く運用を正式化する）

理由:

1. `.tei` は宣言の言語であり、行は実行時のデータである。採番の現在値は特に可変で、Store が正本である（`reservation.go`）。宣言ファイルへ現在値を置くと、発行のたびに意味の版が変わる。
2. 行の置き場所は既に試験が担っており、例の README もその前提を書いている。新しい構文やローダーを足さずに、運用として明示できる。
3. 例にデータを置く必要性がまだ確定していない。必要になったときは、B（seed と明記したサイドカー）を先に検討する方が、A より小さく、正本性も説明しやすい。

覆す条件:

- CLI や Studio で例の採番をデータ付きで実演する必要が出たら、B を選ぶ。サイドカーは「初期値（seed）」と明記し、発行後の正本は Store とする。
- 言語に汎用の seed／instance 宣言が入り、受理と不変 seed の意味が定義されたら、A を再検討する。採番のカウンタは不変 seed としてのみ置き、現在値は Store に残す。
- 法定税率のように、行そのものが意味・知識として受理されるべき対象なら、`.tei` の行ではなく専用の知識成果物を検討する。採番のカウンタとは扱いが違う。

## 既定案のまとめ

| 論点 | 既定案 | 主な理由 | 覆す条件 |
| --- | --- | --- | --- |
| 為替の粒度 | 任意のまま（欠落は観測） | 値の正本は行。表の性質を毎行に複写しない。静かに誤る範囲が狭い | Data レベルの宣言、行ローダー、実測の誤換算、Pair の製品経路 |
| 採番の例の行 | テストフィクスチャに置く | `.tei` は宣言専用。現在値の正本は Store。可変値を意味にしない | データ付き実演（B）、seed 宣言の言語対応（A） |

いずれも**仮承認**であり、決まった設計ではない。受容の採否は別に判定する。

## 読み取れないこと

- 実DBの為替表・採番表のスキーマ（粒度列・フォールバック列・現在値の列）と、行がどう供給されるか。参照実装は行を呼出し側から受け取るだけで、DBを読まない。
- 為替表の「意図した粒度」を行以外から観測する方法。行が唯一の観測点である。
- サイドカーを置いた場合に、受理・Source Index・World 指紋へどう関与するか。今のローダーは `.tei` 以外を読まない。
- CLI・Studio が例の行データを表示・実行する経路。今は `view` が宣言を投影するだけで、行と計算結果を出さない（[例の標準業務Dataの宣言と結合の確認](XTEA_LOWCODE_EXAMPLE_STANDARD_DATA_2026-11.md)）。
- X-TEA の実DBでの粒度・フォールバックの実測（静的な読解のみ）。
- 設計文書 §3 の役割表の「17個」と列挙20個の不一致、および `currency-rate-granularity`・`setting-*` の欠落。必須化・語彙の確定より前に直すかどうかは未決。

## 位置付け

設計判断の材料であり、製品の受入や本番適格性を示しません。コードは変更しておらず、参照実装の挙動は読解で確認したものです。既定案は**仮承認**で、実装・受容の判断ではありません。為替の粒度は「必須にしない」、採番の例の行は「テストフィクスチャに置く」を選びましたが、どちらも上記の条件で覆せます。

## 検証

```powershell
gofmt -l internal/currencyrate internal/numbering internal/syntax
go test ./tools/repository-plan-check -count=1
```

- `gofmt -l` の出力は空（Goのファイルは変更していない）。
- `go test ./tools/repository-plan-check -count=1` は `ok`。
- `go test ./tools/repository-architecture-check -count=1` は、この作業項目の指示（`go test` は plan-check のみ）に従い実行していない。コードを変更していないため影響は無いと考えるが、未実行である。
