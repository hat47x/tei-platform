# ADR-0012: ProviderExpressionのauthor textとprovider-normalized representationを分離する

- 状態: 提案中
- 日付: 2026-09-01
- 関連Issue: #54, #58, #59, #60
- 前提: ADR-0010, ADR-0011 / M0.29, M0.30

## 背景

M0.29では、PostgreSQLのexpression indexとpartial predicateをActual Physical Schemaへ取り込むため、`pg_get_indexdef`や`pg_get_expr`が返す文字列を`ProviderExpression.Source`へ格納しました。

M0.30では、同じ`ProviderExpression`をDesired側にも使い、Persistence authoringに書かれたprovider-native expressionを`Source`へ格納するようにしました。

ここで、同じfieldが二つの異なる性質の文字列を表していることが明確になりました。

- Desired: 利用者が記述したauthoritativeなauthor text
- Actual: DBMSが内部表現から再構成したprovider-decompiled text

PostgreSQL公式ドキュメントでも、`pg_get_expr`はcatalog内の`pg_node_tree`をdecompileする関数、`pg_get_indexdef`はIndexの作成定義を内部表現から再構成する関数と説明されています。どちらも元のSQL文字列を返す契約ではありません。

また、`pg_index.indexprs`と`indpred`はPostgreSQL内部の`pg_node_tree`表現です。これをTEIの安定した共通serializationとして直接比較することもしません。

したがって、DesiredとActualの`Source`をtrimした文字列だけで比較し続けると、意味が同じでも表記が違うだけで`drop-index + add-index`を生成するfalse driftが起こり得ます。

一方、TEI CoreへPostgreSQL式parserや独自の式同値性判定器を組み込むことも避けます。PostgreSQL自身もpartial indexの利用可否について一般的な数学的同値性を証明する仕組みを持たず、限定的な含意判定を除けばpredicateの構造的な一致が重要です。TEIがそれより広い一般解をCoreに持つ必要はありません。

参考:

- PostgreSQL System Information Functions: https://www.postgresql.org/docs/current/functions-info.html
- PostgreSQL `pg_index`: https://www.postgresql.org/docs/current/catalog-pg-index.html
- PostgreSQL Partial Indexes: https://www.postgresql.org/docs/17/indexes-partial.html

## 判断案

`ProviderExpression`では、provider-nativeな文字列であることだけでなく、その文字列がどの表現段階にあるかを明示します。

概念上は次のようにします。

```text
ProviderExpression
  Provider
  Dialect
  Representation
  Source
  Normalization?
```

`Representation`は少なくとも次を区別します。

```text
authored
provider-normalized
```

`Normalization`は`provider-normalized`に必要なprovenanceを保持します。

```text
ExpressionNormalization
  Authority
  ProviderVersion?
```

最終的なfield名・JSON表現は実装時に確定します。重要なのは、author textとproviderが再構成したtextを同じ無印文字列として扱わないことです。

## author textは失わない

Persistence authoringの`ProviderExpressionMapping.Source`は、引き続き利用者が記述したauthoritative textです。

normalizationのためにauthoring sourceそのものを書き換えません。利用者が書いた式と、providerが比較用に再構成した式は別の責務を持ちます。

これにより、Source Index、Editor、レビューでは元の記述へ戻れます。

## Actualはprovider-normalizedとして観測する

PostgreSQL Actualで`pg_get_expr(..., false)`や`pg_get_indexdef(..., false)`から得た式は、`provider-normalized` representationとして扱います。

ここでいうnormalizedは「数学的に同値なすべての式が同じ文字列になる」という意味ではありません。特定provider / dialect / versionの内部表現をprovider自身がdecompileした比較用表現という意味です。

この区別を明示するため、`canonical`という語を無条件には使いません。DBMSが返した一つのdecompile表現を、普遍的な意味canonical formと誤解させないためです。

## provider version

provider-normalized representationは、provider versionによって表記が変わる可能性があります。

そのため、provider-owned normalizerが生成した表現を永続的な比較キーとして使う場合は、少なくともnormalization authorityと、必要に応じてprovider major versionをprovenanceとして保持します。

M0.31でversionを必須fieldにするかは、PostgreSQL 17 / 18のConformanceを通して決めます。根拠なくversion-independentと仮定しません。

## 比較規則

ProviderExpressionの比較は、単純なboolean equalityから「一致 / 不一致 / 未確定」を返せる契約へ拡張します。

少なくとも次の規則を候補とします。

### ProviderまたはDialectが異なる

同一視しません。これはnormalization不足ではなく、明確な構造差です。

### 同じrepresentation authorityでprovider-normalized同士を比較できる

同じnormalization契約を共有すると確認できる場合、normalized Sourceを比較します。

### authored同士

同じprovider / dialectでauthor textが同一なら一致とします。

異なるauthor textを数学的に同値と推測しません。authoring同士の異なる式は、原則として異なるintentです。

### authoredとprovider-normalizedが同じ文字列

同じprovider / dialectで、trim後のSourceがそのまま一致する場合は一致として扱えます。

