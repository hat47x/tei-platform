# M0.30 Advanced Index Desired Authoring Conformance

## 目的

M0.30では、M0.29で観測・比較できるようになったAdvanced Indexの構造を、Database-ownedなPersistence authoringからDesired Physical Schemaへ明示できるようにします。

対象はIssue #51とADR-0011です。Indexの推奨や自動生成ではなく、利用者が選んだphysical intentを損失なく記述し、Actualとの差を既存Schema Diffで比較できることを目的とします。

## authoringの位置づけ

Advanced IndexはSemantic modelの一部ではありません。

業務上の意味から「このIndexが必須である」と自動推論せず、Databaseが所有するPersistence Mappingの中に記述します。これにより、同じSemantic Dataを異なるDatabaseや異なるphysical layoutへrealizeできます。

既存のsimple formは維持します。

```tei
index customers_external_id_idx on customer.external_id
```

複数keyやprovider固有のphysical semanticsが必要な場合だけblock formを使います。

```tei
index customers_name_advanced_idx {
    method btree

    key attribute customer.name {
        collation "pg_catalog.C"
        operator-class "pg_catalog.varchar_pattern_ops"
        order desc
        nulls last
    }

    include customer.external_id

    predicate {
        provider postgres
        dialect postgresql
        source "profile IS NOT NULL"
    }
}
```

## ordered key

Advanced Indexではkeyの順序を保持します。

keyは次のどちらかです。

- `key attribute ...`
- `key expression { ... }`

Attribute keyはPersistence Mapping内のColumnMappingを通じてphysical columnへ解決します。authoringにphysical column名を重複記述しません。

Expression keyはprovider-nativeなopaque expressionとして扱います。

```tei
key expression {
    provider postgres
    dialect postgresql
    source "lower(name)"
}
```

AttributeとExpressionを同時に持つ内部modelはDesired realizationで拒否します。

## per-key semantics

key blockでは、必要な場合に次を指定できます。

- `collation`
- `operator-class`
- `order default | asc | desc`
- `nulls default | first | last`

未指定のorder / null orderingは`default`としてPhysical Schemaへrealizeされます。

collationとoperator classはprovider固有identifierです。Core共通taxonomyへ変換しません。

## INCLUDE

INCLUDE対象もSemantic Attribute IDで記述します。

```tei
include customer.external_id
```

Desired realization時にphysical columnへ解決します。INCLUDE順序は保持します。

## partial predicateとexpression provenance

partial predicateとexpression keyは次のprovenanceを必ず持ちます。

- Provider
- Dialect
- Source

対象Database providerとProviderが一致しない場合はDesired realizationを拒否します。

Provider-native SourceのSQL構文解析はTEI Core parserの責務にしません。PostgreSQL expressionをOracle expressionと、文字列が同じという理由だけで同一視することもありません。

DBMSがcastやparenthesesを補って別の文字列表現を返す場合のnormalization契約はIssue #54へ分離しています。

## unique indexのNULL treatment

unique indexでは次を明示できます。

```tei
null-treatment distinct
null-treatment not-distinct
```

未指定時はDesired realizationがproviderのeffective defaultへ解決します。

non-unique indexに`null-treatment`を指定した場合は拒否します。

## access method

`method`は任意です。

未指定時はprovider既定のmethodへDesired realizationで解決します。PostgreSQLでは現行Conformance上`btree`へ解決します。

## Source Index

Persistence Mapping内にnestedされたIndex declarationにも、独立したsource subjectを与えます。

```text
Kind = persistence-index
ID   = <Persistence Mapping ID>/<Index Name>
```

たとえば次です。

```text
persistence-index:sales_core.customer/customers_name_advanced_idx
```

Source IndexのrangeはPersistence file全体やMapping先頭ではなく、Index名tokenそのものを指します。

このidentityにはfile pathを含めません。ファイルを移動してもIndex subjectは変わらず、Source Indexのartifact locatorだけが変わります。

simple formのIndexも同じ`persistence-index` subjectで追跡します。

## Desired realization

Persistence authoringからM0.29の`physical.Index`へ次をrealizeします。

- ordered Keys
- IncludedColumns
- Predicate
- Unique
- UniqueNullTreatment
- Method
- keyごとのCollation / OperatorClass / Order / NullsOrder

Desired側の`Index.State`は常にnilです。

invalid / non-ready / non-liveといったActual lifecycle stateをDesired authoringから要求することはできません。

## Schema Diff

M0.30ではSchema Diffの比較規則自体を増やしません。

M0.29で導入したAdvanced Index structural comparisonを、Desired authoringから生成したPhysical Schemaにもそのまま利用します。

Conformance testでは次を固定します。

1. Advanced Persistence MappingからDesired Physical Schemaを生成できる
2. 同じ構造を持つusable Actual Indexとは差分0になる
3. INCLUDE等の構造を変えると`drop-index` + `add-index`になる
4. Actual lifecycle stateはDesired structural identityへ混ざらない

## parser Conformance

Parserでは次を固定します。

- 既存simple formとAdvanced block formを同じPersistence Mapping内で併用できる
- standalone Advanced Index parserとPersistence埋め込みが同じbody parserを使う
- keyがないAdvanced Indexを拒否する
- unknown providerを拒否する
- provider / dialect / sourceの不足を拒否する
- unknown order / NULL orderingを拒否する
- non-unique indexの`null-treatment`を拒否する
- `index NAME`の後が`on`でも`{`でもない場合を拒否する

## Workspace Conformance

実際のWorkspace loader経路で、Advanced Indexを含むPersistence fileを読み込めることを固定します。

同じfixtureからSource Indexを構築し、simple / Advanced双方のIndex declarationが正確な名前rangeへ解決されることも検証します。

## migration SQLとの境界

M0.30は「Desiredとして表現できること」までを対象とします。

Advanced Indexを安全なprovider-native DDLへ変換する責務はIssue #55へ分離しています。

現行migration rendererは、対応していないAdvanced構造をsimple indexへ黙って縮退させず、明示的に未対応として拒否します。

## 非目標

- Index recommendation
- query planからの自動authoring
- provider expression normalizationの一般解
- Advanced Index migration SQL生成
- automatic migration apply
- concurrent build policy
- partitioned index semantics
- Desired lifecycle state

## 次の境界

M0.30完了後も、Advanced Indexには少なくとも二つの独立した課題が残ります。

- Issue #54: Desired / Actual間のProviderExpression normalization contract
- Issue #55: provider-owned Advanced Index migration SQL rendering

この分離により、authoring構文の都合でDBMS固有の式正規化やDDL運用判断をCoreへ取り込まないようにします。
