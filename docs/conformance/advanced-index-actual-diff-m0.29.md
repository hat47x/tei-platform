# M0.29 Advanced Index Actual / Diff Conformance

## 目的

M0.29では、PostgreSQLの実DBに存在するIndexを、単純な列名の並びへ潰さずに観測し、Desired Physical Schemaとの差を意味のある単位で説明できるようにします。

対象はIssue #50と[ADR-0010](../adr/0010-advanced-index-semantics.md)で定めたActual / Diff側です。Advanced IndexをDesiredから記述するauthoring構文はIssue #51へ分離し、この段階では追加しません。

## Physical Schemaで保持する意味

`physical.Index`は、従来の`Columns`だけの表現から、順序を持つ`Keys`を中心とする表現へ移行しました。

- column keyとexpression keyを区別する
- key順序を保持する
- INCLUDE columnをsearch keyと分離する
- partial index predicateを保持する
- keyごとのcollation / operator class / sort order / NULL orderingを保持する
- unique indexのNULL treatmentを保持する
- actual indexの`valid / ready / live`をlifecycle stateとして保持する

expressionとpredicateはCore共通SQL ASTへ変換しません。`ProviderExpression`としてprovider / dialect / 正規化済みsourceを保持し、異なるprovider間で文字列だけを根拠に同一意味とはみなしません。

既存のPersistence Mappingにある単純なIndexは、従来と同じauthoringのままcolumn `IndexKey`へrealizeされます。PostgreSQLのunique indexでは、既定のNULL treatmentをeffective semanticsである`distinct`へ解決してから比較します。

## PostgreSQL catalog normalization

PostgreSQL adapterは`pg_index`を中心に、次の情報をTEIのPhysical Schemaへ正規化します。

- `indnkeyatts`と`indkey`からkey / INCLUDEを分離
- expression keyは`pg_get_indexdef`で観測
- partial predicateは`pg_get_expr`で観測
- `indcollation`と`indclass`からcollation / operator classを観測
- btreeの`indoption`からASC / DESCとNULLS FIRST / LASTを観測
- `indnullsnotdistinct`からunique NULL treatmentを観測
- `indisvalid / indisready / indislive`からlifecycle stateを観測

provider既定のcollation / operator classや、btreeの既定sort semanticsは、単純な既存Desired Indexと不要なdriftを起こさないようprovider adapter側で既定値へ正規化します。

PRIMARY KEY / UNIQUE constraintのbacking indexを通常Indexから除外するM0.28までのnormalizationも維持します。

## Schema Diff

Index名はphysical objectを追跡するmatching keyとして維持します。同名Indexについて、次を分けて扱います。

- structural intentが同じ: 同じIndex
- structural intentが異なる: `drop-index` + `add-index`
- structural intentは同じだがactual stateが利用可能でない: `index-state-drift`

`Index.State`はstructural identityへ含めません。これにより、作成途中や失敗後のIndexを「別の定義」と誤認せず、同時に利用不能なIndexを差分0とも扱いません。

structural comparisonには、ordered keys、INCLUDE、predicate、unique / NULL treatment、access method、key optionを含めます。

## actual PostgreSQL Conformance

常設のPostgreSQL integration fixtureでは、従来の単純Index、Foreign Key、Unique Constraint、View / Materialized Viewの検証に加え、次を実DBから観測します。

1. partial + INCLUDE + collation + operator class + DESC NULLS LASTを持つIndex
2. expression keyを持つIndex
3. `NULLS NOT DISTINCT`を持つ明示的unique index
4. `CREATE INDEX CONCURRENTLY`のbuild失敗後に残るinvalid / non-ready index

4は、意図的に評価時エラーになるimmutable functionをfixture内だけに作り、concurrent index buildを失敗させてPostgreSQLが残すlifecycle stateを観測します。失敗が期待どおり発生しなかった場合はConformance自体を失敗させます。

Advanced Indexの観測確認後、それらのfixtureだけを除去してから既存のsimple Desired schemaと比較します。これにより、M0.26〜M0.28で成立した「同じ状態ならbaseline drift 0」という契約を維持したまま、M0.29の観測能力を追加しています。

## Unit Conformance

Unit testでは少なくとも次を固定します。

- usableなActual stateはstructural driftを生まない
- `valid / ready / live`のいずれかを満たさない場合は`index-state-drift`になる
- key順序の違いを保持する
- expression / INCLUDE / predicate / sort / NULL ordering / collation / operator classの差を構造差として扱う
- unique NULL treatmentの差を構造差として扱う
- 既存の単純Index authoringがcolumn `IndexKey`へrealizeされる
- PostgreSQL providerがprovider expressionとlifecycle stateをPhysical Schemaへ写像する

## 非目標

M0.29では次を行いません。

- Advanced IndexのDesired authoring DSL
- provider expressionの共通SQL AST化
- migration SQLの自動生成や自動repair
- partitioned index / partition attachment semanticsの確定
- `indcheckxmin`等、transaction visibilityを含む状態判定

これらは意味境界を混ぜず、必要な後続Issueで扱います。
