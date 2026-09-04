# ADR-0010: Advanced Indexを構造的intentとactual lifecycle stateに分ける

- 状態: 採用
- 日付: 2026-09-01
- 関連Issue: #42, #50, #51
- 前提: M0.26〜M0.28 PostgreSQL Actual / Foreign Key / Unique Constraint

## 背景

M0.26〜M0.28で、Persistence MappingからDesired Physical Schemaを作り、actual PostgreSQLを観測してSchema Diffを返す経路が成立しました。

現在の`physical.Index`は、次の小さな形です。

```text
Index
  Name
  Columns[]
  Unique
  Method
```

単純なcolumn indexを比較するには十分ですが、actual databaseに存在するindexの意味を常に表せるわけではありません。

PostgreSQLでは、少なくとも次の差をcatalogから観測できます。

- column keyとexpression key
- full indexとpartial index predicate
- search keyとINCLUDE column
- keyごとのsort direction / NULL ordering
- collation / operator class
- unique indexのNULL treatment
- valid / ready / liveといったlifecycle state

この違いを失うと、たとえば「同じcolumnを持つfull index」と「一部のrowだけを対象にするpartial index」を同じものとして扱う可能性があります。逆に、PostgreSQLのcatalog fieldをそのまま全部provider-neutral modelへ持ち上げると、TEIのPhysical Schemaが一つのDBMSの内部表現へ引きずられます。

そのため、Indexについて何を**構造的なphysical intent**として保持し、何を**actual observationの状態**として保持するかを先に分けます。

## 判断

Advanced Indexを、次の二つの側面に分けて扱います。

```text
Index structural intent
  ├─ ordered keys
  ├─ included columns
  ├─ predicate
  ├─ unique / null treatment
  └─ access method

Actual index lifecycle state
  ├─ valid
  ├─ ready
  └─ live
```

structural intentはDesired / Actual双方で比較対象になります。

lifecycle stateはActualの観測情報です。Desired authoringで「invalidなindexが欲しい」といった指定は行いません。

## ordered IndexKeyへ移行する

現行`Columns []string`だけでは、columnとexpressionが混在するindexやkeyごとのoptionを表せません。

M0.29ではPhysical Schemaの内部表現を、概念上次へ移行します。

```text
Index
  Name
  Keys[]
  IncludedColumns[]
  Predicate?
  Unique
  UniqueNullTreatment?
  Method
  State?                 // Actual observation only

IndexKey
  Column | Expression    // exactly one
  Collation?
  OperatorClass?
  Order?
  NullsOrder?
```

`Keys`の順序は意味を持ちます。

単純な既存indexは、`Column`だけを持つ`IndexKey`列として同じ意味を表せます。内部JSON shapeを永久互換APIとはみなしませんが、既存のsimple index authoring / realization / diffの意味は維持します。

## column keyとexpression keyを混同しない

`pg_get_indexdef`等から得たexpressionを、存在しないcolumn名として`Columns`へ入れません。

`IndexKey`はcolumn keyとexpression keyを明示的に区別します。

expressionはCoreで汎用SQL ASTへ変換しません。Physical Schemaでは、providerが正規化したopaque expressionとして保持します。

概念上は次です。

```text
ProviderExpression
  Provider / Dialect
  Source
```

同じ文字列だからという理由だけで、PostgreSQL expressionと別providerのexpressionを同一意味とは推測しません。

M0.29ではactual observationを先に実装します。Desired側からexpression indexをauthoringする構文はIssue #51で別に設計します。

## partial predicateもprovider expressionとして保持する

partial indexのpredicateは、indexがどのrowを対象にするかを変えます。したがってstructural identityへ含めます。

ただしpredicateもCoreでSQL AST化しません。providerが正規化したexpressionとして保持し、同一provider内ではその正規形を比較します。

predicateがないfull indexとpredicateを持つpartial indexは、同じkey列・同じ名前であっても同一構造ではありません。

## INCLUDE columnをkeyと分離する

INCLUDE columnはsearch keyではありません。unique indexであっても、INCLUDE columnはuniqueness判定に参加しません。

そのため`Keys`へ混ぜず、`IncludedColumns`として分離します。

一方、Physical Schemaの収束という観点ではINCLUDEの有無は意図の差です。index-only scan等のphysical behaviorやindex layoutを変え得るため、Desired / Actualのstructural comparisonへ含めます。

M0.29では順序も保持します。providerが順序を保持して観測できる情報を、Core側で根拠なくsetへ潰しません。

## per-key option

sort directionとNULL orderingは、keyごとの構造として保持します。

```text
Order      = default | asc | desc
NullsOrder = default | first | last
```

collationとoperator classもIndexの挙動を変え得るため失いません。ただし名称体系はprovider依存なので、Coreで共通taxonomyへ無理に変換せず、Physical Schema上のopaque identifierとして保持します。

provider defaultと明示指定の同値性をどこまでnormalizeするかはprovider adapterの責務です。同じ意味を異なる文字列で返す場合、Coreに文字列置換規則を積み上げません。

## unique indexのNULL treatment

