# ADR-0011: Advanced Index authoringをDatabase-owned Persistence intentとして記述する

- 状態: 採用
- 日付: 2026-09-01
- 関連Issue: #51, #54, #55, #58
- 前提: ADR-0010 / M0.29 Advanced Index Actual / Diff

## 背景

ADR-0010とM0.29によって、Advanced IndexのPhysical Schemaは、ordered key、INCLUDE、predicate、unique NULL treatment、access method、per-key optionを構造的なintentとして保持し、`valid / ready / live`はActual側のlifecycle stateとして分離できるようになりました。

次に必要なのは、Desired側から同じ意味構造を明示するauthoringです。

ここで重要なのは、Advanced IndexをSemantic modelから自動推論しないことです。Indexは業務上の意味そのものではなく、Databaseが所有するaccess path / physical optimizationの意思決定です。したがって、authoritative sourceはPersistence authoring側に置きます。

同時に、expression indexやpartial predicateはprovider固有SQLを含み得ます。これを共通SQL ASTへ偽装したり、Semantic modelへ逆流させたりすると、provider-neutralな意味層とphysical realizationの境界が崩れます。

## 判断

Advanced Index authoringは、既存のPersistence Mappingを拡張して表現します。

既存の単純構文は維持します。

```tei
index orders_customer_idx on sales.order.customer_id
unique-index customers_name_uidx on customer.name
```

Advanced Indexはblock formで記述します。

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

expression keyはattribute keyと明示的に分けます。

```tei
index customers_lower_name_idx {
    key expression {
        provider postgres
        dialect postgresql
        source "lower(name)"
    }
}
```

unique indexの非default NULL semanticsは、unique indexのblock内で明示します。

```tei
unique-index customers_name_nulls_nd_uidx {
    key attribute customer.name
    null-treatment not-distinct
}
```

## simple formとblock formを分ける

既存の`index NAME on ATTRIBUTE` / `unique-index NAME on ATTRIBUTE`は、単純な一列Indexを簡潔に記述する構文として残します。

block formは、ordered keyを複数持つ場合や、expression、INCLUDE、predicate、per-key option、非default NULL semanticsなどを記述する場合に使います。

同じIndex declarationの中でsimple formとblock formを混在させません。

内部の`IndexMapping.Attributes`は既存simple authoringとの互換境界として残し、advanced formは`Keys`などの構造化fieldへ格納します。Desired realizationでは両者を同じ`physical.Index`へ収束させます。

## Index key

Advanced Indexのkeyは、次のどちらか一方です。

- `attribute`: Persistence Mappingが参照するSemantic Attribute ID
- `expression`: provider / dialect provenanceを持つopaque expression

概念上は次です。

```text
IndexKeyMapping
  Attribute | Expression
  Collation?
  OperatorClass?
  Order?
  NullsOrder?
```

`Attribute`と`Expression`を同時に持つkeyは拒否します。

Attribute keyは`RealizeDesired`でMapping内のphysical columnへ解決します。source authoringに生のphysical column名を重複記述しません。

## provider expression

expression keyとpredicateでは、次を明示します。

```text
ProviderExpressionMapping
  Provider
  Dialect
  Source
```

`Provider`はTEIが認識するDatabase provider IDです。`Dialect`はprovider内のSQL方言・表現系を区別するためのprovenanceであり、Core共通SQL taxonomyにはしません。

`Source`はprovider-nativeなopaque textです。ParserやPersistence modelはSQLの意味解析を行いません。

Desired realization時には、対象Database providerとexpressionのProviderが一致することを必須とします。たとえばPostgreSQL向けDesiredを生成しているときにOracle provenanceのexpressionが混在していれば拒否します。

## expression normalizationは別課題とする

M0.29のActual側では、`pg_get_expr`や`pg_get_indexdef`などが返すprovider-normalizedな文字列を保持します。一方、Desired authoringではauthorがprovider-native expressionを記述します。

DBMSがcastやparenthesesなどを追加して同じ式を別文字列として返す可能性があります。この問題を、#51のParser実装のためのad-hocな文字列置換では解決しません。

Desired / Actual間のprovider expression normalization contractはIssue #54で別途設計します。

#51では、provider / dialect / sourceを損失なくDesired Physical Schemaへ渡し、同じ表現同士の比較がdeterministicに成立するところまでを保証します。

## INCLUDE

INCLUDE対象は生のcolumn名ではなくAttribute IDでauthoringします。

```tei
include customer.external_id
```

`RealizeDesired`がMappingのColumnMappingを通じてphysical column名へ解決します。

INCLUDEの記述順は保持します。providerが順序を保持できる情報を、authoring段階で根拠なくsetへ潰しません。

## per-key option

key blockでは次を指定できます。

```text
collation       string
operator-class  string
order           default | asc | desc
nulls           default | first | last
```

collationとoperator classはprovider-specific identifierとして扱います。Core共通taxonomyへ変換しません。

指定がない場合は`default`として保持し、provider向けDesired realizationでActual側と比較できる形へ正規化します。

### PostgreSQL B-treeで確定できるdefaultを揃える

PostgreSQLのB-treeについては、DB接続なしでもeffective defaultが確定するsort / NULL orderingをDesired側でM0.29 Actualと同じ表現へ揃えます。

- `ASC NULLS LAST`は`order=default / nulls=default`
- `DESC NULLS FIRST`は`order=desc / nulls=default`
- `ASC NULLS FIRST`は`order=default / nulls=first`
- `DESC NULLS LAST`は`order=desc / nulls=last`

