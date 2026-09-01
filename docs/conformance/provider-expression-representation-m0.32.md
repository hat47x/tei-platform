# M0.32 ProviderExpression Representation / Comparison Conformance

## 目的

M0.32では、ProviderExpressionに格納された文字列が「利用者が記述した式」なのか、「providerが内部表現から再構成した式」なのかを明示し、表記差だけを根拠に破壊的なSchema Changeを生成しない比較契約を固定します。

対象はIssue #54とADR-0012です。M0.31で導入した`Diff.Unresolved`を比較結果の安全な受け皿として利用します。

## representation

`physical.ProviderExpression`は次のrepresentationを区別します。

```text
authored
provider-normalized
```

`authored`はPersistence authoring等で利用者が記述したauthoritative textです。

`provider-normalized`はprovider自身の内部表現から再構成された比較用表現です。ここでいうnormalizedは数学的なcanonical formを意味しません。

## normalization provenance

provider-normalized表現には、必要に応じて次のprovenanceを付与します。

```text
ExpressionNormalization
  Authority
  ProviderVersion?
```

`Authority`は、どのprovider-ownedな正規化・deparse契約で生成されたかを示します。

`ProviderVersion`は、その表現契約がprovider versionに依存する場合の比較範囲です。値を取得できない場合に推測で補いません。

## Desired

Persistence MappingからDesired Physical Schemaへrealizeしたexpression key / predicateは`authored`です。

Sourceは前後空白を除く既存のauthor textを保持します。normalization provenanceは持ちません。

Source Indexのauthoritative sourceは引き続きPersistence fileであり、provider-normalized表現へ置き換えません。

## PostgreSQL Actual

PostgreSQL catalog adapterが`pg_get_expr` / `pg_get_indexdef`系の経路から取得したexpression / predicateは`provider-normalized`として扱います。

M0.32ではnormalization authorityを次に固定します。

```text
postgresql-catalog-deparse
```

現行CatalogReaderはserver versionをexpression provenanceとして返さないため、`ProviderVersion`は空のままです。

versionを知らない状態で、異なるprovider-normalized textを「意味が違う」と断定しません。

## 比較規則

Provider / Dialectが異なる場合は明確な構造差です。

Provider / Dialectが同じ場合は次の順に判定します。

### source textが同一

representationが異なっても一致とします。

これは数学的同値性の推測ではなく、比較対象のprovider-native text自体が一致しているためです。

### authored / authoredでsourceが異なる

異なるauthoring intentとして構造差とします。

TEIが一方をもう一方へ自動的にcanonicalizeしたものではないため、利用者が明示的に異なる式を書いた事実を保持します。

### provider-normalized / provider-normalizedでsourceが異なる

次の条件をすべて満たす場合だけ構造差と確定します。

- normalization authorityが同一かつ空でない
- provider versionが同一かつ空でない

同じ契約・同じversionのprovider-normalized表現で異なるtextが得られたためです。

version不明、authority不明、または契約が異なる場合は`Unresolved`とします。

### authored / provider-normalizedでsourceが異なる

信頼できるnormalizerをまだ通していないため`Unresolved`とします。

この状態から`drop-index + add-index`を生成しません。

### legacy representation

`Representation`が空の旧来データは、sourceが一致する場合のみ一致できます。

sourceが異なる場合は、representationを推測せず`Unresolved`とします。

## migration safety

M0.31の契約どおり、`Diff.Unresolved`が残る場合はmigration SQLを生成しません。

したがって、Desired authored textとPostgreSQL provider-normalized textが表記だけ異なる可能性のあるケースから、自動的なIndex drop/recreateへ進みません。

## 非目標

- PostgreSQL SQL parserのCore内製
- SQL式の数学的同値性判定
- arbitrary textをPostgreSQL catalog deparseへ通す仕組み
- server version取得SPIの追加
- offline provider normalizer policyの最終形（Issue #59）
- collation / operator class等のnormalization（Issue #58）
- Advanced Index DDL rendering（Issue #55）

## Conformance

M0.32では少なくとも次を固定します。

1. Desired expression / predicateは`authored`になる
2. PostgreSQL Actual expressionは`provider-normalized`になり、authorityを保持する
3. authored / authoredの異なるsourceは構造差になる
4. authored / provider-normalizedの異なるsourceはUnresolvedになる
5. 同じauthority・同じ既知versionのprovider-normalized同士は異なるsourceを構造差と判定できる
6. version不明のprovider-normalized同士は異なるsourceをUnresolvedにする
7. normalization authorityが異なる場合はUnresolvedにする
8. source textが完全一致すればrepresentationが異なっても収束する
9. legacy representationの既存M0.31安全挙動を維持する
10. unresolved comparisonからmigration SQLを生成しない

## 次の段階

M0.32によって、ProviderExpressionの表現段階と比較契約はPhysical Schema上で明示されます。

次に必要なのは、Issue #59でprovider-ownedなoffline realization / normalization policyの所有場所を整理し、必要な場合にDesired authored textから比較可能なprovider-normalized representationを生成する境界を設計することです。