unique indexでNULLを互いに異なる値として扱うか、同じ値として扱うかはuniqueness semanticsそのものです。

したがってadvanced Indexでは、effectiveなNULL treatmentをstructural intentへ含めます。

概念上は次を区別します。

```text
UniqueNullTreatment
  distinct
  not-distinct
```

非unique indexでは値を持ちません。

Desired authoringでprovider defaultを利用する場合、provider向けDesired realization時にeffective semanticsへ解決してからActualと比較します。providerごとのdefault差をCoreが暗黙に同一視しません。

M0.28のUNIQUE constraintについて`NULLS NOT DISTINCT`を通常constraintへ黙って畳まない方針も維持します。

## lifecycle stateはstructural identityへ混ぜない

`valid / ready / live`は、index definitionそのものではなくactual DB上の運用状態です。

そのため`Index.State`としてActual Snapshotには保持しますが、`IndexKey`やpredicate等と同じstructural identityへは混ぜません。

```text
IndexState
  Valid
  Ready
  Live
```

Desired Physical SchemaではStateを指定しません。

ただし、Desiredと同じ構造のindexがActualに存在していても、利用可能状態を満たしていない場合に「差分0」とはしません。Schema Diffでは構造差とは別に、`index-state-drift`のような明示的なstate findingとして報告します。

これにより、invalid indexを完成したindexと同一視せず、同時に「構造が違うからdrop + add」と誤って説明することも避けます。

M0.29で扱うstateは`valid / ready / live`までとします。`indcheckxmin`のようにtransaction visibilityと組み合わせて判断が必要な状態は、単純なboolの追加だけで意味を固定せず後続課題とします。

## Index名はphysical matching identityとして維持する

UNIQUE constraintやForeign Keyではconstraint名をsemantic structural identityから外しました。

一方、explicit IndexはDatabase-owned access pathであり、現行Schema DiffもNameでDesired / Actualを対応付けています。M0.29でもこの挙動を維持します。

- 同じNameで構造が同じ: 同じindex
- 同じNameで構造が異なる: `drop-index` + `add-index`
- DesiredにだけNameがある: `add-index`
- ActualにだけNameがある: `drop-index`
- 同じName / 構造だがActual stateが利用可能でない: structural add/dropではなくstate drift

この判断は「名前が論理意味そのもの」という意味ではありません。physical objectを追跡し、DDLや運用状態と結び付けるmatching keyとして維持します。

## PostgreSQL providerでの観測

PostgreSQL adapterは、Core modelをPostgreSQL catalogへ合わせるのではなく、catalogから上記の意味へ正規化します。

少なくとも次を利用します。

- `indnkeyatts`: key数
- `indnatts`: key + INCLUDE数
- `indkey`: column key / expression keyの判定
- `indexprs`: expression key
- `indpred`: partial predicate
- `indcollation`: collation
- `indclass`: operator class
- `indoption`: sort / NULL ordering等のmethod-specific option
- `indnullsnotdistinct`: unique NULL treatment
- `indisvalid / indisready / indislive`: lifecycle state

PRIMARY KEY / UNIQUE constraint backing indexを通常Indexから除外するM0.28のnormalizationは維持します。

## 実装順序

### Issue #50 — Actual / Diffを先に厚くする

まずactual PostgreSQLを正しく観測し、現在起こり得る偽同一性を解消します。

- `physical.Index`をordered key modelへ移行
- existing simple Desired indexを同じ意味へrealize
- PostgreSQLからpartial / expression / INCLUDE / per-key option / NULL treatment / lifecycleを観測
- Schema Diffへadvanced structureとstate driftを反映
- actual DB Conformanceを追加

この段階では、advanced Desired authoringをまだ増やしません。

### Issue #51 — Desired authoringを後から接続する

次に、Database-owned physical intentとしてadvanced indexをauthoringできる構文を設計します。

provider expressionをSemantic modelへ逆流させず、Source Index / Diagnostic projectionも維持します。

## 採用しない案

### `Columns []string`へexpression文字列も入れる

columnとexpressionを区別できず、Source / Desired realizationも曖昧になるため採用しません。

### partial / INCLUDEを無視する

同名・同key列のindexを誤って同一と判定し得るため採用しません。

### PostgreSQLの`pg_index`全fieldをそのままPhysical Schemaへ複製する

provider内部状態とTEIが比較したい意味の境界を失うため採用しません。

### invalid indexを観測対象から単純に除外する

「存在するがDesiredを満たしていない」という重要な状態を失うため採用しません。

### lifecycle stateをstructural keyへ含める

運用途中のstate変化をindex definition変更と誤認するため採用しません。

## 影響

- M0.29では`physical.Index`の内部shapeが広がる
- simple column indexの既存authoring意味は維持する
- PostgreSQL actual observationがより忠実になる
- Schema Diffはstructural driftとlifecycle driftを区別できるようになる
- advanced authoringはIssue #51へ分離する
- migration SQL生成やautomatic repairは引き続き非目標とする

この分離によって、TEIはPostgreSQL固有catalogをそのまま抱え込まずに、physical convergenceに必要な差を失わないIndex modelへ進めます。