### authoredとprovider-normalizedの文字列が異なる

信頼できるnormalizerを通していない場合、直ちに構造差と断定しません。

Schema DiffではIssue #60の「比較未確定」へ送り、`drop-index + add-index`を自動生成しない方向とします。

この扱いは、false driftからIndex再作成へ進むより安全です。

## 「比較未確定」はChangeではない

normalizationが不足していること自体はDatabase構造の変更ではありません。

したがって、`normalization-required`を`ChangeKind`へ追加しません。

Issue #60で、概念上は次のような結果を検討します。

```text
Diff
  Changes[]
  Unresolved[]
```

`Changes`には確定した構造差だけを含め、`Unresolved`には比較に追加情報が必要な項目を保持します。

migration rendererは`Unresolved`を変更命令として解釈しません。

## arbitrary textをCoreでcanonicalizeしない

PostgreSQLの`pg_get_expr`は、既にcatalogへ保存された内部expression treeをrelation OIDの文脈でdecompileする関数です。任意のtextを渡してread-onlyにcanonicalizeするAPIではありません。

そのため、通常のWorkspace load / validationでDB接続を必須にしたり、Coreが疑似PostgreSQL parserを持ったりしません。

normalizationの実装候補はprovider-ownedな境界に置きます。

候補は次のとおりです。

1. provider versionと対応したoffline parser / deparser
2. shadow databaseや隔離されたprovider環境でserver自身に解釈させる経路
3. connected validationで比較用表現を生成し、source hashとnormalization provenanceを持つ生成物として保持する経路

どの方式を採る場合も、production targetへcanonicalizationだけを目的としたDDLを発行しません。

## runtime Provider SPIへ直接足さない

現行`database/provider.Provider`は、`Capabilities`と`Introspect`を中心に、`context.Context`と`DatabaseInstance`を伴うruntime SPIです。

Desired realizationのoffline normalizationをこのinterfaceへ直接追加すると、Workspace buildに不要なruntime assemblyや接続依存を持ち込む恐れがあります。

Issue #59で、runtime Providerとは別のprovider-ownedなpure policy / normalizer境界を設計します。

ADR-0012では、その具体的なinterface shapeを先取りしません。

## provider identifier normalizationとは分ける

expression / predicateの文字列表現問題と、collation / operator class / access methodなどのprovider identifierやeffective defaultの問題は似ていますが、同じものではありません。

- expression representation / comparison: Issue #54
- expression以外のprovider default / identifier normalization: Issue #58
- provider-owned offline policy境界: Issue #59
- 比較未確定のSchema Diff表現: Issue #60

一つの巨大な`NormalizeEverything()`へまとめません。

## migrationとの境界

Issue #55でAdvanced Index migration SQLを実装するとき、対象Indexにexpression comparisonの`Unresolved`が残る場合は、drop/recreate DDLを生成しません。

これは「変更がない」とみなすのではなく、「安全に変更判断できるだけの比較情報がない」と扱います。

Editor / CLIでも、差分0と同じ表示に潰さないことを前提とします。

## Source Indexとの関係

normalization representationを追加しても、source navigationのidentityは変えません。

Persistence側のauthor textは引き続きauthoritative sourceであり、M0.30で導入した`persistence-index:<mapping>/<index>`からIndex declarationへ戻れます。

将来、expression field単位の精密diagnosticが必要になった場合はnested source anchorを追加できますが、ProviderExpressionのcomparison identityへfile pathを混ぜません。

## 非目標

- SQL式の数学的同値性の一般判定
- PostgreSQL parserのCore内製
- `pg_node_tree`をTEI共通serializationとして固定
- normalization目的のproduction DDL
- provider expressionからSemantic modelへの逆推論
- automatic index recommendation
- #55のmigration SQL renderingそのもの

## 提案段階で確認するConformance

採用前に、少なくとも次をfixtureとして確認します。

1. authored expressionと同一textのprovider-normalized expressionは一致できる
2. authored expressionと異なるprovider-normalized expressionは、normalizerがない限り構造差ではなく比較未確定になる
3. provider / dialectが異なるexpressionは明確な構造差になる
4. provider-normalized同士でもnormalization authority / version契約が異なる場合は、根拠なく同一視しない
5. unresolved expressionを持つIndexからmigrationのdrop/recreateを生成しない
6. simple column indexとexpressionを含まないAdvanced Indexには既存diff behaviorの回帰がない
7. Actual lifecycle state driftは比較未確定へ混ざらない

## 採用までの作業順序

1. Issue #54でRepresentation / Normalization metadataの最小modelを確定する
2. Issue #60で`Diff.Changes`と比較未確定を分離する
3. PostgreSQL Actualにprovider-normalized provenanceを付与する
4. Desired authored expressionとの比較Conformanceを追加する
5. Issue #59でoffline provider policy SPIを設計する
6. 必要なnormalizer経路を選び、provider/version境界をConformanceで確認する
7. ADR-0012を採用へ更新する