`pg_catalog.default` collationをB-tree keyへ明示した場合も、現行ConformanceではActualと同じ空表現へ正規化します。

一方、default operator classや、provider / access method / physical type / provider versionに依存する一般的なcanonicalizationは、このADRへ文字列表として持ち込みません。expression以外のprovider default / identifier normalization contractはIssue #58で別途設計します。

## unique NULL treatment

`null-treatment`はunique indexだけで使用できます。

```text
null-treatment distinct | not-distinct
```

未指定時は、対象providerのeffective defaultへDesired realization時に解決します。

non-unique indexで`null-treatment`が指定されている場合は拒否します。

## access method

`method`はoptionalです。未指定時はprovider既定値をDesired realizationで解決します。

指定されたmethod名はDatabase-owned physical intentとして保持します。providerが対応しているかどうかの完全なcapability validationは、provider capabilityとmigration renderingが必要になる段階で強化します。

## Source Index

Advanced Indexはnested declarationですが、利用者がdiagnosticやEditorからIndex宣言そのものへ戻れる必要があります。

そのため、Index declaration名をpath-independentなsubjectとして扱えるsource anchorを用意します。

概念上は次です。

```text
Subject
  Kind = persistence-index
  ID   = <Persistence Mapping ID>/<Index Name>
```

Source RangeはIndex名tokenを正確に指します。

このsubjectはphysical file pathをidentityへ含めません。file rename / moveではSource Index locatorだけが変わり、Mapping / Index identityは変えません。

`<Persistence Mapping ID>/<Index Name>`を一意なsubjectとして安全に使うため、同一Persistence Mapping内のIndex名は一意でなければなりません。同名Indexが複数ある場合はDesired realizationでもSource Index構築でも拒否します。これにより、二つ目の宣言が一つ目のsource rangeへ誤って解決されることを防ぎます。

Diagnostic projectionも同じsubjectを用い、`persistence-index`をIndex名tokenの定義位置へ戻せることをConformanceで固定します。

## Desired realization

Advanced authoringはM0.29で導入した`physical.Index`へ直接対応します。

```text
Persistence IndexMapping
  -> resolve Attribute IDs
  -> verify provider provenance
  -> resolve deterministic provider defaults
  -> Physical Index structural intent
```

Desired `physical.Index.State`は常にnilです。authoringからinvalid / non-ready / non-liveなIndexを要求することはできません。

## validation責務

少なくともDesired realizationまでに次を拒否します。

- keyが0件
- simple `Attributes`とadvanced `Keys`の同時指定
- Attribute / Expressionの両方またはどちらもないkey
- Mapping内に存在しないAttribute key / INCLUDE Attribute
- expression / predicateのProviderと対象Database providerの不一致
- unknown provider
- 空のdialect / source
- non-unique indexの`null-treatment`
- unknown order / null ordering / NULL treatment
- 同一Persistence Mapping内の重複Index名

Provider固有SQLそのものが構文的・意味的に正しいかはCore Parserの責務にしません。

## migration SQLは別課題とする

Advanced Desired Indexを表現できることと、それをDDLへ安全に変換・適用できることは別の責務です。

M0.29でsimple Index以外をmigration rendererが明示的に拒否する挙動は維持します。Advanced Indexのprovider-owned SQL renderingはIssue #55で設計・実装します。

unsupportedなAdvanced Indexをsimple Indexへ黙って縮退させません。

## Reference Workspaceの扱い

既存Reference Workspaceの`index ... on ...`は、そのままsimple formの後方互換fixtureとして残します。

Advanced block formそのものは、Workspace loader / Source Index / Diagnostic projectionを通る専用Conformance fixtureで検証します。Reference Workspaceへ無理にAdvanced Indexを追加してworld snapshot全体を書き換えることは、M0.30の成立条件にはしません。

これにより、「既存authoringが変わらず動くこと」と「新しいAdvanced authoringが実際のWorkspace経路を通ること」を別々に確認できます。

## 非目標

- Semantic modelからのIndex自動推論
- automatic index recommendation
- query planner suggestionの自動採用
- provider expressionの共通SQL AST化
- provider expression normalizationの一般解（#54）
- Advanced Index migration SQL生成・自動適用（#55）
- provider default / identifier normalizationの一般解（#58）
- partition-specific index semantics（#43）
- lifecycle stateのDesired authoring

## 影響

- Persistence MappingのIndex表現がsimple互換fieldとadvanced fieldを持つ
- M0.29のPhysical SchemaをDesired側から損失なく構成できる
- provider固有expressionはDatabase-owned sourceに留まりSemantic modelへ入らない
- nested Index declarationをSource IndexとDiagnostic projectionから正確に追跡できる
- 同一Mapping内のIndex名一意性がsource identityの前提として明示される
- PostgreSQL B-treeで確定できるeffective defaultはDesired / Actualで同じ表現へ収束する
- expression normalization、一般的なprovider identifier normalization、DDL生成を独立Issueへ分離できる

## 実装順序

1. Persistence `IndexMapping`のadvanced structure
2. Advanced Index block parserとsyntax anchor
3. `RealizeDesired`でPhysical Indexへ変換
4. simple authoring互換test
5. Desired / Actual / Diff unit Conformance
6. Workspace / Source Index / Diagnostic projection統合
7. Index名一意性とPostgreSQL B-tree default normalizationのConformance
8. CI成功後にIssue #51を完了
